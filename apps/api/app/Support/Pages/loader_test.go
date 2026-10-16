package pages

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(_ context.Context, req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestValidateDataRoute(t *testing.T) {
	if err := ValidateDataRoute("https://evil.com/x"); err == nil {
		t.Fatal("absolute url")
	}
	if err := ValidateDataRoute("//evil.com"); err == nil {
		t.Fatal("protocol relative")
	}
	if err := ValidateDataRoute("../x"); err == nil {
		t.Fatal("traversal")
	}
	if err := ValidateDataRoute("/docs/data"); err != nil {
		t.Fatal(err)
	}
	// R2 修正：带尾斜杠的前缀此前只匹配 "//"，/api/... 与 /control-panel/... 漏过校验。
	for _, reserved := range []string{"/api/internal", "/api", "/admin/users", "/control-panel/settings"} {
		if err := ValidateDataRoute(reserved); err == nil {
			t.Fatalf("reserved route %q accepted", reserved)
		}
	}
}

func TestLoaderGatewayUsesAndReleasesRuntimeAdmission(t *testing.T) {
	targetCtx, cancel := context.WithCancel(context.Background())
	var released atomic.Int32
	started := make(chan struct{})
	loader := NewPageDataLoader(roundTripFunc(func(request *http.Request) (*http.Response, error) {
		close(started)
		<-request.Context().Done()
		return nil, request.Context().Err()
	}))
	gateway := NewLoaderGateway(loader, admissionTargets{
		ctx: targetCtx, baseURL: "http://127.0.0.1:19999",
		release: func() {
			released.Add(1)
		},
	})
	result := make(chan LoaderResult, 1)
	go func() {
		result <- gateway.LoadForContribution(context.Background(), PageContribution{
			ExtensionID: "admission.page", DataSource: "plugin", DataRoute: "/data",
		}, nil, "zh-CN", 0)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("page loader did not use the admitted runtime")
	}
	cancel()
	select {
	case loaded := <-result:
		if !loaded.Fallback || loaded.Status != 504 {
			t.Fatalf("cancelled loader result = %#v", loaded)
		}
	case <-time.After(time.Second):
		t.Fatal("page loader ignored runtime drain cancellation")
	}
	if released.Load() != 1 {
		t.Fatalf("runtime admission releases = %d", released.Load())
	}
}

func TestLoaderGatewayReleasesAdmissionWhenTargetIsEmpty(t *testing.T) {
	var released atomic.Int32
	gateway := NewLoaderGateway(NewPageDataLoader(nil), admissionTargets{
		ctx: context.Background(),
		release: func() {
			released.Add(1)
		},
	})
	result := gateway.LoadForContribution(context.Background(), PageContribution{
		ExtensionID: "empty.page", DataSource: "plugin", DataRoute: "/data",
	}, nil, "zh-CN", 0)
	if !result.Fallback || result.Status != 503 || released.Load() != 1 {
		t.Fatalf("empty target result=%#v releases=%d", result, released.Load())
	}
}

func TestLoaderGatewayExactLoadRequiresCompleteRuntimeArtifact(t *testing.T) {
	var calls atomic.Int32
	targets := countingExactTargets{calls: &calls}
	gateway := NewLoaderGateway(NewPageDataLoader(nil), targets)
	base := PageContribution{
		ExtensionID: "plugin.exact", Version: "1.0.0", PackageDigest: strings.Repeat("a", 64),
		RuntimeInstanceID: "runtime-1", DataSource: "plugin", DataRoute: "/page-data",
	}
	for _, mutate := range []func(*PageContribution){
		func(item *PageContribution) { item.Version = "" },
		func(item *PageContribution) { item.PackageDigest = "" },
		func(item *PageContribution) { item.RuntimeInstanceID = "" },
	} {
		contribution := base
		mutate(&contribution)
		result := gateway.LoadExactForContribution(context.Background(), contribution, nil, "zh-CN", 0)
		if !result.Fallback || result.Status != http.StatusServiceUnavailable {
			t.Fatalf("incomplete exact artifact result = %#v", result)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("incomplete exact artifacts acquired runtime %d times", calls.Load())
	}
}

func TestDecodeLoaderDataPreservesJSONNumberPrecision(t *testing.T) {
	decoded, ok := DecodeLoaderData(json.RawMessage(`{"sequence":9007199254740993}`)).(map[string]any)
	if !ok {
		t.Fatalf("decoded loader data=%#v", decoded)
	}
	sequence, ok := decoded["sequence"].(json.Number)
	if !ok || sequence.String() != "9007199254740993" {
		t.Fatalf("decoded sequence=%#v", decoded["sequence"])
	}
	encoded, err := json.Marshal(decoded)
	if err != nil || string(encoded) != `{"sequence":9007199254740993}` {
		t.Fatalf("encoded loader data=%s err=%v", encoded, err)
	}
}

func TestLoaderSuccess(t *testing.T) {
	l := NewPageDataLoader(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Cookie") != "" {
			t.Fatal("must not forward cookie")
		}
		if r.Header.Get("X-SForum-Actor-ID-Hint") != "7" {
			t.Fatalf("actor hint: %s", r.Header.Get("X-SForum-Actor-ID-Hint"))
		}
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Fatal("must not forward auth headers")
		}
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"title":"ok"}`)),
		}, nil
	}))
	res := l.Fetch(context.Background(), LoaderRequest{
		ExtensionID: "p",
		Route:       "/docs/data",
		ActorID:     7,
		TargetBase:  "http://127.0.0.1:9999",
	})
	if res.Error != "" || string(res.Data) != `{"title":"ok"}` {
		t.Fatalf("%#v", res)
	}
}

func TestLoaderRejectsLargeAndSensitive(t *testing.T) {
	l := NewPageDataLoader(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"password":"x"}`)),
		}, nil
	}))
	res := l.Fetch(context.Background(), LoaderRequest{
		ExtensionID: "p", Route: "/d", TargetBase: "http://127.0.0.1:1",
	})
	if res.Error == "" || !res.Fallback {
		t.Fatalf("expected sensitive reject: %#v", res)
	}
}

func TestLoader401(t *testing.T) {
	l := NewPageDataLoader(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 401, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
	}))
	res := l.Fetch(context.Background(), LoaderRequest{
		ExtensionID: "p", Route: "/d", TargetBase: "http://localhost:1",
	})
	if res.Status != 401 {
		t.Fatalf("%#v", res)
	}
}

func TestLoaderRejectsNonLoopback(t *testing.T) {
	l := NewPageDataLoader(nil)
	res := l.Fetch(context.Background(), LoaderRequest{
		ExtensionID: "p", Route: "/d", TargetBase: "https://evil.example",
	})
	if !res.Fallback {
		t.Fatalf("%#v", res)
	}
}

func TestResolveReplaceCarriesDataSchema(t *testing.T) {
	store := NewMemoryStore()
	reg := NewRegistry(store)
	if err := reg.RegisterContributions("plug", []PageContribution{{
		ID: "plug.home", Action: ActionReplace, Target: "forum.home",
		Template: "templates/home.html", Contract: "sforum.page.home@1",
		DataSource: "plugin", DataRoute: "/home/data", DataSchema: "schemas/home.json",
		ExtensionID: "plug", Version: "1.0.0", PackageDigest: "d1",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := reg.ApproveReplace(context.Background(), ProviderBinding{
		PageID: "forum.home", ExtensionID: "plug", ContributionID: "plug.home",
		Version: "1.0.0", PackageDigest: "d1", ContractVersion: "sforum.page.home@1", ApprovedBy: 1,
	}); err != nil {
		t.Fatal(err)
	}
	resolved, err := reg.Resolve(context.Background(), "forum.home")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.DataSchema != "schemas/home.json" {
		t.Fatalf("replace Resolve must carry DataSchema, got %#v", resolved)
	}
	if resolved.DataRoute != "/home/data" || resolved.DataSource != "plugin" {
		t.Fatalf("data fields: %#v", resolved)
	}
}

func TestLoadForResolvedAppliesDataSchema(t *testing.T) {
	dir := t.TempDir()
	schemaPath := dir + "/schemas"
	if err := os.MkdirAll(schemaPath, 0o755); err != nil {
		t.Fatal(err)
	}
	// required: title
	if err := os.WriteFile(dir+"/schemas/home.json", []byte(`{"type":"object","required":["title"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	loader := NewPageDataLoader(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			// missing required title → schema fail
			Body: io.NopCloser(strings.NewReader(`{"slug":"x"}`)),
		}, nil
	}))
	gw := NewLoaderGateway(loader, fakeTargets{bases: map[string]string{"plug": "http://127.0.0.1:19999"}}).
		WithPackages(fakePackages{roots: map[string]string{"plug": dir}})

	resolved := ResolvedPage{
		Provider: "plug", ExtensionID: "plug",
		DataSource: "plugin", DataRoute: "/home/data", DataSchema: "schemas/home.json",
	}
	res := gw.LoadForResolved(context.Background(), resolved, "zh-CN", 0)
	if res.Error == "" || !strings.Contains(res.Error, "schema") {
		t.Fatalf("expected schema validation on replace path, got %#v", res)
	}

	// success path with title
	loaderOK := NewPageDataLoader(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"title":"Home"}`)),
		}, nil
	}))
	gwOK := NewLoaderGateway(loaderOK, fakeTargets{bases: map[string]string{"plug": "http://127.0.0.1:19999"}}).
		WithPackages(fakePackages{roots: map[string]string{"plug": dir}})
	resOK := gwOK.LoadForResolved(context.Background(), resolved, "zh-CN", 0)
	if resOK.Error != "" {
		t.Fatalf("expected schema pass: %#v", resOK)
	}
	if string(resOK.Data) != `{"title":"Home"}` {
		t.Fatalf("data: %s", resOK.Data)
	}
}

type fakeTargets struct {
	bases map[string]string
}

type admissionTargets struct {
	ctx     context.Context
	baseURL string
	invoke  LoaderDocumentFunc
	release func()
}

type countingExactTargets struct {
	calls *atomic.Int32
}

func (f countingExactTargets) AcquireRouteTarget(context.Context, RuntimeArtifact) (LoaderRouteTarget, bool) {
	f.calls.Add(1)
	return LoaderRouteTarget{}, false
}

func (f admissionTargets) AcquireRouteTarget(context.Context, RuntimeArtifact) (LoaderRouteTarget, bool) {
	return LoaderRouteTarget{
		BaseURL: f.baseURL,
		Invoke:  f.invoke,
		Context: f.ctx,
		Release: f.release,
	}, true
}

// Protocol V2 loader 传输：无 loopback，经 exact runtime instance 的调用闭包取数。
// 允许侧证明数据真的到达网关；拒绝侧证明两条传输共用同一份响应后校验
// （route 校验、大小、JSON、敏感键、schema）。
func TestLoaderGatewayPrefersProtocolV2TransportWithoutLoopback(t *testing.T) {
	var calls atomic.Int32
	gateway := NewLoaderGateway(NewPageDataLoader(nil), admissionTargets{
		ctx:     context.Background(),
		release: func() {},
		invoke: func(_ context.Context, route string, params map[string]string, locale string, actorID int64) ([]byte, int, error) {
			calls.Add(1)
			if route != "/page-data/article" || params["slug"] != "hello" || locale != "zh-CN" || actorID != 0 {
				t.Errorf("v2 loader input = route=%q params=%v locale=%q actor=%d", route, params, locale, actorID)
			}
			return []byte(`{"title":"from protocol v2","count":3}`), 200, nil
		},
	})
	result := gateway.LoadForContribution(context.Background(), PageContribution{
		ExtensionID: "v2.page", DataSource: "plugin", DataRoute: "/page-data/article",
	}, map[string]string{"slug": "hello"}, "zh-CN", 0)
	if result.Fallback || result.Status != 200 || string(result.Data) != `{"title":"from protocol v2","count":3}` {
		t.Fatalf("v2 loader result = %#v", result)
	}
	if calls.Load() != 1 {
		t.Fatalf("v2 invoke calls = %d", calls.Load())
	}
}

func TestProtocolV2LoaderTransportSharesValidation(t *testing.T) {
	tests := []struct {
		name       string
		route      string
		document   []byte
		status     int
		wantStatus int
	}{
		{name: "reserved route", route: "/api/internal", document: []byte(`{}`), status: 200, wantStatus: 422},
		{name: "absolute route", route: "http://evil.test/data", document: []byte(`{}`), status: 200, wantStatus: 422},
		{name: "non json body", route: "/data", document: []byte(`not json`), status: 200, wantStatus: 502},
		{name: "sensitive key", route: "/data", document: []byte(`{"authorization":"Bearer x"}`), status: 200, wantStatus: 502},
		{name: "oversize body", route: "/data", document: append([]byte(`{"pad":"`), append(make([]byte, MaxLoaderResponseBytes), []byte(`"}`)...)...), status: 200, wantStatus: 502},
		{name: "plugin error status", route: "/data", document: []byte(`{}`), status: 500, wantStatus: 500},
		{name: "plugin forbidden", route: "/data", document: []byte(`{}`), status: 403, wantStatus: 403},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gateway := NewLoaderGateway(NewPageDataLoader(nil), admissionTargets{
				ctx:     context.Background(),
				release: func() {},
				invoke: func(context.Context, string, map[string]string, string, int64) ([]byte, int, error) {
					return test.document, test.status, nil
				},
			})
			result := gateway.LoadForContribution(context.Background(), PageContribution{
				ExtensionID: "v2.page", DataSource: "plugin", DataRoute: test.route,
				DataSchema: "schemas/page.json",
			}, nil, "zh-CN", 0)
			if !result.Fallback || result.Status != test.wantStatus {
				t.Fatalf("v2 validation result = %#v (want status %d)", result, test.wantStatus)
			}
		})
	}
}

// FetchV2 必须与 HTTP 路径共用同一份响应后校验；schema 分支在 gateway 层依赖包根
// 读文件，因此这里直接对 FetchV2 提供 schema 原文。
func TestFetchV2AppliesDeclaredSchema(t *testing.T) {
	loader := NewPageDataLoader(nil)
	result := loader.FetchV2(context.Background(), LoaderRequest{
		ExtensionID: "v2.page", Route: "/page-data/article",
		SchemaJSON: `{"type":"object","required":["title"]}`,
	}, func(context.Context, string, map[string]string, string, int64) ([]byte, int, error) {
		return []byte(`{"other":1}`), 200, nil
	})
	if !result.Fallback || result.Status != 502 || result.Error == "" {
		t.Fatalf("v2 schema result = %#v", result)
	}
	ok := loader.FetchV2(context.Background(), LoaderRequest{
		ExtensionID: "v2.page", Route: "/page-data/article",
		SchemaJSON: `{"type":"object","required":["title"]}`,
	}, func(context.Context, string, map[string]string, string, int64) ([]byte, int, error) {
		return []byte(`{"title":"ok"}`), 200, nil
	})
	if ok.Fallback || ok.Status != 200 || string(ok.Data) != `{"title":"ok"}` {
		t.Fatalf("v2 schema allowed result = %#v", ok)
	}
	if closed := loader.FetchV2(context.Background(), LoaderRequest{Route: "/data"}, nil); closed.Status != 503 || !closed.Fallback {
		t.Fatalf("nil invoker result = %#v", closed)
	}
}

func TestLoaderGatewayWithoutTransportStaysClosed(t *testing.T) {
	var released atomic.Int32
	gateway := NewLoaderGateway(NewPageDataLoader(nil), admissionTargets{
		ctx: context.Background(),
		release: func() {
			released.Add(1)
		},
	})
	result := gateway.LoadForContribution(context.Background(), PageContribution{
		ExtensionID: "v2.page", DataSource: "plugin", DataRoute: "/data",
	}, nil, "zh-CN", 0)
	if !result.Fallback || result.Status != 503 || released.Load() != 1 {
		t.Fatalf("no-transport result=%#v releases=%d", result, released.Load())
	}
}

func (f fakeTargets) AcquireRouteTarget(ctx context.Context, artifact RuntimeArtifact) (LoaderRouteTarget, bool) {
	b, ok := f.bases[artifact.ExtensionID]
	return LoaderRouteTarget{BaseURL: b, Context: ctx, Release: func() {}}, ok
}

type fakePackages struct {
	roots map[string]string
}

func (f fakePackages) PackageRoot(id string) (string, bool) {
	r, ok := f.roots[id]
	return r, ok
}

package pages

import (
	"context"
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
	release func()
}

func (f admissionTargets) AcquireRouteTarget(context.Context, string) (LoaderRouteTarget, bool) {
	return LoaderRouteTarget{
		BaseURL: f.baseURL,
		Context: f.ctx,
		Release: f.release,
	}, true
}

func (f fakeTargets) AcquireRouteTarget(ctx context.Context, id string) (LoaderRouteTarget, bool) {
	b, ok := f.bases[id]
	return LoaderRouteTarget{BaseURL: b, Context: ctx, Release: func() {}}, ok
}

type fakePackages struct {
	roots map[string]string
}

func (f fakePackages) PackageRoot(id string) (string, bool) {
	r, ok := f.roots[id]
	return r, ok
}

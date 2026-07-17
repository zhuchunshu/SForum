package extensionsruntime

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

func TestRouteGatewayCancelsInflightRequestWithAdmissionContext(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
	}))
	defer server.Close()

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)
	request.Header.SetMethod(fasthttp.MethodGet)
	request.SetRequestURI("/slow")
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- NewRouteGateway().Proxy(&ProxyInput{
			Context: ctx, Request: request, Response: response,
			ExtensionID: "demo.plugin", TargetBase: server.URL, TargetPath: "/slow", Timeout: time.Second,
		})
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("route proxy did not reach the runtime")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled route error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("route proxy ignored admission cancellation")
	}
}

func TestRouteGatewayProxiesRequestAndTrustedHeaders(t *testing.T) {
	var receivedExtensionID string
	server := fasthttp.Server{Handler: func(ctx *fasthttp.RequestCtx) {
		receivedExtensionID = string(ctx.Request.Header.Peek("X-SForum-Extension-ID"))
		ctx.SetStatusCode(fasthttp.StatusCreated)
		ctx.SetBodyString("plugin-ok")
	}}
	listener := listenLocalhost(t)
	defer listener.Close()
	go server.Serve(listener)
	defer server.Shutdown()

	gateway := NewRouteGateway()
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetRequestURI("/api/v1/extensions/demo.plugin/hello")
	req.SetBodyString("payload")

	err := gateway.Proxy(&ProxyInput{
		Request:     req,
		Response:    resp,
		ExtensionID: "demo.plugin",
		TargetBase:  "http://" + listener.Addr().String(),
		TargetPath:  "/hello",
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatalf("Proxy returned error: %v", err)
	}
	if resp.StatusCode() != fasthttp.StatusCreated || string(resp.Body()) != "plugin-ok" {
		t.Fatalf("unexpected proxy response status=%d body=%q", resp.StatusCode(), string(resp.Body()))
	}
	if receivedExtensionID != "demo.plugin" {
		t.Fatalf("expected trusted extension header, got %q", receivedExtensionID)
	}
}

func TestRouteGatewayStripsPluginLinkResponseHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("X-Plugin-Metadata", "kept")
		for _, value := range []string{
			`<https://evil.example/>; rel="canonical"`,
			`</asset.js>; rel="preload canonical"`,
			`</page/2?value=a,b>; REL=Canonical; title="quoted, comma"`,
			`</next>; rel="next"`,
		} {
			writer.Header().Add("Link", value)
		}
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("plugin-ok"))
	}))
	defer server.Close()

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)
	request.Header.SetMethod(fasthttp.MethodGet)
	request.SetRequestURI("/api/v1/extensions/demo.plugin/links")

	if err := NewRouteGateway().Proxy(&ProxyInput{
		Request: request, Response: response, ExtensionID: "demo.plugin",
		TargetBase: server.URL, TargetPath: "/links", Timeout: time.Second,
	}); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode() != http.StatusOK || string(response.Body()) != "plugin-ok" ||
		len(response.Header.PeekAll("Link")) != 0 || string(response.Header.Peek("X-Plugin-Metadata")) != "kept" {
		t.Fatalf("status=%d headers=%v body=%q", response.StatusCode(), &response.Header, response.Body())
	}
	for _, name := range []string{"Link", "link", "lInK"} {
		if routeResponseHeaderAllowed(name) {
			t.Fatalf("reserved response header %q accepted", name)
		}
	}
}

func TestRouteGatewayDoesNotFollowRedirects(t *testing.T) {
	var destinationCalls atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		destinationCalls.Add(1)
	}))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Location", destination.URL)
		writer.WriteHeader(http.StatusPermanentRedirect)
	}))
	defer source.Close()

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)
	request.Header.SetMethod(fasthttp.MethodGet)
	request.SetRequestURI("/redirect")

	err := NewRouteGateway().Proxy(&ProxyInput{
		Request: request, Response: response, ExtensionID: "demo.plugin",
		TargetBase: source.URL, TargetPath: "/redirect", Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode() != http.StatusPermanentRedirect || destinationCalls.Load() != 0 {
		t.Fatalf("status=%d destinationCalls=%d", response.StatusCode(), destinationCalls.Load())
	}
}

// TestRouteGatewayStripsSpoofableClientHeaders 客户端伪造的身份/鉴权头不得转发到插件。
func TestRouteGatewayStripsSpoofableClientHeaders(t *testing.T) {
	var (
		gotActorID     string
		gotExtensionID string
		gotLocale      string
		gotAuth        string
		gotCookie      string
		gotCSRF        string
		gotForged      string
		gotHop         string
		gotTrace       string
	)
	server := fasthttp.Server{Handler: func(ctx *fasthttp.RequestCtx) {
		gotActorID = string(ctx.Request.Header.Peek("X-SForum-Actor-ID"))
		gotExtensionID = string(ctx.Request.Header.Peek("X-SForum-Extension-ID"))
		gotLocale = string(ctx.Request.Header.Peek("X-SForum-Locale"))
		gotAuth = string(ctx.Request.Header.Peek("Authorization"))
		gotCookie = string(ctx.Request.Header.Peek("Cookie"))
		gotCSRF = string(ctx.Request.Header.Peek("X-Csrf-Token"))
		gotForged = string(ctx.Request.Header.Peek("X-SForum-Forged"))
		gotHop = string(ctx.Request.Header.Peek("X-Hop-Secret"))
		gotTrace = string(ctx.Request.Header.Peek("X-Trace-ID"))
		ctx.SetStatusCode(fasthttp.StatusOK)
	}}
	listener := listenLocalhost(t)
	defer listener.Close()
	go server.Serve(listener)
	defer server.Shutdown()

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI("/api/v1/extensions/demo.plugin/hello")
	// 客户端伪造的身份与会话头。
	req.Header.Set("X-SForum-Actor-ID", "spoofed-actor")
	req.Header.Set("X-SForum-Extension-ID", "spoofed.plugin")
	req.Header.Set("X-SForum-Locale", "spoofed-locale")
	req.Header.Set("X-SForum-Forged", "spoofed-authority")
	req.Header.Set("Connection", "keep-alive, X-Hop-Secret")
	req.Header.Set("X-Hop-Secret", "hop-secret")
	req.Header.Set("Authorization", "Bearer stolen-token")
	req.Header.Set("Cookie", "sforum_session=evil")
	req.Header.Set("X-Csrf-Token", "double-submit-secret")
	req.Header.Set("X-Trace-ID", "trace-41")

	err := NewRouteGateway().Proxy(&ProxyInput{
		Request:     req,
		Response:    resp,
		ExtensionID: "demo.plugin",
		// 匿名 public 路由：宿主不设置 ActorID。
		ActorID:    "",
		Locale:     "zh-CN",
		TargetBase: "http://" + listener.Addr().String(),
		TargetPath: "/hello",
		Timeout:    time.Second,
	})
	if err != nil {
		t.Fatalf("Proxy returned error: %v", err)
	}
	if gotActorID != "" {
		t.Fatalf("client-spoofed Actor-ID must not reach plugin, got %q", gotActorID)
	}
	if gotExtensionID != "demo.plugin" {
		t.Fatalf("expected host Extension-ID, got %q", gotExtensionID)
	}
	if gotLocale != "zh-CN" {
		t.Fatalf("expected host Locale, got %q", gotLocale)
	}
	if gotAuth != "" {
		t.Fatalf("Authorization must not be forwarded, got %q", gotAuth)
	}
	if gotCookie != "" {
		t.Fatalf("Cookie must not be forwarded, got %q", gotCookie)
	}
	if gotCSRF != "" {
		t.Fatalf("X-Csrf-Token must not be forwarded, got %q", gotCSRF)
	}
	if gotForged != "" {
		t.Fatalf("client-spoofed X-SForum authority must not reach plugin, got %q", gotForged)
	}
	if gotHop != "" {
		t.Fatalf("Connection-named hop header must not reach plugin, got %q", gotHop)
	}
	if gotTrace != "trace-41" {
		t.Fatalf("ordinary request header was lost, trace=%q", gotTrace)
	}
}

// TestRouteGatewaySetsHostAuthoredActorID 宿主写入的 Actor-ID 应覆盖客户端伪造值。
func TestRouteGatewaySetsHostAuthoredActorID(t *testing.T) {
	var gotActorID string
	server := fasthttp.Server{Handler: func(ctx *fasthttp.RequestCtx) {
		gotActorID = string(ctx.Request.Header.Peek("X-SForum-Actor-ID"))
		ctx.SetStatusCode(fasthttp.StatusOK)
	}}
	listener := listenLocalhost(t)
	defer listener.Close()
	go server.Serve(listener)
	defer server.Shutdown()

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI("/hello")
	req.Header.Set("X-SForum-Actor-ID", "spoofed")

	err := NewRouteGateway().Proxy(&ProxyInput{
		Request:     req,
		Response:    resp,
		ExtensionID: "demo.plugin",
		ActorID:     "42",
		TargetBase:  "http://" + listener.Addr().String(),
		TargetPath:  "/hello",
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatalf("Proxy returned error: %v", err)
	}
	if gotActorID != "42" {
		t.Fatalf("expected host Actor-ID 42, got %q", gotActorID)
	}
}

func listenLocalhost(t testing.TB) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen localhost: %v", err)
	}
	return listener
}

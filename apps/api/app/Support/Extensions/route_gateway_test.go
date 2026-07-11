package extensionsruntime

import (
	"net"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

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

// TestRouteGatewayStripsSpoofableClientHeaders 客户端伪造的身份/鉴权头不得转发到插件。
func TestRouteGatewayStripsSpoofableClientHeaders(t *testing.T) {
	var (
		gotActorID     string
		gotExtensionID string
		gotLocale      string
		gotAuth        string
		gotCookie      string
	)
	server := fasthttp.Server{Handler: func(ctx *fasthttp.RequestCtx) {
		gotActorID = string(ctx.Request.Header.Peek("X-SForum-Actor-ID"))
		gotExtensionID = string(ctx.Request.Header.Peek("X-SForum-Extension-ID"))
		gotLocale = string(ctx.Request.Header.Peek("X-SForum-Locale"))
		gotAuth = string(ctx.Request.Header.Peek("Authorization"))
		gotCookie = string(ctx.Request.Header.Peek("Cookie"))
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
	req.Header.Set("Authorization", "Bearer stolen-token")
	req.Header.Set("Cookie", "sforum_session=evil")

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

func listenLocalhost(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen localhost: %v", err)
	}
	return listener
}

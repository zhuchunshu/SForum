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

func listenLocalhost(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen localhost: %v", err)
	}
	return listener
}

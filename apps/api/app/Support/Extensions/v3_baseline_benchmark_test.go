package extensionsruntime

import (
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

// BenchmarkRouteGatewayV1Baseline 测量当前 namespaced proxy 的 loopback 往返。
func BenchmarkRouteGatewayV1Baseline(b *testing.B) {
	server := fasthttp.Server{Handler: func(ctx *fasthttp.RequestCtx) {
		// 当前 Proxy 每次调用创建 HostClient；基准服务端主动断开，避免基准本身
		// 被遗留 idle 连接耗尽。P0 报告另行记录未主动断开时的超时风险。
		ctx.Response.Header.SetConnectionClose()
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBodyString("ok")
	}}
	listener := listenLocalhost(b)
	defer listener.Close()
	go func() { _ = server.Serve(listener) }()
	defer server.Shutdown()

	gateway := NewRouteGateway()
	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)
	request.Header.SetMethod(fasthttp.MethodGet)
	request.SetRequestURI("/api/v1/extensions/benchmark.plugin/ping")
	input := ProxyInput{
		Request: request, Response: response, ExtensionID: "benchmark.plugin",
		TargetBase: "http://" + listener.Addr().String(), TargetPath: "/ping", Timeout: time.Second,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		response.Reset()
		if err := gateway.Proxy(&input); err != nil {
			b.Fatal(err)
		}
	}
}

package extensionsruntime

import (
	"context"
	"os"
	"path/filepath"
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

// BenchmarkPluginRPCV1Baseline 使用真实 go-plugin 子进程测量 net/rpc health 往返。
func BenchmarkPluginRPCV1Baseline(b *testing.B) {
	packageRoot := filepath.Join(b.TempDir(), "benchmark.plugin", "1.0.0")
	filesRoot := filepath.Join(packageRoot, "files", "backend")
	if err := os.MkdirAll(filesRoot, 0o755); err != nil {
		b.Fatal(err)
	}
	targetBinary := filepath.Join(filesRoot, "plugin")
	if err := os.WriteFile(targetBinary, []byte(helperPluginLauncher(b)), 0o755); err != nil {
		b.Fatal(err)
	}
	starter := NewProtocolStarter(ProtocolStarterConfig{})
	extension := runtimeExtension("benchmark.plugin")
	extension.PackagePath = filepath.Join(packageRoot, "package.zip")
	if _, err := starter.Start(context.Background(), extension); err != nil {
		b.Fatal(err)
	}
	defer starter.Stop(context.Background(), extension)
	protocol := starter.protocolFor(extension.ID)
	if protocol == nil {
		b.Fatal("protocol was not registered")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		health, err := protocol.Health()
		if err != nil || !health.OK {
			b.Fatalf("health=%#v err=%v", health, err)
		}
	}
}

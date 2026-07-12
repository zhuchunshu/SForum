package bootstrap

import (
	"context"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

func TestWorkerStartStopAllowNilClient(t *testing.T) {
	worker := &Worker{}
	if err := worker.Start(context.Background()); err != nil {
		t.Fatalf("start idle worker: %v", err)
	}
	if err := worker.Stop(context.Background()); err != nil {
		t.Fatalf("stop idle worker: %v", err)
	}
}

func TestWorkerCloseRunsCleanupOnce(t *testing.T) {
	calls := 0
	worker := &Worker{
		close: func() {
			calls++
		},
	}

	worker.Close()
	worker.Close()

	if calls != 1 {
		t.Fatalf("expected cleanup once, got %d", calls)
	}
}

// TestResolveWorkerExtensionRuntimeInjectedSkipsStandalone 覆盖 embed 路径：
// 注入 runtime 时不得调用 buildStandalone（否则会二次 Reconcile / 双起插件）。
func TestResolveWorkerExtensionRuntimeInjectedSkipsStandalone(t *testing.T) {
	injected := &countingWorkerRuntime{}
	standaloneBuilds := 0

	runtime, gateway, owns, err := resolveWorkerExtensionRuntime(
		workerRuntimeDeps{ExtensionRuntime: injected, OwnsRuntime: false},
		func() (workerExtensionRuntime, hostAPIGatewayCloser, error) {
			standaloneBuilds++
			return &countingWorkerRuntime{}, &noopHostGateway{}, nil
		},
	)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if runtime != injected {
		t.Fatal("expected injected runtime to be returned")
	}
	if gateway != nil {
		t.Fatal("expected no host gateway when runtime is injected")
	}
	if owns {
		t.Fatal("expected OwnsRuntime=false for injected runtime")
	}
	if standaloneBuilds != 0 {
		t.Fatalf("expected standalone builder not called, got %d", standaloneBuilds)
	}
}

// TestResolveWorkerExtensionRuntimeStandaloneOwns 覆盖独立 worker：自建并拥有 runtime。
func TestResolveWorkerExtensionRuntimeStandaloneOwns(t *testing.T) {
	standalone := &countingWorkerRuntime{}
	gw := &noopHostGateway{}
	standaloneBuilds := 0

	runtime, gateway, owns, err := resolveWorkerExtensionRuntime(
		workerRuntimeDeps{},
		func() (workerExtensionRuntime, hostAPIGatewayCloser, error) {
			standaloneBuilds++
			return standalone, gw, nil
		},
	)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if runtime != standalone {
		t.Fatal("expected standalone runtime")
	}
	if gateway != gw {
		t.Fatal("expected standalone host gateway")
	}
	if !owns {
		t.Fatal("expected OwnsRuntime=true for standalone")
	}
	if standaloneBuilds != 1 {
		t.Fatalf("expected standalone builder once, got %d", standaloneBuilds)
	}
}

// TestEmbedSharedRuntimeSingleStart 模拟 API Reconcile + embed 注入后不再 Start。
// 真实双起根因是 newWorkerWithPool 自建 Manager 再 Reconcile；注入后 Start 计数应保持 1。
func TestEmbedSharedRuntimeSingleStart(t *testing.T) {
	starter := &countingStarter{}
	manager := extensionsruntime.NewManager(extensionsruntime.ManagerConfig{Starter: starter})
	plugin := extensions.Extension{
		ID:     "sforum.smtp",
		Type:   extensions.TypePlugin,
		Status: extensions.StatusEnabled,
		Manifest: extensions.Manifest{
			Backend: extensions.ManifestBackend{Entry: "backend/plugin"},
		},
	}

	// API 路径：Reconcile 一次 → Start 一次。
	manager.Reconcile(context.Background(), []extensions.Extension{plugin})
	if starter.starts["sforum.smtp"] != 1 {
		t.Fatalf("after API reconcile expected 1 start, got %d", starter.starts["sforum.smtp"])
	}

	// Embed 路径：注入同一 manager，resolve 不得触发二次 build/Reconcile。
	runtime, _, owns, err := resolveWorkerExtensionRuntime(
		workerRuntimeDeps{ExtensionRuntime: manager, OwnsRuntime: false},
		func() (workerExtensionRuntime, hostAPIGatewayCloser, error) {
			// 若被调用，会再 NewManager+Reconcile，导致第二次 Start。
			t.Fatal("standalone builder must not run when runtime is injected")
			return nil, nil, nil
		},
	)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if owns {
		t.Fatal("embed must not own shared runtime")
	}
	// 注入后 worker 侧不应再 Reconcile；Start 仍为 1。
	if starter.starts["sforum.smtp"] != 1 {
		t.Fatalf("after embed inject expected still 1 start, got %d", starter.starts["sforum.smtp"])
	}
	// Worker.Close 在 owns=false 时不应 Close runtime；模拟 API 顺序：先 worker close 再 runtime close。
	worker := &Worker{close: nil} // ownsRuntime=false → closeFn nil
	worker.Close()
	if starter.stops["sforum.smtp"] != 0 {
		t.Fatalf("worker close must not stop shared plugin, stops=%d", starter.stops["sforum.smtp"])
	}
	runtime.Close(context.Background())
	if starter.stops["sforum.smtp"] != 1 {
		t.Fatalf("API close should stop plugin once, got %d", starter.stops["sforum.smtp"])
	}
}

type countingWorkerRuntime struct {
	closeCalls int
}

func (c *countingWorkerRuntime) SendMail(context.Context, string, extensionsruntime.MailProviderRequest) (extensionsruntime.MailProviderResponse, error) {
	return extensionsruntime.MailProviderResponse{}, nil
}

func (c *countingWorkerRuntime) Reconcile(context.Context, []extensions.Extension) {}

func (c *countingWorkerRuntime) Close(context.Context) { c.closeCalls++ }

type noopHostGateway struct{}

func (noopHostGateway) Close() error { return nil }

type countingStarter struct {
	starts map[string]int
	stops  map[string]int
}

func (s *countingStarter) Start(_ context.Context, extension extensions.Extension) (extensionsruntime.RouteTarget, error) {
	if s.starts == nil {
		s.starts = map[string]int{}
	}
	s.starts[extension.ID]++
	return extensionsruntime.RouteTarget{}, nil
}

func (s *countingStarter) Stop(_ context.Context, extension extensions.Extension) error {
	if s.stops == nil {
		s.stops = map[string]int{}
	}
	s.stops[extension.ID]++
	return nil
}

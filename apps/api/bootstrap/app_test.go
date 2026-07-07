package bootstrap

import (
	"context"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

func TestAPIAddressUsesConfiguredHostAndPort(t *testing.T) {
	cfg := config.Config{HTTPHost: "127.0.0.1", HTTPPort: "8081"}

	if got := apiAddress(cfg); got != "127.0.0.1:8081" {
		t.Fatalf("expected configured address, got %q", got)
	}
}

func TestAPICloseRunsCleanupOnce(t *testing.T) {
	calls := 0
	api := &API{
		close: func() {
			calls++
		},
	}

	api.Close()
	api.Close()

	if calls != 1 {
		t.Fatalf("expected cleanup once, got %d", calls)
	}
}

func TestShouldEmbedWorkerInAPIRequiresConfigFlag(t *testing.T) {
	if !shouldEmbedWorkerInAPI(config.Config{EmbedWorkerInAPI: true}) {
		t.Fatal("expected embedded worker to start when config flag is enabled")
	}
	if shouldEmbedWorkerInAPI(config.Config{EmbedWorkerInAPI: false}) {
		t.Fatal("expected embedded worker to stay disabled when config flag is disabled")
	}
}

func TestExtensionRuntimeFactoryCanBeReplacedForBootstrapTests(t *testing.T) {
	original := newExtensionRuntimeManager
	defer func() { newExtensionRuntimeManager = original }()

	called := false
	newExtensionRuntimeManager = func(extensions.Store) extensionRuntime {
		called = true
		return fakeBootstrapExtensionRuntime{}
	}

	runtime := newExtensionRuntimeManager(nil)
	runtime.Reconcile(context.Background(), []extensions.Extension{{
		ID:     "demo.plugin",
		Type:   extensions.TypePlugin,
		Status: extensions.StatusEnabled,
	}})
	runtime.Close(context.Background())

	if !called {
		t.Fatal("expected runtime factory replacement to be called")
	}
}

func TestNewHumanVerifyServiceRespectsDisabledProvider(t *testing.T) {
	service, err := newHumanVerifyService(config.Config{
		HumanVerificationProvider: "disabled",
		AltchaChallengeTTL:        time.Minute,
		AltchaCost:                1,
	}, humanverify.NewMemoryStore())
	if err != nil {
		t.Fatalf("newHumanVerifyService returned error: %v", err)
	}
	if service.Enabled() {
		t.Fatal("expected disabled human verifier")
	}
}

type fakeBootstrapExtensionRuntime struct{}

func (fakeBootstrapExtensionRuntime) Check(context.Context, extensions.Extension) error { return nil }

func (fakeBootstrapExtensionRuntime) Start(context.Context, extensions.Extension) error { return nil }

func (fakeBootstrapExtensionRuntime) Stop(context.Context, extensions.Extension) error { return nil }

func (fakeBootstrapExtensionRuntime) Status(context.Context, extensions.Extension) extensions.RuntimeStatus {
	return extensions.RuntimeStatus{State: extensions.RuntimeStopped}
}

func (fakeBootstrapExtensionRuntime) EmitHook(context.Context, string, map[string]any) {}

func (fakeBootstrapExtensionRuntime) Emit(context.Context, appevents.Envelope) appevents.Result {
	return appevents.Result{OK: true}
}

func (fakeBootstrapExtensionRuntime) RouteTarget(string) (extensionsruntime.RouteTarget, bool) {
	return extensionsruntime.RouteTarget{}, false
}

func (fakeBootstrapExtensionRuntime) Reconcile(context.Context, []extensions.Extension) {}

func (fakeBootstrapExtensionRuntime) Close(context.Context) {}

func TestNewHumanVerifyServiceEnablesAltchaProvider(t *testing.T) {
	service, err := newHumanVerifyService(config.Config{
		HumanVerificationProvider: "altcha",
		AltchaSecret:              "test-secret",
		AltchaChallengeTTL:        time.Minute,
		AltchaCost:                1,
	}, humanverify.NewMemoryStore())
	if err != nil {
		t.Fatalf("newHumanVerifyService returned error: %v", err)
	}
	if !service.Enabled() {
		t.Fatal("expected enabled human verifier")
	}
}

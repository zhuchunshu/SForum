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

func TestEnsureInitialWebReleaseQueuesOnlyWithoutLiveRelease(t *testing.T) {
	for _, test := range []struct {
		name      string
		hasLive   bool
		wantCalls int
	}{
		{name: "empty runtime", hasLive: false, wantCalls: 1},
		{name: "existing live release", hasLive: true, wantCalls: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			reader := &fakeInitialReleaseReader{hasLive: test.hasLive}
			queuer := &fakeInitialReleaseQueuer{}
			if err := ensureInitialWebRelease(context.Background(), reader, queuer); err != nil {
				t.Fatal(err)
			}
			if queuer.calls != test.wantCalls {
				t.Fatalf("expected %d queue calls, got %d", test.wantCalls, queuer.calls)
			}
			if test.wantCalls == 1 && (queuer.input.Plan.TriggerKind != extensions.WebReleaseTriggerRebuild || queuer.input.Plan.ReloadMode != extensions.WebReleaseReloadPrompt) {
				t.Fatalf("unexpected initial release plan: %#v", queuer.input.Plan)
			}
		})
	}
}

type fakeInitialReleaseReader struct{ hasLive bool }

func (r *fakeInitialReleaseReader) HasLiveWebRelease(context.Context) (bool, error) {
	return r.hasLive, nil
}

type fakeInitialReleaseQueuer struct {
	calls int
	input extensions.QueueWebReleaseInput
}

func (q *fakeInitialReleaseQueuer) PlanAndQueue(_ context.Context, input extensions.QueueWebReleaseInput) (extensions.WebReleaseQueueResult, error) {
	q.calls++
	q.input = input
	return extensions.WebReleaseQueueResult{}, nil
}

func TestExtensionRuntimeFactoryCanBeReplacedForBootstrapTests(t *testing.T) {
	original := newExtensionRuntimeManager
	defer func() { newExtensionRuntimeManager = original }()

	called := false
	newExtensionRuntimeManager = func(extensions.Store, extensionsruntime.HostAPIRegistrar, extensionsruntime.PluginSettings) extensionRuntime {
		called = true
		return fakeBootstrapExtensionRuntime{}
	}

	runtime := newExtensionRuntimeManager(nil, nil, nil)
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

func (fakeBootstrapExtensionRuntime) SendMail(context.Context, string, extensionsruntime.MailProviderRequest) (extensionsruntime.MailProviderResponse, error) {
	return extensionsruntime.MailProviderResponse{}, nil
}

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

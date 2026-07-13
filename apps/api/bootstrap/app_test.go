package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Crypto"
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
	newExtensionRuntimeManager = func(extensions.Store, extensionsruntime.HostAPIRegistrar, extensionsruntime.PluginSettings, extensionsruntime.RuntimeTrustSource) extensionRuntime {
		called = true
		return fakeBootstrapExtensionRuntime{}
	}

	runtime := newExtensionRuntimeManager(nil, nil, nil, nil)
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

func TestAPIExtensionRuntimeUsesCipherServiceSettings(t *testing.T) {
	original := newExtensionRuntimeManager
	defer func() { newExtensionRuntimeManager = original }()

	cipher, _ := crypto.NewOptionCipher(strings.Repeat("a", 64))
	enc, _ := cipher.Encrypt("api-secret")
	item := runtimeSettingsExtension("api.plugin")
	store := &bootstrapExtensionSettingsStore{
		item:     item,
		settings: map[string]string{"token": enc},
	}
	service := extensions.NewService(store, t.TempDir())
	extensions.WithCipher(cipher)(service)

	var got map[string]string
	newExtensionRuntimeManager = func(_ extensions.Store, _ extensionsruntime.HostAPIRegistrar, settings extensionsruntime.PluginSettings, _ extensionsruntime.RuntimeTrustSource) extensionRuntime {
		var err error
		got, err = settings.ListSettings(context.Background(), item.ID)
		if err != nil {
			t.Fatalf("load API runtime settings: %v", err)
		}
		return fakeBootstrapExtensionRuntime{}
	}
	_ = bindAPIExtensionRuntime(store, nil, service, nil)
	if got["token"] != "api-secret" {
		t.Fatalf("API runtime received %#v", got)
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

func (fakeBootstrapExtensionRuntime) StoragePutBegin(context.Context, string, extensionsruntime.StoragePutBeginRequest) (extensionsruntime.StorageSessionResponse, error) {
	return extensionsruntime.StorageSessionResponse{}, nil
}
func (fakeBootstrapExtensionRuntime) StoragePutChunk(context.Context, string, extensionsruntime.StoragePutChunkRequest) (extensionsruntime.StorageResult, error) {
	return extensionsruntime.StorageResult{}, nil
}
func (fakeBootstrapExtensionRuntime) StorageOpen(context.Context, string, extensionsruntime.StorageOpenRequest) (extensionsruntime.StorageSessionResponse, error) {
	return extensionsruntime.StorageSessionResponse{}, nil
}
func (fakeBootstrapExtensionRuntime) StorageGetChunk(context.Context, string, extensionsruntime.StorageGetChunkRequest) (extensionsruntime.StorageGetChunkResponse, error) {
	return extensionsruntime.StorageGetChunkResponse{}, nil
}
func (fakeBootstrapExtensionRuntime) StorageClose(context.Context, string, extensionsruntime.StorageCloseRequest) (extensionsruntime.StorageResult, error) {
	return extensionsruntime.StorageResult{}, nil
}
func (fakeBootstrapExtensionRuntime) StorageDelete(context.Context, string, extensionsruntime.StorageObjectRequest) (extensionsruntime.StorageResult, error) {
	return extensionsruntime.StorageResult{}, nil
}
func (fakeBootstrapExtensionRuntime) StorageStat(context.Context, string, extensionsruntime.StorageStatRequest) (extensionsruntime.StorageStatResponse, error) {
	return extensionsruntime.StorageStatResponse{}, nil
}
func (fakeBootstrapExtensionRuntime) StorageExists(context.Context, string, extensionsruntime.StorageExistsRequest) (extensionsruntime.StorageExistsResponse, error) {
	return extensionsruntime.StorageExistsResponse{}, nil
}
func (fakeBootstrapExtensionRuntime) StoragePublicURL(context.Context, string, extensionsruntime.StoragePublicURLRequest) (extensionsruntime.StorageURLResponse, error) {
	return extensionsruntime.StorageURLResponse{}, nil
}
func (fakeBootstrapExtensionRuntime) StorageSignedURL(context.Context, string, extensionsruntime.StorageSignedURLRequest) (extensionsruntime.StorageURLResponse, error) {
	return extensionsruntime.StorageURLResponse{}, nil
}
func (fakeBootstrapExtensionRuntime) StorageProbe(context.Context, string, extensionsruntime.StorageProbeRequest) (extensionsruntime.StorageProbeResponse, error) {
	return extensionsruntime.StorageProbeResponse{}, nil
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

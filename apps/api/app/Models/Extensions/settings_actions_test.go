package extensions

import (
	"context"
	"errors"
	"strings"
	"testing"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	secretstore "github.com/zhuchunshu/sforum/apps/api/app/Support/SecretStore"
	settingslifecycle "github.com/zhuchunshu/sforum/apps/api/app/Support/SettingsLifecycle"
)

func TestSettingsActionRunsIsolatedProbeWithExplicitSecretSemantics(t *testing.T) {
	item := installedExtension("probe.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"})
	item.Status = StatusDisabled
	item.Manifest.Providers = []ManifestProvider{{Slot: "mail.provider", Label: "Mail"}}
	item.Manifest.Settings = []ManifestSetting{
		{Key: "host", Label: LocalizedText{Default: "Host"}, Type: "text", Default: "localhost"},
		{Key: "password", Label: LocalizedText{Default: "Password"}, Type: "secret"},
	}
	item.Manifest.SettingsDocument = extensionmanifest.SettingsDocument{
		SchemaVersion: 1,
		UI:            extensionmanifest.SettingsUI{Mode: "schema", Layout: "form"},
		Fields:        item.Manifest.Settings,
		Actions:       []extensionmanifest.SettingsAction{{ID: "probe", Kind: "provider_probe", Label: LocalizedText{Default: "Probe"}, Placement: "footer", UseDraftValues: true}},
		Explicit:      true,
	}
	store := &fakeExtensionStore{
		items:    map[string]Extension{item.ID: item},
		settings: map[string]map[string]string{item.ID: {"password": "stored-secret"}},
	}
	runtime := &recordingSettingsActionRuntime{result: SettingsActionProbeResult{OK: true, Reason: "ok", Message: "connected"}}
	auditor := &recordingAuditor{}
	service := NewService(store, t.TempDir())
	WithSettingsActionRuntime(runtime)(service)
	WithAuditor(auditor)(service)

	result, err := service.ExecuteSettingsAction(context.Background(), extensionManager(), item.ID, "probe", ExecuteSettingsActionInput{
		Values:  map[string]string{"host": "smtp.example.com"},
		Secrets: map[string]SettingsActionSecretInput{"password": {Mode: "preserve"}},
	})
	if err != nil || !result.Success {
		t.Fatalf("probe failed: result=%#v err=%v", result, err)
	}
	if runtime.values["host"] != "smtp.example.com" || runtime.values["password"] != "stored-secret" {
		t.Fatalf("unexpected probe values: %#v", runtime.values)
	}
	if len(auditor.events) != 1 {
		t.Fatalf("expected action audit, got %#v", auditor.events)
	}
	for _, value := range auditor.events[0].Metadata {
		if value == "stored-secret" {
			t.Fatal("secret leaked into audit metadata")
		}
	}
}

func TestSettingsActionUsesLifecycleSecretAndKeepsSecretSetVisible(t *testing.T) {
	item := installedExtension("probe.lifecycle", TypePlugin, ManifestBackend{Entry: "backend/plugin"})
	item.Status = StatusDisabled
	item.Manifest.Providers = []ManifestProvider{{Slot: "mail.provider", Label: "Mail"}}
	item.Manifest.Settings = []ManifestSetting{
		{Key: "host", Label: LocalizedText{Default: "Host"}, Type: "text", Default: "localhost"},
		{Key: "password", Label: LocalizedText{Default: "Password"}, Type: "secret"},
	}
	item.Manifest.SettingsDocument = extensionmanifest.SettingsDocument{
		SchemaVersion: 1,
		UI:            extensionmanifest.SettingsUI{Mode: "schema", Layout: "form"},
		Fields:        item.Manifest.Settings,
		Actions:       []extensionmanifest.SettingsAction{{ID: "probe", Kind: "provider_probe", Label: LocalizedText{Default: "Probe"}, Placement: "footer", UseDraftValues: true}},
		Explicit:      true,
	}
	store := &fakeExtensionStore{items: map[string]Extension{item.ID: item}}
	secrets, err := secretstore.New(secretstore.NewMemoryStore(), nil)
	if err != nil {
		t.Fatal(err)
	}
	lifecycle := settingslifecycle.NewWithStore(settingslifecycle.NewMemoryDocumentStore(), secrets)
	runtime := &recordingSettingsActionRuntime{result: SettingsActionProbeResult{OK: true, Reason: "ok", Message: "connected"}}
	service := NewService(store, t.TempDir())
	WithSettingsLifecycle(lifecycle)(service)
	WithSettingsActionRuntime(runtime)(service)

	updated, err := service.UpdateSettings(context.Background(), extensionManager(), item.ID, UpdateSettingsInput{
		Values: map[string]string{"host": "smtp.example.com", "password": "stored-secret"},
	}, "zh-CN")
	if err != nil {
		t.Fatalf("UpdateSettings returned error: %v", err)
	}
	if settingValue(updated, "password") != "" || !updated.Items[1].SecretSet {
		t.Fatalf("saved secret should remain masked and visible as configured: %#v", updated.Items[1])
	}

	result, err := service.ExecuteSettingsAction(context.Background(), extensionManager(), item.ID, "probe", ExecuteSettingsActionInput{
		Values:  map[string]string{"host": "smtp.example.com"},
		Secrets: map[string]SettingsActionSecretInput{"password": {Mode: "preserve"}},
	})
	if err != nil || !result.Success {
		t.Fatalf("probe failed: result=%#v err=%v", result, err)
	}
	if runtime.values["password"] != "stored-secret" {
		t.Fatalf("probe must receive resolved secret plaintext, got %#v", runtime.values)
	}
	if strings.HasPrefix(runtime.values["password"], "sforum.secret://") {
		t.Fatalf("probe received secret reference instead of plaintext: %#v", runtime.values)
	}
}

func TestSettingsActionRejectsUnknownFieldsAndImplicitSecrets(t *testing.T) {
	item := installedExtension("probe-invalid.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"})
	item.Manifest.Providers = []ManifestProvider{{Slot: "mail.provider", Label: "Mail"}}
	item.Manifest.Settings = []ManifestSetting{{Key: "password", Label: LocalizedText{Default: "Password"}, Type: "secret"}}
	item.Manifest.SettingsDocument = extensionmanifest.SettingsDocument{SchemaVersion: 1, UI: extensionmanifest.SettingsUI{Mode: "schema", Layout: "form"}, Fields: item.Manifest.Settings, Actions: []extensionmanifest.SettingsAction{{ID: "probe", Kind: "provider_probe", Label: LocalizedText{Default: "Probe"}, Placement: "footer", UseDraftValues: true}}, Explicit: true}
	service := NewService(&fakeExtensionStore{items: map[string]Extension{item.ID: item}}, t.TempDir())
	WithSettingsActionRuntime(&recordingSettingsActionRuntime{})(service)

	_, err := service.ExecuteSettingsAction(context.Background(), extensionManager(), item.ID, "probe", ExecuteSettingsActionInput{Values: map[string]string{"unknown": "x"}})
	if !errors.Is(err, ErrSettingsActionInvalid) {
		t.Fatalf("expected invalid unknown field, got %v", err)
	}
	_, err = service.ExecuteSettingsAction(context.Background(), extensionManager(), item.ID, "probe", ExecuteSettingsActionInput{})
	if !errors.Is(err, ErrSettingsActionInvalid) {
		t.Fatalf("expected explicit secret semantics, got %v", err)
	}
}

func TestSettingsActionPermissionAndUnknownAction(t *testing.T) {
	item := installedExtension("probe-policy.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"})
	item.Manifest.Providers = []ManifestProvider{{Slot: "mail.provider", Label: "Mail"}}
	item.Manifest.SettingsDocument = extensionmanifest.SettingsDocument{SchemaVersion: 1, UI: extensionmanifest.SettingsUI{Mode: "schema", Layout: "form"}, Actions: []extensionmanifest.SettingsAction{{ID: "probe", Kind: "provider_probe", Label: LocalizedText{Default: "Probe"}, Placement: "footer"}}, Explicit: true}
	service := NewService(&fakeExtensionStore{items: map[string]Extension{item.ID: item}}, t.TempDir())
	WithSettingsActionRuntime(&recordingSettingsActionRuntime{})(service)
	if _, err := service.ExecuteSettingsAction(context.Background(), identity.Actor{ID: 99, Status: identity.UserStatusActive}, item.ID, "probe", ExecuteSettingsActionInput{}); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denial, got %v", err)
	}
	if _, err := service.ExecuteSettingsAction(context.Background(), extensionManager(), item.ID, "missing", ExecuteSettingsActionInput{}); !errors.Is(err, ErrSettingsActionInvalid) {
		t.Fatalf("expected unknown action rejection, got %v", err)
	}
}

func TestSettingsActionRejectsOversizedInputAndBoundsPluginFailure(t *testing.T) {
	item := installedExtension("probe-bounds.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"})
	item.Manifest.Providers = []ManifestProvider{{Slot: "mail.provider", Label: "Mail"}}
	item.Manifest.Settings = []ManifestSetting{{Key: "host", Label: LocalizedText{Default: "Host"}, Type: "text"}}
	item.Manifest.SettingsDocument = extensionmanifest.SettingsDocument{
		SchemaVersion: 1, UI: extensionmanifest.SettingsUI{Mode: "schema", Layout: "form"}, Fields: item.Manifest.Settings,
		Actions: []extensionmanifest.SettingsAction{{ID: "probe", Kind: "provider_probe", Label: LocalizedText{Default: "Probe"}, Placement: "footer", UseDraftValues: true}}, Explicit: true,
	}
	store := &fakeExtensionStore{items: map[string]Extension{item.ID: item}}
	runtime := &recordingSettingsActionRuntime{err: errors.New(strings.Repeat("provider failed ", 600))}
	service := NewService(store, t.TempDir())
	WithSettingsActionRuntime(runtime)(service)

	if _, err := service.ExecuteSettingsAction(context.Background(), extensionManager(), item.ID, "probe", ExecuteSettingsActionInput{
		Values: map[string]string{"host": strings.Repeat("x", maxSettingsActionInputBytes+1)},
	}); !errors.Is(err, ErrSettingsActionInvalid) {
		t.Fatalf("oversized draft must be rejected, got %v", err)
	}
	if _, err := service.ExecuteSettingsAction(context.Background(), extensionManager(), item.ID, "probe", ExecuteSettingsActionInput{
		Values: map[string]string{"host": "smtp.example.com"},
	}); !errors.Is(err, ErrSettingsActionUnavailable) {
		t.Fatalf("plugin failure must be host-owned unavailable result, got %v", err)
	}
}

func TestSettingsActionHonorsCanceledRequestContext(t *testing.T) {
	item := installedExtension("probe-timeout.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"})
	item.Manifest.Providers = []ManifestProvider{{Slot: "mail.provider", Label: "Mail"}}
	item.Manifest.SettingsDocument = extensionmanifest.SettingsDocument{
		SchemaVersion: 1, UI: extensionmanifest.SettingsUI{Mode: "schema", Layout: "form"},
		Actions: []extensionmanifest.SettingsAction{{ID: "probe", Kind: "provider_probe", Label: LocalizedText{Default: "Probe"}, Placement: "footer"}}, Explicit: true,
	}
	service := NewService(&fakeExtensionStore{items: map[string]Extension{item.ID: item}}, t.TempDir())
	WithSettingsActionRuntime(settingsActionRuntimeFunc(func(ctx context.Context, _ Extension, _ string, _ map[string]string) (SettingsActionProbeResult, error) {
		<-ctx.Done()
		return SettingsActionProbeResult{}, ctx.Err()
	}))(service)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.ExecuteSettingsAction(ctx, extensionManager(), item.ID, "probe", ExecuteSettingsActionInput{}); !errors.Is(err, ErrSettingsActionUnavailable) {
		t.Fatalf("canceled/timeout context must stop the probe, got %v", err)
	}
}

type settingsActionRuntimeFunc func(context.Context, Extension, string, map[string]string) (SettingsActionProbeResult, error)

func (fn settingsActionRuntimeFunc) ProbeSettingsAction(ctx context.Context, extension Extension, slot string, values map[string]string) (SettingsActionProbeResult, error) {
	return fn(ctx, extension, slot, values)
}

type recordingSettingsActionRuntime struct {
	values map[string]string
	result SettingsActionProbeResult
	err    error
}

func (r *recordingSettingsActionRuntime) ProbeSettingsAction(_ context.Context, _ Extension, _ string, values map[string]string) (SettingsActionProbeResult, error) {
	r.values = values
	return r.result, r.err
}

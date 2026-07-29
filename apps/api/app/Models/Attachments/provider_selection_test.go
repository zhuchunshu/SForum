package attachments

import (
	"context"
	"errors"
	"testing"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	storage "github.com/zhuchunshu/sforum/apps/api/app/Support/Storage"
)

type fakeStorageCatalog struct {
	candidates []storage.Candidate
	available  map[string]bool
}

func (f fakeStorageCatalog) ListStorageProviderCandidates(context.Context) ([]storage.Candidate, error) {
	return f.candidates, nil
}

func (f fakeStorageCatalog) IsStorageProviderAvailable(_ context.Context, extensionID string) (bool, error) {
	if f.available == nil {
		return false, nil
	}
	return f.available[extensionID], nil
}

func (f fakeStorageCatalog) StorageProviderSchema(_ context.Context, extensionID, _ string) (storage.ProviderSchema, error) {
	return storage.ProviderSchema{ExtensionID: extensionID, Label: extensionID}, nil
}

func TestSettingsIncludesPluginCandidates(t *testing.T) {
	optionStore := &fakeOptionStore{items: map[string]string{}}
	service := NewService(nil, options.NewServiceWithCacheTTL(optionStore, time.Minute)).
		WithStorageProviderCatalog(fakeStorageCatalog{
			candidates: []storage.Candidate{
				storage.PluginCandidate("acme.store", "Acme Store", "/extensions/acme.store/pages/settings"),
			},
		})

	settings, err := service.Settings(context.Background(), attachmentSettingsActor())
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if settings.ProviderSlot != storage.ProviderSlot {
		t.Fatalf("slot=%s", settings.ProviderSlot)
	}
	found := false
	for _, c := range settings.Candidates {
		if c.Value == "plugin:acme.store" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected plugin candidate, got %#v", settings.Candidates)
	}
	if len(settings.Drivers) != 1 {
		t.Fatalf("drivers=%d", len(settings.Drivers))
	}
}

func TestUpdateSettingsRejectsUnavailablePlugin(t *testing.T) {
	optionStore := &fakeOptionStore{items: map[string]string{}}
	service := NewService(nil, options.NewServiceWithCacheTTL(optionStore, time.Minute)).
		WithStorageProviderCatalog(fakeStorageCatalog{available: map[string]bool{}})

	input := settingsFromValues(map[string]string{}, nil)
	input.Provider = "plugin:missing.store"
	_, err := service.UpdateSettings(context.Background(), attachmentSettingsActor(), input)
	if !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("expected ErrStorageUnavailable, got %v", err)
	}
}

func TestAdapterForPluginSelectionFailClosedWithoutRuntime(t *testing.T) {
	optionStore := &fakeOptionStore{items: map[string]string{
		options.NameAttachmentProvider: "plugin:acme.store",
	}}
	service := NewService(nil, options.NewServiceWithCacheTTL(optionStore, time.Minute)).
		WithStorageProviderCatalog(fakeStorageCatalog{available: map[string]bool{"acme.store": true}})

	settings, err := service.runtimeSettings(context.Background())
	if err != nil {
		t.Fatalf("runtimeSettings: %v", err)
	}
	if settings.Provider != "plugin:acme.store" {
		t.Fatalf("provider=%s", settings.Provider)
	}
	// 有 catalog 无 runtime：仍 fail-closed（E6.2 要求注入 StorageRuntime）。
	_, err = service.adapterForSettings(context.Background(), settings, settings.Provider)
	if !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("plugin path without runtime must fail closed, got %v", err)
	}
}

func TestAdapterForPluginSelectionWithRuntime(t *testing.T) {
	optionStore := &fakeOptionStore{items: map[string]string{
		options.NameAttachmentProvider: "plugin:acme.store",
	}}
	// 最小 stub：只要 NewPluginStorageAdapter 成功即可。
	service := NewService(nil, options.NewServiceWithCacheTTL(optionStore, time.Minute)).
		WithStorageProviderCatalog(fakeStorageCatalog{available: map[string]bool{"acme.store": true}}).
		WithStoragePluginRuntime(extensionsruntime.NewPluginStorageAdapterFactory(stubStorageRuntime{}, 0))

	settings, err := service.runtimeSettings(context.Background())
	if err != nil {
		t.Fatalf("runtimeSettings: %v", err)
	}
	adapter, err := service.adapterForSettings(context.Background(), settings, settings.Provider)
	if err != nil || adapter == nil {
		t.Fatalf("expected plugin adapter, err=%v adapter=%v", err, adapter)
	}
}

// stubStorageRuntime 仅满足接口；本测试不调用 RPC。
type stubStorageRuntime struct{}

func (stubStorageRuntime) StoragePutBegin(context.Context, string, extensionsruntime.StoragePutBeginRequest) (extensionsruntime.StorageSessionResponse, error) {
	return extensionsruntime.StorageSessionResponse{}, nil
}
func (stubStorageRuntime) StoragePutChunk(context.Context, string, extensionsruntime.StoragePutChunkRequest) (extensionsruntime.StorageResult, error) {
	return extensionsruntime.StorageResult{}, nil
}
func (stubStorageRuntime) StorageOpen(context.Context, string, extensionsruntime.StorageOpenRequest) (extensionsruntime.StorageSessionResponse, error) {
	return extensionsruntime.StorageSessionResponse{}, nil
}
func (stubStorageRuntime) StorageGetChunk(context.Context, string, extensionsruntime.StorageGetChunkRequest) (extensionsruntime.StorageGetChunkResponse, error) {
	return extensionsruntime.StorageGetChunkResponse{}, nil
}
func (stubStorageRuntime) StorageClose(context.Context, string, extensionsruntime.StorageCloseRequest) (extensionsruntime.StorageResult, error) {
	return extensionsruntime.StorageResult{}, nil
}
func (stubStorageRuntime) StorageDelete(context.Context, string, extensionsruntime.StorageObjectRequest) (extensionsruntime.StorageResult, error) {
	return extensionsruntime.StorageResult{}, nil
}
func (stubStorageRuntime) StorageStat(context.Context, string, extensionsruntime.StorageStatRequest) (extensionsruntime.StorageStatResponse, error) {
	return extensionsruntime.StorageStatResponse{}, nil
}
func (stubStorageRuntime) StorageExists(context.Context, string, extensionsruntime.StorageExistsRequest) (extensionsruntime.StorageExistsResponse, error) {
	return extensionsruntime.StorageExistsResponse{}, nil
}
func (stubStorageRuntime) StoragePublicURL(context.Context, string, extensionsruntime.StoragePublicURLRequest) (extensionsruntime.StorageURLResponse, error) {
	return extensionsruntime.StorageURLResponse{}, nil
}
func (stubStorageRuntime) StorageSignedURL(context.Context, string, extensionsruntime.StorageSignedURLRequest) (extensionsruntime.StorageURLResponse, error) {
	return extensionsruntime.StorageURLResponse{}, nil
}
func (stubStorageRuntime) StorageProbe(context.Context, string, extensionsruntime.StorageProbeRequest) (extensionsruntime.StorageProbeResponse, error) {
	return extensionsruntime.StorageProbeResponse{}, nil
}

func TestClearStorageProviderSelectionIfMatch(t *testing.T) {
	optionStore := &fakeOptionStore{items: map[string]string{
		options.NameAttachmentProvider: "plugin:acme.store",
	}}
	// seed full defaults so UpdateMany validation passes after switch to local
	opts := options.NewServiceWithCacheTTL(optionStore, time.Minute)
	_ = opts.EnsureDefaults(context.Background())
	optionStore.items[options.NameAttachmentProvider] = "plugin:acme.store"

	service := NewService(nil, opts)
	if err := service.ClearStorageProviderSelectionIfMatch(context.Background(), "acme.store"); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got := optionStore.items[options.NameAttachmentProvider]; got != storage.ProviderLocal {
		t.Fatalf("expected local, got %q", got)
	}
	// 不匹配的扩展 id 不改写
	optionStore.items[options.NameAttachmentProvider] = "plugin:other.store"
	if err := service.ClearStorageProviderSelectionIfMatch(context.Background(), "acme.store"); err != nil {
		t.Fatalf("clear other: %v", err)
	}
	if got := optionStore.items[options.NameAttachmentProvider]; got != "plugin:other.store" {
		t.Fatalf("should keep other selection, got %q", got)
	}
}

func TestEnsureProviderSelectableCoreUnknown(t *testing.T) {
	service := NewService(nil, options.NewServiceWithCacheTTL(&fakeOptionStore{items: map[string]string{}}, time.Minute))
	err := service.ensureProviderSelectable(context.Background(), "not-a-driver")
	if !errors.Is(err, storage.ErrInvalidConfig) {
		t.Fatalf("got %v", err)
	}
	// 权限无关的 actor 仅用于编译
	_ = identity.Actor{}
}

func TestProbeMapsPluginFailureReason(t *testing.T) {
	optionStore := &fakeOptionStore{items: map[string]string{
		options.NameAttachmentProvider: "plugin:acme.store",
	}}
	service := NewService(nil, options.NewServiceWithCacheTTL(optionStore, time.Minute)).
		WithStorageProviderCatalog(fakeStorageCatalog{available: map[string]bool{"acme.store": true}}).
		WithStoragePluginRuntime(extensionsruntime.NewPluginStorageAdapterFactory(failingProbeRuntime{}, 0))

	actor := identity.Actor{
		ID:          1,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionAttachmentSettings: true},
	}
	result, err := service.Probe(context.Background(), actor)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if result.OK || result.Reason != "storage.fs.config" {
		t.Fatalf("expected mapped reason, got %#v", result)
	}
	if result.Message != "root missing" {
		t.Fatalf("message=%q", result.Message)
	}
}

type failingProbeRuntime struct {
	stubStorageRuntime
}

func (failingProbeRuntime) StorageProbe(context.Context, string, extensionsruntime.StorageProbeRequest) (extensionsruntime.StorageProbeResponse, error) {
	// 模拟插件 !OK；Adapter.Probe 转成 StorageRPCError。
	return extensionsruntime.StorageProbeResponse{}, &extensionsruntime.StorageRPCError{
		Reason:  "storage.fs.config",
		Message: "root missing",
	}
}

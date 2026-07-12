package attachments

import (
	"context"
	"errors"
	"testing"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
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
	if len(settings.Drivers) != 5 {
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

func TestAdapterForPluginSelectionFailClosed(t *testing.T) {
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
	_, err = service.adapterForSettings(context.Background(), settings, settings.Provider)
	if !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("plugin path must fail closed until E6.2 RPC, got %v", err)
	}
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

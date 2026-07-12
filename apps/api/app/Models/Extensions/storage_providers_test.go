package extensions

import (
	"context"
	"testing"

	storage "github.com/zhuchunshu/sforum/apps/api/app/Support/Storage"
)

func TestListStorageProviderCandidatesOnlyEnabledSlot(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{}}
	store.items["sforum.smtp"] = Extension{
		ID: "sforum.smtp", Type: TypePlugin, Status: StatusEnabled, Name: "SMTP",
		Manifest: Manifest{
			ID: "sforum.smtp", Type: TypePlugin, Name: "SMTP",
			Providers: []ManifestProvider{{Slot: "mail.provider", Label: "SMTP"}},
		},
	}
	store.items["acme.store"] = Extension{
		ID: "acme.store", Type: TypePlugin, Status: StatusEnabled, Name: "Acme",
		Manifest: Manifest{
			ID: "acme.store", Type: TypePlugin, Name: "Acme Store",
			Providers: []ManifestProvider{{Slot: storage.ProviderSlot, Label: "Acme Object Storage"}},
			Admin:     ManifestAdmin{Entry: "/settings"},
		},
	}
	store.items["disabled.store"] = Extension{
		ID: "disabled.store", Type: TypePlugin, Status: StatusDisabled, Name: "Off",
		Manifest: Manifest{
			ID: "disabled.store", Type: TypePlugin,
			Providers: []ManifestProvider{{Slot: storage.ProviderSlot, Label: "Off"}},
		},
	}
	service := NewService(store, t.TempDir())

	got, err := service.ListStorageProviderCandidates(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].ExtensionID != "acme.store" || got[0].Value != "plugin:acme.store" {
		t.Fatalf("got %#v", got)
	}
	if got[0].SettingsPath == "" {
		t.Fatal("expected settings path")
	}

	ok, err := service.IsStorageProviderAvailable(context.Background(), "acme.store")
	if err != nil || !ok {
		t.Fatalf("available acme: %v %v", ok, err)
	}
	ok, err = service.IsStorageProviderAvailable(context.Background(), "disabled.store")
	if err != nil || ok {
		t.Fatalf("disabled should be unavailable: %v %v", ok, err)
	}
	ok, err = service.IsStorageProviderAvailable(context.Background(), "sforum.smtp")
	if err != nil || ok {
		t.Fatalf("mail plugin is not storage: %v %v", ok, err)
	}
}

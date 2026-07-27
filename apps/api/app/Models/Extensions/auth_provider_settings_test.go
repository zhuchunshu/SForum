package extensions

import (
	"context"
	"errors"
	"testing"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestAuthProviderSettingsConfigured(t *testing.T) {
	item := installedExtension("sforum.auth-github", TypePlugin, ManifestBackend{})
	item.Manifest.Settings = []ManifestSetting{
		{Key: "client_id", Type: "text", Default: ""},
		{Key: "client_secret", Type: "secret", Default: ""},
	}
	item.Manifest.Identity = &ManifestIdentity{
		ContractVersion: "sforum.auth-github.identity@1",
		Providers: []ManifestIdentityProvider{{
			ID: "sforum.auth-github.auth", Kind: "auth", Handler: "sforum.auth-github.identity",
		}},
	}
	store := &fakeExtensionStore{items: map[string]Extension{item.ID: item}, settings: map[string]map[string]string{
		item.ID: {},
	}}
	service := NewService(store, t.TempDir())

	ok, err := service.AuthProviderSettingsConfigured(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("configured check: %v", err)
	}
	if ok {
		t.Fatalf("empty settings must not be configured")
	}

	store.settings[item.ID] = map[string]string{
		"client_id":     "Iv1.example",
		"client_secret": "secret-value",
	}
	ok, err = service.AuthProviderSettingsConfigured(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("configured check after fill: %v", err)
	}
	if !ok {
		t.Fatalf("filled required settings must be configured")
	}
}

func TestCanManageExtensionSettingsAllowsIdentityProviderManageForAuthPlugin(t *testing.T) {
	authPlugin := installedExtension("sforum.auth-github", TypePlugin, ManifestBackend{})
	authPlugin.Manifest.Identity = &ManifestIdentity{
		Providers: []ManifestIdentityProvider{{ID: "sforum.auth-github.auth", Kind: "auth"}},
	}
	plainPlugin := installedExtension("demo.plain", TypePlugin, ManifestBackend{})
	plainPlugin.Manifest.Providers = []ManifestProvider{{Slot: "search.provider", Label: "Search"}}

	actor := identity.Actor{
		ID: 7, Status: identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionIdentityProviderManage: true},
	}
	if !canManageExtensionSettings(actor, authPlugin) {
		t.Fatalf("identity.provider.manage must manage auth provider plugin settings")
	}
	if canManageExtensionSettings(actor, plainPlugin) {
		t.Fatalf("identity.provider.manage must not manage unrelated plugin settings")
	}
}

func TestAdminPageBootstrapAllowsIdentityProviderManageForAuthSettings(t *testing.T) {
	item := installedExtension("sforum.auth-github", TypePlugin, ManifestBackend{})
	item.Manifest.Admin = ManifestAdmin{Pages: []ManifestAdminPage{
		{Path: "/settings", Label: "Settings", View: "settings"},
		{Path: "/about", Label: "About", View: "about"},
	}}
	item.Manifest.Settings = []ManifestSetting{{Key: "client_id", Label: LocalizedText{Default: "Client ID"}, Type: "text", Default: ""}}
	item.Manifest.Identity = &ManifestIdentity{
		Providers: []ManifestIdentityProvider{{ID: "sforum.auth-github.auth", Kind: "auth"}},
	}
	store := &fakeExtensionStore{items: map[string]Extension{item.ID: item}}
	service := NewService(store, t.TempDir())
	actor := identity.Actor{
		ID: 42, Status: identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionIdentityProviderManage: true},
	}

	bootstrap, err := service.AdminPageBootstrap(context.Background(), actor, item.ID, "/settings", "zh-CN")
	if err != nil {
		t.Fatalf("auth settings bootstrap denied: %v", err)
	}
	if bootstrap.Settings == nil {
		t.Fatalf("expected settings document for identity.provider.manage")
	}

	_, err = service.AdminPageBootstrap(context.Background(), actor, item.ID, "/about", "zh-CN")
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("manage-only actor must not read about page, got %v", err)
	}
}

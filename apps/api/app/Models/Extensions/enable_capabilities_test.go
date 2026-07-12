package extensions

import (
	"context"
	"errors"
	"testing"
)

func TestEnableRequiresCapabilityConfirmation(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{
		"demo.plugin": withInstalledPackage(t, installedExtension("demo.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"})),
	}}
	// 无 backend 二进制时用空 backend + settings 推断 settings.own，仍需确认。
	item := store.items["demo.plugin"]
	item.Manifest.Backend = ManifestBackend{}
	item.Manifest.Settings = []ManifestSetting{{Key: "x", Label: LocalizedText{Default: "X"}, Type: "text"}}
	store.items["demo.plugin"] = item

	service := NewServiceWithRuntime(store, t.TempDir(), &fakeRuntimeManager{}, nil)

	_, err := service.Enable(context.Background(), extensionManager(), "demo.plugin", EnableInput{})
	if !errors.Is(err, ErrCapabilityConfirmationRequired) {
		t.Fatalf("expected confirmation required, got %v", err)
	}

	enabled, err := service.Enable(context.Background(), extensionManager(), "demo.plugin", EnableInput{ConfirmCapabilities: true})
	if err != nil {
		t.Fatalf("enable with confirm: %v", err)
	}
	if enabled.Status != StatusEnabled {
		t.Fatalf("status = %s", enabled.Status)
	}
	if len(enabled.CapabilityGrants) == 0 {
		t.Fatal("expected capability grants on enabled extension")
	}
}

func TestEnableRestartSkipsConfirmation(t *testing.T) {
	item := withInstalledPackage(t, installedExtension("demo.plugin", TypePlugin, ManifestBackend{}))
	item.Status = StatusEnabled
	item.Manifest.Settings = []ManifestSetting{{Key: "x", Label: LocalizedText{Default: "X"}, Type: "text"}}
	store := &fakeExtensionStore{items: map[string]Extension{item.ID: item}}
	service := NewServiceWithRuntime(store, t.TempDir(), &fakeRuntimeManager{}, nil)

	// 已启用：不传 confirm 也应成功（重启路径）。
	_, err := service.Enable(context.Background(), extensionManager(), "demo.plugin", EnableInput{})
	if err != nil {
		t.Fatalf("restart enable: %v", err)
	}
}

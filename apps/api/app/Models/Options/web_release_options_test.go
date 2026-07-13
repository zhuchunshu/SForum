package options

import (
	"context"
	"errors"
	"testing"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestWebReleaseTypecheckFailDefaultAndUpdate(t *testing.T) {
	store := &memoryOptionStore{values: map[string]string{}}
	service := NewService(store)
	ctx := context.Background()
	if err := service.EnsureDefaults(ctx); err != nil {
		t.Fatalf("EnsureDefaults: %v", err)
	}
	fail, err := service.WebReleaseTypecheckFail(ctx)
	if err != nil {
		t.Fatalf("WebReleaseTypecheckFail: %v", err)
	}
	if fail {
		t.Fatal("default must be non-blocking")
	}

	manager := identity.Actor{
		ID: 1, Status: identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionExtensionReleaseManage: true},
	}
	if _, err := service.UpdateMany(ctx, manager, []UpdateInput{{
		Name: NameWebReleaseTypecheckFail, Value: "enabled",
	}}); err != nil {
		t.Fatalf("UpdateMany: %v", err)
	}
	fail, err = service.WebReleaseTypecheckFail(ctx)
	if err != nil || !fail {
		t.Fatalf("expected enabled hard-fail, got fail=%v err=%v", fail, err)
	}
	if mode, err := service.WebReleaseTypecheckMode(ctx); err != nil || mode != "block" {
		t.Fatalf("legacy option must synchronize block mode, got mode=%q err=%v", mode, err)
	}

	if _, err := service.UpdateMany(ctx, manager, []UpdateInput{{
		Name: NameWebReleaseTypecheckMode, Value: "off",
	}}); err != nil {
		t.Fatalf("update mode: %v", err)
	}
	if mode, err := service.WebReleaseTypecheckMode(ctx); err != nil || mode != "off" {
		t.Fatalf("expected off mode, got mode=%q err=%v", mode, err)
	}
	if fail, err := service.WebReleaseTypecheckFail(ctx); err != nil || fail {
		t.Fatalf("mode update must synchronize legacy option, got fail=%v err=%v", fail, err)
	}

	// 无 release 权限不可改。
	denied := identity.Actor{
		ID: 2, Status: identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionSettingsSiteManage: true},
	}
	if _, err := service.UpdateMany(ctx, denied, []UpdateInput{{
		Name: NameWebReleaseTypecheckFail, Value: "disabled",
	}}); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

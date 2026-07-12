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

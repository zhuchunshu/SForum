package jobs

import (
	"context"
	"errors"
	"testing"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestServiceEnforcesViewAndManagePermissionsBeforeDataAccess(t *testing.T) {
	service := NewService(nil, nil)
	actor := identity.Actor{ID: 9, Status: identity.UserStatusActive, Permissions: map[string]bool{}}
	if _, err := service.Overview(context.Background(), actor); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("overview permission: %v", err)
	}
	if _, err := service.List(context.Background(), actor, ListInput{}); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("list permission: %v", err)
	}
	if _, err := service.Detail(context.Background(), actor, 1); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("detail permission: %v", err)
	}
	if _, err := service.Retry(context.Background(), actor, 1); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("retry permission: %v", err)
	}
	if _, err := service.Cancel(context.Background(), actor, 1); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("cancel permission: %v", err)
	}
	if err := service.SetQueuePaused(context.Background(), actor, "default", true); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("pause permission: %v", err)
	}
}

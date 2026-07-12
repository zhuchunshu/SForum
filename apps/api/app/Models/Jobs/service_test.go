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
	if _, err := service.Schedules(context.Background(), actor); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("schedules permission: %v", err)
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

func TestServiceSchedulesReturnsCoreCatalog(t *testing.T) {
	service := NewService(nil, nil)
	actor := identity.Actor{
		ID:     1,
		Status: identity.UserStatusActive,
		Permissions: map[string]bool{
			identity.PermissionJobsView: true,
		},
	}
	items, err := service.Schedules(context.Background(), actor)
	if err != nil {
		t.Fatalf("schedules: %v", err)
	}
	if len(items) != 4 {
		t.Fatalf("expected 4 core schedules, got %d", len(items))
	}
	ids := map[string]bool{}
	for _, item := range items {
		ids[item.ID] = true
		if item.JobKind == "" || item.Owner == "" || item.IntervalSeconds <= 0 {
			t.Fatalf("incomplete schedule: %+v", item)
		}
	}
	for _, id := range []string{
		"identity.cleanup_sessions",
		"extension.web_release_cleanup",
		"attachments.cleanup_orphans",
		"audit.cleanup_events",
	} {
		if !ids[id] {
			t.Fatalf("missing schedule %s", id)
		}
	}
}

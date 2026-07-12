package extensions

import (
	"context"
	"errors"
	"testing"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestWebReleaseAdminServiceKeepsPolicyChecksAuthoritative(t *testing.T) {
	store := &fakeWebReleaseAdminStore{}
	commands := &fakeWebReleaseCommander{}
	service := NewWebReleaseAdminService(store, commands)
	denied := identity.Actor{ID: 2, Status: identity.UserStatusActive}

	if _, err := service.List(context.Background(), denied, WebReleaseListInput{}); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected list permission denial, got %v", err)
	}
	if _, err := service.Detail(context.Background(), denied, 1); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected detail permission denial, got %v", err)
	}
	if _, err := service.Rebuild(context.Background(), denied); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected rebuild permission denial, got %v", err)
	}
	if _, err := service.Retry(context.Background(), denied, 1); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected retry permission denial, got %v", err)
	}
	if _, err := service.Rollback(context.Background(), denied, 1); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected rollback permission denial, got %v", err)
	}
	if store.calls != 0 || commands.calls != 0 {
		t.Fatalf("denied actor reached persistence: store=%d commands=%d", store.calls, commands.calls)
	}

	manager := identity.Actor{
		ID: 3, Status: identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionExtensionManage: true},
	}
	if _, err := service.List(context.Background(), manager, WebReleaseListInput{}); err != nil {
		t.Fatalf("manager list: %v", err)
	}
	if _, err := service.Detail(context.Background(), manager, 1); err != nil {
		t.Fatalf("manager detail: %v", err)
	}
	rebuild, err := service.Rebuild(context.Background(), manager)
	if err != nil {
		t.Fatalf("manager rebuild: %v", err)
	}
	if rebuild.WebRelease.ID != 11 || !rebuild.Queued {
		t.Fatalf("unexpected rebuild operation: %#v", rebuild)
	}
	if commands.lastPlan.TriggerKind != WebReleaseTriggerRebuild || commands.lastPlan.RequestedBy != manager.ID {
		t.Fatalf("unexpected rebuild plan: %#v", commands.lastPlan)
	}
	if _, err := service.Retry(context.Background(), manager, 1); err != nil {
		t.Fatalf("manager retry: %v", err)
	}
	if _, err := service.Rollback(context.Background(), manager, 1); err != nil {
		t.Fatalf("manager rollback: %v", err)
	}
}

type fakeWebReleaseAdminStore struct {
	calls int
}

func (s *fakeWebReleaseAdminStore) ListWebReleases(context.Context, WebReleaseListInput) (WebReleasePage, error) {
	s.calls++
	return WebReleasePage{}, nil
}

func (s *fakeWebReleaseAdminStore) WebRelease(context.Context, int64) (WebReleaseDetail, error) {
	s.calls++
	return WebReleaseDetail{}, nil
}

type fakeWebReleaseCommander struct {
	calls    int
	lastPlan PlanWebReleaseInput
}

func (s *fakeWebReleaseCommander) PlanAndQueue(_ context.Context, input QueueWebReleaseInput) (WebReleaseQueueResult, error) {
	s.calls++
	s.lastPlan = input.Plan
	return WebReleaseQueueResult{Release: WebRelease{ID: 11, TriggerKind: input.Plan.TriggerKind}, Created: true}, nil
}

func (s *fakeWebReleaseCommander) Retry(context.Context, int64, int64) (WebReleaseQueueResult, error) {
	s.calls++
	return WebReleaseQueueResult{Release: WebRelease{ID: 2}}, nil
}

func (s *fakeWebReleaseCommander) Rollback(context.Context, int64, int64) (WebReleaseQueueResult, error) {
	s.calls++
	return WebReleaseQueueResult{Release: WebRelease{ID: 3}}, nil
}

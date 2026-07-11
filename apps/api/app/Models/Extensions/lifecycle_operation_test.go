package extensions

import (
	"context"
	"errors"
	"testing"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestFrontendRequiresWebRelease(t *testing.T) {
	tests := []struct {
		state string
		want  bool
	}{
		{FrontendTrustNone, false},
		{FrontendTrustRequired, false},
		{FrontendTrustInvalidated, false},
		{FrontendTrustTrusted, true},
		{FrontendTrustSourceTrusted, true},
		{FrontendTrustRevocationPending, true},
	}
	for _, test := range tests {
		if got := frontendRequiresWebRelease(FrontendStatus{TrustState: test.state}); got != test.want {
			t.Fatalf("state %s: expected %v, got %v", test.state, test.want, got)
		}
	}
}

// TestDisableOperationRequiresPluginManageWithWebRelease 回归：仅有 release.manage
// 不能禁用带受信任前端的插件；须与 Enable 一样同时具备 plugin.manage。
func TestDisableOperationRequiresPluginManageWithWebRelease(t *testing.T) {
	extension := plannerPluginFixture(t, "demo.plugin", SourceUploaded, StatusEnabled)
	store := newFakeExtensionStore(map[string]Extension{extension.ID: extension})
	service := NewService(store, t.TempDir())
	WithWebReleaseLifecycle(
		staticFrontendLifecycle{status: FrontendStatus{TrustState: FrontendTrustTrusted}},
		&fakeFrontendReleaseManager{},
	)(service)

	releaseOnly := identity.Actor{
		ID:     3,
		Status: identity.UserStatusActive,
		Permissions: map[string]bool{
			identity.PermissionExtensionReleaseManage: true,
		},
	}
	_, err := service.DisableOperation(context.Background(), releaseOnly, extension.ID)
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied for release-only actor, got %v", err)
	}
}

type staticFrontendLifecycle struct {
	status FrontendStatus
}

func (s staticFrontendLifecycle) Frontend(context.Context, identity.Actor, string) (FrontendStatus, error) {
	return s.status, nil
}

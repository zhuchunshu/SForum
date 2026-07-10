package extensions

import (
	"context"
	"errors"
	"testing"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestFrontendServiceRequiresActiveSuperAdminForTrustMutations(t *testing.T) {
	extension := plannerPluginFixture(t, "secure.plugin", SourceUploaded, StatusDisabled)
	service := NewFrontendService(
		&fakeFrontendExtensionReader{item: extension},
		&fakeFrontendLifecycleStore{},
		&fakeFrontendReleaseManager{},
		&fakeFrontendActiveReader{activeErr: ErrWebReleaseNotFound},
		plannerHostFixture(),
	)
	actors := []identity.Actor{
		{ID: 2, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionExtensionManage: true}},
		{ID: 3, Status: identity.UserStatusDisabled, RoleKeys: []string{identity.RoleSuperAdmin}},
	}
	for _, actor := range actors {
		if _, err := service.Grant(context.Background(), actor, extension.ID, GrantFrontendInput{PackageDigest: extension.PackageDigest}); !errors.Is(err, identity.ErrPermissionDenied) {
			t.Fatalf("actor %#v unexpectedly granted trust: %v", actor, err)
		}
		if _, err := service.Revoke(context.Background(), actor, extension.ID); !errors.Is(err, identity.ErrPermissionDenied) {
			t.Fatalf("actor %#v unexpectedly revoked trust: %v", actor, err)
		}
		if _, err := service.RestoreDefaults(context.Background(), actor); !errors.Is(err, identity.ErrPermissionDenied) {
			t.Fatalf("actor %#v unexpectedly restored defaults: %v", actor, err)
		}
	}
}

func TestFrontendServiceGrantRechecksDigestAndQueuesOnlyEnabledPlugin(t *testing.T) {
	for _, status := range []string{StatusDisabled, StatusEnabled} {
		t.Run(status, func(t *testing.T) {
			extension := plannerPluginFixture(t, "grant.plugin", SourceUploaded, status)
			trust := &fakeFrontendLifecycleStore{}
			releases := &fakeFrontendReleaseManager{result: WebReleaseQueueResult{Release: WebRelease{ID: 9}, Created: true}}
			service := NewFrontendService(
				&fakeFrontendExtensionReader{item: extension},
				trust,
				releases,
				&fakeFrontendActiveReader{activeErr: ErrWebReleaseNotFound},
				plannerHostFixture(),
			)

			operation, err := service.Grant(context.Background(), frontendSuperAdmin(), extension.ID, GrantFrontendInput{
				PackageDigest: extension.PackageDigest,
			})
			if err != nil {
				t.Fatalf("grant frontend trust: %v", err)
			}
			if trust.created.PackageDigest != extension.PackageDigest || operation.Frontend == nil || operation.Frontend.TrustState != FrontendTrustTrusted {
				t.Fatalf("grant was not persisted exactly: operation=%#v input=%#v", operation, trust.created)
			}
			wantQueue := status == StatusEnabled
			if releases.calls != boolInt(wantQueue) || operation.Queued != wantQueue {
				t.Fatalf("unexpected release queue state: calls=%d operation=%#v", releases.calls, operation)
			}
			if wantQueue && (releases.last.Plan.TriggerKind != WebReleaseTriggerTrustGrant || releases.last.Plan.ReloadMode != WebReleaseReloadPrompt) {
				t.Fatalf("unexpected grant release input: %#v", releases.last)
			}
		})
	}
}

func TestFrontendServiceGrantRejectsStaleDigest(t *testing.T) {
	extension := plannerPluginFixture(t, "stale.plugin", SourceUploaded, StatusDisabled)
	service := NewFrontendService(
		&fakeFrontendExtensionReader{item: extension},
		&fakeFrontendLifecycleStore{},
		&fakeFrontendReleaseManager{},
		&fakeFrontendActiveReader{activeErr: ErrWebReleaseNotFound},
		plannerHostFixture(),
	)

	_, err := service.Grant(context.Background(), frontendSuperAdmin(), extension.ID, GrantFrontendInput{PackageDigest: "stale"})
	if !errors.Is(err, ErrWebReleasePackageChanged) {
		t.Fatalf("expected stale digest error, got %v", err)
	}
}

func TestFrontendServiceRevokeFinalizesAbsentFrontendAndQueuesActiveFrontend(t *testing.T) {
	for _, active := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent", true: "active"}[active], func(t *testing.T) {
			extension := plannerPluginFixture(t, "revoke.plugin", SourceUploaded, StatusEnabled)
			grant := frontendGrantForExtension(extension)
			trust := &fakeFrontendLifecycleStore{grant: grant}
			releases := &fakeFrontendReleaseManager{result: WebReleaseQueueResult{Release: WebRelease{ID: 19}, Created: true}}
			activeReader := &fakeFrontendActiveReader{activeErr: ErrWebReleaseNotFound}
			if active {
				activeReader.active = WebRelease{ID: 7, Status: WebReleaseActive}
				activeReader.detail = WebReleaseDetail{
					WebRelease: activeReader.active,
					Extensions: []WebReleaseExtension{{
						ExtensionID:      extension.ID,
						ExtensionVersion: extension.Version,
						PackageDigest:    extension.PackageDigest,
					}},
				}
				activeReader.activeErr = nil
			}
			service := NewFrontendService(
				&fakeFrontendExtensionReader{item: extension},
				trust,
				releases,
				activeReader,
				plannerHostFixture(),
			)

			operation, err := service.Revoke(context.Background(), frontendSuperAdmin(), extension.ID)
			if err != nil {
				t.Fatalf("revoke frontend trust: %v", err)
			}
			if active {
				if releases.calls != 1 || trust.finalized != 0 || !operation.Queued || releases.last.Plan.ReloadMode != WebReleaseReloadForce {
					t.Fatalf("active frontend did not queue safe release: operation=%#v trust=%#v release=%#v", operation, trust, releases)
				}
			} else if releases.calls != 0 || trust.finalized != 1 || operation.Queued {
				t.Fatalf("absent frontend did not finalize immediately: operation=%#v trust=%#v release=%#v", operation, trust, releases)
			}
		})
	}
}

func TestFrontendServiceRestoreDefaultsKeepsLifecycleEffectsEmpty(t *testing.T) {
	extension := plannerPluginFixture(t, "restore.plugin", SourceUploaded, StatusEnabled)
	grant := frontendGrantForExtension(extension)
	trust := &fakeFrontendLifecycleStore{all: []FrontendTrustGrant{grant}}
	active := WebRelease{ID: 7, Status: WebReleaseActive}
	activeReader := &fakeFrontendActiveReader{
		active: active,
		detail: WebReleaseDetail{WebRelease: active, Extensions: []WebReleaseExtension{{
			ExtensionID:      extension.ID,
			ExtensionVersion: extension.Version,
			PackageDigest:    extension.PackageDigest,
		}}},
	}
	releases := &fakeFrontendReleaseManager{result: WebReleaseQueueResult{Release: WebRelease{ID: 8}, Created: true}}
	service := NewFrontendService(
		&fakeFrontendExtensionReader{item: extension},
		trust,
		releases,
		activeReader,
		plannerHostFixture(),
	)

	operation, err := service.RestoreDefaults(context.Background(), frontendSuperAdmin())
	if err != nil {
		t.Fatalf("restore frontend defaults: %v", err)
	}
	if !operation.Queued || releases.calls != 1 || releases.last.Plan.TriggerKind != WebReleaseTriggerRestore || releases.last.Plan.ReloadMode != WebReleaseReloadForce {
		t.Fatalf("restore did not queue force release: operation=%#v release=%#v", operation, releases)
	}
	if len(releases.last.Effects) != 0 {
		t.Fatalf("restore defaults changed backend lifecycle: %#v", releases.last.Effects)
	}
}

func frontendSuperAdmin() identity.Actor {
	return identity.Actor{ID: 1, Status: identity.UserStatusActive, RoleKeys: []string{identity.RoleSuperAdmin}}
}

func frontendGrantForExtension(extension Extension) FrontendTrustGrant {
	return FrontendTrustGrant{
		ID:               1,
		ExtensionID:      extension.ID,
		ExtensionVersion: extension.Version,
		PackageDigest:    extension.PackageDigest,
		APIVersion:       extension.Manifest.Frontend.Admin.APIVersion,
		GrantedByUserID:  1,
		GrantedAt:        time.Now(),
	}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

type fakeFrontendExtensionReader struct {
	item Extension
	err  error
}

func (r *fakeFrontendExtensionReader) Get(_ context.Context, id string) (Extension, error) {
	if r.err != nil {
		return Extension{}, r.err
	}
	if id != r.item.ID {
		return Extension{}, ErrExtensionNotFound
	}
	return r.item, nil
}

type fakeFrontendLifecycleStore struct {
	grant     FrontendTrustGrant
	all       []FrontendTrustGrant
	created   FrontendTrustGrantInput
	finalized int
}

func (s *fakeFrontendLifecycleStore) FrontendGrant(context.Context, string, string, string) (FrontendTrustGrant, error) {
	if s.grant.ID == 0 {
		return FrontendTrustGrant{}, ErrFrontendGrantNotFound
	}
	return s.grant, nil
}

func (s *fakeFrontendLifecycleStore) LiveFrontendGrants(context.Context, string) ([]FrontendTrustGrant, error) {
	return append([]FrontendTrustGrant(nil), s.all...), nil
}

func (s *fakeFrontendLifecycleStore) CreateFrontendGrant(_ context.Context, input FrontendTrustGrantInput) (FrontendTrustGrant, error) {
	s.created = input
	s.grant = FrontendTrustGrant{
		ID:                 1,
		ExtensionID:        input.ExtensionID,
		ExtensionVersion:   input.ExtensionVersion,
		PackageDigest:      input.PackageDigest,
		APIVersion:         input.APIVersion,
		ContributionPoints: input.ContributionPoints,
		ComponentIDs:       input.ComponentIDs,
		GrantedByUserID:    input.GrantedByUserID,
		GrantedAt:          time.Now(),
	}
	return s.grant, nil
}

func (s *fakeFrontendLifecycleStore) RequestFrontendRevocation(_ context.Context, input FrontendRevocationInput) (FrontendTrustGrant, error) {
	now := time.Now()
	s.grant.RevocationRequestedAt = &now
	s.grant.RevocationRequestedByUserID = input.RequestedByUserID
	return s.grant, nil
}

func (s *fakeFrontendLifecycleStore) RequestAllFrontendRevocations(_ context.Context, actorID int64) ([]FrontendTrustGrant, error) {
	items := append([]FrontendTrustGrant(nil), s.all...)
	for index := range items {
		now := time.Now()
		items[index].RevocationRequestedAt = &now
		items[index].RevocationRequestedByUserID = actorID
	}
	s.all = items
	return items, nil
}

func (s *fakeFrontendLifecycleStore) FinalizeFrontendRevocation(_ context.Context, input FrontendFinalizeInput) (FrontendTrustGrant, error) {
	s.finalized++
	now := time.Now()
	s.grant.RevokedAt = &now
	return s.grant, nil
}

func (s *fakeFrontendLifecycleStore) FinalizeFrontendRevocations(context.Context, int64) error {
	return nil
}

type fakeFrontendReleaseManager struct {
	calls  int
	last   QueueWebReleaseInput
	result WebReleaseQueueResult
	err    error
}

func (m *fakeFrontendReleaseManager) PlanAndQueue(_ context.Context, input QueueWebReleaseInput) (WebReleaseQueueResult, error) {
	m.calls++
	m.last = input
	return m.result, m.err
}

type fakeFrontendActiveReader struct {
	active    WebRelease
	activeErr error
	detail    WebReleaseDetail
	detailErr error
}

func (r *fakeFrontendActiveReader) ActiveWebRelease(context.Context) (WebRelease, error) {
	return r.active, r.activeErr
}

func (r *fakeFrontendActiveReader) WebRelease(context.Context, int64) (WebReleaseDetail, error) {
	return r.detail, r.detailErr
}

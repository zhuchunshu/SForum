package extensions

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	audit "github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
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
		if _, err := service.Grant(context.Background(), actor, extension.ID, GrantFrontendInput{PackageDigest: extension.AdminFrontendDigest}); !errors.Is(err, identity.ErrPermissionDenied) {
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
				PackageDigest: extension.AdminFrontendDigest,
			})
			if err != nil {
				t.Fatalf("grant frontend trust: %v", err)
			}
			if trust.created.PackageDigest != extension.PackageDigest || trust.created.AdminFrontendDigest != extension.AdminFrontendDigest || operation.Frontend == nil || operation.Frontend.TrustState != FrontendTrustTrusted {
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

func TestFrontendServicePrebuiltComponentRequiresExactConfirmationAndNeverQueuesRelease(t *testing.T) {
	extension := prebuiltFrontendFixture(t, SourceUploaded)
	trust := &fakeFrontendLifecycleStore{}
	releases := &fakeFrontendReleaseManager{}
	auditor := &recordingAuditor{}
	service := NewFrontendService(
		&fakeFrontendExtensionReader{item: extension}, trust, releases,
		&fakeFrontendActiveReader{activeErr: ErrWebReleaseNotFound}, plannerHostFixture(),
	).WithAuditor(auditor)
	input := GrantFrontendInput{PackageDigest: extension.AdminFrontendDigest}
	if _, err := service.Grant(context.Background(), frontendSuperAdmin(), extension.ID, input); !errors.Is(err, ErrFrontendTrustUnavailable) {
		t.Fatalf("missing confirmation must be rejected, got %v", err)
	}
	component := *prebuiltSettingsComponent(extension)
	challenge, err := service.Challenge(context.Background(), frontendSuperAdmin(), extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	input.Confirmation = &FrontendTrustConfirmation{
		ChallengeID: challenge.ChallengeID, Code: challenge.Code,
		ExtensionID: extension.ID, Version: extension.Version, Digest: extension.AdminFrontendDigest,
		APIVersion: component.APIVersion, ComponentID: component.ID, Phrase: extension.ID, Acknowledged: true,
	}
	otherAdmin := frontendSuperAdmin()
	otherAdmin.ID = 2
	if _, err := service.Grant(context.Background(), otherAdmin, extension.ID, input); !errors.Is(err, ErrFrontendTrustUnavailable) {
		t.Fatalf("challenge must be actor-bound, got %v", err)
	}
	challenge, err = service.Challenge(context.Background(), frontendSuperAdmin(), extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	input.Confirmation.ChallengeID = challenge.ChallengeID
	input.Confirmation.Code = challenge.Code
	operation, err := service.Grant(context.Background(), frontendSuperAdmin(), extension.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if operation.Queued || releases.calls != 0 || operation.Frontend == nil || operation.Frontend.Kind != AdminFrontendKindPrebuiltComponent || operation.Frontend.BuildRequired {
		t.Fatalf("prebuilt grant must not queue/build: operation=%#v releases=%d", operation, releases.calls)
	}
	if trust.created.AdminFrontendDigest != extension.AdminFrontendDigest || trust.created.APIVersion != component.APIVersion || len(trust.created.ComponentIDs) != 1 || trust.created.ComponentIDs[0] != component.ID {
		t.Fatalf("grant was not bound to exact component identity: %#v", trust.created)
	}
	if !auditor.hasAction(audit.ActionExtensionFrontendGrant) {
		t.Fatal("prebuilt trust grant must be audited")
	}
	if _, err := service.Grant(context.Background(), frontendSuperAdmin(), extension.ID, input); !errors.Is(err, ErrFrontendTrustUnavailable) {
		t.Fatalf("confirmation challenge must be one-use, got %v", err)
	}
}

func TestFrontendServicePrebuiltAssetIsAllowlistedImmutableAndTrustBound(t *testing.T) {
	extension := prebuiltFrontendFixture(t, SourceUploaded)
	component := *prebuiltSettingsComponent(extension)
	grant := FrontendTrustGrant{
		ID: 1, ExtensionID: extension.ID, ExtensionVersion: extension.Version,
		PackageDigest: extension.PackageDigest, AdminFrontendDigest: extension.AdminFrontendDigest,
		APIVersion: component.APIVersion, ComponentIDs: []string{component.ID}, GrantedAt: time.Now(),
	}
	service := NewFrontendService(
		&fakeFrontendExtensionReader{item: extension}, &fakeFrontendLifecycleStore{grant: grant},
		&fakeFrontendReleaseManager{}, &fakeFrontendActiveReader{activeErr: ErrWebReleaseNotFound}, plannerHostFixture(),
	)
	asset, err := service.Asset(context.Background(), frontendSuperAdmin(), extension.ID, extension.AdminFrontendDigest, "entry")
	if err != nil {
		t.Fatal(err)
	}
	if string(asset.Body) != "export const apiVersion = 1\nexport function mount() {}\n" || asset.ContentType != "application/javascript; charset=utf-8" || asset.ETag == "" {
		t.Fatalf("unexpected asset: %#v", asset)
	}
	if _, err := service.Asset(context.Background(), frontendSuperAdmin(), extension.ID, extension.AdminFrontendDigest, "../../backend"); !errors.Is(err, ErrFrontendTrustUnavailable) {
		t.Fatalf("non-allowlisted asset must be rejected, got %v", err)
	}
	if err := os.WriteFile(filepath.Join(extension.PackagePath, "frontend/admin/dist/settings.mjs"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Asset(context.Background(), frontendSuperAdmin(), extension.ID, extension.AdminFrontendDigest, "entry"); !errors.Is(err, ErrWebReleasePackageChanged) {
		t.Fatalf("mutated bytes must invalidate immutable URL, got %v", err)
	}
}

func TestFrontendServiceBuiltinThemePrebuiltComponentUsesSourceTrust(t *testing.T) {
	extension := prebuiltFrontendFixture(t, SourceBuiltin)
	extension.Type = TypeTheme
	extension.Manifest.Type = TypeTheme
	extension.IsSystem = true
	extension.IsDeletable = false
	service := NewFrontendService(
		&fakeFrontendExtensionReader{item: extension}, &fakeFrontendLifecycleStore{},
		&fakeFrontendReleaseManager{}, &fakeFrontendActiveReader{activeErr: ErrWebReleaseNotFound}, plannerHostFixture(),
	)
	status, err := service.Frontend(context.Background(), frontendSuperAdmin(), extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.TrustState != FrontendTrustSourceTrusted || status.Kind != AdminFrontendKindPrebuiltComponent {
		t.Fatalf("builtin theme component should be source trusted: %#v", status)
	}
}

func TestFrontendServicePrebuiltUpgradeInvalidatesOldGrantWithoutBlockingSchemaFallback(t *testing.T) {
	extension := prebuiltFrontendFixture(t, SourceUploaded)
	component := *prebuiltSettingsComponent(extension)
	oldGrant := FrontendTrustGrant{
		ID: 1, ExtensionID: extension.ID, ExtensionVersion: extension.Version,
		PackageDigest: extension.PackageDigest, AdminFrontendDigest: extension.AdminFrontendDigest,
		APIVersion: component.APIVersion, ComponentIDs: []string{component.ID}, GrantedAt: time.Now(),
	}
	if err := os.WriteFile(filepath.Join(extension.PackagePath, component.Entry), []byte("export const apiVersion = 1\nexport function mount() { return () => {} }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := ComputeAdminFrontendDigest(extension.Manifest, extension.PackagePath)
	if err != nil {
		t.Fatal(err)
	}
	extension.Version = "1.0.1"
	extension.Manifest.Version = extension.Version
	extension.AdminFrontendDigest = changed
	service := NewFrontendService(
		&fakeFrontendExtensionReader{item: extension}, &fakeFrontendLifecycleStore{all: []FrontendTrustGrant{oldGrant}},
		&fakeFrontendReleaseManager{}, &fakeFrontendActiveReader{activeErr: ErrWebReleaseNotFound}, plannerHostFixture(),
	)
	status, err := service.Frontend(context.Background(), frontendSuperAdmin(), extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.TrustState != FrontendTrustInvalidated || status.Kind != AdminFrontendKindPrebuiltComponent || status.BuildRequired {
		t.Fatalf("changed component must fall back pending reconfirmation without a build: %#v", status)
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
						ExtensionID:         extension.ID,
						ExtensionVersion:    extension.Version,
						PackageDigest:       extension.PackageDigest,
						AdminFrontendDigest: extension.AdminFrontendDigest,
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
			ExtensionID:         extension.ID,
			ExtensionVersion:    extension.Version,
			PackageDigest:       extension.PackageDigest,
			AdminFrontendDigest: extension.AdminFrontendDigest,
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

func prebuiltFrontendFixture(t *testing.T, source string) Extension {
	t.Helper()
	root := t.TempDir()
	entry := "frontend/admin/dist/settings.mjs"
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, entry)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, entry), []byte("export const apiVersion = 1\nexport function mount() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{
		ID: "prebuilt.plugin", Name: "Prebuilt", Version: "1.0.0", Type: TypePlugin,
		Settings: []ManifestSetting{{Key: "title", Type: "text", Label: LocalizedText{Default: "Title"}}},
		SettingsDocument: SettingsDocument{SchemaVersion: 1, Explicit: true, UI: SettingsUI{
			Mode: "component", Layout: "form", Component: &SettingsComponent{ID: "settings", APIVersion: 1, Entry: entry},
		}},
	}
	manifest.SettingsDocument.Fields = manifest.Settings
	adminDigest, err := ComputeAdminFrontendDigest(manifest, root)
	if err != nil {
		t.Fatal(err)
	}
	packageDigest, err := extensionpackage.DigestTree(root)
	if err != nil {
		t.Fatal(err)
	}
	return Extension{
		ID: manifest.ID, Name: manifest.Name, Version: manifest.Version, Type: TypePlugin,
		Status: StatusEnabled, Source: source, IsDeletable: true, Manifest: manifest,
		PackagePath: root, PackageDigest: packageDigest, AdminFrontendDigest: adminDigest,
	}
}

func frontendGrantForExtension(extension Extension) FrontendTrustGrant {
	return FrontendTrustGrant{
		ID:                  1,
		ExtensionID:         extension.ID,
		ExtensionVersion:    extension.Version,
		PackageDigest:       extension.PackageDigest,
		AdminFrontendDigest: extension.AdminFrontendDigest,
		APIVersion:          extension.Manifest.Frontend.Admin.APIVersion,
		GrantedByUserID:     1,
		GrantedAt:           time.Now(),
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
		ID:                  1,
		ExtensionID:         input.ExtensionID,
		ExtensionVersion:    input.ExtensionVersion,
		PackageDigest:       input.PackageDigest,
		AdminFrontendDigest: input.AdminFrontendDigest,
		APIVersion:          input.APIVersion,
		ContributionPoints:  input.ContributionPoints,
		ComponentIDs:        input.ComponentIDs,
		GrantedByUserID:     input.GrantedByUserID,
		GrantedAt:           time.Now(),
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

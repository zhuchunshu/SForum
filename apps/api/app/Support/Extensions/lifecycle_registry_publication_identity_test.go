package extensionsruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
)

func TestBuildLifecycleIdentityPublicationMapsCanonicalManifest(t *testing.T) {
	extension := lifecycleIdentityExtension("1.0.0", 41, "")
	extension.Manifest.PermissionDefinitions[0].RecommendedRoles = []string{"operator", "member"}
	publication, err := buildLifecycleIdentityPublication(extension, extensions.LifecycleRuntimeBinding{})
	if err != nil {
		t.Fatal(err)
	}
	if publication == nil || publication.Artifact.RuntimeInstanceID != "" || len(publication.Permissions) != 1 ||
		len(publication.Permissions[0].RecommendedRoles) != 2 || publication.Permissions[0].RecommendedRoles[0] != "member" ||
		publication.Identity == nil || len(publication.Identity.UserFields) != 1 {
		t.Fatalf("publication = %#v", publication)
	}

	// Returned material owns its slices and cannot be changed through Manifest aliases.
	extension.Manifest.PermissionDefinitions[0].RecommendedRoles[0] = "changed"
	if publication.Permissions[0].RecommendedRoles[0] != "member" {
		t.Fatalf("publication aliases manifest: %#v", publication.Permissions[0])
	}
}

func TestBuildLifecycleIdentityPublicationRequiresRuntimeOnlyForExecutableIdentity(t *testing.T) {
	inert := lifecycleIdentityExtension("1.0.0", 42, "")
	inert.Manifest.Identity.Providers = []extensionmanifest.ManifestIdentityProvider{{
		ID: inert.ID + ".provider", ContractVersion: inert.ID + ".provider@1",
		Kind: "auth", Handler: "legacy.auth",
	}}
	if publication, err := buildLifecycleIdentityPublication(inert, extensions.LifecycleRuntimeBinding{}); err != nil ||
		publication == nil || publication.Artifact.RuntimeInstanceID != "" || len(publication.Identity.Providers) != 1 {
		t.Fatalf("inert publication = %#v, %v", publication, err)
	}

	executable := lifecycleIdentityExtension("1.0.0", 43, "risk")
	if _, err := buildLifecycleIdentityPublication(executable, extensions.LifecycleRuntimeBinding{}); !errors.Is(err, ErrLifecycleRegistryPublicationInvalid) {
		t.Fatalf("missing runtime error = %v", err)
	}
	binding := extensions.LifecycleRuntimeBinding{
		ExtensionID: executable.ID, ExtensionVersion: executable.Version,
		PackageDigest: executable.PackageDigest, VersionID: executable.ActiveVersionID,
		RuntimeInstanceID: "identity-runtime",
	}
	publication, err := buildLifecycleIdentityPublication(executable, binding)
	if err != nil || publication.Artifact.RuntimeInstanceID != binding.RuntimeInstanceID {
		t.Fatalf("executable publication = %#v, %v", publication, err)
	}
}

func TestLifecycleIdentityGraphRejectsForeignAndDriftedActiveArtifact(t *testing.T) {
	sourceExtension := lifecycleIdentityExtension("1.0.0", 44, "")
	source, err := buildLifecycleIdentityPublication(sourceExtension, extensions.LifecycleRuntimeBinding{})
	if err != nil {
		t.Fatal(err)
	}
	registry := identityregistry.New()
	if _, err := registry.Publish(*source); err != nil {
		t.Fatal(err)
	}
	targetExtension := lifecycleIdentityExtension("2.0.0", 45, "")
	target, err := buildLifecycleIdentityPublication(targetExtension, extensions.LifecycleRuntimeBinding{})
	if err != nil {
		t.Fatal(err)
	}
	if graph, err := lifecycleIdentityGraph(registry.Snapshot(), sourceExtension.ID, target, source, target); err != nil || len(graph) != 1 || graph[0].Artifact != target.Artifact {
		t.Fatalf("upgrade graph = %#v, %v", graph, err)
	}
	if _, err := lifecycleIdentityGraph(registry.Snapshot(), sourceExtension.ID, target, target); !errors.Is(err, ErrLifecycleRegistryPublicationConflict) {
		t.Fatalf("foreign source error = %v", err)
	}
	drifted := *source
	drifted.Permissions = append([]identityregistry.PermissionDefinition(nil), source.Permissions...)
	drifted.Permissions[0].Description = "same artifact drift"
	if _, err := lifecycleIdentityGraph(registry.Snapshot(), sourceExtension.ID, target, &drifted, target); !errors.Is(err, ErrLifecycleRegistryPublicationConflict) {
		t.Fatalf("drift error = %v", err)
	}
}

func TestBuildLifecycleIdentityPublicationSkipsAbsentSurface(t *testing.T) {
	extension := lifecycleIdentityExtension("1.0.0", 46, "")
	extension.Manifest.Identity = nil
	extension.Manifest.PermissionDefinitions = nil
	publication, err := buildLifecycleIdentityPublication(extension, extensions.LifecycleRuntimeBinding{})
	if err != nil || publication != nil {
		t.Fatalf("publication = %#v, %v", publication, err)
	}
}

func TestLifecycleIdentityReconcileEnableUpgradeRollbackDisableAndReplay(t *testing.T) {
	sourceExtension := lifecycleIdentityExtension("1.0.0", 51, "")
	targetExtension := lifecycleIdentityExtension("2.0.0", 52, "")
	sourcePublication, err := buildLifecycleIdentityPublication(sourceExtension, extensions.LifecycleRuntimeBinding{})
	if err != nil {
		t.Fatal(err)
	}
	targetPublication, err := buildLifecycleIdentityPublication(targetExtension, extensions.LifecycleRuntimeBinding{})
	if err != nil {
		t.Fatal(err)
	}
	source := &lifecycleRegistryMaterial{extension: sourceExtension, identityPublication: sourcePublication}
	target := &lifecycleRegistryMaterial{extension: targetExtension, identityPublication: targetPublication}
	store := &memoryIdentityPublicationStore{}
	boundary := &PostgresLifecycleBoundaryRegistries{identity: identityregistry.New(), identityStore: store, identitySet: true}

	enable := LifecycleBoundaryRequest{TargetExtension: sourceExtension, ActorUserID: 41, AuditEventID: 81}
	if err := boundary.reconcileIdentity(t.Context(), enable, nil, source, source); err != nil {
		t.Fatal(err)
	}
	enabled := boundary.identity.Snapshot()
	if len(enabled.Publications) != 1 || enabled.Publications[0].Artifact != sourcePublication.Artifact {
		t.Fatalf("enabled snapshot = %#v", enabled)
	}
	if err := boundary.reconcileIdentity(t.Context(), enable, nil, source, source); err != nil {
		t.Fatal(err)
	}
	if replay := boundary.identity.Snapshot(); replay.Revision != enabled.Revision {
		t.Fatalf("exact replay revision = %d want %d", replay.Revision, enabled.Revision)
	}

	upgrade := LifecycleBoundaryRequest{TargetExtension: targetExtension, ActorUserID: 42, AuditEventID: 82}
	if err := boundary.reconcileIdentity(t.Context(), upgrade, source, target, target); err != nil {
		t.Fatal(err)
	}
	if active, found := boundary.identity.SnapshotPublication(sourceExtension.ID); !found || active.Artifact != targetPublication.Artifact {
		t.Fatalf("upgraded publication = %#v found=%t", active, found)
	}

	rollback := LifecycleBoundaryRequest{TargetExtension: sourceExtension, ActorUserID: 43, AuditEventID: 83}
	if err := boundary.reconcileIdentity(t.Context(), rollback, target, source, source); err != nil {
		t.Fatal(err)
	}
	if active, found := boundary.identity.SnapshotPublication(sourceExtension.ID); !found || active.Artifact != sourcePublication.Artifact {
		t.Fatalf("rolled back publication = %#v found=%t", active, found)
	}

	disable := LifecycleBoundaryRequest{TargetExtension: sourceExtension, ActorUserID: 44, AuditEventID: 84}
	if err := boundary.reconcileIdentity(t.Context(), disable, source, nil, nil); err != nil {
		t.Fatal(err)
	}
	disabled := boundary.identity.Snapshot()
	if len(disabled.Publications) != 0 || len(disabled.Tombstones) != 2 {
		t.Fatalf("disabled snapshot = %#v", disabled)
	}
	if calls := store.Inputs(); len(calls) != 5 || calls[2].Desired == nil || calls[3].Desired == nil || calls[4].Desired != nil {
		t.Fatalf("durable calls = %#v", calls)
	}
}

func TestLifecycleIdentityReconcileIsDurableFirst(t *testing.T) {
	extension := lifecycleIdentityExtension("1.0.0", 53, "")
	publication, err := buildLifecycleIdentityPublication(extension, extensions.LifecycleRuntimeBinding{})
	if err != nil {
		t.Fatal(err)
	}
	material := &lifecycleRegistryMaterial{extension: extension, identityPublication: publication}
	durableErr := errors.New("durable identity unavailable")
	store := &memoryIdentityPublicationStore{failure: durableErr}
	registry := identityregistry.New()
	boundary := &PostgresLifecycleBoundaryRegistries{identity: registry, identityStore: store, identitySet: true}
	request := LifecycleBoundaryRequest{TargetExtension: extension, ActorUserID: 45, AuditEventID: 85}
	if err := boundary.reconcileIdentity(t.Context(), request, nil, material, material); !errors.Is(err, durableErr) {
		t.Fatalf("reconcile error = %v", err)
	}
	if snapshot := registry.Snapshot(); snapshot.Revision != 0 || len(snapshot.Publications) != 0 {
		t.Fatalf("process graph changed before durable commit: %#v", snapshot)
	}
}

func TestLifecycleIdentityRootOnlyRestartRejectsOrphanActivePublication(t *testing.T) {
	extension := lifecycleIdentityExtension("1.0.0", 58, "")
	extension.Status = extensions.StatusEnabled
	extension.Manifest.PermissionDefinitions = nil
	extension.Manifest.Identity.UserFields = nil
	publication, err := buildLifecycleIdentityPublication(extension, extensions.LifecycleRuntimeBinding{})
	if err != nil || publication == nil || publication.Identity == nil ||
		len(publication.Permissions) != 0 || len(publication.Identity.UserFields) != 0 {
		t.Fatalf("root-only publication=%#v err=%v", publication, err)
	}
	material := &lifecycleRegistryMaterial{extension: extension, identityPublication: publication}
	store := &memoryIdentityPublicationStore{}
	registry := identityregistry.New()
	boundary := &PostgresLifecycleBoundaryRegistries{
		identity: registry, identityStore: store, identitySet: true,
	}
	request := LifecycleBoundaryRequest{TargetExtension: extension, ActorUserID: 48, AuditEventID: 88}
	if err := boundary.reconcileIdentity(t.Context(), request, nil, material, material); err != nil {
		t.Fatal(err)
	}
	if snapshot := registry.Snapshot(); len(snapshot.Publications) != 1 || len(snapshot.Tombstones) != 0 {
		t.Fatalf("root-only enabled snapshot=%#v", snapshot)
	}

	restarted := &PostgresLifecycleBoundaryRegistries{
		identity: identityregistry.New(), identityStore: store, identitySet: true,
	}
	if err := restarted.restoreIdentityPublications(t.Context(), []extensions.Extension{extension}, false); err != nil {
		t.Fatal(err)
	}
	if snapshot := restarted.identity.Snapshot(); len(snapshot.Publications) != 1 {
		t.Fatalf("root-only restart snapshot=%#v", snapshot)
	}

	orphan := extension
	orphan.Status = extensions.StatusDisabled
	if err := (&PostgresLifecycleBoundaryRegistries{
		identity: identityregistry.New(), identityStore: store, identitySet: true,
	}).restoreIdentityPublications(t.Context(), []extensions.Extension{orphan}, false); !errors.Is(
		err, ErrLifecycleRegistryPublicationConflict,
	) {
		t.Fatalf("orphan active root restart error=%v", err)
	}

	if err := boundary.reconcileIdentity(t.Context(), request, material, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := restarted.restoreIdentityPublications(t.Context(), []extensions.Extension{orphan}, false); err != nil {
		t.Fatalf("retired root-only restart: %v", err)
	}
	if snapshot := restarted.identity.Snapshot(); len(snapshot.Publications) != 0 {
		t.Fatalf("retired root-only snapshot=%#v", snapshot)
	}
}

// Mirrors the normal-dev DB startup blocker: an enabled permission-only fixture
// (sforum.admin-surface-reference) with empty identity registry ledgers.
// Stores without LegacyPublicationAdopter stay fail-closed on ErrNotFound.
func TestLifecycleIdentityRestoreRejectsPermissionOnlyAdminSurfaceWithoutDurable(t *testing.T) {
	extension := adminSurfaceReferencePermissionOnlyExtension(5866)
	extension.Status = extensions.StatusEnabled
	store := &memoryIdentityPublicationStore{}
	boundary := &PostgresLifecycleBoundaryRegistries{
		identity: identityregistry.New(), identityStore: store, identitySet: true,
	}
	err := boundary.restoreIdentityPublications(t.Context(), []extensions.Extension{extension}, false)
	if !errors.Is(err, identityregistry.ErrNotFound) {
		t.Fatalf("missing durable restore error=%v", err)
	}
	if !strings.Contains(err.Error(), "sforum.admin-surface-reference") {
		t.Fatalf("restore error must name the owner extension: %v", err)
	}
	if !strings.Contains(err.Error(), "validate durable identity publication for sforum.admin-surface-reference") {
		t.Fatalf("restore error must keep the action context: %v", err)
	}
	if snapshot := boundary.identity.Snapshot(); snapshot.Revision != 0 || len(snapshot.Publications) != 0 {
		t.Fatalf("failed restore must not publish process graph: %#v", snapshot)
	}

	// Allowed path: reconcile first (as lifecycle enable does), then restart restore.
	publication, err := buildLifecycleIdentityPublication(extension, extensions.LifecycleRuntimeBinding{})
	if err != nil || publication == nil || publication.Identity != nil || len(publication.Permissions) != 1 {
		t.Fatalf("permission-only publication=%#v err=%v", publication, err)
	}
	material := &lifecycleRegistryMaterial{extension: extension, identityPublication: publication}
	request := LifecycleBoundaryRequest{TargetExtension: extension, ActorUserID: 49, AuditEventID: 89}
	if err := boundary.reconcileIdentity(t.Context(), request, nil, material, material); err != nil {
		t.Fatal(err)
	}
	restarted := &PostgresLifecycleBoundaryRegistries{
		identity: identityregistry.New(), identityStore: store, identitySet: true,
	}
	if err := restarted.restoreIdentityPublications(t.Context(), []extensions.Extension{extension}, false); err != nil {
		t.Fatalf("permission-only restart after reconcile: %v", err)
	}
	if snapshot := restarted.identity.Snapshot(); len(snapshot.Publications) != 1 ||
		snapshot.Publications[0].Artifact.ExtensionID != extension.ID ||
		len(snapshot.Publications[0].Permissions) != 1 {
		t.Fatalf("permission-only restart snapshot=%#v", snapshot)
	}
}

func TestLifecycleIdentityRestoreAdoptsLegacyOnErrNotFoundOnly(t *testing.T) {
	extension := adminSurfaceReferencePermissionOnlyExtension(5866)
	extension.Status = extensions.StatusEnabled
	store := &memoryLegacyAdoptingIdentityStore{}
	boundary := &PostgresLifecycleBoundaryRegistries{
		identity: identityregistry.New(), identityStore: store, identitySet: true,
	}
	if err := boundary.restoreIdentityPublications(t.Context(), []extensions.Extension{extension}, false); err != nil {
		t.Fatalf("legacy adopt restore: %v", err)
	}
	if store.adoptCalls != 1 || store.lastBatch != 1 {
		t.Fatalf("adopt calls=%d batch=%d want calls=1 batch=1", store.adoptCalls, store.lastBatch)
	}
	if snapshot := boundary.identity.Snapshot(); len(snapshot.Publications) != 1 ||
		snapshot.Publications[0].Artifact.ExtensionID != extension.ID {
		t.Fatalf("adopted process graph=%#v", snapshot)
	}

	// Safe Mode must never adopt third-party publications.
	safeStore := &memoryLegacyAdoptingIdentityStore{}
	safeBoundary := &PostgresLifecycleBoundaryRegistries{
		identity: identityregistry.New(), identityStore: safeStore, identitySet: true,
	}
	if err := safeBoundary.restoreIdentityPublications(t.Context(), []extensions.Extension{extension}, true); err != nil {
		t.Fatalf("safe mode restore: %v", err)
	}
	if safeStore.adoptCalls != 0 {
		t.Fatalf("safe mode must not adopt: calls=%d", safeStore.adoptCalls)
	}
	if snapshot := safeBoundary.identity.Snapshot(); len(snapshot.Publications) != 0 {
		t.Fatalf("safe mode process graph=%#v", snapshot)
	}

	// Non-ErrNotFound validation failures must not invoke the adopter.
	publication, err := buildLifecycleIdentityPublication(extension, extensions.LifecycleRuntimeBinding{})
	if err != nil {
		t.Fatal(err)
	}
	foreign := *publication
	foreign.Artifact.PackageDigest = strings.Repeat("f", 64)
	conflictRoot, err := identityregistryDesiredRootForTest(t, foreign)
	if err != nil {
		t.Fatal(err)
	}
	conflictStore := &memoryLegacyAdoptingIdentityStore{
		memoryIdentityPublicationStore: memoryIdentityPublicationStore{
			state: identityregistry.DurableState{RootTips: []identityregistry.DurableRootPublicationTip{conflictRoot}},
		},
	}
	conflictBoundary := &PostgresLifecycleBoundaryRegistries{
		identity: identityregistry.New(), identityStore: conflictStore, identitySet: true,
	}
	err = conflictBoundary.restoreIdentityPublications(t.Context(), []extensions.Extension{extension}, false)
	if !errors.Is(err, ErrLifecycleRegistryPublicationConflict) ||
		!strings.Contains(err.Error(), identityregistry.ErrArtifactConflict.Error()) {
		t.Fatalf("artifact conflict restore error=%v", err)
	}
	if conflictStore.adoptCalls != 0 {
		t.Fatalf("adopter must not run on ErrArtifactConflict: calls=%d", conflictStore.adoptCalls)
	}
}

func TestLifecycleIdentityRestoreAdoptsLegacyBatchOnce(t *testing.T) {
	first := adminSurfaceReferencePermissionOnlyExtension(5866)
	first.Status = extensions.StatusEnabled
	second := adminSurfaceReferencePermissionOnlyExtension(5867)
	second.ID = "sforum.legacy-batch-second"
	second.Manifest.ID = second.ID
	second.ActiveVersionID = 5867
	second.PackageDigest = "91b964f80707b257f6f401faffb07fe0f0a6aa6b5833a6fab0cedaab77b3324f"
	second.Status = extensions.StatusEnabled
	second.Manifest.PermissionDefinitions = []extensions.ManifestPermissionDefinition{{
		Key: second.ID + ".manage", ContractVersion: second.ID + ".permission.manage@1",
		Label: "Second", Description: "Second", RecommendedRoles: []string{"administrator"},
		AssignmentPolicy: "host",
	}}

	store := &memoryLegacyAdoptingIdentityStore{}
	boundary := &PostgresLifecycleBoundaryRegistries{
		identity: identityregistry.New(), identityStore: store, identitySet: true,
	}
	if err := boundary.restoreIdentityPublications(t.Context(), []extensions.Extension{second, first}, false); err != nil {
		t.Fatalf("batch legacy adopt restore: %v", err)
	}
	if store.adoptCalls != 1 || store.lastBatch != 2 {
		t.Fatalf("adopt calls=%d batch=%d want calls=1 batch=2", store.adoptCalls, store.lastBatch)
	}
	if snapshot := boundary.identity.Snapshot(); len(snapshot.Publications) != 2 {
		t.Fatalf("batch process graph=%#v", snapshot)
	}

	// Stores without adopter stay fail-closed with zero process graph.
	noAdopter := &memoryIdentityPublicationStore{}
	noBoundary := &PostgresLifecycleBoundaryRegistries{
		identity: identityregistry.New(), identityStore: noAdopter, identitySet: true,
	}
	err := noBoundary.restoreIdentityPublications(t.Context(), []extensions.Extension{first}, false)
	if !errors.Is(err, identityregistry.ErrNotFound) {
		t.Fatalf("no adopter error=%v", err)
	}
	if snapshot := noBoundary.identity.Snapshot(); len(snapshot.Publications) != 0 {
		t.Fatalf("no adopter must not publish: %#v", snapshot)
	}
}

type memoryLegacyAdoptingIdentityStore struct {
	memoryIdentityPublicationStore
	adoptCalls int
	adoptErr   error
	lastBatch  int
}

func (s *memoryLegacyAdoptingIdentityStore) AdoptLegacyPublications(
	_ context.Context,
	publications []identityregistry.Publication,
) (identityregistry.DurableState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.adoptCalls++
	s.lastBatch = len(publications)
	if s.adoptErr != nil {
		return identityregistry.DurableState{}, s.adoptErr
	}
	if s.failure != nil {
		return identityregistry.DurableState{}, s.failure
	}
	for _, publication := range publications {
		s.state = reconcileMemoryIdentityState(s.state, identityregistry.ReconcilePublicationInput{
			ExtensionID: publication.Artifact.ExtensionID, AllowedTarget: &publication.Artifact,
			Desired: &publication, ActorUserID: 549, AuditEventID: 962,
		})
	}
	return cloneMemoryIdentityState(s.state), nil
}

func identityregistryDesiredRootForTest(
	t *testing.T,
	publication identityregistry.Publication,
) (identityregistry.DurableRootPublicationTip, error) {
	t.Helper()
	state := reconcileMemoryIdentityState(identityregistry.DurableState{}, identityregistry.ReconcilePublicationInput{
		ExtensionID: publication.Artifact.ExtensionID, AllowedTarget: &publication.Artifact,
		Desired: &publication, ActorUserID: 1, AuditEventID: 1,
	})
	if len(state.RootTips) != 1 {
		return identityregistry.DurableRootPublicationTip{}, fmt.Errorf("root tips=%d", len(state.RootTips))
	}
	return state.RootTips[0], nil
}

var _ identityregistry.LegacyPublicationAdopter = (*memoryLegacyAdoptingIdentityStore)(nil)

func adminSurfaceReferencePermissionOnlyExtension(versionID int64) extensions.Extension {
	const id = "sforum.admin-surface-reference"
	extension := extensions.Extension{
		ID: id, Type: extensions.TypePlugin, Version: "1.0.0", ActiveVersionID: versionID,
		PackageDigest: "81b964f80707b257f6f401faffb07fe0f0a6aa6b5833a6fab0cedaab77b3324f",
	}
	extension.Manifest = extensions.Manifest{ID: id, Type: extensions.TypePlugin, Version: "1.0.0"}
	extension.Manifest.PermissionDefinitions = []extensions.ManifestPermissionDefinition{{
		Key: id + ".manage", ContractVersion: id + ".permission.manage@1",
		Label:            "Use admin surface reference",
		Description:      "View and invoke the reference plugin's admin surfaces.",
		RecommendedRoles: []string{"administrator"}, AssignmentPolicy: "host",
	}}
	return extension
}

func TestLifecycleIdentityRestartSafeModeAndConditionalDependencies(t *testing.T) {
	withoutIdentity := lifecycleIdentityExtension("1.0.0", 54, "")
	withoutIdentity.Manifest.Identity = nil
	withoutIdentity.Manifest.PermissionDefinitions = nil
	if err := (*PostgresLifecycleBoundaryRegistries)(nil).restoreIdentityPublications(
		t.Context(), []extensions.Extension{withoutIdentity}, false,
	); err != nil {
		t.Fatalf("no-declaration restore required dependencies: %v", err)
	}
	if err := (&PostgresLifecycleBoundaryRegistries{identitySet: true, identity: identityregistry.New()}).
		restoreIdentityPublications(t.Context(), nil, false); !errors.Is(err, ErrLifecycleRegistryPublicationUnavailable) {
		t.Fatalf("explicit partial dependency error = %v", err)
	}

	extension := lifecycleIdentityExtension("1.0.0", 55, "")
	extension.Status = extensions.StatusEnabled
	publication, err := buildLifecycleIdentityPublication(extension, extensions.LifecycleRuntimeBinding{})
	if err != nil {
		t.Fatal(err)
	}
	store := &memoryIdentityPublicationStore{}
	if _, err := store.Reconcile(t.Context(), identityregistry.ReconcilePublicationInput{
		ExtensionID: extension.ID, AllowedTarget: &publication.Artifact, Desired: publication,
		ActorUserID: 46, AuditEventID: 86,
	}); err != nil {
		t.Fatal(err)
	}
	registry := identityregistry.New()
	coreArtifact, err := identityregistry.NewCoreArtifact("core.identity", "1.0.0", strings.Repeat("c", 64))
	if err != nil {
		t.Fatal(err)
	}
	core := identityregistry.Publication{Artifact: coreArtifact, Permissions: []identityregistry.PermissionDefinition{{
		Key: "core.identity.access", ContractVersion: "core.identity.access@1",
		Label: "Access", Description: "Core identity access.", AssignmentPolicy: "host",
	}}}
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	boundary := &PostgresLifecycleBoundaryRegistries{
		identity: registry, identityStore: store, identitySet: true,
	}
	if err := boundary.restoreIdentityPublications(t.Context(), []extensions.Extension{extension}, false); err != nil {
		t.Fatal(err)
	}
	if snapshot := registry.Snapshot(); snapshot.SafeMode || len(snapshot.Publications) != 2 {
		t.Fatalf("restart snapshot = %#v", snapshot)
	}
	if err := boundary.restoreIdentityPublications(t.Context(), []extensions.Extension{extension}, true); err != nil {
		t.Fatal(err)
	}
	safe := registry.Snapshot()
	if !safe.SafeMode || len(safe.Publications) != 1 || !safe.Publications[0].Artifact.Core || len(safe.Tombstones) != 2 {
		t.Fatalf("safe mode snapshot = %#v", safe)
	}
	if err := boundary.restoreIdentityPublications(t.Context(), []extensions.Extension{extension}, false); err != nil {
		t.Fatal(err)
	}
	if restored := registry.Snapshot(); restored.SafeMode || len(restored.Publications) != 2 || len(restored.Tombstones) != 2 {
		t.Fatalf("restored snapshot = %#v", restored)
	}
	if _, err := store.Reconcile(t.Context(), identityregistry.ReconcilePublicationInput{
		ExtensionID: extension.ID, AllowedSource: &publication.Artifact,
		ActorUserID: 47, AuditEventID: 87,
	}); err != nil {
		t.Fatal(err)
	}
	staleBoundary := &PostgresLifecycleBoundaryRegistries{
		identity: identityregistry.New(), identityStore: store, identitySet: true,
	}
	if err := staleBoundary.restoreIdentityPublications(
		t.Context(), []extensions.Extension{extension}, false,
	); !errors.Is(err, ErrLifecycleRegistryPublicationConflict) {
		t.Fatalf("tombstoned restart error = %v", err)
	}
}

func TestLifecycleIdentityDigestIsAdditiveAndAbsentSurfaceIsByteCompatible(t *testing.T) {
	withoutIdentity := lifecycleIdentityExtension("1.0.0", 56, "")
	withoutIdentity.Manifest.Identity = nil
	withoutIdentity.Manifest.PermissionDefinitions = nil
	material := &lifecycleRegistryMaterial{extension: withoutIdentity}
	legacy, err := encodeLifecycleRegistryMaterialDigest(material, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := refreshLifecycleRegistryMaterialDigest(material); err != nil {
		t.Fatal(err)
	}
	if material.digest != legacy || material.legacyDigest != "" || len(material.compatibleDigests) != 0 {
		t.Fatalf("absent Identity changed digest: %#v", material)
	}

	withIdentity := lifecycleIdentityExtension("1.0.0", 57, "")
	publication, err := buildLifecycleIdentityPublication(withIdentity, extensions.LifecycleRuntimeBinding{})
	if err != nil {
		t.Fatal(err)
	}
	identityMaterial := &lifecycleRegistryMaterial{extension: withIdentity, identityPublication: publication}
	prior, err := encodeLifecycleRegistryMaterialDigest(identityMaterial, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := refreshLifecycleRegistryMaterialDigest(identityMaterial); err != nil {
		t.Fatal(err)
	}
	wantV6, err := encodeLifecycleRegistryMaterialDigestV6(identityMaterial)
	if err != nil {
		t.Fatal(err)
	}
	if identityMaterial.digest != wantV6 || identityMaterial.digest == prior ||
		!reflect.DeepEqual(identityMaterial.compatibleDigests, []string{prior}) {
		t.Fatalf("Identity digest = %#v", identityMaterial)
	}
}

type memoryIdentityPublicationStore struct {
	mu      sync.Mutex
	state   identityregistry.DurableState
	inputs  []identityregistry.ReconcilePublicationInput
	failure error
}

func (s *memoryIdentityPublicationStore) Reconcile(
	_ context.Context,
	input identityregistry.ReconcilePublicationInput,
) (identityregistry.DurableState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inputs = append(s.inputs, input)
	if s.failure != nil {
		return identityregistry.DurableState{}, s.failure
	}
	s.state = reconcileMemoryIdentityState(s.state, input)
	return cloneMemoryIdentityState(s.state), nil
}

func (s *memoryIdentityPublicationStore) LoadDurableState(context.Context) (identityregistry.DurableState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failure != nil {
		return identityregistry.DurableState{}, s.failure
	}
	return cloneMemoryIdentityState(s.state), nil
}

func (s *memoryIdentityPublicationStore) Inputs() []identityregistry.ReconcilePublicationInput {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]identityregistry.ReconcilePublicationInput(nil), s.inputs...)
}

func reconcileMemoryIdentityState(
	state identityregistry.DurableState,
	input identityregistry.ReconcilePublicationInput,
) identityregistry.DurableState {
	desired := memoryIdentityDeclarations(input.Desired)
	owners := make(map[string]identityregistry.DurableOwner, len(state.Owners)+len(desired))
	tips := make(map[string]identityregistry.DurableDeclarationTip, len(state.Tips)+len(desired))
	for _, owner := range state.Owners {
		owners[owner.IdentityKind+"\x00"+owner.StableID] = owner
	}
	for _, tip := range state.Tips {
		tips[tip.IdentityKind+"\x00"+tip.StableID] = tip
	}
	desiredKeys := make(map[string]bool, len(desired))
	for _, declaration := range desired {
		key := declaration.IdentityKind + "\x00" + declaration.StableID
		desiredKeys[key] = true
		owners[key] = identityregistry.DurableOwner{
			IdentityKind: declaration.IdentityKind, StableID: declaration.StableID,
			OwnerExtensionID: input.ExtensionID,
		}
		if current, found := tips[key]; found {
			declaration.Revision = current.Revision
			if current.RegistryState != identityregistry.RegistryStateActive ||
				current.ExtensionVersionID != declaration.ExtensionVersionID ||
				current.PackageDigest != declaration.PackageDigest {
				declaration.Revision++
			}
		}
		tips[key] = declaration
	}
	for key, tip := range tips {
		if tip.OwnerExtensionID != input.ExtensionID || desiredKeys[key] || tip.RegistryState != identityregistry.RegistryStateActive {
			continue
		}
		tip.RegistryState = identityregistry.RegistryStateTombstone
		tip.Revision++
		tips[key] = tip
	}
	rootTips := append([]identityregistry.DurableRootPublicationTip(nil), state.RootTips...)
	var currentRoot *identityregistry.DurableRootPublicationTip
	for index := range rootTips {
		if rootTips[index].OwnerExtensionID != input.ExtensionID ||
			currentRoot != nil && rootTips[index].Revision <= currentRoot.Revision {
			continue
		}
		candidate := rootTips[index]
		currentRoot = &candidate
	}
	if input.Desired == nil {
		if currentRoot != nil && currentRoot.RegistryState == identityregistry.RegistryStateActive {
			tombstone := *currentRoot
			tombstone.Revision++
			tombstone.RegistryState = identityregistry.RegistryStateTombstone
			tombstone.ActorUserID = input.ActorUserID
			tombstone.AuditEventID = input.AuditEventID
			rootTips = append(rootTips, tombstone)
		}
	} else {
		desiredRoot := memoryIdentityRootTip(*input.Desired, input.ActorUserID, input.AuditEventID)
		if currentRoot == nil {
			rootTips = append(rootTips, desiredRoot)
		} else if currentRoot.RegistryState != identityregistry.RegistryStateActive ||
			currentRoot.ExtensionVersionID != desiredRoot.ExtensionVersionID ||
			currentRoot.PackageDigest != desiredRoot.PackageDigest {
			desiredRoot.Revision = currentRoot.Revision + 1
			rootTips = append(rootTips, desiredRoot)
		}
	}
	result := identityregistry.DurableState{
		Owners: make([]identityregistry.DurableOwner, 0, len(owners)),
		Tips:   make([]identityregistry.DurableDeclarationTip, 0, len(tips)),
	}
	for _, owner := range owners {
		result.Owners = append(result.Owners, owner)
	}
	for _, tip := range tips {
		result.Tips = append(result.Tips, tip)
	}
	// Keep the latest root tip per owner so multi-plugin batch adoption does not
	// drop sibling history when reconciling one extension at a time.
	latestByOwner := make(map[string]identityregistry.DurableRootPublicationTip, len(rootTips))
	for _, tip := range rootTips {
		current, found := latestByOwner[tip.OwnerExtensionID]
		if !found || tip.Revision > current.Revision {
			latestByOwner[tip.OwnerExtensionID] = tip
		}
	}
	for _, tip := range latestByOwner {
		result.RootTips = append(result.RootTips, tip)
	}
	return result
}

func memoryIdentityRootTip(
	publication identityregistry.Publication,
	actorUserID, auditEventID int64,
) identityregistry.DurableRootPublicationTip {
	publication.Artifact.RuntimeInstanceID = ""
	raw, _ := json.Marshal(publication)
	sum := sha256.Sum256(append([]byte("sforum.identity-registry.root-publication@1\x00"), raw...))
	return identityregistry.DurableRootPublicationTip{
		OwnerExtensionID: publication.Artifact.ExtensionID,
		Revision:         1, RegistryState: identityregistry.RegistryStateActive,
		ExtensionVersionID: publication.Artifact.VersionID,
		ExtensionVersion:   publication.Artifact.ExtensionVersion,
		PackageDigest:      publication.Artifact.PackageDigest,
		SchemaVersion:      identityregistry.SchemaVersion,
		PublicationDigest:  hex.EncodeToString(sum[:]), PublicationJSON: raw,
		ActorUserID: actorUserID, AuditEventID: auditEventID,
	}
}

func memoryIdentityDeclarations(publication *identityregistry.Publication) []identityregistry.DurableDeclarationTip {
	if publication == nil {
		return nil
	}
	artifact := publication.Artifact
	makeTip := func(kind, id, contract, digest string) identityregistry.DurableDeclarationTip {
		return identityregistry.DurableDeclarationTip{
			IdentityKind: kind, StableID: id, OwnerExtensionID: artifact.ExtensionID,
			Revision: 1, RegistryState: identityregistry.RegistryStateActive,
			ExtensionVersionID: artifact.VersionID, ExtensionVersion: artifact.ExtensionVersion,
			PackageDigest: artifact.PackageDigest, ContractVersion: contract,
			DeclarationDigest: digest,
		}
	}
	result := make([]identityregistry.DurableDeclarationTip, 0, len(publication.Permissions))
	for _, permission := range publication.Permissions {
		result = append(result, makeTip(
			identityregistry.TombstoneKindPermission, permission.Key, permission.ContractVersion,
			memoryIdentityDeclarationDigest(identityregistry.TombstoneKindPermission, permission),
		))
	}
	if publication.Identity != nil {
		for _, field := range publication.Identity.UserFields {
			result = append(result, makeTip(
				identityregistry.TombstoneKindUserField, field.ID, field.ContractVersion,
				memoryIdentityDeclarationDigest(identityregistry.TombstoneKindUserField, field),
			))
		}
		for _, provider := range publication.Identity.Providers {
			result = append(result, makeTip(
				identityregistry.TombstoneKindProvider, provider.ID, provider.ContractVersion,
				memoryIdentityDeclarationDigest(identityregistry.TombstoneKindProvider, provider),
			))
		}
	}
	return result
}

func memoryIdentityDeclarationDigest(kind string, declaration any) string {
	raw, _ := json.Marshal(declaration)
	sum := sha256.Sum256(append([]byte(kind+"\x00"), raw...))
	return hex.EncodeToString(sum[:])
}

func cloneMemoryIdentityState(input identityregistry.DurableState) identityregistry.DurableState {
	result := identityregistry.DurableState{
		Owners:   append([]identityregistry.DurableOwner(nil), input.Owners...),
		Tips:     append([]identityregistry.DurableDeclarationTip(nil), input.Tips...),
		RootTips: append([]identityregistry.DurableRootPublicationTip(nil), input.RootTips...),
	}
	for index := range result.RootTips {
		result.RootTips[index].PublicationJSON = append([]byte(nil), result.RootTips[index].PublicationJSON...)
	}
	return result
}

var _ identityregistry.PublicationStore = (*memoryIdentityPublicationStore)(nil)

func lifecycleIdentityExtension(version string, versionID int64, executable string) extensions.Extension {
	id := "fixture.identity.lifecycle"
	extension := extensions.Extension{
		ID: id, Type: extensions.TypePlugin, Version: version, ActiveVersionID: versionID,
		PackageDigest: strings.Repeat(string(rune('a'+versionID%6)), 64),
	}
	extension.Manifest = extensions.Manifest{ID: id, Type: extensions.TypePlugin, Version: version}
	extension.Manifest.PermissionDefinitions = []extensions.ManifestPermissionDefinition{{
		Key: id + ".profile", ContractVersion: id + ".profile@1", Label: "Profile",
		Description: "Manage plugin profiles.", AssignmentPolicy: "host",
	}}
	extension.Manifest.Identity = &extensions.ManifestIdentity{
		ContractVersion: id + ".contract@1", SessionPolicy: "core.session.default",
		UserFields: []extensionmanifest.ManifestIdentityUserField{{
			ID: id + ".field.code", ContractVersion: id + ".field.code@1",
			Type: "string", Schema: id + ".field.code.schema@1", ReadPermission: id + ".profile",
		}},
	}
	if executable == "risk" {
		extension.Manifest.Identity.RiskHooks = []string{id + ".risk.login"}
	}
	return extension
}

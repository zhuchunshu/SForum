package identityregistry

import (
	"errors"
	"strings"
	"testing"
)

func TestDurableRootPublicationValidatesRootOnlyRuntimeRotationAndDrift(t *testing.T) {
	publication := durableRootTestPublication("runtime-one")
	state := durableRootTestState(t, publication)
	if err := ValidateDurablePublication(state, publication); err != nil {
		t.Fatalf("validate exact root-only publication: %v", err)
	}
	restarted := publication
	restarted.Artifact.RuntimeInstanceID = "runtime-two"
	if err := ValidateDurablePublication(state, restarted); err != nil {
		t.Fatalf("runtime rotation changed declarative evidence: %v", err)
	}
	if err := ValidateDurablePublicationSet(state, []Publication{restarted}); err != nil {
		t.Fatalf("validate root-only publication set: %v", err)
	}
	if err := ValidateDurablePublicationSet(state, nil); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("orphan active root error=%v", err)
	}

	drift := publication
	drift.Identity = &IdentityDeclaration{
		ContractVersion: publication.Identity.ContractVersion,
		SessionPolicy:   publication.Identity.SessionPolicy,
		RiskHooks:       []string{"fixture.identity.risk.changed"},
	}
	if err := ValidateDurablePublication(state, drift); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("same-artifact root drift error=%v", err)
	}
}

func TestDurableRootPublicationRejectsTamperAndProvesRetirement(t *testing.T) {
	publication := durableRootTestPublication("runtime-one")
	state := durableRootTestState(t, publication)
	tampered := state
	tampered.RootTips = append([]DurableRootPublicationTip(nil), state.RootTips...)
	tampered.RootTips[0].PublicationDigest = strings.Repeat("f", 64)
	if _, err := DurableStateToTombstones(tampered); !errors.Is(err, ErrInvalid) {
		t.Fatalf("tampered root digest error=%v", err)
	}

	retired := state
	retired.RootTips = append([]DurableRootPublicationTip(nil), state.RootTips...)
	retired.RootTips[0].RegistryState = RegistryStateTombstone
	retired.RootTips[0].Revision++
	if err := ValidateDurableRetirement(retired, publication.Artifact.ExtensionID); err != nil {
		t.Fatalf("validate root-only retirement: %v", err)
	}
	if err := ValidateDurablePublication(retired, publication); !errors.Is(err, ErrStale) {
		t.Fatalf("retired publication validation error=%v", err)
	}
}

func TestDurablePublicationSetRejectsOrphanActiveLeafWithoutRoot(t *testing.T) {
	publication := publicationStoreFixture(
		Artifact{
			ExtensionID: fixtureExtensionID, ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("a", 64), VersionID: 101,
			RuntimeInstanceID: "runtime-one",
		},
		1,
		nil,
	)
	desired, err := desiredDurableDeclarations(&publication)
	if err != nil {
		t.Fatal(err)
	}
	state := DurableState{}
	for _, declaration := range desired {
		state.Owners = append(state.Owners, DurableOwner{
			IdentityKind: declaration.kind, StableID: declaration.stableID,
			OwnerExtensionID: publication.Artifact.ExtensionID,
		})
		state.Tips = append(state.Tips, DurableDeclarationTip{
			IdentityKind: declaration.kind, StableID: declaration.stableID,
			OwnerExtensionID: publication.Artifact.ExtensionID,
			Revision:         1, RegistryState: RegistryStateActive,
			ExtensionVersionID: publication.Artifact.VersionID,
			ExtensionVersion:   publication.Artifact.ExtensionVersion,
			PackageDigest:      publication.Artifact.PackageDigest,
			ContractVersion:    declaration.contractVersion,
			DeclarationDigest:  declaration.digest,
		})
	}
	if err := ValidateDurablePublicationSet(state, nil); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("orphan active leaf error=%v", err)
	}
	if err := ValidateDurablePublication(state, publication); !errors.Is(err, ErrNotFound) {
		t.Fatalf("leaf-only publication proof error=%v", err)
	}
}

// Reproduces the exact normal-dev startup shape that blocks restore:
// sforum.admin-surface-reference is enabled with one Host-assigned permission,
// but extension_identity_registry_* has no durable root/leaf history.
func TestDurablePublicationRejectsPermissionOnlyAdminSurfaceWithoutRoot(t *testing.T) {
	publication := adminSurfaceReferencePermissionOnlyPublication(5866)
	if err := ValidateDurablePublication(DurableState{}, publication); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty durable for permission-only admin-surface error=%v", err)
	}
	if err := ValidateDurablePublicationSet(DurableState{}, []Publication{publication}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty durable set for permission-only admin-surface error=%v", err)
	}

	state := durableStateForPublication(t, publication, 41, 81)
	if err := ValidateDurablePublication(state, publication); err != nil {
		t.Fatalf("complete durable permission-only admin-surface: %v", err)
	}
	if err := ValidateDurablePublicationSet(state, []Publication{publication}); err != nil {
		t.Fatalf("complete durable set permission-only admin-surface: %v", err)
	}

	// Drifted permission label must not reuse the same durable root tip.
	drift := publication
	drift.Permissions = append([]PermissionDefinition(nil), publication.Permissions...)
	drift.Permissions[0].Label = "changed label"
	if err := ValidateDurablePublication(state, drift); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("permission-only root drift error=%v", err)
	}

	// Orphan active durable root without a matching enabled publication fails closed.
	if err := ValidateDurablePublicationSet(state, nil); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("orphan permission-only root error=%v", err)
	}
}

func adminSurfaceReferencePermissionOnlyPublication(versionID int64) Publication {
	return Publication{
		Artifact: Artifact{
			ExtensionID:      "sforum.admin-surface-reference",
			ExtensionVersion: "1.0.0",
			PackageDigest:    "81b964f80707b257f6f401faffb07fe0f0a6aa6b5833a6fab0cedaab77b3324f",
			VersionID:        versionID,
		},
		Permissions: []PermissionDefinition{{
			Key:              "sforum.admin-surface-reference.manage",
			ContractVersion:  "sforum.admin-surface-reference.permission.manage@1",
			Label:            "Use admin surface reference",
			Description:      "View and invoke the reference plugin's admin surfaces.",
			RecommendedRoles: []string{"administrator"},
			AssignmentPolicy: "host",
		}},
	}
}

func durableStateForPublication(t *testing.T, publication Publication, actorUserID, auditEventID int64) DurableState {
	t.Helper()
	desiredRoot, err := desiredDurableRootPublication(&publication)
	if err != nil {
		t.Fatal(err)
	}
	desiredRoot.tip.ActorUserID = actorUserID
	desiredRoot.tip.AuditEventID = auditEventID
	desiredLeaves, err := desiredDurableDeclarations(&publication)
	if err != nil {
		t.Fatal(err)
	}
	state := DurableState{RootTips: []DurableRootPublicationTip{desiredRoot.tip}}
	for _, declaration := range desiredLeaves {
		state.Owners = append(state.Owners, DurableOwner{
			IdentityKind: declaration.kind, StableID: declaration.stableID,
			OwnerExtensionID: publication.Artifact.ExtensionID,
		})
		state.Tips = append(state.Tips, DurableDeclarationTip{
			IdentityKind: declaration.kind, StableID: declaration.stableID,
			OwnerExtensionID: publication.Artifact.ExtensionID,
			Revision:         1, RegistryState: RegistryStateActive,
			ExtensionVersionID: publication.Artifact.VersionID,
			ExtensionVersion:   publication.Artifact.ExtensionVersion,
			PackageDigest:      publication.Artifact.PackageDigest,
			ContractVersion:    declaration.contractVersion,
			DeclarationDigest:  declaration.digest,
			ActorUserID:        actorUserID, AuditEventID: auditEventID,
		})
	}
	return state
}

func durableRootTestPublication(runtimeID string) Publication {
	return Publication{
		Artifact: Artifact{
			ExtensionID: fixtureExtensionID, ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("a", 64), VersionID: 101,
			RuntimeInstanceID: runtimeID,
		},
		Identity: &IdentityDeclaration{
			ContractVersion: "fixture.identity.contract@1",
			SessionPolicy:   "fixture.identity.session",
			RiskHooks:       []string{"fixture.identity.risk.login"},
		},
	}
}

func durableRootTestState(t *testing.T, publication Publication) DurableState {
	t.Helper()
	desired, err := desiredDurableRootPublication(&publication)
	if err != nil {
		t.Fatal(err)
	}
	desired.tip.ActorUserID = 41
	desired.tip.AuditEventID = 81
	return DurableState{RootTips: []DurableRootPublicationTip{desired.tip}}
}

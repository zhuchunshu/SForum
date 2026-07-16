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
	if err := ValidateDurablePublication(state, publication); !errors.Is(err, ErrInvalid) {
		t.Fatalf("leaf-only publication proof error=%v", err)
	}
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

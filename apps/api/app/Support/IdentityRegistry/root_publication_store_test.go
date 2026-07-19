package identityregistry

import (
	"bytes"
	"encoding/json"
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

func TestDurableRootPublicationRejectsUnboundSessionPolicy(t *testing.T) {
	publication := durableRootTestPublication("runtime-one")
	publication.Identity.SessionPolicy = "fixture.identity.session"
	if _, err := desiredDurableRootPublication(&publication); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unbound session policy durable root error=%v", err)
	}
}

func TestDurableRootPublicationKeepsLegacyUnboundSessionPolicyRecoveryReadable(t *testing.T) {
	publication := durableRootTestPublication("runtime-one")
	publication.Identity.SessionPolicy = "fixture.identity.session"
	validation := publication
	validation.Artifact.RuntimeInstanceID = "durable-publication-validation"
	normalized, err := normalizeHistoricalDurablePublication(validation)
	if err != nil {
		t.Fatal(err)
	}
	normalized = publicationContract(normalized)
	normalized.Artifact.RuntimeInstanceID = ""
	raw, err := json.Marshal(normalized)
	if err != nil {
		t.Fatal(err)
	}
	state := DurableState{RootTips: []DurableRootPublicationTip{{
		OwnerExtensionID: normalized.Artifact.ExtensionID,
		Revision:         1, RegistryState: RegistryStateActive,
		ExtensionVersionID: normalized.Artifact.VersionID,
		ExtensionVersion:   normalized.Artifact.ExtensionVersion,
		PackageDigest:      normalized.Artifact.PackageDigest,
		SchemaVersion:      SchemaVersion,
		PublicationDigest:  durableRootPublicationDigest(raw),
		PublicationJSON:    raw,
		ActorUserID:        41,
		AuditEventID:       81,
	}}}
	if tombstones, err := DurableStateToTombstones(state); err != nil || len(tombstones) != 0 {
		t.Fatalf("legacy recovery tombstones=%#v error=%v", tombstones, err)
	}
	decoded, _, _, err := decodeDurableRootPublication(raw)
	if err != nil || decoded.Identity == nil || decoded.Identity.SessionPolicy != publication.Identity.SessionPolicy {
		t.Fatalf("legacy durable root decode=%#v error=%v", decoded, err)
	}
	if _, _, _, err := canonicalDurableRootPublication(publication); !errors.Is(err, ErrInvalid) {
		t.Fatalf("legacy session policy became live publication error=%v", err)
	}
}

func TestDurableRootPublicationKeepsInspectOnlyProviderRuntimeFree(t *testing.T) {
	publication := Publication{
		Artifact: Artifact{
			ExtensionID: fixtureExtensionID, ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("a", 64), VersionID: 101,
		},
		Identity: &IdentityDeclaration{
			ContractVersion: "fixture.identity.contract@1",
			SessionPolicy:   "core.session.default",
			Providers: []Provider{{
				ID: fixtureExtensionID + ".provider", ContractVersion: fixtureExtensionID + ".provider@1",
				Kind: ProviderKindAuth, Handler: "legacy.auth",
			}},
		},
	}
	state := durableStateForPublication(t, publication, 41, 81)
	if err := ValidateDurablePublication(state, publication); err != nil {
		t.Fatalf("validate runtime-free provider publication: %v", err)
	}
	if len(state.RootTips) != 1 || strings.Contains(string(state.RootTips[0].PublicationJSON), "runtimeInstanceId") {
		t.Fatalf("durable inspect-only root leaked runtime: %#v", state.RootTips)
	}
	wantJSON := `{"artifact":{"extensionId":"fixture.identity","extensionVersion":"1.0.0","packageDigest":"` +
		strings.Repeat("a", 64) +
		`","versionId":101},"identity":{"contractVersion":"fixture.identity.contract@1","providers":[{"id":"fixture.identity.provider","contractVersion":"fixture.identity.provider@1","kind":"auth","handler":"legacy.auth"}],"sessionPolicy":"core.session.default"}}`
	if !bytes.Equal(state.RootTips[0].PublicationJSON, []byte(wantJSON)) ||
		state.RootTips[0].PublicationDigest != "da1b6bc527b617d9a0fd82ad96a21c47e7aa3bc918d2e4630e05ea2ca2321ac5" {
		t.Fatalf("legacy inspect-only root contract drifted: digest=%s json=%s",
			state.RootTips[0].PublicationDigest, state.RootTips[0].PublicationJSON)
	}
	desired, err := desiredDurableDeclarations(&publication)
	if err != nil || len(desired) != 1 ||
		desired[0].digest != "6f892ac3c79eee399d58e0231ab54e4369c294759bb2e7364ea0e4e036d66b0d" {
		t.Fatalf("legacy inspect-only provider leaf drifted: %#v, %v", desired, err)
	}
}

func TestDurableRootPublicationKeepsFullLegacyIdentityDigestsStable(t *testing.T) {
	publication := publicationStoreFixture(
		publicationStoreArtifact(101, "1.0.0", "a", "runtime-v1"), 1, nil,
	)
	state := durableStateForPublication(t, publication, 41, 81)
	if len(state.RootTips) != 1 ||
		state.RootTips[0].PublicationDigest != "d187b6237e0693d41296c5b4da5333e400d48a7750f6f180010e33a126aed115" {
		t.Fatalf("full legacy root digest drifted: %#v", state.RootTips)
	}
	wantLeaves := map[string]string{
		TombstoneKindPermission: "b956fa3979d2c47510ac38ba97df4a0ad62e08887d4b1e7a9d11b96a9798d850",
		TombstoneKindUserField:  "a7a32bd6499bcc50c7b07545bca7ff774a762f095a41a8f28fe61e1132fd0a4e",
		TombstoneKindProvider:   "240fe774d1ee9063ed2d2186491e2963f9e8d1a0256f542d2f7ead6605ebb4b4",
	}
	desired, err := desiredDurableDeclarations(&publication)
	if err != nil || len(desired) != len(wantLeaves) {
		t.Fatalf("full legacy leaves = %#v, %v", desired, err)
	}
	for _, declaration := range desired {
		if declaration.digest != wantLeaves[declaration.kind] {
			t.Fatalf("legacy %s leaf digest=%s", declaration.kind, declaration.digest)
		}
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
			SessionPolicy:   "core.session.default",
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

package identityregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestRegistryPublishesImmutableExactIdentityCatalog(t *testing.T) {
	registry := New()
	publication := testPublication(1)
	publication.Permissions[0].RecommendedRoles = []string{"operator", "member"}

	revision, err := registry.Publish(publication)
	if err != nil || revision != 1 {
		t.Fatalf("Publish() = %d, %v", revision, err)
	}
	permission, err := registry.ResolvePermission("fixture.identity.profile")
	if err != nil || permission.Artifact != publication.Artifact ||
		len(permission.RecommendedRoles) != 2 || permission.RecommendedRoles[0] != "member" {
		t.Fatalf("permission = %#v, %v", permission, err)
	}
	field, err := registry.ResolveUserField("fixture.identity.field.code")
	if err != nil || field.Artifact.RuntimeInstanceID != "runtime-1" || field.Schema != "fixture.identity.field.code.schema@1" {
		t.Fatalf("field = %#v, %v", field, err)
	}
	providers := registry.Providers(ProviderKindRisk)
	if len(providers) != 1 || providers[0].ID != "fixture.identity.provider.risk" {
		t.Fatalf("providers = %#v", providers)
	}

	// Returned documents are copies; callers cannot mutate the active graph.
	permission.RecommendedRoles[0] = "mutated"
	snapshot := registry.Snapshot()
	if snapshot.Permissions[0].RecommendedRoles[0] != "member" || snapshot.Digest == "" || snapshot.SchemaVersion != SchemaVersion {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	if replay, err := registry.Publish(publication); err != nil || replay != revision {
		t.Fatalf("exact replay = %d, %v", replay, err)
	}
	drift := testPublication(1)
	drift.Permissions[0].Description = "changed without an artifact change"
	if got, err := registry.Publish(drift); !errors.Is(err, ErrArtifactConflict) || got != revision {
		t.Fatalf("drift = %d, %v", got, err)
	}
}

func TestRegistryExactReplacementRetainsTombstonesAndFencesStaleRemoval(t *testing.T) {
	registry := New()
	source := testPublication(1)
	if _, err := registry.Publish(source); err != nil {
		t.Fatal(err)
	}
	target := testPublication(2)
	target.Permissions[0].ContractVersion = "fixture.identity.profile@2"
	target.Identity.ContractVersion = "fixture.identity.contract@2"
	target.Identity.UserFields[0].ContractVersion = "fixture.identity.field.code@2"
	target.Identity.Providers[0].ContractVersion = "fixture.identity.provider.risk@2"

	if revision, err := registry.PublishIfArtifact(source.Artifact, target); err != nil || revision != 2 {
		t.Fatalf("replace = %d, %v", revision, err)
	}
	snapshot := registry.Snapshot()
	if len(snapshot.Tombstones) != 3 {
		t.Fatalf("tombstones = %#v", snapshot.Tombstones)
	}
	if permission, err := registry.ResolvePermission("fixture.identity.profile"); err != nil || permission.ContractVersion != "fixture.identity.profile@2" {
		t.Fatalf("active permission = %#v, %v", permission, err)
	}
	if revision, removed, err := registry.Remove(source.Artifact); !errors.Is(err, ErrArtifactConflict) || removed || revision != 2 {
		t.Fatalf("stale remove = %d, %t, %v", revision, removed, err)
	}
	if revision, removed, err := registry.Remove(target.Artifact); err != nil || !removed || revision != 3 {
		t.Fatalf("exact remove = %d, %t, %v", revision, removed, err)
	}
	if len(registry.Snapshot().Tombstones) != 6 {
		t.Fatalf("final tombstones = %#v", registry.Snapshot().Tombstones)
	}
}

func TestRegistryNeverTurnsRoleSuggestionsIntoAuthority(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Publication)
	}{
		{name: "plugin assignment", mutate: func(value *Publication) { value.Permissions[0].AssignmentPolicy = "plugin" }},
		{name: "super admin recommendation", mutate: func(value *Publication) { value.Permissions[0].RecommendedRoles = []string{"super_admin"} }},
		{name: "undeclared field read", mutate: func(value *Publication) { value.Identity.UserFields[0].ReadPermission = "fixture.identity.undeclared" }},
		{name: "provider without runtime", mutate: func(value *Publication) { value.Artifact.RuntimeInstanceID = "" }},
		{name: "unsafe provider handler", mutate: func(value *Publication) { value.Identity.Providers[0].Handler = "https://foreign.invalid" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			publication := testPublication(1)
			test.mutate(&publication)
			if _, err := New().Publish(publication); !errors.Is(err, ErrInvalid) {
				t.Fatalf("Publish() error = %v", err)
			}
		})
	}
}

func TestRegistrySafeModeKeepsCoreAndRejectsThirdPartyWrites(t *testing.T) {
	registry := New()
	coreArtifact, err := NewCoreArtifact("core.identity", "1.0.0", strings.Repeat("c", 64))
	if err != nil {
		t.Fatal(err)
	}
	core := Publication{Artifact: coreArtifact, Permissions: []PermissionDefinition{{
		Key: "core.identity.access", ContractVersion: "core.identity.access@1", Label: "Access",
		Description: "Core access.", AssignmentPolicy: "host",
	}}}
	plugin := testPublication(1)
	if _, err := registry.ReplaceAll([]Publication{plugin, core}, false); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	if _, err := registry.ReplaceAllIfRevision(before.Revision, []Publication{plugin, core}, before.Tombstones, true); err != nil {
		t.Fatal(err)
	}
	safe := registry.Snapshot()
	if !safe.SafeMode || len(safe.Publications) != 1 || !safe.Publications[0].Artifact.Core || len(safe.Tombstones) != 3 {
		t.Fatalf("safe snapshot = %#v", safe)
	}
	if _, err := registry.Publish(plugin); !errors.Is(err, ErrSafeMode) {
		t.Fatalf("safe mode publish error = %v", err)
	}
	if _, err := registry.ReplaceAllIfRevision(safe.Revision, []Publication{plugin, core}, safe.Tombstones, false); err != nil {
		t.Fatal(err)
	}
	if restored := registry.Snapshot(); restored.SafeMode || len(restored.Publications) != 2 {
		t.Fatalf("restored snapshot = %#v", restored)
	}
}

func TestRegistryAllowsHostSealedCoreProviderWithoutPluginRuntime(t *testing.T) {
	artifact, err := NewCoreArtifact("core.identity", "1.0.0", strings.Repeat("c", 64))
	if err != nil {
		t.Fatal(err)
	}
	publication := Publication{Artifact: artifact, Identity: &IdentityDeclaration{
		ContractVersion: "core.identity.contract@1",
		Providers: []Provider{{
			ID: "core.identity.provider.auth", ContractVersion: "core.identity.provider.auth@1",
			Kind: ProviderKindAuth, Handler: "identity.auth", Priority: 100,
		}},
	}}
	if _, err := New().Publish(publication); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
}

func TestRegistryDurableTombstoneRejectsMismatchedOwnerNamespace(t *testing.T) {
	registry := New()
	publication := testPublication(1)
	tombstone := Tombstone{
		Kind: TombstoneKindPermission, ID: publication.Permissions[0].Key,
		ContractVersion: "fixture.identity.profile@1", OwnerExtensionID: "other.extension",
	}
	if _, err := registry.ReplaceAllIfRevision(0, []Publication{publication}, []Tombstone{tombstone}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ReplaceAllIfRevision() error = %v", err)
	}
}

func TestRegistryRejectsJSONForgedCoreArtifact(t *testing.T) {
	coreArtifact, err := NewCoreArtifact("core.identity", "1.0.0", strings.Repeat("c", 64))
	if err != nil {
		t.Fatal(err)
	}
	trusted := Publication{Artifact: coreArtifact, Permissions: []PermissionDefinition{{
		Key: "core.identity.access", ContractVersion: "core.identity.access@1", Label: "Access",
		Description: "Core access.", AssignmentPolicy: "host",
	}}}
	raw, err := json.Marshal(trusted)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Publication
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, err := New().Publish(decoded); !errors.Is(err, ErrInvalid) {
		t.Fatalf("decoded Core publication error = %v", err)
	}

	forged := testPublication(1)
	forged.Artifact = Artifact{
		ExtensionID: "core.forged", ExtensionVersion: "1.0.0",
		PackageDigest: strings.Repeat("f", 64), Core: true,
	}
	forged.Permissions[0].Key = "core.forged.access"
	forged.Permissions[0].ContractVersion = "core.forged.access@1"
	forged.Identity = nil
	if _, err := New().Publish(forged); !errors.Is(err, ErrInvalid) {
		t.Fatalf("forged Core publication error = %v", err)
	}
	thirdPartyCoreNamespace := Publication{Artifact: Artifact{
		ExtensionID: "core.thirdparty", ExtensionVersion: "1.0.0",
		PackageDigest: strings.Repeat("e", 64), VersionID: 1,
	}, Permissions: []PermissionDefinition{{
		Key: "core.thirdparty.access", ContractVersion: "core.thirdparty.access@1",
		Label: "Access", Description: "Spoofed Core access.", AssignmentPolicy: "host",
	}}}
	if _, err := New().Publish(thirdPartyCoreNamespace); !errors.Is(err, ErrInvalid) {
		t.Fatalf("third-party core namespace error = %v", err)
	}
}

func TestRegistryNamespaceCannotTakeOverReservedIdentity(t *testing.T) {
	publication := testPublication(1)
	publication.Artifact.ExtensionID = "other.identity"
	if _, err := New().ReplaceAllIfRevision(0, []Publication{publication}, []Tombstone{{
		Kind: TombstoneKindPermission, ID: "fixture.identity.profile",
		ContractVersion: "fixture.identity.profile@1", OwnerExtensionID: "fixture.identity",
	}}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("namespace takeover error = %v", err)
	}
}

func TestRegistryAppendOnlyTombstoneStopsNestedNamespaceTakeover(t *testing.T) {
	registry := New()
	owner := Publication{Artifact: Artifact{
		ExtensionID: "ab", ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("a", 64), VersionID: 1,
	}, Permissions: []PermissionDefinition{{
		Key: "ab.c.profile", ContractVersion: "ab.c.profile@1", Label: "Profile",
		Description: "Original nested permission.", AssignmentPolicy: "host",
	}}}
	if _, err := registry.Publish(owner); err != nil {
		t.Fatal(err)
	}
	if _, removed, err := registry.Remove(owner.Artifact); err != nil || !removed {
		t.Fatalf("Remove() = %t, %v", removed, err)
	}
	// A partial process-local reconcile cannot erase already observed history.
	removed := registry.Snapshot()
	if _, err := registry.ReplaceAllIfRevision(removed.Revision, nil, nil, false); err != nil {
		t.Fatal(err)
	}
	if len(registry.Snapshot().Tombstones) != 1 {
		t.Fatalf("tombstones = %#v", registry.Snapshot().Tombstones)
	}
	contender := Publication{Artifact: Artifact{
		ExtensionID: "ab.c", ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("b", 64), VersionID: 2,
	}, Permissions: []PermissionDefinition{{
		Key: "ab.c.profile", ContractVersion: "ab.c.profile@1", Label: "Profile",
		Description: "Take over nested permission.", AssignmentPolicy: "host",
	}}}
	if _, err := registry.Publish(contender); !errors.Is(err, ErrConflict) {
		t.Fatalf("nested namespace takeover error = %v", err)
	}
}

func TestRegistryProviderPriorityAndExactDisableFallback(t *testing.T) {
	registry := New()
	high := testPublication(1)
	low := testPublication(2)
	low.Artifact.ExtensionID = "other.identity"
	low.Permissions = nil
	low.Identity.ContractVersion = "other.identity.contract@1"
	low.Identity.UserFields = nil
	low.Identity.RiskHooks = nil
	low.Identity.Providers[0] = Provider{
		ID: "other.identity.provider.risk", ContractVersion: "other.identity.provider.risk@1",
		Kind: ProviderKindRisk, Handler: "identity.risk", Priority: 10,
	}
	if _, err := registry.ReplaceAll([]Publication{low, high}, false); err != nil {
		t.Fatal(err)
	}
	providers := registry.Providers(ProviderKindRisk)
	if len(providers) != 2 || providers[0].ID != "fixture.identity.provider.risk" || providers[1].ID != "other.identity.provider.risk" {
		t.Fatalf("ordered providers = %#v", providers)
	}
	if _, removed, err := registry.Remove(high.Artifact); err != nil || !removed {
		t.Fatalf("remove high provider = %t, %v", removed, err)
	}
	providers = registry.Providers(ProviderKindRisk)
	if len(providers) != 1 || providers[0].ID != "other.identity.provider.risk" {
		t.Fatalf("fallback providers = %#v", providers)
	}
	if _, err := registry.ResolveProvider("fixture.identity.provider.risk"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("disabled provider resolve error = %v", err)
	}
}

func TestRegistryRejectsRevisionAndArtifactRaces(t *testing.T) {
	registry := New()
	source := testPublication(1)
	if _, err := registry.Publish(source); err != nil {
		t.Fatal(err)
	}
	target := testPublication(2)
	if _, err := registry.PublishIfArtifact(source.Artifact, target); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.ReplaceAllIfRevision(1, []Publication{target}, nil, false); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale revision error = %v", err)
	}
	other := testPublication(3)
	if _, err := registry.PublishIfArtifact(source.Artifact, other); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("stale artifact error = %v", err)
	}
}

func TestRegistryBoundsWholeGraphAndUsesStrictSemVer(t *testing.T) {
	tooMany := make([]Publication, maxPublications+1)
	if _, err := New().ReplaceAll(tooMany, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("oversized publication graph error = %v", err)
	}

	publications := make([]Publication, 0, maxPermissionsTotal/maxPermissions+1)
	for publicationIndex := 0; publicationIndex < cap(publications); publicationIndex++ {
		extensionID := fmt.Sprintf("fixture.identity.graph_%d", publicationIndex)
		publication := Publication{Artifact: Artifact{
			ExtensionID: extensionID, ExtensionVersion: "1.0.0",
			PackageDigest: fmt.Sprintf("%064x", publicationIndex+1), VersionID: int64(publicationIndex + 1),
		}}
		for permissionIndex := 0; permissionIndex < maxPermissions; permissionIndex++ {
			key := fmt.Sprintf("%s.permission_%d", extensionID, permissionIndex)
			publication.Permissions = append(publication.Permissions, PermissionDefinition{
				Key: key, ContractVersion: key + "@1", Label: "Permission",
				Description: "Bounded permission catalog.", AssignmentPolicy: "host",
			})
		}
		publications = append(publications, publication)
	}
	if _, err := New().ReplaceAll(publications, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("oversized permission graph error = %v", err)
	}

	valid := Publication{Artifact: Artifact{
		ExtensionID: "fixture.identity.semver", ExtensionVersion: "1.2.3-rc.1+build.7",
		PackageDigest: strings.Repeat("a", 64), VersionID: 1,
	}}
	if _, err := New().Publish(valid); err != nil {
		t.Fatalf("strict SemVer publication error = %v", err)
	}
	valid.Artifact.ExtensionVersion = "01.2.3"
	valid.Artifact.VersionID = 2
	if _, err := New().Publish(valid); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid SemVer publication error = %v", err)
	}
}

func testPublication(version int) Publication {
	versionText := string(rune('0' + version))
	return Publication{
		Artifact: Artifact{
			ExtensionID: "fixture.identity", ExtensionVersion: versionText + ".0.0",
			PackageDigest: strings.Repeat(versionText, 64), VersionID: int64(version), RuntimeInstanceID: "runtime-" + versionText,
		},
		Permissions: []PermissionDefinition{{
			Key: "fixture.identity.profile", ContractVersion: "fixture.identity.profile@1", Label: "Profile",
			Description: "Manage identity profile fields.", RecommendedRoles: []string{"member"}, AssignmentPolicy: "host",
		}},
		Identity: &IdentityDeclaration{
			ContractVersion: "fixture.identity.contract@1", SessionPolicy: "core.session.default",
			RiskHooks: []string{"fixture.identity.risk.login"},
			UserFields: []UserField{{
				ID: "fixture.identity.field.code", ContractVersion: "fixture.identity.field.code@1", Type: "string",
				Schema: "fixture.identity.field.code.schema@1", ReadPermission: "fixture.identity.profile", WritePermission: "fixture.identity.profile",
			}},
			Providers: []Provider{{
				ID: "fixture.identity.provider.risk", ContractVersion: "fixture.identity.provider.risk@1",
				Kind: ProviderKindRisk, Handler: "identity.risk", Priority: 100,
			}},
		},
	}
}

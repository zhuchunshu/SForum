package identityregistry

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestActiveOrphanOwnersDetectsActiveRootAndLeaves(t *testing.T) {
	publication := mustOrphanTestPublication(t, "orphan.plugin", 11, "1.0.0", strings.Repeat("a", 64))
	raw, digest := mustOrphanRootJSON(t, publication)

	state := DurableState{
		RootTips: []DurableRootPublicationTip{{
			OwnerExtensionID: "orphan.plugin", Revision: 1, RegistryState: RegistryStateActive,
			ExtensionVersionID: 11, ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("a", 64),
			SchemaVersion: SchemaVersion, PublicationDigest: digest, PublicationJSON: raw,
			ActorUserID: 7, AuditEventID: 9,
		}},
		Owners: []DurableOwner{{
			IdentityKind: TombstoneKindPermission, StableID: "orphan.plugin.manage",
			OwnerExtensionID: "orphan.plugin",
		}},
		Tips: []DurableDeclarationTip{{
			IdentityKind: TombstoneKindPermission, StableID: "orphan.plugin.manage",
			OwnerExtensionID: "orphan.plugin", Revision: 1, RegistryState: RegistryStateActive,
			ExtensionVersionID: 11, ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("a", 64),
			ContractVersion: "orphan.plugin.permission.manage@1", DeclarationDigest: strings.Repeat("b", 64),
			ActorUserID: 7, AuditEventID: 9,
		}},
	}

	orphans, err := ActiveOrphanOwners(state, nil)
	if err != nil {
		t.Fatalf("ActiveOrphanOwners: %v", err)
	}
	if len(orphans) != 1 || orphans[0] != "orphan.plugin" {
		t.Fatalf("orphans = %#v", orphans)
	}

	// Expected enabled publisher is not treated as orphan.
	orphans, err = ActiveOrphanOwners(state, []string{"orphan.plugin"})
	if err != nil {
		t.Fatalf("ActiveOrphanOwners expected: %v", err)
	}
	if len(orphans) != 0 {
		t.Fatalf("expected no orphans, got %#v", orphans)
	}
}

func TestActiveOrphanOwnersDetectsIncompleteRetirementLeaves(t *testing.T) {
	// Root already tombstoned, leaf still active — the failure mode from force-delete.
	publication := mustOrphanTestPublication(t, "gone.plugin", 22, "1.0.0", strings.Repeat("c", 64))
	raw, digest := mustOrphanRootJSON(t, publication)

	state := DurableState{
		RootTips: []DurableRootPublicationTip{
			{
				OwnerExtensionID: "gone.plugin", Revision: 1, RegistryState: RegistryStateActive,
				ExtensionVersionID: 22, ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("c", 64),
				SchemaVersion: SchemaVersion, PublicationDigest: digest, PublicationJSON: raw,
				ActorUserID: 3, AuditEventID: 4,
			},
			{
				OwnerExtensionID: "gone.plugin", Revision: 2, RegistryState: RegistryStateTombstone,
				ExtensionVersionID: 22, ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("c", 64),
				SchemaVersion: SchemaVersion, PublicationDigest: digest, PublicationJSON: raw,
				ActorUserID: 3, AuditEventID: 4,
			},
		},
		Owners: []DurableOwner{{
			IdentityKind: TombstoneKindPermission, StableID: "gone.plugin.manage",
			OwnerExtensionID: "gone.plugin",
		}},
		Tips: []DurableDeclarationTip{{
			IdentityKind: TombstoneKindPermission, StableID: "gone.plugin.manage",
			OwnerExtensionID: "gone.plugin", Revision: 1, RegistryState: RegistryStateActive,
			ExtensionVersionID: 22, ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("c", 64),
			ContractVersion: "gone.plugin.permission.manage@1", DeclarationDigest: strings.Repeat("d", 64),
			ActorUserID: 3, AuditEventID: 4,
		}},
	}

	orphans, err := ActiveOrphanOwners(state, nil)
	if err != nil {
		t.Fatalf("ActiveOrphanOwners: %v", err)
	}
	if len(orphans) != 1 || orphans[0] != "gone.plugin" {
		t.Fatalf("orphans = %#v", orphans)
	}
}

func TestActiveOrphanOwnersRejectsInvalidExpectedIDs(t *testing.T) {
	if _, err := ActiveOrphanOwners(DurableState{}, []string{"Core.bad"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func mustOrphanTestPublication(t *testing.T, extensionID string, versionID int64, version, digest string) Publication {
	t.Helper()
	return Publication{
		Artifact: Artifact{
			ExtensionID: extensionID, ExtensionVersion: version, PackageDigest: digest, VersionID: versionID,
		},
		Permissions: []PermissionDefinition{{
			Key: extensionID + ".manage", ContractVersion: extensionID + ".permission.manage@1",
			Label: "Manage", Description: "fixture", AssignmentPolicy: "host",
		}},
	}
}

func mustOrphanRootJSON(t *testing.T, publication Publication) (json.RawMessage, string) {
	t.Helper()
	normalized, raw, digest, err := canonicalDurableRootPublication(publication)
	if err != nil {
		t.Fatal(err)
	}
	_ = normalized
	return append(json.RawMessage(nil), raw...), digest
}

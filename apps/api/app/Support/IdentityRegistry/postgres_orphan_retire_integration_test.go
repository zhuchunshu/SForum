package identityregistry

import (
	"errors"
	"testing"
)

// Startup treats non-expected owners as orphans even when the extension row
// still exists (disabled leftover / incomplete uninstall).
func TestRetireOrphanPublicationsRetiresPresentOwner(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	artifact := publicationStoreArtifact(101, "1.0.0", "a", "runtime-v1")
	publication := publicationStoreFixture(artifact, 1, []string{"member"})

	if _, err := fixture.store.Reconcile(fixture.ctx, ReconcilePublicationInput{
		ExtensionID: fixtureExtensionID, AllowedTarget: &artifact,
		Desired: &publication, ActorUserID: fixture.actorID, AuditEventID: 9101,
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	healed, err := fixture.store.RetireOrphanPublications(fixture.ctx, []string{fixtureExtensionID})
	if err != nil {
		t.Fatalf("RetireOrphanPublications: %v", err)
	}
	if err := ValidateDurableRetirement(healed, fixtureExtensionID); err != nil {
		t.Fatalf("retirement: %v", err)
	}
	if err := ValidateDurablePublicationSet(healed, nil); err != nil {
		t.Fatalf("empty expected set: %v", err)
	}

	// Idempotent: already retired owners are a no-op.
	again, err := fixture.store.RetireOrphanPublications(fixture.ctx, []string{fixtureExtensionID})
	if err != nil {
		t.Fatalf("second retire: %v", err)
	}
	if err := ValidateDurableRetirement(again, fixtureExtensionID); err != nil {
		t.Fatalf("second retirement: %v", err)
	}
}

// Force-deleted plugin residue: extension row gone, declaration tip still active.
// Trigger allows tombstone inserts without a live artifact when owner is absent.
func TestRetireOrphanPublicationsCompletesMissingOwnerActiveLeaf(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	artifact := publicationStoreArtifact(101, "1.0.0", "a", "runtime-v1")
	publication := publicationStoreFixture(artifact, 1, []string{"member"})

	if _, err := fixture.store.Reconcile(fixture.ctx, ReconcilePublicationInput{
		ExtensionID: fixtureExtensionID, AllowedTarget: &artifact,
		Desired: &publication, ActorUserID: fixture.actorID, AuditEventID: 9201,
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if _, err := fixture.store.Reconcile(fixture.ctx, ReconcilePublicationInput{
		ExtensionID: fixtureExtensionID, AllowedSource: &artifact,
		Desired: nil, ActorUserID: fixture.actorID, AuditEventID: 9202,
	}); err != nil {
		t.Fatalf("full retire: %v", err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `DELETE FROM extensions WHERE id = $1`, fixtureExtensionID); err != nil {
		t.Fatalf("delete extension: %v", err)
	}

	// Inject incomplete residue: active leaf tip while owner extension is gone.
	tx, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(fixture.ctx) }()
	if _, err := tx.Exec(fixture.ctx, `SET LOCAL session_replication_role = replica`); err != nil {
		t.Fatalf("session_replication_role: %v", err)
	}
	if _, err := tx.Exec(fixture.ctx, `
		INSERT INTO extension_identity_registry_declarations (
			identity_kind, stable_id, owner_extension_id, revision, registry_state,
			extension_version_id, extension_version, package_digest,
			contract_version, declaration_digest, actor_user_id, audit_event_id
		)
		SELECT identity_kind, stable_id, owner_extension_id, revision + 1, 'active',
		       extension_version_id, extension_version, package_digest,
		       contract_version, declaration_digest, $1, 9203
		FROM extension_identity_registry_declarations
		WHERE identity_kind = 'permission'
		  AND stable_id = $2
		ORDER BY revision DESC
		LIMIT 1
	`, fixture.actorID, fixturePermissionKey); err != nil {
		t.Fatalf("inject active leaf: %v", err)
	}
	if err := tx.Commit(fixture.ctx); err != nil {
		t.Fatalf("commit inject: %v", err)
	}

	durable, err := fixture.store.LoadDurableState(fixture.ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	orphans, err := ActiveOrphanOwners(durable, nil)
	if err != nil {
		t.Fatalf("ActiveOrphanOwners: %v", err)
	}
	if len(orphans) != 1 || orphans[0] != fixtureExtensionID {
		t.Fatalf("orphans = %#v", orphans)
	}
	// Fail-closed set validation still rejects the incomplete residue.
	if err := ValidateDurablePublicationSet(durable, nil); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("want ErrArtifactConflict before heal, got %v", err)
	}

	healed, err := fixture.store.RetireOrphanPublications(fixture.ctx, orphans)
	if err != nil {
		t.Fatalf("heal missing-owner leaf: %v", err)
	}
	if err := ValidateDurableRetirement(healed, fixtureExtensionID); err != nil {
		t.Fatalf("healed: %v", err)
	}
	if err := ValidateDurablePublicationSet(healed, nil); err != nil {
		t.Fatalf("set after heal: %v", err)
	}
}

func TestRetireOrphanPublicationsRejectsCoreOwner(t *testing.T) {
	fixture := newIdentityRegistryStoreFixture(t)
	if _, err := fixture.store.RetireOrphanPublications(fixture.ctx, []string{"core.system"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

package identityregistry

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPostgresLegacyPublicationAdoptionRequiresInstanceValidator(t *testing.T) {
	fixture := newLegacyAdoptionFixture(t)
	publication := legacyAdminSurfacePublication(fixture.versionID)

	// Ordinary NewPostgresStore is source-compatible but adoption-fail-closed.
	unconfigured := NewPostgresStore(fixture.pool)
	beforeRoot, beforeLeaf, beforeOwner, beforeCatalog, beforeSuggestion := legacyAdoptionCounts(t, fixture)
	_, err := unconfigured.AdoptLegacyPublications(fixture.ctx, []Publication{publication})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("unconfigured store error=%v want ErrInvalid", err)
	}
	afterRoot, afterLeaf, afterOwner, afterCatalog, afterSuggestion := legacyAdoptionCounts(t, fixture)
	if beforeRoot != afterRoot || beforeLeaf != afterLeaf || beforeOwner != afterOwner ||
		beforeCatalog != afterCatalog || beforeSuggestion != afterSuggestion {
		t.Fatal("unconfigured store wrote durable rows")
	}

	// Two stores with independent validators on the same pool stay isolated.
	reject := NewPostgresStoreWithStoredTrustImpactValidator(
		fixture.pool,
		func([]byte, string) error { return errors.New("reject") },
	)
	accept := NewPostgresStoreWithStoredTrustImpactValidator(fixture.pool, testValidateStoredTrustImpact)
	if _, err := reject.AdoptLegacyPublications(fixture.ctx, []Publication{publication}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("reject store error=%v", err)
	}
	// Reject path must leave zero writes so accept can still succeed.
	state, err := accept.AdoptLegacyPublications(fixture.ctx, []Publication{publication})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDurablePublication(state, publication); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresLegacyPublicationAdoptionPermissionOnlyShape(t *testing.T) {
	fixture := newLegacyAdoptionFixture(t)
	publication := legacyAdminSurfacePublication(fixture.versionID)
	beforeRolePermissions := fixture.countRolePermissions(t)

	state, err := fixture.store.AdoptLegacyPublications(fixture.ctx, []Publication{publication})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDurablePublication(state, publication); err != nil {
		t.Fatalf("adopted state validate: %v", err)
	}
	if err := ValidateDurablePublicationSet(state, []Publication{publication}); err != nil {
		t.Fatalf("adopted set validate: %v", err)
	}
	if len(state.RootTips) != 1 || state.RootTips[0].Revision != 1 ||
		state.RootTips[0].ActorUserID != fixture.actorID ||
		state.RootTips[0].AuditEventID != fixture.auditID {
		t.Fatalf("root tip=%#v actor=%d audit=%d", state.RootTips[0], fixture.actorID, fixture.auditID)
	}
	if len(state.Owners) != 1 || len(state.Tips) != 1 || state.Tips[0].Revision != 1 {
		t.Fatalf("leaf durable state=%#v", state)
	}

	var catalogCount, suggestionCount, nonPending, grantEvidence int
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*) FROM extension_permission_catalog
		WHERE permission_key = $1
	`, publication.Permissions[0].Key).Scan(&catalogCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*), count(*) FILTER (WHERE approval_state <> 'pending')
		FROM extension_permission_role_suggestions
		WHERE permission_key = $1
	`, publication.Permissions[0].Key).Scan(&suggestionCount, &nonPending); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*) FROM extension_permission_role_grants
	`).Scan(&grantEvidence); err != nil {
		t.Fatal(err)
	}
	if catalogCount != 1 || suggestionCount != 1 || nonPending != 0 || grantEvidence != 0 {
		t.Fatalf("catalog=%d suggestions=%d nonPending=%d grants=%d",
			catalogCount, suggestionCount, nonPending, grantEvidence)
	}
	if got := fixture.countRolePermissions(t); got != beforeRolePermissions {
		t.Fatalf("role_permissions changed: got %d want %d", got, beforeRolePermissions)
	}

	// Repeat adoption is idempotent: one root leaf owner, no duplicate catalog.
	replayed, err := fixture.store.AdoptLegacyPublications(fixture.ctx, []Publication{publication})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDurablePublication(replayed, publication); err != nil {
		t.Fatal(err)
	}
	var rootCount, leafCount, ownerCount, replayCatalog, replaySuggestion int
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*) FROM extension_identity_registry_publications
	`).Scan(&rootCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*) FROM extension_identity_registry_declarations
	`).Scan(&leafCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*) FROM extension_identity_registry_owners
	`).Scan(&ownerCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*) FROM extension_permission_catalog WHERE permission_key = $1
	`, publication.Permissions[0].Key).Scan(&replayCatalog); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*) FROM extension_permission_role_suggestions WHERE permission_key = $1
	`, publication.Permissions[0].Key).Scan(&replaySuggestion); err != nil {
		t.Fatal(err)
	}
	if rootCount != 1 || leafCount != 1 || ownerCount != 1 ||
		replayCatalog != 1 || replaySuggestion != 1 {
		t.Fatalf("replay counts root=%d leaf=%d owner=%d catalog=%d suggestion=%d",
			rootCount, leafCount, ownerCount, replayCatalog, replaySuggestion)
	}
}

func TestPostgresLegacyPublicationAdoptionUsesDurableLiveGrantSemantics(t *testing.T) {
	t.Run("grant issuer later disabled", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		publication := legacyAdminSurfacePublication(fixture.versionID)
		if _, err := fixture.pool.Exec(fixture.ctx, `
			UPDATE users SET status = 'disabled' WHERE id = $1
		`, fixture.actorID); err != nil {
			t.Fatal(err)
		}
		state, err := fixture.store.AdoptLegacyPublications(fixture.ctx, []Publication{publication})
		if err != nil {
			t.Fatalf("durable live grant from disabled issuer: %v", err)
		}
		if err := ValidateDurablePublication(state, publication); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("erased grant actor loses exact evidence", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		publication := legacyAdminSurfacePublication(fixture.versionID)
		if _, err := fixture.pool.Exec(fixture.ctx, `DELETE FROM users WHERE id = $1`, fixture.actorID); err != nil {
			t.Fatal(err)
		}
		assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrInvalid)
	})
}

func TestPostgresLegacyPublicationAdoptionRejectsEvidenceFailures(t *testing.T) {
	publication := legacyAdminSurfacePublication(0) // version filled per case

	t.Run("revoked grant", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		publication.Artifact.VersionID = fixture.versionID
		if _, err := fixture.pool.Exec(fixture.ctx, `
			UPDATE extension_trust_grants SET revoked_at = now(), revocation_reason = 'test'
			WHERE id = $1
		`, fixture.grantID); err != nil {
			t.Fatal(err)
		}
		assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrInvalid)
	})

	t.Run("mismatched audit actor", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		publication.Artifact.VersionID = fixture.versionID
		other := fixture.seedRoleManager(t, "active")
		if _, err := fixture.pool.Exec(fixture.ctx, `
			UPDATE audit_events SET actor_user_id = $2 WHERE id = $1
		`, fixture.auditID, other); err != nil {
			t.Fatal(err)
		}
		assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrInvalid)
	})

	t.Run("mismatched audit metadata digest", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		publication.Artifact.VersionID = fixture.versionID
		if _, err := fixture.pool.Exec(fixture.ctx, `
			UPDATE audit_events
			SET metadata = metadata || '{"impactDigest":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}'::jsonb
			WHERE id = $1
		`, fixture.auditID); err != nil {
			t.Fatalf("fixture audit update: %v", err)
		}
		assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrInvalid)
	})

	t.Run("audit action metadata mismatch", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		publication.Artifact.VersionID = fixture.versionID
		if _, err := fixture.pool.Exec(fixture.ctx, `
			UPDATE audit_events
			SET metadata = metadata || '{"action":"upgrade"}'::jsonb
			WHERE id = $1
		`, fixture.auditID); err != nil {
			t.Fatal(err)
		}
		assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrInvalid)
	})

	t.Run("audit created before grant", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		publication.Artifact.VersionID = fixture.versionID
		if _, err := fixture.pool.Exec(fixture.ctx, `
			UPDATE audit_events SET created_at = now() - interval '1 hour' WHERE id = $1
		`, fixture.auditID); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.pool.Exec(fixture.ctx, `
			UPDATE extension_trust_grants SET granted_at = now() WHERE id = $1
		`, fixture.grantID); err != nil {
			t.Fatal(err)
		}
		assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrInvalid)
	})

	t.Run("duplicate live grant", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		publication.Artifact.VersionID = fixture.versionID
		// Second live enable grant with a different impact digest.
		otherDigest := strings.Repeat("1", 64)
		if _, err := fixture.pool.Exec(fixture.ctx, `
			INSERT INTO extension_trust_grants (
				extension_id, extension_version, package_digest, action,
				artifact_digests, impact_document, impact_digest, granted_by_user_id
			) VALUES (
				$1, '1.0.0', $2, 'enable',
				'{}'::jsonb, $3::jsonb, $4, $5
			)
		`, fixture.extensionID, publication.Artifact.PackageDigest,
			string(fixture.impactDocument), otherDigest, fixture.actorID); err != nil {
			t.Fatal(err)
		}
		assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrInvalid)
	})

	t.Run("duplicate audit", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		publication.Artifact.VersionID = fixture.versionID
		if _, err := fixture.pool.Exec(fixture.ctx, `
			INSERT INTO audit_events (actor_user_id, action, metadata)
			SELECT actor_user_id, action, metadata FROM audit_events WHERE id = $1
		`, fixture.auditID); err != nil {
			t.Fatal(err)
		}
		assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrInvalid)
	})

	t.Run("mismatched impact declaration", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		publication.Artifact.VersionID = fixture.versionID
		// Rebuild impact with drifted label and correct digest so full integrity
		// passes but identity surface no longer matches desired publication.
		drifted := publication
		drifted.Permissions = append([]PermissionDefinition(nil), publication.Permissions...)
		drifted.Permissions[0].Label = "changed label"
		body, digest := mustCanonicalTrustImpactForPublication(t, drifted, nil)
		if _, err := fixture.pool.Exec(fixture.ctx, `
			UPDATE extension_trust_grants
			SET impact_document = $2::jsonb, impact_digest = $3
			WHERE id = $1
		`, fixture.grantID, string(body), digest); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.pool.Exec(fixture.ctx, `
			UPDATE audit_events
			SET metadata = jsonb_set(metadata, '{impactDigest}', to_jsonb($2::text))
			WHERE id = $1
		`, fixture.auditID, digest); err != nil {
			t.Fatal(err)
		}
		assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrInvalid)
	})

	t.Run("full impact top-level forgery", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		publication.Artifact.VersionID = fixture.versionID
		// Subtree-correct permissions but action=upgrade with original digest claim.
		var wire testTrustImpactWire
		if err := json.Unmarshal(fixture.impactDocument, &wire); err != nil {
			t.Fatal(err)
		}
		wire.Action = "upgrade"
		// Keep original digest field so integrity recomputation fails.
		body, err := json.Marshal(wire)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.pool.Exec(fixture.ctx, `
			UPDATE extension_trust_grants SET impact_document = $2::jsonb WHERE id = $1
		`, fixture.grantID, string(body)); err != nil {
			t.Fatal(err)
		}
		assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrInvalid)
	})

	t.Run("disabled extension", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		publication.Artifact.VersionID = fixture.versionID
		if _, err := fixture.pool.Exec(fixture.ctx, `
			UPDATE extensions SET status = 'disabled' WHERE id = $1
		`, fixture.extensionID); err != nil {
			t.Fatal(err)
		}
		assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrArtifactConflict)
	})

	t.Run("stale active version", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		stale := publication
		stale.Artifact.VersionID = fixture.versionID
		stale.Artifact.PackageDigest = strings.Repeat("c", 64)
		assertLegacyAdoptionZeroWrites(t, fixture, []Publication{stale}, ErrArtifactConflict)
	})

	t.Run("partial owner history", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		publication.Artifact.VersionID = fixture.versionID
		if _, err := fixture.pool.Exec(fixture.ctx, `
			INSERT INTO extension_identity_registry_owners (
				identity_kind, stable_id, owner_extension_id
			) VALUES ('permission', $1, $2)
		`, publication.Permissions[0].Key, fixture.extensionID); err != nil {
			t.Fatal(err)
		}
		assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrInvalid)
	})

	t.Run("partial root history", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		publication.Artifact.VersionID = fixture.versionID
		// Prior enable+disable leaves a tombstoned tip: adoption must never
		// repair or re-append history for that owner.
		artifact := publication.Artifact
		if _, err := fixture.store.Reconcile(fixture.ctx, ReconcilePublicationInput{
			ExtensionID: fixture.extensionID, AllowedTarget: &artifact, Desired: &publication,
			ActorUserID: fixture.actorID, AuditEventID: fixture.auditID,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.store.Reconcile(fixture.ctx, ReconcilePublicationInput{
			ExtensionID: fixture.extensionID, AllowedSource: &artifact,
			ActorUserID: fixture.actorID, AuditEventID: fixture.auditID,
		}); err != nil {
			t.Fatal(err)
		}
		assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrStale)
	})

	t.Run("host permission collision preflight", func(t *testing.T) {
		fixture := newLegacyAdoptionFixture(t)
		publication.Artifact.VersionID = fixture.versionID
		if _, err := fixture.pool.Exec(fixture.ctx, `
			INSERT INTO permissions (key, module, description)
			VALUES ($1, 'host', 'pre-existing host permission')
		`, publication.Permissions[0].Key); err != nil {
			t.Fatal(err)
		}
		assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrConflict)
		// Zero roots after preflight failure.
		var rootCount int
		if err := fixture.pool.QueryRow(fixture.ctx, `
			SELECT count(*) FROM extension_identity_registry_publications
		`).Scan(&rootCount); err != nil {
			t.Fatal(err)
		}
		if rootCount != 0 {
			t.Fatalf("host collision wrote roots=%d", rootCount)
		}
	})
}

func TestPostgresLegacyPublicationAdoptionBatchAllOrNone(t *testing.T) {
	fixture := newLegacyAdoptionBatchFixture(t)
	first := fixture.plugins[0]
	second := fixture.plugins[1]

	t.Run("cross-owner host permission conflict", func(t *testing.T) {
		// Namespace rules prevent two plugins from claiming one stable id in the
		// same batch. Cross-owner fail-closed is Host permission collision on one
		// member: the whole batch must write zero roots/owners/tips.
		if _, err := fixture.pool.Exec(fixture.ctx, `
			INSERT INTO permissions (key, module, description)
			VALUES ($1, 'host', 'pre-existing host permission')
		`, second.publication.Permissions[0].Key); err != nil {
			t.Fatal(err)
		}
		assertLegacyAdoptionZeroWrites(t, fixture.legacyAdoptionFixture, []Publication{
			first.publication, second.publication,
		}, ErrConflict)
		var roots, owners, tips int
		if err := fixture.pool.QueryRow(fixture.ctx, `
			SELECT
			  (SELECT count(*) FROM extension_identity_registry_publications),
			  (SELECT count(*) FROM extension_identity_registry_owners),
			  (SELECT count(*) FROM extension_identity_registry_declarations)
		`).Scan(&roots, &owners, &tips); err != nil {
			t.Fatal(err)
		}
		if roots != 0 || owners != 0 || tips != 0 {
			t.Fatalf("cross-owner batch wrote roots=%d owners=%d tips=%d", roots, owners, tips)
		}
	})

	t.Run("second missing evidence leaves first unwritten", func(t *testing.T) {
		// Fresh fixture: revoke only the second grant.
		batch := newLegacyAdoptionBatchFixture(t)
		if _, err := batch.pool.Exec(batch.ctx, `
			UPDATE extension_trust_grants SET revoked_at = now(), revocation_reason = 'batch'
			WHERE id = $1
		`, batch.plugins[1].grantID); err != nil {
			t.Fatal(err)
		}
		assertLegacyAdoptionZeroWrites(t, batch.legacyAdoptionFixture, []Publication{
			batch.plugins[0].publication, batch.plugins[1].publication,
		}, ErrInvalid)
		// Explicitly assert first plugin also has zero roots/owners/tips.
		var firstRoots, firstOwners int
		if err := batch.pool.QueryRow(batch.ctx, `
			SELECT count(*) FROM extension_identity_registry_publications
			WHERE owner_extension_id = $1
		`, batch.plugins[0].extensionID).Scan(&firstRoots); err != nil {
			t.Fatal(err)
		}
		if err := batch.pool.QueryRow(batch.ctx, `
			SELECT count(*) FROM extension_identity_registry_owners
			WHERE owner_extension_id = $1
		`, batch.plugins[0].extensionID).Scan(&firstOwners); err != nil {
			t.Fatal(err)
		}
		if firstRoots != 0 || firstOwners != 0 {
			t.Fatalf("first plugin leaked roots=%d owners=%d", firstRoots, firstOwners)
		}
	})

	t.Run("happy path batch", func(t *testing.T) {
		batch := newLegacyAdoptionBatchFixture(t)
		pubs := []Publication{batch.plugins[0].publication, batch.plugins[1].publication}
		state, err := batch.store.AdoptLegacyPublications(batch.ctx, pubs)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateDurablePublicationSet(state, pubs); err != nil {
			t.Fatal(err)
		}
		var roots, owners, tips, grants int
		if err := batch.pool.QueryRow(batch.ctx, `
			SELECT
			  (SELECT count(*) FROM extension_identity_registry_publications),
			  (SELECT count(*) FROM extension_identity_registry_owners),
			  (SELECT count(*) FROM extension_identity_registry_declarations),
			  (SELECT count(*) FROM extension_permission_role_grants)
		`).Scan(&roots, &owners, &tips, &grants); err != nil {
			t.Fatal(err)
		}
		if roots != 2 || owners != 2 || tips != 2 || grants != 0 {
			t.Fatalf("batch durable roots=%d owners=%d tips=%d grants=%d", roots, owners, tips, grants)
		}
	})
}

func TestPostgresLegacyPublicationAdoptionConcurrentIdempotent(t *testing.T) {
	fixture := newLegacyAdoptionFixture(t)
	publication := legacyAdminSurfacePublication(fixture.versionID)

	ctx, cancel := context.WithTimeout(fixture.ctx, 15*time.Second)
	defer cancel()
	start := make(chan struct{})
	var wait sync.WaitGroup
	results := make([]error, 8)
	wait.Add(len(results))
	for index := range results {
		go func(i int) {
			defer wait.Done()
			<-start
			_, results[i] = fixture.store.AdoptLegacyPublications(ctx, []Publication{publication})
		}(index)
	}
	close(start)
	wait.Wait()
	if ctx.Err() != nil {
		t.Fatalf("concurrent adoption deadline: %v", ctx.Err())
	}
	for i, err := range results {
		if err != nil {
			t.Fatalf("worker %d error=%v", i, err)
		}
	}

	var rootCount, leafCount, ownerCount, catalogCount, suggestionCount int
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*) FROM extension_identity_registry_publications
	`).Scan(&rootCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*) FROM extension_identity_registry_declarations
	`).Scan(&leafCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*) FROM extension_identity_registry_owners
	`).Scan(&ownerCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*) FROM extension_permission_catalog
	`).Scan(&catalogCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*) FROM extension_permission_role_suggestions
	`).Scan(&suggestionCount); err != nil {
		t.Fatal(err)
	}
	if rootCount != 1 || leafCount != 1 || ownerCount != 1 || catalogCount != 1 || suggestionCount != 1 {
		t.Fatalf("concurrent counts root=%d leaf=%d owner=%d catalog=%d suggestion=%d",
			rootCount, leafCount, ownerCount, catalogCount, suggestionCount)
	}
	loaded, err := fixture.store.LoadDurableState(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDurablePublication(loaded, publication); err != nil {
		t.Fatalf("post-concurrent validate: %v", err)
	}
}

func TestPostgresLegacyPublicationAdoptionRevokeWins(t *testing.T) {
	fixture := newLegacyAdoptionFixture(t)
	publication := legacyAdminSurfacePublication(fixture.versionID)

	// Hold the grant row so adoption cannot finish before revoke becomes visible.
	// Barrier channel replaces sleep-only synchronization.
	hold, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = hold.Rollback(context.Background()) }()
	var grantID int64
	if err := hold.QueryRow(fixture.ctx, `
		SELECT id FROM extension_trust_grants WHERE id = $1 FOR UPDATE
	`, fixture.grantID).Scan(&grantID); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(fixture.ctx, 10*time.Second)
	defer cancel()
	blocked := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		// Adoption will block on grant FOR UPDATE; signal once the worker starts.
		close(blocked)
		_, adoptErr := fixture.store.AdoptLegacyPublications(ctx, []Publication{publication})
		done <- adoptErr
	}()
	<-blocked

	// Wait until the adopter is waiting on the grant lock (pg_locks), then revoke.
	deadline := time.Now().Add(3 * time.Second)
	for {
		var waiting int
		if err := fixture.pool.QueryRow(fixture.ctx, `
			SELECT count(*) FROM pg_locks
			WHERE locktype = 'transactionid' AND granted = false
		`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting > 0 || time.Now().After(deadline) {
			break
		}
		// Deterministic lock probe; tiny yield only when lock table is still empty.
		time.Sleep(5 * time.Millisecond)
	}
	if _, err := hold.Exec(fixture.ctx, `
		UPDATE extension_trust_grants
		SET revoked_at = now(), revocation_reason = 'concurrent-revoke'
		WHERE id = $1
	`, fixture.grantID); err != nil {
		t.Fatal(err)
	}
	if err := hold.Commit(fixture.ctx); err != nil {
		t.Fatal(err)
	}

	select {
	case adoptErr := <-done:
		if !errors.Is(adoptErr, ErrInvalid) {
			t.Fatalf("revoke-wins adoption error=%v", adoptErr)
		}
	case <-ctx.Done():
		t.Fatal("adoption did not finish after revoke")
	}
	assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrInvalid)
}

func TestPostgresLegacyPublicationAdoptionAuditShareLockBlocksMutation(t *testing.T) {
	fixture := newLegacyAdoptionFixture(t)
	publication := legacyAdminSurfacePublication(fixture.versionID)

	// Hold the audit row with FOR UPDATE (conflicts with FOR SHARE) so adoption
	// cannot validate until the concurrent mutator finishes.
	hold, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = hold.Rollback(context.Background()) }()
	var auditID int64
	if err := hold.QueryRow(fixture.ctx, `
		SELECT id FROM audit_events WHERE id = $1 FOR UPDATE
	`, fixture.auditID).Scan(&auditID); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(fixture.ctx, 10*time.Second)
	defer cancel()
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(started)
		_, adoptErr := fixture.store.AdoptLegacyPublications(ctx, []Publication{publication})
		done <- adoptErr
	}()
	<-started

	// Delete the audit while adoption is blocked on FOR SHARE.
	if _, err := hold.Exec(fixture.ctx, `
		DELETE FROM audit_events WHERE id = $1
	`, fixture.auditID); err != nil {
		t.Fatal(err)
	}
	if err := hold.Commit(fixture.ctx); err != nil {
		t.Fatal(err)
	}

	select {
	case adoptErr := <-done:
		if !errors.Is(adoptErr, ErrInvalid) {
			t.Fatalf("audit-delete race adoption error=%v", adoptErr)
		}
	case <-ctx.Done():
		t.Fatal("adoption did not finish after audit delete")
	}
	assertLegacyAdoptionZeroWrites(t, fixture, []Publication{publication}, ErrInvalid)
}

func TestPostgresLegacyPublicationAdoptionWinsThenRevokeKeepsInertOwnership(t *testing.T) {
	fixture := newLegacyAdoptionFixture(t)
	publication := legacyAdminSurfacePublication(fixture.versionID)

	state, err := fixture.store.AdoptLegacyPublications(fixture.ctx, []Publication{publication})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDurablePublication(state, publication); err != nil {
		t.Fatal(err)
	}

	// Revoke after adoption commits: durable inert ownership remains exact.
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE extension_trust_grants
		SET revoked_at = now(), revocation_reason = 'post-adoption', revoked_by_user_id = $2
		WHERE id = $1
	`, fixture.grantID, fixture.actorID); err != nil {
		t.Fatal(err)
	}
	var revokedAt *time.Time
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT revoked_at FROM extension_trust_grants WHERE id = $1
	`, fixture.grantID).Scan(&revokedAt); err != nil {
		t.Fatal(err)
	}
	if revokedAt == nil {
		t.Fatal("grant must be revoked")
	}

	loaded, err := fixture.store.LoadDurableState(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDurablePublication(loaded, publication); err != nil {
		t.Fatalf("durable ownership after revoke: %v", err)
	}
	var roleGrants int
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT count(*) FROM extension_permission_role_grants
	`).Scan(&roleGrants); err != nil {
		t.Fatal(err)
	}
	if roleGrants != 0 {
		t.Fatalf("role grants after adoption+revoke=%d", roleGrants)
	}
	// Runtime trust remains denied by existing trust semantics (no live grant).
	var live bool
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT EXISTS (
			SELECT 1 FROM extension_trust_grants
			WHERE extension_id = $1 AND extension_version = $2
			  AND package_digest = $3 AND action = 'enable'
			  AND impact_digest = $4 AND revoked_at IS NULL
		)
	`, fixture.extensionID, publication.Artifact.ExtensionVersion,
		publication.Artifact.PackageDigest, fixture.impactDigest).Scan(&live); err != nil {
		t.Fatal(err)
	}
	if live {
		t.Fatal("live grant must be absent after revoke")
	}
}

type legacyAdoptionFixture struct {
	*identityRegistryStoreFixture
	extensionID    string
	versionID      int64
	grantID        int64
	auditID        int64
	impactDigest   string
	impactDocument []byte
}

type legacyAdoptionPlugin struct {
	extensionID string
	versionID   int64
	grantID     int64
	auditID     int64
	publication Publication
}

type legacyAdoptionBatchFixture struct {
	*legacyAdoptionFixture
	plugins []legacyAdoptionPlugin
}

func newLegacyAdoptionFixture(t *testing.T) *legacyAdoptionFixture {
	t.Helper()
	base := newIdentityRegistryStoreFixture(t)
	// Per-instance test verifier: never package-global, parallel-safe.
	base.store = NewPostgresStoreWithStoredTrustImpactValidator(base.pool, testValidateStoredTrustImpact)
	if err := ensureLegacyTrustGrantTable(base.ctx, base); err != nil {
		t.Fatal(err)
	}

	const (
		extensionID = "sforum.admin-surface-reference"
		versionID   = int64(5866)
		digest      = "81b964f80707b257f6f401faffb07fe0f0a6aa6b5833a6fab0cedaab77b3324f"
	)
	publication := legacyAdminSurfacePublication(versionID)
	impactDocument, impactDigest := mustCanonicalTrustImpactForPublication(t, publication, nil)
	if _, err := base.pool.Exec(base.ctx, `
		INSERT INTO extensions (id, type, name, status)
		VALUES ($1, 'plugin', 'Admin Surface Reference', 'enabled')
	`, extensionID); err != nil {
		t.Fatal(err)
	}
	if _, err := base.pool.Exec(base.ctx, `
		INSERT INTO extension_versions (
			id, extension_id, version, manifest, package_path, package_digest
		) VALUES ($1, $2, '1.0.0', '{}'::jsonb, '/tmp/admin-surface', $3)
	`, versionID, extensionID, digest); err != nil {
		t.Fatal(err)
	}
	if _, err := base.pool.Exec(base.ctx, `
		UPDATE extensions SET active_version_id = $2 WHERE id = $1
	`, extensionID, versionID); err != nil {
		t.Fatal(err)
	}

	var grantID int64
	if err := base.pool.QueryRow(base.ctx, `
		INSERT INTO extension_trust_grants (
			extension_id, extension_version, package_digest, action,
			artifact_digests, impact_document, impact_digest, granted_by_user_id
		) VALUES (
			$1, '1.0.0', $2, 'enable',
			'{}'::jsonb, $3::jsonb, $4, $5
		)
		RETURNING id
	`, extensionID, digest, string(impactDocument), impactDigest, base.actorID).Scan(&grantID); err != nil {
		t.Fatal(err)
	}

	metadata, err := json.Marshal(map[string]any{
		"extensionId": extensionID, "version": "1.0.0",
		"packageDigest": digest, "impactDigest": impactDigest, "action": "enable",
	})
	if err != nil {
		t.Fatal(err)
	}
	var auditID int64
	if err := base.pool.QueryRow(base.ctx, `
		INSERT INTO audit_events (actor_user_id, action, metadata)
		VALUES ($1, 'extension.trust_grant', $2::jsonb)
		RETURNING id
	`, base.actorID, string(metadata)).Scan(&auditID); err != nil {
		t.Fatal(err)
	}

	return &legacyAdoptionFixture{
		identityRegistryStoreFixture: base,
		extensionID:                  extensionID,
		versionID:                    versionID,
		grantID:                      grantID,
		auditID:                      auditID,
		impactDigest:                 impactDigest,
		impactDocument:               impactDocument,
	}
}

func newLegacyAdoptionBatchFixture(t *testing.T) *legacyAdoptionBatchFixture {
	t.Helper()
	base := newLegacyAdoptionFixture(t)
	// First plugin already seeded by newLegacyAdoptionFixture.
	plugins := []legacyAdoptionPlugin{{
		extensionID: base.extensionID,
		versionID:   base.versionID,
		grantID:     base.grantID,
		auditID:     base.auditID,
		publication: legacyAdminSurfacePublication(base.versionID),
	}}

	// Second enabled plugin with its own exact grant+audit evidence.
	const (
		extensionID = "sforum.legacy-batch-second"
		versionID   = int64(5867)
		digest      = "91b964f80707b257f6f401faffb07fe0f0a6aa6b5833a6fab0cedaab77b3324f"
	)
	publication := Publication{
		Artifact: Artifact{
			ExtensionID: extensionID, ExtensionVersion: "1.0.0",
			PackageDigest: digest, VersionID: versionID,
		},
		Permissions: []PermissionDefinition{{
			Key: extensionID + ".manage", ContractVersion: extensionID + ".permission.manage@1",
			Label: "Second manage", Description: "Second plugin manage",
			RecommendedRoles: []string{"administrator"}, AssignmentPolicy: "host",
		}},
	}
	impactDocument, impactDigest := mustCanonicalTrustImpactForPublication(t, publication, nil)
	if _, err := base.pool.Exec(base.ctx, `
		INSERT INTO extensions (id, type, name, status)
		VALUES ($1, 'plugin', 'Legacy Batch Second', 'enabled')
	`, extensionID); err != nil {
		t.Fatal(err)
	}
	if _, err := base.pool.Exec(base.ctx, `
		INSERT INTO extension_versions (
			id, extension_id, version, manifest, package_path, package_digest
		) VALUES ($1, $2, '1.0.0', '{}'::jsonb, '/tmp/batch-second', $3)
	`, versionID, extensionID, digest); err != nil {
		t.Fatal(err)
	}
	if _, err := base.pool.Exec(base.ctx, `
		UPDATE extensions SET active_version_id = $2 WHERE id = $1
	`, extensionID, versionID); err != nil {
		t.Fatal(err)
	}
	var grantID int64
	if err := base.pool.QueryRow(base.ctx, `
		INSERT INTO extension_trust_grants (
			extension_id, extension_version, package_digest, action,
			artifact_digests, impact_document, impact_digest, granted_by_user_id
		) VALUES (
			$1, '1.0.0', $2, 'enable',
			'{}'::jsonb, $3::jsonb, $4, $5
		)
		RETURNING id
	`, extensionID, digest, string(impactDocument), impactDigest, base.actorID).Scan(&grantID); err != nil {
		t.Fatal(err)
	}
	metadata, err := json.Marshal(map[string]any{
		"extensionId": extensionID, "version": "1.0.0",
		"packageDigest": digest, "impactDigest": impactDigest, "action": "enable",
	})
	if err != nil {
		t.Fatal(err)
	}
	var auditID int64
	if err := base.pool.QueryRow(base.ctx, `
		INSERT INTO audit_events (actor_user_id, action, metadata)
		VALUES ($1, 'extension.trust_grant', $2::jsonb)
		RETURNING id
	`, base.actorID, string(metadata)).Scan(&auditID); err != nil {
		t.Fatal(err)
	}
	plugins = append(plugins, legacyAdoptionPlugin{
		extensionID: extensionID, versionID: versionID,
		grantID: grantID, auditID: auditID, publication: publication,
	})
	return &legacyAdoptionBatchFixture{
		legacyAdoptionFixture: base,
		plugins:               plugins,
	}
}

func ensureLegacyTrustGrantTable(ctx context.Context, fixture *identityRegistryStoreFixture) error {
	_, err := fixture.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS extension_trust_grants (
			id BIGSERIAL PRIMARY KEY,
			extension_id TEXT NOT NULL,
			extension_version TEXT NOT NULL CHECK (extension_version <> ''),
			package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
			action TEXT NOT NULL
			  CHECK (action IN ('enable', 'upgrade', 'frontend_import', 'authority_change')),
			artifact_digests JSONB NOT NULL DEFAULT '{}'::jsonb
			  CHECK (jsonb_typeof(artifact_digests) = 'object'),
			impact_document JSONB NOT NULL
			  CHECK (jsonb_typeof(impact_document) = 'object'),
			impact_digest TEXT NOT NULL CHECK (impact_digest ~ '^[0-9a-f]{64}$'),
			granted_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
			granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			revoked_at TIMESTAMPTZ,
			revoked_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
			revocation_reason TEXT NOT NULL DEFAULT ''
		);
		CREATE UNIQUE INDEX IF NOT EXISTS extension_trust_grants_live_exact_idx
		  ON extension_trust_grants (
		    extension_id, extension_version, package_digest, action, impact_digest
		  )
		  WHERE revoked_at IS NULL;
	`)
	return err
}

func legacyAdminSurfacePublication(versionID int64) Publication {
	return adminSurfaceReferencePermissionOnlyPublication(versionID)
}

func assertLegacyAdoptionZeroWrites(
	t *testing.T,
	fixture *legacyAdoptionFixture,
	publications []Publication,
	want error,
) {
	t.Helper()
	beforeRoot, beforeLeaf, beforeOwner, beforeCatalog, beforeSuggestion := legacyAdoptionCounts(t, fixture)
	_, err := fixture.store.AdoptLegacyPublications(fixture.ctx, publications)
	if !errors.Is(err, want) {
		t.Fatalf("adoption error=%v want %v", err, want)
	}
	afterRoot, afterLeaf, afterOwner, afterCatalog, afterSuggestion := legacyAdoptionCounts(t, fixture)
	if beforeRoot != afterRoot || beforeLeaf != afterLeaf || beforeOwner != afterOwner ||
		beforeCatalog != afterCatalog || beforeSuggestion != afterSuggestion {
		t.Fatalf("adoption wrote durable rows: before=(%d,%d,%d,%d,%d) after=(%d,%d,%d,%d,%d)",
			beforeRoot, beforeLeaf, beforeOwner, beforeCatalog, beforeSuggestion,
			afterRoot, afterLeaf, afterOwner, afterCatalog, afterSuggestion)
	}
}

func legacyAdoptionCounts(t *testing.T, fixture *legacyAdoptionFixture) (int, int, int, int, int) {
	t.Helper()
	var root, leaf, owner, catalog, suggestion int
	queries := []struct {
		sql  string
		dest *int
	}{
		{`SELECT count(*) FROM extension_identity_registry_publications`, &root},
		{`SELECT count(*) FROM extension_identity_registry_declarations`, &leaf},
		{`SELECT count(*) FROM extension_identity_registry_owners`, &owner},
		{`SELECT count(*) FROM extension_permission_catalog`, &catalog},
		{`SELECT count(*) FROM extension_permission_role_suggestions`, &suggestion},
	}
	for _, query := range queries {
		if err := fixture.pool.QueryRow(fixture.ctx, query.sql).Scan(query.dest); err != nil {
			t.Fatal(err)
		}
	}
	return root, leaf, owner, catalog, suggestion
}

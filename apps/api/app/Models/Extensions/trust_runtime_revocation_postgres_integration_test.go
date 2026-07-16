package extensions

import (
	"strings"
	"testing"
)

func TestPostgresExecutableTrustRevokePublishesRuntimeRemovalAtomically(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "trust_store_revoke")
	if _, err := fixture.pool.Exec(fixture.ctx, `
		CREATE TABLE extension_trust_grants (
			extension_id TEXT NOT NULL,
			revoked_at TIMESTAMPTZ,
			revoked_by_user_id BIGINT,
			revocation_reason TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE extension_trust_challenges (
			extension_id TEXT NOT NULL,
			consumed_at TIMESTAMPTZ,
			invalidated_at TIMESTAMPTZ,
			invalidation_reason TEXT NOT NULL DEFAULT ''
		);
		INSERT INTO extension_trust_grants (extension_id) VALUES ('fixture.plugin');
		INSERT INTO extension_trust_challenges (extension_id) VALUES ('fixture.plugin');
	`); err != nil {
		t.Fatal(err)
	}
	seed, err := fixture.store.publishPluginRuntimePublication(
		fixture.ctx, PluginRuntimePublicationStartupReconcile, 0,
		[]PluginRuntimeMember{fixture.firstMember(), fixture.secondMember()},
	)
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresExecutableTrustStore(fixture.pool)
	if err := store.RevokeAll(fixture.ctx, "fixture.plugin", 0, "operator_revoked"); err != nil {
		t.Fatal(err)
	}
	latest, err := fixture.store.LatestPluginRuntimePublication(fixture.ctx)
	if err != nil || latest.Revision <= seed.Revision || latest.Reason != PluginRuntimePublicationRecovery || latest.ActorUserID != 0 {
		t.Fatalf("latest=%+v err=%v", latest, err)
	}
	assertPluginRuntimePublicationMembers(t, latest, fixture.secondMember())
	var grantRevoked, challengeInvalidated bool
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT revoked_at IS NOT NULL AND revocation_reason = 'operator_revoked'
		FROM extension_trust_grants WHERE extension_id = 'fixture.plugin'
	`).Scan(&grantRevoked); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT invalidated_at IS NOT NULL AND invalidation_reason = 'operator_revoked'
		FROM extension_trust_challenges WHERE extension_id = 'fixture.plugin'
	`).Scan(&challengeInvalidated); err != nil {
		t.Fatal(err)
	}
	if !grantRevoked || !challengeInvalidated {
		t.Fatalf("grantRevoked=%t challengeInvalidated=%t", grantRevoked, challengeInvalidated)
	}

	// A replay keeps the exact recovery publication stable because the member is
	// already absent, while trust history remains revoked.
	if err := store.RevokeAll(fixture.ctx, "fixture.plugin", 0, "operator_revoked_replay"); err != nil {
		t.Fatal(err)
	}
	replayed, err := fixture.store.LatestPluginRuntimePublication(fixture.ctx)
	if err != nil || replayed.Revision != latest.Revision || replayed.MembersDigest != latest.MembersDigest {
		t.Fatalf("replayed=%+v latest=%+v err=%v", replayed, latest, err)
	}
	if latest.MembersDigest == strings.Repeat("0", 64) {
		t.Fatal("unexpected placeholder digest")
	}
}

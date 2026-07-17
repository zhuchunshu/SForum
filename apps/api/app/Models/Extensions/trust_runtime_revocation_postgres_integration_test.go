package extensions

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPostgresExecutableTrustRevokePublishesRuntimeRemovalAtomically(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "trust_store_revoke")
	seedExecutableTrustRevocationTables(t, fixture, true)
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

	// Simulate an old producer/history drift that reintroduced the member after
	// the grant was already revoked. RowsAffected is now zero, but replay must
	// still repair the authoritative full-set.
	drifted, err := fixture.store.publishPluginRuntimePublication(
		fixture.ctx, PluginRuntimePublicationStartupReconcile, 0,
		[]PluginRuntimeMember{fixture.firstMember(), fixture.secondMember()},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeAll(fixture.ctx, "fixture.plugin", 0, "repair_drift"); err != nil {
		t.Fatal(err)
	}
	repaired, err := fixture.store.LatestPluginRuntimePublication(fixture.ctx)
	if err != nil || repaired.Revision <= drifted.Revision {
		t.Fatalf("repaired=%+v drifted=%+v err=%v", repaired, drifted, err)
	}
	assertPluginRuntimePublicationMembers(t, repaired, fixture.secondMember())
}

func TestPostgresExecutableTrustRevokeRecoversCommittedResultAfterCommitError(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "trust_revoke_commit_readback")
	seedExecutableTrustRevocationTables(t, fixture, true)
	seed, err := fixture.store.publishPluginRuntimePublication(
		fixture.ctx, PluginRuntimePublicationStartupReconcile, 0,
		[]PluginRuntimeMember{fixture.firstMember(), fixture.secondMember()},
	)
	if err != nil {
		t.Fatal(err)
	}

	store := NewPostgresExecutableTrustStore(fixture.pool)
	commitErr := errors.New("injected lost COMMIT response")
	revokeCtx, cancelRevoke := context.WithCancel(fixture.ctx)
	store.commitRevokeAll = func(ctx context.Context, tx pgx.Tx) error {
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		// The recovery read must not inherit caller cancellation after PostgreSQL
		// has made the transaction durable.
		cancelRevoke()
		return commitErr
	}
	if err := store.RevokeAll(revokeCtx, "fixture.plugin", 0, "commit_readback"); err != nil {
		t.Fatalf("committed revocation was not recovered: %v", err)
	}
	latest, err := fixture.store.LatestPluginRuntimePublication(fixture.ctx)
	if err != nil || latest.Revision <= seed.Revision {
		t.Fatalf("latest=%+v seed=%+v err=%v", latest, seed, err)
	}
	assertPluginRuntimePublicationMembers(t, latest, fixture.secondMember())
	assertExecutableTrustRevocationState(t, fixture, true, true)
}

func TestPostgresExecutableTrustRevokeReturnsTypedUnknownWhenReadbackCannotProveCommit(t *testing.T) {
	for _, test := range []struct {
		name      string
		closePool bool
	}{
		{name: "durable state does not match"},
		{name: "readback unavailable", closePool: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPluginRuntimePublicationPGFixture(t, "trust_revoke_unknown_"+strings.ReplaceAll(test.name, " ", "_"))
			seedExecutableTrustRevocationTables(t, fixture, true)
			if _, err := fixture.store.publishPluginRuntimePublication(
				fixture.ctx, PluginRuntimePublicationStartupReconcile, 0,
				[]PluginRuntimeMember{fixture.firstMember(), fixture.secondMember()},
			); err != nil {
				t.Fatal(err)
			}
			store := NewPostgresExecutableTrustStore(fixture.pool)
			commitErr := errors.New("injected ambiguous COMMIT response")
			store.commitRevokeAll = func(ctx context.Context, tx pgx.Tx) error {
				if err := tx.Rollback(ctx); err != nil {
					return err
				}
				if test.closePool {
					fixture.pool.Close()
				}
				return commitErr
			}
			err := store.RevokeAll(fixture.ctx, "fixture.plugin", 0, "commit_unknown")
			var typed *TrustRevocationCommitUnknownError
			if !errors.As(err, &typed) || !errors.Is(err, ErrTrustRevocationCommitUnknown) ||
				!errors.Is(err, commitErr) {
				t.Fatalf("unknown commit error=%v", err)
			}
			if !test.closePool {
				assertExecutableTrustRevocationState(t, fixture, false, false)
			}
		})
	}
}

func TestPostgresExecutableTrustRevokeTreats08007AsUnknownAgainstDurableState(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "trust_revoke_08007")
	seedExecutableTrustRevocationTables(t, fixture, true)
	seed, err := fixture.store.publishPluginRuntimePublication(
		fixture.ctx, PluginRuntimePublicationStartupReconcile, 0,
		[]PluginRuntimeMember{fixture.firstMember(), fixture.secondMember()},
	)
	if err != nil {
		t.Fatal(err)
	}

	store := NewPostgresExecutableTrustStore(fixture.pool)
	commitErr := &pgconn.PgError{Code: "08007", Message: "transaction resolution unknown"}
	store.commitRevokeAll = func(ctx context.Context, tx pgx.Tx) error {
		// Use a real PostgreSQL transaction whose durable state proves that this
		// injected 08007 did not apply. Classification must still remain unknown.
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return commitErr
	}
	err = store.RevokeAll(fixture.ctx, "fixture.plugin", 0, "commit_08007")
	var typed *TrustRevocationCommitUnknownError
	if !errors.As(err, &typed) || !errors.Is(err, ErrTrustRevocationCommitUnknown) ||
		!errors.Is(err, commitErr) {
		t.Fatalf("08007 commit error=%v", err)
	}
	assertExecutableTrustRevocationState(t, fixture, false, false)
	latest, latestErr := fixture.store.LatestPluginRuntimePublication(fixture.ctx)
	if latestErr != nil || latest.Revision != seed.Revision || latest.MembersDigest != seed.MembersDigest {
		t.Fatalf("08007 changed durable runtime set: seed=%+v latest=%+v err=%v", seed, latest, latestErr)
	}
	assertPluginRuntimePublicationMembers(t, latest, fixture.firstMember(), fixture.secondMember())
}

func TestPostgresExecutableTrustRevokeWithoutGrantHistoryPreservesRuntimeMember(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "trust_store_no_history")
	seedExecutableTrustRevocationTables(t, fixture, false)
	seed, err := fixture.store.publishPluginRuntimePublication(
		fixture.ctx, PluginRuntimePublicationStartupReconcile, 0,
		[]PluginRuntimeMember{fixture.firstMember(), fixture.secondMember()},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := NewPostgresExecutableTrustStore(fixture.pool).RevokeAll(
		fixture.ctx, "fixture.plugin", 0, "no_history",
	); err != nil {
		t.Fatal(err)
	}
	latest, err := fixture.store.LatestPluginRuntimePublication(fixture.ctx)
	if err != nil || latest.Revision != seed.Revision || latest.MembersDigest != seed.MembersDigest {
		t.Fatalf("no-history revoke changed runtime set: seed=%+v latest=%+v err=%v", seed, latest, err)
	}
}

func seedExecutableTrustRevocationTables(
	t *testing.T,
	fixture *pluginRuntimePublicationPGFixture,
	withOpenState bool,
) {
	t.Helper()
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
	`); err != nil {
		t.Fatal(err)
	}
	if withOpenState {
		if _, err := fixture.pool.Exec(fixture.ctx, `
			INSERT INTO extension_trust_grants (extension_id) VALUES ('fixture.plugin');
			INSERT INTO extension_trust_challenges (extension_id) VALUES ('fixture.plugin');
		`); err != nil {
			t.Fatal(err)
		}
	}
}

func assertExecutableTrustRevocationState(
	t *testing.T,
	fixture *pluginRuntimePublicationPGFixture,
	grantRevoked bool,
	challengeInvalidated bool,
) {
	t.Helper()
	var gotGrantRevoked, gotChallengeInvalidated bool
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT revoked_at IS NOT NULL FROM extension_trust_grants
		WHERE extension_id = 'fixture.plugin'
	`).Scan(&gotGrantRevoked); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT invalidated_at IS NOT NULL FROM extension_trust_challenges
		WHERE extension_id = 'fixture.plugin'
	`).Scan(&gotChallengeInvalidated); err != nil {
		t.Fatal(err)
	}
	if gotGrantRevoked != grantRevoked || gotChallengeInvalidated != challengeInvalidated {
		t.Fatalf(
			"grant revoked=%t want=%t challenge invalidated=%t want=%t",
			gotGrantRevoked, grantRevoked, gotChallengeInvalidated, challengeInvalidated,
		)
	}
}

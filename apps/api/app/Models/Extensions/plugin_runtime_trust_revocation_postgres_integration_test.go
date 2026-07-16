package extensions

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestPublishPluginRuntimeTrustRevocationTxRemovesExactMemberAndAlwaysAdvances(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "trust_revoke")
	revoked := transitionFixturePlugin(
		t, "fixture.plugin", 101, "1.0.0", strings.Repeat("b", 64), "backend/plugin",
	)
	unrelated := transitionFixturePlugin(
		t, "second.plugin", 102, "2.0.0", strings.Repeat("c", 64), "backend/plugin",
	)
	seed, err := fixture.store.publishPluginRuntimePublication(
		fixture.ctx, PluginRuntimePublicationStartupReconcile, 0,
		[]PluginRuntimeMember{transitionMember(revoked), transitionMember(unrelated)},
	)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	publication, err := PublishPluginRuntimeTrustRevocationTx(
		fixture.ctx, tx, revoked.ID, 17,
	)
	if err != nil {
		_ = tx.Rollback(fixture.ctx)
		t.Fatal(err)
	}
	if publication.Revision <= seed.Revision || publication.Reason != PluginRuntimePublicationRecovery || publication.ActorUserID != 17 {
		_ = tx.Rollback(fixture.ctx)
		t.Fatalf("trust revoke publication=%+v", publication)
	}
	assertPluginRuntimePublicationMembers(t, publication, transitionMember(unrelated))
	if err := tx.Commit(fixture.ctx); err != nil {
		t.Fatal(err)
	}

	// Idempotent replay still advances the immutable revision and emits NOTIFY
	// after commit, even though the revoked member is already absent.
	tx, err = fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := PublishPluginRuntimeTrustRevocationTx(
		fixture.ctx, tx, revoked.ID, 18,
	)
	if err != nil {
		_ = tx.Rollback(fixture.ctx)
		t.Fatal(err)
	}
	if replayed.Revision <= publication.Revision || replayed.MembersDigest != publication.MembersDigest || replayed.ActorUserID != 18 {
		_ = tx.Rollback(fixture.ctx)
		t.Fatalf("replayed trust revoke=%+v prior=%+v", replayed, publication)
	}
	if err := tx.Commit(fixture.ctx); err != nil {
		t.Fatal(err)
	}
	assertPluginRuntimePublicationCount(t, fixture, 3)
}

func TestPublishPluginRuntimeTrustRevocationTxRejectsMissingAuthorityAndIsolation(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "trust_revoke_invalid")
	tx, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PublishPluginRuntimeTrustRevocationTx(fixture.ctx, tx, "missing.plugin", 1); !errors.Is(err, ErrPluginRuntimePublicationNotFound) {
		_ = tx.Rollback(fixture.ctx)
		t.Fatalf("missing latest error=%v", err)
	}
	if err := tx.Rollback(fixture.ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		t.Fatal(err)
	}

	for _, isolation := range []pgx.TxIsoLevel{pgx.RepeatableRead, pgx.Serializable} {
		t.Run(string(isolation), func(t *testing.T) {
			tx, err := fixture.pool.BeginTx(fixture.ctx, pgx.TxOptions{IsoLevel: isolation})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(context.Background()) }()
			if _, err := PublishPluginRuntimeTrustRevocationTx(fixture.ctx, tx, "missing.plugin", 1); !errors.Is(err, ErrPluginRuntimePublicationConflict) {
				t.Fatalf("isolation=%q error=%v", isolation, err)
			}
		})
	}
}

package extensions

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestPublishPluginRuntimeRecoveryTxCommitsOneFilteredFullSet(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "recovery_filtered")
	preparePluginRuntimeRecoveryFixture(t, fixture)
	first, second := fixture.firstMember(), fixture.secondMember()
	genesis, err := fixture.store.publishPluginRuntimePublication(
		fixture.ctx,
		PluginRuntimePublicationStartupReconcile,
		0,
		[]PluginRuntimeMember{first, second},
	)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	mutated := false
	publication, err := PublishPluginRuntimeRecoveryTx(fixture.ctx, tx, func() ([]string, error) {
		mutated = true
		if _, err := tx.Exec(fixture.ctx, `
			UPDATE extensions SET status = 'disabled' WHERE id = ANY($1::text[])
		`, []string{first.ExtensionID, fixture.themeMember().ExtensionID}); err != nil {
			return nil, err
		}
		// Theme ids are accepted by CLI recovery but never enter subprocess sets.
		return []string{first.ExtensionID, fixture.themeMember().ExtensionID}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !mutated || publication.Revision <= genesis.Revision ||
		publication.Reason != PluginRuntimePublicationRecovery || publication.ActorUserID != 0 {
		t.Fatalf("mutated=%t publication=%+v", mutated, publication)
	}
	assertPluginRuntimePublicationMembers(t, publication, second)
	if err := tx.Commit(fixture.ctx); err != nil {
		t.Fatal(err)
	}

	latest, err := fixture.store.LatestPluginRuntimePublication(fixture.ctx)
	if err != nil || !samePluginRuntimePublication(latest, publication) {
		t.Fatalf("latest=%+v error=%v", latest, err)
	}
	assertPluginRuntimePublicationCount(t, fixture, 2)
}

func TestPublishPluginRuntimeRecoveryTxCreatesGenesisAfterMalformedTargetDisabled(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "recovery_genesis")
	preparePluginRuntimeRecoveryFixture(t, fixture)
	first, second := fixture.firstMember(), fixture.secondMember()
	// The recovery target deliberately retains its invalid persisted manifest.
	// The surviving plugin is the only artifact that genesis must decode.
	setPluginRuntimeFixtureVersion(
		t,
		fixture,
		second.ExtensionID,
		second.ExtensionVersionID,
		StatusEnabled,
		runtimeManifestBody(t, second.ExtensionID, second.ExtensionVersion, TypePlugin, "backend/second"),
	)
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE extensions SET status = 'enabled', active_version_id = $2 WHERE id = $1
	`, first.ExtensionID, first.ExtensionVersionID); err != nil {
		t.Fatal(err)
	}

	tx, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	publication, err := PublishPluginRuntimeRecoveryTx(fixture.ctx, tx, func() ([]string, error) {
		_, err := tx.Exec(fixture.ctx, `
			UPDATE extensions SET status = 'disabled' WHERE id = $1
		`, first.ExtensionID)
		return []string{first.ExtensionID}, err
	})
	if err != nil {
		t.Fatal(err)
	}
	if publication.Reason != PluginRuntimePublicationRecovery || publication.ActorUserID != 0 {
		t.Fatalf("recovery publication=%+v", publication)
	}
	assertPluginRuntimePublicationMembers(t, publication, second)
	if err := tx.Commit(fixture.ctx); err != nil {
		t.Fatal(err)
	}

	genesis, err := loadPluginRuntimePublication(
		fixture.ctx,
		fixture.pool,
		pluginRuntimePublicationSelect+` ORDER BY revision ASC LIMIT 1`,
	)
	if err != nil {
		t.Fatal(err)
	}
	if genesis.Reason != PluginRuntimePublicationStartupReconcile || genesis.ActorUserID != 0 ||
		genesis.Revision >= publication.Revision {
		t.Fatalf("genesis=%+v recovery=%+v", genesis, publication)
	}
	assertPluginRuntimePublicationMembers(t, genesis, second)
	assertPluginRuntimePublicationCount(t, fixture, 2)
}

func TestPublishPluginRuntimeRecoveryTxRejectsInvalidGenesisBeforeMutation(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "recovery_invalid_genesis")
	preparePluginRuntimeRecoveryFixture(t, fixture)
	if _, err := fixture.store.publishPluginRuntimePublication(
		fixture.ctx,
		PluginRuntimePublicationEnable,
		7,
		nil,
	); err != nil {
		t.Fatal(err)
	}

	tx, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	called := false
	_, err = PublishPluginRuntimeRecoveryTx(fixture.ctx, tx, func() ([]string, error) {
		called = true
		return nil, nil
	})
	if !errors.Is(err, ErrPluginRuntimePublicationConflict) || called {
		t.Fatalf("error=%v callback called=%t", err, called)
	}
	assertPluginRuntimePublicationCount(t, fixture, 1)
}

func TestPublishPluginRuntimeRecoveryTxRequiresReadCommitted(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "recovery_isolation")
	preparePluginRuntimeRecoveryFixture(t, fixture)
	tx, err := fixture.pool.BeginTx(
		fixture.ctx,
		pgx.TxOptions{IsoLevel: pgx.RepeatableRead},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	called := false
	_, err = PublishPluginRuntimeRecoveryTx(fixture.ctx, tx, func() ([]string, error) {
		called = true
		return nil, nil
	})
	if !errors.Is(err, ErrPluginRuntimePublicationConflict) || called {
		t.Fatalf("error=%v callback called=%t", err, called)
	}
}

func TestPublishPluginRuntimeRecoveryTxMutationFailureRollsBackState(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "recovery_rollback")
	preparePluginRuntimeRecoveryFixture(t, fixture)
	if _, err := fixture.store.publishPluginRuntimePublication(
		fixture.ctx,
		PluginRuntimePublicationStartupReconcile,
		0,
		[]PluginRuntimeMember{fixture.firstMember()},
	); err != nil {
		t.Fatal(err)
	}

	tx, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	mutationErr := errors.New("recovery audit failed")
	_, err = PublishPluginRuntimeRecoveryTx(fixture.ctx, tx, func() ([]string, error) {
		if _, err := tx.Exec(fixture.ctx, `
			UPDATE extensions SET status = 'disabled' WHERE id = $1
		`, fixture.firstMember().ExtensionID); err != nil {
			return nil, err
		}
		return nil, mutationErr
	})
	if !errors.Is(err, mutationErr) {
		_ = tx.Rollback(fixture.ctx)
		t.Fatalf("error=%v", err)
	}
	if err := tx.Rollback(fixture.ctx); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT status FROM extensions WHERE id = $1
	`, fixture.firstMember().ExtensionID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != StatusInstalled {
		t.Fatalf("rolled-back status=%q", status)
	}
	assertPluginRuntimePublicationCount(t, fixture, 1)
}

func TestPublishPluginRuntimeRecoveryTxConcurrentCommandsPreserveFullSet(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "recovery_concurrent")
	preparePluginRuntimeRecoveryFixture(t, fixture)
	members := []PluginRuntimeMember{fixture.firstMember(), fixture.secondMember()}
	if _, err := fixture.store.publishPluginRuntimePublication(
		fixture.ctx,
		PluginRuntimePublicationStartupReconcile,
		0,
		members,
	); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	errorsByMember := make(chan error, len(members))
	var wait sync.WaitGroup
	for _, member := range members {
		member := member
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			tx, err := fixture.pool.Begin(fixture.ctx)
			if err != nil {
				errorsByMember <- err
				return
			}
			defer func() { _ = tx.Rollback(context.Background()) }()
			_, err = PublishPluginRuntimeRecoveryTx(fixture.ctx, tx, func() ([]string, error) {
				_, err := tx.Exec(fixture.ctx, `
					UPDATE extensions SET status = 'disabled' WHERE id = $1
				`, member.ExtensionID)
				return []string{member.ExtensionID}, err
			})
			if err == nil {
				err = tx.Commit(fixture.ctx)
			}
			errorsByMember <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errorsByMember)
	for err := range errorsByMember {
		if err != nil {
			t.Fatal(err)
		}
	}

	latest, err := fixture.store.LatestPluginRuntimePublication(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Reason != PluginRuntimePublicationRecovery || latest.ActorUserID != 0 {
		t.Fatalf("latest=%+v", latest)
	}
	assertPluginRuntimePublicationMembers(t, latest)
	assertPluginRuntimePublicationCount(t, fixture, 3)
}

func TestPublishPluginRuntimeRecoveryTxRejectsProtectedOrUnchangedRows(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		system bool
		mutate bool
	}{
		{name: "protected builtin", source: SourceBuiltin, mutate: true},
		{name: "protected system", source: SourceUploaded, system: true, mutate: true},
		{name: "not disabled", source: SourceUploaded, mutate: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPluginRuntimePublicationPGFixture(t, "recovery_protected")
			preparePluginRuntimeRecoveryFixture(t, fixture)
			if _, err := fixture.pool.Exec(fixture.ctx, `
				UPDATE extensions SET source = $2, is_system = $3 WHERE id = $1
			`, fixture.firstMember().ExtensionID, test.source, test.system); err != nil {
				t.Fatal(err)
			}
			if _, err := fixture.store.publishPluginRuntimePublication(
				fixture.ctx, PluginRuntimePublicationStartupReconcile, 0, nil,
			); err != nil {
				t.Fatal(err)
			}

			tx, err := fixture.pool.Begin(fixture.ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(context.Background()) }()
			_, err = PublishPluginRuntimeRecoveryTx(fixture.ctx, tx, func() ([]string, error) {
				if test.mutate {
					if _, err := tx.Exec(fixture.ctx, `
						UPDATE extensions SET status = 'disabled' WHERE id = $1
					`, fixture.firstMember().ExtensionID); err != nil {
						return nil, err
					}
				}
				return []string{fixture.firstMember().ExtensionID}, nil
			})
			if !errors.Is(err, ErrPluginRuntimePublicationConflict) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestCanonicalRecoveryExtensionIDsRejectsNonCanonicalInput(t *testing.T) {
	for _, ids := range [][]string{{""}, {" plugin.id"}, {"plugin.id", "plugin.id"}} {
		if _, err := canonicalRecoveryExtensionIDs(ids); !errors.Is(err, ErrPluginRuntimePublicationConflict) {
			t.Fatalf("ids=%q error=%v", ids, err)
		}
	}
	canonical, err := canonicalRecoveryExtensionIDs([]string{"b.plugin", "a.plugin"})
	if err != nil || len(canonical) != 2 {
		t.Fatalf("canonical=%v error=%v", canonical, err)
	}
	if _, ok := canonical["a.plugin"]; !ok {
		t.Fatalf("canonical=%v", canonical)
	}
}

func preparePluginRuntimeRecoveryFixture(
	t *testing.T,
	fixture *pluginRuntimePublicationPGFixture,
) {
	t.Helper()
	if _, err := fixture.pool.Exec(fixture.ctx, `
		ALTER TABLE extensions
			ADD COLUMN source TEXT NOT NULL DEFAULT 'uploaded',
			ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT false;
	`); err != nil {
		t.Fatal(err)
	}
}

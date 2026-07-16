package extensionsruntime

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestLifecycleRuntimeActivationPublishesAfterConcurrentExactMigrationProof(t *testing.T) {
	fixture := newLifecycleMigrationRuntimeActivationFixture(t, false)

	const migrationWorkers = 8
	migrationErrors := make(chan error, migrationWorkers)
	var migrations sync.WaitGroup
	for range migrationWorkers {
		migrations.Add(1)
		go func() {
			defer migrations.Done()
			engine := NewPostgresLifecycleMigrationEngine(fixture.pool, nil)
			boundary := NewProductionLifecycleBoundaryMigrations(fixture.pool, engine)
			migrationErrors <- boundary.ReconcileLifecycleMigrations(
				fixture.ctx, fixture.migrationRequest, LifecycleBoundaryMigrationUpgrade,
			)
		}()
	}
	migrations.Wait()
	close(migrationErrors)
	for err := range migrationErrors {
		if err != nil {
			t.Fatal(err)
		}
	}

	probeTable := pgx.Identifier{fixture.identifiers.Schema, "rollout_probe"}.Sanitize()
	var migratedRows int
	if err := fixture.pool.QueryRow(fixture.ctx, "SELECT count(*) FROM "+probeTable).Scan(&migratedRows); err != nil {
		t.Fatal(err)
	}
	if migratedRows != 1 {
		t.Fatalf("migration rows=%d want=1", migratedRows)
	}

	if err := fixture.journal.PrepareLifecyclePublication(
		fixture.ctx, fixture.publicationRequest, LifecycleBoundaryActivate,
	); err != nil {
		t.Fatal(err)
	}
	drifted := cloneLifecycleBoundaryRequest(fixture.publicationRequest)
	drifted.TargetExtension.Manifest.Migrations[0].Digest = strings.Repeat("f", 64)
	if err := fixture.journal.CommitLifecyclePublication(
		fixture.ctx, drifted, LifecycleBoundaryActivate,
	); !errors.Is(err, ErrLifecycleMigrationsConflict) {
		t.Fatalf("drifted migration plan publication error=%v", err)
	}
	fixture.assertPublicationUncommitted(t)

	plan, err := lifecycleMigrationPlanFor(
		fixture.publicationRequest, LifecycleBoundaryMigrationUpgrade, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	proofLock, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer proofLock.Rollback(fixture.ctx)
	if _, err := loadLifecycleMigrationProof(
		fixture.ctx, proofLock, plan.OperationID, plan.Mode, true,
	); err != nil {
		t.Fatal(err)
	}

	const publicationWorkers = 8
	publicationErrors := make(chan error, publicationWorkers)
	publicationCtx, cancelPublications := context.WithTimeout(fixture.ctx, 10*time.Second)
	defer cancelPublications()
	var publications sync.WaitGroup
	for range publicationWorkers {
		publications.Add(1)
		go func() {
			defer publications.Done()
			journal := NewPostgresLifecycleBoundaryPublicationJournal(fixture.pool)
			publicationErrors <- journal.CommitLifecyclePublication(
				publicationCtx, fixture.publicationRequest, LifecycleBoundaryActivate,
			)
		}()
	}
	completedBeforeProofRelease := false
	var earlyPublicationError error
	select {
	case earlyPublicationError = <-publicationErrors:
		completedBeforeProofRelease = true
	case <-time.After(150 * time.Millisecond):
	}
	if err := proofLock.Commit(fixture.ctx); err != nil {
		t.Fatal(err)
	}
	publications.Wait()
	close(publicationErrors)
	if completedBeforeProofRelease {
		t.Fatalf("publication bypassed exact proof row lock: %v", earlyPublicationError)
	}
	for err := range publicationErrors {
		if err != nil {
			t.Fatal(err)
		}
	}

	fixture.assertLatestRuntimeArtifact(t, fixture.target)
	fixture.assertRuntimePublicationCount(t, 2)
	var marker bool
	var binding int64
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT commit_marker, plugin_runtime_publication_revision
		FROM extension_lifecycle_publications
		WHERE operation_id = $1 AND step_id = $2 AND publication_mode = 'activate'
	`, fixture.publicationRequest.OperationID, fixture.publicationRequest.StepID).Scan(
		&marker, &binding,
	); err != nil {
		t.Fatal(err)
	}
	if !marker || binding <= fixture.initialPublication.Revision {
		t.Fatalf("publication marker=%v binding=%d initial=%d", marker, binding, fixture.initialPublication.Revision)
	}

	restartedConfig, err := pgxpool.ParseConfig(fixture.databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	restartedConfig.ConnConfig.RuntimeParams["search_path"] = fixture.schema + ",public"
	restartedPool, err := pgxpool.NewWithConfig(fixture.ctx, restartedConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(restartedPool.Close)
	restarted := NewPostgresLifecycleBoundaryPublicationJournal(restartedPool)
	if err := restarted.CommitLifecyclePublication(
		fixture.ctx, fixture.publicationRequest, LifecycleBoundaryActivate,
	); err != nil {
		t.Fatalf("restart replay: %v", err)
	}
	fixture.assertRuntimePublicationCount(t, 2)
}

func TestLifecycleRuntimeActivationRejectsFailedMigrationProof(t *testing.T) {
	fixture := newLifecycleMigrationRuntimeActivationFixture(t, true)
	engine := NewPostgresLifecycleMigrationEngine(fixture.pool, nil)
	boundary := NewProductionLifecycleBoundaryMigrations(fixture.pool, engine)
	if err := boundary.ReconcileLifecycleMigrations(
		fixture.ctx, fixture.migrationRequest, LifecycleBoundaryMigrationUpgrade,
	); err == nil {
		t.Fatal("failing migration unexpectedly produced a ready proof")
	}
	if err := fixture.journal.PrepareLifecyclePublication(
		fixture.ctx, fixture.publicationRequest, LifecycleBoundaryActivate,
	); err != nil {
		t.Fatal(err)
	}
	err := fixture.journal.CommitLifecyclePublication(
		fixture.ctx, fixture.publicationRequest, LifecycleBoundaryActivate,
	)
	if !errors.Is(err, ErrLifecycleMigrationProofRequired) {
		t.Fatalf("failed migration publication error=%v", err)
	}
	fixture.assertLatestRuntimeArtifact(t, fixture.source)
	fixture.assertRuntimePublicationCount(t, 1)
	fixture.assertPublicationUncommitted(t)
}

func (f *lifecycleMigrationRuntimeActivationFixture) assertPublicationUncommitted(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(f.ctx, 5*time.Second)
	defer cancel()
	var marker bool
	var binding sql.NullInt64
	if err := f.pool.QueryRow(ctx, `
		SELECT commit_marker, plugin_runtime_publication_revision
		FROM extension_lifecycle_publications
		WHERE operation_id = $1 AND step_id = $2 AND publication_mode = 'activate'
	`, f.publicationRequest.OperationID, f.publicationRequest.StepID).Scan(
		&marker, &binding,
	); err != nil {
		t.Fatal(err)
	}
	if marker || binding.Valid {
		t.Fatalf("uncommitted publication marker=%v binding=%#v", marker, binding)
	}
}

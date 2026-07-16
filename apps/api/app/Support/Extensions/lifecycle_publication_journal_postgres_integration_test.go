package extensionsruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	"github.com/zhuchunshu/sforum/apps/api/database/migrations"
)

func TestPostgresLifecyclePublicationJournalConcurrentCommitBindsOneRevision(t *testing.T) {
	fixture := newLifecyclePublicationIntegrationFixture(t)
	ctx, pool, journal, request := fixture.ctx, fixture.pool, fixture.journal, fixture.request

	committed, err := journal.LifecyclePublicationCommitted(ctx, request, LifecycleBoundaryActivate)
	if err != nil || committed {
		t.Fatalf("unprepared publication = %v, %v", committed, err)
	}
	missing := request
	missing.StepID += ".missing"
	if err := journal.CommitLifecyclePublication(ctx, missing, LifecycleBoundaryActivate); !errors.Is(err, ErrLifecyclePublicationJournalNotPrepared) {
		t.Fatalf("unprepared commit error = %v", err)
	}

	seed := seedLifecycleRuntimePublication(t, fixture, request, LifecycleBoundaryActivate)
	if err := journal.PrepareLifecyclePublication(ctx, request, LifecycleBoundaryActivate); err != nil {
		t.Fatal(err)
	}
	restarted := NewPostgresLifecycleBoundaryPublicationJournal(pool)
	rebound := request
	rebound.TargetBinding.RuntimeInstanceID = "restarted-target-runtime"
	if err := restarted.PrepareLifecyclePublication(ctx, rebound, LifecycleBoundaryActivate); err != nil {
		t.Fatalf("same-attempt runtime rebind: %v", err)
	}
	if _, err := restarted.LifecyclePublicationCommitted(ctx, request, LifecycleBoundaryActivate); !errors.Is(err, ErrLifecyclePublicationJournalConflict) {
		t.Fatalf("superseded runtime inspection error = %v", err)
	}

	next := rebound
	next.Attempt++
	next.TargetBinding.RuntimeInstanceID = "later-attempt-target-runtime"
	if err := restarted.PrepareLifecyclePublication(ctx, next, LifecycleBoundaryActivate); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.LifecyclePublicationCommitted(ctx, rebound, LifecycleBoundaryActivate); !errors.Is(err, ErrLifecyclePublicationJournalConflict) {
		t.Fatalf("stale strict inspection error = %v", err)
	}
	early := next
	early.Attempt += 20
	early.TargetBinding.RuntimeInstanceID = "fresh-host-target-runtime"
	committed, err = restarted.LifecyclePublicationCommittedForOperation(ctx, early, LifecycleBoundaryActivate)
	if err != nil || committed {
		t.Fatalf("early operation inspection = %v, %v", committed, err)
	}

	const workers = 24
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- restarted.CommitLifecyclePublication(ctx, next, LifecycleBoundaryActivate)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	afterCommitRestart := NewPostgresLifecycleBoundaryPublicationJournal(pool)
	committed, err = afterCommitRestart.LifecyclePublicationCommitted(ctx, next, LifecycleBoundaryActivate)
	if err != nil || !committed {
		t.Fatalf("committed publication after restart = %v, %v", committed, err)
	}
	committed, err = afterCommitRestart.LifecyclePublicationCommittedForOperation(ctx, early, LifecycleBoundaryActivate)
	if err != nil || !committed {
		t.Fatalf("committed operation marker = %v, %v", committed, err)
	}

	var firstAttempt, lastAttempt, committedAttempt, runtimeAttempts int
	var marker bool
	var boundRevision int64
	if err := pool.QueryRow(ctx, `
		SELECT first_attempt, last_attempt, committed_attempt, commit_marker,
		       jsonb_array_length(runtime_attempts), plugin_runtime_publication_revision
		FROM extension_lifecycle_publications
		WHERE operation_id = $1 AND step_id = $2 AND publication_mode = 'activate'
	`, request.OperationID, request.StepID).Scan(
		&firstAttempt, &lastAttempt, &committedAttempt, &marker, &runtimeAttempts, &boundRevision,
	); err != nil {
		t.Fatal(err)
	}
	if firstAttempt != request.Attempt || lastAttempt != next.Attempt ||
		committedAttempt != next.Attempt || !marker || runtimeAttempts != 3 ||
		boundRevision <= seed.Revision {
		t.Fatalf(
			"attempt marker = first:%d last:%d committed:%d marker:%v runtime attempts:%d revision:%d",
			firstAttempt, lastAttempt, committedAttempt, marker, runtimeAttempts, boundRevision,
		)
	}

	var reason, digest, extensionID, version, packageDigest string
	var actorUserID, versionID int64
	var memberCount int
	if err := pool.QueryRow(ctx, `
		SELECT p.reason, p.actor_user_id, p.member_count, p.members_digest,
		       m.extension_id, m.extension_version_id, m.extension_version, m.package_digest
		FROM plugin_runtime_publications AS p
		JOIN plugin_runtime_publication_members AS m
		  ON m.publication_revision = p.revision
		WHERE p.revision = $1
	`, boundRevision).Scan(
		&reason, &actorUserID, &memberCount, &digest,
		&extensionID, &versionID, &version, &packageDigest,
	); err != nil {
		t.Fatal(err)
	}
	if reason != string(extensions.PluginRuntimePublicationEnable) ||
		actorUserID != request.ActorUserID || memberCount != 1 ||
		extensionID != request.TargetExtension.ID ||
		versionID != request.TargetExtension.ActiveVersionID ||
		version != request.TargetExtension.Version ||
		packageDigest != request.TargetExtension.PackageDigest {
		t.Fatalf("bound publication = reason:%s actor:%d count:%d member:%s/%d/%s/%s",
			reason, actorUserID, memberCount, extensionID, versionID, version, packageDigest)
	}
	if digest != seed.MembersDigest {
		t.Fatalf("same full-set digest changed: seed=%s lifecycle=%s", seed.MembersDigest, digest)
	}
	assertLifecycleRuntimePublicationCount(t, fixture, 2)
	if err := afterCommitRestart.CommitLifecyclePublication(ctx, next, LifecycleBoundaryActivate); err != nil {
		t.Fatalf("idempotent commit replay: %v", err)
	}
	assertLifecycleRuntimePublicationCount(t, fixture, 2)

	if _, err := pool.Exec(ctx, `DELETE FROM extensions WHERE id = $1`, fixture.extensionID); err == nil {
		t.Fatal("published exact version was physically deleted")
	}
	committed, err = afterCommitRestart.LifecyclePublicationCommitted(ctx, next, LifecycleBoundaryActivate)
	if err != nil || !committed {
		t.Fatalf("publication after rejected extension deletion = %v, %v", committed, err)
	}
}

func TestPostgresLifecyclePublicationJournalMarkerCASRollbackDropsRevision(t *testing.T) {
	fixture := newLifecyclePublicationIntegrationFixture(t)
	if err := fixture.journal.PrepareLifecyclePublication(
		fixture.ctx, fixture.request, LifecycleBoundaryActivate,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		CREATE FUNCTION test_skip_lifecycle_marker_update() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
		  RETURN NULL;
		END;
		$$;
		CREATE TRIGGER zz_test_skip_lifecycle_marker_update
		BEFORE UPDATE OF commit_marker ON extension_lifecycle_publications
		FOR EACH ROW
		WHEN (OLD.commit_marker IS FALSE AND NEW.commit_marker IS TRUE)
		EXECUTE FUNCTION test_skip_lifecycle_marker_update();
	`); err != nil {
		t.Fatal(err)
	}

	err := fixture.journal.CommitLifecyclePublication(
		fixture.ctx, fixture.request, LifecycleBoundaryActivate,
	)
	if !errors.Is(err, ErrLifecyclePublicationJournalConflict) {
		t.Fatalf("zero-row marker update error = %v", err)
	}
	assertLifecycleRuntimePublicationCount(t, fixture, 0)
	var marker bool
	var binding sql.NullInt64
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT commit_marker, plugin_runtime_publication_revision
		FROM extension_lifecycle_publications
		WHERE operation_id = $1 AND step_id = $2 AND publication_mode = 'activate'
	`, fixture.request.OperationID, fixture.request.StepID).Scan(&marker, &binding); err != nil {
		t.Fatal(err)
	}
	if marker || binding.Valid {
		t.Fatalf("rolled back marker = %v, binding = %#v", marker, binding)
	}
}

func TestPostgresLifecyclePublicationJournalCommitUnknownIsInspectable(t *testing.T) {
	fixture := newLifecyclePublicationIntegrationFixture(t)
	if err := fixture.journal.PrepareLifecyclePublication(
		fixture.ctx, fixture.request, LifecycleBoundaryActivate,
	); err != nil {
		t.Fatal(err)
	}
	// 模拟服务端已成功提交，但调用端在处理返回值前丢失响应。恢复只检查新连接
	// 可见的 durable marker/binding，不通过裸 UPDATE 制造不可能的新历史。
	if err := fixture.journal.CommitLifecyclePublication(
		fixture.ctx, fixture.request, LifecycleBoundaryActivate,
	); err != nil {
		t.Fatal(err)
	}

	restarted := NewPostgresLifecycleBoundaryPublicationJournal(fixture.pool)
	committed, err := restarted.LifecyclePublicationCommitted(
		fixture.ctx, fixture.request, LifecycleBoundaryActivate,
	)
	if err != nil || !committed {
		t.Fatalf("inspect unknown commit = %v, %v", committed, err)
	}
	var binding int64
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT plugin_runtime_publication_revision
		FROM extension_lifecycle_publications
		WHERE operation_id = $1 AND step_id = $2 AND publication_mode = 'activate'
	`, fixture.request.OperationID, fixture.request.StepID).Scan(&binding); err != nil {
		t.Fatal(err)
	}
	if binding <= 0 {
		t.Fatalf("unknown commit binding = %d", binding)
	}
	if err := restarted.CommitLifecyclePublication(
		fixture.ctx, fixture.request, LifecycleBoundaryActivate,
	); err != nil {
		t.Fatalf("idempotent unknown commit replay: %v", err)
	}
	assertLifecycleRuntimePublicationCount(t, fixture, 1)
}

type lifecyclePublicationIntegration struct {
	ctx         context.Context
	pool        *pgxpool.Pool
	journal     *PostgresLifecycleBoundaryPublicationJournal
	request     LifecycleBoundaryRequest
	extensionID string
}

func newLifecyclePublicationIntegration(
	t *testing.T,
) (context.Context, *pgxpool.Pool, *PostgresLifecycleBoundaryPublicationJournal, LifecycleBoundaryRequest, string) {
	t.Helper()
	fixture := newLifecyclePublicationIntegrationFixture(t)
	return fixture.ctx, fixture.pool, fixture.journal, fixture.request, fixture.extensionID
}

func newLifecyclePublicationIntegrationFixture(t *testing.T) *lifecyclePublicationIntegration {
	t.Helper()
	databaseURL := strings.TrimSpace(os.Getenv("SFORUM_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("SFORUM_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("lpj_%d", time.Now().UnixNano())
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	var pool *pgxpool.Pool
	var db *sql.DB
	t.Cleanup(func() {
		if pool != nil {
			pool.Close()
		}
		if db != nil {
			_ = db.Close()
		}
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+quotedSchema+" CASCADE")
		admin.Close()
	})

	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.RuntimeParams["search_path"] = schema + ",public"
	db = stdlib.OpenDB(*config)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE users (
			id BIGSERIAL PRIMARY KEY,
			username TEXT NOT NULL,
			username_lower TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL,
			email_lower TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL
		);
		CREATE TABLE extension_trust_grants (
			id BIGSERIAL PRIMARY KEY
		);
		CREATE TABLE extensions (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL CHECK (type IN ('plugin', 'theme')),
			name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'installed',
			active_version_id BIGINT
		);
		CREATE TABLE extension_versions (
			id BIGINT PRIMARY KEY,
			extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE CASCADE,
			version TEXT NOT NULL,
			package_digest TEXT NOT NULL,
			manifest JSONB NOT NULL
		);
		ALTER TABLE extensions
		  ADD CONSTRAINT extensions_active_version_fk
		  FOREIGN KEY (active_version_id) REFERENCES extension_versions(id) ON DELETE SET NULL;
	`); err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(
		goose.DialectPostgres, db, migrations.Files(), goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []int64{
		202607140001,
		202607140007,
		202607160027,
		202607160030,
		202607160031,
	} {
		if _, err := provider.ApplyVersion(ctx, version, true); err != nil {
			t.Fatalf("apply migration %d: %v", version, err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = nil

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	poolConfig.ConnConfig.RuntimeParams["application_name"] = schema
	pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatal(err)
	}

	request := lifecyclePublicationTestRequest(t, extensions.LifecycleMachineEnable, 5)
	const extensionID = "publication.integration"
	request.TargetExtension.ID = extensionID
	request.TargetExtension.Manifest.ID = extensionID
	request.TargetExtension.Manifest.PackageFiles[0].ID = extensionID + ".backend"
	request.TargetBinding.ExtensionID = extensionID
	source := request.TargetExtension
	request.SourceExtension = &source
	request.SourceBinding = request.TargetBinding
	if err := extensionmanifest.Validate(request.TargetExtension.Manifest); err != nil {
		t.Fatalf("integration manifest: %v", err)
	}
	manifest, err := json.Marshal(request.TargetExtension.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO extensions (id, type, name, status)
		VALUES ($1, 'plugin', 'Publication Integration', 'enabled')
	`, extensionID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO extension_versions (
			id, extension_id, version, package_digest, manifest
		) VALUES ($1, $2, $3, $4, $5::jsonb)
	`, request.TargetExtension.ActiveVersionID, extensionID,
		request.TargetExtension.Version, request.TargetExtension.PackageDigest, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE extensions SET active_version_id = $2 WHERE id = $1
	`, extensionID, request.TargetExtension.ActiveVersionID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO extension_lifecycle_operations (
			extension_id, extension_version, package_digest, artifact_digests,
			operation, plan_version, idempotency_key, request_fingerprint,
			authority_type, authority_snapshot
		) VALUES ($1, $2, $3, '{}'::jsonb, 'enable', 'publication.integration@1',
		          $4, $5, 'builtin', '{}'::jsonb)
		RETURNING id
	`, extensionID, request.TargetExtension.Version, request.TargetExtension.PackageDigest,
		"publication:"+extensionID, strings.Repeat("c", 64)).Scan(&request.OperationID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	return &lifecyclePublicationIntegration{
		ctx: ctx, pool: pool,
		journal: NewPostgresLifecycleBoundaryPublicationJournal(pool),
		request: request, extensionID: extensionID,
	}
}

func seedLifecycleRuntimePublication(
	t *testing.T,
	fixture *lifecyclePublicationIntegration,
	request LifecycleBoundaryRequest,
	mode LifecycleBoundaryPublicationMode,
) extensions.PluginRuntimePublication {
	t.Helper()
	transition, err := lifecyclePluginRuntimePublicationTransition(request, mode)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(fixture.ctx)
	publication, err := extensions.PublishPluginRuntimePublicationTransitionTx(
		fixture.ctx, tx, transition,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(fixture.ctx); err != nil {
		t.Fatal(err)
	}
	return publication
}

func assertLifecycleRuntimePublicationCount(
	t *testing.T,
	fixture *lifecyclePublicationIntegration,
	want int,
) {
	t.Helper()
	var got int
	if err := fixture.pool.QueryRow(
		fixture.ctx, `SELECT count(*) FROM plugin_runtime_publications`,
	).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("plugin runtime publication count = %d, want %d", got, want)
	}
}

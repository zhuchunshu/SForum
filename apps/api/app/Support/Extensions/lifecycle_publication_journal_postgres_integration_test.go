package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestPostgresLifecyclePublicationJournalConcurrentCrashRestartAndRetention(t *testing.T) {
	ctx, pool, journal, request, extensionID := newLifecyclePublicationIntegration(t)

	committed, err := journal.LifecyclePublicationCommitted(ctx, request, LifecycleBoundaryActivate)
	if err != nil || committed {
		t.Fatalf("unprepared publication = %v, %v", committed, err)
	}
	missing := request
	missing.StepID += ".missing"
	if err := journal.CommitLifecyclePublication(ctx, missing, LifecycleBoundaryActivate); !errors.Is(err, ErrLifecyclePublicationJournalNotPrepared) {
		t.Fatalf("unprepared commit error = %v", err)
	}

	if err := journal.PrepareLifecyclePublication(ctx, request, LifecycleBoundaryActivate); err != nil {
		t.Fatal(err)
	}
	restarted := NewPostgresLifecycleBoundaryPublicationJournal(pool)
	committed, err = restarted.LifecyclePublicationCommitted(ctx, request, LifecycleBoundaryActivate)
	if err != nil || committed {
		t.Fatalf("prepared publication after restart = %v, %v", committed, err)
	}
	rebound := request
	rebound.TargetBinding.RuntimeInstanceID = "restarted-target-runtime"
	if err := restarted.PrepareLifecyclePublication(ctx, rebound, LifecycleBoundaryActivate); err != nil {
		t.Fatalf("same-attempt runtime rebind: %v", err)
	}
	if _, err := restarted.LifecyclePublicationCommitted(ctx, request, LifecycleBoundaryActivate); !errors.Is(err, ErrLifecyclePublicationJournalConflict) {
		t.Fatalf("superseded runtime inspection error = %v", err)
	}
	committed, err = restarted.LifecyclePublicationCommitted(ctx, rebound, LifecycleBoundaryActivate)
	if err != nil || committed {
		t.Fatalf("rebound publication = %v, %v", committed, err)
	}

	conflicting := rebound
	conflicting.TargetExtension.ActiveVersionID++
	conflicting.TargetBinding.VersionID++
	const workers = 24
	var wg sync.WaitGroup
	errs := make(chan error, workers*2)
	for range workers {
		wg.Add(2)
		go func() {
			defer wg.Done()
			errs <- restarted.PrepareLifecyclePublication(ctx, rebound, LifecycleBoundaryActivate)
		}()
		go func() {
			defer wg.Done()
			err := restarted.PrepareLifecyclePublication(ctx, conflicting, LifecycleBoundaryActivate)
			if !errors.Is(err, ErrLifecyclePublicationJournalConflict) {
				errs <- fmt.Errorf("conflicting prepare = %w", err)
				return
			}
			errs <- nil
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
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

	errs = make(chan error, workers)
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
	if err := pool.QueryRow(ctx, `
		SELECT first_attempt, last_attempt, committed_attempt, commit_marker,
		       jsonb_array_length(runtime_attempts)
		FROM extension_lifecycle_publications
		WHERE operation_id = $1 AND step_id = $2 AND publication_mode = 'activate'
	`, request.OperationID, request.StepID).Scan(
		&firstAttempt, &lastAttempt, &committedAttempt, &marker, &runtimeAttempts,
	); err != nil {
		t.Fatal(err)
	}
	if firstAttempt != request.Attempt || lastAttempt != next.Attempt || committedAttempt != next.Attempt || !marker || runtimeAttempts != 3 {
		t.Fatalf(
			"attempt marker = first:%d last:%d committed:%d marker:%v runtime attempts:%d",
			firstAttempt, lastAttempt, committedAttempt, marker, runtimeAttempts,
		)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM extensions WHERE id = $1`, extensionID); err != nil {
		t.Fatalf("delete extension while retaining publication: %v", err)
	}
	committed, err = afterCommitRestart.LifecyclePublicationCommitted(ctx, next, LifecycleBoundaryActivate)
	if err != nil || !committed {
		t.Fatalf("publication after extension deletion = %v, %v", committed, err)
	}
}

func TestPostgresLifecyclePublicationJournalCommitUnknownIsInspectable(t *testing.T) {
	ctx, pool, journal, request, _ := newLifecyclePublicationIntegration(t)
	if err := journal.PrepareLifecyclePublication(ctx, request, LifecycleBoundaryActivate); err != nil {
		t.Fatal(err)
	}
	// 模拟服务端已提交 marker、客户端在收到确认前断线。恢复只读取 durable
	// marker，不能依据前一调用返回的 error 猜测提交结果。
	tag, err := pool.Exec(ctx, `
		UPDATE extension_lifecycle_publications
		SET commit_marker = TRUE,
		    committed_attempt = last_attempt,
		    committed_at = statement_timestamp(),
		    revision = revision + 1,
		    updated_at = statement_timestamp()
		WHERE operation_id = $1 AND step_id = $2 AND publication_mode = 'activate'
		  AND commit_marker = FALSE
	`, request.OperationID, request.StepID)
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("simulate unknown commit: rows=%d, err=%v", tag.RowsAffected(), err)
	}
	restarted := NewPostgresLifecycleBoundaryPublicationJournal(pool)
	committed, err := restarted.LifecyclePublicationCommitted(ctx, request, LifecycleBoundaryActivate)
	if err != nil || !committed {
		t.Fatalf("inspect unknown commit = %v, %v", committed, err)
	}
	if err := restarted.CommitLifecyclePublication(ctx, request, LifecycleBoundaryActivate); err != nil {
		t.Fatalf("idempotent commit replay: %v", err)
	}
}

func newLifecyclePublicationIntegration(
	t *testing.T,
) (context.Context, *pgxpool.Pool, *PostgresLifecycleBoundaryPublicationJournal, LifecycleBoundaryRequest, string) {
	t.Helper()
	databaseURL := strings.TrimSpace(os.Getenv("SFORUM_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("SFORUM_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	extensionID := fmt.Sprintf("publication.integration.%d", time.Now().UnixNano())
	request := lifecyclePublicationTestRequest(t, extensions.LifecycleMachineEnable, 5)
	request.TargetExtension.ID = extensionID
	request.TargetExtension.Manifest.ID = extensionID
	request.TargetBinding.ExtensionID = extensionID
	if _, err := pool.Exec(ctx, `
		INSERT INTO extensions (id, type, name, status, source, is_system, is_deletable)
		VALUES ($1, 'plugin', 'Publication Integration', 'installed', 'builtin', true, false)
	`, extensionID); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	err = pool.QueryRow(ctx, `
		INSERT INTO extension_lifecycle_operations (
			extension_id, extension_version, package_digest, artifact_digests,
			operation, plan_version, idempotency_key, request_fingerprint,
			authority_type, authority_snapshot
		) VALUES ($1, $2, $3, '{}'::jsonb, 'enable', 'publication.integration@1',
		          $4, $5, 'builtin', '{}'::jsonb)
		RETURNING id
	`, extensionID, request.TargetExtension.Version, request.TargetExtension.PackageDigest,
		"publication:"+extensionID, strings.Repeat("c", 64)).Scan(&request.OperationID)
	if err != nil {
		_, _ = pool.Exec(ctx, `DELETE FROM extensions WHERE id = $1`, extensionID)
		pool.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM extension_lifecycle_operations WHERE id = $1`, request.OperationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM extensions WHERE id = $1`, extensionID)
		pool.Close()
	})
	return ctx, pool, NewPostgresLifecycleBoundaryPublicationJournal(pool), request, extensionID
}

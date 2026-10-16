package extensions

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// insertOrphanTestOperation 直接落一条开放 lifecycle operation 行，绕过
// AcquireOperation：孤儿回收的判定依赖 lease/updated_at 时间控制，必须手工
// 排布时间戳。created_at/updated_at 统一回拨 10 分钟，保证后续把 updated_at
// 再回拨到 2 分钟前仍满足 (updated_at >= created_at) 的表约束。
func insertOrphanTestOperation(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	extensionID string,
) int64 {
	t.Helper()
	var operationID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO extension_lifecycle_operations (
			extension_id, extension_version, package_digest, artifact_digests,
			operation, plan_version, idempotency_key, request_fingerprint,
			authority_type, authority_snapshot, created_at, updated_at
		) VALUES (
			$1, '1.0.0', $2, '{}'::jsonb, 'disable', 'orphan.test@1', $3, $4,
			'builtin', '{}'::jsonb, now() - interval '10 minutes', now() - interval '10 minutes'
		)
		RETURNING id
	`, extensionID, strings.Repeat("c", 64), "disable:first", strings.Repeat("d", 64),
	).Scan(&operationID); err != nil {
		t.Fatalf("insert orphan test operation: %v", err)
	}
	return operationID
}

// setOrphanOperationAge 把操作的 updated_at 设为 fixture 指定的时间，
// 用于控制"从未 claim 过 lease"的死亡宽限判定。
func setOrphanOperationAge(t *testing.T, ctx context.Context, pool *pgxpool.Pool, operationID int64, age string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		UPDATE extension_lifecycle_operations
		SET updated_at = statement_timestamp() - $2::interval
		WHERE id = $1
	`, operationID, age); err != nil {
		t.Fatalf("age orphan test operation: %v", err)
	}
}

// insertOrphanClaimedStep 落一条 running step 及其 lease；leaseExpiry 是
// lease_expires_at 相对当前时刻的偏移（"1 minute"=未过期，"-1 minute"=过期）。
func insertOrphanClaimedStep(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	operationID int64,
	stepID string,
	leaseExpiry string,
) int64 {
	t.Helper()
	var stepIDRow int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO extension_lifecycle_steps (
			operation_id, step_id, lifecycle_action, plan_version, status,
			started_at, lease_owner_token, lease_expires_at, lease_heartbeat_at, lease_revision,
			created_at
		) VALUES (
			$1, $2, 'disable', 'orphan.test@1', 'running',
			now() - interval '9 minutes', $3, now() + $4::interval, now() - interval '9 minutes', 1,
			now() - interval '10 minutes'
		)
		RETURNING id
	`, operationID, stepID, "lifecycle-coordinator:test:"+stepID, leaseExpiry,
	).Scan(&stepIDRow); err != nil {
		t.Fatalf("insert orphan claimed step: %v", err)
	}
	return stepIDRow
}

// insertOrphanPlannedStep 落一条从未被 claim 的 planned step（lease 字段为空）。
func insertOrphanPlannedStep(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	operationID int64,
	stepID string,
) int64 {
	t.Helper()
	var stepIDRow int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO extension_lifecycle_steps (
			operation_id, step_id, lifecycle_action, plan_version, status
		) VALUES ($1, $2, 'disable', 'orphan.test@1', 'planned')
		RETURNING id
	`, operationID, stepID).Scan(&stepIDRow); err != nil {
		t.Fatalf("insert orphan planned step: %v", err)
	}
	return stepIDRow
}

func loadOrphanOperationState(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	operationID int64,
) (state string, terminalResult string, errorCode string, revision int64, completedAt *time.Time) {
	t.Helper()
	if err := pool.QueryRow(ctx, `
		SELECT state, COALESCE(terminal_result, ''), error_code, revision, completed_at
		FROM extension_lifecycle_operations WHERE id = $1
	`, operationID).Scan(&state, &terminalResult, &errorCode, &revision, &completedAt); err != nil {
		t.Fatalf("load orphan test operation: %v", err)
	}
	return state, terminalResult, errorCode, revision, completedAt
}

func loadOrphanStepStates(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	operationID int64,
) map[string]string {
	t.Helper()
	rows, err := pool.Query(ctx, `
		SELECT step_id, status, error_code, lease_owner_token, lease_expires_at
		FROM extension_lifecycle_steps WHERE operation_id = $1
	`, operationID)
	if err != nil {
		t.Fatalf("load orphan test steps: %v", err)
	}
	defer rows.Close()
	states := map[string]string{}
	for rows.Next() {
		var stepID, status, errorCode, ownerToken string
		var leaseExpires *time.Time
		if err := rows.Scan(&stepID, &status, &errorCode, &ownerToken, &leaseExpires); err != nil {
			t.Fatalf("scan orphan test steps: %v", err)
		}
		states[stepID] = strings.Join([]string{status, errorCode, ownerToken}, "|")
		if leaseExpires != nil {
			states[stepID] += "|lease-set"
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate orphan test steps: %v", err)
	}
	return states
}

func countOrphanReclaimAudits(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	operationID int64,
) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_events
		WHERE action = 'extension.lifecycle_orphan_reclaim'
		  AND metadata->>'operationId' = $1
	`, fmt.Sprintf("%d", operationID)).Scan(&count); err != nil {
		t.Fatalf("count orphan reclaim audits: %v", err)
	}
	return count
}

// 用例 1：任何一个 step 仍持有未过期 lease 的操作绝不回收（栅栏红线）。
// 操作里同时存在"已过期 lease 的 running step"与"未过期 lease 的 running
// step"，证明判定是针对整条操作、而不是针对单条 step。
func TestReclaimSkipsOperationWithLiveLease(t *testing.T) {
	ctx, pool, repository, extensionID := newLifecycleRepositoryIntegration(t)
	operationID := insertOrphanTestOperation(t, ctx, pool, extensionID)
	insertOrphanClaimedStep(t, ctx, pool, operationID, "disable.01.dead", "-1 minute")
	insertOrphanClaimedStep(t, ctx, pool, operationID, "disable.02.live", "1 minute")

	reclaimed, err := repository.ReclaimOrphanedLifecycleOperations(ctx, 0)
	if err != nil || len(reclaimed) != 0 {
		t.Fatalf("reclaimed=%v err=%v, want none", reclaimed, err)
	}
	_, _, _, revision, completedAt := loadOrphanOperationState(t, ctx, pool, operationID)
	if revision != 1 || completedAt != nil {
		t.Fatalf("operation mutated by reclaim: revision=%d completed=%v", revision, completedAt)
	}
	states := loadOrphanStepStates(t, ctx, pool, operationID)
	if states["disable.01.dead"] != "running||lifecycle-coordinator:test:disable.01.dead|lease-set" {
		t.Fatalf("expired step mutated: %q", states["disable.01.dead"])
	}
}

// 用例 2：lease 过期的孤儿操作被回收；终态 cancelled、step failed、
// lease 清空、审计落库、revision+1。
func TestReclaimClosesExpiredLeaseOperation(t *testing.T) {
	ctx, pool, repository, extensionID := newLifecycleRepositoryIntegration(t)
	operationID := insertOrphanTestOperation(t, ctx, pool, extensionID)
	insertOrphanClaimedStep(t, ctx, pool, operationID, "disable.01.host", "-1 minute")

	reclaimed, err := repository.ReclaimOrphanedLifecycleOperations(ctx, 0)
	if err != nil || len(reclaimed) != 1 || reclaimed[0] != operationID {
		t.Fatalf("reclaimed=%v err=%v, want [%d]", reclaimed, err, operationID)
	}
	state, terminal, errorCode, revision, completedAt := loadOrphanOperationState(t, ctx, pool, operationID)
	if state != LifecycleStateFailed || terminal != LifecycleTerminalCancelled ||
		errorCode != "lifecycle.orphaned" || revision != 2 || completedAt == nil {
		t.Fatalf("operation after reclaim: state=%s terminal=%s code=%s revision=%d completed=%v",
			state, terminal, errorCode, revision, completedAt)
	}
	states := loadOrphanStepStates(t, ctx, pool, operationID)
	if states["disable.01.host"] != "failed|lifecycle.orphaned|" {
		t.Fatalf("step after reclaim: %q", states["disable.01.host"])
	}
	if countOrphanReclaimAudits(t, ctx, pool, operationID) != 1 {
		t.Fatal("orphan reclaim audit row missing")
	}
}

// 用例 3：从未 claim 过 lease 的操作（只有 planned step）需要 45s 死亡宽限：
// 新鲜的不回收；超过宽限的回收。
func TestReclaimRequiresIdleGraceForUnclaimedOperations(t *testing.T) {
	ctx, pool, repository, extensionID := newLifecycleRepositoryIntegration(t)
	operationID := insertOrphanTestOperation(t, ctx, pool, extensionID)
	insertOrphanPlannedStep(t, ctx, pool, operationID, "disable.01.plan")
	setOrphanOperationAge(t, ctx, pool, operationID, "0 seconds")

	reclaimed, err := repository.ReclaimOrphanedLifecycleOperations(ctx, 0)
	if err != nil || len(reclaimed) != 0 {
		t.Fatalf("fresh unclaimed operation reclaimed=%v err=%v", reclaimed, err)
	}

	setOrphanOperationAge(t, ctx, pool, operationID, "2 minutes")
	reclaimed, err = repository.ReclaimOrphanedLifecycleOperations(ctx, 0)
	if err != nil || len(reclaimed) != 1 || reclaimed[0] != operationID {
		t.Fatalf("stale unclaimed operation reclaimed=%v err=%v", reclaimed, err)
	}
	state, terminal, _, _, completedAt := loadOrphanOperationState(t, ctx, pool, operationID)
	if state != LifecycleStateFailed || terminal != LifecycleTerminalCancelled || completedAt == nil {
		t.Fatalf("stale unclaimed operation not closed: %s/%s/%v", state, terminal, completedAt)
	}
}

// 用例 4：已 completed 的操作永远不在候选之列。
func TestReclaimSkipsClosedOperations(t *testing.T) {
	ctx, pool, repository, extensionID := newLifecycleRepositoryIntegration(t)
	operationID := insertOrphanTestOperation(t, ctx, pool, extensionID)
	if _, err := pool.Exec(ctx, `
		UPDATE extension_lifecycle_operations
		SET state = 'failed', terminal_result = 'cancelled',
		    error_code = 'lifecycle.cli_closed', error_reason = 'lifecycle.cli_closed',
		    completed_at = statement_timestamp(), updated_at = statement_timestamp()
		WHERE id = $1
	`, operationID); err != nil {
		t.Fatal(err)
	}
	setOrphanOperationAge(t, ctx, pool, operationID, "2 minutes")
	reclaimed, err := repository.ReclaimOrphanedLifecycleOperations(ctx, 0)
	if err != nil || len(reclaimed) != 0 {
		t.Fatalf("closed operation reclaimed=%v err=%v", reclaimed, err)
	}
}

// 用例 5：并发回收同一条操作只成功一次；第二次 CAS/复核落空但不出错，
// revision 只前进一格、审计只有一行。
func TestReclaimIsIdempotentUnderConcurrentCallers(t *testing.T) {
	ctx, pool, repository, extensionID := newLifecycleRepositoryIntegration(t)
	operationID := insertOrphanTestOperation(t, ctx, pool, extensionID)
	insertOrphanClaimedStep(t, ctx, pool, operationID, "disable.01.host", "-1 minute")

	const callers = 4
	results := make(chan []int64, callers)
	errorsCh := make(chan error, callers)
	start := make(chan struct{})
	var wait sync.WaitGroup
	for range callers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			reclaimed, err := repository.ReclaimOrphanedLifecycleOperations(ctx, 0)
			results <- reclaimed
			errorsCh <- err
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	close(errorsCh)
	reclaimedTotal := 0
	for err := range errorsCh {
		if err != nil {
			t.Fatalf("concurrent reclaim: %v (want CAS misses silently skipped)", err)
		}
	}
	for reclaimed := range results {
		for _, id := range reclaimed {
			if id != operationID {
				t.Fatalf("unexpected reclaim id %d", id)
			}
			reclaimedTotal++
		}
	}
	if reclaimedTotal != 1 {
		t.Fatalf("reclaimed=%d, want exactly once", reclaimedTotal)
	}
	state, terminal, _, revision, _ := loadOrphanOperationState(t, ctx, pool, operationID)
	if state != LifecycleStateFailed || terminal != LifecycleTerminalCancelled || revision != 2 {
		t.Fatalf("operation after concurrent reclaim: %s/%s/%d", state, terminal, revision)
	}
	if countOrphanReclaimAudits(t, ctx, pool, operationID) != 1 {
		t.Fatal("duplicate orphan reclaim audit rows")
	}
	// 回收后的操作必须满足 RecoverLifecycleOperation 的入口条件（终态为
	// failed/cancelled 且已 completed），这是闭环设计的衔接点。
	if _, err := pool.Exec(ctx, `SELECT 1`); err != nil {
		t.Fatal(err)
	}
	var recoverable bool
	if err := pool.QueryRow(ctx, `
		SELECT completed_at IS NOT NULL
		   AND terminal_result IN ('failed', 'cancelled')
		FROM extension_lifecycle_operations WHERE id = $1
	`, operationID).Scan(&recoverable); err != nil || !recoverable {
		t.Fatalf("reclaimed operation not recoverable: recoverable=%v err=%v", recoverable, err)
	}
}

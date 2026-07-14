package extensions

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresLifecycleRepositoryStepLeaseCAS(t *testing.T) {
	ctx, pool, repository, _, attempt := newLifecycleLeaseTestAttempt(t)

	const claimers = 8
	type claimResult struct {
		owner   string
		attempt LifecycleStepAttempt
		err     error
	}
	start := make(chan struct{})
	results := make(chan claimResult, claimers)
	var wait sync.WaitGroup
	for index := range claimers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			owner := fmt.Sprintf("lease-worker-%d", index)
			claimed, err := repository.ClaimStepLease(ctx, ClaimLifecycleStepLeaseInput{
				AttemptID: attempt.ID, ExpectedRevision: attempt.LeaseRevision,
				OwnerToken: owner, DurationMS: 60_000,
			})
			results <- claimResult{owner: owner, attempt: claimed, err: err}
		}()
	}
	close(start)
	wait.Wait()
	close(results)

	var winner claimResult
	succeeded := 0
	conflicted := 0
	for result := range results {
		switch {
		case result.err == nil:
			succeeded++
			winner = result
		case errors.Is(result.err, ErrLifecycleStepLeaseConflict):
			conflicted++
		default:
			t.Fatalf("unexpected concurrent claim error: %v", result.err)
		}
	}
	if succeeded != 1 || conflicted != claimers-1 {
		t.Fatalf("concurrent claims succeeded=%d conflicted=%d", succeeded, conflicted)
	}
	if winner.attempt.Status != LifecycleStepRunning || winner.attempt.StartedAt == nil ||
		winner.attempt.LeaseOwnerToken != winner.owner || winner.attempt.LeaseRevision != attempt.LeaseRevision+1 ||
		winner.attempt.LeaseExpiresAt == nil || winner.attempt.LeaseHeartbeatAt == nil {
		t.Fatalf("winning lease = %#v, owner=%q", winner.attempt, winner.owner)
	}

	if _, err := repository.ClaimStepLease(ctx, ClaimLifecycleStepLeaseInput{
		AttemptID: attempt.ID, ExpectedRevision: winner.attempt.LeaseRevision,
		OwnerToken: "blocked-worker", DurationMS: 60_000,
	}); !errors.Is(err, ErrLifecycleStepLeaseConflict) {
		t.Fatalf("unexpired lease claim error = %v", err)
	}

	// 使用数据库时间构造过期租约，避免依赖调度和 wall-clock sleep。
	if _, err := pool.Exec(ctx, `
		UPDATE extension_lifecycle_steps
		SET lease_heartbeat_at = transaction_timestamp() - interval '2 seconds',
		    lease_expires_at = transaction_timestamp() - interval '1 second'
		WHERE id = $1
	`, attempt.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.HeartbeatStepLease(ctx, HeartbeatLifecycleStepLeaseInput{
		AttemptID: attempt.ID, OwnerToken: winner.owner,
		Revision: winner.attempt.LeaseRevision, DurationMS: 60_000,
	}); !errors.Is(err, ErrLifecycleStepLeaseExpired) {
		t.Fatalf("expired current owner heartbeat error = %v", err)
	}
	if _, err := repository.UpdateStepProgress(ctx, UpdateLifecycleStepProgressInput{
		AttemptID: attempt.ID, LeaseOwnerToken: winner.owner,
		LeaseRevision: winner.attempt.LeaseRevision, Status: LifecycleStepRunning,
		CompletedUnits: 1, TotalUnits: 2,
	}); !errors.Is(err, ErrLifecycleStepLeaseExpired) {
		t.Fatalf("expired current owner progress error = %v", err)
	}
	if _, err := repository.CompleteStepAttempt(ctx, CompleteLifecycleStepAttemptInput{
		AttemptID: attempt.ID, LeaseOwnerToken: winner.owner,
		LeaseRevision: winner.attempt.LeaseRevision, Status: LifecycleStepSucceeded,
		CompletedUnits: 2, TotalUnits: 2,
	}); !errors.Is(err, ErrLifecycleStepLeaseExpired) {
		t.Fatalf("expired current owner completion error = %v", err)
	}
	expired, err := repository.LatestStepAttempt(ctx, attempt.OperationID, attempt.StepID)
	if err != nil {
		t.Fatal(err)
	}
	if expired.Status != LifecycleStepRunning || expired.LeaseOwnerToken != winner.owner ||
		expired.LeaseRevision != winner.attempt.LeaseRevision || expired.CompletedUnits != 0 ||
		expired.TotalUnits != 0 || expired.CompletedAt != nil {
		t.Fatalf("expired lease calls changed step state: %#v", expired)
	}
	takeover, err := repository.ClaimStepLease(ctx, ClaimLifecycleStepLeaseInput{
		AttemptID: attempt.ID, ExpectedRevision: winner.attempt.LeaseRevision,
		OwnerToken: "takeover-worker", DurationMS: 60_000,
	})
	if err != nil {
		t.Fatalf("expired lease takeover: %v", err)
	}
	if takeover.LeaseOwnerToken != "takeover-worker" || takeover.LeaseRevision != winner.attempt.LeaseRevision+1 {
		t.Fatalf("takeover lease = %#v", takeover)
	}

	if _, err := repository.HeartbeatStepLease(ctx, HeartbeatLifecycleStepLeaseInput{
		AttemptID: attempt.ID, OwnerToken: winner.owner,
		Revision: winner.attempt.LeaseRevision, DurationMS: 60_000,
	}); !errors.Is(err, ErrLifecycleStepLeaseConflict) {
		t.Fatalf("stale owner heartbeat error = %v", err)
	}
	if _, err := repository.UpdateStepProgress(ctx, UpdateLifecycleStepProgressInput{
		AttemptID: attempt.ID, LeaseOwnerToken: winner.owner,
		LeaseRevision: winner.attempt.LeaseRevision, Status: LifecycleStepRunning,
		CompletedUnits: 1, TotalUnits: 2,
	}); !errors.Is(err, ErrLifecycleStepLeaseConflict) {
		t.Fatalf("stale owner progress error = %v", err)
	}
	if _, err := repository.CompleteStepAttempt(ctx, CompleteLifecycleStepAttemptInput{
		AttemptID: attempt.ID, LeaseOwnerToken: winner.owner,
		LeaseRevision: winner.attempt.LeaseRevision, Status: LifecycleStepSucceeded,
		CompletedUnits: 2, TotalUnits: 2,
	}); !errors.Is(err, ErrLifecycleStepLeaseConflict) {
		t.Fatalf("stale owner completion error = %v", err)
	}

	heartbeat, err := repository.HeartbeatStepLease(ctx, HeartbeatLifecycleStepLeaseInput{
		AttemptID: attempt.ID, OwnerToken: takeover.LeaseOwnerToken,
		Revision: takeover.LeaseRevision, DurationMS: 60_000,
	})
	if err != nil {
		t.Fatalf("current owner heartbeat: %v", err)
	}
	if heartbeat.LeaseRevision != takeover.LeaseRevision+1 || heartbeat.LeaseOwnerToken != takeover.LeaseOwnerToken {
		t.Fatalf("heartbeat lease = %#v", heartbeat)
	}
	if _, err := repository.HeartbeatStepLease(ctx, HeartbeatLifecycleStepLeaseInput{
		AttemptID: attempt.ID, OwnerToken: takeover.LeaseOwnerToken,
		Revision: takeover.LeaseRevision, DurationMS: 60_000,
	}); !errors.Is(err, ErrLifecycleStepLeaseConflict) {
		t.Fatalf("stale revision heartbeat error = %v", err)
	}

	progress, err := repository.UpdateStepProgress(ctx, UpdateLifecycleStepProgressInput{
		AttemptID: attempt.ID, LeaseOwnerToken: heartbeat.LeaseOwnerToken,
		LeaseRevision: heartbeat.LeaseRevision, Status: LifecycleStepRunning,
		Checkpoint: "leased-progress", CompletedUnits: 1, TotalUnits: 2,
	})
	if err != nil {
		t.Fatalf("current owner progress: %v", err)
	}
	if progress.Checkpoint != "leased-progress" || progress.LeaseRevision != heartbeat.LeaseRevision {
		t.Fatalf("leased progress = %#v", progress)
	}
	if _, err := repository.UpdateStepProgress(ctx, UpdateLifecycleStepProgressInput{
		AttemptID: attempt.ID, LeaseOwnerToken: heartbeat.LeaseOwnerToken,
		LeaseRevision: takeover.LeaseRevision, Status: LifecycleStepRunning,
		CompletedUnits: 1, TotalUnits: 2,
	}); !errors.Is(err, ErrLifecycleStepLeaseConflict) {
		t.Fatalf("stale revision progress error = %v", err)
	}
	if _, err := repository.CompleteStepAttempt(ctx, CompleteLifecycleStepAttemptInput{
		AttemptID: attempt.ID, LeaseOwnerToken: heartbeat.LeaseOwnerToken,
		LeaseRevision: takeover.LeaseRevision, Status: LifecycleStepSucceeded,
		CompletedUnits: 2, TotalUnits: 2,
	}); !errors.Is(err, ErrLifecycleStepLeaseConflict) {
		t.Fatalf("stale revision completion error = %v", err)
	}

	completed, err := repository.CompleteStepAttempt(ctx, CompleteLifecycleStepAttemptInput{
		AttemptID: attempt.ID, LeaseOwnerToken: heartbeat.LeaseOwnerToken,
		LeaseRevision: heartbeat.LeaseRevision, Status: LifecycleStepSucceeded,
		Checkpoint: "leased-complete", CompletedUnits: 2, TotalUnits: 2,
	})
	if err != nil {
		t.Fatalf("current owner completion: %v", err)
	}
	if completed.Status != LifecycleStepSucceeded || completed.LeaseOwnerToken != "" ||
		completed.LeaseExpiresAt != nil || completed.LeaseHeartbeatAt != nil ||
		completed.LeaseRevision != heartbeat.LeaseRevision+1 {
		t.Fatalf("completed lease state = %#v", completed)
	}
	var status, owner string
	var revision int64
	var expiryCleared, heartbeatCleared bool
	if err := pool.QueryRow(ctx, `
		SELECT status, lease_owner_token, lease_revision,
		       lease_expires_at IS NULL, lease_heartbeat_at IS NULL
		FROM extension_lifecycle_steps WHERE id = $1
	`, attempt.ID).Scan(&status, &owner, &revision, &expiryCleared, &heartbeatCleared); err != nil {
		t.Fatal(err)
	}
	if status != LifecycleStepSucceeded || owner != "" || revision != heartbeat.LeaseRevision+1 ||
		!expiryCleared || !heartbeatCleared {
		t.Fatalf("stored atomic completion status=%q owner=%q revision=%d expiryCleared=%t heartbeatCleared=%t",
			status, owner, revision, expiryCleared, heartbeatCleared)
	}
}

func TestPostgresLifecycleRepositoryStepLeaseReleaseAndBoundaries(t *testing.T) {
	t.Run("release expired lease and retain legacy calls", func(t *testing.T) {
		ctx, pool, repository, _, attempt := newLifecycleLeaseTestAttempt(t)
		claimed, err := repository.ClaimStepLease(ctx, ClaimLifecycleStepLeaseInput{
			AttemptID: attempt.ID, ExpectedRevision: attempt.LeaseRevision,
			OwnerToken: "release-worker", DurationMS: 60_000,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			UPDATE extension_lifecycle_steps
			SET lease_heartbeat_at = transaction_timestamp() - interval '2 seconds',
			    lease_expires_at = transaction_timestamp() - interval '1 second'
			WHERE id = $1
		`, attempt.ID); err != nil {
			t.Fatal(err)
		}
		released, err := repository.ReleaseStepLease(ctx, ReleaseLifecycleStepLeaseInput{
			AttemptID: attempt.ID, OwnerToken: claimed.LeaseOwnerToken, Revision: claimed.LeaseRevision,
		})
		if err != nil {
			t.Fatalf("release expired lease: %v", err)
		}
		if released.LeaseOwnerToken != "" || released.LeaseExpiresAt != nil ||
			released.LeaseHeartbeatAt != nil || released.LeaseRevision != claimed.LeaseRevision+1 {
			t.Fatalf("released lease = %#v", released)
		}

		progress, err := repository.UpdateStepProgress(ctx, UpdateLifecycleStepProgressInput{
			AttemptID: attempt.ID, Status: LifecycleStepRunning, Checkpoint: "legacy-progress",
			CompletedUnits: 1, TotalUnits: 2,
		})
		if err != nil || progress.Checkpoint != "legacy-progress" {
			t.Fatalf("legacy progress = %#v, err=%v", progress, err)
		}
		completed, err := repository.CompleteStepAttempt(ctx, CompleteLifecycleStepAttemptInput{
			AttemptID: attempt.ID, Status: LifecycleStepSucceeded, Checkpoint: "legacy-complete",
			CompletedUnits: 2, TotalUnits: 2,
		})
		if err != nil || completed.Status != LifecycleStepSucceeded {
			t.Fatalf("legacy completion = %#v, err=%v", completed, err)
		}
		if _, err := repository.ClaimStepLease(ctx, ClaimLifecycleStepLeaseInput{
			AttemptID: attempt.ID, ExpectedRevision: completed.LeaseRevision,
			OwnerToken: "terminal-worker", DurationMS: 60_000,
		}); !errors.Is(err, ErrLifecycleStepClosed) {
			t.Fatalf("terminal step claim error = %v", err)
		}
	})

	t.Run("closed operation cannot be claimed", func(t *testing.T) {
		ctx, _, repository, operation, attempt := newLifecycleLeaseTestAttempt(t)
		if _, err := repository.CompleteOperation(ctx, CompleteLifecycleOperationInput{
			OperationID: operation.ID, ExpectedRevision: operation.Revision,
			ExpectedState: operation.State, State: LifecycleStateEnabled,
			TerminalResult: LifecycleTerminalSucceeded,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := repository.ClaimStepLease(ctx, ClaimLifecycleStepLeaseInput{
			AttemptID: attempt.ID, ExpectedRevision: attempt.LeaseRevision,
			OwnerToken: "closed-operation-worker", DurationMS: 60_000,
		}); !errors.Is(err, ErrLifecycleOperationClosed) {
			t.Fatalf("closed operation claim error = %v", err)
		}
	})
}

func newLifecycleLeaseTestAttempt(t *testing.T) (
	context.Context,
	*pgxpool.Pool,
	*PostgresLifecycleRepository,
	LifecycleOperation,
	LifecycleStepAttempt,
) {
	t.Helper()
	ctx, pool, repository, extensionID := newLifecycleRepositoryIntegration(t)
	input := lifecycleAcquireTestInput(extensionID, LifecycleOperationEnable)
	acquired, err := repository.AcquireOperation(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	step, err := repository.BeginStepAttempt(ctx, BeginLifecycleStepAttemptInput{
		OperationID: acquired.Operation.ID,
		StepID:      "enable.leased-execute", LifecycleAction: "enable",
		PlanVersion: input.PlanVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	return ctx, pool, repository, acquired.Operation, step.Attempt
}

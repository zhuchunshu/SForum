package extensions

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *PostgresLifecycleRepository) BeginStepAttempt(ctx context.Context, input BeginLifecycleStepAttemptInput) (BeginLifecycleStepAttemptResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return BeginLifecycleStepAttemptResult{}, fmt.Errorf("begin lifecycle step attempt: %w", err)
	}
	defer tx.Rollback(ctx)
	operation, err := loadLifecycleOperationByID(ctx, tx, input.OperationID, true)
	if err != nil {
		return BeginLifecycleStepAttemptResult{}, mapLifecycleOperationLoadError("lock lifecycle step operation", err)
	}
	if operation.CompletedAt != nil {
		return BeginLifecycleStepAttemptResult{}, ErrLifecycleOperationClosed
	}

	open, err := loadOpenLifecycleStepAttempt(ctx, tx, input.OperationID, input.StepID, true)
	if err == nil {
		if open.LifecycleAction != input.LifecycleAction || open.PlanVersion != input.PlanVersion ||
			!lifecycleJSONEqual(open.InputDocument, input.InputDocument) {
			return BeginLifecycleStepAttemptResult{}, ErrLifecycleStepConflict
		}
		if err := tx.Commit(ctx); err != nil {
			return BeginLifecycleStepAttemptResult{}, fmt.Errorf("commit existing lifecycle step attempt: %w", err)
		}
		return BeginLifecycleStepAttemptResult{Attempt: open}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return BeginLifecycleStepAttemptResult{}, fmt.Errorf("load open lifecycle step attempt: %w", err)
	}

	var nextAttempt int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(attempt), 0) + 1
		FROM extension_lifecycle_steps
		WHERE operation_id = $1 AND step_id = $2
	`, input.OperationID, input.StepID).Scan(&nextAttempt); err != nil {
		return BeginLifecycleStepAttemptResult{}, fmt.Errorf("allocate lifecycle step attempt: %w", err)
	}
	var attemptID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO extension_lifecycle_steps (
			operation_id, step_id, lifecycle_action, plan_version, attempt,
			input_document, checkpoint, actor_user_id, audit_event_id
		) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9)
		RETURNING id
	`, input.OperationID, input.StepID, input.LifecycleAction, input.PlanVersion,
		nextAttempt, lifecycleNullableJSON(input.InputDocument), input.Checkpoint,
		nullableLifecycleID(input.ActorUserID), nullableLifecycleID(input.AuditEventID)).Scan(&attemptID)
	if err != nil {
		return BeginLifecycleStepAttemptResult{}, mapLifecycleStepWriteError("insert lifecycle step attempt", err)
	}
	attempt, err := loadLifecycleStepAttemptByID(ctx, tx, attemptID, false)
	if err != nil {
		return BeginLifecycleStepAttemptResult{}, fmt.Errorf("load created lifecycle step attempt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return BeginLifecycleStepAttemptResult{}, fmt.Errorf("commit lifecycle step attempt: %w", err)
	}
	return BeginLifecycleStepAttemptResult{Attempt: attempt, Created: true}, nil
}

func (r *PostgresLifecycleRepository) UpdateStepProgress(ctx context.Context, input UpdateLifecycleStepProgressInput) (LifecycleStepAttempt, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("begin lifecycle step progress: %w", err)
	}
	defer tx.Rollback(ctx)
	operationID, err := lifecycleStepOperationID(ctx, tx, input.AttemptID)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleStepLoadError("load lifecycle step operation id", err)
	}
	operation, err := loadLifecycleOperationByID(ctx, tx, operationID, true)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleOperationLoadError("lock lifecycle progress operation", err)
	}
	if operation.CompletedAt != nil {
		return LifecycleStepAttempt{}, ErrLifecycleOperationClosed
	}
	current, err := loadLifecycleStepAttemptByID(ctx, tx, input.AttemptID, true)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleStepLoadError("lock lifecycle step progress", err)
	}
	if lifecycleStepTerminal(current.Status) {
		return LifecycleStepAttempt{}, ErrLifecycleStepClosed
	}
	if err := authorizeLifecycleStepLease(ctx, tx, current, input.LeaseOwnerToken, input.LeaseRevision); err != nil {
		return LifecycleStepAttempt{}, err
	}
	if input.CompletedUnits < current.CompletedUnits || input.TotalUnits < current.TotalUnits ||
		(input.TotalUnits > 0 && input.CompletedUnits > input.TotalUnits) {
		return LifecycleStepAttempt{}, ErrLifecycleProgressRegression
	}
	tag, err := tx.Exec(ctx, `
		UPDATE extension_lifecycle_steps
		SET status = $2,
		    checkpoint = CASE WHEN $3 = '' THEN checkpoint ELSE $3 END,
		    completed_units = $4, total_units = $5, progress_message = $6,
		    started_at = CASE WHEN $2 IN ('running', 'waiting')
		      THEN COALESCE(started_at, now()) ELSE started_at END,
		    updated_at = now()
		WHERE id = $1 AND status IN ('planned', 'running', 'waiting')
		  AND ((lease_owner_token = '' AND $7 = '')
		    OR (lease_owner_token = $7 AND lease_revision = $8
		      AND lease_expires_at > statement_timestamp()))
	`, input.AttemptID, input.Status, input.Checkpoint, input.CompletedUnits, input.TotalUnits,
		input.Message, input.LeaseOwnerToken, input.LeaseRevision)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleStepWriteError("update lifecycle step progress", err)
	}
	if tag.RowsAffected() != 1 {
		return LifecycleStepAttempt{}, ErrLifecycleStepLeaseConflict
	}
	updated, err := loadLifecycleStepAttemptByID(ctx, tx, input.AttemptID, false)
	if err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("load updated lifecycle step progress: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("commit lifecycle step progress: %w", err)
	}
	return updated, nil
}

func (r *PostgresLifecycleRepository) CompleteStepAttempt(ctx context.Context, input CompleteLifecycleStepAttemptInput) (LifecycleStepAttempt, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("begin lifecycle step completion: %w", err)
	}
	defer tx.Rollback(ctx)
	operationID, err := lifecycleStepOperationID(ctx, tx, input.AttemptID)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleStepLoadError("load lifecycle completion operation id", err)
	}
	operation, err := loadLifecycleOperationByID(ctx, tx, operationID, true)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleOperationLoadError("lock lifecycle completion operation", err)
	}
	if operation.CompletedAt != nil {
		return LifecycleStepAttempt{}, ErrLifecycleOperationClosed
	}
	current, err := loadLifecycleStepAttemptByID(ctx, tx, input.AttemptID, true)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleStepLoadError("lock lifecycle step completion", err)
	}
	if lifecycleStepTerminal(current.Status) {
		return LifecycleStepAttempt{}, ErrLifecycleStepClosed
	}
	if err := authorizeLifecycleStepLease(ctx, tx, current, input.LeaseOwnerToken, input.LeaseRevision); err != nil {
		return LifecycleStepAttempt{}, err
	}
	if input.CompletedUnits < current.CompletedUnits || input.TotalUnits < current.TotalUnits ||
		(input.TotalUnits > 0 && input.CompletedUnits > input.TotalUnits) {
		return LifecycleStepAttempt{}, ErrLifecycleProgressRegression
	}
	tag, err := tx.Exec(ctx, `
		UPDATE extension_lifecycle_steps
		SET status = $2,
		    checkpoint = CASE WHEN $3 = '' THEN checkpoint ELSE $3 END,
		    completed_units = $4, total_units = $5, progress_message = $6,
		    result_document = $7::jsonb,
		    error_code = $8, error_reason = $9, error_message = $10,
		    error_retryable = $11, error_retry_after = $12,
		    error_metadata = $13::jsonb,
		    skip_reason = $14, forced = $15,
		    actor_user_id = COALESCE($16, actor_user_id),
		    audit_event_id = COALESCE($17, audit_event_id),
		    lease_revision = lease_revision + CASE WHEN lease_owner_token <> '' THEN 1 ELSE 0 END,
		    lease_owner_token = '', lease_expires_at = NULL, lease_heartbeat_at = NULL,
		    started_at = CASE WHEN $2 = 'skipped' THEN started_at
		      ELSE COALESCE(started_at, now()) END,
		    completed_at = now(), updated_at = now()
		WHERE id = $1 AND status IN ('planned', 'running', 'waiting')
		  AND ((lease_owner_token = '' AND $18 = '')
		    OR (lease_owner_token = $18 AND lease_revision = $19
		      AND lease_expires_at > statement_timestamp()))
	`, input.AttemptID, input.Status, input.Checkpoint, input.CompletedUnits,
		input.TotalUnits, input.Message, lifecycleNullableJSON(input.ResultDocument),
		input.Error.Code, input.Error.Reason, input.Error.Message, input.Error.Retryable,
		input.Error.RetryAfter, lifecycleJSONObject(input.Error.Metadata), input.SkipReason,
		input.Forced, nullableLifecycleID(input.ActorUserID), nullableLifecycleID(input.AuditEventID),
		input.LeaseOwnerToken, input.LeaseRevision)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleStepWriteError("complete lifecycle step attempt", err)
	}
	if tag.RowsAffected() != 1 {
		return LifecycleStepAttempt{}, ErrLifecycleStepLeaseConflict
	}
	completed, err := loadLifecycleStepAttemptByID(ctx, tx, input.AttemptID, false)
	if err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("load completed lifecycle step attempt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("commit lifecycle step completion: %w", err)
	}
	return completed, nil
}

func (r *PostgresLifecycleRepository) LatestStepAttempt(ctx context.Context, operationID int64, stepID string) (LifecycleStepAttempt, error) {
	attempt, err := scanLifecycleStepAttempt(r.pool.QueryRow(ctx, lifecycleStepAttemptSelectSQL()+`
		WHERE operation_id = $1 AND step_id = $2
		ORDER BY attempt DESC, id DESC
		LIMIT 1
	`, operationID, stepID))
	if errors.Is(err, pgx.ErrNoRows) {
		return LifecycleStepAttempt{}, ErrLifecycleStepNotFound
	}
	if err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("load latest lifecycle step attempt: %w", err)
	}
	return attempt, nil
}

func (r *PostgresLifecycleRepository) ListStepAttempts(ctx context.Context, operationID int64) ([]LifecycleStepAttempt, error) {
	rows, err := r.pool.Query(ctx, lifecycleStepAttemptSelectSQL()+`
		WHERE operation_id = $1
		ORDER BY created_at ASC, id ASC
	`, operationID)
	if err != nil {
		return nil, fmt.Errorf("list lifecycle step attempts: %w", err)
	}
	defer rows.Close()
	items := make([]LifecycleStepAttempt, 0)
	for rows.Next() {
		item, err := scanLifecycleStepAttempt(rows)
		if err != nil {
			return nil, fmt.Errorf("scan lifecycle step attempt: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate lifecycle step attempts: %w", err)
	}
	return items, nil
}

func lifecycleStepOperationID(ctx context.Context, tx pgx.Tx, attemptID int64) (int64, error) {
	var operationID int64
	err := tx.QueryRow(ctx, `SELECT operation_id FROM extension_lifecycle_steps WHERE id = $1`, attemptID).Scan(&operationID)
	return operationID, err
}

func loadLifecycleStepAttemptByID(ctx context.Context, tx pgx.Tx, id int64, forUpdate bool) (LifecycleStepAttempt, error) {
	suffix := ""
	if forUpdate {
		suffix = " FOR UPDATE"
	}
	return scanLifecycleStepAttempt(tx.QueryRow(ctx, lifecycleStepAttemptSelectSQL()+` WHERE id = $1`+suffix, id))
}

func loadOpenLifecycleStepAttempt(ctx context.Context, tx pgx.Tx, operationID int64, stepID string, forUpdate bool) (LifecycleStepAttempt, error) {
	suffix := ""
	if forUpdate {
		suffix = " FOR UPDATE"
	}
	return scanLifecycleStepAttempt(tx.QueryRow(ctx, lifecycleStepAttemptSelectSQL()+`
		WHERE operation_id = $1 AND step_id = $2
		  AND status IN ('planned', 'running', 'waiting')
		ORDER BY attempt DESC, id DESC LIMIT 1`+suffix, operationID, stepID))
}

func lifecycleStepTerminal(status string) bool {
	switch status {
	case LifecycleStepSucceeded, LifecycleStepFailed, LifecycleStepCancelled, LifecycleStepSkipped:
		return true
	default:
		return false
	}
}

func mapLifecycleStepLoadError(action string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrLifecycleStepNotFound
	}
	return fmt.Errorf("%s: %w", action, err)
}

func mapLifecycleStepWriteError(action string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%w: %s", ErrLifecycleStepConflict, pgErr.ConstraintName)
		case "23503", "23514":
			return fmt.Errorf("%w: %s", ErrLifecycleInvalidInput, pgErr.ConstraintName)
		}
	}
	return fmt.Errorf("%s: %w", action, err)
}

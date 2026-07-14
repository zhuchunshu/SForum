package extensions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *PostgresLifecycleRepository) AcquireOperation(ctx context.Context, input AcquireLifecycleOperationInput) (AcquireLifecycleOperationResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AcquireLifecycleOperationResult{}, fmt.Errorf("begin lifecycle operation acquire: %w", err)
	}
	defer tx.Rollback(ctx)

	var lockedExtensionID string
	err = tx.QueryRow(ctx, `SELECT id FROM extensions WHERE id = $1 FOR UPDATE`, input.ExtensionID).Scan(&lockedExtensionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AcquireLifecycleOperationResult{}, ErrExtensionNotFound
	}
	if err != nil {
		return AcquireLifecycleOperationResult{}, fmt.Errorf("lock lifecycle extension: %w", err)
	}

	existing, err := loadLifecycleOperationByKey(ctx, tx, input.ExtensionID, input.IdempotencyKey, true)
	if err == nil {
		if existing.RequestFingerprint != input.RequestFingerprint {
			return AcquireLifecycleOperationResult{}, ErrLifecycleFingerprintConflict
		}
		if err := tx.Commit(ctx); err != nil {
			return AcquireLifecycleOperationResult{}, fmt.Errorf("commit lifecycle operation acquire: %w", err)
		}
		return AcquireLifecycleOperationResult{Operation: existing}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return AcquireLifecycleOperationResult{}, fmt.Errorf("load lifecycle idempotency key: %w", err)
	}
	if input.ExistingOnly {
		return AcquireLifecycleOperationResult{}, ErrLifecycleOperationNotFound
	}

	if _, err := loadOpenLifecycleOperation(ctx, tx, input.ExtensionID, true); err == nil {
		return AcquireLifecycleOperationResult{}, ErrLifecycleOperationInProgress
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return AcquireLifecycleOperationResult{}, fmt.Errorf("load open lifecycle operation: %w", err)
	}

	var operationID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO extension_lifecycle_operations (
			extension_id, extension_version, package_digest, artifact_digests,
			operation, plan_version, idempotency_key, request_fingerprint,
			authority_type, trust_grant_id, authority_snapshot,
			requested_by_user_id, audit_event_id, removal_mode, forced
		) VALUES (
			$1, $2, $3, $4::jsonb, $5, $6, $7, $8, $9, $10,
			$11::jsonb, $12, $13, $14, $15
		)
		RETURNING id
	`, input.ExtensionID, input.ExtensionVersion, input.PackageDigest,
		lifecycleJSONObject(input.ArtifactDigests), input.Operation, input.PlanVersion,
		input.IdempotencyKey, input.RequestFingerprint, input.AuthorityType,
		nullableLifecycleID(input.TrustGrantID), lifecycleJSONObject(input.AuthoritySnapshot),
		nullableLifecycleID(input.RequestedByUserID), nullableLifecycleID(input.AuditEventID),
		nullableLifecycleString(input.RemovalMode), input.Forced).Scan(&operationID)
	if err != nil {
		return AcquireLifecycleOperationResult{}, mapLifecycleOperationWriteError("insert lifecycle operation", err)
	}
	operation, err := loadLifecycleOperationByID(ctx, tx, operationID, false)
	if err != nil {
		return AcquireLifecycleOperationResult{}, fmt.Errorf("load acquired lifecycle operation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return AcquireLifecycleOperationResult{}, fmt.Errorf("commit lifecycle operation acquire: %w", err)
	}
	return AcquireLifecycleOperationResult{Operation: operation, Created: true}, nil
}

func (r *PostgresLifecycleRepository) OperationByIdempotencyKey(ctx context.Context, extensionID, idempotencyKey string) (LifecycleOperation, error) {
	operation, err := scanLifecycleOperation(r.pool.QueryRow(ctx, lifecycleOperationSelectSQL()+`
		WHERE extension_id = $1 AND idempotency_key = $2
	`, extensionID, idempotencyKey))
	if errors.Is(err, pgx.ErrNoRows) {
		return LifecycleOperation{}, ErrLifecycleOperationNotFound
	}
	if err != nil {
		return LifecycleOperation{}, fmt.Errorf("load lifecycle operation by idempotency key: %w", err)
	}
	return operation, nil
}

func (r *PostgresLifecycleRepository) Operation(ctx context.Context, extensionID string, operationID int64) (LifecycleOperation, error) {
	if r == nil || r.pool == nil || ctx == nil || extensionID == "" || extensionID != normalizeID(extensionID) || operationID <= 0 {
		return LifecycleOperation{}, ErrLifecycleInvalidInput
	}
	operation, err := scanLifecycleOperation(r.pool.QueryRow(ctx, lifecycleOperationSelectSQL()+`
		WHERE extension_id = $1 AND id = $2
	`, extensionID, operationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return LifecycleOperation{}, ErrLifecycleOperationNotFound
	}
	if err != nil {
		return LifecycleOperation{}, fmt.Errorf("load lifecycle operation: %w", err)
	}
	return operation, nil
}

func (r *PostgresLifecycleRepository) ListOperations(ctx context.Context, extensionID string, limit int) ([]LifecycleOperation, error) {
	if r == nil || r.pool == nil || ctx == nil || extensionID == "" || extensionID != normalizeID(extensionID) {
		return nil, ErrLifecycleInvalidInput
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := r.pool.Query(ctx, lifecycleOperationSelectSQL()+`
		WHERE extension_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, extensionID, limit)
	if err != nil {
		return nil, fmt.Errorf("list lifecycle operations: %w", err)
	}
	defer rows.Close()
	items := make([]LifecycleOperation, 0)
	for rows.Next() {
		item, scanErr := scanLifecycleOperation(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan lifecycle operation: %w", scanErr)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate lifecycle operations: %w", err)
	}
	return items, nil
}

func (r *PostgresLifecycleRepository) OpenOperation(ctx context.Context, extensionID string) (LifecycleOperation, error) {
	operation, err := scanLifecycleOperation(r.pool.QueryRow(ctx, lifecycleOperationSelectSQL()+`
		WHERE extension_id = $1 AND completed_at IS NULL
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, extensionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return LifecycleOperation{}, ErrLifecycleOperationNotFound
	}
	if err != nil {
		return LifecycleOperation{}, fmt.Errorf("load open lifecycle operation: %w", err)
	}
	return operation, nil
}

func (r *PostgresLifecycleRepository) ListOpenOperations(ctx context.Context, limit int) ([]LifecycleOperation, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := r.pool.Query(ctx, lifecycleOperationSelectSQL()+`
		WHERE completed_at IS NULL
		ORDER BY updated_at ASC, id ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list open lifecycle operations: %w", err)
	}
	defer rows.Close()
	items := make([]LifecycleOperation, 0)
	for rows.Next() {
		item, err := scanLifecycleOperation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan open lifecycle operation: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate open lifecycle operations: %w", err)
	}
	return items, nil
}

func (r *PostgresLifecycleRepository) TransitionOperation(ctx context.Context, input TransitionLifecycleOperationInput) (LifecycleOperation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return LifecycleOperation{}, fmt.Errorf("begin lifecycle operation transition: %w", err)
	}
	defer tx.Rollback(ctx)
	current, err := loadLifecycleOperationByID(ctx, tx, input.OperationID, true)
	if err != nil {
		return LifecycleOperation{}, mapLifecycleOperationLoadError("lock lifecycle operation transition", err)
	}
	if err := checkLifecycleOperationCAS(current, input.ExpectedRevision, input.ExpectedState); err != nil {
		return LifecycleOperation{}, err
	}
	if current.CompletedAt != nil {
		return LifecycleOperation{}, ErrLifecycleOperationClosed
	}
	tag, err := tx.Exec(ctx, `
		UPDATE extension_lifecycle_operations
		SET state = $4,
		    current_step_id = CASE WHEN $5 = '' THEN current_step_id ELSE $5 END,
		    checkpoint = COALESCE($6::jsonb, checkpoint),
		    progress = COALESCE($7::jsonb, progress),
		    error_code = $8, error_reason = $9, error_message = $10,
		    error_retryable = $11, error_retry_after = $12,
		    error_metadata = $13::jsonb,
		    revision = revision + 1,
		    started_at = COALESCE(started_at, now()),
		    updated_at = now()
		WHERE id = $1 AND revision = $2 AND state = $3 AND completed_at IS NULL
	`, input.OperationID, input.ExpectedRevision, input.ExpectedState, input.State,
		input.CurrentStepID, lifecycleNullableJSON(input.Checkpoint), lifecycleNullableJSON(input.Progress),
		input.Error.Code, input.Error.Reason, input.Error.Message, input.Error.Retryable,
		input.Error.RetryAfter, lifecycleJSONObject(input.Error.Metadata))
	if err != nil {
		return LifecycleOperation{}, mapLifecycleOperationWriteError("transition lifecycle operation", err)
	}
	if tag.RowsAffected() != 1 {
		return LifecycleOperation{}, ErrLifecycleRevisionConflict
	}
	updated, err := loadLifecycleOperationByID(ctx, tx, input.OperationID, false)
	if err != nil {
		return LifecycleOperation{}, fmt.Errorf("load transitioned lifecycle operation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return LifecycleOperation{}, fmt.Errorf("commit lifecycle operation transition: %w", err)
	}
	return updated, nil
}

func (r *PostgresLifecycleRepository) CompleteOperation(ctx context.Context, input CompleteLifecycleOperationInput) (LifecycleOperation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return LifecycleOperation{}, fmt.Errorf("begin lifecycle operation completion: %w", err)
	}
	defer tx.Rollback(ctx)
	current, err := loadLifecycleOperationByID(ctx, tx, input.OperationID, true)
	if err != nil {
		return LifecycleOperation{}, mapLifecycleOperationLoadError("lock lifecycle operation completion", err)
	}
	if err := checkLifecycleOperationCAS(current, input.ExpectedRevision, input.ExpectedState); err != nil {
		return LifecycleOperation{}, err
	}
	if current.CompletedAt != nil {
		return LifecycleOperation{}, ErrLifecycleOperationClosed
	}
	tag, err := tx.Exec(ctx, `
		UPDATE extension_lifecycle_operations
		SET state = $4, terminal_result = $5, result_document = $6::jsonb,
		    error_code = $7, error_reason = $8, error_message = $9,
		    error_retryable = $10, error_retry_after = $11,
		    error_metadata = $12::jsonb,
		    audit_event_id = COALESCE(audit_event_id, $13),
		    completed_at = now(), updated_at = now(), revision = revision + 1
		WHERE id = $1 AND revision = $2 AND state = $3 AND completed_at IS NULL
	`, input.OperationID, input.ExpectedRevision, input.ExpectedState, input.State,
		input.TerminalResult, lifecycleNullableJSON(input.ResultDocument),
		input.Error.Code, input.Error.Reason, input.Error.Message, input.Error.Retryable,
		input.Error.RetryAfter, lifecycleJSONObject(input.Error.Metadata), nullableLifecycleID(input.AuditEventID))
	if err != nil {
		return LifecycleOperation{}, mapLifecycleOperationWriteError("complete lifecycle operation", err)
	}
	if tag.RowsAffected() != 1 {
		return LifecycleOperation{}, ErrLifecycleRevisionConflict
	}
	completed, err := loadLifecycleOperationByID(ctx, tx, input.OperationID, false)
	if err != nil {
		return LifecycleOperation{}, fmt.Errorf("load completed lifecycle operation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return LifecycleOperation{}, fmt.Errorf("commit lifecycle operation completion: %w", err)
	}
	return completed, nil
}

func (r *PostgresLifecycleRepository) ResumeOperation(ctx context.Context, input ResumeLifecycleOperationInput) (LifecycleOperation, error) {
	if err := validateLifecycleRecoveryInput(input); err != nil {
		return LifecycleOperation{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return LifecycleOperation{}, fmt.Errorf("begin lifecycle operation recovery: %w", err)
	}
	defer tx.Rollback(ctx)
	current, err := loadLifecycleOperationByID(ctx, tx, input.OperationID, true)
	if err != nil {
		return LifecycleOperation{}, mapLifecycleOperationLoadError("lock lifecycle operation recovery", err)
	}
	if err := checkLifecycleOperationCAS(current, input.ExpectedRevision, input.ExpectedState); err != nil {
		return LifecycleOperation{}, err
	}
	if current.CompletedAt == nil || (current.TerminalResult != LifecycleTerminalFailed && current.TerminalResult != LifecycleTerminalCancelled) {
		return LifecycleOperation{}, ErrLifecycleNotRecoverable
	}
	if input.EscalateForced && (current.Operation != LifecycleOperationUninstall || current.Forced) {
		return LifecycleOperation{}, ErrLifecycleInvalidInput
	}
	nextAttempt := current.AttemptCount + 1
	if _, err := tx.Exec(ctx, `
		INSERT INTO extension_lifecycle_recovery_decisions (
			operation_id, operation_attempt, decision, escalate_forced,
			reason, actor_user_id, audit_event_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, input.OperationID, nextAttempt, input.Decision, input.EscalateForced,
		input.Reason, input.ActorUserID, input.AuditEventID); err != nil {
		return LifecycleOperation{}, mapLifecycleRecoveryWriteError("insert lifecycle recovery decision", err)
	}
	tag, err := tx.Exec(ctx, `
		UPDATE extension_lifecycle_operations
		SET state = 'recovery', terminal_result = NULL, completed_at = NULL,
		    attempt_count = attempt_count + 1, revision = revision + 1,
		    recovery_actor_user_id = $4, recovery_audit_event_id = $5,
		    forced = forced OR $6,
		    updated_at = now()
		WHERE id = $1 AND revision = $2 AND state = $3
		  AND completed_at IS NOT NULL AND terminal_result IN ('failed', 'cancelled')
	`, input.OperationID, input.ExpectedRevision, input.ExpectedState,
		input.ActorUserID, input.AuditEventID, input.EscalateForced)
	if err != nil {
		return LifecycleOperation{}, mapLifecycleOperationWriteError("resume lifecycle operation", err)
	}
	if tag.RowsAffected() != 1 {
		return LifecycleOperation{}, ErrLifecycleRevisionConflict
	}
	resumed, err := loadLifecycleOperationByID(ctx, tx, input.OperationID, false)
	if err != nil {
		return LifecycleOperation{}, fmt.Errorf("load resumed lifecycle operation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return LifecycleOperation{}, fmt.Errorf("commit lifecycle operation recovery: %w", err)
	}
	return resumed, nil
}

func validateLifecycleRecoveryInput(input ResumeLifecycleOperationInput) error {
	if input.OperationID <= 0 || input.ExpectedRevision <= 0 || input.ExpectedState == "" ||
		input.ActorUserID <= 0 || input.AuditEventID <= 0 ||
		len(input.Reason) > 4096 || input.Reason != strings.TrimSpace(input.Reason) {
		return ErrLifecycleInvalidInput
	}
	switch input.Decision {
	case LifecycleRecoveryRetry:
	case LifecycleRecoverySkipStep:
		if input.Reason == "" {
			return ErrLifecycleInvalidInput
		}
	default:
		return ErrLifecycleInvalidInput
	}
	if input.EscalateForced && input.Reason == "" {
		return ErrLifecycleInvalidInput
	}
	return nil
}

func mapLifecycleRecoveryWriteError(action string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%w: %s", ErrLifecycleRevisionConflict, pgErr.ConstraintName)
		case "23503", "23514":
			return fmt.Errorf("%w: %s", ErrLifecycleInvalidInput, pgErr.ConstraintName)
		}
	}
	return fmt.Errorf("%s: %w", action, err)
}

func loadLifecycleOperationByID(ctx context.Context, tx pgx.Tx, id int64, forUpdate bool) (LifecycleOperation, error) {
	suffix := ""
	if forUpdate {
		suffix = " FOR UPDATE"
	}
	return scanLifecycleOperation(tx.QueryRow(ctx, lifecycleOperationSelectSQL()+` WHERE id = $1`+suffix, id))
}

func loadLifecycleOperationByKey(ctx context.Context, tx pgx.Tx, extensionID, key string, forUpdate bool) (LifecycleOperation, error) {
	suffix := ""
	if forUpdate {
		suffix = " FOR UPDATE"
	}
	return scanLifecycleOperation(tx.QueryRow(ctx, lifecycleOperationSelectSQL()+`
		WHERE extension_id = $1 AND idempotency_key = $2`+suffix, extensionID, key))
}

func loadOpenLifecycleOperation(ctx context.Context, tx pgx.Tx, extensionID string, forUpdate bool) (LifecycleOperation, error) {
	suffix := ""
	if forUpdate {
		suffix = " FOR UPDATE"
	}
	return scanLifecycleOperation(tx.QueryRow(ctx, lifecycleOperationSelectSQL()+`
		WHERE extension_id = $1 AND completed_at IS NULL
		ORDER BY created_at DESC, id DESC LIMIT 1`+suffix, extensionID))
}

func checkLifecycleOperationCAS(operation LifecycleOperation, expectedRevision int64, expectedState string) error {
	if operation.Revision != expectedRevision || operation.State != expectedState {
		return ErrLifecycleRevisionConflict
	}
	return nil
}

func mapLifecycleOperationLoadError(action string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrLifecycleOperationNotFound
	}
	return fmt.Errorf("%s: %w", action, err)
}

func mapLifecycleOperationWriteError(action string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%w: %s", ErrLifecycleOperationInProgress, pgErr.ConstraintName)
		case "23503", "23514":
			return fmt.Errorf("%w: %s", ErrLifecycleInvalidInput, pgErr.ConstraintName)
		}
	}
	return fmt.Errorf("%s: %w", action, err)
}

func nullableLifecycleString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

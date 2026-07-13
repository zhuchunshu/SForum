package extensions

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresLifecycleRepository) ClaimStepLease(ctx context.Context, input ClaimLifecycleStepLeaseInput) (LifecycleStepAttempt, error) {
	if !validLifecycleLeaseOwner(input.OwnerToken) || input.ExpectedRevision < 0 || input.DurationMS <= 0 {
		return LifecycleStepAttempt{}, ErrLifecycleInvalidInput
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("begin lifecycle step lease claim: %w", err)
	}
	defer tx.Rollback(ctx)
	operationID, err := lifecycleStepOperationID(ctx, tx, input.AttemptID)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleStepLoadError("load lifecycle lease operation id", err)
	}
	operation, err := loadLifecycleOperationByID(ctx, tx, operationID, true)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleOperationLoadError("lock lifecycle lease operation", err)
	}
	if operation.CompletedAt != nil {
		return LifecycleStepAttempt{}, ErrLifecycleOperationClosed
	}
	current, err := loadLifecycleStepAttemptByID(ctx, tx, input.AttemptID, true)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleStepLoadError("lock lifecycle step lease claim", err)
	}
	if lifecycleStepTerminal(current.Status) {
		return LifecycleStepAttempt{}, ErrLifecycleStepClosed
	}
	tag, err := tx.Exec(ctx, `
		UPDATE extension_lifecycle_steps
		SET lease_owner_token = $3,
		    lease_heartbeat_at = statement_timestamp(),
		    lease_expires_at = statement_timestamp() + ($4 * interval '1 millisecond'),
		    lease_revision = lease_revision + 1,
		    updated_at = statement_timestamp()
		WHERE id = $1 AND lease_revision = $2
		  AND status IN ('planned', 'running', 'waiting')
		  AND (lease_owner_token = '' OR lease_expires_at <= statement_timestamp())
	`, input.AttemptID, input.ExpectedRevision, input.OwnerToken, input.DurationMS)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleStepWriteError("claim lifecycle step lease", err)
	}
	if tag.RowsAffected() != 1 {
		return LifecycleStepAttempt{}, ErrLifecycleStepLeaseConflict
	}
	claimed, err := loadLifecycleStepAttemptByID(ctx, tx, input.AttemptID, false)
	if err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("load claimed lifecycle step lease: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("commit lifecycle step lease claim: %w", err)
	}
	return claimed, nil
}

func (r *PostgresLifecycleRepository) HeartbeatStepLease(ctx context.Context, input HeartbeatLifecycleStepLeaseInput) (LifecycleStepAttempt, error) {
	if !validLifecycleLeaseOwner(input.OwnerToken) || input.Revision <= 0 || input.DurationMS <= 0 {
		return LifecycleStepAttempt{}, ErrLifecycleInvalidInput
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("begin lifecycle step lease heartbeat: %w", err)
	}
	defer tx.Rollback(ctx)
	operationID, err := lifecycleStepOperationID(ctx, tx, input.AttemptID)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleStepLoadError("load lifecycle heartbeat operation id", err)
	}
	operation, err := loadLifecycleOperationByID(ctx, tx, operationID, true)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleOperationLoadError("lock lifecycle heartbeat operation", err)
	}
	if operation.CompletedAt != nil {
		return LifecycleStepAttempt{}, ErrLifecycleOperationClosed
	}
	current, err := loadLifecycleStepAttemptByID(ctx, tx, input.AttemptID, true)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleStepLoadError("lock lifecycle step lease heartbeat", err)
	}
	if lifecycleStepTerminal(current.Status) {
		return LifecycleStepAttempt{}, ErrLifecycleStepClosed
	}
	tag, err := tx.Exec(ctx, `
		UPDATE extension_lifecycle_steps
		SET lease_heartbeat_at = statement_timestamp(),
		    lease_expires_at = statement_timestamp() + ($4 * interval '1 millisecond'),
		    lease_revision = lease_revision + 1,
		    updated_at = statement_timestamp()
		WHERE id = $1 AND lease_owner_token = $2 AND lease_revision = $3
		  AND status IN ('planned', 'running', 'waiting')
		  AND lease_expires_at > statement_timestamp()
	`, input.AttemptID, input.OwnerToken, input.Revision, input.DurationMS)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleStepWriteError("heartbeat lifecycle step lease", err)
	}
	if tag.RowsAffected() != 1 {
		if err := authorizeClaimedLifecycleStepLease(ctx, tx, current, input.OwnerToken, input.Revision); err != nil {
			return LifecycleStepAttempt{}, err
		}
		return LifecycleStepAttempt{}, ErrLifecycleStepLeaseConflict
	}
	heartbeat, err := loadLifecycleStepAttemptByID(ctx, tx, input.AttemptID, false)
	if err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("load heartbeat lifecycle step lease: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("commit lifecycle step lease heartbeat: %w", err)
	}
	return heartbeat, nil
}

func (r *PostgresLifecycleRepository) ReleaseStepLease(ctx context.Context, input ReleaseLifecycleStepLeaseInput) (LifecycleStepAttempt, error) {
	if !validLifecycleLeaseOwner(input.OwnerToken) || input.Revision <= 0 {
		return LifecycleStepAttempt{}, ErrLifecycleInvalidInput
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("begin lifecycle step lease release: %w", err)
	}
	defer tx.Rollback(ctx)
	operationID, err := lifecycleStepOperationID(ctx, tx, input.AttemptID)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleStepLoadError("load lifecycle release operation id", err)
	}
	operation, err := loadLifecycleOperationByID(ctx, tx, operationID, true)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleOperationLoadError("lock lifecycle release operation", err)
	}
	if operation.CompletedAt != nil {
		return LifecycleStepAttempt{}, ErrLifecycleOperationClosed
	}
	current, err := loadLifecycleStepAttemptByID(ctx, tx, input.AttemptID, true)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleStepLoadError("lock lifecycle step lease release", err)
	}
	if lifecycleStepTerminal(current.Status) {
		return LifecycleStepAttempt{}, ErrLifecycleStepClosed
	}
	tag, err := tx.Exec(ctx, `
		UPDATE extension_lifecycle_steps
		SET lease_owner_token = '', lease_expires_at = NULL,
		    lease_heartbeat_at = NULL, lease_revision = lease_revision + 1,
		    updated_at = statement_timestamp()
		WHERE id = $1 AND lease_owner_token = $2 AND lease_revision = $3
		  AND status IN ('planned', 'running', 'waiting')
	`, input.AttemptID, input.OwnerToken, input.Revision)
	if err != nil {
		return LifecycleStepAttempt{}, mapLifecycleStepWriteError("release lifecycle step lease", err)
	}
	if tag.RowsAffected() != 1 {
		return LifecycleStepAttempt{}, ErrLifecycleStepLeaseConflict
	}
	released, err := loadLifecycleStepAttemptByID(ctx, tx, input.AttemptID, false)
	if err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("load released lifecycle step lease: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return LifecycleStepAttempt{}, fmt.Errorf("commit lifecycle step lease release: %w", err)
	}
	return released, nil
}

func validLifecycleLeaseOwner(ownerToken string) bool {
	return len(ownerToken) >= 1 && len(ownerToken) <= 512
}

func authorizeLifecycleStepLease(
	ctx context.Context,
	tx pgx.Tx,
	current LifecycleStepAttempt,
	ownerToken string,
	revision int64,
) error {
	if ownerToken == "" {
		if revision != 0 {
			return ErrLifecycleInvalidInput
		}
		if current.LeaseOwnerToken == "" {
			return nil
		}
		return ErrLifecycleStepLeaseConflict
	}
	if !validLifecycleLeaseOwner(ownerToken) || revision <= 0 {
		return ErrLifecycleInvalidInput
	}
	return authorizeClaimedLifecycleStepLease(ctx, tx, current, ownerToken, revision)
}

func authorizeClaimedLifecycleStepLease(
	ctx context.Context,
	tx pgx.Tx,
	current LifecycleStepAttempt,
	ownerToken string,
	revision int64,
) error {
	if current.LeaseOwnerToken != ownerToken || current.LeaseRevision != revision || current.LeaseExpiresAt == nil {
		return ErrLifecycleStepLeaseConflict
	}
	var expired bool
	if err := tx.QueryRow(ctx, `
		SELECT lease_expires_at <= statement_timestamp()
		FROM extension_lifecycle_steps
		WHERE id = $1
	`, current.ID).Scan(&expired); err != nil {
		return mapLifecycleStepLoadError("check lifecycle step lease expiry", err)
	}
	if expired {
		return ErrLifecycleStepLeaseExpired
	}
	return nil
}

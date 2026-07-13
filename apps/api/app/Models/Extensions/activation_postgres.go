package extensions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresStore) LatestActivationAttempt(ctx context.Context, extensionID, packageDigest string) (ActivationAttempt, error) {
	attempt, err := scanActivationAttempt(s.pool.QueryRow(ctx, activationAttemptSelectSQL()+`
		WHERE extension_id = $1 AND package_digest = $2
		ORDER BY started_at DESC, id DESC
		LIMIT 1
	`, normalizeID(extensionID), packageDigest))
	if errors.Is(err, pgx.ErrNoRows) {
		return ActivationAttempt{}, ErrActivationAttemptNotFound
	}
	if err != nil {
		return ActivationAttempt{}, fmt.Errorf("load latest extension activation attempt: %w", err)
	}
	return attempt, nil
}

func (s *PostgresStore) BeginActivationAttempt(ctx context.Context, extension Extension, trigger, bootID string, actorUserID int64) (ActivationAttempt, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ActivationAttempt{}, fmt.Errorf("begin extension activation attempt: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE extension_activation_attempts
		SET status = 'failed', completed_at = now(), failure_reason = 'superseded_by_new_attempt'
		WHERE extension_id = $1 AND status = 'starting'
	`, extension.ID); err != nil {
		return ActivationAttempt{}, fmt.Errorf("close prior extension activation attempt: %w", err)
	}
	attempt, err := scanActivationAttempt(tx.QueryRow(ctx, `
		INSERT INTO extension_activation_attempts (
			extension_id, extension_version, package_digest, boot_id, trigger, status, actor_user_id
		) VALUES ($1, $2, $3, $4, $5, 'starting', $6)
		RETURNING id, extension_id, extension_version, package_digest, boot_id, trigger,
		          status, COALESCE(actor_user_id, 0), failure_reason, started_at, completed_at
	`, extension.ID, extension.Version, extension.PackageDigest, bootID, trigger, nullableTrustActor(actorUserID)))
	if err != nil {
		return ActivationAttempt{}, fmt.Errorf("insert extension activation attempt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ActivationAttempt{}, fmt.Errorf("commit extension activation attempt: %w", err)
	}
	return attempt, nil
}

func (s *PostgresStore) CompleteActivationAttempt(ctx context.Context, attemptID int64, status, reason string) error {
	if status != ActivationStatusHealthy && status != ActivationStatusFailed {
		return fmt.Errorf("invalid activation completion status %q", status)
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE extension_activation_attempts
		SET status = $2, failure_reason = $3, completed_at = now()
		WHERE id = $1 AND status = 'starting'
	`, attemptID, status, boundedActivationReason(reason))
	if err != nil {
		return fmt.Errorf("complete extension activation attempt: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrActivationAttemptNotFound
	}
	return nil
}

func (s *PostgresStore) RecordSkippedActivation(ctx context.Context, extension Extension, bootID, reason string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO extension_activation_attempts (
			extension_id, extension_version, package_digest, boot_id, trigger,
			status, failure_reason, completed_at
		) VALUES ($1, $2, $3, $4, 'startup', 'skipped', $5, now())
	`, extension.ID, extension.Version, extension.PackageDigest, bootID, boundedActivationReason(reason))
	if err != nil {
		return fmt.Errorf("record skipped extension activation: %w", err)
	}
	return nil
}

func activationAttemptSelectSQL() string {
	return `
		SELECT id, extension_id, extension_version, package_digest, boot_id, trigger,
		       status, COALESCE(actor_user_id, 0), failure_reason, started_at, completed_at
		FROM extension_activation_attempts
	`
}

type activationScanner interface {
	Scan(...any) error
}

func scanActivationAttempt(scanner activationScanner) (ActivationAttempt, error) {
	var attempt ActivationAttempt
	var completedAt *time.Time
	if err := scanner.Scan(
		&attempt.ID, &attempt.ExtensionID, &attempt.ExtensionVersion,
		&attempt.PackageDigest, &attempt.BootID, &attempt.Trigger, &attempt.Status,
		&attempt.ActorUserID, &attempt.FailureReason, &attempt.StartedAt, &completedAt,
	); err != nil {
		return ActivationAttempt{}, err
	}
	attempt.CompletedAt = completedAt
	return attempt, nil
}

package entitlements

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Grant(ctx context.Context, input GrantInput) (MutationResult, error) {
	return r.inTransaction(ctx, func(tx pgx.Tx) (MutationResult, error) {
		return r.GrantTx(ctx, tx, input)
	})
}

func (r *PostgresRepository) Revoke(ctx context.Context, input TransitionInput) (MutationResult, error) {
	return r.inTransaction(ctx, func(tx pgx.Tx) (MutationResult, error) {
		return r.RevokeTx(ctx, tx, input)
	})
}

func (r *PostgresRepository) Expire(ctx context.Context, input TransitionInput) (MutationResult, error) {
	return r.inTransaction(ctx, func(tx pgx.Tx) (MutationResult, error) {
		return r.ExpireTx(ctx, tx, input)
	})
}

func (r *PostgresRepository) GrantTx(ctx context.Context, tx pgx.Tx, input GrantInput) (MutationResult, error) {
	prepared, fingerprint, err := prepareGrant(input)
	if err != nil {
		return MutationResult{}, err
	}
	if err := lockIdempotencyKey(ctx, tx, prepared.IdempotencyKey); err != nil {
		return MutationResult{}, err
	}
	if replay, found, err := replayMutation(ctx, tx, prepared.IdempotencyKey, ActionGrant, fingerprint); err != nil || found {
		return replay, err
	}

	entitlement, err := insertEntitlement(ctx, tx, prepared)
	if err != nil {
		return MutationResult{}, fmt.Errorf("grant entitlement: %w", err)
	}
	auditID, err := insertAuditEvent(ctx, tx, ActionGrant, entitlement, prepared.ActorUserID)
	if err != nil {
		return MutationResult{}, err
	}
	event, err := insertEntitlementEvent(
		ctx, tx, entitlement.ID, ActionGrant, prepared.IdempotencyKey, fingerprint,
		"", StatusActive, prepared.ActorUserID, auditID,
	)
	if err != nil {
		return MutationResult{}, fmt.Errorf("record entitlement grant: %w", err)
	}
	return MutationResult{Entitlement: entitlement, Event: event}, nil
}

func (r *PostgresRepository) RevokeTx(ctx context.Context, tx pgx.Tx, input TransitionInput) (MutationResult, error) {
	return transitionTx(ctx, tx, ActionRevoke, input)
}

func (r *PostgresRepository) ExpireTx(ctx context.Context, tx pgx.Tx, input TransitionInput) (MutationResult, error) {
	return transitionTx(ctx, tx, ActionExpire, input)
}

func transitionTx(ctx context.Context, tx pgx.Tx, action string, input TransitionInput) (MutationResult, error) {
	prepared, fingerprint, err := prepareTransition(action, input)
	if err != nil {
		return MutationResult{}, err
	}
	if err := lockIdempotencyKey(ctx, tx, prepared.IdempotencyKey); err != nil {
		return MutationResult{}, err
	}
	if replay, found, err := replayMutation(ctx, tx, prepared.IdempotencyKey, action, fingerprint); err != nil || found {
		return replay, err
	}

	entitlement, err := loadEntitlement(ctx, tx, prepared.EntitlementID, true)
	if err != nil {
		return MutationResult{}, err
	}
	if entitlement.Status != StatusActive {
		return MutationResult{}, fmt.Errorf("%w: cannot %s %s entitlement", ErrStateConflict, action, entitlement.Status)
	}
	if action == ActionExpire {
		// 使用数据库时间裁决有效期，避免应用节点时钟漂移导致提前过期。
		var eligible bool
		if err := tx.QueryRow(ctx, `
			SELECT valid_until IS NOT NULL AND valid_until <= transaction_timestamp()
			FROM entitlements WHERE id = $1
		`, entitlement.ID).Scan(&eligible); err != nil {
			return MutationResult{}, fmt.Errorf("check entitlement expiry: %w", err)
		}
		if !eligible {
			return MutationResult{}, ErrNotYetExpired
		}
	}

	nextStatus := StatusRevoked
	timestampColumn := "revoked_at"
	if action == ActionExpire {
		nextStatus = StatusExpired
		timestampColumn = "expired_at"
	}
	query := `
		UPDATE entitlements
		SET status = $2, ` + timestampColumn + ` = transaction_timestamp(),
		    revision = revision + 1, updated_at = transaction_timestamp()
		WHERE id = $1 AND status = 'active'`
	if action == ActionExpire {
		query += ` AND valid_until IS NOT NULL AND valid_until <= transaction_timestamp()`
	}
	query += `
		RETURNING id, subject_type, subject_id, scope_kind, resource_type, resource_id, capability,
		          status, source_type, source_id, valid_from, valid_until, revoked_at, expired_at,
		          revision, created_at, updated_at`
	entitlement, err = scanEntitlement(tx.QueryRow(ctx, query, entitlement.ID, nextStatus))
	if errors.Is(err, pgx.ErrNoRows) {
		return MutationResult{}, ErrStateConflict
	}
	if err != nil {
		return MutationResult{}, fmt.Errorf("%s entitlement: %w", action, err)
	}
	auditID, err := insertAuditEvent(ctx, tx, action, entitlement, prepared.ActorUserID)
	if err != nil {
		return MutationResult{}, err
	}
	event, err := insertEntitlementEvent(
		ctx, tx, entitlement.ID, action, prepared.IdempotencyKey, fingerprint,
		StatusActive, nextStatus, prepared.ActorUserID, auditID,
	)
	if err != nil {
		return MutationResult{}, fmt.Errorf("record entitlement %s: %w", action, err)
	}
	return MutationResult{Entitlement: entitlement, Event: event}, nil
}

func (r *PostgresRepository) inTransaction(
	ctx context.Context,
	operation func(pgx.Tx) (MutationResult, error),
) (MutationResult, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MutationResult{}, fmt.Errorf("begin entitlement transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	result, err := operation(tx)
	if err != nil {
		return MutationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MutationResult{}, fmt.Errorf("commit entitlement transaction: %w", err)
	}
	return result, nil
}

func lockIdempotencyKey(ctx context.Context, tx pgx.Tx, key string) error {
	digest, err := requestFingerprint(key)
	if err != nil {
		return err
	}
	decoded, err := hex.DecodeString(digest[:16])
	if err != nil {
		return fmt.Errorf("lock entitlement idempotency key: %w", err)
	}
	lockID := int64(binary.BigEndian.Uint64(decoded))
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, lockID); err != nil {
		return fmt.Errorf("lock entitlement idempotency key: %w", err)
	}
	return nil
}

func replayMutation(
	ctx context.Context,
	tx pgx.Tx,
	key, action, fingerprint string,
) (MutationResult, bool, error) {
	var event Event
	var previous *string
	var actorUserID *int64
	err := tx.QueryRow(ctx, `
		SELECT id, entitlement_id, action, idempotency_key, request_fingerprint,
		       previous_status, next_status, actor_user_id, audit_event_id, created_at
		FROM entitlement_events WHERE idempotency_key = $1
	`, key).Scan(
		&event.ID, &event.EntitlementID, &event.Action, &event.IdempotencyKey,
		&event.RequestFingerprint, &previous, &event.NextStatus, &actorUserID,
		&event.AuditEventID, &event.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return MutationResult{}, false, nil
	}
	if err != nil {
		return MutationResult{}, false, fmt.Errorf("load entitlement idempotency event: %w", err)
	}
	if previous != nil {
		event.PreviousStatus = *previous
	}
	if actorUserID != nil {
		event.ActorUserID = *actorUserID
	}
	if event.Action != action || event.RequestFingerprint != fingerprint {
		return MutationResult{}, true, ErrIdempotencyConflict
	}
	entitlement, err := loadEntitlement(ctx, tx, event.EntitlementID, false)
	if err != nil {
		return MutationResult{}, true, err
	}
	return MutationResult{Entitlement: entitlement, Event: event, Replayed: true}, true, nil
}

func insertEntitlement(ctx context.Context, tx pgx.Tx, input preparedGrant) (Entitlement, error) {
	return scanEntitlement(tx.QueryRow(ctx, `
		INSERT INTO entitlements (
		  subject_type, subject_id, scope_kind, resource_type, resource_id, capability,
		  status, source_type, source_id, valid_from, valid_until
		)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''),
		        'active', $7, $8, $9, $10)
		RETURNING id, subject_type, subject_id, scope_kind, resource_type, resource_id, capability,
		          status, source_type, source_id, valid_from, valid_until, revoked_at, expired_at,
		          revision, created_at, updated_at
	`, input.Subject.Type, input.Subject.ID, input.Scope.Kind, input.Scope.ResourceType,
		input.Scope.ResourceID, input.Scope.Capability, input.Source.Type, input.Source.ID,
		input.ValidFrom, input.ValidUntil))
}

func insertAuditEvent(
	ctx context.Context,
	tx pgx.Tx,
	action string,
	entitlement Entitlement,
	actorUserID int64,
) (int64, error) {
	metadata, err := json.Marshal(map[string]any{
		"entitlementId": entitlement.ID, "subject": entitlement.Subject,
		"scope": entitlement.Scope, "source": entitlement.Source,
		"status": entitlement.Status, "revision": entitlement.Revision,
	})
	if err != nil {
		return 0, fmt.Errorf("encode entitlement audit evidence: %w", err)
	}
	var auditID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO audit_events (actor_user_id, action, metadata)
		VALUES ($1, $2, $3)
		RETURNING id
	`, nullableInt64(actorUserID), "entitlement."+action, metadata).Scan(&auditID)
	if err != nil {
		return 0, fmt.Errorf("record entitlement audit event: %w", err)
	}
	return auditID, nil
}

func insertEntitlementEvent(
	ctx context.Context,
	tx pgx.Tx,
	entitlementID int64,
	action, key, fingerprint, previousStatus, nextStatus string,
	actorUserID, auditEventID int64,
) (Event, error) {
	var event Event
	var previous *string
	var storedActorUserID *int64
	err := tx.QueryRow(ctx, `
		INSERT INTO entitlement_events (
		  entitlement_id, action, idempotency_key, request_fingerprint,
		  previous_status, next_status, actor_user_id, audit_event_id
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8)
		RETURNING id, entitlement_id, action, idempotency_key, request_fingerprint,
		          previous_status, next_status, actor_user_id, audit_event_id, created_at
	`, entitlementID, action, key, fingerprint, previousStatus, nextStatus,
		nullableInt64(actorUserID), auditEventID).Scan(
		&event.ID, &event.EntitlementID, &event.Action, &event.IdempotencyKey,
		&event.RequestFingerprint, &previous, &event.NextStatus, &storedActorUserID,
		&event.AuditEventID, &event.CreatedAt,
	)
	if previous != nil {
		event.PreviousStatus = *previous
	}
	if storedActorUserID != nil {
		event.ActorUserID = *storedActorUserID
	}
	return event, err
}

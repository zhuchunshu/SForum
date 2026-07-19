package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type externalIdentityLinkAuditInput struct {
	Action                  string
	LinkID                  int64
	UserID                  int64
	ProviderID              string
	ProviderContractVersion string
	ProviderOperation       string
	OwnerExtensionID        string
	OwnerExtensionVersionID int64
	OwnerExtensionVersion   string
	OwnerPackageDigest      string
	DeclarationRevision     int64
	DeclarationDigest       string
	PreviousRevision        int64
	NextRevision            int64
	PreviousStatus          string
	NextStatus              string
	ActorUserID             int64
}

func insertExternalIdentityLinkAudit(
	ctx context.Context,
	tx pgx.Tx,
	input externalIdentityLinkAuditInput,
) (int64, error) {
	metadata, err := json.Marshal(map[string]any{
		"linkId": input.LinkID, "providerId": input.ProviderID,
		"providerContractVersion": input.ProviderContractVersion,
		"providerOperation":       input.ProviderOperation,
		"ownerExtensionId":        input.OwnerExtensionID,
		"ownerExtensionVersionId": input.OwnerExtensionVersionID,
		"ownerExtensionVersion":   input.OwnerExtensionVersion,
		"ownerPackageDigest":      input.OwnerPackageDigest,
		"declarationRevision":     input.DeclarationRevision,
		"declarationDigest":       input.DeclarationDigest,
		"previousRevision":        input.PreviousRevision, "nextRevision": input.NextRevision,
		"previousStatus": input.PreviousStatus, "nextStatus": input.NextStatus,
	})
	if err != nil {
		return 0, fmt.Errorf("encode external identity link audit: %w", err)
	}
	var auditID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO audit_events (actor_user_id, target_user_id, action, metadata)
		VALUES ($1, $2, $3, $4::jsonb)
		RETURNING id
	`, nullableExternalIdentityActor(input.ActorUserID), input.UserID,
		"identity.external_link."+input.Action, metadata).Scan(&auditID); err != nil {
		return 0, fmt.Errorf("record external identity link audit: %w", err)
	}
	return auditID, nil
}

func insertExternalIdentityLinkEvent(
	ctx context.Context,
	tx pgx.Tx,
	link ExternalIdentityLink,
	action string,
	idempotencyKey string,
	fingerprint string,
	previousRevision int64,
	previousStatus string,
	actorUserID int64,
	auditID int64,
) (ExternalIdentityLinkEvent, error) {
	return scanExternalIdentityLinkEvent(tx.QueryRow(ctx, `
		INSERT INTO identity_external_link_events (
			link_id, provider_id, provider_contract_version, owner_extension_id,
			action, idempotency_key, request_fingerprint,
			previous_revision, next_revision, previous_status, next_status,
			actor_user_id, audit_event_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13
		)
		RETURNING id, link_id, provider_id, provider_contract_version,
		          owner_extension_id, action, idempotency_key, request_fingerprint,
		          previous_revision, next_revision, previous_status, next_status,
		          actor_user_id, audit_event_id, created_at
	`, link.ID, link.ProviderID, link.ProviderContractVersion, link.OwnerExtensionID,
		action, idempotencyKey, fingerprint, nullableExternalIdentityRevision(previousRevision),
		link.Revision, nullableExternalIdentityStatus(previousStatus), link.Status,
		nullableExternalIdentityActor(actorUserID), auditID))
}

func replayExternalIdentityMutation(
	ctx context.Context,
	tx pgx.Tx,
	key string,
	action string,
	fingerprint string,
	lock bool,
) (ExternalIdentityLinkMutation, bool, error) {
	query := `
		SELECT id, link_id, provider_id, provider_contract_version,
		       owner_extension_id, action, idempotency_key, request_fingerprint,
		       previous_revision, next_revision, previous_status, next_status,
		       actor_user_id, audit_event_id, created_at
		FROM identity_external_link_events
		WHERE idempotency_key = $1`
	if lock {
		query += ` FOR UPDATE`
	}
	event, err := scanExternalIdentityLinkEvent(tx.QueryRow(ctx, query, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return ExternalIdentityLinkMutation{}, false, nil
	}
	if err != nil {
		return ExternalIdentityLinkMutation{}, false, err
	}
	if event.Action != action || event.RequestFingerprint != fingerprint {
		return ExternalIdentityLinkMutation{}, true, ErrExternalIdentityLinkIdempotencyConflict
	}
	link, err := scanExternalIdentityLink(tx.QueryRow(ctx, externalIdentityLinkSelect+` WHERE id = $1`, event.LinkID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ExternalIdentityLinkMutation{}, true, ErrExternalIdentityLinkNotFound
	}
	if err != nil {
		return ExternalIdentityLinkMutation{}, true, err
	}
	// Event 是原请求的稳定 receipt；Link 始终返回当前状态，避免旧 link
	// receipt 在后续 unlink/erase 后伪装成仍然 active 的实时记录。
	return ExternalIdentityLinkMutation{Link: link, Event: event, Replayed: true}, true, nil
}

func (s *PostgresExternalIdentityLinkStore) readbackMutation(
	ctx context.Context,
	key string,
	action string,
	fingerprint string,
) (ExternalIdentityLinkMutation, bool, error) {
	readbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), externalIdentityLinkReadbackTimeout)
	defer cancel()
	tx, err := s.pool.BeginTx(readbackCtx, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return ExternalIdentityLinkMutation{}, false, fmt.Errorf("begin external identity link readback: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	result, found, err := replayExternalIdentityMutation(readbackCtx, tx, key, action, fingerprint, false)
	if err != nil || !found {
		return ExternalIdentityLinkMutation{}, found, err
	}
	if err := tx.Commit(readbackCtx); err != nil {
		return ExternalIdentityLinkMutation{}, false, fmt.Errorf("commit external identity link readback: %w", err)
	}
	return result, true, nil
}

func externalIdentityLinkCommitDefinitelyFailed(commitErr error) bool {
	if errors.Is(commitErr, pgx.ErrTxCommitRollback) || pgconn.SafeToRetry(commitErr) {
		return true
	}
	var postgresErr *pgconn.PgError
	if !errors.As(commitErr, &postgresErr) {
		return false
	}
	return strings.HasPrefix(postgresErr.Code, "40") && postgresErr.Code != "40003"
}

func mapExternalIdentityLinkStoreError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded),
		errors.Is(err, ErrExternalIdentityLinkInvalid),
		errors.Is(err, ErrExternalIdentityLinkNotFound),
		errors.Is(err, ErrExternalIdentityLinkStateConflict),
		errors.Is(err, ErrExternalIdentityLinkIdempotencyConflict),
		errors.Is(err, ErrExternalIdentitySubjectConflict),
		errors.Is(err, ErrExternalIdentityProviderStale),
		errors.Is(err, errExternalIdentityLinkRetry):
		return err
	}
	var postgresErr *pgconn.PgError
	if !errors.As(err, &postgresErr) {
		return ErrExternalIdentityLinkStoreUnavailable
	}
	switch postgresErr.Code {
	case "40001", "40P01":
		return errExternalIdentityLinkRetry
	case "23505":
		switch postgresErr.ConstraintName {
		case "identity_external_links_active_provider_digest_uidx":
			return ErrExternalIdentitySubjectConflict
		case "identity_external_link_events_idempotency_key_key":
			return ErrExternalIdentityLinkIdempotencyConflict
		default:
			return ErrExternalIdentityLinkStateConflict
		}
	case "23503", "23514", "22P02":
		return ErrExternalIdentityLinkInvalid
	default:
		return ErrExternalIdentityLinkStoreUnavailable
	}
}

func publicExternalIdentityLinkStoreError(err error) error {
	if errors.Is(err, errExternalIdentityLinkRetry) {
		return ErrExternalIdentityLinkStateConflict
	}
	return err
}

const externalIdentityLinkColumns = `
	id, user_id, provider_id, provider_contract_version,
	owner_extension_id, COALESCE(owner_extension_version_id, 0), owner_extension_version,
	owner_package_digest, declaration_revision, status, revision,
	linked_at, unlinked_at, erased_at, COALESCE(actor_user_id, 0), audit_event_id,
	created_at, updated_at`

const externalIdentityLinkSelect = `SELECT ` + externalIdentityLinkColumns + ` FROM identity_external_links`

type externalIdentityLinkScanner interface {
	Scan(dest ...any) error
}

func scanExternalIdentityLink(scanner externalIdentityLinkScanner) (ExternalIdentityLink, error) {
	var link ExternalIdentityLink
	err := scanner.Scan(
		&link.ID, &link.UserID, &link.ProviderID, &link.ProviderContractVersion,
		&link.OwnerExtensionID, &link.OwnerExtensionVersionID, &link.OwnerExtensionVersion,
		&link.OwnerPackageDigest, &link.DeclarationRevision, &link.Status, &link.Revision,
		&link.LinkedAt, &link.UnlinkedAt, &link.ErasedAt, &link.ActorUserID, &link.AuditEventID,
		&link.CreatedAt, &link.UpdatedAt,
	)
	return link, err
}

func scanExternalIdentityLinkEvent(scanner externalIdentityLinkScanner) (ExternalIdentityLinkEvent, error) {
	var event ExternalIdentityLinkEvent
	var previousRevision *int64
	var previousStatus *string
	var actorUserID *int64
	err := scanner.Scan(
		&event.ID, &event.LinkID, &event.ProviderID, &event.ProviderContractVersion,
		&event.OwnerExtensionID, &event.Action, &event.IdempotencyKey, &event.RequestFingerprint,
		&previousRevision, &event.NextRevision, &previousStatus, &event.NextStatus,
		&actorUserID, &event.AuditEventID, &event.CreatedAt,
	)
	if previousRevision != nil {
		event.PreviousRevision = *previousRevision
	}
	if previousStatus != nil {
		event.PreviousStatus = *previousStatus
	}
	if actorUserID != nil {
		event.ActorUserID = *actorUserID
	}
	return event, err
}

func nullableExternalIdentityActor(actorUserID int64) any {
	if actorUserID <= 0 {
		return nil
	}
	return actorUserID
}

func nullableExternalIdentityRevision(revision int64) any {
	if revision <= 0 {
		return nil
	}
	return revision
}

func nullableExternalIdentityStatus(status string) any {
	if status == "" {
		return nil
	}
	return status
}

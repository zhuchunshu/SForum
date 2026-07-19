package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type storedIdentityUserFieldValue struct {
	metadata IdentityUserFieldValue
	raw      json.RawMessage
	decoded  any
}

type identityUserFieldAuditInput struct {
	action               string
	mode                 string
	userID               int64
	fieldID              string
	ownerExtensionID     string
	fieldContractVersion string
	fieldSchemaDigest    string
	declarationRevision  int64
	declarationDigest    string
	previousRevision     int64
	nextRevision         int64
	previousValueDigest  string
	nextValueDigest      string
	actorUserID          int64
}

func insertIdentityUserFieldAudit(
	ctx context.Context,
	tx pgx.Tx,
	input identityUserFieldAuditInput,
) (int64, error) {
	metadata, err := json.Marshal(map[string]any{
		"fieldId": input.fieldID, "ownerExtensionId": input.ownerExtensionID,
		"fieldContractVersion": input.fieldContractVersion,
		"fieldSchemaDigest":    input.fieldSchemaDigest,
		"declarationRevision":  input.declarationRevision,
		"declarationDigest":    input.declarationDigest,
		"previousRevision":     input.previousRevision,
		"nextRevision":         input.nextRevision,
		"previousValueDigest":  input.previousValueDigest,
		"nextValueDigest":      input.nextValueDigest,
		"mode":                 input.mode,
	})
	if err != nil {
		return 0, fmt.Errorf("encode identity user-field audit: %w", err)
	}
	auditAction := "identity.user_field." + input.action
	if input.mode == "privacy" {
		auditAction = "identity.user_field.privacy_erase"
	}
	var auditID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO audit_events (actor_user_id, target_user_id, action, metadata)
		VALUES ($1, $2, $3, $4::jsonb)
		RETURNING id
	`, nullableIdentityUserFieldActor(input.actorUserID), input.userID, auditAction, metadata).Scan(&auditID); err != nil {
		return 0, fmt.Errorf("record identity user-field audit: %w", err)
	}
	return auditID, nil
}

func insertIdentityUserFieldEvent(
	ctx context.Context,
	tx pgx.Tx,
	value IdentityUserFieldValue,
	action string,
	idempotencyKey string,
	fingerprint string,
	previousRevision int64,
	previousValueDigest string,
	nextValueDigest string,
	actorUserID int64,
	auditID int64,
) (IdentityUserFieldValueEvent, error) {
	return scanIdentityUserFieldEvent(tx.QueryRow(ctx, `
		INSERT INTO identity_user_field_value_events (
			user_id, field_id, owner_extension_id, field_contract_version,
			field_schema_digest, declaration_revision, action,
			previous_revision, next_revision,
			previous_value_digest, next_value_digest,
			idempotency_key, request_fingerprint, actor_user_id, audit_event_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, $14, $15
		)
		RETURNING id, user_id, field_id, owner_extension_id,
		          field_contract_version, field_schema_digest, declaration_revision,
		          action, previous_revision, next_revision,
		          previous_value_digest, next_value_digest,
		          idempotency_key, request_fingerprint,
		          actor_user_id, audit_event_id, created_at
	`, value.UserID, value.FieldID, value.OwnerExtensionID, value.FieldContractVersion,
		value.FieldSchemaDigest, value.DeclarationRevision, action,
		nullableIdentityUserFieldRevision(previousRevision), value.Revision,
		nullableIdentityUserFieldDigest(previousValueDigest), nullableIdentityUserFieldDigest(nextValueDigest),
		idempotencyKey, fingerprint, nullableIdentityUserFieldActor(actorUserID), auditID))
}

func replayIdentityUserFieldMutation(
	ctx context.Context,
	tx pgx.Tx,
	key string,
	action string,
	fingerprint string,
	valueDigest func(int64, string, []byte) string,
) (IdentityUserFieldValueMutation, bool, error) {
	event, found, err := loadIdentityUserFieldEvent(ctx, tx, key)
	if err != nil || !found {
		return IdentityUserFieldValueMutation{}, found, err
	}
	mutation, err := resolveIdentityUserFieldReplay(
		ctx, tx, event, action, fingerprint, valueDigest,
	)
	return mutation, true, err
}

func loadIdentityUserFieldEvent(
	ctx context.Context,
	tx pgx.Tx,
	key string,
) (IdentityUserFieldValueEvent, bool, error) {
	query := `
			SELECT id, user_id, field_id, owner_extension_id,
		       field_contract_version, field_schema_digest, declaration_revision,
		       action, previous_revision, next_revision,
		       previous_value_digest, next_value_digest,
		       idempotency_key, request_fingerprint,
		       actor_user_id, audit_event_id, created_at
			FROM identity_user_field_value_events
			WHERE idempotency_key = $1`
	event, err := scanIdentityUserFieldEvent(tx.QueryRow(ctx, query, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return IdentityUserFieldValueEvent{}, false, nil
	}
	if err != nil {
		return IdentityUserFieldValueEvent{}, false, err
	}
	return event, true, nil
}

func resolveIdentityUserFieldReplay(
	ctx context.Context,
	tx pgx.Tx,
	event IdentityUserFieldValueEvent,
	action string,
	fingerprint string,
	valueDigest func(int64, string, []byte) string,
) (IdentityUserFieldValueMutation, error) {
	if event.Action != action || event.RequestFingerprint != fingerprint {
		return IdentityUserFieldValueMutation{}, ErrIdentityUserFieldValueIdempotencyConflict
	}
	current, err := scanStoredIdentityUserFieldValue(
		tx.QueryRow(
			ctx,
			identityUserFieldValueSelect+` WHERE user_id = $1 AND field_id = $2`,
			event.UserID,
			event.FieldID,
		),
		valueDigest,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		value := IdentityUserFieldValue{
			UserID: event.UserID, FieldID: event.FieldID,
			OwnerExtensionID:     event.OwnerExtensionID,
			FieldContractVersion: event.FieldContractVersion,
			FieldSchemaDigest:    event.FieldSchemaDigest,
			DeclarationRevision:  event.DeclarationRevision,
			State:                IdentityUserFieldValueStateActive,
			Revision:             event.NextRevision,
			ValueDigest:          event.NextValueDigest,
			UpdatedByUserID:      event.ActorUserID,
			UpdatedAuditEventID:  event.AuditEventID,
			CreatedAt:            event.CreatedAt,
			UpdatedAt:            event.CreatedAt,
		}
		if event.Action == IdentityUserFieldValueActionErase {
			erasedAt := event.CreatedAt
			value.State = IdentityUserFieldValueStateErased
			value.ValueDigest = ""
			value.ErasedAt = &erasedAt
			value.ErasedByUserID = event.ActorUserID
			value.EraseAuditEventID = event.AuditEventID
		}
		return IdentityUserFieldValueMutation{Value: value, Event: event, Replayed: true}, nil
	}
	if err != nil {
		return IdentityUserFieldValueMutation{}, err
	}
	// Event is the immutable receipt for the original request. Value is current
	// redacted metadata, so an old set receipt never presents a later erase or
	// update as if the original active row were still current.
	return IdentityUserFieldValueMutation{
		Value: current.metadata, Event: event, Replayed: true, CurrentAvailable: true,
	}, nil
}

func identityUserFieldValueProvenanceMatches(
	stored IdentityUserFieldValue,
	fieldID string,
	ownerExtensionID string,
	contractVersion string,
	schemaDigest string,
	declarationRevision int64,
) bool {
	return stored.FieldID == fieldID && stored.OwnerExtensionID == ownerExtensionID &&
		stored.FieldContractVersion == contractVersion && stored.FieldSchemaDigest == schemaDigest &&
		stored.DeclarationRevision == declarationRevision
}

func decodeStoredIdentityUserFieldJSON(raw []byte) (json.RawMessage, any, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || len(raw) > maximumIdentityUserFieldValueBytes {
		return nil, nil, ErrIdentityUserFieldValueStateConflict
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, nil, ErrIdentityUserFieldValueStateConflict
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, nil, ErrIdentityUserFieldValueStateConflict
	}
	return append(json.RawMessage(nil), raw...), decoded, nil
}

func mapIdentityUserFieldStoreError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded),
		errors.Is(err, ErrIdentityUserFieldValueInvalid),
		errors.Is(err, ErrIdentityUserFieldValueNotFound),
		errors.Is(err, ErrIdentityUserFieldValueStateConflict),
		errors.Is(err, ErrIdentityUserFieldValueIdempotencyConflict),
		errors.Is(err, ErrIdentityUserFieldDeclarationStale),
		errors.Is(err, ErrIdentityUserFieldPermissionDenied),
		errors.Is(err, ErrIdentityUserFieldSchemaUnavailable),
		errors.Is(err, ErrIdentityUserFieldSchemaInvalid),
		errors.Is(err, ErrIdentityUserFieldTransactionIsolation),
		errors.Is(err, errIdentityUserFieldRetry):
		return err
	}
	var postgresErr *pgconn.PgError
	if !errors.As(err, &postgresErr) {
		return ErrIdentityUserFieldValueStoreUnavailable
	}
	switch postgresErr.Code {
	case "40001", "40P01":
		return errIdentityUserFieldRetry
	case "23505":
		switch postgresErr.ConstraintName {
		case "identity_user_field_values_pkey":
			return errIdentityUserFieldRetry
		case "identity_user_field_value_events_idempotency_key_key":
			return ErrIdentityUserFieldValueIdempotencyConflict
		}
		return ErrIdentityUserFieldValueStateConflict
	case "23503", "23514", "22P02":
		return ErrIdentityUserFieldValueInvalid
	default:
		return ErrIdentityUserFieldValueStoreUnavailable
	}
}

func identityUserFieldErrorMayFollowConcurrentCommit(err error) bool {
	if errors.Is(err, errIdentityUserFieldRetry) {
		return true
	}
	var postgresErr *pgconn.PgError
	if !errors.As(err, &postgresErr) {
		return false
	}
	return postgresErr.Code == "40001" || postgresErr.Code == "40P01" || postgresErr.Code == "23505"
}

func publicIdentityUserFieldStoreError(err error) error {
	if errors.Is(err, errIdentityUserFieldRetry) {
		return ErrIdentityUserFieldValueStateConflict
	}
	return err
}

func callerIdentityUserFieldStoreError(err error) error {
	if errors.Is(err, errIdentityUserFieldRetry) {
		return ErrIdentityUserFieldTransactionRetry
	}
	return err
}

func identityUserFieldCommitDefinitelyFailed(commitErr error) bool {
	if errors.Is(commitErr, pgx.ErrTxCommitRollback) || pgconn.SafeToRetry(commitErr) {
		return true
	}
	var postgresErr *pgconn.PgError
	if !errors.As(commitErr, &postgresErr) {
		return false
	}
	return strings.HasPrefix(postgresErr.Code, "40") && postgresErr.Code != "40003"
}

const identityUserFieldValueColumns = `
	user_id, field_id, owner_extension_id, field_contract_version,
	field_schema_digest, declaration_revision, value_json, state, revision,
	COALESCE(updated_by_user_id, 0), updated_audit_event_id,
	created_at, updated_at, erased_at,
	COALESCE(erased_by_user_id, 0), COALESCE(erase_audit_event_id, 0)`

const identityUserFieldValueSelect = `SELECT ` + identityUserFieldValueColumns + ` FROM identity_user_field_values`

type identityUserFieldScanner interface {
	Scan(dest ...any) error
}

func scanStoredIdentityUserFieldValue(
	scanner identityUserFieldScanner,
	valueDigest func(int64, string, []byte) string,
) (storedIdentityUserFieldValue, error) {
	var result storedIdentityUserFieldValue
	var raw []byte
	err := scanner.Scan(
		&result.metadata.UserID, &result.metadata.FieldID, &result.metadata.OwnerExtensionID,
		&result.metadata.FieldContractVersion, &result.metadata.FieldSchemaDigest,
		&result.metadata.DeclarationRevision, &raw, &result.metadata.State,
		&result.metadata.Revision, &result.metadata.UpdatedByUserID,
		&result.metadata.UpdatedAuditEventID, &result.metadata.CreatedAt,
		&result.metadata.UpdatedAt, &result.metadata.ErasedAt,
		&result.metadata.ErasedByUserID, &result.metadata.EraseAuditEventID,
	)
	if err != nil {
		return storedIdentityUserFieldValue{}, err
	}
	if result.metadata.State == IdentityUserFieldValueStateActive {
		canonical, decoded, decodeErr := decodeStoredIdentityUserFieldJSON(raw)
		if decodeErr != nil {
			return storedIdentityUserFieldValue{}, decodeErr
		}
		if valueDigest == nil {
			return storedIdentityUserFieldValue{}, ErrIdentityUserFieldValueStoreUnavailable
		}
		result.raw = canonical
		result.decoded = decoded
		result.metadata.ValueDigest = valueDigest(result.metadata.UserID, result.metadata.FieldID, canonical)
	} else if result.metadata.State != IdentityUserFieldValueStateErased || len(raw) != 0 {
		return storedIdentityUserFieldValue{}, ErrIdentityUserFieldValueStateConflict
	}
	return result, nil
}

func scanIdentityUserFieldEvent(scanner identityUserFieldScanner) (IdentityUserFieldValueEvent, error) {
	var event IdentityUserFieldValueEvent
	var previousRevision *int64
	var previousDigest, nextDigest *string
	var actorUserID *int64
	err := scanner.Scan(
		&event.ID, &event.UserID, &event.FieldID, &event.OwnerExtensionID,
		&event.FieldContractVersion, &event.FieldSchemaDigest, &event.DeclarationRevision,
		&event.Action, &previousRevision, &event.NextRevision,
		&previousDigest, &nextDigest, &event.IdempotencyKey,
		&event.RequestFingerprint, &actorUserID, &event.AuditEventID, &event.CreatedAt,
	)
	if previousRevision != nil {
		event.PreviousRevision = *previousRevision
	}
	if previousDigest != nil {
		event.PreviousValueDigest = *previousDigest
	}
	if nextDigest != nil {
		event.NextValueDigest = *nextDigest
	}
	if actorUserID != nil {
		event.ActorUserID = *actorUserID
	}
	return event, err
}

func nullableIdentityUserFieldActor(actorUserID int64) any {
	if actorUserID <= 0 {
		return nil
	}
	return actorUserID
}

func nullableIdentityUserFieldRevision(revision int64) any {
	if revision <= 0 {
		return nil
	}
	return revision
}

func nullableIdentityUserFieldDigest(digest string) any {
	if digest == "" {
		return nil
	}
	return digest
}

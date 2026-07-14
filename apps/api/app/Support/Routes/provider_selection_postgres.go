package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresProviderSelectionStore struct {
	pool *pgxpool.Pool
}

func NewPostgresProviderSelectionStore(pool *pgxpool.Pool) *PostgresProviderSelectionStore {
	return &PostgresProviderSelectionStore{pool: pool}
}

func (s *PostgresProviderSelectionStore) Selected(ctx context.Context, key ProviderSelectionKey) (ProviderSelection, error) {
	if s == nil || s.pool == nil || validateProviderSelectionKey(key) != nil {
		return ProviderSelection{}, ErrProviderSelectionInvalid
	}
	selection, err := scanProviderSelection(s.pool.QueryRow(ctx, providerSelectionLiveSQL+`
		WHERE s.target_route_id = $1 AND s.target_contract_version = $2
		  AND s.method = $3 AND s.path_signature = $4
	`, key.TargetRouteID, key.TargetContractVersion, key.Method, key.PathSignature))
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if existsErr := s.pool.QueryRow(ctx, `SELECT EXISTS (
			SELECT 1 FROM extension_route_provider_selections
			WHERE target_route_id=$1 AND method=$2 AND path_signature=$3
		)`, key.TargetRouteID, key.Method, key.PathSignature).Scan(&exists); existsErr != nil {
			return ProviderSelection{}, fmt.Errorf("inspect stale route provider selection: %w", existsErr)
		}
		if exists {
			return ProviderSelection{}, ErrProviderSelectionStale
		}
		return ProviderSelection{}, ErrProviderSelectionNotFound
	}
	return selection, err
}

func (s *PostgresProviderSelectionStore) Select(ctx context.Context, request SelectProviderRequest) (ProviderSelection, error) {
	if s == nil || s.pool == nil || request.ExpectedRevision < 0 || request.ActorUserID <= 0 ||
		request.AuditEventID <= 0 || validateProviderSelectionKey(request.Key) != nil ||
		!routeIDPattern.MatchString(request.ProviderRouteID) || !contractPattern.MatchString(request.ProviderContractVersion) ||
		validatePluginArtifact(request.ProviderArtifact) != nil {
		return ProviderSelection{}, ErrProviderSelectionInvalid
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return ProviderSelection{}, fmt.Errorf("begin route provider selection: %w", err)
	}
	defer tx.Rollback(ctx)

	versionID, err := activeExactExtensionVersion(ctx, tx, request.ProviderArtifact)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProviderSelection{}, ErrProviderSelectionStale
	}
	if err != nil {
		return ProviderSelection{}, fmt.Errorf("validate exact route provider artifact: %w", err)
	}
	previous, previousErr := selectedProviderForUpdate(ctx, tx, request.Key)
	if previousErr != nil && !errors.Is(previousErr, pgx.ErrNoRows) {
		return ProviderSelection{}, previousErr
	}
	if errors.Is(previousErr, pgx.ErrNoRows) {
		if request.ExpectedRevision != 0 {
			return ProviderSelection{}, ErrProviderSelectionRevisionConflict
		}
	} else if previous.Revision != request.ExpectedRevision {
		return ProviderSelection{}, ErrProviderSelectionRevisionConflict
	}

	revision := int64(1)
	if previousErr == nil {
		revision = previous.Revision + 1
	}
	selection := ProviderSelection{
		Key: request.Key, ProviderRouteID: request.ProviderRouteID,
		ProviderContractVersion:    request.ProviderContractVersion,
		ProviderExtensionID:        request.ProviderArtifact.ExtensionID,
		ProviderExtensionVersionID: versionID,
		ProviderExtensionVersion:   request.ProviderArtifact.ExtensionVersion,
		ProviderPackageDigest:      request.ProviderArtifact.PackageDigest,
		SelectedByUserID:           request.ActorUserID, SelectionAuditEventID: request.AuditEventID,
		Revision: revision,
	}
	if previousErr == nil {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `
			UPDATE extension_route_provider_selections SET
				target_contract_version=$4, provider_route_id=$5, provider_contract_version=$6,
				provider_extension_id=$7, provider_extension_version_id=$8,
				provider_extension_version=$9, provider_package_digest=$10,
				selected_by_user_id=$11, selection_audit_event_id=$12,
				revision=$13, updated_at=statement_timestamp()
			WHERE target_route_id=$1 AND method=$2 AND path_signature=$3 AND revision=$14
		`, request.Key.TargetRouteID, request.Key.Method, request.Key.PathSignature,
			request.Key.TargetContractVersion, selection.ProviderRouteID, selection.ProviderContractVersion,
			selection.ProviderExtensionID, selection.ProviderExtensionVersionID, selection.ProviderExtensionVersion,
			selection.ProviderPackageDigest, request.ActorUserID, request.AuditEventID, revision, request.ExpectedRevision)
		if err == nil && tag.RowsAffected() != 1 {
			return ProviderSelection{}, ErrProviderSelectionRevisionConflict
		}
	} else {
		_, err = tx.Exec(ctx, `
			INSERT INTO extension_route_provider_selections (
				target_route_id, target_contract_version, method, path_signature,
				provider_route_id, provider_contract_version, provider_extension_id,
				provider_extension_version_id, provider_extension_version, provider_package_digest,
				selected_by_user_id, selection_audit_event_id, revision
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		`, request.Key.TargetRouteID, request.Key.TargetContractVersion, request.Key.Method, request.Key.PathSignature,
			selection.ProviderRouteID, selection.ProviderContractVersion, selection.ProviderExtensionID,
			selection.ProviderExtensionVersionID, selection.ProviderExtensionVersion, selection.ProviderPackageDigest,
			request.ActorUserID, request.AuditEventID, revision)
	}
	if err != nil {
		return ProviderSelection{}, mapProviderSelectionWriteError(err)
	}
	if err := insertProviderSelectionEvent(ctx, tx, "select", selection, providerSelectionPointer(previous, previousErr == nil), &selection, request.ActorUserID, request.AuditEventID, ""); err != nil {
		return ProviderSelection{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProviderSelection{}, fmt.Errorf("commit route provider selection: %w", err)
	}
	return s.Selected(ctx, request.Key)
}

func (s *PostgresProviderSelectionStore) Reset(ctx context.Context, request ResetProviderRequest) error {
	if s == nil || s.pool == nil || request.ExpectedRevision <= 0 || request.ActorUserID <= 0 ||
		request.AuditEventID <= 0 || validateProviderSelectionKey(request.Key) != nil || !validReasonCode(request.ReasonCode, false) {
		return ErrProviderSelectionInvalid
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("begin route provider reset: %w", err)
	}
	defer tx.Rollback(ctx)
	previous, err := selectedProviderForUpdate(ctx, tx, request.Key)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrProviderSelectionNotFound
	}
	if err != nil {
		return err
	}
	if previous.Key.TargetContractVersion != request.Key.TargetContractVersion {
		return ErrProviderSelectionStale
	}
	if previous.Revision != request.ExpectedRevision {
		return ErrProviderSelectionRevisionConflict
	}
	tag, err := tx.Exec(ctx, `DELETE FROM extension_route_provider_selections
		WHERE target_route_id=$1 AND target_contract_version=$2
		  AND method=$3 AND path_signature=$4 AND revision=$5`,
		request.Key.TargetRouteID, request.Key.TargetContractVersion, request.Key.Method,
		request.Key.PathSignature, request.ExpectedRevision)
	if err != nil {
		return fmt.Errorf("delete route provider selection: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrProviderSelectionRevisionConflict
	}
	if err := insertProviderSelectionEvent(ctx, tx, "reset", previous, &previous, nil, request.ActorUserID, request.AuditEventID, request.ReasonCode); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit route provider reset: %w", err)
	}
	return nil
}

func (s *PostgresProviderSelectionStore) InvalidateExtension(ctx context.Context, request InvalidateProviderRequest) (int64, error) {
	if s == nil || s.pool == nil || !routeIDPattern.MatchString(request.ExtensionID) || request.ActorUserID <= 0 ||
		request.AuditEventID <= 0 || !validReasonCode(request.ReasonCode, true) {
		return 0, ErrProviderSelectionInvalid
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return 0, fmt.Errorf("begin route provider invalidation: %w", err)
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, providerSelectionSQL+`
		WHERE provider_extension_id=$1 ORDER BY target_route_id, method, path_signature FOR UPDATE
	`, request.ExtensionID)
	if err != nil {
		return 0, fmt.Errorf("lock route provider selections: %w", err)
	}
	selections := make([]ProviderSelection, 0)
	for rows.Next() {
		selection, scanErr := scanProviderSelection(rows)
		if scanErr != nil {
			rows.Close()
			return 0, scanErr
		}
		selections = append(selections, selection)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	for _, selection := range selections {
		tag, deleteErr := tx.Exec(ctx, `DELETE FROM extension_route_provider_selections
			WHERE target_route_id=$1 AND method=$2 AND path_signature=$3 AND revision=$4`,
			selection.Key.TargetRouteID, selection.Key.Method, selection.Key.PathSignature, selection.Revision)
		if deleteErr != nil {
			return 0, fmt.Errorf("delete invalid route provider selection: %w", deleteErr)
		}
		if tag.RowsAffected() != 1 {
			return 0, ErrProviderSelectionRevisionConflict
		}
		if eventErr := insertProviderSelectionEvent(ctx, tx, "invalidate", selection, &selection, nil,
			request.ActorUserID, request.AuditEventID, request.ReasonCode); eventErr != nil {
			return 0, eventErr
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit route provider invalidation: %w", err)
	}
	return int64(len(selections)), nil
}

func (s *PostgresProviderSelectionStore) ListEvents(ctx context.Context, key ProviderSelectionKey, limit int) ([]ProviderSelectionEvent, error) {
	if s == nil || s.pool == nil || validateProviderSelectionKey(key) != nil {
		return nil, ErrProviderSelectionInvalid
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, target_route_id, target_contract_version, method, path_signature,
		       action, previous_provider, selected_provider, COALESCE(actor_user_id,0),
		       audit_event_id, reason_code, selection_revision, created_at
		FROM extension_route_provider_selection_events
		WHERE target_route_id=$1 AND target_contract_version=$2
		  AND method=$3 AND path_signature=$4
		ORDER BY created_at DESC, id DESC LIMIT $5
	`, key.TargetRouteID, key.TargetContractVersion, key.Method, key.PathSignature, limit)
	if err != nil {
		return nil, fmt.Errorf("list route provider selection events: %w", err)
	}
	defer rows.Close()
	result := make([]ProviderSelectionEvent, 0)
	for rows.Next() {
		var event ProviderSelectionEvent
		var previousJSON, selectedJSON []byte
		if err := rows.Scan(&event.ID, &event.Key.TargetRouteID, &event.Key.TargetContractVersion,
			&event.Key.Method, &event.Key.PathSignature, &event.Action, &previousJSON, &selectedJSON,
			&event.ActorUserID, &event.AuditEventID, &event.ReasonCode, &event.SelectionRevision, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan route provider selection event: %w", err)
		}
		if len(previousJSON) > 0 {
			var previous ProviderSelection
			if err := json.Unmarshal(previousJSON, &previous); err != nil {
				return nil, fmt.Errorf("decode previous route provider: %w", err)
			}
			event.PreviousProvider = &previous
		}
		if len(selectedJSON) > 0 {
			var selected ProviderSelection
			if err := json.Unmarshal(selectedJSON, &selected); err != nil {
				return nil, fmt.Errorf("decode selected route provider: %w", err)
			}
			event.SelectedProvider = &selected
		}
		result = append(result, event)
	}
	return result, rows.Err()
}

const providerSelectionSQL = `
	SELECT target_route_id, target_contract_version, method, path_signature,
	       provider_route_id, provider_contract_version, provider_extension_id,
	       provider_extension_version_id, provider_extension_version, provider_package_digest,
	       COALESCE(selected_by_user_id,0), selection_audit_event_id, revision, selected_at, updated_at
	FROM extension_route_provider_selections
`

// Resolution proves that the durable version id is still the enabled active
// exact artifact. A same-version reinstall with another immutable row cannot
// inherit the old administrator choice accidentally.
const providerSelectionLiveSQL = `
	SELECT s.target_route_id, s.target_contract_version, s.method, s.path_signature,
	       s.provider_route_id, s.provider_contract_version, s.provider_extension_id,
	       s.provider_extension_version_id, s.provider_extension_version, s.provider_package_digest,
	       COALESCE(s.selected_by_user_id,0), s.selection_audit_event_id, s.revision, s.selected_at, s.updated_at
	FROM extension_route_provider_selections s
	JOIN extensions e ON e.id=s.provider_extension_id AND e.type='plugin' AND e.status='enabled'
	  AND e.active_version_id=s.provider_extension_version_id
	JOIN extension_versions v ON v.id=s.provider_extension_version_id AND v.extension_id=e.id
	  AND v.version=s.provider_extension_version AND v.package_digest=s.provider_package_digest
`

type providerSelectionScanner interface{ Scan(...any) error }

func scanProviderSelection(scanner providerSelectionScanner) (ProviderSelection, error) {
	var result ProviderSelection
	err := scanner.Scan(&result.Key.TargetRouteID, &result.Key.TargetContractVersion, &result.Key.Method,
		&result.Key.PathSignature, &result.ProviderRouteID, &result.ProviderContractVersion,
		&result.ProviderExtensionID, &result.ProviderExtensionVersionID, &result.ProviderExtensionVersion,
		&result.ProviderPackageDigest, &result.SelectedByUserID, &result.SelectionAuditEventID,
		&result.Revision, &result.SelectedAt, &result.UpdatedAt)
	if err != nil {
		return ProviderSelection{}, err
	}
	return result, nil
}

func selectedProviderForUpdate(ctx context.Context, tx pgx.Tx, key ProviderSelectionKey) (ProviderSelection, error) {
	return scanProviderSelection(tx.QueryRow(ctx, providerSelectionSQL+`
		WHERE target_route_id=$1 AND method=$2 AND path_signature=$3 FOR UPDATE
	`, key.TargetRouteID, key.Method, key.PathSignature))
}

func activeExactExtensionVersion(ctx context.Context, tx pgx.Tx, artifact PluginArtifact) (int64, error) {
	var versionID int64
	err := tx.QueryRow(ctx, `
		SELECT v.id FROM extensions e
		JOIN extension_versions v ON v.id=e.active_version_id AND v.extension_id=e.id
		WHERE e.id=$1 AND e.type='plugin' AND e.status='enabled'
		  AND v.version=$2 AND v.package_digest=$3
		FOR SHARE OF e, v
	`, artifact.ExtensionID, artifact.ExtensionVersion, artifact.PackageDigest).Scan(&versionID)
	return versionID, err
}

func insertProviderSelectionEvent(ctx context.Context, tx pgx.Tx, action string, selection ProviderSelection,
	previous, selected *ProviderSelection, actorUserID, auditEventID int64, reason string) error {
	previousJSON, err := marshalProviderSelection(previous)
	if err != nil {
		return err
	}
	selectedJSON, err := marshalProviderSelection(selected)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO extension_route_provider_selection_events (
			target_route_id, target_contract_version, method, path_signature, action,
			previous_provider, selected_provider, actor_user_id, audit_event_id,
			reason_code, selection_revision
		) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9,$10,$11)
	`, selection.Key.TargetRouteID, selection.Key.TargetContractVersion, selection.Key.Method,
		selection.Key.PathSignature, action, previousJSON, selectedJSON, actorUserID, auditEventID,
		reason, selection.Revision+boolToInt64(action != "select"))
	if err != nil {
		return fmt.Errorf("append route provider selection event: %w", err)
	}
	return nil
}

func marshalProviderSelection(selection *ProviderSelection) (any, error) {
	if selection == nil {
		return nil, nil
	}
	value, err := json.Marshal(selection)
	if err != nil {
		return nil, fmt.Errorf("encode route provider selection: %w", err)
	}
	return string(value), nil
}

func providerSelectionPointer(value ProviderSelection, present bool) *ProviderSelection {
	if !present {
		return nil
	}
	return &value
}

func mapProviderSelectionWriteError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrProviderSelectionRevisionConflict
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23505" || pgErr.Code == "40001") {
		return ErrProviderSelectionRevisionConflict
	}
	return fmt.Errorf("write route provider selection: %w", err)
}

func boolToInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

var _ ProviderSelectionStore = (*PostgresProviderSelectionStore)(nil)
var _ = time.Time{}

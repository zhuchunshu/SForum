package extensionsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresProviderSlotSelectionStore struct{ pool *pgxpool.Pool }

func NewPostgresProviderSlotSelectionStore(pool *pgxpool.Pool) *PostgresProviderSlotSelectionStore {
	return &PostgresProviderSlotSelectionStore{pool: pool}
}

func (s *PostgresProviderSlotSelectionStore) Desired(ctx context.Context, contractID string) (ProviderSlotSelection, error) {
	contractID = strings.TrimSpace(contractID)
	if s == nil || s.pool == nil || ctx == nil || contractID == "" {
		return ProviderSlotSelection{}, ErrProviderSlotSelectionInvalid
	}
	selection, err := scanProviderSlotSelection(s.pool.QueryRow(ctx, providerSlotSelectionSQL+` WHERE contract_id=$1`, contractID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ProviderSlotSelection{}, ErrProviderSlotSelectionNotFound
	}
	return selection, err
}

func (s *PostgresProviderSlotSelectionStore) Selected(ctx context.Context, contractID string) (ProviderSlotSelection, error) {
	contractID = strings.TrimSpace(contractID)
	if s == nil || s.pool == nil || ctx == nil || contractID == "" {
		return ProviderSlotSelection{}, ErrProviderSlotSelectionInvalid
	}
	selection, err := scanProviderSlotSelection(s.pool.QueryRow(ctx, providerSlotSelectionLiveSQL+` WHERE s.contract_id=$1`, contractID))
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if existsErr := s.pool.QueryRow(ctx, `SELECT EXISTS (
			SELECT 1 FROM extension_provider_slot_selections WHERE contract_id=$1
		)`, contractID).Scan(&exists); existsErr != nil {
			return ProviderSlotSelection{}, fmt.Errorf("inspect stale provider slot selection: %w", existsErr)
		}
		if exists {
			return ProviderSlotSelection{}, ErrProviderSlotSelectionStale
		}
		return ProviderSlotSelection{}, ErrProviderSlotSelectionNotFound
	}
	return selection, err
}

func (s *PostgresProviderSlotSelectionStore) Select(ctx context.Context, request SelectProviderSlotRequest) (ProviderSlotSelection, error) {
	if s == nil || s.pool == nil || ctx == nil || request.ExpectedRevision < 0 || request.ActorUserID <= 0 ||
		request.AuditEventID <= 0 || !validProviderSlotSelectionRequest(request) {
		return ProviderSlotSelection{}, ErrProviderSlotSelectionInvalid
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return ProviderSlotSelection{}, fmt.Errorf("begin provider slot selection: %w", err)
	}
	defer tx.Rollback(ctx)

	contractVersionID, err := activeExactProviderSlotExtensionVersion(ctx, tx, request.Contract.Artifact)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProviderSlotSelection{}, ErrProviderSlotSelectionStale
	}
	if err != nil {
		return ProviderSlotSelection{}, fmt.Errorf("validate provider slot contract artifact: %w", err)
	}
	providerVersionID, err := activeExactProviderSlotExtensionVersion(ctx, tx, request.Candidate.Artifact)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProviderSlotSelection{}, ErrProviderSlotSelectionStale
	}
	if err != nil {
		return ProviderSlotSelection{}, fmt.Errorf("validate provider slot candidate artifact: %w", err)
	}
	previous, previousErr := providerSlotSelectionForUpdate(ctx, tx, request.Contract.ID)
	if previousErr != nil && !errors.Is(previousErr, pgx.ErrNoRows) {
		return ProviderSlotSelection{}, previousErr
	}
	if errors.Is(previousErr, pgx.ErrNoRows) {
		if request.ExpectedRevision != 0 {
			return ProviderSlotSelection{}, ErrProviderSlotSelectionRevisionConflict
		}
	} else if previous.Revision != request.ExpectedRevision {
		return ProviderSlotSelection{}, ErrProviderSlotSelectionRevisionConflict
	}

	revision := int64(1)
	if previousErr == nil {
		revision = previous.Revision + 1
	}
	selection := ProviderSlotSelection{
		ContractID: request.Contract.ID, ContractVersion: request.Contract.ContractVersion, Slot: request.Contract.Slot,
		ContractArtifact: request.Contract.Artifact, ContractVersionID: contractVersionID,
		CandidateID: request.Candidate.ID, ProviderArtifact: request.Candidate.Artifact, ProviderVersionID: providerVersionID,
		SelectedByUserID: request.ActorUserID, SelectionAuditID: request.AuditEventID, Revision: revision,
	}
	if previousErr == nil {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `
			UPDATE extension_provider_slot_selections SET
				contract_version=$2, slot=$3, contract_extension_id=$4,
				contract_extension_version_id=$5, contract_extension_version=$6,
				contract_package_digest=$7, candidate_id=$8, provider_extension_id=$9,
				provider_extension_version_id=$10, provider_extension_version=$11,
				provider_package_digest=$12, selected_by_user_id=$13,
				selection_audit_event_id=$14, revision=$15, updated_at=statement_timestamp()
			WHERE contract_id=$1 AND revision=$16
		`, selection.ContractID, selection.ContractVersion, selection.Slot,
			selection.ContractArtifact.ExtensionID, selection.ContractVersionID,
			selection.ContractArtifact.ExtensionVersion, selection.ContractArtifact.PackageDigest,
			selection.CandidateID, selection.ProviderArtifact.ExtensionID, selection.ProviderVersionID,
			selection.ProviderArtifact.ExtensionVersion, selection.ProviderArtifact.PackageDigest,
			selection.SelectedByUserID, selection.SelectionAuditID, selection.Revision, request.ExpectedRevision)
		if err == nil && tag.RowsAffected() != 1 {
			return ProviderSlotSelection{}, ErrProviderSlotSelectionRevisionConflict
		}
	} else {
		_, err = tx.Exec(ctx, `
			INSERT INTO extension_provider_slot_selections (
				contract_id, contract_version, slot, contract_extension_id,
				contract_extension_version_id, contract_extension_version, contract_package_digest,
				candidate_id, provider_extension_id, provider_extension_version_id,
				provider_extension_version, provider_package_digest,
				selected_by_user_id, selection_audit_event_id, revision
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		`, selection.ContractID, selection.ContractVersion, selection.Slot,
			selection.ContractArtifact.ExtensionID, selection.ContractVersionID,
			selection.ContractArtifact.ExtensionVersion, selection.ContractArtifact.PackageDigest,
			selection.CandidateID, selection.ProviderArtifact.ExtensionID, selection.ProviderVersionID,
			selection.ProviderArtifact.ExtensionVersion, selection.ProviderArtifact.PackageDigest,
			selection.SelectedByUserID, selection.SelectionAuditID, selection.Revision)
	}
	if err != nil {
		return ProviderSlotSelection{}, mapProviderSlotSelectionWriteError(err)
	}
	if err := insertProviderSlotSelectionEvent(ctx, tx, "select", selection,
		providerSlotSelectionPointer(previous, previousErr == nil), &selection,
		request.ActorUserID, request.AuditEventID, ""); err != nil {
		return ProviderSlotSelection{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProviderSlotSelection{}, fmt.Errorf("commit provider slot selection: %w", err)
	}
	return s.Selected(ctx, request.Contract.ID)
}

func (s *PostgresProviderSlotSelectionStore) Reset(ctx context.Context, request ResetProviderSlotRequest) error {
	if s == nil || s.pool == nil || ctx == nil || request.ContractID == "" || request.ExpectedRevision <= 0 ||
		request.ActorUserID <= 0 || request.AuditEventID <= 0 || !validProviderSlotReason(request.ReasonCode, false) {
		return ErrProviderSlotSelectionInvalid
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("begin provider slot reset: %w", err)
	}
	defer tx.Rollback(ctx)
	previous, err := providerSlotSelectionForUpdate(ctx, tx, request.ContractID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrProviderSlotSelectionNotFound
	}
	if err != nil {
		return err
	}
	if previous.Revision != request.ExpectedRevision {
		return ErrProviderSlotSelectionRevisionConflict
	}
	tag, err := tx.Exec(ctx, `DELETE FROM extension_provider_slot_selections WHERE contract_id=$1 AND revision=$2`,
		request.ContractID, request.ExpectedRevision)
	if err != nil {
		return fmt.Errorf("delete provider slot selection: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrProviderSlotSelectionRevisionConflict
	}
	if err := insertProviderSlotSelectionEvent(ctx, tx, "reset", previous, &previous, nil,
		request.ActorUserID, request.AuditEventID, request.ReasonCode); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit provider slot reset: %w", err)
	}
	return nil
}

func (s *PostgresProviderSlotSelectionStore) InvalidateExtension(ctx context.Context, request InvalidateProviderSlotRequest) (int64, error) {
	if s == nil || s.pool == nil || ctx == nil || request.ExtensionID == "" || request.ActorUserID <= 0 ||
		request.AuditEventID <= 0 || !validProviderSlotReason(request.ReasonCode, true) {
		return 0, ErrProviderSlotSelectionInvalid
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return 0, fmt.Errorf("begin provider slot invalidation: %w", err)
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, providerSlotSelectionSQL+`
		WHERE contract_extension_id=$1 OR provider_extension_id=$1 ORDER BY contract_id FOR UPDATE
	`, request.ExtensionID)
	if err != nil {
		return 0, fmt.Errorf("lock provider slot selections: %w", err)
	}
	selections := make([]ProviderSlotSelection, 0)
	for rows.Next() {
		selection, scanErr := scanProviderSlotSelection(rows)
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
		tag, deleteErr := tx.Exec(ctx, `DELETE FROM extension_provider_slot_selections WHERE contract_id=$1 AND revision=$2`,
			selection.ContractID, selection.Revision)
		if deleteErr != nil {
			return 0, fmt.Errorf("delete invalid provider slot selection: %w", deleteErr)
		}
		if tag.RowsAffected() != 1 {
			return 0, ErrProviderSlotSelectionRevisionConflict
		}
		if eventErr := insertProviderSlotSelectionEvent(ctx, tx, "invalidate", selection, &selection, nil,
			request.ActorUserID, request.AuditEventID, request.ReasonCode); eventErr != nil {
			return 0, eventErr
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit provider slot invalidation: %w", err)
	}
	return int64(len(selections)), nil
}

func (s *PostgresProviderSlotSelectionStore) ListEvents(ctx context.Context, contractID string, limit int) ([]ProviderSlotSelectionEvent, error) {
	contractID = strings.TrimSpace(contractID)
	if s == nil || s.pool == nil || ctx == nil || contractID == "" {
		return nil, ErrProviderSlotSelectionInvalid
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, contract_id, contract_version, slot, action,
		       previous_selection, selected_selection, COALESCE(actor_user_id,0),
		       audit_event_id, reason_code, selection_revision, created_at
		FROM extension_provider_slot_selection_events
		WHERE contract_id=$1 ORDER BY created_at DESC, id DESC LIMIT $2
	`, contractID, limit)
	if err != nil {
		return nil, fmt.Errorf("list provider slot selection events: %w", err)
	}
	defer rows.Close()
	result := make([]ProviderSlotSelectionEvent, 0)
	for rows.Next() {
		var event ProviderSlotSelectionEvent
		var previousJSON, selectedJSON []byte
		if err := rows.Scan(&event.ID, &event.ContractID, &event.ContractVersion, &event.Slot, &event.Action,
			&previousJSON, &selectedJSON, &event.ActorUserID, &event.AuditEventID, &event.ReasonCode,
			&event.SelectionRevision, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan provider slot selection event: %w", err)
		}
		if len(previousJSON) > 0 {
			var previous ProviderSlotSelection
			if err := json.Unmarshal(previousJSON, &previous); err != nil {
				return nil, fmt.Errorf("decode previous provider slot selection: %w", err)
			}
			event.PreviousSelection = &previous
		}
		if len(selectedJSON) > 0 {
			var selected ProviderSlotSelection
			if err := json.Unmarshal(selectedJSON, &selected); err != nil {
				return nil, fmt.Errorf("decode selected provider slot selection: %w", err)
			}
			event.SelectedSelection = &selected
		}
		result = append(result, event)
	}
	return result, rows.Err()
}

const providerSlotSelectionSQL = `
	SELECT contract_id, contract_version, slot, contract_extension_id,
	       contract_extension_version_id, contract_extension_version, contract_package_digest,
	       candidate_id, provider_extension_id, provider_extension_version_id,
	       provider_extension_version, provider_package_digest,
	       COALESCE(selected_by_user_id,0), selection_audit_event_id, revision, selected_at, updated_at
	FROM extension_provider_slot_selections
`

const providerSlotSelectionLiveSQL = `
	SELECT s.contract_id, s.contract_version, s.slot, s.contract_extension_id,
	       s.contract_extension_version_id, s.contract_extension_version, s.contract_package_digest,
	       s.candidate_id, s.provider_extension_id, s.provider_extension_version_id,
	       s.provider_extension_version, s.provider_package_digest,
	       COALESCE(s.selected_by_user_id,0), s.selection_audit_event_id, s.revision, s.selected_at, s.updated_at
	FROM extension_provider_slot_selections s
	JOIN extensions contract_extension
	  ON contract_extension.id=s.contract_extension_id AND contract_extension.type='plugin'
	 AND contract_extension.status='enabled' AND contract_extension.active_version_id=s.contract_extension_version_id
	JOIN extension_versions contract_version
	  ON contract_version.id=s.contract_extension_version_id AND contract_version.extension_id=contract_extension.id
	 AND contract_version.version=s.contract_extension_version AND contract_version.package_digest=s.contract_package_digest
	JOIN extensions provider_extension
	  ON provider_extension.id=s.provider_extension_id AND provider_extension.type='plugin'
	 AND provider_extension.status='enabled' AND provider_extension.active_version_id=s.provider_extension_version_id
	JOIN extension_versions provider_version
	  ON provider_version.id=s.provider_extension_version_id AND provider_version.extension_id=provider_extension.id
	 AND provider_version.version=s.provider_extension_version AND provider_version.package_digest=s.provider_package_digest
`

type providerSlotSelectionScanner interface{ Scan(...any) error }

func scanProviderSlotSelection(scanner providerSlotSelectionScanner) (ProviderSlotSelection, error) {
	var result ProviderSlotSelection
	err := scanner.Scan(
		&result.ContractID, &result.ContractVersion, &result.Slot, &result.ContractArtifact.ExtensionID,
		&result.ContractVersionID, &result.ContractArtifact.ExtensionVersion, &result.ContractArtifact.PackageDigest,
		&result.CandidateID, &result.ProviderArtifact.ExtensionID, &result.ProviderVersionID,
		&result.ProviderArtifact.ExtensionVersion, &result.ProviderArtifact.PackageDigest,
		&result.SelectedByUserID, &result.SelectionAuditID, &result.Revision, &result.SelectedAt, &result.UpdatedAt,
	)
	return result, err
}

func providerSlotSelectionForUpdate(ctx context.Context, tx pgx.Tx, contractID string) (ProviderSlotSelection, error) {
	return scanProviderSlotSelection(tx.QueryRow(ctx, providerSlotSelectionSQL+` WHERE contract_id=$1 FOR UPDATE`, contractID))
}

func activeExactProviderSlotExtensionVersion(ctx context.Context, tx pgx.Tx, artifact HookArtifact) (int64, error) {
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

func insertProviderSlotSelectionEvent(ctx context.Context, tx pgx.Tx, action string, selection ProviderSlotSelection,
	previous, selected *ProviderSlotSelection, actorUserID, auditEventID int64, reason string) error {
	previousJSON, err := marshalProviderSlotSelection(previous)
	if err != nil {
		return err
	}
	selectedJSON, err := marshalProviderSlotSelection(selected)
	if err != nil {
		return err
	}
	revision := selection.Revision
	if action != "select" {
		revision++
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO extension_provider_slot_selection_events (
			contract_id, contract_version, slot, action, previous_selection,
			selected_selection, actor_user_id, audit_event_id, reason_code, selection_revision
		) VALUES ($1,$2,$3,$4,$5::jsonb,$6::jsonb,$7,$8,$9,$10)
	`, selection.ContractID, selection.ContractVersion, selection.Slot, action, previousJSON, selectedJSON,
		actorUserID, auditEventID, reason, revision)
	if err != nil {
		return fmt.Errorf("append provider slot selection event: %w", err)
	}
	return nil
}

func marshalProviderSlotSelection(selection *ProviderSlotSelection) (any, error) {
	if selection == nil {
		return nil, nil
	}
	value, err := json.Marshal(selection)
	if err != nil {
		return nil, fmt.Errorf("encode provider slot selection: %w", err)
	}
	return string(value), nil
}

func providerSlotSelectionPointer(value ProviderSlotSelection, present bool) *ProviderSlotSelection {
	if !present {
		return nil
	}
	return &value
}

func validProviderSlotSelectionRequest(request SelectProviderSlotRequest) bool {
	return request.Contract.ID != "" && request.Contract.Slot != "" && request.Candidate.ID != "" &&
		request.Candidate.TargetID == request.Contract.ID &&
		extensionDatabaseContractPattern.MatchString(request.Contract.ContractVersion) &&
		validProviderSlotArtifact(request.Contract.Artifact) && validProviderSlotArtifact(request.Candidate.Artifact)
}

func validProviderSlotArtifact(artifact HookArtifact) bool {
	return artifact.ExtensionID != "" && artifact.ExtensionVersion != "" && validLifecycleCleanupDigest(artifact.PackageDigest)
}

func mapProviderSlotSelectionWriteError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrProviderSlotSelectionRevisionConflict
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23505" || pgErr.Code == "40001") {
		return ErrProviderSlotSelectionRevisionConflict
	}
	return providerSlotSelectionError("write", err)
}

var _ ProviderSlotSelectionStore = (*PostgresProviderSlotSelectionStore)(nil)

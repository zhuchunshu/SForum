package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var errIdentitySessionPolicyRetry = errors.New("identity: session policy transaction should retry")

const identitySessionPolicySelectionColumns = `
	policy_id, provider_contract_version, owner_extension_id,
	owner_extension_version_id, owner_extension_version, owner_package_digest,
	declaration_revision, revision, COALESCE(selected_by_user_id, 0),
	selection_audit_event_id, selected_at, updated_at`

const identitySessionPolicySelectionSelect = `SELECT ` + identitySessionPolicySelectionColumns + `
	FROM identity_session_policy_selection WHERE singleton = TRUE`

type identitySessionPolicyScanner interface {
	Scan(dest ...any) error
}

func scanIdentitySessionPolicySelection(
	scanner identitySessionPolicyScanner,
) (IdentitySessionPolicySelection, error) {
	var selection IdentitySessionPolicySelection
	var providerContractVersion, ownerExtensionID, ownerExtensionVersion, ownerPackageDigest *string
	var ownerExtensionVersionID, declarationRevision *int64
	err := scanner.Scan(
		&selection.PolicyID,
		&providerContractVersion,
		&ownerExtensionID,
		&ownerExtensionVersionID,
		&ownerExtensionVersion,
		&ownerPackageDigest,
		&declarationRevision,
		&selection.Revision,
		&selection.SelectedByUserID,
		&selection.SelectionAuditEventID,
		&selection.SelectedAt,
		&selection.UpdatedAt,
	)
	if err != nil {
		return IdentitySessionPolicySelection{}, err
	}
	if providerContractVersion != nil {
		selection.ProviderContractVersion = *providerContractVersion
	}
	if ownerExtensionID != nil {
		selection.OwnerExtensionID = *ownerExtensionID
	}
	if ownerExtensionVersionID != nil {
		selection.OwnerExtensionVersionID = *ownerExtensionVersionID
	}
	if ownerExtensionVersion != nil {
		selection.OwnerExtensionVersion = *ownerExtensionVersion
	}
	if ownerPackageDigest != nil {
		selection.OwnerPackageDigest = *ownerPackageDigest
	}
	if declarationRevision != nil {
		selection.DeclarationRevision = *declarationRevision
	}
	if !validIdentitySessionPolicySelection(selection) {
		return IdentitySessionPolicySelection{}, ErrIdentitySessionPolicyStoreUnavailable
	}
	return selection, nil
}

func currentIdentitySessionPolicySelectionTx(
	ctx context.Context,
	tx pgx.Tx,
	lock bool,
) (IdentitySessionPolicySelection, bool, error) {
	query := identitySessionPolicySelectionSelect
	if lock {
		query += ` FOR UPDATE`
	}
	selection, err := scanIdentitySessionPolicySelection(tx.QueryRow(ctx, query))
	if errors.Is(err, pgx.ErrNoRows) {
		var evidenceExists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM identity_session_policy_selection_events)
		`).Scan(&evidenceExists); err != nil {
			return IdentitySessionPolicySelection{}, false, err
		}
		if evidenceExists {
			return IdentitySessionPolicySelection{}, false, ErrIdentitySessionPolicyStoreUnavailable
		}
		return identitySessionPolicyCoreSelection(0, true), false, nil
	}
	return selection, err == nil, err
}

func validIdentitySessionPolicySelection(selection IdentitySessionPolicySelection) bool {
	if selection.Revision <= 0 || selection.SelectionAuditEventID <= 0 || selection.SelectedAt.IsZero() ||
		selection.UpdatedAt.Before(selection.SelectedAt) {
		return false
	}
	return validIdentitySessionPolicyEvidence(selection.IdentitySessionPolicyEvidence)
}

func validIdentitySessionPolicyEvidence(evidence IdentitySessionPolicyEvidence) bool {
	if evidence.PolicyID == IdentitySessionPolicyCoreDefault {
		return evidence.ProviderContractVersion == "" && evidence.OwnerExtensionID == "" &&
			evidence.OwnerExtensionVersionID == 0 && evidence.OwnerExtensionVersion == "" &&
			evidence.OwnerPackageDigest == "" && evidence.DeclarationRevision == 0
	}
	return validIdentityUserFieldID(evidence.PolicyID) && evidence.ProviderContractVersion != "" &&
		validIdentityUserFieldID(evidence.OwnerExtensionID) && evidence.OwnerExtensionVersionID > 0 &&
		evidence.OwnerExtensionVersion != "" && validExternalIdentityDigest(evidence.OwnerPackageDigest) &&
		evidence.DeclarationRevision > 0
}

type identitySessionPolicyAuditMetadata struct {
	PreviousSelection     *IdentitySessionPolicyEvidence `json:"previousSelection"`
	SelectedSelection     *IdentitySessionPolicyEvidence `json:"selectedSelection"`
	SelectionRevision     int64                          `json:"selectionRevision"`
	ReasonCode            string                         `json:"reasonCode,omitempty"`
	LifecycleAuditEventID int64                          `json:"lifecycleAuditEventId,omitempty"`
}

func insertIdentitySessionPolicyAudit(
	ctx context.Context,
	tx pgx.Tx,
	action string,
	previous *IdentitySessionPolicyEvidence,
	selected *IdentitySessionPolicyEvidence,
	selectionRevision int64,
	actorUserID int64,
	reasonCode string,
	lifecycleAuditEventID int64,
) (int64, error) {
	metadata, err := json.Marshal(identitySessionPolicyAuditMetadata{
		PreviousSelection:     previous,
		SelectedSelection:     selected,
		SelectionRevision:     selectionRevision,
		ReasonCode:            reasonCode,
		LifecycleAuditEventID: lifecycleAuditEventID,
	})
	if err != nil {
		return 0, fmt.Errorf("encode identity session policy audit: %w", err)
	}
	var auditID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO audit_events (actor_user_id, action, metadata)
		VALUES ($1, $2, $3::jsonb)
		RETURNING id
	`, nullableIdentitySessionPolicyActor(actorUserID), "identity.session_policy."+action, metadata).Scan(&auditID); err != nil {
		return 0, fmt.Errorf("record identity session policy audit: %w", err)
	}
	return auditID, nil
}

func insertIdentitySessionPolicyEvent(
	ctx context.Context,
	tx pgx.Tx,
	action string,
	previous *IdentitySessionPolicyEvidence,
	selected *IdentitySessionPolicyEvidence,
	actorUserID int64,
	auditID int64,
	reasonCode string,
	selectionRevision int64,
) (IdentitySessionPolicyEvent, error) {
	previousJSON, err := marshalIdentitySessionPolicyEvidence(previous)
	if err != nil {
		return IdentitySessionPolicyEvent{}, err
	}
	selectedJSON, err := marshalIdentitySessionPolicyEvidence(selected)
	if err != nil {
		return IdentitySessionPolicyEvent{}, err
	}
	return scanIdentitySessionPolicyEvent(tx.QueryRow(ctx, `
		INSERT INTO identity_session_policy_selection_events (
			action, previous_selection, selected_selection, actor_user_id,
			audit_event_id, reason_code, selection_revision
		) VALUES ($1, $2::jsonb, $3::jsonb, $4, $5, $6, $7)
		RETURNING id, action, previous_selection, selected_selection,
		          actor_user_id, audit_event_id, reason_code, selection_revision, created_at
	`, action, previousJSON, selectedJSON, nullableIdentitySessionPolicyActor(actorUserID),
		auditID, reasonCode, selectionRevision))
}

func scanIdentitySessionPolicyEvent(
	scanner identitySessionPolicyScanner,
) (IdentitySessionPolicyEvent, error) {
	var event IdentitySessionPolicyEvent
	var previousJSON, selectedJSON []byte
	var actorUserID *int64
	if err := scanner.Scan(
		&event.ID,
		&event.Action,
		&previousJSON,
		&selectedJSON,
		&actorUserID,
		&event.AuditEventID,
		&event.ReasonCode,
		&event.SelectionRevision,
		&event.CreatedAt,
	); err != nil {
		return IdentitySessionPolicyEvent{}, err
	}
	var err error
	event.PreviousSelection, err = unmarshalIdentitySessionPolicyEvidence(previousJSON)
	if err != nil {
		return IdentitySessionPolicyEvent{}, err
	}
	event.SelectedSelection, err = unmarshalIdentitySessionPolicyEvidence(selectedJSON)
	if err != nil {
		return IdentitySessionPolicyEvent{}, err
	}
	if actorUserID != nil {
		event.ActorUserID = *actorUserID
	}
	if !validIdentitySessionPolicyEvent(event) {
		return IdentitySessionPolicyEvent{}, ErrIdentitySessionPolicyStoreUnavailable
	}
	return event, nil
}

func validIdentitySessionPolicyEvent(event IdentitySessionPolicyEvent) bool {
	if event.ID <= 0 || event.AuditEventID <= 0 || event.SelectionRevision <= 0 || event.CreatedAt.IsZero() ||
		!validIdentitySessionPolicyReason(event.ReasonCode, event.Action == IdentitySessionPolicyActionInvalidate) {
		return false
	}
	switch event.Action {
	case IdentitySessionPolicyActionSelect:
		return event.SelectedSelection != nil &&
			(event.PreviousSelection != nil || event.SelectionRevision == 1)
	case IdentitySessionPolicyActionReset, IdentitySessionPolicyActionInvalidate:
		return event.PreviousSelection != nil && event.SelectedSelection == nil
	default:
		return false
	}
}

func marshalIdentitySessionPolicyEvidence(evidence *IdentitySessionPolicyEvidence) (any, error) {
	if evidence == nil {
		return nil, nil
	}
	if !validIdentitySessionPolicyEvidence(*evidence) {
		return nil, ErrIdentitySessionPolicyInvalid
	}
	body, err := json.Marshal(evidence)
	if err != nil {
		return nil, fmt.Errorf("encode identity session policy evidence: %w", err)
	}
	return string(body), nil
}

func unmarshalIdentitySessionPolicyEvidence(body []byte) (*IdentitySessionPolicyEvidence, error) {
	if len(body) == 0 || bytes.Equal(body, []byte("null")) {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var evidence IdentitySessionPolicyEvidence
	if err := decoder.Decode(&evidence); err != nil {
		return nil, ErrIdentitySessionPolicyStoreUnavailable
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) ||
		!validIdentitySessionPolicyEvidence(evidence) {
		return nil, ErrIdentitySessionPolicyStoreUnavailable
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil ||
		!validIdentitySessionPolicyEvidenceKeys(fields, evidence.PolicyID == IdentitySessionPolicyCoreDefault) {
		return nil, ErrIdentitySessionPolicyStoreUnavailable
	}
	return &evidence, nil
}

func validIdentitySessionPolicyEvidenceKeys(fields map[string]json.RawMessage, core bool) bool {
	expected := map[string]struct{}{"policyId": {}}
	if !core {
		for _, key := range []string{
			"providerContractVersion",
			"ownerExtensionId",
			"ownerExtensionVersionId",
			"ownerExtensionVersion",
			"ownerPackageDigest",
			"declarationRevision",
		} {
			expected[key] = struct{}{}
		}
	}
	if len(fields) != len(expected) {
		return false
	}
	for key := range fields {
		if _, found := expected[key]; !found {
			return false
		}
	}
	return true
}

func identitySessionPolicyEvidenceEqual(
	left *IdentitySessionPolicyEvidence,
	right *IdentitySessionPolicyEvidence,
) bool {
	return reflect.DeepEqual(left, right)
}

func mapIdentitySessionPolicyStoreError(err error) error {
	if err == nil {
		return nil
	}
	var commitUnknown *IdentitySessionPolicyCommitUnknownError
	if errors.As(err, &commitUnknown) {
		return err
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded),
		errors.Is(err, ErrIdentitySessionPolicyInvalid),
		errors.Is(err, ErrIdentitySessionPolicyRevisionConflict),
		errors.Is(err, ErrIdentitySessionPolicyDeclarationStale),
		errors.Is(err, ErrIdentitySessionPolicyPermissionDenied),
		errors.Is(err, ErrIdentitySessionPolicySafeMode),
		errors.Is(err, ErrIdentitySessionPolicyStoreUnavailable),
		errors.Is(err, errIdentitySessionPolicyRetry):
		return err
	}
	var postgresErr *pgconn.PgError
	if !errors.As(err, &postgresErr) {
		return ErrIdentitySessionPolicyStoreUnavailable
	}
	switch postgresErr.Code {
	case "40001", "40P01":
		return errIdentitySessionPolicyRetry
	case "23505":
		switch postgresErr.ConstraintName {
		case "identity_session_policy_selection_pkey",
			"identity_session_policy_selection_events_revision_key":
			return ErrIdentitySessionPolicyRevisionConflict
		default:
			return ErrIdentitySessionPolicyStoreUnavailable
		}
	case "23503", "23514", "22P02":
		return ErrIdentitySessionPolicyInvalid
	default:
		return ErrIdentitySessionPolicyStoreUnavailable
	}
}

func identitySessionPolicyCommitDefinitelyFailed(commitErr error) bool {
	if errors.Is(commitErr, pgx.ErrTxCommitRollback) || pgconn.SafeToRetry(commitErr) {
		return true
	}
	var postgresErr *pgconn.PgError
	if !errors.As(commitErr, &postgresErr) {
		return false
	}
	return strings.HasPrefix(postgresErr.Code, "40") && postgresErr.Code != "40003"
}

func nullableIdentitySessionPolicyActor(actorUserID int64) any {
	if actorUserID <= 0 {
		return nil
	}
	return actorUserID
}

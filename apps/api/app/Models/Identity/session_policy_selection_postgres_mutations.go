package identity

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresIdentitySessionPolicyStore) resolveIdentitySessionPolicyCommit(
	ctx context.Context,
	expected IdentitySessionPolicyMutation,
	commitErr error,
) (IdentitySessionPolicyMutation, error) {
	mapped := mapIdentitySessionPolicyStoreError(commitErr)
	if identitySessionPolicyCommitDefinitelyFailed(commitErr) {
		return IdentitySessionPolicyMutation{}, mapped
	}
	verified, found, verificationErr := s.readbackIdentitySessionPolicyMutation(ctx, expected)
	if verificationErr == nil && found {
		return verified, nil
	}
	if errors.Is(verificationErr, ErrIdentitySessionPolicyRevisionConflict) {
		return IdentitySessionPolicyMutation{}, verificationErr
	}
	if verificationErr == nil {
		verificationErr = errors.New("identity: session policy commit marker was not found")
	}
	return IdentitySessionPolicyMutation{}, &IdentitySessionPolicyCommitUnknownError{
		CommitError:       commitErr,
		VerificationError: verificationErr,
	}
}

func (s *PostgresIdentitySessionPolicyStore) readbackIdentitySessionPolicyMutation(
	ctx context.Context,
	expected IdentitySessionPolicyMutation,
) (IdentitySessionPolicyMutation, bool, error) {
	if expected.Event == nil {
		return IdentitySessionPolicyMutation{}, false, ErrIdentitySessionPolicyStoreUnavailable
	}
	readbackCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		identitySessionPolicyReadbackTimeout,
	)
	defer cancel()
	event, err := scanIdentitySessionPolicyEvent(s.pool.QueryRow(readbackCtx, `
		SELECT id, action, previous_selection, selected_selection,
		       actor_user_id, audit_event_id, reason_code, selection_revision, created_at
		FROM identity_session_policy_selection_events
		WHERE selection_revision = $1
	`, expected.Event.SelectionRevision))
	if errors.Is(err, pgx.ErrNoRows) {
		return IdentitySessionPolicyMutation{}, false, nil
	}
	if err != nil {
		return IdentitySessionPolicyMutation{}, false, err
	}
	if event.ID != expected.Event.ID || event.AuditEventID != expected.Event.AuditEventID ||
		event.Action != expected.Event.Action ||
		event.ReasonCode != expected.Event.ReasonCode ||
		!identitySessionPolicyEvidenceEqual(event.PreviousSelection, expected.Event.PreviousSelection) ||
		!identitySessionPolicyEvidenceEqual(event.SelectedSelection, expected.Event.SelectedSelection) {
		return IdentitySessionPolicyMutation{}, false, ErrIdentitySessionPolicyRevisionConflict
	}
	selection := expected.Selection
	// FK-driven actor deletion may race detached readback. The immutable event
	// is authoritative for retained actor provenance; timestamps remain those
	// returned by the actual singleton INSERT/UPDATE, not the later event row.
	selection.SelectedByUserID = event.ActorUserID
	return IdentitySessionPolicyMutation{Selection: selection, Event: &event, Changed: true}, true, nil
}

func (s *PostgresIdentitySessionPolicyStore) configured() bool {
	return s != nil && s.pool != nil && s.registry != nil && s.effectConnectionSlots != nil
}

func publicIdentitySessionPolicyStoreError(err error) error {
	err = mapIdentitySessionPolicyStoreError(err)
	if errors.Is(err, errIdentitySessionPolicyRetry) {
		return ErrIdentitySessionPolicyRevisionConflict
	}
	return err
}

func writeIdentitySessionPolicySelection(
	ctx context.Context,
	tx pgx.Tx,
	evidence IdentitySessionPolicyEvidence,
	expectedRevision int64,
	actorUserID int64,
	auditID int64,
	present bool,
) (IdentitySessionPolicySelection, error) {
	if !present {
		return scanIdentitySessionPolicySelection(tx.QueryRow(ctx, `
			INSERT INTO identity_session_policy_selection (
				singleton, policy_id, provider_contract_version, owner_extension_id,
				owner_extension_version_id, owner_extension_version, owner_package_digest,
				declaration_revision, revision, selected_by_user_id, selection_audit_event_id
			) VALUES (TRUE, $1, $2, $3, $4, $5, $6, $7, 1, $8, $9)
			RETURNING `+identitySessionPolicySelectionColumns,
			evidence.PolicyID,
			evidence.ProviderContractVersion,
			evidence.OwnerExtensionID,
			evidence.OwnerExtensionVersionID,
			evidence.OwnerExtensionVersion,
			evidence.OwnerPackageDigest,
			evidence.DeclarationRevision,
			actorUserID,
			auditID,
		))
	}
	selection, err := scanIdentitySessionPolicySelection(tx.QueryRow(ctx, `
		UPDATE identity_session_policy_selection
		SET policy_id = $1,
		    provider_contract_version = $2,
		    owner_extension_id = $3,
		    owner_extension_version_id = $4,
		    owner_extension_version = $5,
		    owner_package_digest = $6,
		    declaration_revision = $7,
		    revision = revision + 1,
		    selected_by_user_id = $8,
		    selection_audit_event_id = $9,
		    selected_at = transaction_timestamp(),
		    updated_at = transaction_timestamp()
		WHERE singleton = TRUE AND revision = $10
		RETURNING `+identitySessionPolicySelectionColumns,
		evidence.PolicyID,
		evidence.ProviderContractVersion,
		evidence.OwnerExtensionID,
		evidence.OwnerExtensionVersionID,
		evidence.OwnerExtensionVersion,
		evidence.OwnerPackageDigest,
		evidence.DeclarationRevision,
		actorUserID,
		auditID,
		expectedRevision,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return IdentitySessionPolicySelection{}, ErrIdentitySessionPolicyRevisionConflict
	}
	return selection, err
}

func resetIdentitySessionPolicySelection(
	ctx context.Context,
	tx pgx.Tx,
	expectedRevision int64,
	actorUserID int64,
	auditID int64,
) (IdentitySessionPolicySelection, error) {
	selection, err := scanIdentitySessionPolicySelection(tx.QueryRow(ctx, `
		UPDATE identity_session_policy_selection
		SET policy_id = $1,
		    provider_contract_version = NULL,
		    owner_extension_id = NULL,
		    owner_extension_version_id = NULL,
		    owner_extension_version = NULL,
		    owner_package_digest = NULL,
		    declaration_revision = NULL,
		    revision = revision + 1,
		    selected_by_user_id = $2,
		    selection_audit_event_id = $3,
		    selected_at = transaction_timestamp(),
		    updated_at = transaction_timestamp()
		WHERE singleton = TRUE AND revision = $4
		RETURNING `+identitySessionPolicySelectionColumns,
		IdentitySessionPolicyCoreDefault,
		nullableIdentitySessionPolicyActor(actorUserID),
		auditID,
		expectedRevision,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return IdentitySessionPolicySelection{}, ErrIdentitySessionPolicyRevisionConflict
	}
	return selection, err
}

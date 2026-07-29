package identityregistry

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// decideRoleSuggestionOnce owns one serializable role-suggestion decision.
// The retry policy and ambiguous-commit readback remain with the repository.
func (s *PostgresStore) decideRoleSuggestionOnce(
	ctx context.Context,
	input DecideRoleSuggestionInput,
	approvalState string,
	action string,
) (RoleSuggestion, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return RoleSuggestion{}, mapStoreError(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	// Suggestion is the CAS root. Exact artifact/owner locks then serialize this
	// approval with disable, upgrade, and uninstall publication.
	current, err := lockRoleSuggestion(ctx, tx, input.ID)
	if err != nil {
		return RoleSuggestion{}, err
	}

	// Terminal rows: exact replay returns the durable result; legacy approved
	// without grant evidence may be applied with expected revision 2.
	if current.ApprovalState != RoleSuggestionPending {
		updated, terminalErr := decideTerminalRoleSuggestion(ctx, tx, current, input, approvalState, action)
		if terminalErr != nil {
			return RoleSuggestion{}, terminalErr
		}
		if err := tx.Commit(ctx); err != nil {
			return s.readbackRoleSuggestionDecision(ctx, input, approvalState, err)
		}
		return updated, nil
	}

	if current.Revision != input.ExpectedRevision {
		return RoleSuggestion{}, ErrRevisionConflict
	}
	// Approval consumes live extension authority; rejection only closes a Host
	// recommendation and must remain possible after disable or uninstall.
	if approvalState == RoleSuggestionApproved {
		if err := lockActiveRoleSuggestionArtifact(ctx, tx, current); err != nil {
			return RoleSuggestion{}, err
		}
	}
	// Actor uses KEY SHARE only and is checked before role_permissions writes so
	// concurrent role replacement cannot deadlock on users then mappings.
	if err := lockAndAuthorizeRoleSuggestionActor(ctx, tx, input.ActorUserID); err != nil {
		return RoleSuggestion{}, err
	}

	rolePermissionAdded := false
	roleGrantApplied := false
	var roleID int64
	if approvalState == RoleSuggestionApproved {
		roleID, err = lockRoleSuggestionTarget(ctx, tx, current.RoleKey)
		if err != nil {
			return RoleSuggestion{}, err
		}
		if err := requireRoleSuggestionCatalog(ctx, tx, current); err != nil {
			return RoleSuggestion{}, err
		}
		rolePermissionAdded, err = addRoleSuggestionPermission(ctx, tx, roleID, current.PermissionKey)
		if err != nil {
			return RoleSuggestion{}, err
		}
		roleGrantApplied = true
	}

	auditEventID, err := insertRoleSuggestionAuditEvent(
		ctx, tx, action, input.ActorUserID, current, approvalState,
		input.ExpectedRevision, rolePermissionAdded, roleGrantApplied,
	)
	if err != nil {
		return RoleSuggestion{}, err
	}
	if roleGrantApplied {
		if err := insertRoleSuggestionGrant(
			ctx, tx, current, roleID, input.ActorUserID, auditEventID,
		); err != nil {
			return RoleSuggestion{}, err
		}
	}

	updated, err := casDecideRoleSuggestion(
		ctx,
		tx,
		input.ID,
		input.ExpectedRevision,
		approvalState,
		input.ActorUserID,
		auditEventID,
	)
	if err != nil {
		return RoleSuggestion{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return s.readbackRoleSuggestionDecision(ctx, input, approvalState, err)
	}
	return updated, nil
}

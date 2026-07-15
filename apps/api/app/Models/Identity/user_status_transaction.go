package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

var ErrUserStatusRevisionConflict = errors.New("identity: user status revision conflict")

const RevokeReasonAdminDisabled = "admin_disabled"

type UserStatusMutationInput struct {
	ActorUserID      int64
	TargetUserID     int64
	Status           UserStatus
	ExpectedRevision int64
	Reason           string
}

type UserStatusMutationPlan struct {
	UserID            int64
	PreviousStatus    UserStatus
	Status            UserStatus
	CurrentRevision   int64
	InitialSuperAdmin bool
}

type UserStatusMutationResult struct {
	UserID          int64      `json:"userId"`
	PreviousStatus  UserStatus `json:"previousStatus"`
	Status          UserStatus `json:"status"`
	Revision        int64      `json:"revision"`
	RevokedSessions int64      `json:"revokedSessions"`
}

// PrepareUserStatusTx locks and validates a narrow active/disabled transition
// without writing it. The caller must already have authorized user.manage.
func PrepareUserStatusTx(ctx context.Context, tx pgx.Tx, input UserStatusMutationInput) (UserStatusMutationPlan, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if tx == nil || input.ActorUserID <= 0 || input.TargetUserID <= 0 || input.ActorUserID == input.TargetUserID ||
		(input.Status != UserStatusActive && input.Status != UserStatusDisabled) || input.ExpectedRevision < 0 || len(input.Reason) > 200 {
		if input.ActorUserID > 0 && input.ActorUserID == input.TargetUserID {
			return UserStatusMutationPlan{}, ErrSelfStatusChange
		}
		return UserStatusMutationPlan{}, ErrInvalidUserUpdate
	}
	plan := UserStatusMutationPlan{UserID: input.TargetUserID, Status: input.Status}
	if err := tx.QueryRow(ctx, `
		SELECT status, current_token_version, is_initial_super_admin
		FROM users
		WHERE id = $1
		FOR UPDATE
	`, input.TargetUserID).Scan(&plan.PreviousStatus, &plan.CurrentRevision, &plan.InitialSuperAdmin); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserStatusMutationPlan{}, ErrUserNotFound
		}
		return UserStatusMutationPlan{}, fmt.Errorf("lock Host Command user status: %w", err)
	}
	if plan.CurrentRevision != input.ExpectedRevision {
		return UserStatusMutationPlan{}, ErrUserStatusRevisionConflict
	}
	actorSuperAdmin, err := transactionalUserHasRole(ctx, tx, input.ActorUserID, RoleSuperAdmin)
	if err != nil {
		return UserStatusMutationPlan{}, err
	}
	targetSuperAdmin, err := transactionalUserHasRole(ctx, tx, input.TargetUserID, RoleSuperAdmin)
	if err != nil {
		return UserStatusMutationPlan{}, err
	}
	if targetSuperAdmin && !actorSuperAdmin {
		return UserStatusMutationPlan{}, ErrSuperAdminSessionLocked
	}
	if plan.InitialSuperAdmin && input.Status != UserStatusActive {
		return UserStatusMutationPlan{}, ErrInitialSuperAdminLocked
	}
	return plan, nil
}

// SetUserStatusTx applies the prepared account/session transition inside the
// caller-owned transaction. A no-op transition preserves the current revision.
func SetUserStatusTx(ctx context.Context, tx pgx.Tx, input UserStatusMutationInput) (UserStatusMutationResult, error) {
	plan, err := PrepareUserStatusTx(ctx, tx, input)
	if err != nil {
		return UserStatusMutationResult{}, err
	}
	result := UserStatusMutationResult{
		UserID: plan.UserID, PreviousStatus: plan.PreviousStatus,
		Status: plan.Status, Revision: plan.CurrentRevision,
	}
	if plan.PreviousStatus == plan.Status {
		return result, nil
	}
	if err := tx.QueryRow(ctx, `
		UPDATE users
		SET status = $2, current_token_version = current_token_version + 1, updated_at = transaction_timestamp()
		WHERE id = $1 AND current_token_version = $3
		RETURNING current_token_version
	`, input.TargetUserID, input.Status, input.ExpectedRevision).Scan(&result.Revision); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserStatusMutationResult{}, ErrUserStatusRevisionConflict
		}
		return UserStatusMutationResult{}, fmt.Errorf("update Host Command user status: %w", err)
	}
	if input.Status == UserStatusDisabled {
		reason := strings.TrimSpace(input.Reason)
		if reason == "" {
			reason = RevokeReasonAdminDisabled
		}
		tag, err := tx.Exec(ctx, `
			UPDATE user_sessions
			SET revoked_at = transaction_timestamp(), revoke_reason = $2
			WHERE user_id = $1 AND revoked_at IS NULL
		`, input.TargetUserID, reason)
		if err != nil {
			return UserStatusMutationResult{}, fmt.Errorf("revoke disabled user sessions: %w", err)
		}
		result.RevokedSessions = tag.RowsAffected()
	}
	return result, nil
}

func transactionalUserHasRole(ctx context.Context, tx pgx.Tx, userID int64, roleKey string) (bool, error) {
	rows, err := tx.Query(ctx, `
		SELECT roles.key
		FROM user_roles
		JOIN roles ON roles.id = user_roles.role_id
		WHERE user_roles.user_id = $1 AND roles.key = $2 AND roles.is_enabled = TRUE
		FOR SHARE OF user_roles, roles
	`, userID, roleKey)
	if err != nil {
		return false, fmt.Errorf("load transactional user role: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return false, fmt.Errorf("iterate transactional user role: %w", err)
		}
		return false, nil
	}
	var found string
	if err := rows.Scan(&found); err != nil {
		return false, fmt.Errorf("scan transactional user role: %w", err)
	}
	return found == roleKey, nil
}

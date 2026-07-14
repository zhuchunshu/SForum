package hostapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
)

func (b *PostgresProtocolV2CommandBackend) AuthorizeActorDelegation(
	ctx context.Context,
	tx pgx.Tx,
	scope protocolV2CommandScope,
	delegation protocolV2VerifiedActorDelegation,
	fingerprint string,
	receiptExists bool,
	requiredPermissions []string,
) (int64, error) {
	if b == nil || tx == nil || !validResolvedProtocolV2CommandScope(scope) ||
		delegation.ActorUserID <= 0 || !protocolV2SHA256Hex(delegation.DelegationIDDigest) ||
		delegation.RuntimeEpoch <= 0 || delegation.RuntimeInstanceID == "" || scope.RuntimeEpoch <= 0 || scope.RuntimeInstanceID == "" ||
		delegation.RuntimeEpoch != scope.RuntimeEpoch || delegation.RuntimeInstanceID != scope.RuntimeInstanceID ||
		!protocolV2SHA256Hex(fingerprint) {
		return 0, invalidProtocolV2CommandActorDelegation()
	}
	actor, err := loadProtocolV2CommandActor(ctx, tx, delegation.ActorUserID)
	if err != nil {
		return 0, err
	}
	for _, permission := range requiredPermissions {
		if !actor.Can(permission) {
			return 0, deniedProtocolV2CommandActorPermission()
		}
	}

	var existingActorID int64
	var existingFingerprint string
	err = tx.QueryRow(ctx, `
		SELECT actor_user_id, request_fingerprint
		FROM extension_host_command_actor_delegation_consumptions
		WHERE extension_id = $1 AND command_id = $2
		  AND command_version = $3 AND idempotency_key = $4
		FOR UPDATE
	`, scope.ExtensionID, scope.CommandID, scope.CommandVersion, scope.IdempotencyKey).Scan(
		&existingActorID, &existingFingerprint,
	)
	switch {
	case err == nil:
		if !receiptExists || existingActorID != delegation.ActorUserID || existingFingerprint != fingerprint {
			return 0, replayedProtocolV2CommandActorDelegation()
		}
		return actor.ID, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return 0, fmt.Errorf("load Host Command actor delegation consumption: %w", err)
	case receiptExists:
		return 0, missingProtocolV2CommandActorDelegationEvidence()
	}

	commandTag, err := tx.Exec(ctx, `
		INSERT INTO extension_host_command_actor_delegation_consumptions (
			delegation_id_digest,
			extension_id, extension_version_id, extension_version, package_digest,
			runtime_epoch, runtime_instance_id, actor_user_id,
			command_id, command_version, audience, idempotency_key, request_fingerprint,
			issued_at, not_before, expires_at, consumed_at
		)
		SELECT
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, statement_timestamp()
		WHERE $15 <= statement_timestamp() AND $16 >= statement_timestamp()
	`, delegation.DelegationIDDigest,
		scope.ExtensionID, scope.ExtensionVersionID, scope.ExtensionVersion, scope.PackageDigest,
		scope.RuntimeEpoch, scope.RuntimeInstanceID, actor.ID,
		scope.CommandID, scope.CommandVersion, ProtocolV2ActorDelegationAudience,
		scope.IdempotencyKey, fingerprint,
		delegation.IssuedAt, delegation.NotBefore, delegation.ExpiresAt,
	)
	if err != nil {
		var postgresErr *pgconn.PgError
		if errors.As(err, &postgresErr) && postgresErr.Code == "23505" {
			return 0, replayedProtocolV2CommandActorDelegation()
		}
		return 0, fmt.Errorf("consume Host Command actor delegation: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return 0, invalidProtocolV2CommandActorDelegation()
	}
	return actor.ID, nil
}

func loadProtocolV2CommandActor(ctx context.Context, tx pgx.Tx, userID int64) (identity.Actor, error) {
	if ctx == nil || tx == nil || userID <= 0 {
		return identity.Actor{}, inactiveProtocolV2CommandActor()
	}
	actor := identity.Actor{ID: userID, Permissions: make(map[string]bool)}
	if err := tx.QueryRow(ctx, `
		SELECT status
		FROM users
		WHERE id = $1
		FOR SHARE
	`, userID).Scan(&actor.Status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.Actor{}, inactiveProtocolV2CommandActor()
		}
		return identity.Actor{}, fmt.Errorf("load Host Command actor: %w", err)
	}
	if !actor.IsActive() {
		return identity.Actor{}, inactiveProtocolV2CommandActor()
	}

	roleRows, err := tx.Query(ctx, `
		SELECT roles.key
		FROM user_roles
		JOIN roles ON roles.id = user_roles.role_id
		WHERE user_roles.user_id = $1 AND roles.is_enabled = TRUE
		ORDER BY roles.key
		FOR SHARE OF user_roles, roles
	`, userID)
	if err != nil {
		return identity.Actor{}, fmt.Errorf("load Host Command actor roles: %w", err)
	}
	for roleRows.Next() {
		var roleKey string
		if err := roleRows.Scan(&roleKey); err != nil {
			roleRows.Close()
			return identity.Actor{}, fmt.Errorf("scan Host Command actor role: %w", err)
		}
		actor.RoleKeys = append(actor.RoleKeys, roleKey)
	}
	if err := roleRows.Err(); err != nil {
		roleRows.Close()
		return identity.Actor{}, fmt.Errorf("iterate Host Command actor roles: %w", err)
	}
	roleRows.Close()

	permissionRows, err := tx.Query(ctx, `
		SELECT permissions.key
		FROM user_roles
		JOIN roles ON roles.id = user_roles.role_id
		JOIN role_permissions ON role_permissions.role_id = roles.id
		JOIN permissions ON permissions.key = role_permissions.permission_key
		WHERE user_roles.user_id = $1 AND roles.is_enabled = TRUE
		FOR SHARE OF user_roles, roles, role_permissions, permissions
	`, userID)
	if err != nil {
		return identity.Actor{}, fmt.Errorf("load Host Command actor permissions: %w", err)
	}
	for permissionRows.Next() {
		var permission string
		if err := permissionRows.Scan(&permission); err != nil {
			permissionRows.Close()
			return identity.Actor{}, fmt.Errorf("scan Host Command actor permission: %w", err)
		}
		actor.Permissions[permission] = true
	}
	if err := permissionRows.Err(); err != nil {
		permissionRows.Close()
		return identity.Actor{}, fmt.Errorf("iterate Host Command actor permissions: %w", err)
	}
	permissionRows.Close()

	overrideRows, err := tx.Query(ctx, `
		SELECT permission_key, effect
		FROM user_permission_overrides
		WHERE user_id = $1
		FOR SHARE
	`, userID)
	if err != nil {
		return identity.Actor{}, fmt.Errorf("load Host Command actor permission overrides: %w", err)
	}
	for overrideRows.Next() {
		var permission, effect string
		if err := overrideRows.Scan(&permission, &effect); err != nil {
			overrideRows.Close()
			return identity.Actor{}, fmt.Errorf("scan Host Command actor permission override: %w", err)
		}
		if effect == "allow" {
			actor.Permissions[permission] = true
		} else if effect == "deny" {
			delete(actor.Permissions, permission)
		}
	}
	if err := overrideRows.Err(); err != nil {
		overrideRows.Close()
		return identity.Actor{}, fmt.Errorf("iterate Host Command actor permission overrides: %w", err)
	}
	overrideRows.Close()
	return actor, nil
}

func invalidProtocolV2CommandActorDelegation() error {
	return newProtocolV2CommandError(
		protocolv2.ErrorCode_ERROR_CODE_UNAUTHENTICATED,
		"host.command_actor_delegation_invalid",
		"The Host-signed actor delegation is invalid or stale.",
		false,
	)
}

func replayedProtocolV2CommandActorDelegation() error {
	return newProtocolV2CommandError(
		protocolv2.ErrorCode_ERROR_CODE_CONFLICT,
		"host.command_actor_delegation_replayed",
		"The Host-signed actor delegation was already used for another command request.",
		false,
	)
}

func missingProtocolV2CommandActorDelegationEvidence() error {
	return newProtocolV2CommandError(
		protocolv2.ErrorCode_ERROR_CODE_FAILED_PRECONDITION,
		"host.command_actor_delegation_evidence_missing",
		"The committed command has no matching actor delegation evidence.",
		false,
	)
}

func inactiveProtocolV2CommandActor() error {
	return newProtocolV2CommandError(
		protocolv2.ErrorCode_ERROR_CODE_PERMISSION_DENIED,
		"host.command_actor_inactive",
		"The delegated actor is missing or inactive.",
		false,
	)
}

func deniedProtocolV2CommandActorPermission() error {
	return newProtocolV2CommandError(
		protocolv2.ErrorCode_ERROR_CODE_PERMISSION_DENIED,
		"host.command_actor_permission_denied",
		"The delegated actor no longer has permission to execute this Host Command.",
		false,
	)
}

var _ protocolV2CommandBackend = (*PostgresProtocolV2CommandBackend)(nil)

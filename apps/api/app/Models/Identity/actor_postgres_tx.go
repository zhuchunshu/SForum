package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var ErrActorInactive = errors.New("identity: actor is missing or inactive")

// LoadEffectiveActorTx resolves current Host RBAC inside the caller's
// transaction. Consumers must not authorize from a session or preflight copy
// when the protected effect is committed later in this transaction.
func LoadEffectiveActorTx(ctx context.Context, tx pgx.Tx, userID int64) (Actor, error) {
	if ctx == nil || tx == nil || userID <= 0 {
		return Actor{}, ErrActorInactive
	}
	actor := Actor{ID: userID, Permissions: make(map[string]bool)}
	if err := tx.QueryRow(ctx, `
		SELECT status
		FROM users
		WHERE id = $1
		FOR SHARE
	`, userID).Scan(&actor.Status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Actor{}, ErrActorInactive
		}
		return Actor{}, fmt.Errorf("load effective actor: %w", err)
	}
	if !actor.IsActive() {
		return Actor{}, ErrActorInactive
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
		return Actor{}, fmt.Errorf("load effective actor roles: %w", err)
	}
	for roleRows.Next() {
		var roleKey string
		if err := roleRows.Scan(&roleKey); err != nil {
			roleRows.Close()
			return Actor{}, fmt.Errorf("scan effective actor role: %w", err)
		}
		actor.RoleKeys = append(actor.RoleKeys, roleKey)
	}
	if err := roleRows.Err(); err != nil {
		roleRows.Close()
		return Actor{}, fmt.Errorf("iterate effective actor roles: %w", err)
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
		return Actor{}, fmt.Errorf("load effective actor permissions: %w", err)
	}
	for permissionRows.Next() {
		var permission string
		if err := permissionRows.Scan(&permission); err != nil {
			permissionRows.Close()
			return Actor{}, fmt.Errorf("scan effective actor permission: %w", err)
		}
		actor.Permissions[permission] = true
	}
	if err := permissionRows.Err(); err != nil {
		permissionRows.Close()
		return Actor{}, fmt.Errorf("iterate effective actor permissions: %w", err)
	}
	permissionRows.Close()

	overrideRows, err := tx.Query(ctx, `
		SELECT permission_key, effect
		FROM user_permission_overrides
		WHERE user_id = $1
		FOR SHARE
	`, userID)
	if err != nil {
		return Actor{}, fmt.Errorf("load effective actor permission overrides: %w", err)
	}
	for overrideRows.Next() {
		var permission, effect string
		if err := overrideRows.Scan(&permission, &effect); err != nil {
			overrideRows.Close()
			return Actor{}, fmt.Errorf("scan effective actor permission override: %w", err)
		}
		if effect == "allow" {
			actor.Permissions[permission] = true
		} else if effect == "deny" {
			delete(actor.Permissions, permission)
		}
	}
	if err := overrideRows.Err(); err != nil {
		overrideRows.Close()
		return Actor{}, fmt.Errorf("iterate effective actor permission overrides: %w", err)
	}
	overrideRows.Close()
	return actor, nil
}

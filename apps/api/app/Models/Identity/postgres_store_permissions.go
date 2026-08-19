package identity

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ListActiveUserIDsWithPermissionTx resolves exact Host RBAC recipients inside
// the caller's transaction. super_admin remains authoritative even when an
// ordinary deny override exists, matching Actor.Can.
func (s *PostgresStore) ListActiveUserIDsWithPermissionTx(ctx context.Context, tx pgx.Tx, permission string) ([]int64, error) {
	if tx == nil || permission == "" {
		return nil, fmt.Errorf("identity permission recipient transaction is required")
	}
	rows, err := tx.Query(ctx, `
		SELECT users.id
		FROM users
		WHERE users.status = 'active'
		  AND (
		    EXISTS (
		      SELECT 1
		      FROM user_roles
		      JOIN roles ON roles.id = user_roles.role_id
		      WHERE user_roles.user_id = users.id
		        AND roles.is_enabled = TRUE
		        AND roles.key = 'super_admin'
		    )
		    OR (
		      (
		        EXISTS (
		          SELECT 1
		          FROM user_roles
		          JOIN roles ON roles.id = user_roles.role_id
		          JOIN role_permissions ON role_permissions.role_id = roles.id
		          WHERE user_roles.user_id = users.id
		            AND roles.is_enabled = TRUE
		            AND role_permissions.permission_key = $1
		        )
		        OR EXISTS (
		          SELECT 1 FROM user_permission_overrides
		          WHERE user_id = users.id AND permission_key = $1 AND effect = 'allow'
		        )
		      )
		      AND NOT EXISTS (
		        SELECT 1 FROM user_permission_overrides
		        WHERE user_id = users.id AND permission_key = $1 AND effect = 'deny'
		      )
		    )
		  )
		ORDER BY users.id
	`, permission)
	if err != nil {
		return nil, fmt.Errorf("list active users with permission %s: %w", permission, err)
	}
	defer rows.Close()
	result := []int64{}
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scan permission recipient: %w", err)
		}
		result = append(result, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate permission recipients: %w", err)
	}
	return result, nil
}

func (s *PostgresStore) ListPermissions(ctx context.Context) ([]Permission, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT permission.key,
		       permission.module,
		       permission.label,
		       permission.description,
		       permission.label_locales,
		       permission.description_locales
		FROM permissions AS permission
		LEFT JOIN extension_permission_catalog AS extension_catalog
		  ON extension_catalog.permission_key = permission.key
		LEFT JOIN LATERAL (
			SELECT declaration.registry_state
			FROM extension_identity_registry_declarations AS declaration
			WHERE declaration.identity_kind = 'permission'
			  AND declaration.stable_id = extension_catalog.permission_key
			ORDER BY declaration.revision DESC
			LIMIT 1
		) AS extension_tip ON TRUE
		WHERE extension_catalog.permission_key IS NULL
		   OR extension_tip.registry_state = 'active'
		ORDER BY permission.module ASC, permission.key ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	defer rows.Close()

	permissions := []Permission{}
	for rows.Next() {
		var permission Permission
		var labelLocales, descriptionLocales []byte
		if err := rows.Scan(
			&permission.Key, &permission.Module, &permission.Label, &permission.Description,
			&labelLocales, &descriptionLocales,
		); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		if err := json.Unmarshal(labelLocales, &permission.LabelLocales); err != nil {
			return nil, fmt.Errorf("decode permission label locales: %w", err)
		}
		if err := json.Unmarshal(descriptionLocales, &permission.DescriptionLocales); err != nil {
			return nil, fmt.Errorf("decode permission description locales: %w", err)
		}
		permissions = append(permissions, permission)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate permissions: %w", err)
	}
	return permissions, nil
}

func (s *PostgresStore) ListPermissionMatrix(ctx context.Context) (PermissionMatrix, error) {
	permissions, err := s.ListPermissions(ctx)
	if err != nil {
		return PermissionMatrix{}, err
	}
	roles, err := s.ListRoles(ctx)
	if err != nil {
		return PermissionMatrix{}, err
	}

	matrix := PermissionMatrix{
		Permissions: permissions,
		Roles:       make([]RolePermissionSet, 0, len(roles)),
	}
	for _, role := range roles {
		matrix.Roles = append(matrix.Roles, RolePermissionSet{
			RoleKey:        role.Key,
			PermissionKeys: role.PermissionKeys,
		})
	}
	return matrix, nil
}

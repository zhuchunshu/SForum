package identity

import (
	"context"
	"encoding/json"
	"fmt"
)

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

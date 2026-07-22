package migrator

import (
	"context"
	"database/sql"
	"fmt"
)

const coreDatabaseExtensionsSchema = "sforum_host_extensions"

// ensureCoreDatabaseExtensions 必须使用 migration login，而不是受限 Core owner。
// PostgreSQL extension 是数据库级依赖，不属于 Core schema 的对象所有权边界。
func ensureCoreDatabaseExtensions(ctx context.Context, adminConnection *sql.Conn) error {
	if adminConnection == nil {
		return fmt.Errorf("migration administrator connection is required")
	}
	if _, err := adminConnection.ExecContext(ctx, `
		CREATE SCHEMA IF NOT EXISTS sforum_host_extensions AUTHORIZATION CURRENT_USER;
		REVOKE CREATE ON SCHEMA sforum_host_extensions FROM PUBLIC;
		GRANT USAGE ON SCHEMA sforum_host_extensions TO PUBLIC;
		CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA sforum_host_extensions
	`); err != nil {
		return fmt.Errorf("create trusted pg_trgm extension: %w", err)
	}
	var extensionSchema string
	if err := adminConnection.QueryRowContext(ctx, `
		SELECT namespaces.nspname
		FROM pg_extension AS extensions
		JOIN pg_namespace AS namespaces ON namespaces.oid = extensions.extnamespace
		WHERE extensions.extname = 'pg_trgm'
	`).Scan(&extensionSchema); err != nil {
		return fmt.Errorf("inspect pg_trgm extension schema: %w", err)
	}
	if extensionSchema != coreDatabaseExtensionsSchema {
		if _, err := adminConnection.ExecContext(ctx, `ALTER EXTENSION pg_trgm SET SCHEMA sforum_host_extensions`); err != nil {
			return fmt.Errorf("move pg_trgm to Host extension schema: %w", err)
		}
	}
	return nil
}

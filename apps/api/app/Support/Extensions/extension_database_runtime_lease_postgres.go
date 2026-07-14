package extensionsruntime

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

const extensionDatabaseCoreViewSchema = "sforum_core_v1"

func createExtensionDatabaseRuntimeLeaseRole(
	ctx context.Context,
	tx pgx.Tx,
	identifiers ExtensionDatabaseIdentifiers,
	roleName string,
	databaseName string,
	powers []string,
	password string,
	expiresAt time.Time,
) (string, error) {
	if !identifiers.valid() || !validPostgresIdentifier(roleName) ||
		!validPostgresCatalogName(databaseName) || !extensionDatabasePasswordPattern.MatchString(password) ||
		expiresAt.IsZero() || len(powers) == 0 {
		return "", ErrExtensionDatabaseRegistryInvalid
	}
	var roleExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, roleName).Scan(&roleExists); err != nil {
		return "", fmt.Errorf("inspect extension database runtime lease role: %w", err)
	}
	if roleExists {
		return "", ErrExtensionDatabaseRuntimeLeaseConflict
	}
	role := pgx.Identifier{roleName}.Sanitize()
	database := pgx.Identifier{databaseName}.Sanitize()
	owner := pgx.Identifier{identifiers.OwnerRole}.Sanitize()
	if _, err := tx.Exec(ctx, `CREATE ROLE `+role+` NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`); err != nil {
		return "", fmt.Errorf("%w: create runtime lease role: %v", ErrExtensionDatabaseCredential, err)
	}

	searchPath := extensionDatabaseRuntimeLeaseSearchPath(identifiers.Schema, powers)
	queries := []string{`GRANT CONNECT ON DATABASE ` + database + ` TO ` + role}
	if containsExtensionDatabasePower(powers, extensionmanifest.DatabaseGrantOwnSchema) {
		schema := pgx.Identifier{identifiers.Schema}.Sanitize()
		queries = append(queries,
			`GRANT `+owner+` TO `+role,
			`ALTER DEFAULT PRIVILEGES FOR ROLE `+role+` IN SCHEMA `+schema+` GRANT ALL ON TABLES TO `+owner,
			`ALTER DEFAULT PRIVILEGES FOR ROLE `+role+` IN SCHEMA `+schema+` GRANT ALL ON SEQUENCES TO `+owner,
			`ALTER DEFAULT PRIVILEGES FOR ROLE `+role+` IN SCHEMA `+schema+` GRANT ALL ON FUNCTIONS TO `+owner,
			`ALTER DEFAULT PRIVILEGES FOR ROLE `+role+` IN SCHEMA `+schema+` GRANT USAGE ON TYPES TO `+owner,
		)
	}
	if containsExtensionDatabasePower(powers, extensionmanifest.DatabaseGrantCoreViews) {
		coreViews := pgx.Identifier{extensionDatabaseCoreViewSchema}.Sanitize()
		queries = append(queries,
			`GRANT USAGE ON SCHEMA `+coreViews+` TO `+role,
			`GRANT SELECT ON ALL TABLES IN SCHEMA `+coreViews+` TO `+role,
		)
	}
	if containsExtensionDatabasePower(powers, extensionmanifest.DatabaseGrantRawCore) ||
		containsExtensionDatabasePower(powers, extensionmanifest.DatabaseGrantKernel) {
		publicSchema := pgx.Identifier{"public"}.Sanitize()
		queries = append(queries,
			`GRANT USAGE ON SCHEMA `+publicSchema+` TO `+role,
			`GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA `+publicSchema+` TO `+role,
			`GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA `+publicSchema+` TO `+role,
			`GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA `+publicSchema+` TO `+role,
		)
	}
	queries = append(queries,
		`ALTER ROLE `+role+` LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS CONNECTION LIMIT `+
			strconv.Itoa(extensionDatabaseRuntimeLeaseConnectionLimit)+` VALID UNTIL '`+expiresAt.UTC().Format(time.RFC3339Nano)+`' PASSWORD '`+password+`'`,
		`ALTER ROLE `+role+` IN DATABASE `+database+` SET search_path TO `+searchPath,
		`ALTER ROLE `+role+` IN DATABASE `+database+` SET statement_timeout TO '`+extensionDatabaseRuntimeStatementTimeout+`'`,
		`ALTER ROLE `+role+` IN DATABASE `+database+` SET idle_in_transaction_session_timeout TO '`+extensionDatabaseRuntimeIdleTransactionTimeout+`'`,
	)
	for _, query := range queries {
		if _, err := tx.Exec(ctx, query); err != nil {
			return "", fmt.Errorf("%w: configure runtime lease role: %v", ErrExtensionDatabaseCredential, err)
		}
	}
	if err := validateExtensionDatabaseRuntimeLeaseRole(
		ctx, tx, identifiers, roleName, databaseName, powers, searchPath, expiresAt,
	); err != nil {
		return "", err
	}
	return strings.ReplaceAll(searchPath, `"`, ""), nil
}

func validateExtensionDatabaseRuntimeLeaseRole(
	ctx context.Context,
	tx pgx.Tx,
	identifiers ExtensionDatabaseIdentifiers,
	roleName string,
	databaseName string,
	powers []string,
	searchPath string,
	expiresAt time.Time,
) error {
	var canLogin, inherit, superuser, createDatabase, createRole, replication, bypassRLS bool
	var connectionLimit int
	var validUntil *time.Time
	if err := tx.QueryRow(ctx, `
		SELECT rolcanlogin, rolinherit, rolsuper, rolcreatedb, rolcreaterole,
		       rolreplication, rolbypassrls, rolconnlimit, rolvaliduntil
		FROM pg_roles WHERE rolname = $1
	`, roleName).Scan(
		&canLogin, &inherit, &superuser, &createDatabase, &createRole,
		&replication, &bypassRLS, &connectionLimit, &validUntil,
	); err != nil {
		return fmt.Errorf("inspect runtime lease role: %w", err)
	}
	if !canLogin || !inherit || superuser || createDatabase || createRole || replication || bypassRLS ||
		connectionLimit != extensionDatabaseRuntimeLeaseConnectionLimit || validUntil == nil ||
		!validUntil.UTC().Equal(expiresAt.UTC()) {
		return ErrExtensionDatabaseResourceConflict
	}

	expectedMembership := ""
	if containsExtensionDatabasePower(powers, extensionmanifest.DatabaseGrantOwnSchema) {
		expectedMembership = identifiers.OwnerRole
	}
	rows, err := tx.Query(ctx, `
		SELECT granted.rolname
		FROM pg_auth_members AS memberships
		JOIN pg_roles AS member ON member.oid = memberships.member
		JOIN pg_roles AS granted ON granted.oid = memberships.roleid
		WHERE member.rolname = $1
	`, roleName)
	if err != nil {
		return fmt.Errorf("inspect runtime lease memberships: %w", err)
	}
	defer rows.Close()
	memberships := 0
	for rows.Next() {
		var granted string
		if err := rows.Scan(&granted); err != nil {
			return err
		}
		memberships++
		if expectedMembership == "" || granted != expectedMembership {
			return ErrExtensionDatabaseResourceConflict
		}
	}
	if err := rows.Err(); err != nil || (expectedMembership != "" && memberships != 1) {
		if err != nil {
			return err
		}
		return ErrExtensionDatabaseResourceConflict
	}

	expectedSettings := map[string]struct{}{
		normalizeExtensionDatabaseLeaseSetting("search_path=" + searchPath):                     {},
		"statement_timeout=" + extensionDatabaseRuntimeStatementTimeout:                         {},
		"idle_in_transaction_session_timeout=" + extensionDatabaseRuntimeIdleTransactionTimeout: {},
	}
	settings, err := tx.Query(ctx, `
		SELECT setting
		FROM pg_db_role_setting AS role_settings
		JOIN pg_database AS databases ON databases.oid = role_settings.setdatabase
		CROSS JOIN LATERAL unnest(role_settings.setconfig) AS setting
		WHERE role_settings.setrole = (SELECT oid FROM pg_roles WHERE rolname = $1)
		  AND databases.datname = $2
	`, roleName, databaseName)
	if err != nil {
		return fmt.Errorf("inspect runtime lease settings: %w", err)
	}
	defer settings.Close()
	seen := map[string]struct{}{}
	for settings.Next() {
		var setting string
		if err := settings.Scan(&setting); err != nil {
			return err
		}
		normalized := normalizeExtensionDatabaseLeaseSetting(setting)
		if _, ok := expectedSettings[normalized]; !ok {
			return ErrExtensionDatabaseResourceConflict
		}
		seen[normalized] = struct{}{}
	}
	if err := settings.Err(); err != nil {
		return err
	}
	if len(seen) != len(expectedSettings) {
		return ErrExtensionDatabaseResourceConflict
	}
	return nil
}

func normalizeExtensionDatabaseLeaseSetting(value string) string {
	return strings.NewReplacer(" ", "", `"`, "").Replace(value)
}

func revokeExtensionDatabaseRuntimeLeaseRole(
	ctx context.Context,
	tx pgx.Tx,
	roleName string,
	ownerRoleName string,
	databaseName string,
) error {
	if !validPostgresIdentifier(roleName) || !validPostgresIdentifier(ownerRoleName) ||
		!validPostgresCatalogName(databaseName) {
		return ErrExtensionDatabaseRegistryInvalid
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, roleName).Scan(&exists); err != nil {
		return fmt.Errorf("inspect revoked runtime lease role: %w", err)
	}
	if !exists {
		return nil
	}
	role := pgx.Identifier{roleName}.Sanitize()
	owner := pgx.Identifier{ownerRoleName}.Sanitize()
	database := pgx.Identifier{databaseName}.Sanitize()
	queries := []string{
		`ALTER ROLE ` + role + ` NOLOGIN PASSWORD NULL VALID UNTIL 'epoch'`,
		`REVOKE CONNECT ON DATABASE ` + database + ` FROM ` + role,
	}
	for _, query := range queries {
		if _, err := tx.Exec(ctx, query); err != nil {
			return fmt.Errorf("%w: disable runtime lease role: %v", ErrExtensionDatabaseCredential, err)
		}
	}
	if _, err := tx.Exec(ctx, `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE usename = $1 AND datname = $2 AND pid <> pg_backend_pid()
	`, roleName, databaseName); err != nil {
		return fmt.Errorf("%w: terminate runtime lease sessions: %v", ErrExtensionDatabaseCredential, err)
	}
	for _, query := range []string{
		`REASSIGN OWNED BY ` + role + ` TO ` + owner,
		`DROP OWNED BY ` + role,
		`DROP ROLE ` + role,
	} {
		if _, err := tx.Exec(ctx, query); err != nil {
			return fmt.Errorf("%w: retire runtime lease role: %v", ErrExtensionDatabaseCredential, err)
		}
	}
	return nil
}

func extendExtensionDatabaseRuntimeLeaseRole(
	ctx context.Context,
	tx pgx.Tx,
	roleName string,
	expiresAt time.Time,
) error {
	if !validPostgresIdentifier(roleName) || expiresAt.IsZero() {
		return ErrExtensionDatabaseRegistryInvalid
	}
	role := pgx.Identifier{roleName}.Sanitize()
	if _, err := tx.Exec(ctx, `ALTER ROLE `+role+` VALID UNTIL '`+expiresAt.UTC().Format(time.RFC3339Nano)+`'`); err != nil {
		return fmt.Errorf("%w: extend runtime lease role: %v", ErrExtensionDatabaseCredential, err)
	}
	var validUntil *time.Time
	if err := tx.QueryRow(ctx, `SELECT rolvaliduntil FROM pg_roles WHERE rolname = $1`, roleName).Scan(&validUntil); err != nil {
		return fmt.Errorf("inspect extended runtime lease role: %w", err)
	}
	if validUntil == nil || !validUntil.UTC().Equal(expiresAt.UTC()) {
		return ErrExtensionDatabaseResourceConflict
	}
	return nil
}

func extensionDatabaseRuntimeLeaseSearchPath(schemaName string, powers []string) string {
	parts := make([]string, 0, 4)
	if containsExtensionDatabasePower(powers, extensionmanifest.DatabaseGrantOwnSchema) {
		parts = append(parts, pgx.Identifier{schemaName}.Sanitize())
	}
	if containsExtensionDatabasePower(powers, extensionmanifest.DatabaseGrantCoreViews) {
		parts = append(parts, pgx.Identifier{extensionDatabaseCoreViewSchema}.Sanitize())
	}
	if containsExtensionDatabasePower(powers, extensionmanifest.DatabaseGrantRawCore) ||
		containsExtensionDatabasePower(powers, extensionmanifest.DatabaseGrantKernel) {
		parts = append(parts, pgx.Identifier{"public"}.Sanitize())
	}
	parts = append(parts, "pg_catalog")
	return strings.Join(parts, ", ")
}

func containsExtensionDatabasePower(powers []string, expected string) bool {
	for _, power := range powers {
		if power == expected {
			return true
		}
	}
	return false
}

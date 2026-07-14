package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func ensureExtensionDatabaseResources(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
	identifiers ExtensionDatabaseIdentifiers,
) (string, error) {
	databaseName, err := currentExtensionDatabaseName(ctx, tx)
	if err != nil {
		return "", err
	}
	if err := ensureExtensionDatabaseRole(ctx, tx, identifiers.OwnerRole, false, ""); err != nil {
		return "", err
	}
	if err := ensureExtensionDatabaseRole(ctx, tx, identifiers.RuntimeRole, true, identifiers.OwnerRole); err != nil {
		return "", err
	}
	if err := ensureExtensionDatabaseSchema(ctx, tx, identifiers.Schema, identifiers.OwnerRole); err != nil {
		return "", err
	}
	if err := validateExtensionDatabaseRoleIsolation(
		ctx, tx, identifiers.OwnerRole, identifiers.Schema, "", databaseName, false,
	); err != nil {
		return "", err
	}
	if err := validateExtensionDatabaseRoleIsolation(
		ctx, tx, identifiers.RuntimeRole, identifiers.Schema,
		identifiers.OwnerRole, databaseName, true,
	); err != nil {
		return "", err
	}
	if err := configureExtensionDatabaseOwnership(ctx, tx, identifiers); err != nil {
		return "", err
	}
	if err := validateExtensionDatabaseRoleIsolation(
		ctx, tx, identifiers.RuntimeRole, identifiers.Schema,
		identifiers.OwnerRole, databaseName, true,
	); err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO extension_database_resources (
			extension_id, schema_name, owner_role_name, runtime_role_name
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (extension_id) DO NOTHING
	`, extensionID, identifiers.Schema, identifiers.OwnerRole, identifiers.RuntimeRole)
	if err != nil {
		return "", fmt.Errorf("insert extension database resources: %w", err)
	}
	resource, err := loadExtensionDatabaseResource(ctx, tx, extensionID, true)
	if err != nil {
		return "", err
	}
	if !resource.matches(identifiers) {
		return "", ErrExtensionDatabaseResourceConflict
	}
	return databaseName, nil
}

func currentExtensionDatabaseName(
	ctx context.Context,
	querier interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
) (string, error) {
	var databaseName string
	if err := querier.QueryRow(ctx, `SELECT current_database()`).Scan(&databaseName); err != nil {
		return "", fmt.Errorf("read extension database name: %w", err)
	}
	if !validPostgresCatalogName(databaseName) {
		return "", ErrExtensionDatabaseResourceConflict
	}
	return databaseName, nil
}

func ensureExtensionDatabaseRole(
	ctx context.Context,
	tx pgx.Tx,
	roleName string,
	allowLogin bool,
	allowedMembership string,
) error {
	if !validPostgresIdentifier(roleName) {
		return ErrExtensionDatabaseIdentifier
	}
	var canLogin, superuser, createDatabase, createRole, replication, bypassRLS bool
	err := tx.QueryRow(ctx, `
		SELECT rolcanlogin, rolsuper, rolcreatedb, rolcreaterole, rolreplication, rolbypassrls
		FROM pg_roles WHERE rolname = $1
	`, roleName).Scan(&canLogin, &superuser, &createDatabase, &createRole, &replication, &bypassRLS)
	if errors.Is(err, pgx.ErrNoRows) {
		quoted := pgx.Identifier{roleName}.Sanitize()
		if _, err := tx.Exec(ctx, `CREATE ROLE `+quoted+` NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`); err != nil {
			return fmt.Errorf("create extension database role %s: %w", roleName, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect extension database role %s: %w", roleName, err)
	}
	if superuser || createDatabase || createRole || replication || bypassRLS || (!allowLogin && canLogin) {
		return fmt.Errorf("%w: role %s has elevated attributes", ErrExtensionDatabaseResourceConflict, roleName)
	}
	rows, err := tx.Query(ctx, `
		SELECT granted.rolname
		FROM pg_auth_members AS memberships
		JOIN pg_roles AS member ON member.oid = memberships.member
		JOIN pg_roles AS granted ON granted.oid = memberships.roleid
		WHERE member.rolname = $1
		ORDER BY granted.rolname
	`, roleName)
	if err != nil {
		return fmt.Errorf("inspect extension database role memberships: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var grantedRole string
		if err := rows.Scan(&grantedRole); err != nil {
			return err
		}
		if allowedMembership == "" || grantedRole != allowedMembership {
			return fmt.Errorf(
				"%w: role %s inherits unexpected role %s",
				ErrExtensionDatabaseResourceConflict, roleName, grantedRole,
			)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

func ensureExtensionDatabaseSchema(
	ctx context.Context,
	tx pgx.Tx,
	schemaName string,
	ownerRoleName string,
) error {
	if !validPostgresIdentifier(schemaName) || !validPostgresIdentifier(ownerRoleName) {
		return ErrExtensionDatabaseIdentifier
	}
	var existingOwner string
	err := tx.QueryRow(ctx, `
		SELECT roles.rolname
		FROM pg_namespace
		JOIN pg_roles AS roles ON roles.oid = pg_namespace.nspowner
		WHERE pg_namespace.nspname = $1
	`, schemaName).Scan(&existingOwner)
	if errors.Is(err, pgx.ErrNoRows) {
		query := `CREATE SCHEMA ` + pgx.Identifier{schemaName}.Sanitize() +
			` AUTHORIZATION ` + pgx.Identifier{ownerRoleName}.Sanitize()
		if _, err := tx.Exec(ctx, query); err != nil {
			return fmt.Errorf("create extension database schema %s: %w", schemaName, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect extension database schema %s: %w", schemaName, err)
	}
	if existingOwner != ownerRoleName {
		return ErrExtensionDatabaseResourceConflict
	}
	return nil
}

func validateExtensionDatabaseRoleIsolation(
	ctx context.Context,
	tx pgx.Tx,
	roleName string,
	schemaName string,
	allowedMembership string,
	databaseName string,
	allowExactSearchPath bool,
) error {
	if !validPostgresIdentifier(roleName) || !validPostgresIdentifier(schemaName) ||
		!validPostgresCatalogName(databaseName) {
		return ErrExtensionDatabaseIdentifier
	}
	rows, err := tx.Query(ctx, `
		SELECT settings.setdatabase, COALESCE(databases.datname, ''), setting
		FROM pg_db_role_setting AS settings
		LEFT JOIN pg_database AS databases ON databases.oid = settings.setdatabase
		CROSS JOIN LATERAL unnest(settings.setconfig) AS setting
		WHERE settings.setrole = (SELECT oid FROM pg_roles WHERE rolname = $1)
		ORDER BY settings.setdatabase, setting
	`, roleName)
	if err != nil {
		return fmt.Errorf("inspect extension database role settings: %w", err)
	}
	expectedSearchPath := "search_path=" + schemaName + ",pg_catalog"
	seenSearchPath := false
	for rows.Next() {
		var databaseOID uint32
		var configuredDatabase, setting string
		if err := rows.Scan(&databaseOID, &configuredDatabase, &setting); err != nil {
			rows.Close()
			return err
		}
		normalized := strings.ReplaceAll(setting, " ", "")
		if !allowExactSearchPath || seenSearchPath || databaseOID == 0 ||
			configuredDatabase != databaseName || normalized != expectedSearchPath {
			rows.Close()
			return fmt.Errorf(
				"%w: role %s has unexpected setting %q for database %q",
				ErrExtensionDatabaseResourceConflict, roleName, setting, configuredDatabase,
			)
		}
		seenSearchPath = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	var ownershipConflict bool
	err = tx.QueryRow(ctx, `
		WITH role_identity AS (
			SELECT oid FROM pg_roles WHERE rolname = $1
		)
		SELECT
		  EXISTS (SELECT 1 FROM pg_database, role_identity WHERE datdba = role_identity.oid)
		  OR EXISTS (
			SELECT 1 FROM pg_namespace, role_identity
			WHERE nspowner = role_identity.oid AND nspname <> $2
		  )
		  OR EXISTS (
			SELECT 1 FROM pg_class
			JOIN pg_namespace ON pg_namespace.oid = pg_class.relnamespace
			CROSS JOIN role_identity
			WHERE pg_class.relowner = role_identity.oid
			  AND pg_namespace.nspname <> $2
			  AND pg_namespace.nspname <> 'pg_toast'
			  AND pg_namespace.nspname NOT LIKE 'pg_temp_%'
			  AND pg_namespace.nspname NOT LIKE 'pg_toast_temp_%'
		  )
		  OR EXISTS (
			SELECT 1 FROM pg_proc
			JOIN pg_namespace ON pg_namespace.oid = pg_proc.pronamespace
			CROSS JOIN role_identity
			WHERE pg_proc.proowner = role_identity.oid AND pg_namespace.nspname <> $2
		  )
		  OR EXISTS (
			SELECT 1 FROM pg_type
			JOIN pg_namespace ON pg_namespace.oid = pg_type.typnamespace
			CROSS JOIN role_identity
			WHERE pg_type.typowner = role_identity.oid
			  AND pg_namespace.nspname <> $2
			  AND pg_namespace.nspname <> 'pg_toast'
			  AND pg_namespace.nspname NOT LIKE 'pg_temp_%'
			  AND pg_namespace.nspname NOT LIKE 'pg_toast_temp_%'
		  )
		  OR EXISTS (SELECT 1 FROM pg_extension, role_identity WHERE extowner = role_identity.oid)
		  OR EXISTS (SELECT 1 FROM pg_tablespace, role_identity WHERE spcowner = role_identity.oid)
	`, roleName, schemaName).Scan(&ownershipConflict)
	if err != nil {
		return fmt.Errorf("inspect extension database role ownership: %w", err)
	}
	if ownershipConflict {
		return fmt.Errorf(
			"%w: role %s owns objects outside schema %s",
			ErrExtensionDatabaseResourceConflict, roleName, schemaName,
		)
	}

	membershipRows, err := tx.Query(ctx, `
		SELECT granted.rolname
		FROM pg_auth_members AS memberships
		JOIN pg_roles AS member ON member.oid = memberships.member
		JOIN pg_roles AS granted ON granted.oid = memberships.roleid
		WHERE member.rolname = $1
	`, roleName)
	if err != nil {
		return fmt.Errorf("verify extension database role memberships: %w", err)
	}
	defer membershipRows.Close()
	for membershipRows.Next() {
		var granted string
		if err := membershipRows.Scan(&granted); err != nil {
			return err
		}
		if allowedMembership == "" || granted != allowedMembership {
			return fmt.Errorf(
				"%w: role %s inherits unexpected role %s",
				ErrExtensionDatabaseResourceConflict, roleName, granted,
			)
		}
	}
	return membershipRows.Err()
}

func configureExtensionDatabaseOwnership(
	ctx context.Context,
	tx pgx.Tx,
	identifiers ExtensionDatabaseIdentifiers,
) error {
	owner := pgx.Identifier{identifiers.OwnerRole}.Sanitize()
	runtime := pgx.Identifier{identifiers.RuntimeRole}.Sanitize()
	schema := pgx.Identifier{identifiers.Schema}.Sanitize()
	queries := []string{
		`ALTER ROLE ` + owner + ` NOLOGIN`,
		`ALTER ROLE ` + runtime + ` INHERIT`,
		`GRANT ` + owner + ` TO ` + runtime,
		`REVOKE ALL ON SCHEMA ` + schema + ` FROM PUBLIC`,
		`GRANT USAGE, CREATE ON SCHEMA ` + schema + ` TO ` + owner,
		`ALTER DEFAULT PRIVILEGES FOR ROLE ` + owner + ` IN SCHEMA ` + schema + ` REVOKE ALL ON TABLES FROM PUBLIC`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE ` + owner + ` IN SCHEMA ` + schema + ` REVOKE ALL ON SEQUENCES FROM PUBLIC`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE ` + owner + ` IN SCHEMA ` + schema + ` REVOKE ALL ON FUNCTIONS FROM PUBLIC`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE ` + owner + ` IN SCHEMA ` + schema + ` REVOKE ALL ON TYPES FROM PUBLIC`,
	}
	for _, query := range queries {
		if _, err := tx.Exec(ctx, query); err != nil {
			return fmt.Errorf("configure extension database ownership: %w", err)
		}
	}
	return nil
}

func activateExtensionDatabaseRuntimeRole(
	ctx context.Context,
	tx pgx.Tx,
	identifiers ExtensionDatabaseIdentifiers,
	databaseName string,
	password string,
) error {
	if !identifiers.valid() || !validPostgresCatalogName(databaseName) ||
		!extensionDatabasePasswordPattern.MatchString(password) {
		return ErrExtensionDatabaseRegistryInvalid
	}
	runtime := pgx.Identifier{identifiers.RuntimeRole}.Sanitize()
	database := pgx.Identifier{databaseName}.Sanitize()
	schema := pgx.Identifier{identifiers.Schema}.Sanitize()
	queries := []string{
		`ALTER ROLE ` + runtime + ` LOGIN PASSWORD '` + password + `'`,
		`GRANT CONNECT ON DATABASE ` + database + ` TO ` + runtime,
		`ALTER ROLE ` + runtime + ` IN DATABASE ` + database + ` SET search_path TO ` + schema + `, pg_catalog`,
	}
	for _, query := range queries {
		if _, err := tx.Exec(ctx, query); err != nil {
			return fmt.Errorf("%w: activate runtime role: %v", ErrExtensionDatabaseCredential, err)
		}
	}
	return nil
}

func disableExtensionDatabaseRuntimeRole(
	ctx context.Context,
	tx pgx.Tx,
	roleName string,
	databaseName string,
) error {
	if !validPostgresIdentifier(roleName) || !validPostgresCatalogName(databaseName) {
		return ErrExtensionDatabaseRegistryInvalid
	}
	role := pgx.Identifier{roleName}.Sanitize()
	database := pgx.Identifier{databaseName}.Sanitize()
	if _, err := tx.Exec(ctx, `ALTER ROLE `+role+` NOLOGIN PASSWORD NULL`); err != nil {
		return fmt.Errorf("%w: disable runtime role: %v", ErrExtensionDatabaseCredential, err)
	}
	if _, err := tx.Exec(ctx, `REVOKE CONNECT ON DATABASE `+database+` FROM `+role); err != nil {
		return fmt.Errorf("%w: revoke database connect: %v", ErrExtensionDatabaseCredential, err)
	}
	if _, err := tx.Exec(ctx, `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE usename = $1 AND datname = $2 AND pid <> pg_backend_pid()
	`, roleName, databaseName); err != nil {
		return fmt.Errorf("%w: terminate revoked role sessions: %v", ErrExtensionDatabaseCredential, err)
	}
	return nil
}

package extensionsruntime

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type extensionDatabaseKernelRelation struct {
	schema string
	name   string
	kind   string
}

type extensionDatabaseKernelRoutine struct {
	schema    string
	name      string
	arguments string
	kind      string
}

type extensionDatabaseKernelType struct {
	schema string
	name   string
}

func reassignExtensionDatabaseKernelCoreObjects(ctx context.Context, tx pgx.Tx, roleName string) error {
	if !validPostgresIdentifier(roleName) {
		return ErrExtensionDatabaseRegistryInvalid
	}
	identity, err := loadExtensionDatabaseCoreIdentity(ctx, tx)
	if err != nil {
		return err
	}
	owner := pgx.Identifier{identity.ownerRole}.Sanitize()

	relations, err := loadExtensionDatabaseKernelRelations(ctx, tx, roleName)
	if err != nil {
		return err
	}
	for _, relation := range relations {
		object := pgx.Identifier{relation.schema, relation.name}.Sanitize()
		command := "TABLE"
		switch relation.kind {
		case "S":
			command = "SEQUENCE"
		case "v":
			command = "VIEW"
		case "m":
			command = "MATERIALIZED VIEW"
		case "f":
			command = "FOREIGN TABLE"
		case "c":
			command = "TYPE"
		}
		if _, err := tx.Exec(ctx, `ALTER `+command+` `+object+` OWNER TO `+owner); err != nil {
			return fmt.Errorf("transfer kernel Core relation %s.%s: %w", relation.schema, relation.name, err)
		}
	}

	routines, err := loadExtensionDatabaseKernelRoutines(ctx, tx, roleName)
	if err != nil {
		return err
	}
	for _, routine := range routines {
		object := pgx.Identifier{routine.schema, routine.name}.Sanitize()
		command := "FUNCTION"
		if routine.kind == "p" {
			command = "PROCEDURE"
		} else if routine.kind == "a" {
			command = "AGGREGATE"
		}
		if _, err := tx.Exec(ctx, `ALTER `+command+` `+object+`(`+routine.arguments+`) OWNER TO `+owner); err != nil {
			return fmt.Errorf("transfer kernel Core routine %s.%s: %w", routine.schema, routine.name, err)
		}
	}

	types, err := loadExtensionDatabaseKernelTypes(ctx, tx, roleName)
	if err != nil {
		return err
	}
	for _, coreType := range types {
		object := pgx.Identifier{coreType.schema, coreType.name}.Sanitize()
		if _, err := tx.Exec(ctx, `ALTER TYPE `+object+` OWNER TO `+owner); err != nil {
			return fmt.Errorf("transfer kernel Core type %s.%s: %w", coreType.schema, coreType.name, err)
		}
	}

	var remaining int
	if err := tx.QueryRow(ctx, `
		SELECT
		  (SELECT count(*)
		   FROM pg_class AS classes
		   JOIN pg_namespace AS namespaces ON namespaces.oid = classes.relnamespace
		   WHERE namespaces.nspname IN ($1, $2)
		     AND classes.relowner = (SELECT oid FROM pg_roles WHERE rolname = $3))
		  + (SELECT count(*)
		     FROM pg_proc AS routines
		     JOIN pg_namespace AS namespaces ON namespaces.oid = routines.pronamespace
		     WHERE namespaces.nspname IN ($1, $2)
		       AND routines.proowner = (SELECT oid FROM pg_roles WHERE rolname = $3))
		  + (SELECT count(*)
		     FROM pg_type AS types
		     JOIN pg_namespace AS namespaces ON namespaces.oid = types.typnamespace
		     WHERE namespaces.nspname IN ($1, $2)
		       AND types.typowner = (SELECT oid FROM pg_roles WHERE rolname = $3))
	`, extensionDatabaseCoreSchema, extensionDatabaseCoreViewSchema, roleName).Scan(&remaining); err != nil {
		return fmt.Errorf("verify kernel Core ownership transfer: %w", err)
	}
	if remaining != 0 {
		return ErrExtensionDatabaseResourceConflict
	}
	var unsupportedCoreObject bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_shdepend AS dependencies
			CROSS JOIN LATERAL pg_identify_object_as_address(
			  dependencies.classid, dependencies.objid, dependencies.objsubid
			) AS identified
			WHERE dependencies.refclassid = 'pg_authid'::regclass
			  AND dependencies.refobjid = (SELECT oid FROM pg_roles WHERE rolname = $3)
			  AND dependencies.deptype = 'o'
			  AND dependencies.dbid = (SELECT oid FROM pg_database WHERE datname = current_database())
			  AND identified.object_names[1] IN ($1, $2)
		)
	`, extensionDatabaseCoreSchema, extensionDatabaseCoreViewSchema, roleName).Scan(&unsupportedCoreObject); err != nil {
		return fmt.Errorf("inspect unsupported kernel Core ownership: %w", err)
	}
	if unsupportedCoreObject {
		return ErrExtensionDatabaseResourceConflict
	}
	return nil
}

// Core 对象转移后只允许 own_schema 对象和随后由 DROP OWNED 清除的默认 ACL 留在租约名下。
func validateExtensionDatabaseKernelRemainingOwnership(
	ctx context.Context,
	tx pgx.Tx,
	roleName string,
	schemaName string,
	allowOwnSchema bool,
) error {
	if !validPostgresIdentifier(roleName) || !validPostgresIdentifier(schemaName) {
		return ErrExtensionDatabaseRegistryInvalid
	}
	rows, err := tx.Query(ctx, `
		SELECT dependencies.classid::regclass::TEXT,
		       identified.object_names, default_namespaces.nspname
		FROM pg_shdepend AS dependencies
		CROSS JOIN LATERAL pg_identify_object_as_address(
		  dependencies.classid, dependencies.objid, dependencies.objsubid
		) AS identified
		LEFT JOIN pg_default_acl AS defaults
		  ON dependencies.classid = 'pg_default_acl'::regclass
		 AND defaults.oid = dependencies.objid
		LEFT JOIN pg_namespace AS default_namespaces
		  ON default_namespaces.oid = defaults.defaclnamespace
		WHERE dependencies.refclassid = 'pg_authid'::regclass
		  AND dependencies.refobjid = (SELECT oid FROM pg_roles WHERE rolname = $1)
		  AND dependencies.deptype = 'o'
		  AND dependencies.dbid = (SELECT oid FROM pg_database WHERE datname = current_database())
		ORDER BY dependencies.classid, dependencies.objid, dependencies.objsubid
	`, roleName)
	if err != nil {
		return fmt.Errorf("inspect remaining kernel lease ownership: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var catalog string
		var objectNames []string
		var defaultSchema *string
		if err := rows.Scan(&catalog, &objectNames, &defaultSchema); err != nil {
			return err
		}
		if catalog == "pg_default_acl" && defaultSchema == nil {
			continue
		}
		if allowOwnSchema && catalog == "pg_default_acl" && defaultSchema != nil && *defaultSchema == schemaName {
			continue
		}
		if allowOwnSchema && len(objectNames) > 0 && objectNames[0] == schemaName {
			continue
		}
		return ErrExtensionDatabaseResourceConflict
	}
	return rows.Err()
}

func loadExtensionDatabaseKernelRelations(
	ctx context.Context,
	tx pgx.Tx,
	roleName string,
) ([]extensionDatabaseKernelRelation, error) {
	rows, err := tx.Query(ctx, `
		SELECT namespaces.nspname, classes.relname, classes.relkind::TEXT
		FROM pg_class AS classes
		JOIN pg_namespace AS namespaces ON namespaces.oid = classes.relnamespace
		WHERE namespaces.nspname IN ($1, $2)
		  AND classes.relkind IN ('r', 'p', 'S', 'v', 'm', 'f', 'c')
		  AND NOT (namespaces.nspname = $1 AND classes.relname ~ '^_?river_')
		  AND classes.relowner = (SELECT oid FROM pg_roles WHERE rolname = $3)
		  AND NOT (
		    classes.relkind = 'S' AND EXISTS (
		      SELECT 1 FROM pg_depend AS dependencies
		      WHERE dependencies.classid = 'pg_class'::regclass
		        AND dependencies.objid = classes.oid
		        AND dependencies.refclassid = 'pg_class'::regclass
		        AND dependencies.deptype IN ('a', 'i')
		    )
		  )
		ORDER BY classes.relkind, namespaces.nspname, classes.relname
	`, extensionDatabaseCoreSchema, extensionDatabaseCoreViewSchema, roleName)
	if err != nil {
		return nil, fmt.Errorf("load kernel Core relations: %w", err)
	}
	defer rows.Close()
	var relations []extensionDatabaseKernelRelation
	for rows.Next() {
		var relation extensionDatabaseKernelRelation
		if err := rows.Scan(&relation.schema, &relation.name, &relation.kind); err != nil {
			return nil, err
		}
		relations = append(relations, relation)
	}
	return relations, rows.Err()
}

func loadExtensionDatabaseKernelRoutines(
	ctx context.Context,
	tx pgx.Tx,
	roleName string,
) ([]extensionDatabaseKernelRoutine, error) {
	rows, err := tx.Query(ctx, `
		SELECT namespaces.nspname, routines.proname,
		       pg_get_function_identity_arguments(routines.oid), routines.prokind::TEXT
		FROM pg_proc AS routines
		JOIN pg_namespace AS namespaces ON namespaces.oid = routines.pronamespace
		WHERE namespaces.nspname IN ($1, $2)
		  AND NOT (namespaces.nspname = $1 AND routines.proname ~ '^_?river_')
		  AND routines.proowner = (SELECT oid FROM pg_roles WHERE rolname = $3)
		ORDER BY namespaces.nspname, routines.proname, routines.oid
	`, extensionDatabaseCoreSchema, extensionDatabaseCoreViewSchema, roleName)
	if err != nil {
		return nil, fmt.Errorf("load kernel Core routines: %w", err)
	}
	defer rows.Close()
	var routines []extensionDatabaseKernelRoutine
	for rows.Next() {
		var routine extensionDatabaseKernelRoutine
		if err := rows.Scan(&routine.schema, &routine.name, &routine.arguments, &routine.kind); err != nil {
			return nil, err
		}
		routines = append(routines, routine)
	}
	return routines, rows.Err()
}

func loadExtensionDatabaseKernelTypes(
	ctx context.Context,
	tx pgx.Tx,
	roleName string,
) ([]extensionDatabaseKernelType, error) {
	rows, err := tx.Query(ctx, `
		SELECT namespaces.nspname, types.typname
		FROM pg_type AS types
		JOIN pg_namespace AS namespaces ON namespaces.oid = types.typnamespace
		WHERE namespaces.nspname IN ($1, $2)
		  AND types.typtype IN ('d', 'e', 'r', 'm')
		  AND NOT (namespaces.nspname = $1 AND types.typname ~ '^_{0,2}river_')
		  AND types.typowner = (SELECT oid FROM pg_roles WHERE rolname = $3)
		ORDER BY namespaces.nspname, types.typname
	`, extensionDatabaseCoreSchema, extensionDatabaseCoreViewSchema, roleName)
	if err != nil {
		return nil, fmt.Errorf("load kernel Core types: %w", err)
	}
	defer rows.Close()
	var types []extensionDatabaseKernelType
	for rows.Next() {
		var coreType extensionDatabaseKernelType
		if err := rows.Scan(&coreType.schema, &coreType.name); err != nil {
			return nil, err
		}
		types = append(types, coreType)
	}
	return types, rows.Err()
}

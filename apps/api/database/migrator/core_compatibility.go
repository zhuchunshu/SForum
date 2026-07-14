package migrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	semver "github.com/Masterminds/semver/v3"
)

var ErrCoreUpgradeIncompatible = errors.New("core upgrade is incompatible with an enabled raw database extension")

type coreCompatibilityBlocker struct {
	ExtensionID      string
	ExtensionVersion string
	Authority        string
	Constraint       string
}

func checkCoreUpgradeCompatibility(ctx context.Context, db *sql.DB, targetVersion string) error {
	target, err := semver.StrictNewVersion(strings.TrimSpace(targetVersion))
	if err != nil {
		return fmt.Errorf("invalid target SForum version %q: %w", targetVersion, err)
	}
	ready, err := coreCompatibilityFactsReady(ctx, db)
	if err != nil {
		return fmt.Errorf("inspect core compatibility schema: %w", err)
	}
	if !ready {
		// Databases predating executable V3 trust cannot contain a live raw-core
		// grant, so there is no compatibility declaration to enforce yet.
		return nil
	}

	rows, err := db.QueryContext(ctx, `
		SELECT extensions.id, extension_versions.version,
		       extension_versions.manifest #>> '{database,authority}',
		       COALESCE(extension_versions.manifest #>> '{database,coreCompatibility}', '')
		FROM extensions
		JOIN extension_versions ON extension_versions.id = extensions.active_version_id
		WHERE extensions.type = 'plugin'
		  AND extensions.status = 'enabled'
		  AND extension_versions.manifest #>> '{database,authority}' IN ('raw_core', 'kernel')
		  AND (
		    (extensions.source = 'builtin' AND extensions.is_system)
		    OR (
		      extensions.source = 'uploaded' AND NOT extensions.is_system
		      AND EXISTS (
		        SELECT 1 FROM extension_trust_grants
		        WHERE extension_trust_grants.extension_id = extensions.id
		          AND extension_trust_grants.extension_version = extension_versions.version
		          AND extension_trust_grants.package_digest = extension_versions.package_digest
		          AND extension_trust_grants.action = 'enable'
		          AND extension_trust_grants.revoked_at IS NULL
		          AND extension_trust_grants.impact_document #>> '{database,authority}' =
		              extension_versions.manifest #>> '{database,authority}'
		      )
		    )
		  )
		ORDER BY extensions.id
	`)
	if err != nil {
		return fmt.Errorf("load raw database compatibility declarations: %w", err)
	}
	defer rows.Close()

	blockers := make([]coreCompatibilityBlocker, 0)
	for rows.Next() {
		var blocker coreCompatibilityBlocker
		if err := rows.Scan(&blocker.ExtensionID, &blocker.ExtensionVersion, &blocker.Authority, &blocker.Constraint); err != nil {
			return fmt.Errorf("scan raw database compatibility declaration: %w", err)
		}
		constraint, constraintErr := semver.NewConstraint(blocker.Constraint)
		if constraintErr != nil || !constraint.Check(target) {
			blockers = append(blockers, blocker)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read raw database compatibility declarations: %w", err)
	}
	if len(blockers) == 0 {
		return nil
	}
	first := blockers[0]
	return fmt.Errorf(
		"%w: target=%s extension=%s version=%s authority=%s constraint=%q blockers=%d",
		ErrCoreUpgradeIncompatible, target.String(), first.ExtensionID, first.ExtensionVersion,
		first.Authority, first.Constraint, len(blockers),
	)
}

func coreCompatibilityFactsReady(ctx context.Context, db *sql.DB) (bool, error) {
	if db == nil {
		return false, errors.New("database is required")
	}
	var ready bool
	err := db.QueryRowContext(ctx, `
		SELECT to_regclass('public.extensions') IS NOT NULL
		   AND to_regclass('public.extension_versions') IS NOT NULL
		   AND to_regclass('public.extension_trust_grants') IS NOT NULL
		   AND EXISTS (
		     SELECT 1 FROM information_schema.columns
		     WHERE table_schema = 'public' AND table_name = 'extensions'
		       AND column_name = 'active_version_id'
		   )
	`).Scan(&ready)
	return ready, err
}

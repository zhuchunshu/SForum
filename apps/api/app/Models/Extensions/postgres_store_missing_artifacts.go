package extensions

import (
	"context"
	"fmt"
)

func (s *PostgresStore) CleanupMissingArtifacts(
	ctx context.Context,
	actorUserID int64,
	items []MissingArtifactCleanupItem,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin missing artifact cleanup: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, item := range items {
		var extensionType, status, source, version, digest, packagePath string
		var isSystem, isDeletable bool
		if err := tx.QueryRow(ctx, `
			SELECT e.type, e.status, e.source, e.is_system, e.is_deletable,
			       v.version, v.package_digest, v.package_path
			FROM extensions AS e
			JOIN extension_versions AS v ON v.id = e.active_version_id
			LEFT JOIN extension_missing_artifact_removals AS removed
			  ON removed.extension_id = e.id
			WHERE e.id = $1 AND removed.extension_id IS NULL
			FOR UPDATE OF e, v
		`, item.ExtensionID).Scan(
			&extensionType, &status, &source, &isSystem, &isDeletable,
			&version, &digest, &packagePath,
		); err != nil {
			return fmt.Errorf("%w: lock %s: %v", ErrMissingArtifactCleanupInvalid, item.ExtensionID, err)
		}
		if extensionType != item.Type || status == StatusEnabled || source == SourceBuiltin ||
			isSystem || !isDeletable || version != item.Version ||
			digest != item.PackageDigest || packagePath != item.PackagePath {
			return fmt.Errorf("%w: stale extension %s", ErrMissingArtifactCleanupInvalid, item.ExtensionID)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO extension_missing_artifact_removals (
			  extension_id, extension_type, extension_version, package_digest,
			  package_path, data_mode, requested_by_user_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, item.ExtensionID, item.Type, item.Version, item.PackageDigest,
			item.PackagePath, item.DataMode, actorUserID,
		); err != nil {
			return fmt.Errorf("record missing artifact cleanup for %s: %w", item.ExtensionID, err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM mail_provider_selection WHERE extension_id = $1`, item.ExtensionID); err != nil {
			return fmt.Errorf("clear provider selection for %s: %w", item.ExtensionID, err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM page_provider_bindings WHERE extension_id = $1`, item.ExtensionID); err != nil {
			return fmt.Errorf("clear page bindings for %s: %w", item.ExtensionID, err)
		}
		if item.DataMode == MissingArtifactDataDiscardSettings {
			if _, err := tx.Exec(ctx, `DELETE FROM extension_settings WHERE extension_id = $1`, item.ExtensionID); err != nil {
				return fmt.Errorf("delete host settings for %s: %w", item.ExtensionID, err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit missing artifact cleanup: %w", err)
	}
	return nil
}

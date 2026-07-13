package extensions

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// PromoteStagedVersion atomically publishes the exact staged artifact as the
// active version. The immutable version row remains retained for rollback.
func (s *PostgresStore) PromoteStagedVersion(ctx context.Context, input StagedVersionCASInput) (Extension, error) {
	if err := validateStagedVersionCASInput(input); err != nil {
		return Extension{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Extension{}, fmt.Errorf("begin staged version promotion: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := lockExactStagedVersion(ctx, tx, input); err != nil {
		return Extension{}, err
	}
	command, err := tx.Exec(ctx, `
		UPDATE extensions
		SET active_version_id = staged_version_id,
		    staged_version_id = NULL,
		    updated_at = now()
		WHERE id = $1
		  AND staged_version_id = $2
		  AND EXISTS (
		    SELECT 1
		    FROM extension_versions
		    WHERE extension_versions.id = $2
		      AND extension_versions.extension_id = $1
		      AND extension_versions.package_digest = $3
		  )
	`, input.ExtensionID, input.ExpectedStagedVersionID, input.ExpectedPackageDigest)
	if err != nil {
		return Extension{}, fmt.Errorf("promote staged extension version: %w", err)
	}
	if command.RowsAffected() != 1 {
		return Extension{}, ErrStagedVersionConflict
	}
	promoted, err := getExtensionInTransaction(ctx, tx, input.ExtensionID)
	if err != nil {
		return Extension{}, err
	}
	if promoted.ActiveVersionID != input.ExpectedStagedVersionID || promoted.PackageDigest != input.ExpectedPackageDigest || promoted.StagedVersion != nil {
		return Extension{}, ErrStagedVersionConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return Extension{}, fmt.Errorf("commit staged version promotion: %w", err)
	}
	return promoted, nil
}

// DiscardStagedVersion clears only the exact candidate pointer. It never
// deletes the immutable extension_versions row.
func (s *PostgresStore) DiscardStagedVersion(ctx context.Context, input StagedVersionCASInput) (Extension, error) {
	if err := validateStagedVersionCASInput(input); err != nil {
		return Extension{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Extension{}, fmt.Errorf("begin staged version discard: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := lockExactStagedVersion(ctx, tx, input); err != nil {
		return Extension{}, err
	}
	command, err := tx.Exec(ctx, `
		UPDATE extensions
		SET staged_version_id = NULL,
		    updated_at = now()
		WHERE id = $1
		  AND staged_version_id = $2
		  AND EXISTS (
		    SELECT 1
		    FROM extension_versions
		    WHERE extension_versions.id = $2
		      AND extension_versions.extension_id = $1
		      AND extension_versions.package_digest = $3
		  )
	`, input.ExtensionID, input.ExpectedStagedVersionID, input.ExpectedPackageDigest)
	if err != nil {
		return Extension{}, fmt.Errorf("discard staged extension version: %w", err)
	}
	if command.RowsAffected() != 1 {
		return Extension{}, ErrStagedVersionConflict
	}
	current, err := getExtensionInTransaction(ctx, tx, input.ExtensionID)
	if err != nil {
		return Extension{}, err
	}
	if current.StagedVersion != nil {
		return Extension{}, ErrStagedVersionConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return Extension{}, fmt.Errorf("commit staged version discard: %w", err)
	}
	return current, nil
}

func lockExactStagedVersion(ctx context.Context, tx pgx.Tx, input StagedVersionCASInput) error {
	var stagedVersionID int64
	var stagedExtensionID, packageDigest string
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(extensions.staged_version_id, 0),
		       COALESCE(extension_versions.extension_id, ''),
		       COALESCE(extension_versions.package_digest, '')
		FROM extensions
		LEFT JOIN extension_versions
		  ON extension_versions.id = extensions.staged_version_id
		WHERE extensions.id = $1
		FOR UPDATE OF extensions
	`, input.ExtensionID).Scan(&stagedVersionID, &stagedExtensionID, &packageDigest)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrExtensionNotFound
	}
	if err != nil {
		return fmt.Errorf("lock staged extension version: %w", err)
	}
	if stagedVersionID == 0 {
		return ErrStagedVersionNotFound
	}
	if stagedVersionID != input.ExpectedStagedVersionID || stagedExtensionID != input.ExtensionID || packageDigest != input.ExpectedPackageDigest {
		return ErrStagedVersionConflict
	}
	return nil
}

func getExtensionInTransaction(ctx context.Context, tx pgx.Tx, extensionID string) (Extension, error) {
	item, err := scanExtension(tx.QueryRow(ctx, extensionSelectSQL()+`
		WHERE extensions.id = $1
	`, extensionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Extension{}, ErrExtensionNotFound
	}
	if err != nil {
		return Extension{}, fmt.Errorf("read staged extension version result: %w", err)
	}
	return item, nil
}

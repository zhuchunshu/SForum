package extensions

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// LastSuccessfulLifecycleAuthority returns an immutable exact-artifact grant
// snapshot. It intentionally ignores current trust-row revocation state so an
// operator can still disable or rollback code that was previously approved.
func (r *PostgresLifecycleRepository) LastSuccessfulLifecycleAuthority(
	ctx context.Context,
	input ExactExtensionVersionInput,
) (LifecycleAuthoritySnapshot, error) {
	if r == nil || r.pool == nil {
		return LifecycleAuthoritySnapshot{}, ErrLifecycleCoordinatorUnavailable
	}
	if err := validateExactExtensionVersionInput(input); err != nil {
		return LifecycleAuthoritySnapshot{}, err
	}
	var document []byte
	err := r.pool.QueryRow(ctx, `
		SELECT authority_snapshot
		FROM extension_lifecycle_operations
		WHERE extension_id = $1
		  AND extension_version = $2
		  AND package_digest = $3
		  AND terminal_result = $4
		  AND completed_at IS NOT NULL
		ORDER BY completed_at DESC, id DESC
		LIMIT 1
	`, input.ExtensionID, input.Version, input.PackageDigest, LifecycleTerminalSucceeded).Scan(&document)
	if errors.Is(err, pgx.ErrNoRows) {
		return LifecycleAuthoritySnapshot{}, ErrLifecycleAuthorityNotFound
	}
	if err != nil {
		return LifecycleAuthoritySnapshot{}, fmt.Errorf("load successful lifecycle authority: %w", err)
	}
	return decodeLifecycleAuthoritySnapshot(document)
}

var _ LifecycleAuthorityRepository = (*PostgresLifecycleRepository)(nil)

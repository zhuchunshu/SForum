package extensionsruntime

import (
	"context"
	"fmt"
)

const (
	DefaultExtensionDatabaseRuntimeLeaseReapLimit = 64
	extensionDatabaseRuntimeLeaseExpiredCode      = "lease_expired"
	extensionDatabaseRuntimeLeaseExpiredAudit     = "extension.database_runtime_lease.expired"
)

// ReapExpiredRuntimeLeases retires physical login roles left behind when a
// Host process exits before it can revoke an exact runtime lease.
func (r *PostgresExtensionDatabaseRegistry) ReapExpiredRuntimeLeases(
	ctx context.Context,
	limit int,
) (int, error) {
	if r == nil || r.pool == nil || ctx == nil || limit <= 0 || limit > 1000 {
		return 0, ErrExtensionDatabaseRegistryInvalid
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	extensionIDs, err := listExpiredExtensionDatabaseRuntimeLeaseExtensions(ctx, r.pool, limit)
	if err != nil {
		return 0, err
	}
	reaped := 0
	for _, extensionID := range extensionIDs {
		if reaped >= limit {
			break
		}
		count, err := r.reapExpiredRuntimeLeasesForExtension(ctx, extensionID, limit-reaped)
		if err != nil {
			return reaped, err
		}
		reaped += count
	}
	return reaped, nil
}

func (r *PostgresExtensionDatabaseRegistry) reapExpiredRuntimeLeasesForExtension(
	ctx context.Context,
	extensionID string,
	limit int,
) (int, error) {
	identifiers, err := ExtensionDatabaseIdentifiersFor(extensionID)
	if err != nil || limit <= 0 {
		return 0, ErrExtensionDatabaseRegistryInvalid
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin expired runtime lease reap: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := lockExtensionDatabaseResource(ctx, tx, identifiers.LockKey); err != nil {
		return 0, err
	}
	databaseName, err := currentExtensionDatabaseName(ctx, tx)
	if err != nil {
		return 0, err
	}
	reaped, err := reapExpiredExtensionDatabaseRuntimeLeasesLocked(
		ctx, tx, extensionID, identifiers, databaseName, limit,
	)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit expired runtime lease reap: %w", err)
	}
	return reaped, nil
}

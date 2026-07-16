package extensions

import (
	"context"
	"fmt"
)

// publishPluginRuntimePublication is deliberately test-only. Production
// producers must derive the complete set from PostgreSQL under the shared
// desired-set fence.
func (s *PostgresStore) publishPluginRuntimePublication(
	ctx context.Context,
	reason PluginRuntimePublicationReason,
	actorUserID int64,
	members []PluginRuntimeMember,
) (PluginRuntimePublication, error) {
	if s == nil || s.pool == nil || ctx == nil || actorUserID < 0 || !validPluginRuntimePublicationReason(reason) {
		return PluginRuntimePublication{}, ErrPluginRuntimePublicationConflict
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PluginRuntimePublication{}, fmt.Errorf("begin plugin runtime publication: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	publication, err := insertPluginRuntimePublication(ctx, tx, reason, actorUserID, members)
	if err != nil {
		return PluginRuntimePublication{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PluginRuntimePublication{}, fmt.Errorf(
			"commit plugin runtime publication: %w",
			mapPluginRuntimePostgresError(err, ErrPluginRuntimePublicationConflict),
		)
	}
	return publication, nil
}

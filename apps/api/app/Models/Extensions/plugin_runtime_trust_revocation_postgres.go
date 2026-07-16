package extensions

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// PublishPluginRuntimeTrustRevocationTx appends the recovery full-set that
// removes a revoked extension. The caller owns commit/rollback so this revision
// can be atomic with the authoritative trust-grant revocation.
func PublishPluginRuntimeTrustRevocationTx(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
	actorUserID int64,
) (PluginRuntimePublication, error) {
	if ctx == nil || tx == nil || actorUserID < 0 {
		return PluginRuntimePublication{}, ErrPluginRuntimePublicationConflict
	}
	latest, err := lockLatestPluginRuntimePublication(ctx, tx)
	if err != nil {
		return PluginRuntimePublication{}, err
	}
	next, err := TransitionPluginRuntimeTrustRevocationMembers(latest.Members, extensionID)
	if err != nil {
		return PluginRuntimePublication{}, err
	}
	publication, err := insertPluginRuntimePublication(
		ctx, tx, PluginRuntimePublicationRecovery, actorUserID, next,
	)
	if err != nil {
		return PluginRuntimePublication{}, fmt.Errorf("publish plugin runtime trust revocation: %w", err)
	}
	return publication, nil
}

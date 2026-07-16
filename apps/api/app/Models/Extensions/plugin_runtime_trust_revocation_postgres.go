package extensions

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// PublishPluginRuntimeTrustRevocationTx appends the recovery full-set that
// removes a revoked extension. It returns published=false when no active member
// exists. The caller owns commit/rollback so a revision can be atomic with the
// authoritative trust-grant revocation.
func PublishPluginRuntimeTrustRevocationTx(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
	actorUserID int64,
) (PluginRuntimePublication, bool, error) {
	if ctx == nil || tx == nil || actorUserID < 0 {
		return PluginRuntimePublication{}, false, ErrPluginRuntimePublicationConflict
	}
	latest, err := lockLatestPluginRuntimePublication(ctx, tx)
	if errors.Is(err, ErrPluginRuntimePublicationNotFound) {
		return PluginRuntimePublication{}, false, nil
	}
	if err != nil {
		return PluginRuntimePublication{}, false, err
	}
	if _, found := pluginRuntimeMemberForExtension(latest.Members, extensionID); !found {
		if _, err := TransitionPluginRuntimeTrustRevocationMembers(latest.Members, extensionID); err != nil {
			return PluginRuntimePublication{}, false, err
		}
		return PluginRuntimePublication{}, false, nil
	}
	next, err := TransitionPluginRuntimeTrustRevocationMembers(latest.Members, extensionID)
	if err != nil {
		return PluginRuntimePublication{}, false, err
	}
	publication, err := insertPluginRuntimePublication(
		ctx, tx, PluginRuntimePublicationRecovery, actorUserID, next,
	)
	if err != nil {
		return PluginRuntimePublication{}, false, fmt.Errorf("publish plugin runtime trust revocation: %w", err)
	}
	return publication, true, nil
}

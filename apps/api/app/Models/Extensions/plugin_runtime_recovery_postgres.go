package extensions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// PublishPluginRuntimeRecoveryTx serializes an out-of-band recovery mutation
// with every other desired-set producer and appends one authoritative recovery
// revision in the caller-owned transaction. mutate must only change Host-owned
// durable state; it returns the complete set of extension ids disabled by that
// one recovery command.
//
// The callback runs only after existing immutable authority is validated and
// while the shared desired-set lock is held. This keeps the CLI independent of
// package files and prevents lock inversion with normal lifecycle writers.
func PublishPluginRuntimeRecoveryTx(
	ctx context.Context,
	tx pgx.Tx,
	mutate func() ([]string, error),
) (PluginRuntimePublication, error) {
	return publishPluginRuntimeRecoveryTx(ctx, tx, mutate, false)
}

// PublishProtectedPluginRuntimeQuarantineTx is the exact-artifact emergency
// path for built-in/system executables. The caller must CAS the protected row
// before returning its id; this function only accepts protected disabled rows.
func PublishProtectedPluginRuntimeQuarantineTx(
	ctx context.Context,
	tx pgx.Tx,
	mutate func() ([]string, error),
) (PluginRuntimePublication, error) {
	return publishPluginRuntimeRecoveryTx(ctx, tx, mutate, true)
}

func publishPluginRuntimeRecoveryTx(
	ctx context.Context,
	tx pgx.Tx,
	mutate func() ([]string, error),
	protected bool,
) (PluginRuntimePublication, error) {
	if ctx == nil || tx == nil || mutate == nil {
		return PluginRuntimePublication{}, ErrPluginRuntimePublicationConflict
	}
	var isolation string
	if err := tx.QueryRow(ctx, `SHOW transaction_isolation`).Scan(&isolation); err != nil {
		return PluginRuntimePublication{}, fmt.Errorf("read plugin runtime recovery isolation: %w", err)
	}
	// Waiting for the advisory lock must be followed by a fresh statement
	// snapshot; snapshot isolation could otherwise retain a stale full-set.
	if strings.TrimSpace(strings.ToLower(isolation)) != "read committed" {
		return PluginRuntimePublication{}, ErrPluginRuntimePublicationConflict
	}
	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended($1, 0))
	`, pluginRuntimeDesiredSetLock); err != nil {
		return PluginRuntimePublication{}, fmt.Errorf("lock plugin runtime recovery desired set: %w", err)
	}

	latest, err := loadPluginRuntimePublication(
		ctx,
		tx,
		pluginRuntimePublicationSelect+` ORDER BY revision DESC LIMIT 1`,
	)
	missingGenesis := errors.Is(err, ErrPluginRuntimePublicationNotFound)
	if err != nil && !missingGenesis {
		return PluginRuntimePublication{}, fmt.Errorf("load plugin runtime recovery full set: %w", err)
	}
	if !missingGenesis {
		// A structurally valid latest row is insufficient when the immutable
		// history did not begin with the one-time Host genesis projection.
		if err := requireLegacyPluginRuntimeGenesis(ctx, tx); err != nil {
			return PluginRuntimePublication{}, err
		}
	}

	disabledIDs, err := mutate()
	if err != nil {
		return PluginRuntimePublication{}, err
	}
	disabled, err := canonicalRecoveryExtensionIDs(disabledIDs)
	if err != nil {
		return PluginRuntimePublication{}, err
	}
	if err := validateRecoveryDisabledExtensionsTx(ctx, tx, disabledIDs, disabled, protected); err != nil {
		return PluginRuntimePublication{}, err
	}

	baseMembers := latest.Members
	if missingGenesis {
		// Recovery first changes mutable status, then imports only the surviving
		// enabled plugins. A broken target manifest is therefore never decoded.
		baseMembers, err = loadLegacyInitialPluginRuntimeMembers(ctx, tx)
		if err != nil {
			return PluginRuntimePublication{}, err
		}
		if _, err := insertPluginRuntimePublication(
			ctx, tx, PluginRuntimePublicationStartupReconcile, 0, baseMembers,
		); err != nil {
			return PluginRuntimePublication{}, err
		}
	}

	nextMembers := make([]PluginRuntimeMember, 0, len(baseMembers))
	for _, member := range baseMembers {
		if _, remove := disabled[member.ExtensionID]; !remove {
			nextMembers = append(nextMembers, member)
		}
	}
	return insertPluginRuntimePublication(
		ctx, tx, PluginRuntimePublicationRecovery, 0, nextMembers,
	)
}

func validateRecoveryDisabledExtensionsTx(
	ctx context.Context,
	tx pgx.Tx,
	ids []string,
	canonical map[string]struct{},
	protected bool,
) error {
	if len(canonical) == 0 {
		return nil
	}
	rows, err := tx.Query(ctx, `
		SELECT id, status, source, is_system
		FROM extensions
		WHERE id = ANY($1::text[])
		ORDER BY id COLLATE "C"
		FOR SHARE
	`, ids)
	if err != nil {
		return fmt.Errorf("validate recovery disabled extensions: %w", err)
	}
	defer rows.Close()
	seen := make(map[string]struct{}, len(canonical))
	for rows.Next() {
		var id, status, source string
		var isSystem bool
		if err := rows.Scan(&id, &status, &source, &isSystem); err != nil {
			return fmt.Errorf("scan recovery disabled extension: %w", err)
		}
		isProtected := source == SourceBuiltin || isSystem
		if _, expected := canonical[id]; !expected || status != StatusDisabled ||
			isProtected != protected {
			return ErrPluginRuntimePublicationConflict
		}
		seen[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate recovery disabled extensions: %w", err)
	}
	if len(seen) != len(canonical) {
		return ErrPluginRuntimePublicationConflict
	}
	return nil
}

func canonicalRecoveryExtensionIDs(ids []string) (map[string]struct{}, error) {
	canonical := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" || id != strings.TrimSpace(id) {
			return nil, ErrPluginRuntimePublicationConflict
		}
		if _, duplicate := canonical[id]; duplicate {
			return nil, ErrPluginRuntimePublicationConflict
		}
		canonical[id] = struct{}{}
	}
	return canonical, nil
}

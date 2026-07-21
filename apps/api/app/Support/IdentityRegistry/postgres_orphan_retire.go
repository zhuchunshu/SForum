package identityregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

const startupOrphanRetireAuditAction = "identity.registry.startup_orphan_retire"

// OrphanIdentityRetirer completes durable identity retirement for owners that
// are no longer expected enabled publishers. Used only by Host startup restore
// after force-delete or incomplete uninstall left active tips behind.
type OrphanIdentityRetirer interface {
	RetireOrphanPublications(ctx context.Context, extensionIDs []string) (DurableState, error)
}

// RetireOrphanPublications tombstones every active root and declaration tip for
// the given owners. Owners still present as extension rows lock their artifacts
// normally; owners already deleted skip the live artifact lock (DB trigger
// permits tombstone-only inserts when the extension row is gone).
func (s *PostgresStore) RetireOrphanPublications(
	ctx context.Context,
	extensionIDs []string,
) (DurableState, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return DurableState{}, ErrInvalid
	}
	owners, err := normalizeOrphanOwnerIDs(extensionIDs)
	if err != nil {
		return DurableState{}, err
	}
	if len(owners) == 0 {
		return s.LoadDurableState(ctx)
	}

	run := func() (DurableState, error) {
		for attempt := 0; attempt < maxPublicationReconcileAttempts; attempt++ {
			state, retireErr := s.retireOrphanPublicationsOnce(ctx, owners)
			if retireErr == nil {
				return state, nil
			}
			if !errors.Is(retireErr, errRetryableIdentityRegistryTransaction) {
				return DurableState{}, retireErr
			}
			if err := ctx.Err(); err != nil {
				return DurableState{}, mapStoreError(err)
			}
		}
		return DurableState{}, ErrRevisionConflict
	}
	return runSessionPolicyMutationGate(ctx, s.sessionPolicyMutationGate, run)
}

func (s *PostgresStore) retireOrphanPublicationsOnce(
	ctx context.Context,
	owners []string,
) (DurableState, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return DurableState{}, mapStoreError(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	auditEventID, err := insertStartupOrphanRetireAudit(ctx, tx, owners)
	if err != nil {
		return DurableState{}, err
	}

	for _, owner := range owners {
		if err := s.retireOneOrphanOwner(ctx, tx, owner, auditEventID); err != nil {
			return DurableState{}, err
		}
	}

	state, err := loadDurableStateFrom(ctx, tx)
	if err != nil {
		return DurableState{}, err
	}
	// After retirement every requested owner must be fully inactive.
	for _, owner := range owners {
		if err := ValidateDurableRetirement(state, owner); err != nil {
			return DurableState{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return DurableState{}, mapStoreError(err)
	}
	return state, nil
}

func (s *PostgresStore) retireOneOrphanOwner(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
	auditEventID int64,
) error {
	extensionPresent, err := lockOrphanOwnerExtensionPresence(ctx, tx, extensionID)
	if err != nil {
		return err
	}

	currentRoot, err := lockRootPublicationTip(ctx, tx, extensionID)
	if err != nil {
		return err
	}
	current, err := lockExtensionDeclarationTips(ctx, tx, extensionID)
	if err != nil {
		return err
	}

	activeArtifacts := collectActiveOrphanArtifacts(currentRoot, current)
	if len(activeArtifacts) == 0 {
		return nil
	}
	if extensionPresent {
		if err := lockReconcileArtifacts(ctx, tx, extensionID, activeArtifacts); err != nil {
			return err
		}
	}

	actorUserID, err := resolveOrphanRetireActor(currentRoot, current, activeArtifacts)
	if err != nil {
		return err
	}

	if s.sessionPolicyInvalidator != nil {
		if err := s.sessionPolicyInvalidator.InvalidateSessionPolicySelectionTx(
			ctx,
			tx,
			SessionPolicyLifecycleTransition{
				OwnerExtensionID:      extensionID,
				ActorUserID:           actorUserID,
				LifecycleAuditEventID: auditEventID,
			},
		); err != nil {
			return mapStoreError(err)
		}
	}

	input := normalizedReconcilePublicationInput{
		extensionID:  extensionID,
		actorUserID:  actorUserID,
		auditEventID: auditEventID,
	}
	if _, err := reconcileDurableRootPublication(ctx, tx, currentRoot, nil, input); err != nil {
		return err
	}

	currentKeys := make([]string, 0, len(current))
	for key := range current {
		currentKeys = append(currentKeys, key)
	}
	sort.Strings(currentKeys)
	for _, key := range currentKeys {
		tip := current[key]
		if tip.RegistryState != RegistryStateActive {
			continue
		}
		tip.RegistryState = RegistryStateTombstone
		tip.Revision++
		tip.ActorUserID = actorUserID
		tip.AuditEventID = auditEventID
		if _, err := insertDurableDeclaration(ctx, tx, tip); err != nil {
			return err
		}
	}
	return nil
}

func normalizeOrphanOwnerIDs(extensionIDs []string) ([]string, error) {
	seen := make(map[string]struct{}, len(extensionIDs))
	result := make([]string, 0, len(extensionIDs))
	for _, raw := range extensionIDs {
		id := strings.ToLower(strings.TrimSpace(raw))
		if id == "" {
			continue
		}
		if !idPattern.MatchString(id) || strings.HasPrefix(id, "core.") {
			return nil, ErrInvalid
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Strings(result)
	return result, nil
}

func lockOrphanOwnerExtensionPresence(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
) (bool, error) {
	var id string
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM extensions
		WHERE id = $1
		FOR UPDATE
	`, extensionID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, mapStoreError(err)
	}
	return true, nil
}

func collectActiveOrphanArtifacts(
	root *DurableRootPublicationTip,
	tips map[string]DurableDeclarationTip,
) []Artifact {
	byIdentity := make(map[durableArtifactIdentity]Artifact)
	if root != nil && root.RegistryState == RegistryStateActive {
		artifact := Artifact{
			ExtensionID:      root.OwnerExtensionID,
			ExtensionVersion: root.ExtensionVersion,
			PackageDigest:    root.PackageDigest,
			VersionID:        root.ExtensionVersionID,
		}
		byIdentity[durableArtifactIdentityOf(artifact)] = artifact
	}
	for _, tip := range tips {
		if tip.RegistryState != RegistryStateActive {
			continue
		}
		artifact := Artifact{
			ExtensionID:      tip.OwnerExtensionID,
			ExtensionVersion: tip.ExtensionVersion,
			PackageDigest:    tip.PackageDigest,
			VersionID:        tip.ExtensionVersionID,
		}
		byIdentity[durableArtifactIdentityOf(artifact)] = artifact
	}
	result := make([]Artifact, 0, len(byIdentity))
	for _, artifact := range byIdentity {
		result = append(result, artifact)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := durableArtifactIdentityOf(result[i]), durableArtifactIdentityOf(result[j])
		if left.VersionID != right.VersionID {
			return left.VersionID < right.VersionID
		}
		if left.Version != right.Version {
			return left.Version < right.Version
		}
		return left.PackageDigest < right.PackageDigest
	})
	return result
}

func resolveOrphanRetireActor(
	root *DurableRootPublicationTip,
	tips map[string]DurableDeclarationTip,
	artifacts []Artifact,
) (int64, error) {
	if root != nil && root.RegistryState == RegistryStateActive && root.ActorUserID > 0 {
		return root.ActorUserID, nil
	}
	for _, tip := range tips {
		if tip.RegistryState == RegistryStateActive && tip.ActorUserID > 0 {
			return tip.ActorUserID, nil
		}
	}
	// Root tip history may still carry the original actor after partial retirement.
	if root != nil && root.ActorUserID > 0 {
		return root.ActorUserID, nil
	}
	for _, tip := range tips {
		if tip.ActorUserID > 0 {
			return tip.ActorUserID, nil
		}
	}
	if len(artifacts) == 0 {
		return 0, ErrInvalid
	}
	return 0, fmt.Errorf("%w: startup orphan identity retire needs a historical actor", ErrInvalid)
}

func insertStartupOrphanRetireAudit(
	ctx context.Context,
	tx pgx.Tx,
	owners []string,
) (int64, error) {
	metadata, err := json.Marshal(map[string]any{
		"ownerExtensionIds": owners,
		"reason":            "startup_orphan_retire",
	})
	if err != nil {
		return 0, ErrInvalid
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO audit_events (actor_user_id, action, metadata)
		VALUES (NULL, $1, $2::jsonb)
		RETURNING id
	`, startupOrphanRetireAuditAction, string(metadata)).Scan(&auditEventID); err != nil {
		return 0, mapStoreError(err)
	}
	if auditEventID <= 0 {
		return 0, ErrInvalid
	}
	return auditEventID, nil
}

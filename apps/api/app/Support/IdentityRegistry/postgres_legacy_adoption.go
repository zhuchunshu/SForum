package identityregistry

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
)

const maxLegacyPublicationAdoptAttempts = 3

// AdoptLegacyPublications appends the first durable Identity Registry history
// for a batch of pre-feature enabled plugins when, and only when, live exact-
// artifact trust evidence still proves the operator already authorized each
// surface. The batch is all-or-none: any plugin failure writes zero rows.
//
// Transaction / lock order (SERIALIZABLE), publications stable-sorted by
// extension id:
//  1. normalize + preflight the complete desired publication graph
//  2. lock every exact enabled extension/version (FOR NO KEY UPDATE)
//  3. inspect every root/owner/declaration history (FOR UPDATE)
//  4. lock/validate every enable grant (FOR UPDATE) + audit (FOR SHARE)
//     with full TrustImpact digest integrity
//  5. preflight every desired stable ownership + Host permission/catalog
//  6. only after the entire batch passes, append missing root/owners/tips/
//     catalog/pending suggestions under the original actor+audit
//  7. reload + ValidateDurablePublicationSet before commit
func (s *PostgresStore) AdoptLegacyPublications(
	ctx context.Context,
	publications []Publication,
) (DurableState, error) {
	if s == nil || s.pool == nil || ctx == nil || len(publications) == 0 {
		return DurableState{}, ErrInvalid
	}
	// Fail closed without an instance-scoped integrity verifier. Ordinary
	// NewPostgresStore(pool) must never adopt under a package-global default.
	if s.trustImpactValidator == nil {
		return DurableState{}, ErrInvalid
	}

	prepared, err := prepareLegacyAdoptionBatch(publications)
	if err != nil {
		return DurableState{}, err
	}

	for attempt := 0; attempt < maxLegacyPublicationAdoptAttempts; attempt++ {
		state, adoptErr := s.adoptLegacyPublicationsOnce(ctx, prepared)
		if adoptErr == nil {
			return state, nil
		}
		if !errors.Is(adoptErr, errRetryableIdentityRegistryTransaction) {
			return DurableState{}, adoptErr
		}
		if err := ctx.Err(); err != nil {
			return DurableState{}, mapStoreError(err)
		}
	}
	return DurableState{}, ErrRevisionConflict
}

type legacyAdoptionItem struct {
	desired       Publication
	desiredRoot   *durableDesiredRootPublication
	desiredLeaves []durableDesiredDeclaration
}

func prepareLegacyAdoptionBatch(publications []Publication) ([]legacyAdoptionItem, error) {
	items := make([]legacyAdoptionItem, 0, len(publications))
	seenOwners := make(map[string]struct{}, len(publications))
	// Cross-batch ownership keys detect duplicate stable ids before any lock.
	ownershipOwners := make(map[string]string)

	for _, publication := range publications {
		normalized, err := normalizePublication(publication)
		if err != nil || normalized.Artifact.Core {
			return nil, ErrInvalid
		}
		if _, duplicate := seenOwners[normalized.Artifact.ExtensionID]; duplicate {
			return nil, ErrConflict
		}
		seenOwners[normalized.Artifact.ExtensionID] = struct{}{}

		desiredLeaves, err := desiredDurableDeclarations(&normalized)
		if err != nil {
			return nil, err
		}
		desiredRoot, err := desiredDurableRootPublication(&normalized)
		if err != nil || desiredRoot == nil {
			return nil, ErrInvalid
		}
		for _, declaration := range desiredLeaves {
			key := ownershipKey(declaration.kind, declaration.stableID)
			if owner, found := ownershipOwners[key]; found && owner != normalized.Artifact.ExtensionID {
				return nil, ErrConflict
			}
			ownershipOwners[key] = normalized.Artifact.ExtensionID
		}
		items = append(items, legacyAdoptionItem{
			desired:       normalized,
			desiredRoot:   desiredRoot,
			desiredLeaves: desiredLeaves,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].desired.Artifact.ExtensionID < items[j].desired.Artifact.ExtensionID
	})
	return items, nil
}

func (s *PostgresStore) adoptLegacyPublicationsOnce(
	ctx context.Context,
	items []legacyAdoptionItem,
) (DurableState, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return DurableState{}, mapStoreError(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	type pendingWrite struct {
		item     legacyAdoptionItem
		evidence legacyTrustEvidence
	}
	pending := make([]pendingWrite, 0, len(items))
	// Desired set for final ValidateDurablePublicationSet includes winners
	// that already have exact active durable history plus newly written ones.
	desiredSet := make([]Publication, 0, len(items))

	// (2) Lock every exact enabled extension/version in stable extension-id order.
	for _, item := range items {
		if err := lockEnabledExactExtension(ctx, tx, item.desired.Artifact); err != nil {
			return DurableState{}, err
		}
	}

	// (3) Inspect every root/owner/declaration history. Concurrent winners with
	// exact active publications are accepted without rewrite; partial/tombstone
	// /conflicting history fails the whole batch.
	for _, item := range items {
		extensionID := item.desired.Artifact.ExtensionID
		currentRoot, err := lockRootPublicationTip(ctx, tx, extensionID)
		if err != nil {
			return DurableState{}, err
		}
		if currentRoot != nil {
			// Any root tip (active or tombstone) means history exists. Only an
			// exact active publication may short-circuit writes for this owner.
			state, loadErr := loadDurableStateFrom(ctx, tx)
			if loadErr != nil {
				return DurableState{}, loadErr
			}
			if validateErr := ValidateDurablePublication(state, item.desired); validateErr != nil {
				return DurableState{}, validateErr
			}
			desiredSet = append(desiredSet, item.desired)
			continue
		}
		if err := rejectLegacyOwnerHistory(ctx, tx, extensionID); err != nil {
			return DurableState{}, err
		}
		if err := rejectLegacyDeclarationHistory(ctx, tx, extensionID); err != nil {
			return DurableState{}, err
		}
		desiredSet = append(desiredSet, item.desired)
		pending = append(pending, pendingWrite{item: item})
	}

	// (4) Lock/validate grant+audit+full impact for every still-missing owner.
	for index := range pending {
		evidence, err := lockLegacyTrustEvidence(ctx, tx, pending[index].item.desired.Artifact)
		if err != nil {
			return DurableState{}, err
		}
		if err := trustImpactAuthorizesDesiredPublication(
			s.trustImpactValidator,
			evidence.impactDocument, evidence.impactDigest, pending[index].item.desired,
		); err != nil {
			return DurableState{}, err
		}
		pending[index].evidence = evidence
	}

	// (5) Preflight ownership + Host permission/catalog collisions for every
	// write candidate before any durable append. Failures leave zero writes.
	for _, write := range pending {
		if err := preflightLegacyOwnershipAndCatalog(ctx, tx, write.item); err != nil {
			return DurableState{}, err
		}
	}

	// (6) Append every missing root/owner/tip/catalog/pending suggestion.
	for _, write := range pending {
		input := normalizedReconcilePublicationInput{
			extensionID:  write.item.desired.Artifact.ExtensionID,
			desired:      &write.item.desired,
			actorUserID:  write.evidence.actorUserID,
			auditEventID: write.evidence.auditEventID,
		}
		if _, err := reconcileDurableRootPublication(ctx, tx, nil, write.item.desiredRoot, input); err != nil {
			return DurableState{}, err
		}
		for _, declaration := range write.item.desiredLeaves {
			if err := ensureDurableOwner(ctx, tx, declaration, input.extensionID); err != nil {
				return DurableState{}, err
			}
			tip := DurableDeclarationTip{
				IdentityKind: declaration.kind, StableID: declaration.stableID,
				OwnerExtensionID: input.extensionID, Revision: 1,
				RegistryState:      RegistryStateActive,
				ExtensionVersionID: declaration.artifact.VersionID,
				ExtensionVersion:   declaration.artifact.ExtensionVersion,
				PackageDigest:      declaration.artifact.PackageDigest,
				ContractVersion:    declaration.contractVersion,
				DeclarationDigest:  declaration.digest,
				ActorUserID:        input.actorUserID, AuditEventID: input.auditEventID,
			}
			inserted, insertErr := insertDurableDeclaration(ctx, tx, tip)
			if insertErr != nil {
				return DurableState{}, insertErr
			}
			if declaration.permission != nil {
				if err := ensurePermissionCatalog(ctx, tx, *declaration.permission, inserted); err != nil {
					return DurableState{}, err
				}
				if err := insertPendingRoleSuggestions(ctx, tx, *declaration.permission, inserted); err != nil {
					return DurableState{}, err
				}
			}
		}
	}

	// (7) Reload and validate the complete desired set under the same locks.
	state, err := loadDurableStateFrom(ctx, tx)
	if err != nil {
		return DurableState{}, err
	}
	if err := ValidateDurablePublicationSet(state, desiredSet); err != nil {
		return DurableState{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DurableState{}, mapStoreError(err)
	}
	return state, nil
}

func lockEnabledExactExtension(ctx context.Context, tx pgx.Tx, artifact Artifact) error {
	var versionID int64
	err := tx.QueryRow(ctx, `
		SELECT version.id
		FROM extension_versions AS version
		JOIN extensions AS extension ON extension.id = version.extension_id
		WHERE extension.id = $1
		  AND extension.type = 'plugin'
		  AND extension.status = 'enabled'
		  AND extension.active_version_id = $2
		  AND version.id = $2
		  AND version.version = $3
		  AND version.package_digest = $4
		FOR NO KEY UPDATE OF extension, version
	`, artifact.ExtensionID, artifact.VersionID, artifact.ExtensionVersion, artifact.PackageDigest).Scan(&versionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrArtifactConflict
	}
	if err != nil {
		return mapStoreError(err)
	}
	return nil
}

// rejectLegacyOwnerHistory refuses any permanent ownership row for the owner.
// Partial or foreign history is never repaired or deleted by adoption.
func rejectLegacyOwnerHistory(ctx context.Context, tx pgx.Tx, extensionID string) error {
	rows, err := tx.Query(ctx, `
		SELECT identity_kind, stable_id
		FROM extension_identity_registry_owners
		WHERE owner_extension_id = $1
		ORDER BY identity_kind ASC, stable_id ASC
		FOR UPDATE
	`, extensionID)
	if err != nil {
		return mapStoreError(err)
	}
	defer rows.Close()
	if rows.Next() {
		return ErrInvalid
	}
	if err := rows.Err(); err != nil {
		return mapStoreError(err)
	}
	return nil
}

func rejectLegacyDeclarationHistory(ctx context.Context, tx pgx.Tx, extensionID string) error {
	rows, err := tx.Query(ctx, `
		SELECT identity_kind, stable_id, revision
		FROM extension_identity_registry_declarations
		WHERE owner_extension_id = $1
		ORDER BY identity_kind ASC, stable_id ASC, revision ASC
		FOR UPDATE
	`, extensionID)
	if err != nil {
		return mapStoreError(err)
	}
	defer rows.Close()
	if rows.Next() {
		return ErrInvalid
	}
	if err := rows.Err(); err != nil {
		return mapStoreError(err)
	}
	return nil
}

// preflightLegacyOwnershipAndCatalog proves every desired stable id is free or
// already owned by this extension, and every permission key is free of foreign
// catalog ownership and untracked Host permissions. No rows are inserted.
func preflightLegacyOwnershipAndCatalog(
	ctx context.Context,
	tx pgx.Tx,
	item legacyAdoptionItem,
) error {
	for _, declaration := range item.desiredLeaves {
		var owner string
		err := tx.QueryRow(ctx, `
			SELECT owner_extension_id
			FROM extension_identity_registry_owners
			WHERE identity_kind = $1 AND stable_id = $2
			FOR UPDATE
		`, declaration.kind, declaration.stableID).Scan(&owner)
		if err == nil {
			if owner != item.desired.Artifact.ExtensionID {
				return ErrConflict
			}
			// Owner already claimed by this extension is impossible after
			// rejectLegacyOwnerHistory; treat as fail-closed inconsistency.
			return ErrInvalid
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return mapStoreError(err)
		}
		if declaration.permission == nil {
			continue
		}
		var catalogOwner string
		err = tx.QueryRow(ctx, `
			SELECT owner_extension_id
			FROM extension_permission_catalog
			WHERE permission_key = $1
			FOR KEY SHARE
		`, declaration.permission.Key).Scan(&catalogOwner)
		if err == nil {
			if catalogOwner != item.desired.Artifact.ExtensionID {
				return ErrConflict
			}
			return ErrInvalid
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return mapStoreError(err)
		}
		var existing string
		err = tx.QueryRow(ctx, `
			SELECT key FROM permissions WHERE key = $1 FOR KEY SHARE
		`, declaration.permission.Key).Scan(&existing)
		if err == nil {
			// Untracked Host permission must never be retroactively claimed.
			return ErrConflict
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return mapStoreError(err)
		}
	}
	return nil
}

type legacyTrustEvidence struct {
	actorUserID    int64
	auditEventID   int64
	impactDocument []byte
	impactDigest   string
}

func lockLegacyTrustEvidence(
	ctx context.Context,
	tx pgx.Tx,
	artifact Artifact,
) (legacyTrustEvidence, error) {
	// Lock every live exact enable grant for this artifact. Ambiguous grants
	// with different impact digests must fail closed rather than LIMIT 1.
	rows, err := tx.Query(ctx, `
		SELECT grant_row.id,
		       grant_row.extension_version,
		       grant_row.package_digest,
		       grant_row.impact_document,
			       grant_row.impact_digest,
			       grant_row.granted_by_user_id,
			       grant_row.granted_at
			FROM extension_trust_grants AS grant_row
			WHERE grant_row.extension_id = $1
			  AND grant_row.extension_version = $2
			  AND grant_row.package_digest = $3
			  AND grant_row.action = $4
			  AND grant_row.revoked_at IS NULL
			  AND grant_row.granted_by_user_id IS NOT NULL
			ORDER BY grant_row.granted_at ASC, grant_row.id ASC
		FOR UPDATE OF grant_row
	`, artifact.ExtensionID, artifact.ExtensionVersion, artifact.PackageDigest,
		trustGrantEnableAction)
	if err != nil {
		return legacyTrustEvidence{}, mapStoreError(err)
	}
	defer rows.Close()

	type grantCandidate struct {
		id              int64
		version         string
		packageDigest   string
		impactDocument  []byte
		impactDigest    string
		grantedByUserID int64
		grantedAt       time.Time
	}
	candidates := make([]grantCandidate, 0, 1)
	for rows.Next() {
		var candidate grantCandidate
		if err := rows.Scan(
			&candidate.id, &candidate.version, &candidate.packageDigest,
			&candidate.impactDocument, &candidate.impactDigest,
			&candidate.grantedByUserID, &candidate.grantedAt,
		); err != nil {
			return legacyTrustEvidence{}, mapStoreError(err)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return legacyTrustEvidence{}, mapStoreError(err)
	}
	if len(candidates) != 1 {
		return legacyTrustEvidence{}, ErrInvalid
	}
	grant := candidates[0]
	if grant.id <= 0 || grant.grantedByUserID <= 0 ||
		grant.version != artifact.ExtensionVersion ||
		grant.packageDigest != artifact.PackageDigest ||
		!digestPattern.MatchString(grant.impactDigest) || len(grant.impactDocument) == 0 {
		return legacyTrustEvidence{}, ErrInvalid
	}

	// Exactly one matching trust-grant audit row, locked FOR SHARE through
	// commit so concurrent metadata update/delete cannot slip past validation.
	auditRows, err := tx.Query(ctx, `
		SELECT id, COALESCE(actor_user_id, 0), metadata, created_at
		FROM audit_events
		WHERE action = $1
		  AND actor_user_id = $2
		  AND metadata->>'extensionId' = $3
		  AND metadata->>'version' = $4
		  AND metadata->>'packageDigest' = $5
		  AND metadata->>'impactDigest' = $6
		ORDER BY id ASC
		FOR SHARE
	`, trustGrantAuditAction, grant.grantedByUserID, artifact.ExtensionID,
		artifact.ExtensionVersion, artifact.PackageDigest, grant.impactDigest)
	if err != nil {
		return legacyTrustEvidence{}, mapStoreError(err)
	}
	defer auditRows.Close()

	type auditCandidate struct {
		id        int64
		actorID   int64
		metadata  []byte
		createdAt time.Time
	}
	audits := make([]auditCandidate, 0, 1)
	for auditRows.Next() {
		var candidate auditCandidate
		if err := auditRows.Scan(&candidate.id, &candidate.actorID, &candidate.metadata, &candidate.createdAt); err != nil {
			return legacyTrustEvidence{}, mapStoreError(err)
		}
		audits = append(audits, candidate)
	}
	if err := auditRows.Err(); err != nil {
		return legacyTrustEvidence{}, mapStoreError(err)
	}
	if len(audits) != 1 {
		return legacyTrustEvidence{}, ErrInvalid
	}
	audit := audits[0]
	if audit.id <= 0 || audit.actorID != grant.grantedByUserID {
		return legacyTrustEvidence{}, ErrInvalid
	}
	// Audit must not predate the grant it claims to record.
	if audit.createdAt.Before(grant.grantedAt) {
		return legacyTrustEvidence{}, ErrInvalid
	}
	var metadata map[string]any
	if err := json.Unmarshal(audit.metadata, &metadata); err != nil {
		return legacyTrustEvidence{}, ErrInvalid
	}
	if auditMetadataString(metadata, "action") != trustGrantEnableAction ||
		auditMetadataString(metadata, "extensionId") != artifact.ExtensionID ||
		auditMetadataString(metadata, "version") != artifact.ExtensionVersion ||
		auditMetadataString(metadata, "packageDigest") != artifact.PackageDigest ||
		auditMetadataString(metadata, "impactDigest") != grant.impactDigest {
		return legacyTrustEvidence{}, ErrInvalid
	}

	return legacyTrustEvidence{
		actorUserID:    grant.grantedByUserID,
		auditEventID:   audit.id,
		impactDocument: append([]byte(nil), grant.impactDocument...),
		impactDigest:   grant.impactDigest,
	}, nil
}

var _ LegacyPublicationAdopter = (*PostgresStore)(nil)

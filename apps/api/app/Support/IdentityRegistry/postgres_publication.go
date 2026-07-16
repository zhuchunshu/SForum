package identityregistry

import (
	"bytes"
	"context"
	"errors"
	"sort"

	"github.com/jackc/pgx/v5"
)

const maxPublicationReconcileAttempts = 3

func (s *PostgresStore) Reconcile(
	ctx context.Context,
	input ReconcilePublicationInput,
) (DurableState, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return DurableState{}, ErrInvalid
	}
	normalized, err := normalizeReconcilePublicationInput(input)
	if err != nil {
		return DurableState{}, err
	}
	desired, err := desiredDurableDeclarations(normalized.desired)
	if err != nil {
		return DurableState{}, err
	}
	desiredRoot, err := desiredDurableRootPublication(normalized.desired)
	if err != nil {
		return DurableState{}, err
	}

	for attempt := 0; attempt < maxPublicationReconcileAttempts; attempt++ {
		state, reconcileErr := s.reconcilePublicationOnce(ctx, normalized, desiredRoot, desired)
		if reconcileErr == nil {
			return state, nil
		}
		if !errors.Is(reconcileErr, errRetryableIdentityRegistryTransaction) {
			return DurableState{}, reconcileErr
		}
		if err := ctx.Err(); err != nil {
			return DurableState{}, mapStoreError(err)
		}
	}
	return DurableState{}, ErrRevisionConflict
}

func (s *PostgresStore) reconcilePublicationOnce(
	ctx context.Context,
	input normalizedReconcilePublicationInput,
	desiredRoot *durableDesiredRootPublication,
	desired []durableDesiredDeclaration,
) (DurableState, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return DurableState{}, mapStoreError(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	if err := lockReconcileArtifacts(ctx, tx, input.extensionID, input.artifacts); err != nil {
		return DurableState{}, err
	}
	currentRoot, err := lockRootPublicationTip(ctx, tx, input.extensionID)
	if err != nil {
		return DurableState{}, err
	}
	if currentRoot != nil && currentRoot.RegistryState == RegistryStateActive {
		if _, allowed := input.allowed[durableRootTipArtifactIdentity(*currentRoot)]; !allowed {
			return DurableState{}, ErrArtifactConflict
		}
	}
	current, err := lockExtensionDeclarationTips(ctx, tx, input.extensionID)
	if err != nil {
		return DurableState{}, err
	}
	for _, tip := range current {
		if tip.RegistryState != RegistryStateActive {
			continue
		}
		if _, allowed := input.allowed[durableTipArtifactIdentity(tip)]; !allowed {
			return DurableState{}, ErrArtifactConflict
		}
	}
	if _, err := reconcileDurableRootPublication(ctx, tx, currentRoot, desiredRoot, input); err != nil {
		return DurableState{}, err
	}

	desiredByKey := make(map[string]durableDesiredDeclaration, len(desired))
	for _, declaration := range desired {
		key := ownershipKey(declaration.kind, declaration.stableID)
		desiredByKey[key] = declaration
		if err := ensureDurableOwner(ctx, tx, declaration, input.extensionID); err != nil {
			return DurableState{}, err
		}
	}

	// Retire removed declarations first. Tombstones copy the current exact
	// declaration byte-for-byte, so a stale disable cannot retire a winner.
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
		if _, remainsActive := desiredByKey[key]; remainsActive {
			continue
		}
		tip.RegistryState = RegistryStateTombstone
		tip.Revision++
		tip.ActorUserID = input.actorUserID
		tip.AuditEventID = input.auditEventID
		inserted, err := insertDurableDeclaration(ctx, tx, tip)
		if err != nil {
			return DurableState{}, err
		}
		current[key] = inserted
	}

	for _, declaration := range desired {
		key := ownershipKey(declaration.kind, declaration.stableID)
		tip, found := current[key]
		if found && durableTipArtifactIdentity(tip) == durableArtifactIdentityOf(declaration.artifact) {
			if tip.ContractVersion != declaration.contractVersion || tip.DeclarationDigest != declaration.digest {
				return DurableState{}, ErrArtifactConflict
			}
		}

		if !found || tip.RegistryState != RegistryStateActive ||
			durableTipArtifactIdentity(tip) != durableArtifactIdentityOf(declaration.artifact) {
			revision := int64(1)
			if found {
				revision = tip.Revision + 1
			}
			tip = DurableDeclarationTip{
				IdentityKind: declaration.kind, StableID: declaration.stableID,
				OwnerExtensionID: input.extensionID, Revision: revision,
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
			tip = inserted
			current[key] = inserted
		}

		if declaration.permission != nil {
			if err := ensurePermissionCatalog(ctx, tx, *declaration.permission, tip); err != nil {
				return DurableState{}, err
			}
			if err := insertPendingRoleSuggestions(ctx, tx, *declaration.permission, tip); err != nil {
				return DurableState{}, err
			}
		}
	}

	state, err := loadDurableStateFrom(ctx, tx)
	if err != nil {
		return DurableState{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DurableState{}, mapStoreError(err)
	}
	return state, nil
}

func lockRootPublicationTip(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
) (*DurableRootPublicationTip, error) {
	row := tx.QueryRow(ctx, `
		SELECT owner_extension_id, revision, registry_state,
		       extension_version_id, extension_version, package_digest,
		       schema_version, publication_digest, publication_json,
		       actor_user_id, audit_event_id, created_at
		FROM extension_identity_registry_publications
		WHERE owner_extension_id = $1
		ORDER BY revision DESC
		LIMIT 1
		FOR UPDATE
	`, extensionID)
	tip, err := scanDurableRootPublicationTip(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, mapStoreError(err)
	}
	roots, err := durableRootPublications(DurableState{RootTips: []DurableRootPublicationTip{tip}})
	if err != nil {
		return nil, err
	}
	normalized := roots[extensionID].tip
	return &normalized, nil
}

func reconcileDurableRootPublication(
	ctx context.Context,
	tx pgx.Tx,
	current *DurableRootPublicationTip,
	desired *durableDesiredRootPublication,
	input normalizedReconcilePublicationInput,
) (*DurableRootPublicationTip, error) {
	if desired == nil {
		if current == nil || current.RegistryState != RegistryStateActive {
			return current, nil
		}
		tombstone := *current
		tombstone.Revision++
		tombstone.RegistryState = RegistryStateTombstone
		tombstone.ActorUserID = input.actorUserID
		tombstone.AuditEventID = input.auditEventID
		inserted, err := insertDurableRootPublication(ctx, tx, tombstone)
		return &inserted, err
	}

	if desired.tip.OwnerExtensionID != input.extensionID {
		return nil, ErrInvalid
	}
	if current != nil && durableRootTipArtifactIdentity(*current) == durableRootTipArtifactIdentity(desired.tip) {
		if current.SchemaVersion != desired.tip.SchemaVersion ||
			current.PublicationDigest != desired.tip.PublicationDigest ||
			!bytes.Equal(current.PublicationJSON, desired.tip.PublicationJSON) {
			return nil, ErrArtifactConflict
		}
		if current.RegistryState == RegistryStateActive {
			return current, nil
		}
	}

	tip := desired.tip
	if current != nil {
		tip.Revision = current.Revision + 1
	}
	tip.ActorUserID = input.actorUserID
	tip.AuditEventID = input.auditEventID
	inserted, err := insertDurableRootPublication(ctx, tx, tip)
	return &inserted, err
}

func insertDurableRootPublication(
	ctx context.Context,
	tx pgx.Tx,
	tip DurableRootPublicationTip,
) (DurableRootPublicationTip, error) {
	err := tx.QueryRow(ctx, `
		INSERT INTO extension_identity_registry_publications (
			owner_extension_id, revision, registry_state,
			extension_version_id, extension_version, package_digest,
			schema_version, publication_digest, publication_json,
			actor_user_id, audit_event_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11)
		RETURNING created_at
	`, tip.OwnerExtensionID, tip.Revision, tip.RegistryState,
		tip.ExtensionVersionID, tip.ExtensionVersion, tip.PackageDigest,
		tip.SchemaVersion, tip.PublicationDigest, string(tip.PublicationJSON),
		tip.ActorUserID, tip.AuditEventID).Scan(&tip.CreatedAt)
	if err != nil {
		return DurableRootPublicationTip{}, mapStoreError(err)
	}
	return tip, nil
}

func lockReconcileArtifacts(ctx context.Context, tx pgx.Tx, extensionID string, artifacts []Artifact) error {
	for _, artifact := range artifacts {
		var versionID int64
		err := tx.QueryRow(ctx, `
			SELECT version.id
			FROM extension_versions AS version
			JOIN extensions AS extension ON extension.id = version.extension_id
			WHERE extension.id = $1
			  AND extension.type = 'plugin'
			  AND version.id = $2
			  AND version.version = $3
			  AND version.package_digest = $4
			FOR NO KEY UPDATE OF version, extension
		`, extensionID, artifact.VersionID, artifact.ExtensionVersion, artifact.PackageDigest).Scan(&versionID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrArtifactConflict
		}
		if err != nil {
			return mapStoreError(err)
		}
	}
	return nil
}

func lockExtensionDeclarationTips(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
) (map[string]DurableDeclarationTip, error) {
	ownerRows, err := tx.Query(ctx, `
		SELECT identity_kind, stable_id
		FROM extension_identity_registry_owners
		WHERE owner_extension_id = $1
		ORDER BY identity_kind ASC, stable_id ASC
		FOR UPDATE
	`, extensionID)
	if err != nil {
		return nil, mapStoreError(err)
	}
	ownerKeys := make(map[string]struct{})
	for ownerRows.Next() {
		var kind, stableID string
		if err := ownerRows.Scan(&kind, &stableID); err != nil {
			ownerRows.Close()
			return nil, mapStoreError(err)
		}
		ownerKeys[ownershipKey(kind, stableID)] = struct{}{}
	}
	if err := ownerRows.Err(); err != nil {
		ownerRows.Close()
		return nil, mapStoreError(err)
	}
	ownerRows.Close()

	tipRows, err := tx.Query(ctx, `
		SELECT declaration.identity_kind, declaration.stable_id,
		       declaration.owner_extension_id, declaration.revision,
		       declaration.registry_state, declaration.extension_version_id,
		       declaration.extension_version, declaration.package_digest,
		       declaration.contract_version, declaration.declaration_digest,
		       declaration.actor_user_id, declaration.audit_event_id,
		       declaration.created_at
		FROM extension_identity_registry_declarations AS declaration
		JOIN extension_identity_registry_owners AS owner
		  ON owner.identity_kind = declaration.identity_kind
		 AND owner.stable_id = declaration.stable_id
		WHERE owner.owner_extension_id = $1
		  AND declaration.revision = (
		    SELECT max(candidate.revision)
		    FROM extension_identity_registry_declarations AS candidate
		    WHERE candidate.identity_kind = declaration.identity_kind
		      AND candidate.stable_id = declaration.stable_id
		  )
		ORDER BY declaration.identity_kind ASC, declaration.stable_id ASC
		FOR UPDATE OF declaration
	`, extensionID)
	if err != nil {
		return nil, mapStoreError(err)
	}
	result := make(map[string]DurableDeclarationTip, len(ownerKeys))
	for tipRows.Next() {
		tip, err := scanDurableDeclarationTip(tipRows)
		if err != nil {
			tipRows.Close()
			return nil, err
		}
		result[ownershipKey(tip.IdentityKind, tip.StableID)] = tip
	}
	if err := tipRows.Err(); err != nil {
		tipRows.Close()
		return nil, mapStoreError(err)
	}
	tipRows.Close()
	if len(result) != len(ownerKeys) {
		return nil, ErrInvalid
	}
	return result, nil
}

func ensureDurableOwner(
	ctx context.Context,
	tx pgx.Tx,
	declaration durableDesiredDeclaration,
	extensionID string,
) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO extension_identity_registry_owners (
			identity_kind, stable_id, owner_extension_id
		) VALUES ($1, $2, $3)
		ON CONFLICT (identity_kind, stable_id) DO NOTHING
	`, declaration.kind, declaration.stableID, extensionID); err != nil {
		return mapStoreError(err)
	}
	var owner string
	if err := tx.QueryRow(ctx, `
		SELECT owner_extension_id
		FROM extension_identity_registry_owners
		WHERE identity_kind = $1 AND stable_id = $2
		FOR UPDATE
	`, declaration.kind, declaration.stableID).Scan(&owner); err != nil {
		return mapStoreError(err)
	}
	if owner != extensionID {
		return ErrConflict
	}
	return nil
}

func insertDurableDeclaration(
	ctx context.Context,
	tx pgx.Tx,
	tip DurableDeclarationTip,
) (DurableDeclarationTip, error) {
	err := tx.QueryRow(ctx, `
		INSERT INTO extension_identity_registry_declarations (
			identity_kind, stable_id, owner_extension_id, revision,
			registry_state, extension_version_id, extension_version,
			package_digest, contract_version, declaration_digest,
			actor_user_id, audit_event_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
		RETURNING created_at
	`, tip.IdentityKind, tip.StableID, tip.OwnerExtensionID, tip.Revision,
		tip.RegistryState, tip.ExtensionVersionID, tip.ExtensionVersion,
		tip.PackageDigest, tip.ContractVersion, tip.DeclarationDigest,
		tip.ActorUserID, tip.AuditEventID).Scan(&tip.CreatedAt)
	if err != nil {
		return DurableDeclarationTip{}, mapStoreError(err)
	}
	return tip, nil
}

func ensurePermissionCatalog(
	ctx context.Context,
	tx pgx.Tx,
	permission PermissionDefinition,
	tip DurableDeclarationTip,
) error {
	var catalogOwner string
	err := tx.QueryRow(ctx, `
		SELECT owner_extension_id
		FROM extension_permission_catalog
		WHERE permission_key = $1
		FOR KEY SHARE
	`, permission.Key).Scan(&catalogOwner)
	if err == nil {
		if catalogOwner != tip.OwnerExtensionID {
			return ErrConflict
		}
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return mapStoreError(err)
	}

	var existing string
	err = tx.QueryRow(ctx, `
		SELECT key FROM permissions WHERE key = $1 FOR KEY SHARE
	`, permission.Key).Scan(&existing)
	if err == nil {
		// An untracked Host permission must never be retroactively claimed by a
		// package, even when its key happens to match the package namespace.
		return ErrConflict
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return mapStoreError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO permissions (key, module, description)
		VALUES ($1, 'extension', $2)
	`, permission.Key, permission.Description); err != nil {
		return mapStoreError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO extension_permission_catalog (
			permission_key, owner_extension_id, declaration_revision,
			extension_version_id, extension_version, package_digest,
			contract_version, declaration_digest
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, permission.Key, tip.OwnerExtensionID, tip.Revision,
		tip.ExtensionVersionID, tip.ExtensionVersion, tip.PackageDigest,
		tip.ContractVersion, tip.DeclarationDigest); err != nil {
		return mapStoreError(err)
	}
	return nil
}

func insertPendingRoleSuggestions(
	ctx context.Context,
	tx pgx.Tx,
	permission PermissionDefinition,
	tip DurableDeclarationTip,
) error {
	for _, roleKey := range permission.RecommendedRoles {
		if _, err := tx.Exec(ctx, `
			INSERT INTO extension_permission_role_suggestions (
				permission_key, owner_extension_id, extension_version_id,
				extension_version, package_digest, permission_contract_version,
				declaration_digest, role_key, approval_state, revision
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending', 1)
			ON CONFLICT (
				permission_key, owner_extension_id, extension_version_id,
				package_digest, permission_contract_version, role_key
			) DO NOTHING
		`, permission.Key, tip.OwnerExtensionID, tip.ExtensionVersionID,
			tip.ExtensionVersion, tip.PackageDigest, tip.ContractVersion,
			tip.DeclarationDigest, roleKey); err != nil {
			return mapStoreError(err)
		}
	}
	return nil
}

func loadDurableStateFrom(ctx context.Context, tx pgx.Tx) (DurableState, error) {
	ownerRows, err := tx.Query(ctx, `
		SELECT identity_kind, stable_id, owner_extension_id, claimed_at
		FROM extension_identity_registry_owners
		ORDER BY identity_kind ASC, stable_id ASC
	`)
	if err != nil {
		return DurableState{}, mapStoreError(err)
	}
	owners := make([]DurableOwner, 0)
	for ownerRows.Next() {
		var owner DurableOwner
		if err := ownerRows.Scan(
			&owner.IdentityKind, &owner.StableID, &owner.OwnerExtensionID, &owner.ClaimedAt,
		); err != nil {
			ownerRows.Close()
			return DurableState{}, mapStoreError(err)
		}
		owners = append(owners, owner)
	}
	if err := ownerRows.Err(); err != nil {
		ownerRows.Close()
		return DurableState{}, mapStoreError(err)
	}
	ownerRows.Close()

	tipRows, err := tx.Query(ctx, `
		SELECT DISTINCT ON (identity_kind, stable_id)
		       identity_kind, stable_id, owner_extension_id, revision,
		       registry_state, extension_version_id, extension_version,
		       package_digest, contract_version, declaration_digest,
		       actor_user_id, audit_event_id, created_at
		FROM extension_identity_registry_declarations
		ORDER BY identity_kind ASC, stable_id ASC, revision DESC
	`)
	if err != nil {
		return DurableState{}, mapStoreError(err)
	}
	tips := make([]DurableDeclarationTip, 0)
	for tipRows.Next() {
		tip, err := scanDurableDeclarationTip(tipRows)
		if err != nil {
			tipRows.Close()
			return DurableState{}, err
		}
		tips = append(tips, tip)
	}
	if err := tipRows.Err(); err != nil {
		tipRows.Close()
		return DurableState{}, mapStoreError(err)
	}
	tipRows.Close()

	rootRows, err := tx.Query(ctx, `
		SELECT DISTINCT ON (owner_extension_id)
		       owner_extension_id, revision, registry_state,
		       extension_version_id, extension_version, package_digest,
		       schema_version, publication_digest, publication_json,
		       actor_user_id, audit_event_id, created_at
		FROM extension_identity_registry_publications
		ORDER BY owner_extension_id ASC, revision DESC
	`)
	if err != nil {
		return DurableState{}, mapStoreError(err)
	}
	rootTips := make([]DurableRootPublicationTip, 0)
	for rootRows.Next() {
		tip, scanErr := scanDurableRootPublicationTip(rootRows)
		if scanErr != nil {
			rootRows.Close()
			return DurableState{}, mapStoreError(scanErr)
		}
		rootTips = append(rootTips, tip)
	}
	if err := rootRows.Err(); err != nil {
		rootRows.Close()
		return DurableState{}, mapStoreError(err)
	}
	rootRows.Close()
	return DurableState{Owners: owners, Tips: tips, RootTips: rootTips}, nil
}

type durableDeclarationTipScanner interface {
	Scan(dest ...any) error
}

func scanDurableDeclarationTip(scanner durableDeclarationTipScanner) (DurableDeclarationTip, error) {
	var tip DurableDeclarationTip
	var actorUserID, auditEventID *int64
	if err := scanner.Scan(
		&tip.IdentityKind, &tip.StableID, &tip.OwnerExtensionID, &tip.Revision,
		&tip.RegistryState, &tip.ExtensionVersionID, &tip.ExtensionVersion,
		&tip.PackageDigest, &tip.ContractVersion, &tip.DeclarationDigest,
		&actorUserID, &auditEventID, &tip.CreatedAt,
	); err != nil {
		return DurableDeclarationTip{}, mapStoreError(err)
	}
	if actorUserID != nil {
		tip.ActorUserID = *actorUserID
	}
	if auditEventID != nil {
		tip.AuditEventID = *auditEventID
	}
	return tip, nil
}

func scanDurableRootPublicationTip(scanner durableDeclarationTipScanner) (DurableRootPublicationTip, error) {
	var tip DurableRootPublicationTip
	if err := scanner.Scan(
		&tip.OwnerExtensionID, &tip.Revision, &tip.RegistryState,
		&tip.ExtensionVersionID, &tip.ExtensionVersion, &tip.PackageDigest,
		&tip.SchemaVersion, &tip.PublicationDigest, &tip.PublicationJSON,
		&tip.ActorUserID, &tip.AuditEventID, &tip.CreatedAt,
	); err != nil {
		return DurableRootPublicationTip{}, err
	}
	return tip, nil
}

var _ PublicationStore = (*PostgresStore)(nil)

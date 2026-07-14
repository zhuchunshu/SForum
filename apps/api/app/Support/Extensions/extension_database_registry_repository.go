package extensionsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

type extensionDatabaseResourceRecord struct {
	ExtensionID     string
	SchemaName      string
	OwnerRoleName   string
	RuntimeRoleName string
	Status          string
	Revision        int64
}

func (r extensionDatabaseResourceRecord) matches(identifiers ExtensionDatabaseIdentifiers) bool {
	return r.SchemaName == identifiers.Schema && r.OwnerRoleName == identifiers.OwnerRole &&
		r.RuntimeRoleName == identifiers.RuntimeRole
}

type extensionDatabaseGrantRecord struct {
	ID                       int64
	ExtensionID              string
	VersionID                int64
	Version                  string
	PackageDigest            string
	DatabaseContractVersion  string
	Authority                string
	RequestedSchema          string
	RequestedRole            string
	Status                   string
	ActiveCredentialRevision int64
	Revision                 int64
}

func (r extensionDatabaseGrantRecord) matchesDeclaration(declaration extensions.ManifestDatabase) bool {
	return r.DatabaseContractVersion == declaration.ContractVersion &&
		r.Authority == declaration.Authority && r.RequestedSchema == declaration.Schema &&
		r.RequestedRole == declaration.Role
}

type extensionDatabaseRow interface {
	Scan(...any) error
}

type extensionDatabaseQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadExactExtensionDatabaseDeclaration(
	ctx context.Context,
	querier extensionDatabaseQuerier,
	artifact ExtensionDatabaseArtifact,
) (extensions.ManifestDatabase, error) {
	if validateExtensionDatabaseArtifact(artifact) != nil {
		return extensions.ManifestDatabase{}, ErrExtensionDatabaseRegistryInvalid
	}
	var extensionID, version, packageDigest string
	var manifestJSON []byte
	err := querier.QueryRow(ctx, `
		SELECT extension_id, version, package_digest, manifest
		FROM extension_versions
		WHERE id = $1
	`, artifact.VersionID).Scan(&extensionID, &version, &packageDigest, &manifestJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return extensions.ManifestDatabase{}, ErrExtensionDatabaseArtifactConflict
	}
	if err != nil {
		return extensions.ManifestDatabase{}, fmt.Errorf("load exact extension database artifact: %w", err)
	}
	if extensionID != artifact.ExtensionID || version != artifact.Version || packageDigest != artifact.PackageDigest {
		return extensions.ManifestDatabase{}, ErrExtensionDatabaseArtifactConflict
	}
	var manifest extensions.Manifest
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		return extensions.ManifestDatabase{}, fmt.Errorf("decode exact extension database manifest: %w", err)
	}
	if manifest.ID != artifact.ExtensionID || manifest.Version != artifact.Version || manifest.Database == nil {
		return extensions.ManifestDatabase{}, ErrExtensionDatabaseArtifactConflict
	}
	declaration := *manifest.Database
	if !extensionDatabaseContractPattern.MatchString(declaration.ContractVersion) ||
		declaration.Authority == "" {
		return extensions.ManifestDatabase{}, ErrExtensionDatabaseArtifactConflict
	}
	return declaration, nil
}

func loadExtensionDatabaseResource(
	ctx context.Context,
	querier extensionDatabaseQuerier,
	extensionID string,
	forUpdate bool,
) (extensionDatabaseResourceRecord, error) {
	query := `
		SELECT extension_id, schema_name, owner_role_name, runtime_role_name,
		       status, resource_revision
		FROM extension_database_resources
		WHERE extension_id = $1
	`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	var record extensionDatabaseResourceRecord
	err := querier.QueryRow(ctx, query, extensionID).Scan(
		&record.ExtensionID, &record.SchemaName, &record.OwnerRoleName,
		&record.RuntimeRoleName, &record.Status, &record.Revision,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return extensionDatabaseResourceRecord{}, ErrExtensionDatabaseGrantNotFound
	}
	if err != nil {
		return extensionDatabaseResourceRecord{}, fmt.Errorf("load extension database resource: %w", err)
	}
	return record, nil
}

func revokeActiveExtensionDatabaseGrants(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
	actorUserID int64,
	auditEventID int64,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE extension_database_grants
		SET status = 'revoked', revoked_by_user_id = $2, revoke_audit_event_id = $3,
		    revoked_at = statement_timestamp(), grant_revision = grant_revision + 1,
		    updated_at = statement_timestamp()
		WHERE extension_id = $1 AND status = 'active'
	`, extensionID, actorUserID, auditEventID)
	if err != nil {
		return fmt.Errorf("revoke previous extension database grants: %w", err)
	}
	return nil
}

func upsertExtensionDatabaseGrant(
	ctx context.Context,
	tx pgx.Tx,
	request ExtensionDatabaseGrantRequest,
	declaration extensions.ManifestDatabase,
) (extensionDatabaseGrantRecord, error) {
	row := tx.QueryRow(ctx, `
		INSERT INTO extension_database_grants (
			extension_id, extension_version_id, extension_version, package_digest,
			database_contract_version, authority, requested_schema, requested_role,
			granted_by_user_id, grant_audit_event_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (extension_id, extension_version_id, extension_version, package_digest)
		DO UPDATE SET
			database_contract_version = EXCLUDED.database_contract_version,
			authority = EXCLUDED.authority,
			requested_schema = EXCLUDED.requested_schema,
			requested_role = EXCLUDED.requested_role,
			status = 'active', granted_by_user_id = EXCLUDED.granted_by_user_id,
			grant_audit_event_id = EXCLUDED.grant_audit_event_id,
			revoked_by_user_id = NULL, revoke_audit_event_id = NULL,
			revoked_at = NULL, failure_code = '',
			grant_revision = extension_database_grants.grant_revision + 1,
			updated_at = statement_timestamp()
		RETURNING id, extension_id, extension_version_id, extension_version,
		          package_digest, database_contract_version, authority,
		          requested_schema, requested_role, status,
		          active_credential_revision, grant_revision
	`, request.Artifact.ExtensionID, request.Artifact.VersionID, request.Artifact.Version,
		request.Artifact.PackageDigest, declaration.ContractVersion, declaration.Authority,
		declaration.Schema, declaration.Role, request.ActorUserID, request.AuditEventID)
	record, err := scanExtensionDatabaseGrant(row)
	if err != nil {
		return extensionDatabaseGrantRecord{}, fmt.Errorf("upsert exact extension database grant: %w", err)
	}
	return record, nil
}

func loadExtensionDatabaseGrant(
	ctx context.Context,
	querier extensionDatabaseQuerier,
	artifact ExtensionDatabaseArtifact,
	activeOnly bool,
	forUpdate bool,
) (extensionDatabaseGrantRecord, error) {
	query := `
		SELECT id, extension_id, extension_version_id, extension_version,
		       package_digest, database_contract_version, authority,
		       requested_schema, requested_role, status,
		       active_credential_revision, grant_revision
		FROM extension_database_grants
		WHERE extension_id = $1 AND extension_version_id = $2
		  AND extension_version = $3 AND package_digest = $4
	`
	if activeOnly {
		query += ` AND status = 'active'`
	}
	if forUpdate {
		query += ` FOR UPDATE`
	}
	record, err := scanExtensionDatabaseGrant(querier.QueryRow(
		ctx, query, artifact.ExtensionID, artifact.VersionID, artifact.Version, artifact.PackageDigest,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return extensionDatabaseGrantRecord{}, ErrExtensionDatabaseGrantNotFound
	}
	if err != nil {
		return extensionDatabaseGrantRecord{}, fmt.Errorf("load exact extension database grant: %w", err)
	}
	return record, nil
}

func scanExtensionDatabaseGrant(row extensionDatabaseRow) (extensionDatabaseGrantRecord, error) {
	var record extensionDatabaseGrantRecord
	err := row.Scan(
		&record.ID, &record.ExtensionID, &record.VersionID, &record.Version,
		&record.PackageDigest, &record.DatabaseContractVersion, &record.Authority,
		&record.RequestedSchema, &record.RequestedRole, &record.Status,
		&record.ActiveCredentialRevision, &record.Revision,
	)
	return record, err
}

func nextExtensionDatabaseCredentialRevision(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
) (int64, error) {
	var revision int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(credential_revision), 0) + 1
		FROM extension_database_credentials
		WHERE extension_id = $1
	`, extensionID).Scan(&revision); err != nil {
		return 0, fmt.Errorf("allocate extension database credential revision: %w", err)
	}
	if revision <= 0 {
		return 0, ErrExtensionDatabaseCredential
	}
	return revision, nil
}

func revokeActiveExtensionDatabaseCredentials(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
	actorUserID int64,
	auditEventID int64,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE extension_database_credentials
		SET status = 'revoked', revoked_by_user_id = $2, revoke_audit_event_id = $3,
		    revoked_at = statement_timestamp()
		WHERE extension_id = $1 AND status = 'active'
	`, extensionID, actorUserID, auditEventID)
	if err != nil {
		return fmt.Errorf("revoke previous extension database credentials: %w", err)
	}
	return nil
}

func insertExtensionDatabaseCredential(
	ctx context.Context,
	tx pgx.Tx,
	grantID int64,
	request ExtensionDatabaseGrantRequest,
	roleName string,
	revision int64,
	fingerprint string,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO extension_database_credentials (
			grant_id, extension_id, credential_revision, role_name,
			credential_fingerprint, issued_by_user_id, issue_audit_event_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, grantID, request.Artifact.ExtensionID, revision, roleName, fingerprint,
		request.ActorUserID, request.AuditEventID)
	if err != nil {
		return fmt.Errorf("insert extension database credential revision: %w", err)
	}
	return nil
}

func activateExtensionDatabaseGrantCredential(
	ctx context.Context,
	tx pgx.Tx,
	grant extensionDatabaseGrantRecord,
	revision int64,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE extension_database_grants
		SET active_credential_revision = $2,
		    grant_revision = grant_revision + 1,
		    updated_at = statement_timestamp()
		WHERE id = $1 AND grant_revision = $3 AND status = 'active'
	`, grant.ID, revision, grant.Revision)
	if err != nil {
		return fmt.Errorf("activate extension database credential revision: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrExtensionDatabaseResourceConflict
	}
	return nil
}

func revokeExtensionDatabaseGrant(
	ctx context.Context,
	tx pgx.Tx,
	grant extensionDatabaseGrantRecord,
	actorUserID int64,
	auditEventID int64,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE extension_database_grants
		SET status = 'revoked', revoked_by_user_id = $2, revoke_audit_event_id = $3,
		    revoked_at = statement_timestamp(), grant_revision = grant_revision + 1,
		    updated_at = statement_timestamp()
		WHERE id = $1 AND grant_revision = $4 AND status = 'active'
	`, grant.ID, actorUserID, auditEventID, grant.Revision)
	if err != nil {
		return fmt.Errorf("revoke exact extension database grant: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrExtensionDatabaseResourceConflict
	}
	return nil
}

func markExtensionDatabaseResourceProvisioned(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE extension_database_resources
		SET status = 'provisioned', failure_code = '', revoked_at = NULL,
		    resource_revision = resource_revision + 1,
		    updated_at = statement_timestamp()
		WHERE extension_id = $1
	`, extensionID)
	if err != nil {
		return fmt.Errorf("mark extension database resource provisioned: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrExtensionDatabaseResourceConflict
	}
	return nil
}

func markExtensionDatabaseResourceRevoked(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE extension_database_resources
		SET status = 'revoked', revoked_at = statement_timestamp(),
		    schema_retained = TRUE, resource_revision = resource_revision + 1,
		    updated_at = statement_timestamp()
		WHERE extension_id = $1
	`, extensionID)
	if err != nil {
		return fmt.Errorf("mark extension database resource revoked: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrExtensionDatabaseResourceConflict
	}
	return nil
}

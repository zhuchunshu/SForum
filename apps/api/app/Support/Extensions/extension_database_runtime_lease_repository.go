package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type extensionDatabaseRuntimeLeaseQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func listExpiredExtensionDatabaseRuntimeLeaseExtensions(
	ctx context.Context,
	querier interface {
		Query(context.Context, string, ...any) (pgx.Rows, error)
	},
	limit int,
) ([]string, error) {
	rows, err := querier.Query(ctx, `
		SELECT extension_id
		FROM extension_database_runtime_leases
		WHERE status IN ('active', 'draining')
		  AND lease_expires_at <= statement_timestamp()
		GROUP BY extension_id
		ORDER BY min(lease_expires_at), extension_id
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list expired runtime lease extensions: %w", err)
	}
	defer rows.Close()
	extensionIDs := make([]string, 0, limit)
	for rows.Next() {
		var extensionID string
		if err := rows.Scan(&extensionID); err != nil {
			return nil, err
		}
		extensionIDs = append(extensionIDs, extensionID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return extensionIDs, nil
}

func reapExpiredExtensionDatabaseRuntimeLeasesLocked(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
	identifiers ExtensionDatabaseIdentifiers,
	databaseName string,
	limit int,
) (int, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, lease_id, grant_id, extension_id, extension_version_id,
		       extension_version, package_digest, runtime_instance_id, role_name,
		       status, issued_by, COALESCE(issued_by_user_id, 0), issue_audit_event_id,
		       issued_at, last_heartbeat_at, lease_expires_at, draining_at,
		       revoked_at, failure_code, lease_revision
		FROM extension_database_runtime_leases
		WHERE extension_id = $1 AND status IN ('active', 'draining')
		  AND lease_expires_at <= statement_timestamp()
		ORDER BY lease_expires_at, id
		LIMIT $2
		FOR UPDATE
	`, extensionID, limit)
	if err != nil {
		return 0, fmt.Errorf("lock expired runtime leases: %w", err)
	}
	leases := make([]ExtensionDatabaseRuntimeLeaseSnapshot, 0, limit)
	for rows.Next() {
		lease, err := scanExtensionDatabaseRuntimeLease(rows)
		if err != nil {
			rows.Close()
			return 0, err
		}
		leases = append(leases, lease)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	for _, lease := range leases {
		auditEventID, err := insertExtensionDatabaseRuntimeLeaseExpiryAudit(ctx, tx, lease)
		if err != nil {
			return 0, err
		}
		if err := revokeExtensionDatabaseRuntimeLeaseRole(
			ctx, tx, lease.RoleName, identifiers.OwnerRole, databaseName,
		); err != nil {
			return 0, err
		}
		if _, err := expireExtensionDatabaseRuntimeLease(ctx, tx, lease, auditEventID); err != nil {
			return 0, err
		}
	}
	return len(leases), nil
}

func insertExtensionDatabaseRuntimeLeaseExpiryAudit(
	ctx context.Context,
	tx pgx.Tx,
	lease ExtensionDatabaseRuntimeLeaseSnapshot,
) (int64, error) {
	var auditEventID int64
	err := tx.QueryRow(ctx, `
		INSERT INTO audit_events (action, metadata)
		VALUES ($1, jsonb_build_object(
			'extensionId', $2::text,
			'extensionVersion', $3::text,
			'packageDigest', $4::text,
			'runtimeInstanceId', $5::text,
			'leaseId', $6::text,
			'failureCode', $7::text
		))
		RETURNING id
	`, extensionDatabaseRuntimeLeaseExpiredAudit, lease.Artifact.ExtensionID,
		lease.Artifact.Version, lease.Artifact.PackageDigest, lease.RuntimeInstanceID,
		lease.LeaseID, extensionDatabaseRuntimeLeaseExpiredCode).Scan(&auditEventID)
	if err != nil {
		return 0, fmt.Errorf("record expired runtime lease audit: %w", err)
	}
	if auditEventID <= 0 {
		return 0, ErrExtensionDatabaseRuntimeLeaseConflict
	}
	return auditEventID, nil
}

func expireExtensionDatabaseRuntimeLease(
	ctx context.Context,
	tx pgx.Tx,
	lease ExtensionDatabaseRuntimeLeaseSnapshot,
	auditEventID int64,
) (ExtensionDatabaseRuntimeLeaseSnapshot, error) {
	row := tx.QueryRow(ctx, `
		UPDATE extension_database_runtime_leases
		SET status = 'failed', revoked_by = 'host', revoked_by_user_id = NULL,
		    revoke_audit_event_id = $2, revoked_at = statement_timestamp(),
		    failure_code = $3, lease_revision = lease_revision + 1
		WHERE id = $1 AND lease_revision = $4
		  AND status IN ('active', 'draining')
		  AND lease_expires_at <= statement_timestamp()
		RETURNING id, lease_id, grant_id, extension_id, extension_version_id,
		          extension_version, package_digest, runtime_instance_id, role_name,
		          status, issued_by, COALESCE(issued_by_user_id, 0), issue_audit_event_id,
		          issued_at, last_heartbeat_at, lease_expires_at, draining_at,
		          revoked_at, failure_code, lease_revision
	`, lease.ID, auditEventID, extensionDatabaseRuntimeLeaseExpiredCode, lease.Revision)
	next, err := scanExtensionDatabaseRuntimeLease(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ExtensionDatabaseRuntimeLeaseSnapshot{}, ErrExtensionDatabaseRuntimeLeaseConflict
	}
	if err != nil {
		return ExtensionDatabaseRuntimeLeaseSnapshot{}, fmt.Errorf("expire extension database runtime lease: %w", err)
	}
	return next, nil
}

func ensureExtensionDatabaseRuntimeLeaseCapacity(ctx context.Context, tx pgx.Tx, extensionID string) error {
	var count int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		FROM extension_database_runtime_leases
		WHERE extension_id = $1 AND status IN ('active', 'draining')
	`, extensionID).Scan(&count); err != nil {
		return fmt.Errorf("count live extension database runtime leases: %w", err)
	}
	if count >= extensionDatabaseMaximumLiveRuntimeLeases {
		return ErrExtensionDatabaseRuntimeLeaseConflict
	}
	return nil
}

func extensionDatabaseRuntimeLeaseWindow(ctx context.Context, tx pgx.Tx) (time.Time, time.Time, error) {
	var issuedAt, expiresAt time.Time
	if err := tx.QueryRow(ctx, `
		SELECT statement_timestamp(), statement_timestamp() + interval '2 minutes'
	`).Scan(&issuedAt, &expiresAt); err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("read extension database runtime lease window: %w", err)
	}
	if !expiresAt.Equal(issuedAt.Add(extensionDatabaseRuntimeLeaseTTL)) {
		return time.Time{}, time.Time{}, ErrExtensionDatabaseRuntimeLeaseConflict
	}
	return issuedAt.UTC(), expiresAt.UTC(), nil
}

func insertExtensionDatabaseRuntimeLease(
	ctx context.Context,
	tx pgx.Tx,
	request ExtensionDatabaseRuntimeLeaseIssue,
	grantID int64,
	leaseID string,
	roleName string,
	fingerprint string,
	issuedAt time.Time,
	expiresAt time.Time,
) (ExtensionDatabaseRuntimeLeaseSnapshot, error) {
	row := tx.QueryRow(ctx, `
		INSERT INTO extension_database_runtime_leases (
			lease_id, grant_id, extension_id, extension_version_id,
			extension_version, package_digest, runtime_instance_id, role_name,
			credential_fingerprint, issued_by, issued_by_user_id,
			issue_audit_event_id, issued_at, last_heartbeat_at, lease_expires_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11,
			$12, $13, $13, $14
		)
		RETURNING id, lease_id, grant_id, extension_id, extension_version_id,
		          extension_version, package_digest, runtime_instance_id, role_name,
		          status, issued_by, COALESCE(issued_by_user_id, 0), issue_audit_event_id,
		          issued_at, last_heartbeat_at, lease_expires_at, draining_at,
		          revoked_at, failure_code, lease_revision
	`, leaseID, grantID, request.Artifact.ExtensionID, request.Artifact.VersionID,
		request.Artifact.Version, request.Artifact.PackageDigest, request.RuntimeInstanceID,
		roleName, fingerprint, request.Authority.Kind, nullableExtensionDatabaseLeaseActor(request.Authority),
		request.Authority.AuditEventID, issuedAt, expiresAt)
	lease, err := scanExtensionDatabaseRuntimeLease(row)
	if err != nil {
		if isPostgresConstraintError(err) {
			return ExtensionDatabaseRuntimeLeaseSnapshot{}, ErrExtensionDatabaseRuntimeLeaseConflict
		}
		return ExtensionDatabaseRuntimeLeaseSnapshot{}, fmt.Errorf("insert extension database runtime lease: %w", err)
	}
	return lease, nil
}

func loadExtensionDatabaseRuntimeLease(
	ctx context.Context,
	querier extensionDatabaseRuntimeLeaseQuerier,
	ref ExtensionDatabaseRuntimeLeaseRef,
	forUpdate bool,
) (ExtensionDatabaseRuntimeLeaseSnapshot, error) {
	query := `
		SELECT id, lease_id, grant_id, extension_id, extension_version_id,
		       extension_version, package_digest, runtime_instance_id, role_name,
		       status, issued_by, COALESCE(issued_by_user_id, 0), issue_audit_event_id,
		       issued_at, last_heartbeat_at, lease_expires_at, draining_at,
		       revoked_at, failure_code, lease_revision
		FROM extension_database_runtime_leases
		WHERE lease_id = $1 AND extension_id = $2 AND extension_version_id = $3
		  AND extension_version = $4 AND package_digest = $5
		  AND runtime_instance_id = $6
	`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	lease, err := scanExtensionDatabaseRuntimeLease(querier.QueryRow(
		ctx, query, ref.LeaseID, ref.Artifact.ExtensionID, ref.Artifact.VersionID,
		ref.Artifact.Version, ref.Artifact.PackageDigest, ref.RuntimeInstanceID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return ExtensionDatabaseRuntimeLeaseSnapshot{}, ErrExtensionDatabaseRuntimeLeaseNotFound
	}
	if err != nil {
		return ExtensionDatabaseRuntimeLeaseSnapshot{}, fmt.Errorf("load extension database runtime lease: %w", err)
	}
	return lease, nil
}

func revokeExtensionDatabaseRuntimeLease(
	ctx context.Context,
	tx pgx.Tx,
	lease ExtensionDatabaseRuntimeLeaseSnapshot,
	authority ExtensionDatabaseLeaseAuthority,
) (ExtensionDatabaseRuntimeLeaseSnapshot, error) {
	row := tx.QueryRow(ctx, `
		UPDATE extension_database_runtime_leases
		SET status = 'revoked', revoked_by = $2, revoked_by_user_id = $3,
		    revoke_audit_event_id = $4, revoked_at = statement_timestamp(),
		    lease_revision = lease_revision + 1
		WHERE id = $1 AND lease_revision = $5
		  AND status IN ('active', 'draining')
		RETURNING id, lease_id, grant_id, extension_id, extension_version_id,
		          extension_version, package_digest, runtime_instance_id, role_name,
		          status, issued_by, COALESCE(issued_by_user_id, 0), issue_audit_event_id,
		          issued_at, last_heartbeat_at, lease_expires_at, draining_at,
		          revoked_at, failure_code, lease_revision
	`, lease.ID, authority.Kind, nullableExtensionDatabaseLeaseActor(authority),
		authority.AuditEventID, lease.Revision)
	next, err := scanExtensionDatabaseRuntimeLease(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ExtensionDatabaseRuntimeLeaseSnapshot{}, ErrExtensionDatabaseRuntimeLeaseConflict
	}
	if err != nil {
		return ExtensionDatabaseRuntimeLeaseSnapshot{}, fmt.Errorf("revoke extension database runtime lease: %w", err)
	}
	return next, nil
}

func heartbeatExtensionDatabaseRuntimeLease(
	ctx context.Context,
	tx pgx.Tx,
	lease ExtensionDatabaseRuntimeLeaseSnapshot,
	heartbeatAt time.Time,
	expiresAt time.Time,
) (ExtensionDatabaseRuntimeLeaseSnapshot, error) {
	row := tx.QueryRow(ctx, `
		UPDATE extension_database_runtime_leases
		SET last_heartbeat_at = $2, lease_expires_at = $3,
		    lease_revision = lease_revision + 1
		WHERE id = $1 AND lease_revision = $4 AND status = 'active'
		  AND lease_expires_at > $2
		RETURNING id, lease_id, grant_id, extension_id, extension_version_id,
		          extension_version, package_digest, runtime_instance_id, role_name,
		          status, issued_by, COALESCE(issued_by_user_id, 0), issue_audit_event_id,
		          issued_at, last_heartbeat_at, lease_expires_at, draining_at,
		          revoked_at, failure_code, lease_revision
	`, lease.ID, heartbeatAt, expiresAt, lease.Revision)
	next, err := scanExtensionDatabaseRuntimeLease(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ExtensionDatabaseRuntimeLeaseSnapshot{}, ErrExtensionDatabaseRuntimeLeaseConflict
	}
	if err != nil {
		return ExtensionDatabaseRuntimeLeaseSnapshot{}, fmt.Errorf("heartbeat extension database runtime lease: %w", err)
	}
	return next, nil
}

func drainExtensionDatabaseRuntimeLease(
	ctx context.Context,
	tx pgx.Tx,
	lease ExtensionDatabaseRuntimeLeaseSnapshot,
) (ExtensionDatabaseRuntimeLeaseSnapshot, error) {
	row := tx.QueryRow(ctx, `
		UPDATE extension_database_runtime_leases
		SET status = 'draining', draining_at = statement_timestamp(),
		    lease_revision = lease_revision + 1
		WHERE id = $1 AND lease_revision = $2 AND status = 'active'
		RETURNING id, lease_id, grant_id, extension_id, extension_version_id,
		          extension_version, package_digest, runtime_instance_id, role_name,
		          status, issued_by, COALESCE(issued_by_user_id, 0), issue_audit_event_id,
		          issued_at, last_heartbeat_at, lease_expires_at, draining_at,
		          revoked_at, failure_code, lease_revision
	`, lease.ID, lease.Revision)
	next, err := scanExtensionDatabaseRuntimeLease(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ExtensionDatabaseRuntimeLeaseSnapshot{}, ErrExtensionDatabaseRuntimeLeaseConflict
	}
	if err != nil {
		return ExtensionDatabaseRuntimeLeaseSnapshot{}, fmt.Errorf("drain extension database runtime lease: %w", err)
	}
	return next, nil
}

func scanExtensionDatabaseRuntimeLease(row pgx.Row) (ExtensionDatabaseRuntimeLeaseSnapshot, error) {
	var lease ExtensionDatabaseRuntimeLeaseSnapshot
	var drainingAt, revokedAt *time.Time
	err := row.Scan(
		&lease.ID, &lease.LeaseID, &lease.GrantID, &lease.Artifact.ExtensionID,
		&lease.Artifact.VersionID, &lease.Artifact.Version, &lease.Artifact.PackageDigest,
		&lease.RuntimeInstanceID, &lease.RoleName, &lease.Status, &lease.IssuerKind,
		&lease.IssuedByUserID, &lease.IssueAuditEventID, &lease.IssuedAt,
		&lease.LastHeartbeatAt, &lease.ExpiresAt, &drainingAt, &revokedAt,
		&lease.FailureCode, &lease.Revision,
	)
	if err != nil {
		return ExtensionDatabaseRuntimeLeaseSnapshot{}, err
	}
	lease.IssuedAt = lease.IssuedAt.UTC()
	lease.LastHeartbeatAt = lease.LastHeartbeatAt.UTC()
	lease.ExpiresAt = lease.ExpiresAt.UTC()
	lease.DrainingAt = utcExtensionDatabaseLeaseTime(drainingAt)
	lease.RevokedAt = utcExtensionDatabaseLeaseTime(revokedAt)
	return lease, nil
}

func nullableExtensionDatabaseLeaseActor(authority ExtensionDatabaseLeaseAuthority) any {
	if authority.Kind == ExtensionDatabaseLeaseIssuerActor {
		return authority.ActorUserID
	}
	return nil
}

func utcExtensionDatabaseLeaseTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func isPostgresConstraintError(err error) bool {
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return false
	}
	switch postgresError.Code {
	case "23503", "23505", "23514":
		return true
	default:
		return false
	}
}

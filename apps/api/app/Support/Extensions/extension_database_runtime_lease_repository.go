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

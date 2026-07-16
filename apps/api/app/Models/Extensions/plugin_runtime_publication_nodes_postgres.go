package extensions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresStore) RegisterPluginRuntimeNode(
	ctx context.Context,
	identity PluginRuntimeNodeIdentity,
	lease time.Duration,
) (PluginRuntimeNode, error) {
	if s == nil || s.pool == nil || ctx == nil ||
		!validPluginRuntimeNodeIdentity(identity) || !validPluginRuntimeNodeLease(lease) {
		return PluginRuntimeNode{}, ErrPluginRuntimeNodeInvalid
	}
	node, err := scanPluginRuntimeNode(s.pool.QueryRow(ctx, `
		INSERT INTO plugin_runtime_nodes (
			node_id, process_role, boot_id, last_seen_at, lease_expires_at
		) VALUES (
			$1, $2, $3, statement_timestamp(),
			statement_timestamp() + ($4 * interval '1 millisecond')
		)
		ON CONFLICT (node_id, process_role, boot_id) DO UPDATE
		SET last_seen_at = statement_timestamp(),
		    lease_expires_at = statement_timestamp() + ($4 * interval '1 millisecond')
		WHERE plugin_runtime_nodes.lease_expires_at > statement_timestamp()
		RETURNING node_id, process_role, boot_id, last_applied_revision,
		          first_seen_at, last_seen_at, lease_expires_at
	`, identity.NodeID, identity.ProcessRole, identity.BootID, lease.Milliseconds()))
	if errors.Is(err, pgx.ErrNoRows) {
		return PluginRuntimeNode{}, ErrPluginRuntimeNodeLeaseLost
	}
	return node, err
}

func (s *PostgresStore) HeartbeatPluginRuntimeNode(
	ctx context.Context,
	identity PluginRuntimeNodeIdentity,
	lease time.Duration,
) (PluginRuntimeNode, error) {
	if s == nil || s.pool == nil || ctx == nil ||
		!validPluginRuntimeNodeIdentity(identity) || !validPluginRuntimeNodeLease(lease) {
		return PluginRuntimeNode{}, ErrPluginRuntimeNodeInvalid
	}
	node, err := scanPluginRuntimeNode(s.pool.QueryRow(ctx, `
		UPDATE plugin_runtime_nodes
		SET last_seen_at = statement_timestamp(),
		    lease_expires_at = statement_timestamp() + ($4 * interval '1 millisecond')
		WHERE node_id = $1 AND process_role = $2 AND boot_id = $3
		  AND lease_expires_at > statement_timestamp()
		RETURNING node_id, process_role, boot_id, last_applied_revision,
		          first_seen_at, last_seen_at, lease_expires_at
	`, identity.NodeID, identity.ProcessRole, identity.BootID, lease.Milliseconds()))
	if errors.Is(err, pgx.ErrNoRows) {
		return PluginRuntimeNode{}, ErrPluginRuntimeNodeLeaseLost
	}
	return node, err
}

func (s *PostgresStore) GetPluginRuntimeNode(
	ctx context.Context,
	identity PluginRuntimeNodeIdentity,
) (PluginRuntimeNode, error) {
	if s == nil || s.pool == nil || ctx == nil || !validPluginRuntimeNodeIdentity(identity) {
		return PluginRuntimeNode{}, ErrPluginRuntimeNodeInvalid
	}
	node, err := scanPluginRuntimeNode(s.pool.QueryRow(ctx, `
		SELECT node_id, process_role, boot_id, last_applied_revision,
		       first_seen_at, last_seen_at, lease_expires_at
		FROM plugin_runtime_nodes
		WHERE node_id = $1 AND process_role = $2 AND boot_id = $3
		  AND lease_expires_at > statement_timestamp()
	`, identity.NodeID, identity.ProcessRole, identity.BootID))
	if errors.Is(err, pgx.ErrNoRows) {
		return PluginRuntimeNode{}, ErrPluginRuntimeNodeLeaseLost
	}
	return node, err
}

func (s *PostgresStore) BeginPluginRuntimePublicationApply(
	ctx context.Context,
	identity PluginRuntimeNodeIdentity,
	publicationRevision int64,
) (PluginRuntimePublicationAck, error) {
	if s == nil || s.pool == nil || ctx == nil ||
		!validPluginRuntimeNodeIdentity(identity) || publicationRevision <= 0 {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PluginRuntimePublicationAck{}, fmt.Errorf("begin plugin runtime acknowledgement: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	node, err := lockLivePluginRuntimeNode(ctx, tx, identity)
	if err != nil {
		return PluginRuntimePublicationAck{}, err
	}
	if publicationRevision <= node.LastAppliedRevision {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM plugin_runtime_publications WHERE revision = $1
		)
	`, publicationRevision).Scan(&exists); err != nil {
		return PluginRuntimePublicationAck{}, err
	}
	if !exists {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}

	ack, err := scanPluginRuntimePublicationAck(tx.QueryRow(ctx, `
		INSERT INTO plugin_runtime_publication_acks (
			publication_revision, node_id, process_role, boot_id, status
		) VALUES ($1, $2, $3, $4, 'applying')
		ON CONFLICT (publication_revision, node_id, process_role, boot_id) DO UPDATE
		SET status = 'applying', applied_member_count = NULL,
		    applied_members_digest = NULL, error_reason = '', applied_at = NULL,
		    attempt_count = plugin_runtime_publication_acks.attempt_count + 1,
		    revision = plugin_runtime_publication_acks.revision + 1,
		    started_at = statement_timestamp(), updated_at = statement_timestamp()
		WHERE plugin_runtime_publication_acks.status = 'failed'
		RETURNING publication_revision, node_id, process_role, boot_id, status,
		          applied_member_count, applied_members_digest, error_reason,
		          attempt_count, revision, started_at, updated_at, applied_at
	`, publicationRevision, identity.NodeID, identity.ProcessRole, identity.BootID))
	if errors.Is(err, pgx.ErrNoRows) {
		ack, err = loadPluginRuntimePublicationAck(ctx, tx, identity, publicationRevision, true)
	}
	if err != nil {
		return PluginRuntimePublicationAck{}, fmt.Errorf(
			"write plugin runtime acknowledgement: %w",
			mapPluginRuntimePostgresError(err, ErrPluginRuntimeAckConflict),
		)
	}
	if ack.Status != PluginRuntimeAckApplying {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return PluginRuntimePublicationAck{}, fmt.Errorf(
			"commit plugin runtime acknowledgement: %w",
			mapPluginRuntimePostgresError(err, ErrPluginRuntimeAckConflict),
		)
	}
	return ack, nil
}

func (s *PostgresStore) CompletePluginRuntimePublicationApply(
	ctx context.Context,
	identity PluginRuntimeNodeIdentity,
	publication PluginRuntimePublication,
	expectedAckRevision int64,
	appliedMembers []PluginRuntimeAppliedMember,
) (PluginRuntimePublicationAck, error) {
	publication, publicationErr := normalizedPluginRuntimePublication(publication)
	if s == nil || s.pool == nil || ctx == nil || !validPluginRuntimeNodeIdentity(identity) ||
		expectedAckRevision <= 0 || publicationErr != nil {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}
	canonicalApplied, appliedDigest, err := canonicalPluginRuntimeAppliedMembers(publication.Members, appliedMembers)
	if err != nil || appliedDigest != publication.MembersDigest {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PluginRuntimePublicationAck{}, fmt.Errorf("begin complete plugin runtime acknowledgement: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	node, err := lockLivePluginRuntimeNode(ctx, tx, identity)
	if err != nil {
		return PluginRuntimePublicationAck{}, err
	}
	if publication.Revision <= node.LastAppliedRevision {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}
	stored, err := loadPluginRuntimePublication(
		ctx, tx, pluginRuntimePublicationSelect+` WHERE revision = $1`, publication.Revision,
	)
	if errors.Is(err, ErrPluginRuntimePublicationNotFound) {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}
	if err != nil {
		return PluginRuntimePublicationAck{}, fmt.Errorf("load plugin runtime publication: %w", err)
	}
	if !samePluginRuntimePublication(stored, publication) {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}

	ack, err := scanPluginRuntimePublicationAck(tx.QueryRow(ctx, `
		UPDATE plugin_runtime_publication_acks
		SET status = 'applied', applied_member_count = $5,
		    applied_members_digest = $6, error_reason = '',
		    applied_at = statement_timestamp(), updated_at = statement_timestamp(),
		    revision = revision + 1
		WHERE publication_revision = $1 AND node_id = $2
		  AND process_role = $3 AND boot_id = $4
		  AND status = 'applying' AND revision = $7
		RETURNING publication_revision, node_id, process_role, boot_id, status,
		          applied_member_count, applied_members_digest, error_reason,
		          attempt_count, revision, started_at, updated_at, applied_at
	`, publication.Revision, identity.NodeID, identity.ProcessRole, identity.BootID,
		len(canonicalApplied), appliedDigest, expectedAckRevision))
	if errors.Is(err, pgx.ErrNoRows) {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}
	if err != nil {
		return PluginRuntimePublicationAck{}, fmt.Errorf(
			"complete plugin runtime acknowledgement: %w",
			mapPluginRuntimePostgresError(err, ErrPluginRuntimeAckConflict),
		)
	}
	for _, member := range canonicalApplied {
		if _, err := tx.Exec(ctx, `
			INSERT INTO plugin_runtime_applied_members (
				publication_revision, node_id, process_role, boot_id,
				extension_id, extension_version_id, extension_version,
				package_digest, runtime_instance_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, publication.Revision, identity.NodeID, identity.ProcessRole, identity.BootID,
			member.ExtensionID, member.ExtensionVersionID, member.ExtensionVersion,
			member.PackageDigest, member.RuntimeInstanceID); err != nil {
			return PluginRuntimePublicationAck{}, fmt.Errorf(
				"insert applied plugin runtime member: %w",
				mapPluginRuntimePostgresError(err, ErrPluginRuntimeAckConflict),
			)
		}
	}
	tag, err := tx.Exec(ctx, `
		UPDATE plugin_runtime_nodes
		SET last_applied_revision = $4
		WHERE node_id = $1 AND process_role = $2 AND boot_id = $3
		  AND last_applied_revision < $4
		  AND lease_expires_at > statement_timestamp()
	`, identity.NodeID, identity.ProcessRole, identity.BootID, publication.Revision)
	if err != nil {
		return PluginRuntimePublicationAck{}, fmt.Errorf(
			"advance plugin runtime node: %w",
			mapPluginRuntimePostgresError(err, ErrPluginRuntimeAckConflict),
		)
	}
	if tag.RowsAffected() != 1 {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeNodeLeaseLost
	}
	if err := tx.Commit(ctx); err != nil {
		return PluginRuntimePublicationAck{}, fmt.Errorf(
			"commit completed plugin runtime acknowledgement: %w",
			mapPluginRuntimePostgresError(err, ErrPluginRuntimeAckConflict),
		)
	}
	return ack, nil
}

func (s *PostgresStore) FailPluginRuntimePublicationApply(
	ctx context.Context,
	identity PluginRuntimeNodeIdentity,
	publicationRevision int64,
	expectedAckRevision int64,
	reason string,
) (PluginRuntimePublicationAck, error) {
	reason = strings.TrimSpace(reason)
	if s == nil || s.pool == nil || ctx == nil || !validPluginRuntimeNodeIdentity(identity) ||
		publicationRevision <= 0 || expectedAckRevision <= 0 || reason == "" || len([]byte(reason)) > 2048 {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PluginRuntimePublicationAck{}, fmt.Errorf("begin failed plugin runtime acknowledgement: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := lockLivePluginRuntimeNode(ctx, tx, identity); err != nil {
		return PluginRuntimePublicationAck{}, err
	}
	ack, err := scanPluginRuntimePublicationAck(tx.QueryRow(ctx, `
		UPDATE plugin_runtime_publication_acks
		SET status = 'failed', applied_member_count = NULL,
		    applied_members_digest = NULL, error_reason = $5, applied_at = NULL,
		    updated_at = statement_timestamp(), revision = revision + 1
		WHERE publication_revision = $1 AND node_id = $2
		  AND process_role = $3 AND boot_id = $4
		  AND status = 'applying' AND revision = $6
		RETURNING publication_revision, node_id, process_role, boot_id, status,
		          applied_member_count, applied_members_digest, error_reason,
		          attempt_count, revision, started_at, updated_at, applied_at
	`, publicationRevision, identity.NodeID, identity.ProcessRole, identity.BootID,
		reason, expectedAckRevision))
	if errors.Is(err, pgx.ErrNoRows) {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}
	if err != nil {
		return PluginRuntimePublicationAck{}, fmt.Errorf(
			"fail plugin runtime acknowledgement: %w",
			mapPluginRuntimePostgresError(err, ErrPluginRuntimeAckConflict),
		)
	}
	if err := tx.Commit(ctx); err != nil {
		return PluginRuntimePublicationAck{}, fmt.Errorf(
			"commit failed plugin runtime acknowledgement: %w",
			mapPluginRuntimePostgresError(err, ErrPluginRuntimeAckConflict),
		)
	}
	return ack, nil
}

func lockLivePluginRuntimeNode(
	ctx context.Context,
	tx pgx.Tx,
	identity PluginRuntimeNodeIdentity,
) (PluginRuntimeNode, error) {
	node, err := scanPluginRuntimeNode(tx.QueryRow(ctx, `
		SELECT node_id, process_role, boot_id, last_applied_revision,
		       first_seen_at, last_seen_at, lease_expires_at
		FROM plugin_runtime_nodes
		WHERE node_id = $1 AND process_role = $2 AND boot_id = $3
		  AND lease_expires_at > statement_timestamp()
		FOR UPDATE
	`, identity.NodeID, identity.ProcessRole, identity.BootID))
	if errors.Is(err, pgx.ErrNoRows) {
		return PluginRuntimeNode{}, ErrPluginRuntimeNodeLeaseLost
	}
	return node, err
}

func loadPluginRuntimePublicationAck(
	ctx context.Context,
	query pluginRuntimePublicationQuerier,
	identity PluginRuntimeNodeIdentity,
	publicationRevision int64,
	lock bool,
) (PluginRuntimePublicationAck, error) {
	suffix := ""
	if lock {
		suffix = " FOR UPDATE"
	}
	return scanPluginRuntimePublicationAck(query.QueryRow(ctx, `
		SELECT publication_revision, node_id, process_role, boot_id, status,
		       applied_member_count, applied_members_digest, error_reason,
		       attempt_count, revision, started_at, updated_at, applied_at
		FROM plugin_runtime_publication_acks
		WHERE publication_revision = $1 AND node_id = $2
		  AND process_role = $3 AND boot_id = $4
	`+suffix, publicationRevision, identity.NodeID, identity.ProcessRole, identity.BootID))
}

type pluginRuntimeNodeRow interface {
	Scan(...any) error
}

func scanPluginRuntimeNode(row pluginRuntimeNodeRow) (PluginRuntimeNode, error) {
	var node PluginRuntimeNode
	err := row.Scan(
		&node.NodeID,
		&node.ProcessRole,
		&node.BootID,
		&node.LastAppliedRevision,
		&node.FirstSeenAt,
		&node.LastSeenAt,
		&node.LeaseExpiresAt,
	)
	return node, err
}

func scanPluginRuntimePublicationAck(row pluginRuntimeNodeRow) (PluginRuntimePublicationAck, error) {
	var ack PluginRuntimePublicationAck
	var appliedCount sql.NullInt64
	var appliedDigest sql.NullString
	var appliedAt sql.NullTime
	err := row.Scan(
		&ack.PublicationRevision,
		&ack.NodeID,
		&ack.ProcessRole,
		&ack.BootID,
		&ack.Status,
		&appliedCount,
		&appliedDigest,
		&ack.ErrorReason,
		&ack.AttemptCount,
		&ack.Revision,
		&ack.StartedAt,
		&ack.UpdatedAt,
		&appliedAt,
	)
	if appliedCount.Valid {
		value := int(appliedCount.Int64)
		ack.AppliedMemberCount = &value
	}
	ack.AppliedMembersDigest = appliedDigest.String
	if appliedAt.Valid {
		value := appliedAt.Time
		ack.AppliedAt = &value
	}
	return ack, err
}

var _ PluginRuntimeNodeRepository = (*PostgresStore)(nil)

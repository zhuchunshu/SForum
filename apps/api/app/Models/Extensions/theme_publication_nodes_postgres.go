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

const maxThemeRuntimeNodeLease = 24 * time.Hour

func (s *PostgresStore) RegisterThemeRuntimeNode(
	ctx context.Context,
	identity ThemeRuntimeNodeIdentity,
	lease time.Duration,
) (ThemeRuntimeNode, error) {
	if s == nil || s.pool == nil || ctx == nil || !validThemeRuntimeNodeIdentity(identity) || !validThemeRuntimeNodeLease(lease) {
		return ThemeRuntimeNode{}, ErrThemeRuntimeNodeInvalid
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO theme_runtime_nodes (
			node_id, boot_id, last_seen_at, lease_expires_at
		) VALUES ($1, $2, statement_timestamp(), statement_timestamp() + ($3 * interval '1 millisecond'))
		ON CONFLICT (node_id, boot_id) DO UPDATE
		SET last_seen_at = statement_timestamp(),
		    lease_expires_at = statement_timestamp() + ($3 * interval '1 millisecond')
		WHERE theme_runtime_nodes.lease_expires_at > statement_timestamp()
		RETURNING node_id, boot_id, last_applied_revision,
		          first_seen_at, last_seen_at, lease_expires_at
	`, identity.NodeID, identity.BootID, lease.Milliseconds())
	node, err := scanThemeRuntimeNode(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ThemeRuntimeNode{}, ErrThemeRuntimeNodeLeaseLost
	}
	return node, err
}

func (s *PostgresStore) HeartbeatThemeRuntimeNode(
	ctx context.Context,
	identity ThemeRuntimeNodeIdentity,
	lease time.Duration,
) (ThemeRuntimeNode, error) {
	if s == nil || s.pool == nil || ctx == nil || !validThemeRuntimeNodeIdentity(identity) || !validThemeRuntimeNodeLease(lease) {
		return ThemeRuntimeNode{}, ErrThemeRuntimeNodeInvalid
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE theme_runtime_nodes
		SET last_seen_at = statement_timestamp(),
		    lease_expires_at = statement_timestamp() + ($3 * interval '1 millisecond')
		WHERE node_id = $1 AND boot_id = $2
		  AND lease_expires_at > statement_timestamp()
		RETURNING node_id, boot_id, last_applied_revision,
		          first_seen_at, last_seen_at, lease_expires_at
	`, identity.NodeID, identity.BootID, lease.Milliseconds())
	node, err := scanThemeRuntimeNode(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ThemeRuntimeNode{}, ErrThemeRuntimeNodeLeaseLost
	}
	return node, err
}

func (s *PostgresStore) GetThemeRuntimeNode(
	ctx context.Context,
	identity ThemeRuntimeNodeIdentity,
) (ThemeRuntimeNode, error) {
	if s == nil || s.pool == nil || ctx == nil || !validThemeRuntimeNodeIdentity(identity) {
		return ThemeRuntimeNode{}, ErrThemeRuntimeNodeInvalid
	}
	node, err := scanThemeRuntimeNode(s.pool.QueryRow(ctx, `
		SELECT node_id, boot_id, last_applied_revision,
		       first_seen_at, last_seen_at, lease_expires_at
		FROM theme_runtime_nodes WHERE node_id = $1 AND boot_id = $2
	`, identity.NodeID, identity.BootID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ThemeRuntimeNode{}, ErrThemeRuntimeNodeLeaseLost
	}
	return node, err
}

func (s *PostgresStore) BeginThemeRuntimePublicationApply(
	ctx context.Context,
	identity ThemeRuntimeNodeIdentity,
	publicationRevision int64,
) (ThemeRuntimePublicationAck, error) {
	if s == nil || s.pool == nil || ctx == nil || !validThemeRuntimeNodeIdentity(identity) || publicationRevision <= 0 {
		return ThemeRuntimePublicationAck{}, ErrThemeRuntimeAckConflict
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ThemeRuntimePublicationAck{}, fmt.Errorf("begin theme runtime acknowledgement: %w", err)
	}
	defer tx.Rollback(ctx)
	node, err := lockLiveThemeRuntimeNode(ctx, tx, identity)
	if err != nil {
		return ThemeRuntimePublicationAck{}, err
	}
	if publicationRevision <= node.LastAppliedRevision {
		return ThemeRuntimePublicationAck{}, ErrThemeRuntimeAckConflict
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM theme_runtime_publications WHERE revision = $1
	)`, publicationRevision).Scan(&exists); err != nil || !exists {
		return ThemeRuntimePublicationAck{}, ErrThemeRuntimeAckConflict
	}
	ack, err := scanThemeRuntimePublicationAck(tx.QueryRow(ctx, `
		INSERT INTO theme_runtime_publication_acks (
			publication_revision, node_id, boot_id, status
		) VALUES ($1, $2, $3, 'applying')
		ON CONFLICT (publication_revision, node_id, boot_id) DO UPDATE
		SET status = 'applying', applied_state = NULL,
		    applied_theme_id = NULL, applied_theme_version = NULL,
		    applied_package_digest = NULL, error_reason = '',
		    applied_at = NULL, attempt_count = theme_runtime_publication_acks.attempt_count + 1,
		    revision = theme_runtime_publication_acks.revision + 1,
		    started_at = statement_timestamp(), updated_at = statement_timestamp()
		WHERE theme_runtime_publication_acks.status = 'failed'
		RETURNING publication_revision, node_id, boot_id, status,
		          applied_state, applied_theme_id, applied_theme_version, applied_package_digest,
		          error_reason, attempt_count, revision, started_at, updated_at, applied_at
	`, publicationRevision, identity.NodeID, identity.BootID))
	if errors.Is(err, pgx.ErrNoRows) {
		ack, err = loadThemeRuntimePublicationAck(ctx, tx, identity, publicationRevision, true)
	}
	if err != nil {
		return ThemeRuntimePublicationAck{}, err
	}
	if ack.Status == ThemeRuntimeAckApplied {
		return ThemeRuntimePublicationAck{}, ErrThemeRuntimeAckConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return ThemeRuntimePublicationAck{}, fmt.Errorf("commit theme runtime acknowledgement: %w", err)
	}
	return ack, nil
}

func (s *PostgresStore) CompleteThemeRuntimePublicationApply(
	ctx context.Context,
	identity ThemeRuntimeNodeIdentity,
	publication ThemeRuntimePublication,
	expectedAckRevision int64,
) (ThemeRuntimePublicationAck, error) {
	if s == nil || s.pool == nil || ctx == nil || !validThemeRuntimeNodeIdentity(identity) ||
		publication.Revision <= 0 || expectedAckRevision <= 0 || !validThemeRuntimePublication(publication) {
		return ThemeRuntimePublicationAck{}, ErrThemeRuntimeAckConflict
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ThemeRuntimePublicationAck{}, fmt.Errorf("begin complete theme runtime acknowledgement: %w", err)
	}
	defer tx.Rollback(ctx)
	node, err := lockLiveThemeRuntimeNode(ctx, tx, identity)
	if err != nil {
		return ThemeRuntimePublicationAck{}, err
	}
	if publication.Revision <= node.LastAppliedRevision {
		return ThemeRuntimePublicationAck{}, ErrThemeRuntimeAckConflict
	}
	stored, err := loadThemeRuntimePublication(ctx, tx, publication.Revision, true)
	if err != nil || !sameThemeRuntimePublication(stored, publication) {
		return ThemeRuntimePublicationAck{}, ErrThemeRuntimeAckConflict
	}
	ack, err := scanThemeRuntimePublicationAck(tx.QueryRow(ctx, `
		UPDATE theme_runtime_publication_acks
		SET status = 'applied', applied_state = $5,
		    applied_theme_id = $6, applied_theme_version = $7,
		    applied_package_digest = $8, error_reason = '',
		    applied_at = statement_timestamp(), updated_at = statement_timestamp(),
		    revision = revision + 1
		WHERE publication_revision = $1 AND node_id = $2 AND boot_id = $3
		  AND status = 'applying' AND revision = $4
		RETURNING publication_revision, node_id, boot_id, status,
		          applied_state, applied_theme_id, applied_theme_version, applied_package_digest,
		          error_reason, attempt_count, revision, started_at, updated_at, applied_at
	`, publication.Revision, identity.NodeID, identity.BootID, expectedAckRevision,
		publication.DesiredState,
		nullableThemePublicationValue(publication.ThemeID), nullableThemePublicationValue(publication.ThemeVersion),
		nullableThemePublicationValue(publication.PackageDigest)))
	if errors.Is(err, pgx.ErrNoRows) {
		return ThemeRuntimePublicationAck{}, ErrThemeRuntimeAckConflict
	}
	if err != nil {
		return ThemeRuntimePublicationAck{}, err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE theme_runtime_nodes
		SET last_applied_revision = $3
		WHERE node_id = $1 AND boot_id = $2
		  AND last_applied_revision < $3
	`, identity.NodeID, identity.BootID, publication.Revision)
	if err != nil || tag.RowsAffected() != 1 {
		return ThemeRuntimePublicationAck{}, ErrThemeRuntimeAckConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return ThemeRuntimePublicationAck{}, fmt.Errorf("commit completed theme runtime acknowledgement: %w", err)
	}
	return ack, nil
}

func (s *PostgresStore) FailThemeRuntimePublicationApply(
	ctx context.Context,
	identity ThemeRuntimeNodeIdentity,
	publicationRevision int64,
	expectedAckRevision int64,
	reason string,
) (ThemeRuntimePublicationAck, error) {
	reason = strings.TrimSpace(reason)
	if s == nil || s.pool == nil || ctx == nil || !validThemeRuntimeNodeIdentity(identity) ||
		publicationRevision <= 0 || expectedAckRevision <= 0 || reason == "" || len([]byte(reason)) > 2048 {
		return ThemeRuntimePublicationAck{}, ErrThemeRuntimeAckConflict
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ThemeRuntimePublicationAck{}, fmt.Errorf("begin failed theme runtime acknowledgement: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := lockLiveThemeRuntimeNode(ctx, tx, identity); err != nil {
		return ThemeRuntimePublicationAck{}, err
	}
	ack, err := scanThemeRuntimePublicationAck(tx.QueryRow(ctx, `
		UPDATE theme_runtime_publication_acks
		SET status = 'failed', applied_state = NULL,
		    applied_theme_id = NULL, applied_theme_version = NULL,
		    applied_package_digest = NULL, error_reason = $5,
		    applied_at = NULL, updated_at = statement_timestamp(), revision = revision + 1
		WHERE publication_revision = $1 AND node_id = $2 AND boot_id = $3
		  AND status = 'applying' AND revision = $4
		RETURNING publication_revision, node_id, boot_id, status,
		          applied_state, applied_theme_id, applied_theme_version, applied_package_digest,
		          error_reason, attempt_count, revision, started_at, updated_at, applied_at
	`, publicationRevision, identity.NodeID, identity.BootID, expectedAckRevision, reason))
	if errors.Is(err, pgx.ErrNoRows) {
		return ThemeRuntimePublicationAck{}, ErrThemeRuntimeAckConflict
	}
	if err != nil {
		return ThemeRuntimePublicationAck{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ThemeRuntimePublicationAck{}, fmt.Errorf("commit failed theme runtime acknowledgement: %w", err)
	}
	return ack, nil
}

func loadThemeRuntimePublication(
	ctx context.Context,
	query themePublicationQuerier,
	revision int64,
	lock bool,
) (ThemeRuntimePublication, error) {
	suffix := ` WHERE revision = $1`
	if lock {
		suffix += ` FOR UPDATE`
	}
	return scanThemeRuntimePublication(query.QueryRow(ctx, themeRuntimePublicationSelect+suffix, revision))
}

func lockLiveThemeRuntimeNode(
	ctx context.Context,
	tx pgx.Tx,
	identity ThemeRuntimeNodeIdentity,
) (ThemeRuntimeNode, error) {
	node, err := scanThemeRuntimeNode(tx.QueryRow(ctx, `
		SELECT node_id, boot_id, last_applied_revision,
		       first_seen_at, last_seen_at, lease_expires_at
		FROM theme_runtime_nodes
		WHERE node_id = $1 AND boot_id = $2
		  AND lease_expires_at > statement_timestamp()
		FOR UPDATE
	`, identity.NodeID, identity.BootID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ThemeRuntimeNode{}, ErrThemeRuntimeNodeLeaseLost
	}
	return node, err
}

type themeRuntimeNodeRow interface {
	Scan(...any) error
}

func scanThemeRuntimeNode(row themeRuntimeNodeRow) (ThemeRuntimeNode, error) {
	var node ThemeRuntimeNode
	err := row.Scan(
		&node.NodeID, &node.BootID, &node.LastAppliedRevision,
		&node.FirstSeenAt, &node.LastSeenAt, &node.LeaseExpiresAt,
	)
	return node, err
}

func loadThemeRuntimePublicationAck(
	ctx context.Context,
	query themePublicationQuerier,
	identity ThemeRuntimeNodeIdentity,
	publicationRevision int64,
	lock bool,
) (ThemeRuntimePublicationAck, error) {
	suffix := ``
	if lock {
		suffix = ` FOR UPDATE`
	}
	return scanThemeRuntimePublicationAck(query.QueryRow(ctx, `
		SELECT publication_revision, node_id, boot_id, status,
		       applied_state, applied_theme_id, applied_theme_version, applied_package_digest,
		       error_reason, attempt_count, revision, started_at, updated_at, applied_at
		FROM theme_runtime_publication_acks
		WHERE publication_revision = $1 AND node_id = $2 AND boot_id = $3
	`+suffix, publicationRevision, identity.NodeID, identity.BootID))
}

func scanThemeRuntimePublicationAck(row themeRuntimeNodeRow) (ThemeRuntimePublicationAck, error) {
	var ack ThemeRuntimePublicationAck
	var state, themeID, version, digest sql.NullString
	var appliedAt sql.NullTime
	err := row.Scan(
		&ack.PublicationRevision, &ack.NodeID, &ack.BootID, &ack.Status,
		&state, &themeID, &version, &digest,
		&ack.ErrorReason, &ack.AttemptCount, &ack.Revision,
		&ack.StartedAt, &ack.UpdatedAt, &appliedAt,
	)
	ack.AppliedState = ThemeRuntimePublicationState(state.String)
	ack.AppliedThemeID, ack.AppliedThemeVersion, ack.AppliedPackageDigest = themeID.String, version.String, digest.String
	if appliedAt.Valid {
		value := appliedAt.Time
		ack.AppliedAt = &value
	}
	return ack, err
}

func validThemeRuntimeNodeLease(value time.Duration) bool {
	return value > 0 && value <= maxThemeRuntimeNodeLease && value.Milliseconds() > 0
}

var _ ThemeRuntimeNodeRepository = (*PostgresStore)(nil)

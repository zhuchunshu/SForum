package extensions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresDevTrustSessionStore 是 DevTrustSessionStore 的生产实现。
// 活跃判定（revoked_at IS NULL AND expires_at > now()）由部分唯一索引
// extension_dev_trust_sessions_active_slug_idx 在数据库层强制一包一会话。
type PostgresDevTrustSessionStore struct {
	pool *pgxpool.Pool
}

func NewPostgresDevTrustSessionStore(pool *pgxpool.Pool) *PostgresDevTrustSessionStore {
	return &PostgresDevTrustSessionStore{pool: pool}
}

const devTrustSessionColumns = `id, extension_id, package_slug, root_path, actor_user_id,
	granted_at, expires_at, last_package_digest, revoked_at, revoked_reason`

func scanDevTrustSession(row pgx.Row) (DevTrustSession, error) {
	var session DevTrustSession
	err := row.Scan(&session.ID, &session.ExtensionID, &session.PackageSlug, &session.RootPath,
		&session.ActorUserID, &session.GrantedAt, &session.ExpiresAt, &session.LastPackageDigest,
		&session.RevokedAt, &session.RevokedReason)
	if errors.Is(err, pgx.ErrNoRows) {
		return DevTrustSession{}, ErrDevTrustSessionNotFound
	}
	if err != nil {
		return DevTrustSession{}, fmt.Errorf("scan dev trust session: %w", err)
	}
	return session, nil
}

func (s *PostgresDevTrustSessionStore) Create(ctx context.Context, session DevTrustSession) (DevTrustSession, error) {
	// 唯一索引只锚定 revoked_at IS NULL；过期未撤销的行在这里显式回收，
	// 之后剩下的未撤销行只能是真正的活跃会话（触发 ErrDevTrustSessionExists）。
	if _, err := s.pool.Exec(ctx, `
		UPDATE extension_dev_trust_sessions
		SET revoked_at = now(), revoked_reason = 'expired superseded by a new session'
		WHERE package_slug = $1 AND revoked_at IS NULL AND expires_at <= now()`,
		session.PackageSlug); err != nil {
		return DevTrustSession{}, fmt.Errorf("expire stale dev trust sessions: %w", err)
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO extension_dev_trust_sessions (
			id, extension_id, package_slug, root_path, actor_user_id,
			granted_at, expires_at, last_package_digest
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT DO NOTHING
		RETURNING `+devTrustSessionColumns,
		session.ID, session.ExtensionID, session.PackageSlug, session.RootPath,
		session.ActorUserID, session.GrantedAt, session.ExpiresAt, session.LastPackageDigest)
	created, err := scanDevTrustSession(row)
	if errors.Is(err, ErrDevTrustSessionNotFound) {
		// 唯一活跃索引冲突：同包已存在活跃会话。
		return DevTrustSession{}, ErrDevTrustSessionExists
	}
	return created, err
}

func (s *PostgresDevTrustSessionStore) List(ctx context.Context, input DevTrustSessionListInput) ([]DevTrustSession, error) {
	query := `SELECT ` + devTrustSessionColumns + ` FROM extension_dev_trust_sessions`
	conditions := []string{}
	args := []any{}
	if strings.TrimSpace(input.ExtensionID) != "" {
		args = append(args, normalizeID(input.ExtensionID))
		conditions = append(conditions, fmt.Sprintf("extension_id = $%d", len(args)))
	}
	if !input.IncludeRevoked {
		conditions = append(conditions, "revoked_at IS NULL AND expires_at > now()")
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY granted_at DESC, id DESC LIMIT 200"
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list dev trust sessions: %w", err)
	}
	defer rows.Close()
	sessions := []DevTrustSession{}
	for rows.Next() {
		var session DevTrustSession
		if err := rows.Scan(&session.ID, &session.ExtensionID, &session.PackageSlug, &session.RootPath,
			&session.ActorUserID, &session.GrantedAt, &session.ExpiresAt, &session.LastPackageDigest,
			&session.RevokedAt, &session.RevokedReason); err != nil {
			return nil, fmt.Errorf("scan dev trust session row: %w", err)
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (s *PostgresDevTrustSessionStore) Active(ctx context.Context, packageSlug string) (DevTrustSession, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT `+devTrustSessionColumns+` FROM extension_dev_trust_sessions
		WHERE package_slug = $1 AND revoked_at IS NULL AND expires_at > now()
		ORDER BY granted_at DESC LIMIT 1`, strings.TrimSpace(packageSlug))
	return scanDevTrustSession(row)
}

func (s *PostgresDevTrustSessionStore) TouchDigest(ctx context.Context, sessionID, packageDigest string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE extension_dev_trust_sessions SET last_package_digest = $2
		WHERE id = $1 AND revoked_at IS NULL AND expires_at > now()`,
		sessionID, strings.TrimSpace(packageDigest))
	if err != nil {
		return fmt.Errorf("touch dev trust session digest: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDevTrustSessionExpired
	}
	return nil
}

func (s *PostgresDevTrustSessionStore) Revoke(ctx context.Context, sessionID string, actorUserID int64, reason string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE extension_dev_trust_sessions
		SET revoked_at = now(), revoked_reason = $3
		WHERE id = $1 AND revoked_at IS NULL`,
		sessionID, actorUserID, strings.TrimSpace(reason))
	if err != nil {
		return fmt.Errorf("revoke dev trust session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDevTrustSessionNotFound
	}
	return nil
}

func (s *PostgresDevTrustSessionStore) RevokeAllForExtension(ctx context.Context, extensionID string, actorUserID int64, reason string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE extension_dev_trust_sessions
		SET revoked_at = now(), revoked_reason = $3
		WHERE extension_id = $1 AND revoked_at IS NULL`,
		normalizeID(extensionID), actorUserID, strings.TrimSpace(reason))
	if err != nil {
		return fmt.Errorf("revoke dev trust sessions for extension: %w", err)
	}
	return nil
}

// RevokeAllActive 供 Safe Mode 启动时一次性吊销（开发装配专用）。
func (s *PostgresDevTrustSessionStore) RevokeAllActive(ctx context.Context, reason string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE extension_dev_trust_sessions
		SET revoked_at = now(), revoked_reason = $1
		WHERE revoked_at IS NULL`, strings.TrimSpace(reason))
	if err != nil {
		return fmt.Errorf("revoke all dev trust sessions: %w", err)
	}
	return nil
}

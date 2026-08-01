package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// 外部认证 Host 存储：
//   - 外部账号注册（无密码凭据行）；
//   - 会话绑定的最近认证标记（敏感操作前置）；
//   - 登录方式计数（last login method 保护）；
//   - 外部认证审计。
//
// 安全约束（见 plans/2026-07-27-github-social-login-builtin-plugin.md）：
//   - 用户可以在没有 user_credentials 行的情况下存在；绝不伪造密码；
//   - password_hash 对存在行始终非 null；
//   - 密码登录对无凭据用户视为 invalid credentials（GetCredentialByLogin
//     inner-join，外部账号不在结果集）；
//   - 最近认证绑定 (user_id, session_fingerprint)，跨 session 隔离；
//   - 删除最后一个可用登录方式必须被拒绝；
//   - 审计 payload 不得包含 raw subject/digest/token/state/verifier/secret。

// SessionFingerprint 对 opaque SID 做不可逆 SHA-256 hex。
// 空 SID 返回空串（调用方必须 fail closed）。
func SessionFingerprint(sid string) string {
	sid = strings.TrimSpace(sid)
	if sid == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(sid))
	return hex.EncodeToString(sum[:])
}

// ExternalAuthAuditInput 是审计写入输入。
type ExternalAuthAuditInput struct {
	ActorUserID int64
	Action      string
	IPAddress   string
	UserAgent   string
	SessionHash string // 已脱敏/哈希后的值；可空
}

// PostgresExternalAuthStore 提供：
//   - HasPasswordCredential / CountActiveExternalLinks（读，可脱离事务）；
//   - MarkSessionRecentlyAuthenticated / IsSessionRecentlyAuthenticated；
//   - RecordExternalAuthAudit（写 audit_events）。
type PostgresExternalAuthStore struct {
	pool *pgxpool.Pool
}

func NewPostgresExternalAuthStore(pool *pgxpool.Pool) *PostgresExternalAuthStore {
	return &PostgresExternalAuthStore{pool: pool}
}

// HasPasswordCredential 返回该用户是否已设置密码凭据。
// external-only 用户没有 credential 行，因此返回 false。
func (s *PostgresExternalAuthStore) HasPasswordCredential(ctx context.Context, userID int64) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM user_credentials WHERE user_id = $1)
	`, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has password credential: %w", err)
	}
	return exists, nil
}

// CountActiveExternalLinks 返回该用户的活跃外部 link 数量。
// provider_id 为空时统计全部 provider；非空时只统计该 provider。
// 用于 last-login-method 保护。
func (s *PostgresExternalAuthStore) CountActiveExternalLinks(ctx context.Context, userID int64, providerID string) (int, error) {
	if strings.TrimSpace(providerID) == "" {
		var count int
		err := s.pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM identity_external_links
			WHERE user_id = $1 AND status = 'active'
		`, userID).Scan(&count)
		return count, err
	}
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM identity_external_links
		WHERE user_id = $1 AND provider_id = $2 AND status = 'active'
	`, userID, providerID).Scan(&count)
	return count, err
}

// MarkSessionRecentlyAuthenticated 写入/更新当前会话的最近认证标记。
// method 必须是 password 或 external（external 必须带 providerID）。
// sessionFingerprint 必须是 64 位小写 hex（SessionFingerprint 输出）。
func (s *PostgresExternalAuthStore) MarkSessionRecentlyAuthenticated(
	ctx context.Context,
	userID int64,
	sessionFingerprint string,
	method, providerID string,
	ttl time.Duration,
) error {
	if userID <= 0 {
		return fmt.Errorf("invalid recent-auth user id")
	}
	sessionFingerprint = strings.ToLower(strings.TrimSpace(sessionFingerprint))
	if !validSessionFingerprint(sessionFingerprint) {
		return fmt.Errorf("invalid recent-auth session fingerprint")
	}
	if method != "password" && method != "external" {
		return fmt.Errorf("invalid recent-auth method: %s", method)
	}
	if method == "external" && strings.TrimSpace(providerID) == "" {
		return fmt.Errorf("external recent-auth requires providerId")
	}
	if ttl <= 0 {
		ttl = RecentAuthDefaultTTL
	}
	expires := time.Now().Add(ttl).UTC()
	var provider any
	if method == "external" {
		provider = providerID
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_recent_auth (
		  user_id, session_fingerprint, auth_method, auth_provider_id,
		  authenticated_at, expires_at, updated_at
		)
		VALUES ($1, $2, $3, $4, now(), $5, now())
		ON CONFLICT (user_id, session_fingerprint) DO UPDATE SET
		  auth_method = EXCLUDED.auth_method,
		  auth_provider_id = EXCLUDED.auth_provider_id,
		  authenticated_at = now(),
		  expires_at = EXCLUDED.expires_at,
		  updated_at = now()
	`, userID, sessionFingerprint, method, provider, expires)
	if err != nil {
		return fmt.Errorf("mark session recently authenticated: %w", err)
	}
	return nil
}

// IsSessionRecentlyAuthenticated 检查指定会话在 TTL 窗口内是否有有效认证记录。
// 空 fingerprint 一律 false（fail closed）。
func (s *PostgresExternalAuthStore) IsSessionRecentlyAuthenticated(
	ctx context.Context,
	userID int64,
	sessionFingerprint string,
) (bool, error) {
	sessionFingerprint = strings.ToLower(strings.TrimSpace(sessionFingerprint))
	if userID <= 0 || !validSessionFingerprint(sessionFingerprint) {
		return false, nil
	}
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM user_recent_auth
		  WHERE user_id = $1
		    AND session_fingerprint = $2
		    AND expires_at > now()
		)
	`, userID, sessionFingerprint).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is session recently authenticated: %w", err)
	}
	return exists, nil
}

// IsUserRecentlyAuthenticated 兼容旧签名：无 session 上下文时 fail closed。
// 生产路径必须使用 IsSessionRecentlyAuthenticated。
func (s *PostgresExternalAuthStore) IsUserRecentlyAuthenticated(ctx context.Context, userID int64) (bool, error) {
	_ = ctx
	_ = userID
	return false, nil
}

// RecordExternalAuthAudit 写入一条 audit_events 记录。
// action 形如 external_login/external_registration/external_link/external_unlink。
// payload 不得包含 raw subject/digest/token/state/verifier/secret。
func (s *PostgresExternalAuthStore) RecordExternalAuthAudit(ctx context.Context, input ExternalAuthAuditInput) error {
	if input.ActorUserID == 0 {
		return fmt.Errorf("actor user id required")
	}
	action := strings.TrimSpace(input.Action)
	if action == "" {
		return fmt.Errorf("audit action required")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_events (user_id, action, ip_address, user_agent, session_hash, created_at)
		VALUES ($1, $2, $3, $4, $5, now())
	`, input.ActorUserID, action, truncate(input.IPAddress, 100), truncate(input.UserAgent, 400), input.SessionHash)
	if err != nil {
		return fmt.Errorf("record external auth audit: %w", err)
	}
	return nil
}

func validSessionFingerprint(value string) bool {
	if len(value) != 64 {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// --- 事务内辅助：tx-aware 函数（供 ExternalAuthService 在同事务里调用） ---

// createUserWithoutCredentialTx 在传入的 pgx.Tx 上创建用户（无凭据），返回基础字段。
func createUserWithoutCredentialTx(ctx context.Context, tx pgx.Tx, input CreateUserInput) (CurrentUser, error) {
	row := tx.QueryRow(ctx, `
		INSERT INTO users (username, username_lower, email, email_lower, display_name, locale, status, is_initial_super_admin, email_verified_at, created_at, updated_at)
		VALUES ($1, lower($1), $2, lower($2), $3, $4, 'active', $5, CASE WHEN $6::boolean THEN now() ELSE NULL END, now(), now())
		RETURNING id, username, display_name, locale, status, is_initial_super_admin, email_verified_at IS NOT NULL
	`, input.Username, input.Email, input.DisplayName, input.Locale, input.IsInitialSuperAdmin, input.EmailVerified)
	var current CurrentUser
	if err := row.Scan(&current.ID, &current.Username, &current.DisplayName, &current.Locale, &current.Status, &current.IsInitialSuperAdmin, &current.EmailVerified); err != nil {
		return CurrentUser{}, fmt.Errorf("create external user in tx: %w", err)
	}
	return current, nil
}

// assignDefaultRoleTx 在事务里把默认角色赋予新用户。
// 必须恰好影响 1 行（一个 is_default 角色）；0 或多于 1 则失败，由调用方回滚。
func assignDefaultRoleTx(ctx context.Context, tx pgx.Tx, userID int64) error {
	tag, err := tx.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id FROM roles WHERE is_default = TRUE
	`, userID)
	if err != nil {
		return fmt.Errorf("assign default role: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrExternalAuthDefaultRoleFailed
	}
	return nil
}

// hasPasswordCredentialTx 在事务里检查密码凭据行是否存在。
func hasPasswordCredentialTx(ctx context.Context, tx pgx.Tx, userID int64) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM user_credentials WHERE user_id = $1)
	`, userID).Scan(&exists)
	return exists, err
}

// countActiveExternalLinksTx 在事务里统计活跃外部 link。
// excludeLinkID > 0 时排除该 link（用于 unlink 前判断“删除后是否仍有方法”）。
func countActiveExternalLinksTx(ctx context.Context, tx pgx.Tx, userID, excludeLinkID int64) (int, error) {
	var count int
	err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM identity_external_links
		WHERE user_id = $1 AND status = 'active' AND ($2 = 0 OR id <> $2)
	`, userID, excludeLinkID).Scan(&count)
	return count, err
}

// truncateForAudit 截断字符串到列长度上限。
func truncateForAudit(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// 为兼容旧名保留（其他文件可能引用 truncate）。
func truncate(s string, max int) string { return truncateForAudit(s, max) }

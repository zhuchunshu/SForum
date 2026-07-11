package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// PasswordResetToken 是 password_reset_tokens 行的领域结构。
type PasswordResetToken struct {
	ID             int64
	UserID         int64
	TokenHash      string
	ExpiresAt      time.Time
	ConsumedAt     *time.Time
	CreatedAt      time.Time
	RequestIPHash  string
}

// CreatePasswordResetToken 插入一条密码重置令牌。
func (s *PostgresStore) CreatePasswordResetToken(ctx context.Context, input CreatePasswordResetTokenInput) (PasswordResetToken, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at, request_ip_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, token_hash, expires_at, consumed_at, created_at, request_ip_hash
	`, input.UserID, input.TokenHash, input.ExpiresAt, input.RequestIPHash)
	token, err := scanPasswordResetToken(row)
	if err != nil {
		return PasswordResetToken{}, fmt.Errorf("create password reset token: %w", err)
	}
	return token, nil
}

// FindValidPasswordResetToken 按令牌哈希查找未消费且未过期的令牌。
func (s *PostgresStore) FindValidPasswordResetToken(ctx context.Context, tokenHash string) (PasswordResetToken, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, consumed_at, created_at, request_ip_hash
		FROM password_reset_tokens
		WHERE token_hash = $1
		  AND consumed_at IS NULL
		  AND expires_at > now()
	`, tokenHash)
	token, err := scanPasswordResetToken(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return PasswordResetToken{}, ErrPasswordResetTokenNotFound
	}
	if err != nil {
		return PasswordResetToken{}, fmt.Errorf("find password reset token: %w", err)
	}
	return token, nil
}

// ConsumePasswordResetToken 原子性地消费令牌并返回 user_id；已消费或不存在返回错误。
func (s *PostgresStore) ConsumePasswordResetToken(ctx context.Context, tokenHash string) (int64, error) {
	var userID int64
	err := s.pool.QueryRow(ctx, `
		UPDATE password_reset_tokens
		SET consumed_at = now()
		WHERE token_hash = $1
		  AND consumed_at IS NULL
		  AND expires_at > now()
		RETURNING user_id
	`, tokenHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrPasswordResetTokenNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("consume password reset token: %w", err)
	}
	return userID, nil
}

// ConfirmPasswordResetAtomic 在同一事务内完成令牌消费、密码更新、token version 递增与会话撤销，
// 避免「令牌已消费但密码未更新」或「密码已改但旧会话仍有效」的中间态。
func (s *PostgresStore) ConfirmPasswordResetAtomic(ctx context.Context, tokenHash string, passwordHash string, revokeReason string) (int64, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin password reset confirm: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID int64
	err = tx.QueryRow(ctx, `
		UPDATE password_reset_tokens
		SET consumed_at = now()
		WHERE token_hash = $1
		  AND consumed_at IS NULL
		  AND expires_at > now()
		RETURNING user_id
	`, tokenHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrPasswordResetTokenNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("consume password reset token: %w", err)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE user_credentials
		SET password_hash = $2, password_changed_at = now(), updated_at = now()
		WHERE user_id = $1
	`, userID, passwordHash)
	if err != nil {
		return 0, fmt.Errorf("update user password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return 0, ErrCredentialNotFound
	}

	if _, err := tx.Exec(ctx, `
		UPDATE users SET current_token_version = current_token_version + 1, updated_at = now()
		WHERE id = $1
	`, userID); err != nil {
		return 0, fmt.Errorf("increment user token version: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE user_sessions
		SET revoked_at = now(), revoke_reason = $2
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID, revokeReason); err != nil {
		return 0, fmt.Errorf("revoke user sessions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit password reset confirm: %w", err)
	}
	return userID, nil
}

// UpdateUserPassword 更新用户凭据哈希。
func (s *PostgresStore) UpdateUserPassword(ctx context.Context, userID int64, passwordHash string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE user_credentials
		SET password_hash = $2, password_changed_at = now(), updated_at = now()
		WHERE user_id = $1
	`, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("update user password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCredentialNotFound
	}
	return nil
}

// GetUserTokenVersion 返回用户当前的令牌版本号，用于会话失效校验（M8）。
func (s *PostgresStore) GetUserTokenVersion(ctx context.Context, userID int64) (int64, error) {
	var version int64
	err := s.pool.QueryRow(ctx, `
		SELECT current_token_version FROM users WHERE id = $1
	`, userID).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrCredentialNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("get user token version: %w", err)
	}
	return version, nil
}

// IncrementUserTokenVersion 递增用户令牌版本号，使该用户所有旧会话失效（M8）。
// 用于密码重置成功后，让攻击者已持有的旧会话不再有效。
func (s *PostgresStore) IncrementUserTokenVersion(ctx context.Context, userID int64) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET current_token_version = current_token_version + 1, updated_at = now()
		WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("increment user token version: %w", err)
	}
	return nil
}

// CreatePasswordResetTokenInput 是创建重置令牌的入参。
type CreatePasswordResetTokenInput struct {
	UserID        int64
	TokenHash     string
	ExpiresAt     time.Time
	RequestIPHash string
}

type passwordResetScanner interface {
	Scan(dest ...any) error
}

func scanPasswordResetToken(row passwordResetScanner) (PasswordResetToken, error) {
	var token PasswordResetToken
	var consumedAt *time.Time
	if err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&consumedAt,
		&token.CreatedAt,
		&token.RequestIPHash,
	); err != nil {
		return PasswordResetToken{}, err
	}
	token.ConsumedAt = consumedAt
	return token, nil
}

package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type EmailVerificationToken struct {
	ID            int64
	UserID        int64
	Email         string
	TokenHash     string
	ExpiresAt     time.Time
	ConsumedAt    *time.Time
	CreatedAt     time.Time
	RequestIPHash string
}

type CreateEmailVerificationTokenInput struct {
	UserID        int64
	Email         string
	TokenHash     string
	ExpiresAt     time.Time
	RequestIPHash string
}

type EmailVerificationTarget struct {
	UserID        int64
	Email         string
	Username      string
	Locale        string
	Status        UserStatus
	EmailVerified bool
}

type EmailVerificationStore interface {
	GetEmailVerificationTarget(context.Context, int64) (EmailVerificationTarget, error)
	CreateEmailVerificationToken(context.Context, CreateEmailVerificationTokenInput) (EmailVerificationToken, error)
	ConfirmEmailVerification(context.Context, string) (int64, error)
	IsEmailVerified(context.Context, int64) (bool, error)
}

func (s *PostgresStore) GetEmailVerificationTarget(ctx context.Context, userID int64) (EmailVerificationTarget, error) {
	var target EmailVerificationTarget
	var status string
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, username, locale, status, email_verified_at IS NOT NULL
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&target.UserID, &target.Email, &target.Username, &target.Locale, &status, &target.EmailVerified,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return EmailVerificationTarget{}, ErrUserNotFound
	}
	if err != nil {
		return EmailVerificationTarget{}, fmt.Errorf("get email verification target: %w", err)
	}
	target.Status = UserStatus(status)
	return target, nil
}

func (s *PostgresStore) CreateEmailVerificationToken(ctx context.Context, input CreateEmailVerificationTokenInput) (EmailVerificationToken, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return EmailVerificationToken{}, fmt.Errorf("begin email verification token: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE email_verification_tokens
		SET consumed_at = now()
		WHERE user_id = $1 AND consumed_at IS NULL
	`, input.UserID); err != nil {
		return EmailVerificationToken{}, fmt.Errorf("expire email verification tokens: %w", err)
	}
	row := tx.QueryRow(ctx, `
		INSERT INTO email_verification_tokens (user_id, email, token_hash, expires_at, request_ip_hash)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, email, token_hash, expires_at, consumed_at, created_at, request_ip_hash
	`, input.UserID, strings.TrimSpace(strings.ToLower(input.Email)), input.TokenHash, input.ExpiresAt, input.RequestIPHash)
	token, err := scanEmailVerificationToken(row)
	if err != nil {
		return EmailVerificationToken{}, fmt.Errorf("create email verification token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return EmailVerificationToken{}, fmt.Errorf("commit email verification token: %w", err)
	}
	return token, nil
}

func (s *PostgresStore) ConfirmEmailVerification(ctx context.Context, tokenHash string) (int64, error) {
	var userID int64
	err := s.pool.QueryRow(ctx, `
		WITH consumed AS (
			UPDATE email_verification_tokens
			SET consumed_at = now()
			WHERE token_hash = $1
			  AND consumed_at IS NULL
			  AND expires_at > now()
			RETURNING user_id, email
		)
		UPDATE users
		SET email_verified_at = COALESCE(email_verified_at, now()), updated_at = now()
		FROM consumed
		WHERE users.id = consumed.user_id
		  AND users.email_lower = lower(consumed.email)
		RETURNING users.id
	`, tokenHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrEmailVerificationTokenNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("confirm email verification: %w", err)
	}
	return userID, nil
}

func (s *PostgresStore) IsEmailVerified(ctx context.Context, userID int64) (bool, error) {
	var verified bool
	err := s.pool.QueryRow(ctx, `
		SELECT email_verified_at IS NOT NULL FROM users WHERE id = $1
	`, userID).Scan(&verified)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrUserNotFound
	}
	if err != nil {
		return false, fmt.Errorf("read email verification state: %w", err)
	}
	return verified, nil
}

type emailVerificationScanner interface {
	Scan(dest ...any) error
}

func scanEmailVerificationToken(row emailVerificationScanner) (EmailVerificationToken, error) {
	var token EmailVerificationToken
	if err := row.Scan(
		&token.ID, &token.UserID, &token.Email, &token.TokenHash, &token.ExpiresAt,
		&token.ConsumedAt, &token.CreatedAt, &token.RequestIPHash,
	); err != nil {
		return EmailVerificationToken{}, err
	}
	return token, nil
}

package apitokens

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type GuardSubject struct {
	TokenID     int64
	OwnerUserID int64
	Exists      bool
	Revoked     bool
}

// LoadGuardSubject 只读取 PAT 管理 Guard 所需的所有权与状态，不加载哈希或 scopes。
func (s *PostgresStore) LoadGuardSubject(ctx context.Context, tokenID int64) (GuardSubject, error) {
	if s == nil || s.pool == nil || ctx == nil || tokenID <= 0 {
		return GuardSubject{}, ErrTokenNotFound
	}
	var subject GuardSubject
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, revoked_at IS NOT NULL
		FROM api_tokens
		WHERE id = $1
	`, tokenID).Scan(&subject.TokenID, &subject.OwnerUserID, &subject.Revoked)
	if errors.Is(err, pgx.ErrNoRows) {
		return GuardSubject{}, ErrTokenNotFound
	}
	if err != nil {
		return GuardSubject{}, err
	}
	subject.Exists = true
	return subject, nil
}

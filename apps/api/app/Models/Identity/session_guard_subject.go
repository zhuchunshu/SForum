package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type SessionGuardSubject struct {
	SID         string
	OwnerUserID int64
	Exists      bool
	Revoked     bool
}

// LoadSessionGuardSubject 是单会话撤销 inherited Guard 的权威所有权查询。
// SID 不存在与不属于当前用户都由 Guard 统一 fail closed，避免跨用户探测。
func (s *PostgresStore) LoadSessionGuardSubject(ctx context.Context, sid string) (SessionGuardSubject, error) {
	sid = strings.TrimSpace(sid)
	if s == nil || s.pool == nil || ctx == nil || sid == "" {
		return SessionGuardSubject{}, ErrSessionNotFound
	}
	var subject SessionGuardSubject
	err := s.pool.QueryRow(ctx, `
		SELECT sid, user_id, revoked_at IS NOT NULL
		FROM user_sessions
		WHERE sid = $1
	`, sid).Scan(&subject.SID, &subject.OwnerUserID, &subject.Revoked)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionGuardSubject{}, ErrSessionNotFound
	}
	if err != nil {
		return SessionGuardSubject{}, fmt.Errorf("load session guard subject: %w", err)
	}
	subject.Exists = true
	return subject, nil
}

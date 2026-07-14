package identity

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// AdminGuardSubject 是 inherited identity admin Guard 判断目标保护等级所需的
// 最小权威视图。它刻意不携带权限或会话数据，避免扩大 Guard 查询面。
type AdminGuardSubject struct {
	UserID              int64
	Exists              bool
	IsInitialSuperAdmin bool
	IsSuperAdmin        bool
}

// LoadAdminGuardSubject 只服务五条低频管理路由。这里选择每次读取 PostgreSQL，
// 因为进程内缓存无法观察另一 API 节点刚完成的 super_admin 角色变更。
func (s *PostgresStore) LoadAdminGuardSubject(ctx context.Context, userID int64) (AdminGuardSubject, error) {
	if s == nil || s.pool == nil || ctx == nil || userID <= 0 {
		return AdminGuardSubject{}, ErrUserNotFound
	}
	var subject AdminGuardSubject
	err := s.pool.QueryRow(ctx, `
		SELECT u.id, u.is_initial_super_admin, EXISTS (
			SELECT 1
			FROM user_roles ur
			JOIN roles r ON r.id = ur.role_id
			WHERE ur.user_id = u.id AND r.key = $2
		)
		FROM users u
		WHERE u.id = $1
	`, userID, RoleSuperAdmin).Scan(
		&subject.UserID,
		&subject.IsInitialSuperAdmin,
		&subject.IsSuperAdmin,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminGuardSubject{}, ErrUserNotFound
	}
	if err != nil {
		return AdminGuardSubject{}, err
	}
	subject.Exists = true
	return subject, nil
}

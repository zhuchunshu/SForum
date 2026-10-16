package identity

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

// PluginSessionStateInvalidator 是宿主「插件命名空间会话状态」存储的最窄形状。
//
// 与 AuthSession 侧同样的理由声明在消费方：Identity 只负责会话目录与撤销事实，
// 插件状态的字节由装配层注入的适配器擦除，本包不依赖存储实现或 Redis。
//
// 语义要点：这是**擦除**钩子，不是访问控制。一个被撤销的 sid 已经无法再发起请求，
// 所以它的插件状态本来就不可达；这里的目的是不让字节在会话失效后继续留着。因此
// 失败必须是 best-effort，绝不能让撤销本身失败（撤销是安全操作，不能被清理拖死）。
type PluginSessionStateInvalidator interface {
	InvalidateSessionBySID(ctx context.Context, sid string) (int, error)
}

// WithPluginSessionStateInvalidator 注入插件会话状态擦除钩子。未注入时撤销照常工作。
func (s *PostgresStore) WithPluginSessionStateInvalidator(invalidator PluginSessionStateInvalidator) *PostgresStore {
	if s != nil {
		s.pluginSessionState = invalidator
	}
	return s
}

// dropPluginSessionState 尽力擦除若干 sid 的插件状态：不返回错误、不 panic、不阻塞。
// 撤销/踢出/改密/重置密码都走它；任何一个失败都不允许影响调用方的主操作。
func (s *PostgresStore) dropPluginSessionState(ctx context.Context, sids []string) {
	if s == nil || s.pluginSessionState == nil || len(sids) == 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for _, sid := range sids {
		sid = strings.TrimSpace(sid)
		if sid == "" {
			continue
		}
		func() {
			defer func() { _ = recover() }()
			_, _ = s.pluginSessionState.InvalidateSessionBySID(ctx, sid)
		}()
	}
}

// collectRevokedSIDs 读尽 RETURNING sid 的行集。必须在事务提交之前调用：pgx 要求
// 上一个查询的行集读完后才能提交，否则提交会失败。
func collectRevokedSIDs(rows pgx.Rows) ([]string, error) {
	defer rows.Close()
	var sids []string
	for rows.Next() {
		var sid string
		if err := rows.Scan(&sid); err != nil {
			return nil, err
		}
		sids = append(sids, sid)
	}
	return sids, rows.Err()
}

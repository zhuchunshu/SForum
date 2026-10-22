package extensionsruntime

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// insertLifecycleActorFixture 插入一个集成测试用的真实账号并返回 id。
//
// 为什么重试：`users.id` 是序列，而同一轮 `go test -p 1` 里更早的包可能用
// **显式 id** 造 fixture（不推进序列）。两者相遇时这次 `RETURNING id` 会撞
// `users_pkey`——看起来像「隔离性坏了」，其实是共享测试库里的序列与显式插入
// 不一致。每次重试都会推进序列，因此最多几次就能拿到空闲 id；这不是掩盖
// 问题，而是让 fixture 不依赖「别的包怎么写 fixture」。
func insertLifecycleActorFixture(
	ctx context.Context,
	pool *pgxpool.Pool,
	username string,
	displayName string,
) (int64, error) {
	var lastErr error
	for attempt := 0; attempt < 8; attempt++ {
		var actorUserID int64
		err := pool.QueryRow(ctx, `
			INSERT INTO users (username, username_lower, email, email_lower, display_name)
			VALUES ($1, $1, $2, $2, $3)
			RETURNING id
		`, username, username+"@example.test", displayName).Scan(&actorUserID)
		if err == nil {
			return actorUserID, nil
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
			return 0, err
		}
		lastErr = err
	}
	return 0, lastErr
}

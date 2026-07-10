package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AdvisoryLocker struct {
	pool *pgxpool.Pool
}

func NewAdvisoryLocker(pool *pgxpool.Pool) *AdvisoryLocker {
	return &AdvisoryLocker{pool: pool}
}

// WithLock pins one pool connection because PostgreSQL advisory locks are session-scoped.
func (l *AdvisoryLocker) WithLock(ctx context.Context, key string, action func(context.Context) error) error {
	if l == nil || l.pool == nil {
		return fmt.Errorf("postgres advisory lock pool is unavailable")
	}
	connection, err := l.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire advisory lock connection: %w", err)
	}
	defer connection.Release()
	if _, err := connection.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended($1, 0))`, key); err != nil {
		return fmt.Errorf("acquire advisory lock %s: %w", key, err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_, _ = connection.Exec(unlockCtx, `SELECT pg_advisory_unlock(hashtextextended($1, 0))`, key)
	}()
	return action(ctx)
}

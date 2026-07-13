package health

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/meilisearch/meilisearch-go"
	"github.com/redis/go-redis/v9"
)

// PostgresChecker 必检：SELECT 1。
type PostgresChecker struct {
	Pool *pgxpool.Pool
}

func (c PostgresChecker) Name() string   { return "postgres" }
func (c PostgresChecker) Required() bool { return true }

func (c PostgresChecker) Check(ctx context.Context) error {
	if c.Pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}
	return c.Pool.Ping(ctx)
}

// RedisChecker 非必检（F1：degraded-ready）。会话/缓存不可用时站点可降级，但不拒收流量。
type RedisChecker struct {
	Client *redis.Client
}

func (c RedisChecker) Name() string   { return "redis" }
func (c RedisChecker) Required() bool { return false }

func (c RedisChecker) Check(ctx context.Context) error {
	if c.Client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return c.Client.Ping(ctx).Err()
}

// MeiliChecker 非必检（F1：degraded-ready）。搜索失败不影响主论坛读写。
type MeiliChecker struct {
	Client meilisearch.ServiceManager
}

func (c MeiliChecker) Name() string   { return "meilisearch" }
func (c MeiliChecker) Required() bool { return false }

func (c MeiliChecker) Check(ctx context.Context) error {
	if c.Client == nil {
		return fmt.Errorf("meilisearch client is nil")
	}
	// HealthWithContext 在不可达时返回 error。
	_, err := c.Client.HealthWithContext(ctx)
	return err
}

// FuncChecker 便于单测注入。
type FuncChecker struct {
	ComponentName string
	IsRequired    bool
	Fn            func(ctx context.Context) error
}

func (c FuncChecker) Name() string   { return c.ComponentName }
func (c FuncChecker) Required() bool { return c.IsRequired }
func (c FuncChecker) Check(ctx context.Context) error {
	if c.Fn == nil {
		return nil
	}
	return c.Fn(ctx)
}

package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolOptions 封装 pgxpool 连接池的可调参数。
// 零值字段在 BuildPoolConfig 中保留 pgxpool 解析出的默认值（来自 DATABASE_URL）。
type PoolOptions struct {
	MaxConns          int32
	MinConns          int32
	MaxConnIdleTime   time.Duration
	MaxConnLifetime   time.Duration
	HealthCheckPeriod time.Duration
	// ConnectTimeout 仅在 > 0 时覆盖 config.ConnConfig.ConnectTimeout。
	ConnectTimeout time.Duration
}

// BuildPoolConfig 解析 DATABASE_URL 并应用连接池选项。
// maxConns > 0 时覆盖最大连接数（保留旧签名供 NewPool 可变参数使用）。
func BuildPoolConfig(databaseURL string, maxConns int32) (*pgxpool.Config, error) {
	return BuildPoolConfigWithOptions(databaseURL, PoolOptions{MaxConns: maxConns})
}

// BuildPoolConfigWithOptions 解析 DATABASE_URL 并应用完整 PoolOptions。
// 零值字段不覆盖 pgxpool 默认值，与改造前行为兼容。
func BuildPoolConfigWithOptions(databaseURL string, opts PoolOptions) (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	if opts.MaxConns > 0 {
		config.MaxConns = opts.MaxConns
	}
	if opts.MinConns > 0 {
		config.MinConns = opts.MinConns
	}
	if opts.MaxConnIdleTime > 0 {
		config.MaxConnIdleTime = opts.MaxConnIdleTime
	}
	if opts.MaxConnLifetime > 0 {
		config.MaxConnLifetime = opts.MaxConnLifetime
	}
	if opts.HealthCheckPeriod > 0 {
		config.HealthCheckPeriod = opts.HealthCheckPeriod
	} else {
		// 默认 30s 健康检查，与改造前一致。
		config.HealthCheckPeriod = 30 * time.Second
	}
	if opts.ConnectTimeout > 0 {
		config.ConnConfig.ConnectTimeout = opts.ConnectTimeout
	}
	return config, nil
}

// NewPool 创建连接池。maxConns 为可选的可变参数，保留向后兼容。
// 新调用方应优先使用 NewPoolWithOptions 以传递完整 PoolOptions。
func NewPool(ctx context.Context, databaseURL string, maxConns ...int32) (*pgxpool.Pool, error) {
	var limit int32
	if len(maxConns) > 0 {
		limit = maxConns[0]
	}
	return NewPoolWithOptions(ctx, databaseURL, PoolOptions{MaxConns: limit})
}

// NewPoolWithOptions 创建连接池并应用完整 PoolOptions，启动时 Ping 验活。
func NewPoolWithOptions(ctx context.Context, databaseURL string, opts PoolOptions) (*pgxpool.Pool, error) {
	config, err := BuildPoolConfigWithOptions(databaseURL, opts)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

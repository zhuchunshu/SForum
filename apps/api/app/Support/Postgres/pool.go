package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func BuildPoolConfig(databaseURL string, maxConns int32) (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	if maxConns > 0 {
		config.MaxConns = maxConns
	}
	config.HealthCheckPeriod = 30 * time.Second
	return config, nil
}

func NewPool(ctx context.Context, databaseURL string, maxConns ...int32) (*pgxpool.Pool, error) {
	var limit int32
	if len(maxConns) > 0 {
		limit = maxConns[0]
	}
	config, err := BuildPoolConfig(databaseURL, limit)
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

package jobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresOptionStore 用 web_options 持久化 schedule 启用开关。
// 不走 Options 模块 catalog：这些 key 是运维内部状态，不出现在站点设置 UI。
type PostgresOptionStore struct {
	pool *pgxpool.Pool
}

func NewPostgresOptionStore(pool *pgxpool.Pool) *PostgresOptionStore {
	return &PostgresOptionStore{pool: pool}
}

func (s *PostgresOptionStore) Get(ctx context.Context, name string) (string, bool, error) {
	if s == nil || s.pool == nil {
		return "", false, nil
	}
	var value string
	err := s.pool.QueryRow(ctx, `SELECT value FROM web_options WHERE name = $1`, name).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get schedule option %q: %w", name, err)
	}
	return value, true, nil
}

func (s *PostgresOptionStore) Set(ctx context.Context, name, value string) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("schedule option store is nil")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO web_options (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
	`, name, value)
	if err != nil {
		return fmt.Errorf("set schedule option %q: %w", name, err)
	}
	return nil
}

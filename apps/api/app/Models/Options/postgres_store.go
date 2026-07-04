package options

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) List(ctx context.Context) ([]Option, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT name, value
		FROM web_options
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list web options: %w", err)
	}
	defer rows.Close()

	options := []Option{}
	for rows.Next() {
		var option Option
		if err := rows.Scan(&option.Name, &option.Value); err != nil {
			return nil, fmt.Errorf("scan web option: %w", err)
		}
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate web options: %w", err)
	}
	return options, nil
}

func (s *PostgresStore) InsertMissing(ctx context.Context, input UpdateInput) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO web_options (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO NOTHING
	`, input.Name, input.Value)
	if err != nil {
		return fmt.Errorf("insert missing web option: %w", err)
	}
	return nil
}

func (s *PostgresStore) Upsert(ctx context.Context, input UpdateInput) (Option, error) {
	var option Option
	err := s.pool.QueryRow(ctx, `
		INSERT INTO web_options (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE
		SET value = EXCLUDED.value
		RETURNING name, value
	`, input.Name, input.Value).Scan(&option.Name, &option.Value)
	if err != nil {
		return Option{}, fmt.Errorf("upsert web option: %w", err)
	}
	return option, nil
}

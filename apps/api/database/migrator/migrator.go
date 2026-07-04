package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"github.com/zhuchunshu/sforum/apps/api/database/migrations"
)

type Config struct {
	DatabaseURL string
	Logger      *slog.Logger
}

func Up(ctx context.Context, cfg Config) error {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// Goose 的 table lock 会持有连接并维护心跳；至少保留两个连接，避免未来 Go 迁移死锁。
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	locker, err := lock.NewPostgresTableLocker(lock.WithTableLogger(logger))
	if err != nil {
		return fmt.Errorf("create migration lock: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		migrations.Files(),
		goose.WithDisableGlobalRegistry(true),
		goose.WithLocker(locker),
		goose.WithSlog(logger),
	)
	if err != nil {
		return fmt.Errorf("create goose provider: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	if len(results) == 0 {
		logger.InfoContext(ctx, "database migrations already up to date")
		return nil
	}

	for _, result := range results {
		logger.InfoContext(ctx, "database migration applied",
			slog.Int64("version", result.Source.Version),
			slog.String("path", result.Source.Path),
			slog.Duration("duration", result.Duration),
		)
	}
	logger.InfoContext(ctx, "database migrations complete", slog.Int("applied", len(results)))
	return nil
}

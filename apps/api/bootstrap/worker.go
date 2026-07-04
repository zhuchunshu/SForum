package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Postgres"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

type Worker struct {
	Client *supportjobs.Client

	closeOnce sync.Once
	close     func()
}

func NewWorker(ctx context.Context, cfg config.Config, logger *slog.Logger) (*Worker, error) {
	if err := runStartupMigrations(ctx, cfg, logger); err != nil {
		return nil, err
	}

	registry := supportjobs.NewRegistry()
	if registry.IsEmpty() {
		// 当前队列基础设施已就绪，但业务模块的 worker 会随模块实现逐步注入。
		return &Worker{}, nil
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.WorkerDatabaseMaxConns)
	if err != nil {
		return nil, fmt.Errorf("postgres setup failed: %w", err)
	}

	workers, err := registry.Build()
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("worker registration failed: %w", err)
	}

	client, err := supportjobs.NewClient(pool, supportjobs.FromAppConfig(cfg), workers)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("job client setup failed: %w", err)
	}

	return &Worker{
		Client: client,
		close:  pool.Close,
	}, nil
}

func (w *Worker) Start(ctx context.Context) error {
	return supportjobs.Start(ctx, w.Client)
}

func (w *Worker) Stop(ctx context.Context) error {
	return supportjobs.Stop(ctx, w.Client)
}

func (w *Worker) Close() {
	if w == nil {
		return
	}
	w.closeOnce.Do(func() {
		if w.close != nil {
			w.close()
		}
	})
}

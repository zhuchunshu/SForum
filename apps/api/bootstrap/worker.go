package bootstrap

import (
	"context"
	"fmt"
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

func NewWorker(ctx context.Context, cfg config.Config) (*Worker, error) {
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.WorkerDatabaseMaxConns)
	if err != nil {
		return nil, fmt.Errorf("postgres setup failed: %w", err)
	}

	registry := supportjobs.NewRegistry()
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

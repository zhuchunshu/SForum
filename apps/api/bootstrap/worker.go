package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	extensionjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Extensions"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Postgres"
	themeruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeRuntime"
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

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.WorkerDatabaseMaxConns)
	if err != nil {
		return nil, fmt.Errorf("postgres setup failed: %w", err)
	}

	registry := supportjobs.NewRegistry()
	extensionStore := extensions.NewPostgresStore(pool)
	themeBuilder := themeruntime.NewBuilder(themeruntime.Config{
		ReleaseRoot:    cfg.ThemeReleaseRoot,
		WebRoot:        cfg.ThemeWebRoot,
		BunPath:        cfg.ThemeBunPath,
		BuildTimeout:   cfg.ThemeBuildTimeout,
		PreviewTimeout: cfg.ThemePreviewTimeout,
		PreviewPath:    cfg.ThemePreviewPath,
	})
	extensionjobs.RegisterThemeActivationWorker(registry, extensionStore, themeBuilder)
	if registry.IsEmpty() {
		pool.Close()
		return &Worker{}, nil
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

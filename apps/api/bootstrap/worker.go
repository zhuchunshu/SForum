package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	extensionjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Extensions"
	identityjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Identity"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Postgres"
	themeruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeRuntime"
	webreleaseruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/WebReleaseRuntime"
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

	pool, err := postgres.NewPoolWithOptions(ctx, cfg.DatabaseURL, postgres.PoolOptions{
		MaxConns:        cfg.WorkerDatabaseMaxConns,
		MinConns:        cfg.WorkerDatabaseMinConns,
		MaxConnIdleTime: cfg.WorkerDatabaseMaxConnIdleTime,
		MaxConnLifetime: cfg.WorkerDatabaseMaxConnLifetime,
		ConnectTimeout:  cfg.WorkerDatabaseConnectTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("postgres setup failed: %w", err)
	}

	worker, err := newWorkerWithPool(cfg, pool, logger)
	if err != nil {
		pool.Close()
		return nil, err
	}
	worker.close = pool.Close
	return worker, nil
}

func newWorkerWithPool(cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger) (*Worker, error) {
	registry := supportjobs.NewRegistry()
	extensionStore := extensions.NewPostgresStore(pool)
	webReleaseStore := extensions.NewPostgresWebReleaseStore(pool)
	themeBuilder := themeruntime.NewBuilder(themeruntime.Config{
		ReleaseRoot:    cfg.ThemeReleaseRoot,
		WebRoot:        cfg.ThemeWebRoot,
		BunPath:        cfg.ThemeBunPath,
		BuildTimeout:   cfg.ThemeBuildTimeout,
		PreviewTimeout: cfg.ThemePreviewTimeout,
		PreviewPath:    cfg.ThemePreviewPath,
	})
	extensionjobs.RegisterThemeActivationWorker(registry, extensionStore, themeBuilder)
	webReleaseBuilder := webreleaseruntime.NewBuilder(webreleaseruntime.Config{
		ReleaseRoot:    cfg.WebReleaseRoot,
		WebRoot:        cfg.WebReleaseWebRoot,
		ExtensionRoot:  cfg.ExtensionRoot,
		BunPath:        cfg.WebReleaseBunPath,
		BuildTimeout:   cfg.WebReleaseBuildTimeout,
		PreviewTimeout: cfg.WebReleasePreviewTimeout,
		PreviewPath:    cfg.WebReleasePreviewPath,
		HostPeers:      webreleaseruntime.HostPeers(),
	})
	extensionjobs.RegisterWebReleaseBuildWorker(registry, webReleaseStore, webReleaseBuilder, postgres.NewAdvisoryLocker(pool))
	extensionjobs.RegisterWebReleaseCleanupWorker(registry, webReleaseStore, cfg.WebReleaseRoot)
	registerSearchWorkers(registry, cfg, pool)
	// 周期任务通过返回值显式传递，避免用包级全局变量在多次构造（独立 worker + API 内嵌
	// worker）之间互相覆盖或丢失注册。
	var periodicJobs []*river.PeriodicJob
	if cleanup := registerIdentityCleanupWorker(registry, cfg, pool, logger); cleanup != nil {
		periodicJobs = append(periodicJobs, cleanup)
	}
	periodicJobs = append(periodicJobs, river.NewPeriodicJob(
		river.PeriodicInterval(24*time.Hour),
		func() (river.JobArgs, *river.InsertOpts) {
			return extensionjobs.WebReleaseCleanupArgs{}, nil
		},
		&river.PeriodicJobOpts{RunOnStart: false},
	))
	if registry.IsEmpty() {
		return &Worker{}, nil
	}

	workers, err := registry.Build()
	if err != nil {
		return nil, fmt.Errorf("worker registration failed: %w", err)
	}

	client, err := supportjobs.NewClientWithPeriodic(pool, supportjobs.FromAppConfig(cfg), workers, periodicJobs)
	if err != nil {
		return nil, fmt.Errorf("job client setup failed: %w", err)
	}

	return &Worker{Client: client}, nil
}

// registerIdentityCleanupWorker 注册历史会话清理 worker 与每天一次的 periodic job，返回该周期任务。
// keep_days 从 runtime option 读取（每次执行时实时解析，admin 改动即生效）。
func registerIdentityCleanupWorker(registry *supportjobs.Registry, cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger) *river.PeriodicJob {
	optionStore := options.NewPostgresStore(pool)
	optionsService := options.NewServiceWithDefaults(optionStore, optionsDefaultsFromConfig(cfg))
	identityStore := identity.NewPostgresStore(pool)

	registry.Add(func(workers *river.Workers) error {
		river.AddWorker(workers, &identityjobs.CleanupSessionsWorker{
			Store: identityStore,
			KeepDays: func(ctx context.Context) (int, error) {
				raw, err := optionsService.WebOption(ctx, options.NameIdentitySessionsKeepDays)
				if err != nil {
					return 0, err
				}
				if days, err := strconv.Atoi(raw); err == nil && days > 0 {
					return days, nil
				}
				return identity.RecommendedSessionsKeepDays, nil
			},
			Logger: logger,
		})
		return nil
	})

	return river.NewPeriodicJob(
		river.PeriodicInterval(24*time.Hour),
		func() (river.JobArgs, *river.InsertOpts) {
			return identityjobs.CleanupSessionsArgs{}, nil
		},
		&river.PeriodicJobOpts{RunOnStart: false},
	)
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

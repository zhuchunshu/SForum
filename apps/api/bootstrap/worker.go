package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	attachmentjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Attachments"
	extensionjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Extensions"
	identityjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Identity"
	notificationjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Notifications"
	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Postgres"
	themeruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeRuntime"
	webreleaseruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/WebReleaseRuntime"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

type Worker struct {
	Client *supportjobs.Client
	// Schedules 是宿主 schedule catalog（含元数据）；River PeriodicJobs 由其 Build 得到。
	Schedules *supportjobs.ScheduleRegistry

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
	runtimeClose := worker.close
	worker.close = func() {
		if runtimeClose != nil {
			runtimeClose()
		}
		pool.Close()
	}
	return worker, nil
}

func newWorkerWithPool(cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger) (*Worker, error) {
	registry := supportjobs.NewRegistry()
	extensionStore := extensions.NewPostgresStore(pool)
	extensionRuntime := extensionsruntime.NewManager(extensionsruntime.ManagerConfig{
		Starter:       extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{Settings: extensionStore}),
		DeliveryStore: extensionStore,
	})
	extensionService := extensions.NewServiceWithBuiltins(extensionStore, cfg.ExtensionRoot, cfg.BuiltinExtensionRoot)
	if _, err := extensionService.SyncBuiltins(context.Background()); err != nil {
		return nil, fmt.Errorf("sync worker builtin extensions: %w", err)
	}
	workerOptions := options.NewServiceWithDefaults(options.NewPostgresStore(pool), optionsDefaultsFromConfig(cfg))
	legacyMailValues, err := workerOptions.InternalValues(context.Background())
	if err != nil {
		return nil, fmt.Errorf("load worker legacy mail options: %w", err)
	}
	notificationStore := notifications.NewPostgresStore(pool)
	if err := notificationStore.AdoptLegacyMail(context.Background(), legacyMailValues); err != nil {
		return nil, fmt.Errorf("adopt worker legacy mail settings: %w", err)
	}
	items, err := extensionStore.List(context.Background())
	if err != nil {
		return nil, fmt.Errorf("list worker extensions: %w", err)
	}
	extensionRuntime.Reconcile(context.Background(), items)
	mailProviders := extensionsruntime.NewMailProviderRegistry(extensionStore)
	notificationjobs.Register(registry, &notificationjobs.DeliverMailWorker{Store: notificationStore, Providers: mailProviders, Sender: extensionRuntime})
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
		ReleaseRoot:       cfg.WebReleaseRoot,
		WebRoot:           cfg.WebReleaseWebRoot,
		ExtensionRoot:     cfg.ExtensionRoot,
		DefaultThemeLayer: filepath.Join(cfg.BuiltinExtensionRoot, "themes", "sforum-default", "layer"),
		BunPath:           cfg.WebReleaseBunPath,
		BuildTimeout:      cfg.WebReleaseBuildTimeout,
		PreviewTimeout:    cfg.WebReleasePreviewTimeout,
		PreviewPath:       cfg.WebReleasePreviewPath,
		HostPeers:         webreleaseruntime.HostPeers(),
	})
	extensionjobs.RegisterWebReleaseBuildWorker(registry, webReleaseStore, webReleaseBuilder, postgres.NewAdvisoryLocker(pool))
	extensionjobs.RegisterWebReleaseCleanupWorker(registry, webReleaseStore, cfg.WebReleaseRoot)
	// 孤儿附件清理：handler 已存在，F1 通过 schedule registry 挂上 daily maintenance。
	attachmentStore := attachments.NewPostgresStore(pool)
	attachmentService := attachments.NewService(attachmentStore, workerOptions)
	attachmentjobs.Register(registry, attachmentService)
	registerSearchWorkers(registry, cfg, pool)
	registerIdentityCleanupWorker(registry, cfg, pool, logger)

	// 周期任务统一经 Schedule Registry 注册，禁止在 bootstrap 散落 NewPeriodicJob。
	scheduleRegistry, err := supportjobs.NewCoreScheduleRegistry(map[string]river.PeriodicJobConstructor{
		supportjobs.ScheduleIdentityCleanupSessions: func() (river.JobArgs, *river.InsertOpts) {
			return identityjobs.CleanupSessionsArgs{}, nil
		},
		supportjobs.ScheduleExtensionWebReleaseCleanup: func() (river.JobArgs, *river.InsertOpts) {
			return extensionjobs.WebReleaseCleanupArgs{}, nil
		},
		supportjobs.ScheduleAttachmentsCleanupOrphans: func() (river.JobArgs, *river.InsertOpts) {
			// Limit=0 时 worker 使用默认批大小 100。
			return attachmentjobs.CleanupOrphansArgs{}, nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("schedule registry: %w", err)
	}
	periodicJobs, err := scheduleRegistry.BuildPeriodicJobs()
	if err != nil {
		return nil, fmt.Errorf("build periodic jobs: %w", err)
	}

	if registry.IsEmpty() {
		return &Worker{Schedules: scheduleRegistry}, nil
	}

	workers, err := registry.Build()
	if err != nil {
		return nil, fmt.Errorf("worker registration failed: %w", err)
	}

	client, err := supportjobs.NewClientWithPeriodic(pool, supportjobs.FromAppConfig(cfg), workers, periodicJobs)
	if err != nil {
		return nil, fmt.Errorf("job client setup failed: %w", err)
	}

	return &Worker{
		Client:    client,
		Schedules: scheduleRegistry,
		close:     func() { extensionRuntime.Close(context.Background()) },
	}, nil
}

// registerIdentityCleanupWorker 注册历史会话清理 worker。
// keep_days 从 runtime option 读取（每次执行时实时解析，admin 改动即生效）。
// 对应的 daily schedule 由 Schedule Registry 统一拥有，不在此返回 PeriodicJob。
func registerIdentityCleanupWorker(registry *supportjobs.Registry, cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger) {
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

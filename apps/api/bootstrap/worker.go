package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	attachmentjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Attachments"
	auditjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Audit"
	forumjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Forum"
	identityjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Identity"
	notificationjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Notifications"
	webhookjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Webhooks"
	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	jobsmodel "github.com/zhuchunshu/sforum/apps/api/app/Models/Jobs"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	webhooks "github.com/zhuchunshu/sforum/apps/api/app/Models/Webhooks"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Crypto"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	health "github.com/zhuchunshu/sforum/apps/api/app/Support/Health"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Postgres"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

type Worker struct {
	Client *supportjobs.Client
	// Schedules 是宿主 schedule catalog（含元数据）；River PeriodicJobs 由其 Build 得到。
	Schedules *supportjobs.ScheduleRegistry

	closeOnce sync.Once
	close     func()
}

// workerExtensionRuntime 是 worker 对 extension runtime 的最小依赖面：
// mail.deliver 需要 SendMail；附件清理需要 StorageRuntime；独立进程还需 Reconcile/Close。
// API embed 注入的 *extensionsruntime.Manager 满足此接口。
type workerExtensionRuntime interface {
	notificationjobs.ProviderSender
	extensionsruntime.StorageRuntime
	Reconcile(ctx context.Context, items []extensions.Extension)
	Close(ctx context.Context)
	protocolV2ProviderBrokerSource
}

// workerRuntimeDeps 控制 extension runtime / Host API 的所有权。
// ExtensionRuntime 非 nil 时复用注入实例（embed 模式），不再二次 Reconcile，也不在 Worker.Close 中关闭。
// ExtensionRuntime 为 nil 时由 worker 自建 Manager + Host API gateway（独立 worker 路径）。
type workerRuntimeDeps struct {
	ExtensionRuntime workerExtensionRuntime
	// OwnsRuntime 为 true 时 Worker.Close 关闭 runtime（及自建的 Host API gateway）。
	// 注入共享 runtime 时必须为 false，由 API shutdown 负责 Close。
	OwnsRuntime bool
}

// hostAPIGatewayCloser 仅用于 worker 自建 Host API 时的关闭钩子。
type hostAPIGatewayCloser interface {
	Close() error
}

type pluginJobRuntimeResolver struct {
	store extensions.Store
	trust extensionsruntime.RuntimeTrustSource
}

func (r *pluginJobRuntimeResolver) ResolvePluginJobRuntime(ctx context.Context, extensionID, jobName string) (hostapi.PluginJobRuntimeContract, error) {
	if r == nil || r.store == nil || r.trust == nil {
		return hostapi.PluginJobRuntimeContract{}, fmt.Errorf("plugin job resolver is not configured")
	}
	extension, err := r.store.Get(ctx, extensionID)
	if err != nil {
		if errors.Is(err, extensions.ErrExtensionNotFound) {
			return hostapi.PluginJobRuntimeContract{}, supportjobs.ErrPluginJobRuntimeStale
		}
		return hostapi.PluginJobRuntimeContract{}, err
	}
	if extension.Type != extensions.TypePlugin || extension.Status != extensions.StatusEnabled {
		return hostapi.PluginJobRuntimeContract{}, supportjobs.ErrPluginJobRuntimeStale
	}
	contract, err := extensions.PluginJobContractForExtension(extension, jobName)
	if err != nil {
		if errors.Is(err, extensions.ErrExtensionNotFound) || errors.Is(err, extensions.ErrInvalidManifest) {
			return hostapi.PluginJobRuntimeContract{}, supportjobs.ErrPluginJobRuntimeStale
		}
		return hostapi.PluginJobRuntimeContract{}, err
	}
	identity, err := r.trust.RuntimeIdentity(ctx, extension)
	if err != nil {
		if errors.Is(err, extensions.ErrTrustGrantNotFound) {
			return hostapi.PluginJobRuntimeContract{}, supportjobs.ErrPluginJobRuntimeStale
		}
		return hostapi.PluginJobRuntimeContract{}, err
	}
	return hostapi.PluginJobRuntimeContract{Contract: contract, TrustGrantID: identity.TrustGrantID}, nil
}

var newStandaloneWorkerRuntimeManager = func(store extensions.Store, hostAPI extensionsruntime.HostAPIRegistrar, settings extensionsruntime.PluginSettings, trust extensionsruntime.RuntimeTrustSource) workerExtensionRuntime {
	return extensionsruntime.NewManager(extensionsruntime.ManagerConfig{
		Starter: extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
			Settings: settings,
			HostAPI:  hostAPI,
			Trust:    trust,
		}),
		DeliveryStore: store,
	})
}

func buildStandaloneWorkerExtensionRuntime(
	ctx context.Context,
	cfg config.Config,
	databaseBinderFactory protocolV2DatabaseCatalogBinderFactory,
	store extensions.Store,
	cipher *crypto.OptionCipher,
	trust extensionsruntime.RuntimeTrustSource,
	coordinators ...*extensions.ActivationCoordinator,
) (workerExtensionRuntime, hostAPIGatewayCloser, error) {
	service := extensions.NewServiceWithBuiltins(store, cfg.ExtensionRoot, cfg.BuiltinExtensionRoot)
	extensions.WithCipher(cipher)(service)
	extensions.WithSafeMode(cfg.SafeMode)(service)
	var activation *extensions.ActivationCoordinator
	if len(coordinators) > 0 {
		activation = coordinators[0]
	} else if activationStore, ok := store.(extensions.ActivationAttemptStore); ok {
		activation = extensions.NewActivationCoordinator(activationStore)
	}
	extensions.WithActivationCoordinator(activation)(service)
	workerHostAPI := hostapi.New(hostapi.Config{Settings: service})
	workerHostGateway := hostapi.NewGateway(workerHostAPI)
	if databaseBinderFactory == nil {
		_ = workerHostGateway.Close()
		return nil, nil, fmt.Errorf("worker DatabaseService catalog binder is required")
	}
	databaseBinder := databaseBinderFactory(workerHostGateway)
	managedRuntime := newStandaloneWorkerRuntimeManager(store, workerHostGateway, service, trust)
	if err := bindProtocolV2ProviderBroker(workerHostGateway, managedRuntime); err != nil {
		managedRuntime.Close(ctx)
		_ = workerHostGateway.Close()
		return nil, nil, err
	}
	if runtime, ok := managedRuntime.(interface {
		SetStartPreparer(func(context.Context, extensions.Extension) error)
	}); ok {
		runtime.SetStartPreparer(protocolV2DatabaseStartPreparer(store, databaseBinder, cfg.SafeMode))
	}
	if runtime, ok := managedRuntime.(interface {
		WithActivation(*extensions.ActivationCoordinator, string) *extensionsruntime.Manager
	}); ok {
		runtime.WithActivation(activation, extensions.NewActivationBootID())
	}
	workerHostAPI.BindCapabilitySource(service)
	if _, err := service.SyncBuiltins(ctx); err != nil {
		managedRuntime.Close(ctx)
		_ = workerHostGateway.Close()
		return nil, nil, fmt.Errorf("sync worker builtin extensions: %w", err)
	}
	items, err := store.List(ctx)
	if err != nil {
		managedRuntime.Close(ctx)
		_ = workerHostGateway.Close()
		return nil, nil, fmt.Errorf("list worker extensions: %w", err)
	}
	// 独立 worker 同样会启动插件 broker，必须先绑定同一份精确 SQL catalog。
	if err := bindProtocolV2DatabaseRuntime(databaseBinder, items, cfg.SafeMode); err != nil {
		managedRuntime.Close(ctx)
		_ = workerHostGateway.Close()
		return nil, nil, fmt.Errorf("bind worker DatabaseService runtime: %w", err)
	}
	if cfg.SafeMode {
		managedRuntime.Reconcile(ctx, nil)
	} else {
		managedRuntime.Reconcile(ctx, items)
	}
	return managedRuntime, workerHostGateway, nil
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

	// 独立 worker：自建 runtime，OwnsRuntime 由 newWorkerWithPool 在 nil inject 时设为 true。
	worker, err := newWorkerWithPool(cfg, pool, logger, workerRuntimeDeps{})
	if err != nil {
		pool.Close()
		return nil, err
	}
	if cfg.SafeMode {
		if err := audit.NewPostgresWriter(pool).Append(ctx, audit.Event{
			Action:   audit.ActionExtensionSafeModeBoot,
			Metadata: map[string]any{"process": "worker"},
		}); err != nil && logger != nil {
			logger.Warn("record worker safe mode boot audit failed", "error", err)
		}
	}

	// 独立 worker 进程发布心跳，供 API overview / 运维判断 stale。
	// 嵌入 API 的 worker 由 bootstrap.NewAPI 发布，避免双写无妨但这里只在独立进程路径启用。
	redisClient := humanverify.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, humanverify.RedisClientOptions{
		PoolSize:        cfg.RedisPoolSize,
		MinIdleConns:    cfg.RedisMinIdleConns,
		DialTimeout:     cfg.RedisDialTimeout,
		ReadTimeout:     cfg.RedisReadTimeout,
		WriteTimeout:    cfg.RedisWriteTimeout,
		ConnMaxIdleTime: cfg.RedisConnMaxIdleTime,
		ConnMaxLifetime: cfg.RedisConnMaxLifetime,
	})
	heartbeatStore := health.NewRedisHeartbeatStore(redisClient)
	heartbeatCtx, heartbeatCancel := context.WithCancel(context.Background())
	go (&health.Publisher{Store: heartbeatStore}).Run(heartbeatCtx)

	runtimeClose := worker.close
	worker.close = func() {
		heartbeatCancel()
		if err := redisClient.Close(); err != nil && logger != nil {
			logger.Warn("worker heartbeat redis close failed", "error", err)
		}
		if runtimeClose != nil {
			runtimeClose()
		}
		pool.Close()
	}
	return worker, nil
}

// resolveWorkerExtensionRuntime 决定 embed 注入 vs 独立自建。
// 注入时 ownsRuntime=false 且不调用 buildStandalone（因此不会二次 Reconcile/Start）。
func resolveWorkerExtensionRuntime(
	deps workerRuntimeDeps,
	buildStandalone func() (workerExtensionRuntime, hostAPIGatewayCloser, error),
) (workerExtensionRuntime, hostAPIGatewayCloser, bool, error) {
	if deps.ExtensionRuntime != nil {
		return deps.ExtensionRuntime, nil, false, nil
	}
	runtime, gateway, err := buildStandalone()
	if err != nil {
		return nil, nil, false, err
	}
	return runtime, gateway, true, nil
}

func newWorkerWithPool(cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger, deps workerRuntimeDeps) (*Worker, error) {
	registry := supportjobs.NewRegistry()
	extensionStore := extensions.NewPostgresStore(pool)
	runtimeTrust := extensions.NewExecutableTrustService(extensionStore, extensions.NewPostgresExecutableTrustStore(pool))
	optionCipher, err := crypto.NewOptionCipher(cfg.OptionEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create worker option cipher: %w", err)
	}

	extensionRuntime, hostGateway, ownsRuntime, err := resolveWorkerExtensionRuntime(deps, func() (workerExtensionRuntime, hostAPIGatewayCloser, error) {
		activation := extensions.NewActivationCoordinator(extensionStore).WithAuditor(audit.NewPostgresWriter(pool))
		return buildStandaloneWorkerExtensionRuntime(
			context.Background(), cfg,
			postgresProtocolV2DatabaseCatalogBinderFactory(
				pool, hostapi.WithProtocolV2DatabaseTraceSink(hostapi.NewSlogDatabaseTraceSink(logger)),
			),
			extensionStore, optionCipher, runtimeTrust, activation,
		)
	})
	if err != nil {
		return nil, err
	}

	workerOptions := options.NewServiceWithDefaults(options.NewPostgresStore(pool), optionsDefaultsFromConfig(cfg)).
		WithCipher(optionCipher)
	legacyMailValues, err := workerOptions.InternalValues(context.Background())
	if err != nil {
		return nil, fmt.Errorf("load worker legacy mail options: %w", err)
	}
	notificationStore := notifications.NewPostgresStore(pool)
	if err := notificationStore.AdoptLegacyMail(context.Background(), legacyMailValues); err != nil {
		return nil, fmt.Errorf("adopt worker legacy mail settings: %w", err)
	}
	// mail.deliver 必须命中共享/本进程同一 runtime，避免 embed 时打到第二套插件进程。
	mailProviders := extensionsruntime.NewMailProviderRegistry(extensionStore)
	notificationjobs.Register(registry, &notificationjobs.DeliverMailWorker{Store: notificationStore, Providers: mailProviders, Sender: extensionRuntime})
	// F3.3：出站 webhook 投递（SSRF 安全客户端；生产禁 http；secret 解密）。
	webhookStore := webhooks.NewPostgresStore(pool)
	webhookSecretSvc := webhooks.NewService(webhookStore, nil, nil).WithCipher(optionCipher)
	webhookjobs.Register(registry, &webhookjobs.DeliverWorker{
		Store:     webhookStore,
		Secrets:   webhookSecretSvc,
		Migrator:  webhookStore,
		AllowHTTP: !strings.EqualFold(cfg.AppEnv, "production"),
	})
	// 孤儿附件清理：handler 已存在，F1 通过 schedule registry 挂上 daily maintenance。
	// 与 API 相同：插件存储路径需注入 runtime（E6.2）。
	attachmentStore := attachments.NewPostgresStore(pool)
	attachmentService := attachments.NewService(attachmentStore, workerOptions).
		WithStoragePluginRuntime(extensionRuntime)
	attachmentjobs.Register(registry, attachmentService)
	// 审计日志保留期清理（F1.4）：默认 90 天，handler 可后续接 runtime option。
	auditWriter := audit.NewPostgresWriter(pool)
	auditjobs.Register(registry, &auditjobs.CleanupEventsWorker{
		Cleaner: auditWriter,
		KeepDays: func(context.Context) (int, error) {
			return audit.RecommendedRetentionDays, nil
		},
		Logger: logger,
	})
	registerSearchWorkers(registry, cfg, pool)
	registerIdentityCleanupWorker(registry, cfg, pool, logger)
	registerForumAutoLockWorker(registry, cfg, pool, logger)
	// F2.2：插件经 Host API 入队的 extension.plugin_job。
	registry.Add(func(workers *river.Workers) error {
		var executor hostapi.PluginJobExecutor
		if candidate, ok := extensionRuntime.(hostapi.PluginJobExecutor); ok {
			executor = candidate
		}
		river.AddWorker(workers, &hostapi.PluginJobWorker{
			Resolver: &pluginJobRuntimeResolver{store: extensionStore, trust: runtimeTrust},
			Executor: executor,
		})
		return nil
	})

	// 周期任务统一经 Schedule Registry 注册，禁止在 bootstrap 散落 NewPeriodicJob。
	// 启用状态读 web_options；constructor 返回 nil 时 River 跳过本次插入（无需重启 worker）。
	scheduleOptions := jobsmodel.NewPostgresOptionStore(pool)
	wrapEnabled := func(scheduleID string, ctor river.PeriodicJobConstructor) river.PeriodicJobConstructor {
		return func() (river.JobArgs, *river.InsertOpts) {
			value, ok, err := scheduleOptions.Get(context.Background(), supportjobs.ScheduleEnabledOptionName(scheduleID))
			if err == nil && !supportjobs.ParseScheduleEnabled(value, ok) {
				return nil, nil
			}
			return ctor()
		}
	}
	scheduleRegistry, err := supportjobs.NewCoreScheduleRegistry(map[string]river.PeriodicJobConstructor{
		supportjobs.ScheduleIdentityCleanupSessions: wrapEnabled(
			supportjobs.ScheduleIdentityCleanupSessions,
			func() (river.JobArgs, *river.InsertOpts) {
				return identityjobs.CleanupSessionsArgs{}, nil
			},
		),
		supportjobs.ScheduleAttachmentsCleanupOrphans: wrapEnabled(
			supportjobs.ScheduleAttachmentsCleanupOrphans,
			func() (river.JobArgs, *river.InsertOpts) {
				// Limit=0 时 worker 使用默认批大小 100。
				return attachmentjobs.CleanupOrphansArgs{}, nil
			},
		),
		supportjobs.ScheduleAuditCleanupEvents: wrapEnabled(
			supportjobs.ScheduleAuditCleanupEvents,
			func() (river.JobArgs, *river.InsertOpts) {
				return auditjobs.CleanupEventsArgs{}, nil
			},
		),
		supportjobs.ScheduleForumAutoLockIdle: wrapEnabled(
			supportjobs.ScheduleForumAutoLockIdle,
			func() (river.JobArgs, *river.InsertOpts) {
				return forumjobs.AutoLockIdleArgs{}, nil
			},
		),
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

	// 仅 worker 拥有 runtime 时关闭；embed 注入路径由 API 在 River stop 之后关闭。
	var closeFn func()
	if ownsRuntime {
		closeFn = func() {
			if extensionRuntime != nil {
				extensionRuntime.Close(context.Background())
			}
			if hostGateway != nil {
				_ = hostGateway.Close()
			}
		}
	}

	return &Worker{
		Client:    client,
		Schedules: scheduleRegistry,
		close:     closeFn,
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

// registerForumAutoLockWorker 注册闲置主题自动锁帖 worker。
// idle days 每次执行时从 web_options 读取；0 或缺失时 job 空跑。
func registerForumAutoLockWorker(registry *supportjobs.Registry, cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger) {
	optionStore := options.NewPostgresStore(pool)
	optionsService := options.NewServiceWithDefaults(optionStore, optionsDefaultsFromConfig(cfg))
	forumStore := forum.NewPostgresStore(pool)

	registry.Add(func(workers *river.Workers) error {
		return river.AddWorkerSafely[forumjobs.AutoLockIdleArgs](workers, &forumjobs.AutoLockIdleWorker{
			Locker: forumStore,
			IdleDays: func(ctx context.Context) (int, error) {
				raw, err := optionsService.WebOption(ctx, options.NameForumTopicsAutoLockIdleDays)
				if err != nil {
					return 0, err
				}
				if days, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && days > 0 {
					return days, nil
				}
				return 0, nil
			},
			Logger: logger,
		})
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

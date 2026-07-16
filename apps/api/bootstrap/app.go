package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/session"

	httpserver "github.com/zhuchunshu/sforum/apps/api/app/Http"
	apitokens "github.com/zhuchunshu/sforum/apps/api/app/Models/APITokens"
	adminoverview "github.com/zhuchunshu/sforum/apps/api/app/Models/AdminOverview"
	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	database "github.com/zhuchunshu/sforum/apps/api/app/Models/Database"
	entitymeta "github.com/zhuchunshu/sforum/apps/api/app/Models/EntityMeta"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	moderation "github.com/zhuchunshu/sforum/apps/api/app/Models/Moderation"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	pageviewmodels "github.com/zhuchunshu/sforum/apps/api/app/Models/PageViewModels"
	profile "github.com/zhuchunshu/sforum/apps/api/app/Models/Profile"
	sitechrome "github.com/zhuchunshu/sforum/apps/api/app/Models/SiteChrome"
	webhooks "github.com/zhuchunshu/sforum/apps/api/app/Models/Webhooks"
	"github.com/zhuchunshu/sforum/apps/api/app/Providers"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	authsupport "github.com/zhuchunshu/sforum/apps/api/app/Support/Auth"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	avatar "github.com/zhuchunshu/sforum/apps/api/app/Support/Avatar"
	cache "github.com/zhuchunshu/sforum/apps/api/app/Support/Cache"
	cacheregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/CacheRegistry"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Crypto"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	health "github.com/zhuchunshu/sforum/apps/api/app/Support/Health"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Idempotency"
	installationidentity "github.com/zhuchunshu/sforum/apps/api/app/Support/InstallationIdentity"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Postgres"
	redisplatform "github.com/zhuchunshu/sforum/apps/api/app/Support/Redis"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
	search "github.com/zhuchunshu/sforum/apps/api/app/Support/Search"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

type API struct {
	App  *fiber.App
	Addr string

	closeOnce sync.Once
	close     func()
}

type extensionRuntime interface {
	extensions.RuntimeManager
	appevents.Publisher
	RouteTarget(extensionID string) (extensionsruntime.RouteTarget, bool)
	AcquireActiveRuntimeCall(context.Context, string, extensionsruntime.RuntimeCallClass) (extensionsruntime.RuntimeInstanceSnapshot, *extensionsruntime.RuntimeAdmissionLease, error)
	AcquireRuntimeCall(context.Context, extensionsruntime.RuntimeInstanceIdentity, extensionsruntime.RuntimeCallClass) (*extensionsruntime.RuntimeAdmissionLease, error)
	AdminSurfaceSnapshot(string) extensionsruntime.AdminSurfaceRegistrySnapshot
	ResolveAdminSurface(string) (extensionsruntime.AdminSurfaceContract, error)
	InvokeAdminSurface(context.Context, extensionsruntime.AdminSurfaceInvocation) (extensionsruntime.AdminSurfaceInvocationResult, error)
	Reconcile(ctx context.Context, items []extensions.Extension)
	Close(ctx context.Context)
	// SendMail 供 embed worker 的 mail.deliver 复用同一 runtime（P0 共享插件进程）。
	SendMail(ctx context.Context, extensionID string, request extensionsruntime.MailProviderRequest) (extensionsruntime.MailProviderResponse, error)
	// 附件存储槽 RPC（E6.2）；与 StorageRuntime 对齐。
	extensionsruntime.StorageRuntime
	protocolV2ProviderBrokerSource
}

var newExtensionRuntimeManager = func(
	store extensions.Store,
	hostAPI extensionsruntime.HostAPIRegistrar,
	settings extensionsruntime.PluginSettings,
	trust extensionsruntime.RuntimeTrustSource,
	databaseLeases extensionsruntime.RuntimeDatabaseLeaseRegistry,
) extensionRuntime {
	if settings == nil {
		settings = store
	}
	return extensionsruntime.NewManager(extensionsruntime.ManagerConfig{
		Starter: extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
			Settings:       settings,
			HostAPI:        hostAPI,
			Trust:          trust,
			DatabaseLeases: databaseLeases,
		}),
		DeliveryStore: store,
	})
}

func bindAPIExtensionRuntime(
	store extensions.Store,
	hostAPI extensionsruntime.HostAPIRegistrar,
	service *extensions.Service,
	trust extensionsruntime.RuntimeTrustSource,
	databaseLeases extensionsruntime.RuntimeDatabaseLeaseRegistry,
) extensionRuntime {
	runtime := newExtensionRuntimeManager(store, hostAPI, service, trust, databaseLeases)
	extensions.WithRuntimeManager(runtime)(service)
	return runtime
}

func shouldEmbedWorkerInAPI(cfg config.Config) bool {
	return cfg.EmbedWorkerInAPI
}

func NewAPI(ctx context.Context, cfg config.Config, logger *slog.Logger) (*API, error) {
	if err := runStartupMigrations(ctx, cfg, logger); err != nil {
		return nil, err
	}

	pool, err := postgres.NewPoolWithOptions(ctx, cfg.DatabaseURL, postgres.PoolOptions{
		MaxConns:        cfg.DatabaseMaxConns,
		MinConns:        cfg.DatabaseMinConns,
		MaxConnIdleTime: cfg.DatabaseMaxConnIdleTime,
		MaxConnLifetime: cfg.DatabaseMaxConnLifetime,
		ConnectTimeout:  cfg.DatabaseConnectTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("postgres setup failed: %w", err)
	}
	databaseLeaseRegistry := extensionsruntime.NewPostgresExtensionDatabaseRegistry(pool, nil)
	if _, err := databaseLeaseRegistry.ReapExpiredRuntimeLeases(
		ctx, extensionsruntime.DefaultExtensionDatabaseRuntimeLeaseReapLimit,
	); err != nil {
		pool.Close()
		return nil, fmt.Errorf("reap expired extension database runtime leases: %w", err)
	}
	hostInstallationID, err := installationidentity.NewPostgresRepository(pool).Ensure(ctx)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("ensure Host installation identity: %w", err)
	}

	redisStorage, err := redisplatform.NewStorage(cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("redis session storage setup failed: %w", err)
	}
	// 共享 Redis client：humanverify 与 forum 业务读缓存复用同一连接池，避免多套连接。
	// session storage 走 fiber Storage 接口（内部封装独立连接），不参与合并。
	sharedRedisClient := humanverify.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, humanverify.RedisClientOptions{
		PoolSize:        cfg.RedisPoolSize,
		MinIdleConns:    cfg.RedisMinIdleConns,
		DialTimeout:     cfg.RedisDialTimeout,
		ReadTimeout:     cfg.RedisReadTimeout,
		WriteTimeout:    cfg.RedisWriteTimeout,
		ConnMaxIdleTime: cfg.RedisConnMaxIdleTime,
		ConnMaxLifetime: cfg.RedisConnMaxLifetime,
	})
	humanVerifyStore := humanverify.NewRedisStore(sharedRedisClient)
	optionStore := options.NewPostgresStore(pool)
	// H2a：构造敏感值加密器；未配置 APP_OPTION_ENC_KEY 时为透明模式（开发环境），生产由 C2 强制要求。
	optionCipher, err := crypto.NewOptionCipher(cfg.OptionEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create option cipher: %w", err)
	}
	auditWriter := audit.NewPostgresWriter(pool)
	if cfg.SafeMode {
		if err := auditWriter.Append(ctx, audit.Event{
			Action:   audit.ActionExtensionSafeModeBoot,
			Metadata: map[string]any{"process": "api"},
		}); err != nil {
			logger.Warn("record safe mode boot audit failed", "error", err)
		}
	}
	optionsService := options.NewServiceWithDefaults(optionStore, optionsDefaultsFromConfig(cfg)).
		WithCipher(optionCipher).
		WithAuditor(auditWriter)
	if err := optionsService.EnsureDefaults(ctx); err != nil {
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("ensure runtime options defaults failed: %w", err)
	}
	if err := optionsService.RefreshForumReadPolicy(ctx); err != nil {
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("refresh forum read policy failed: %w", err)
	}
	humanVerifier := humanverify.NewRuntimeService(optionsService, humanVerifyStore, humanVerifyConfigFromConfig(cfg))

	// CookieSecure：生产环境强制 true；此外当 APP_URL 是 https 时也启用，
	// 避免 APP_ENV=staging 等"非 production 但走 HTTPS"的部署误把 Secure 关掉导致 cookie 走 HTTP。
	sessionStore := session.NewStore(session.Config{
		Storage:         redisStorage,
		KeyGenerator:    secureSessionID,
		Extractor:       extractors.FromCookie("sforum_session"),
		IdleTimeout:     cfg.SessionIdleTimeout,
		AbsoluteTimeout: cfg.SessionAbsoluteTimeout,
		CookieHTTPOnly:  true,
		CookieSameSite:  fiber.CookieSameSiteLaxMode,
		CookiePath:      "/",
		CookieSecure:    shouldUseSecureCookie(cfg),
	})
	avatarOptions := avatarOptionsAdapter{options: optionsService}
	identityStore := identity.NewPostgresStoreWithAvatar(pool, avatarOptions)
	authSessions := authsession.NewManager(sessionStore, authsession.Config{
		RenewalInterval: cfg.SessionRenewalInterval,
		HashSecret:      cfg.SessionHashSecret,
		// M8：注入令牌版本号来源，使密码重置/封禁后旧会话失效。
		TokenVersion: func(ctx context.Context, userID int64) (int64, error) {
			return identityStore.GetUserTokenVersion(ctx, userID)
		},
		// 会话目录：登录时登记设备、CurrentUserID 校验是否被下线、logout 时标记。
		// identityStore 满足 authsession.SessionStore 接口（结构化匹配）。
		SessionStore: identityStore,
	})
	adminOverviewStore := adminoverview.NewPostgresStore(pool)
	forumStore := forum.NewPostgresStoreWithAvatar(pool, avatarOptions)
	// 业务读缓存复用 sharedRedisClient（与 humanverify 共享连接池）。
	// 失败不阻断启动——缓存为可重建的派生数据，降级为直连 PG。
	forumCachedStore := forum.NewCachedStore(forumStore, cache.NewRedisCache(sharedRedisClient))
	profileStore := profile.NewPostgresStore(pool)
	moderationStore := moderation.NewPostgresStore(pool)
	attachmentStore := attachments.NewPostgresStore(pool)
	// E6.1：附件服务先创建，稍后注入扩展目录与禁用回落。
	attachmentService := attachments.NewServiceWithEvents(attachmentStore, optionsService, nil)
	databaseStore := database.NewPostgresStore(pool)
	extensionStore := extensions.NewPostgresStore(pool)
	frontendTrustStore := extensions.NewPostgresFrontendTrustStore(pool)
	executableTrustStore := extensions.NewPostgresExecutableTrustStore(pool)
	activationCoordinator := extensions.NewActivationCoordinator(extensionStore).WithAuditor(auditWriter)
	// Host API 需要 extensionService 的能力解析；先建 service 再绑 gateway。
	// 运行时 manager 在 service 创建后注入 HostAPI registrar。
	jobClient, err := supportjobs.NewInsertOnlyClient(pool, supportjobs.FromAppConfig(cfg))
	if err != nil {
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("job dispatcher setup failed: %w", err)
	}
	jobDispatcher := supportjobs.NewDispatcher(jobClient)
	executableTrustService := extensions.NewExecutableTrustService(extensionStore, executableTrustStore).
		WithAuditor(auditWriter).
		WithTTL(cfg.TrustChallengeTTL)
	frontendService := extensions.NewFrontendService(extensionStore, frontendTrustStore).
		WithAuditor(auditWriter).
		WithSafeMode(cfg.SafeMode).
		WithExecutableTrust(executableTrustService, cfg.V3TrustChallenges).
		WithPublicL2(cfg.V3PublicL2)
	// Page Registry：运行时主题 L0/L1，主题激活不重建 Nuxt、不写 current.json。
	pageRegistryStore := pages.NewPostgresStore(pool)
	pageRegistry := pages.NewRegistry(pageRegistryStore)
	if err := pageRegistry.RestoreBindings(ctx); err != nil {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("restore page provider bindings failed: %w", err)
	}
	themeRuntime := pages.NewThemeRuntimeRegistry()
	themeSiteName, _ := optionsService.SiteName(ctx)
	if strings.TrimSpace(themeSiteName) == "" {
		themeSiteName = cfg.AppName
	}
	pageRegistryAdapter := extensions.NewPageRegistryAdapter(pageRegistry).
		WithThemeRuntime(themeRuntime, themeSiteName, cfg.SupportedLocales)
	// 先构造带 Cipher 的 Service，插件启动与 Host API 才能共享同一个解密设置源。
	// 主题激活走 Page Registry 同步路径；不再注入 theme_activate dispatcher。
	extensionService := extensions.NewServiceWithOptions(
		extensionStore, cfg.ExtensionRoot, cfg.BuiltinExtensionRoot,
		nil,
		extensions.WithAuditor(auditWriter),
		extensions.WithExecutableTrust(executableTrustService, cfg.V3TrustChallenges),
		extensions.WithSafeMode(cfg.SafeMode),
		extensions.WithActivationCoordinator(activationCoordinator),
		// F2.4：同 id 升级且 digest 变化时吊销该扩展前端信任，要求重新授权。
		extensions.WithTrustRevoker(frontendService),
		// F4.5：启用时校验 manifest requiresFeatures。
		extensions.WithFeatureFlags(optionsService),
		// 扩展 secret 设置与 web_options 共用 AES-GCM 密钥。
		extensions.WithCipher(optionCipher),
		// E6.1：禁用声明存储槽位的插件时，attachment.provider 从 plugin:<id> 回落 local。
		extensions.WithStorageSelectionClearer(attachmentService),
		// 运行时 Page Registry（主题激活同步注册页面贡献）。
		extensions.WithPageRegistry(pageRegistryAdapter),
	)
	// F2.2 Host API 与 ProtocolStarter 共用 extensionService；错误密钥不得降级读取 store。
	hostAPIService := hostapi.New(hostapi.Config{
		Settings: extensionService,
		Jobs:     &hostapi.RiverJobEnqueuer{Dispatcher: jobDispatcher},
		Auditor:  auditWriter,
	})
	hostAPIGateway := hostapi.NewGateway(hostAPIService)
	databaseCatalogBinder := postgresProtocolV2DatabaseCatalogBinder{
		pool: pool, gateway: hostAPIGateway,
		options: []hostapi.ProtocolV2DatabaseRuntimeOption{
			hostapi.WithProtocolV2DatabaseTraceSink(hostapi.NewSlogDatabaseTraceSink(logger)),
		},
	}
	queryAuthority := hostapi.NewPostgresProtocolV2QueryAuthorityResolver(pool)
	queryRuntime, err := hostapi.NewPostgresProtocolV2QueryRuntime(
		pool, queryAuthority,
		hostapi.WithProtocolV2QueryTraceSink(hostapi.NewSlogQueryTraceSink(logger)),
	)
	if err == nil {
		err = hostAPIGateway.BindProtocolV2QueryRuntime(queryRuntime)
	}
	if err != nil {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("Host Query runtime setup failed: %w", err)
	}
	// 六域 Host Command 必须在首个插件 broker 注册前冻结；否则同一启动中
	// 已运行插件会看到缺失目录，而后启动插件看到另一份能力集合。
	if err := bindPostgresProtocolV2CommandRuntime(
		hostAPIGateway, pool, jobDispatcher, moderationStore, attachmentStore,
	); err != nil {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("Host Command runtime setup failed: %w", err)
	}
	extensionRuntime := bindAPIExtensionRuntime(
		extensionStore, hostAPIGateway, extensionService, executableTrustService, databaseLeaseRegistry,
	)
	if runtime, ok := extensionRuntime.(interface {
		SetStartPreparer(func(context.Context, extensions.Extension) error)
	}); ok {
		runtime.SetStartPreparer(protocolV2DatabaseStartPreparer(extensionStore, databaseCatalogBinder, cfg.SafeMode))
	}
	hostAPIService.BindPluginJobAdmission(newPluginJobEnqueueAdmission(extensionRuntime))
	serviceAdmission := newPluginServiceProviderAdmission(extensionRuntime)
	hostAPIService.BindServiceProviderAdmission(serviceAdmission)
	lifecycleRuntime, err := requireProductionExtensionRuntime(extensionRuntime)
	if err != nil {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		extensionRuntime.Close(ctx)
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, err
	}
	hostCacheRuntime, err := bindProductionHostCache(
		hostInstallationID, logger, sharedRedisClient, cacheregistry.New(), serviceAdmission, hostAPIGateway,
	)
	if err != nil {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		extensionRuntime.Close(ctx)
		_ = hostAPIGateway.Close()
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("Host Cache runtime setup failed: %w", err)
	}
	if err := bindProtocolV2ProviderBroker(hostAPIGateway, lifecycleRuntime); err != nil {
		extensionRuntime.Close(ctx)
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, err
	}
	lifecycleRuntime.WithActivation(activationCoordinator, extensions.NewActivationBootID())
	lifecycleMigrationEngine := extensionsruntime.NewPostgresLifecycleMigrationEngine(pool, nil)
	lifecycleDatabaseDisposition := extensionsruntime.NewPostgresExtensionDatabaseDisposition(pool)
	lifecycleStack, err := newProductionLifecycleStack(productionLifecycleStackConfig{
		Pool: pool, Store: extensionStore, Features: optionsService,
		Trust: executableTrustService, Runtime: lifecycleRuntime, Pages: pageRegistry,
		ThemeRuntime: themeRuntime, PageSiteName: themeSiteName, PageLocales: cfg.SupportedLocales,
		Services: hostAPIGateway.ProtocolV2ServiceRegistry(), Caches: hostCacheRuntime.Registry, River: jobClient,
		ExtensionRoot: cfg.ExtensionRoot, MigrationEngine: lifecycleMigrationEngine,
		Database: lifecycleDatabaseDisposition, SafeMode: cfg.SafeMode,
	})
	if err != nil {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		extensionRuntime.Close(ctx)
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("extension lifecycle setup failed: %w", err)
	}
	frontendService.WithPublicComponentAdmission(lifecycleStack.ComponentRegistry)
	if err := lifecycleStack.bindService(extensionService); err != nil {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		extensionRuntime.Close(ctx)
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("bind extension lifecycle service failed: %w", err)
	}
	// 把已构造的 extensionService 接到 Host API 能力/权限解析（避免循环构造）。
	hostAPIService.BindCapabilitySource(extensionService)
	hostAPIService.BindPermissions(identityPermissionAdapter{store: identityStore})
	hostAPIService.BindUsers(identityUserAdapter{store: identityStore})
	if _, err := extensionService.SyncBuiltins(ctx); err != nil {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		extensionRuntime.Close(ctx)
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("sync builtin extensions failed: %w", err)
	}
	// DatabaseService 的 SQL catalog 必须在首次插件 Reconcile/broker 注册前冻结。
	if err := bindProductionProtocolV2DatabaseRuntime(ctx, databaseCatalogBinder, extensionStore, cfg.SafeMode); err != nil {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		extensionRuntime.Close(ctx)
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("DatabaseService runtime setup failed: %w", err)
	}
	// API 启动恢复：活动主题 L0/L1 + 已启用插件页面贡献。Asset Registry
	// 此时尚未注入 Service；唯一权威恢复在 runtime reconciliation 之后。
	// 无效主题安全回退默认；失败不得留下空 Registry 却 DB 指向主题的分裂状态。
	if cfg.SafeMode {
		if err := extensionService.RestoreSafeModeThemeRegistry(ctx); err != nil {
			logger.Warn("restore safe mode default theme registry failed", "error", err)
		}
	} else if err := extensionService.RestoreActiveThemeRegistry(ctx); err != nil {
		logger.Warn("restore active theme page registry failed", "error", err)
	}
	var themeRuntimeWatcher *extensions.ThemeRuntimeWatcher
	if !cfg.SafeMode {
		themeRuntimeWatcher, err = newAPIThemeRuntimeWatcher(extensionStore, extensionService, logger)
		if err == nil {
			err = themeRuntimeWatcher.Initialize(ctx)
		}
		if err != nil {
			if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
				logger.Warn("job dispatcher stop failed", "error", stopErr)
			}
			extensionRuntime.Close(ctx)
			sharedRedisClient.Close()
			if closeErr := redisStorage.Close(); closeErr != nil {
				logger.Warn("redis session storage close failed", "error", closeErr)
			}
			pool.Close()
			return nil, fmt.Errorf("initialize theme runtime watcher failed: %w", err)
		}
	}
	legacyMailValues, err := optionsService.InternalValues(ctx)
	if err != nil {
		return nil, fmt.Errorf("load legacy mail options: %w", err)
	}
	if err := notifications.NewPostgresStore(pool).AdoptLegacyMail(ctx, legacyMailValues); err != nil {
		return nil, fmt.Errorf("adopt legacy mail settings: %w", err)
	}
	reconciledExtensions, err := reconcileAPIExtensionRuntime(ctx, cfg.SafeMode, extensionStore, extensionRuntime)
	if err != nil {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		extensionRuntime.Close(ctx)
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("list extensions for runtime reconciliation failed: %w", err)
	}
	if err := lifecycleStack.Registries.RestoreRoutePublications(ctx, reconciledExtensions, cfg.SafeMode); err != nil {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		extensionRuntime.Close(ctx)
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("restore extension route publications failed: %w", err)
	}
	if err := lifecycleStack.bindAssetRegistryConsumers(
		extensionService, frontendService, executableTrustService,
	); err != nil {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		extensionRuntime.Close(ctx)
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("bind extension asset registry consumers failed: %w", err)
	}
	notificationStore := notifications.NewPostgresStore(pool)
	mailOutbox := notifications.NewOutbox(pool, notificationStore, jobDispatcher).WithPolicyReader(optionsService)
	forumStore.WithCommentNotifications(forumNotificationAdapter{outbox: mailOutbox})
	moderationStore.WithDecisionNotifications(moderationNotificationAdapter{outbox: mailOutbox})
	siteName, _ := optionsService.SiteName(ctx)
	siteURL, _ := optionsService.WebOption(ctx, "site.url")
	passwordResetService := identity.NewPasswordResetServiceWithPasswordPolicy(identityStore, passwordResetOutbox{outbox: mailOutbox}, identity.PasswordResetConfig{
		SiteName: siteName,
		SiteURL:  siteURL,
	}, optionsService).WithRateLimiter(authsupport.NewPasswordResetLimiter(sharedRedisClient))
	loginLockout := authsupport.NewLoginLockout(sharedRedisClient)
	// F3.3：出站 webhook 扇出包装在事件发布路径上（observe 成功后异步入队）。
	webhookStore := webhooks.NewPostgresStore(pool)
	// 生产默认仅 https；非生产允许 http 便于本地联调（连接时仍禁止私网 SSRF）。
	webhookService := webhooks.NewService(webhookStore, pool, jobDispatcher).
		WithAllowHTTP(!strings.EqualFold(cfg.AppEnv, "production")).
		WithCipher(optionCipher)
	eventPublisher := webhooks.BridgePublisher{Inner: extensionRuntime, Fanout: webhookService}
	// 与 extensionService 共享同一 attachmentService 实例（禁用回落 + 候选目录 + 事件 + 存储 RPC）。
	_ = attachmentService.WithEvents(eventPublisher).
		WithStorageProviderCatalog(extensionService).
		WithStoragePluginRuntime(extensionRuntime)

	// F3.4：个人访问令牌；管理走 cookie，调用走 Bearer。
	apiTokenStore := apitokens.NewPostgresStore(pool)
	apiTokenService := apitokens.NewService(apiTokenStore, identityStore).WithAuditor(auditWriter)

	identityProvider := providers.NewIdentityProviderWithPasswordResetAndLockout(identityStore, authSessions, humanVerifier, eventPublisher, passwordResetService, mailOutbox, optionsService, loginLockout).
		WithAPITokens(apiTokenService)
	notificationsProvider := providers.NewNotificationsProvider(notificationStore, identityStore, authSessions)
	mailProvider := providers.NewMailProvider(extensionStore, notificationStore, extensionsruntime.NewMailProviderRegistry(extensionStore), identityStore, authSessions, optionsService)
	// Worker 心跳 store 尽早创建，供 overview 与嵌入 worker 共用。
	heartbeatStore := health.NewRedisHeartbeatStore(sharedRedisClient)
	adminOverviewProvider := providers.NewAdminOverviewProviderWithWidgets(
		adminOverviewStore,
		adminoverview.NewRuntimeCollector(time.Now().UTC(), pool).
			WithHeartbeat(heartbeatStore).
			WithQueueLag(pool),
		identityStore,
		authSessions,
		providers.NewExtensionDashboardWidgetProvider(extensionService),
	)
	// 搜索：API 进程持有只入队的 indexer（EnqueueIndex/EnqueueDelete）和查询用的 search service。
	// Meilisearch client 不可达时，索引调度静默失败、搜索端点返回 503，主流程不受影响。
	meiliClient := search.NewClientWithTimeout(cfg.MeiliHost, cfg.MeiliMasterKey, cfg.MeiliTimeout)
	searchIndexer := search.NewIndexer(meiliClient, nil, jobDispatcher)
	forumSettingsResolver := providers.NewForumSettingsResolver(optionsService)
	searchService := search.NewService(meiliClient, forumSettingsResolver)
	// 搜索索引重建：forumStore 提供 ListAllTopicIDs（TopicIDSource），
	// reindexStore 记录运行状态，dispatcher 批量入队 IndexTopicArgs。
	reindexManager := search.NewReindexManager(forumStore, search.NewPostgresReindexStore(pool), jobDispatcher)

	// F3.2：发帖/评论写路径可选 Idempotency-Key；存储复用 shared Redis。
	idempotencyStore := idempotency.NewStore(idempotency.NewRedisBackend(sharedRedisClient), idempotency.DefaultTTL)
	forumProvider := providers.NewForumProviderWithPublicContributions(
		forumCachedStore,
		optionsService,
		identityStore,
		authSessions,
		eventPublisher,
		searchIndexer,
		searchServiceAdapter{inner: searchService},
		reindexServiceAdapter{inner: reindexManager},
		providers.NewExtensionTopicActionProvider(extensionService),
		providers.NewExtensionCommentActionProvider(extensionService),
		providers.NewExtensionTopicSurfaceProvider(extensionService),
		providers.NewExtensionComposerToolbarProvider(extensionService),
		providers.NewModerationPublicationPolicy(moderationStore, optionsService),
	).WithIdempotency(idempotencyStore)
	// 头像与附件管理共用带存储候选目录的服务实例。
	avatarAttachmentService := attachmentService
	profileProvider := providers.NewProfileProviderWithAvatarAndTabs(
		profileStore,
		identityStore,
		authSessions,
		avatarAttachmentService,
		optionsService,
		providers.NewExtensionProfileTabProvider(extensionService),
	)
	moderationProvider := providers.NewModerationWorkbenchProviderWithIndexer(moderationStore, forumStore, identityStore, authSessions, searchIndexer)
	optionsProvider := providers.NewOptionsProviderWithService(optionsService, identityStore, authSessions)
	siteChromeStore := sitechrome.NewPostgresStore(pool)
	// E2.3：公开顶栏合并 forum.nav.items（核心/运营项之后）。
	siteChromeProvider := providers.NewSiteChromeProviderWithExtensionNav(
		siteChromeStore,
		identityStore,
		authSessions,
		providers.NewExtensionNavItemProvider(extensionService),
	)
	attachmentsProvider := providers.NewAttachmentsProviderWithService(attachmentService, attachmentStore, identityStore, authSessions)
	seoProvider := providers.NewSEOProvider(pool, optionsService)
	databaseProvider := providers.NewDatabaseProvider(databaseStore, identityStore, authSessions)
	jobsProvider := providers.NewJobsProvider(pool, jobClient, identityStore, authSessions)
	routeTraceRing := routes.NewRouteTraceRing(0)
	extensionsProvider := providers.NewExtensionsProviderWithService(extensionService, identityStore, authSessions, extensionRuntime, frontendService).
		WithRouteProviderSelection(lifecycleStack.RouteProviders, auditWriter).
		WithRouteContractCatalog(lifecycleStack.RouteSchemas).
		WithProviderSlotSelection(lifecycleStack.ProviderSlots, lifecycleRuntime, auditWriter).
		WithRouteInspector(routes.NewProviderSelectionInspector(lifecycleStack.RouteProviders, routeTraceRing)).
		WithAdminSurfaces(extensionRuntime, auditWriter)
	webhooksProvider := providers.NewWebhooksProvider(webhookService, identityStore, authSessions)
	// PageDataLoader 网关：仅从运行中插件 RouteTarget 拉数据（严格 loopback）。
	pageLoaderGateway := pages.NewLoaderGateway(
		pages.NewPageDataLoader(nil),
		pageRouteTargetAdapter{runtime: extensionRuntime},
	).WithPackages(pagePackageRootAdapter{store: extensionStore})
	// Core Page ViewModels reuse the same domain services and policy sources as
	// their JSON endpoints. Only reviewed presentation DTOs cross into themes.
	pageForumService := forum.NewServiceWithSettingsAndEvents(forumCachedStore, forumSettingsResolver, eventPublisher)
	pageProfileService := profile.NewServiceWithAvatar(profileStore, avatarAttachmentService, optionsService).
		WithProfileTabs(providers.NewExtensionProfileTabProvider(extensionService))
	pageModerationService := moderation.NewServiceWithWorkbench(
		moderationStore, moderation.NewForumTargetValidator(forumStore), moderationStore, moderationStore,
	)
	pageIdentityService := identity.NewServiceWithPolicies(identityStore, eventPublisher, optionsService, optionsService)
	pageSiteChromeService := sitechrome.NewService(siteChromeStore).
		WithExtensionNavItems(providers.NewExtensionNavItemProvider(extensionService))
	corePageViews := pageviewmodels.NewCorePageViewModelSource(pageviewmodels.CorePageViewModelDependencies{
		Forum: pageForumService, Profiles: pageProfileService, Notifications: notificationStore,
		Moderation: pageModerationService, Options: optionsService, Registration: pageIdentityService,
		Sessions: identityStore, SiteChrome: pageSiteChromeService, Search: searchService,
	})
	pagesProvider := providers.NewPagesProviderWithThemes(pageRegistry, identityStore, authSessions, extensionStore).
		WithAuditor(auditWriter).
		WithLoader(pageLoaderGateway).
		WithThemeRuntime(themeRuntime).
		WithCorePageViewModels(corePageViews)

	// F4.4：实体自定义字段（EAV，无 per-plugin core ALTER）。
	entityMetaStore := entitymeta.NewPostgresStore(pool)
	entityMetaService := entitymeta.NewService(entityMetaStore).WithPublisher(eventPublisher)
	entityMetaProvider := providers.NewEntityMetaProvider(entityMetaService, identityStore, authSessions)

	// Readiness：PG 必检；Redis/Meili 失败记 degraded 仍 ready（见 Support/Health）。
	// F4.3：合并 system.health.checks 贡献（不调用插件 RPC）。
	readyEvaluate := func(ctx context.Context) health.ReadyReport {
		return health.EvaluateWithExtensionContributions(ctx, []health.Checker{
			health.PostgresChecker{Pool: pool},
			health.RedisChecker{Client: sharedRedisClient},
			health.MeiliChecker{Client: meiliClient},
		}, extensionService, extensionRuntime)
	}
	extensionGuardPolicy := extensions.NewGuardPolicyCatalog(
		extensionStore,
		executableTrustService,
		frontendTrustStore,
		extensions.GuardPolicyConfig{
			SafeMode: cfg.SafeMode, TrustChallengesEnabled: cfg.V3TrustChallenges,
		},
	)
	if err := extensionGuardPolicy.Refresh(ctx); err != nil {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		extensionRuntime.Close(ctx)
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("refresh extension guard policy failed: %w", err)
	}
	// P6 buffered HTTP dispatcher consumes the durable provider selection before
	// Fiber's hardcoded core providers. Declared schemas remain fail-closed until
	// their exact package catalog is production-published.
	routeDispatcher := routes.NewDispatcher(routes.DispatcherConfig{
		Plans: lifecycleStack.RouteProviders,
		Steps: httpserver.NewBufferedRouteStepInvoker(lifecycleStack.RuntimeManager),
		Guard: httpserver.NewProductionRouteGuardAuthorizerWithPolicies(httpserver.ProductionRouteGuardPolicies{
			ForumRead:         optionsService,
			Extensions:        extensionGuardPolicy,
			DeclaredRoutes:    extensionGuardPolicy,
			Options:           optionsService,
			Pages:             pageRegistry,
			IdentityAdmins:    identityStore,
			IdentitySessions:  identityStore,
			IdentityAPITokens: apiTokenStore,
			EntityMetaValues:  entityMetaStore,
			AttachmentReads:   attachmentStore,
			ForumComments:     forumStore,
			ForumResources:    forumStore,
			PluginGuards:      httpserver.NewRuntimePluginRouteGuardEvaluator(lifecycleStack.RuntimeManager, extensionGuardPolicy),
		}),
		Schemas:     httpserver.CatalogRouteSchemaValidator{Catalog: lifecycleStack.RouteSchemas},
		Trace:       routeTraceRing,
		Policies:    lifecycleStack.RouteSchemas,
		Idempotency: httpserver.NewRequiredRouteIdempotency(idempotencyStore),
	})

	app := httpserver.NewApp(cfg, logger, httpserver.Dependencies{
		RouteProviders:  []httpserver.RouteProvider{identityProvider, notificationsProvider, mailProvider, adminOverviewProvider, forumProvider, profileProvider, moderationProvider, optionsProvider, siteChromeProvider, attachmentsProvider, seoProvider, databaseProvider, jobsProvider, extensionsProvider, webhooksProvider, entityMetaProvider, pagesProvider},
		RoutePlans:      lifecycleStack.RouteProviders,
		RouteDispatcher: routeDispatcher,
		RouteActors: func(c fiber.Ctx) (identity.Actor, error) {
			return httpserver.OptionalActor(c, authSessions, identityStore)
		},
		Options:      optionsService,
		Storage:      redisStorage,
		Ready:        readyEvaluate,
		BearerTokens: httpserver.TokenServiceAdapter{Service: apiTokenService},
		Auditor:      auditWriter,
	})

	// Worker 心跳：嵌入 worker 时由 API 进程发布；独立 worker 在 NewWorker 内发布。
	// 未嵌入时 overview 仍可读独立 worker 写入的同一 Redis key。
	heartbeatCtx, heartbeatCancel := context.WithCancel(context.Background())

	var embeddedWorker *Worker
	if shouldEmbedWorkerInAPI(cfg) {
		// Embed 时复用 API 已 Reconcile 的 extensionRuntime，避免每个后端插件双起子进程。
		// OwnsRuntime=false：Worker.Close 不关 runtime；API close 在 River stop 之后再关。
		embeddedWorker, err = newWorkerWithPool(cfg, pool, logger, workerRuntimeDeps{
			ExtensionRuntime: extensionRuntime,
			PluginSchedules:  lifecycleStack.Schedules,
			OwnsRuntime:      false,
		})
		if err != nil {
			heartbeatCancel()
			if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
				logger.Warn("job dispatcher stop failed", "error", stopErr)
			}
			extensionRuntime.Close(ctx)
			sharedRedisClient.Close()
			if closeErr := redisStorage.Close(); closeErr != nil {
				logger.Warn("redis session storage close failed", "error", closeErr)
			}
			pool.Close()
			return nil, fmt.Errorf("embedded worker setup failed: %w", err)
		}
		if err := embeddedWorker.Start(ctx); err != nil {
			heartbeatCancel()
			embeddedWorker.Close()
			if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
				logger.Warn("job dispatcher stop failed", "error", stopErr)
			}
			extensionRuntime.Close(ctx)
			sharedRedisClient.Close()
			if closeErr := redisStorage.Close(); closeErr != nil {
				logger.Warn("redis session storage close failed", "error", closeErr)
			}
			pool.Close()
			return nil, fmt.Errorf("embedded worker start failed: %w", err)
		}
		go (&health.Publisher{Store: heartbeatStore}).Run(heartbeatCtx)
		logger.InfoContext(ctx, "embedded api worker started")
	}
	forumReadPolicyCtx, forumReadPolicyCancel := context.WithCancel(context.Background())
	go optionsService.RunForumReadPolicyRefresh(forumReadPolicyCtx, options.RecommendedForumReadPolicyRefreshInterval)
	extensionGuardPolicyCtx, extensionGuardPolicyCancel := context.WithCancel(context.Background())
	go extensionGuardPolicy.RunRefresh(extensionGuardPolicyCtx, extensions.RecommendedGuardPolicyRefreshInterval)
	databaseLeaseReaperCtx, databaseLeaseReaperCancel := context.WithCancel(context.Background())
	var databaseLeaseReaperWait sync.WaitGroup
	databaseLeaseReaperWait.Add(1)
	go func() {
		defer databaseLeaseReaperWait.Done()
		ticker := time.NewTicker(extensionsruntime.RecommendedExtensionDatabaseRuntimeLeaseReapInterval)
		defer ticker.Stop()
		for {
			select {
			case <-databaseLeaseReaperCtx.Done():
				return
			case <-ticker.C:
				if _, err := databaseLeaseRegistry.ReapExpiredRuntimeLeases(
					databaseLeaseReaperCtx, extensionsruntime.DefaultExtensionDatabaseRuntimeLeaseReapLimit,
				); err != nil && databaseLeaseReaperCtx.Err() == nil {
					logger.Warn("extension database runtime lease reaper degraded", "error", err)
				}
			}
		}
	}()
	themeRuntimeWatcherCtx, themeRuntimeWatcherCancel := context.WithCancel(context.Background())
	var themeRuntimeWatcherWait sync.WaitGroup
	if themeRuntimeWatcher != nil {
		themeRuntimeWatcherWait.Add(1)
		go func() {
			defer themeRuntimeWatcherWait.Done()
			if err := themeRuntimeWatcher.Run(themeRuntimeWatcherCtx); err != nil {
				logger.Error("theme runtime watcher stopped", "error", err)
			}
		}()
	}

	return &API{
		App:  app,
		Addr: apiAddress(cfg),
		close: func() {
			themeRuntimeWatcherCancel()
			themeRuntimeWatcherWait.Wait()
			databaseLeaseReaperCancel()
			databaseLeaseReaperWait.Wait()
			extensionGuardPolicyCancel()
			forumReadPolicyCancel()
			heartbeatCancel()
			if embeddedWorker != nil {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.WorkerShutdownTimeout)
				defer cancel()
				if err := embeddedWorker.Stop(shutdownCtx); err != nil {
					logger.Warn("embedded worker stop failed", "error", err)
				}
				embeddedWorker.Close()
			}
			if err := supportjobs.Stop(context.Background(), jobClient); err != nil {
				logger.Warn("job dispatcher stop failed", "error", err)
			}
			extensionRuntime.Close(context.Background())
			_ = hostAPIGateway.Close()
			if err := redisStorage.Close(); err != nil {
				logger.Warn("redis session storage close failed", "error", err)
			}
			// sharedRedisClient 被 humanverify 与 forum 缓存共用，关闭一次即可。
			if err := sharedRedisClient.Close(); err != nil {
				logger.Warn("shared redis client close failed", "error", err)
			}
			pool.Close()
		},
	}, nil
}

func secureSessionID() string {
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		panic(fmt.Errorf("generate session id: %w", err))
	}
	return base64.RawURLEncoding.EncodeToString(token[:])
}

// shouldUseSecureCookie 委托 config 包，与 CSRF CookieSecure 共用同一判定。
func shouldUseSecureCookie(cfg config.Config) bool {
	return config.ShouldUseSecureCookie(cfg)
}

func (api *API) Close() {
	if api == nil {
		return
	}
	api.closeOnce.Do(func() {
		if api.close != nil {
			api.close()
		}
	})
}

// identityPermissionAdapter 将 identity store 接到 Host API 权限检查。
type identityPermissionAdapter struct {
	store *identity.PostgresStore
}

func (a identityPermissionAdapter) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	if a.store == nil {
		return false, fmt.Errorf("identity store unavailable")
	}
	actor, err := a.store.LoadActor(ctx, userID)
	if err != nil {
		return false, err
	}
	return actor.Can(permission), nil
}

// identityUserAdapter 提供安全用户字段给 Host API。
type identityUserAdapter struct {
	store *identity.PostgresStore
}

func (a identityUserAdapter) GetUserSafe(ctx context.Context, userID int64) (map[string]any, error) {
	if a.store == nil {
		return nil, fmt.Errorf("identity store unavailable")
	}
	user, err := a.store.GetCurrentUser(ctx, userID)
	if err != nil {
		return nil, hostapi.ErrNotFound
	}
	return map[string]any{
		"id":          user.ID,
		"username":    user.Username,
		"displayName": user.DisplayName,
		"status":      user.Status,
	}, nil
}

func apiAddress(cfg config.Config) string {
	return fmt.Sprintf("%s:%s", cfg.HTTPHost, cfg.HTTPPort)
}

func newHumanVerifyService(cfg config.Config, store humanverify.Store) (*humanverify.Service, error) {
	return humanverify.NewConfiguredService(humanVerifyConfigFromConfig(cfg), store)
}

type avatarOptionsAdapter struct {
	options *options.Service
}

func (a avatarOptionsAdapter) AvatarOptions(ctx context.Context) (avatar.Options, error) {
	if a.options == nil {
		return avatar.DefaultOptions(), nil
	}
	resolved, err := a.options.AvatarOptions(ctx)
	if err != nil {
		return avatar.Options{}, err
	}
	return avatar.Options{
		AllowUpload:           resolved.AllowUpload,
		DefaultProvider:       resolved.DefaultProvider,
		GravatarBaseURL:       resolved.GravatarBaseURL,
		GravatarHashAlgorithm: resolved.GravatarHashAlgorithm,
		DefaultStaticURL:      resolved.DefaultStaticURL,
		MaxSizeKB:             resolved.MaxSizeKB,
		MaxDimension:          resolved.MaxDimension,
		AllowGIF:              resolved.AllowGIF,
		CompressEnabled:       resolved.CompressEnabled,
		TargetDimension:       resolved.TargetDimension,
		CompressQuality:       resolved.CompressQuality,
	}, nil
}

func optionsDefaultsFromConfig(cfg config.Config) options.Defaults {
	return options.Defaults{
		SiteName:                  cfg.AppName,
		SiteURL:                   cfg.AppURL,
		DefaultLocale:             cfg.AppLocale,
		SupportedLocales:          cfg.SupportedLocales,
		HumanVerificationProvider: cfg.HumanVerificationProvider,
		AltchaSecret:              cfg.AltchaSecret,
		AltchaChallengeTTL:        cfg.AltchaChallengeTTL,
		AltchaCost:                cfg.AltchaCost,
	}
}

func humanVerifyConfigFromConfig(cfg config.Config) humanverify.RuntimeConfig {
	return humanverify.RuntimeConfig{
		Provider:        cfg.HumanVerificationProvider,
		AltchaSecret:    cfg.AltchaSecret,
		AltchaTTL:       cfg.AltchaChallengeTTL,
		AltchaCost:      cfg.AltchaCost,
		RateLimit:       60,
		RateLimitWindow: time.Minute,
	}
}

// pageRouteTargetAdapter 为 PageDataLoader 绑定 exact runtime admission lease。
type pageRouteTargetAdapter struct {
	runtime extensionRuntime
}

func (a pageRouteTargetAdapter) AcquireRouteTarget(ctx context.Context, artifact pages.RuntimeArtifact) (pages.LoaderRouteTarget, bool) {
	if a.runtime == nil {
		return pages.LoaderRouteTarget{}, false
	}
	target, admission, err := a.runtime.AcquireActiveRuntimeCall(ctx, artifact.ExtensionID, extensionsruntime.RuntimeCallPage)
	if err != nil {
		return pages.LoaderRouteTarget{}, false
	}
	if artifact.ExtensionVersion != "" && target.ExtensionVersion != artifact.ExtensionVersion ||
		artifact.PackageDigest != "" && target.ArtifactDigest != artifact.PackageDigest ||
		artifact.RuntimeInstanceID != "" && target.Identity.InstanceID != artifact.RuntimeInstanceID {
		admission.Release()
		return pages.LoaderRouteTarget{}, false
	}
	if strings.TrimSpace(target.Target.BaseURL) == "" {
		admission.Release()
		return pages.LoaderRouteTarget{}, false
	}
	return pages.LoaderRouteTarget{
		BaseURL: target.Target.BaseURL,
		Context: admission.Context,
		Release: admission.Release,
	}, true
}

// pagePackageRootAdapter 解析扩展包内容根（loader schema 文件）。
type pagePackageRootAdapter struct {
	store extensions.Store
}

func (a pagePackageRootAdapter) PackageRoot(extensionID string) (string, bool) {
	if a.store == nil {
		return "", false
	}
	item, err := a.store.Get(context.Background(), extensionID)
	if err != nil {
		return "", false
	}
	root := extensions.PackageContentRoot(item)
	if root == "" {
		return "", false
	}
	return root, true
}

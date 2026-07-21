package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/session"

	redisstorage "github.com/gofiber/storage/redis/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
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
	providers "github.com/zhuchunshu/sforum/apps/api/app/Providers"
	audit "github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	authsupport "github.com/zhuchunshu/sforum/apps/api/app/Support/Auth"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	cache "github.com/zhuchunshu/sforum/apps/api/app/Support/Cache"
	cacheregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/CacheRegistry"
	contentregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/ContentRegistry"
	crypto "github.com/zhuchunshu/sforum/apps/api/app/Support/Crypto"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	health "github.com/zhuchunshu/sforum/apps/api/app/Support/Health"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	idempotency "github.com/zhuchunshu/sforum/apps/api/app/Support/Idempotency"
	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
	installationidentity "github.com/zhuchunshu/sforum/apps/api/app/Support/InstallationIdentity"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	pages "github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
	postgres "github.com/zhuchunshu/sforum/apps/api/app/Support/Postgres"
	redisplatform "github.com/zhuchunshu/sforum/apps/api/app/Support/Redis"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
	seoregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/SEORegistry"
	search "github.com/zhuchunshu/sforum/apps/api/app/Support/Search"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

// Production API 分阶段装配（NewAPI 见 app.go 薄封装）。

// apiCoreStack 是 foundation+domain 装配完成后交给 HTTP/worker 阶段的依赖包。
// 失败路径的资源关闭仍在 wireAPICoreStack 内联处理；成功后由 API.close 统一收尾。
type apiCoreStack struct {
	adminOverviewProvider        *providers.AdminOverviewProvider
	apiTokenService              *apitokens.Service
	attachmentsProvider          *providers.AttachmentsProvider
	auditWriter                  audit.Writer
	authSessions                 *authsession.Manager
	closePluginRuntime           func()
	closeRouteFailureRecorder    func()
	databaseLeaseRegistry        *extensionsruntime.PostgresExtensionDatabaseRegistry
	databaseProvider             *providers.DatabaseProvider
	entityMetaProvider           *providers.EntityMetaProvider
	extensionGuardPolicy         *extensions.GuardPolicyCatalog
	extensionRuntime             extensionRuntime
	extensionService             *extensions.Service
	extensionStore               *extensions.PostgresStore
	extensionsProvider           *providers.ExtensionsProvider
	forumProvider                *providers.ForumProvider
	heartbeatStore               *health.RedisHeartbeatStore
	hostAPIGateway               *hostapi.Gateway
	hostInstallationID           string
	identityProvider             *providers.IdentityProvider
	identityStore                *identity.PostgresStore
	jobClient                    *supportjobs.Client
	jobsProvider                 *providers.JobsProvider
	lifecycleStack               *productionLifecycleStack
	mailProvider                 *providers.MailProvider
	moderationProvider           *providers.ModerationProvider
	notificationsProvider        *providers.NotificationsProvider
	optionsProvider              *providers.OptionsProvider
	optionsService               *options.Service
	pagesProvider                *providers.PagesProvider
	pluginRuntimeCoordinator     *pluginRuntimeCoordinatorRuntime
	pluginRuntimeStopTimeout     time.Duration
	pool                         *pgxpool.Pool
	productionSEO                *productionSEORegistry
	profileProvider              *providers.ProfileProvider
	queryResultCache             *productionQueryResultCacheRuntime
	readyEvaluate                func(context.Context) health.ReadyReport
	redisStorage                 *redisstorage.Storage
	routeDispatcher              *routes.Dispatcher
	seoProvider                  *providers.SEOProvider
	sharedRedisClient            *redis.Client
	siteChromeProvider           *providers.SiteChromeProvider
	stopPluginRuntimeCoordinator func()
	webhooksProvider             *providers.WebhooksProvider
}

func buildProductionAPI(ctx context.Context, cfg config.Config, logger *slog.Logger) (*API, error) {
	core, err := wireAPICoreStack(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}
	return finishAPIHTTP(ctx, cfg, logger, core)
}

// wireAPICoreStack：迁移、连接池、会话、扩展 lifecycle、领域 provider 与路由分发器。
func wireAPICoreStack(ctx context.Context, cfg config.Config, logger *slog.Logger) (*apiCoreStack, error) {
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
	queryCursorSecret, err := deriveQueryRegistryCursorSecret(cfg.SessionHashSecret, hostInstallationID)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("derive Query Registry cursor secret: %w", err)
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
	queryResultCache, err := loadOptionalProductionQueryResultCache(ctx, cfg, hostInstallationID, logger)
	if err != nil {
		_ = sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("Query result cache setup failed: %w", err)
	}
	// 阶段移交：成功时 HandOff 给 finishAPIHTTP；失败路径由 defer 关闭，避免泄漏。
	queryCacheOwnership := newQueryResultCacheStageHandoff(queryResultCache, logger)
	defer queryCacheOwnership.CloseUnlessHandedOff()
	humanVerifyStore := humanverify.NewRedisStore(sharedRedisClient)
	optionStore := options.NewPostgresStore(pool)
	// H2a：构造敏感值加密器；未配置 APP_OPTION_ENC_KEY 时为透明模式（开发环境），生产由 C2 强制要求。
	optionCipher, err := crypto.NewOptionCipher(cfg.OptionEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create option cipher: %w", err)
	}
	requiredReplayCipher, err := idempotency.NewRequiredReplayCipher(cfg.OptionEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create required replay cipher: %w", err)
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
	seoSettings, err := optionsService.RuntimeSettings(ctx)
	if err != nil {
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("load SEO Host policy settings: %w", err)
	}
	seoValues, err := optionsService.InternalValues(ctx)
	if err != nil {
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("load SEO Host policy options: %w", err)
	}
	if strings.TrimSpace(seoSettings.SiteURL) == "" {
		seoSettings.SiteURL = cfg.AppURL
	}
	if len(seoSettings.SupportedLocales) == 0 {
		seoSettings.SupportedLocales = append([]string(nil), cfg.SupportedLocales...)
	}
	seoHostPolicy := seoregistry.HostFinalPolicyConfig{
		SiteURL: seoSettings.SiteURL, SupportedLocales: seoSettings.SupportedLocales,
		AllowIndexing:         productionSEOEnabled(seoValues[options.NameSEOAllowIndexing], true),
		SitemapEnabled:        productionSEOEnabled(seoValues[options.NameSEOSitemapEnabled], true),
		StructuredDataEnabled: productionSEOEnabled(seoValues[options.NameSEOSchemaOrgEnabled], true),
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
	// Session policy renewal gate is bound after the lifecycle stack publishes
	// the Identity Registry and Session Policy Store. Until then renew is a
	// Core-equivalent no-op; revocation never uses this gate.
	sessionPolicyRenewal := &sessionPolicyRenewalGate{}
	// Host Command catalog freezes before lifecycle; adopt the Session Policy
	// writer fence once the store is published so status commands cannot race
	// accepted session effects.
	identityAuthorityGate := &deferredIdentityAuthorityMutationGate{}
	authSessions := authsession.NewManager(sessionStore, authsession.Config{
		RenewalInterval: cfg.SessionRenewalInterval,
		HashSecret:      cfg.SessionHashSecret,
		// M8：注入令牌版本号来源，使密码重置/封禁后旧会话失效。
		TokenVersion: func(ctx context.Context, userID int64) (int64, error) {
			return identityStore.GetUserTokenVersion(ctx, userID)
		},
		// 会话目录：登录时登记设备、CurrentUserID 校验是否被下线、logout 时标记。
		// identityStore 满足 authsession.SessionStore 接口（结构化匹配）。
		SessionStore:      identityStore,
		RenewalEffectGate: sessionPolicyRenewal.Evaluate,
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
			hostapi.WithProtocolV2DatabaseQueryInvalidationJobs(jobDispatcher),
		},
	}
	queryTraceSink := hostapi.NewSlogQueryTraceSink(logger)
	queryAuthority := hostapi.NewPostgresProtocolV2QueryAuthorityResolver(pool)
	queryRuntime, err := hostapi.NewPostgresProtocolV2QueryRuntime(
		pool, queryAuthority,
		hostapi.WithProtocolV2QueryTraceSink(queryTraceSink),
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
		hostAPIGateway, pool, jobDispatcher, moderationStore, attachmentStore, identityAuthorityGate,
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
		ExtensionRoot: cfg.ExtensionRoot, QueryCursorSecret: queryCursorSecret, MigrationEngine: lifecycleMigrationEngine,
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
	identityReviewStore, ok := lifecycleStack.IdentityStore.(identityregistry.Store)
	if !ok {
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
		return nil, fmt.Errorf("extension lifecycle identity store does not expose Host review operations")
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
	// Query Registry 必须在首个插件 broker 注册前绑定。它复用 lifecycle
	// stack 的不可变 Core snapshot，并让 caller/provider 都经过 exact-runtime gate。
	if _, err := bindProductionQueryRegistryWithCache(
		lifecycleStack.QueryRegistry,
		lifecycleStack.QueryCoreCatalog,
		queryRuntime,
		identityStore,
		lifecycleRuntime,
		hostAPIGateway,
		queryTraceSink,
		queryResultCache.Cache(),
	); err != nil {
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
		return nil, fmt.Errorf("Query Registry production setup failed: %w", err)
	}
	productionSEO, err := bindProductionSEORegistry(
		lifecycleStack.SEORegistry,
		lifecycleRuntime,
		seoHostPolicy,
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
		return nil, fmt.Errorf("SEO Registry production setup failed: %w", err)
	}
	// 把已构造的 extensionService 接到 Host API 能力/权限解析（避免循环构造）。
	hostAPIService.BindCapabilitySource(extensionService)
	hostAPIService.BindExtensionInventory(extensions.HostInventoryAdapter{Service: extensionService})
	hostAPIGateway.BindCommandCapabilitySource(extensionService)
	hostAPIService.BindPermissions(identityPermissionAdapter{store: identityStore})
	// Identity user-field digests are installation-bound. GetUserSafe resolves
	// extension declared_fields through the Host-owned store with live actor
	// permission checks; core safe fields remain available without the store.
	userFieldDigestKey, err := deriveIdentityUserFieldDigestKey(cfg.SessionHashSecret, hostInstallationID)
	if err != nil {
		return nil, fmt.Errorf("derive identity user-field digest key: %w", err)
	}
	userFieldValueStore, err := identity.NewPostgresIdentityUserFieldValueStore(
		pool, lifecycleStack.IdentityRegistry, userFieldDigestKey,
	)
	if err != nil {
		return nil, fmt.Errorf("create identity user-field value store: %w", err)
	}
	hostAPIService.BindUsers(identityUserAdapter{store: identityStore, fields: userFieldValueStore})
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
	legacyMailValues, err := optionsService.InternalValues(ctx)
	if err != nil {
		return nil, fmt.Errorf("load legacy mail options: %w", err)
	}
	if err := notifications.NewPostgresStore(pool).AdoptLegacyMail(ctx, legacyMailValues); err != nil {
		return nil, fmt.Errorf("adopt legacy mail settings: %w", err)
	}
	var pluginRuntimeCoordinator *pluginRuntimeCoordinatorRuntime
	pluginRuntimeStopTimeout := normalizedPluginRuntimeCoordinatorStopTimeout(cfg.WorkerShutdownTimeout)
	stopPluginRuntimeCoordinator := func() {
		if pluginRuntimeCoordinator == nil {
			return
		}
		if stopErr := pluginRuntimeCoordinator.Stop(context.Background()); stopErr != nil && logger != nil {
			logger.Warn("plugin runtime coordinator stop failed", "error", stopErr)
		}
	}
	var closePluginRuntimeOnce sync.Once
	closePluginRuntime := func() {
		closePluginRuntimeOnce.Do(func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), pluginRuntimeStopTimeout)
			defer cancel()
			extensionRuntime.Close(shutdownCtx)
		})
	}
	var reconciledExtensions []extensions.Extension
	if cfg.SafeMode {
		reconciledExtensions, err = reconcileAPIExtensionRuntime(ctx, true, extensionStore, extensionRuntime)
	} else {
		pluginRuntimeCoordinator, err = startPluginRuntimeCoordinator(ctx, pluginRuntimeCoordinatorBootstrapConfig{
			ProcessRole: extensions.PluginRuntimeProcessAPI,
			Store:       extensionStore,
			Manager:     lifecycleRuntime,
			Logger:      logger,
			StopTimeout: cfg.WorkerShutdownTimeout,
		})
		if err == nil {
			reconciledExtensions, err = extensionStore.List(ctx)
		}
	}
	if err != nil {
		stopPluginRuntimeCoordinator()
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		closePluginRuntime()
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("start exact plugin runtime reconciliation failed: %w", err)
	}
	if err := lifecycleStack.Registries.RestoreRoutePublications(ctx, reconciledExtensions, cfg.SafeMode); err != nil {
		stopPluginRuntimeCoordinator()
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		closePluginRuntime()
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
		stopPluginRuntimeCoordinator()
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		closePluginRuntime()
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
	// MediaRegistry MIME 策略在已发布时叠加；无策略时 no-op。
	_ = attachmentService.WithEvents(eventPublisher).
		WithStorageProviderCatalog(extensionService).
		WithStoragePluginRuntime(extensionRuntime).
		WithMediaRegistry(lifecycleStack.MediaRegistry)

	// F3.4：个人访问令牌；管理走 cookie，调用走 Bearer。
	apiTokenStore := apitokens.NewPostgresStore(pool)
	apiTokenService := apitokens.NewService(apiTokenStore, identityStore).WithAuditor(auditWriter)

	sessionPolicyStepUpStore, err := identity.NewPostgresSessionPolicyStepUpStore(pool)
	if err != nil {
		return nil, fmt.Errorf("create session policy step-up store: %w", err)
	}
	sessionPolicyEvaluator, err := newSessionPolicyEvaluator(
		lifecycleStack.RuntimeManager,
		lifecycleStack.IdentityRegistry,
		lifecycleStack.SessionPolicyStore,
		sessionPolicyStepUpStore,
	)
	if err != nil {
		return nil, err
	}
	riskEvaluator, err := newRiskEvaluator(lifecycleStack.RuntimeManager, lifecycleStack.IdentityRegistry)
	if err != nil {
		return nil, err
	}
	externalLinkStore := identity.NewPostgresExternalIdentityLinkStore(pool)
	authProviderFlow, err := newAuthProviderFlow(
		lifecycleStack.RuntimeManager, lifecycleStack.IdentityRegistry, externalLinkStore,
	)
	if err != nil {
		return nil, err
	}
	profileProviderComposer, err := newProfileProviderComposer(
		lifecycleStack.RuntimeManager, lifecycleStack.IdentityRegistry,
	)
	if err != nil {
		return nil, err
	}
	recoveryProviderFlow, err := newRecoveryProviderFlow(
		lifecycleStack.RuntimeManager, lifecycleStack.IdentityRegistry,
	)
	if err != nil {
		return nil, err
	}
	identityStore.WithAuthorityMutationGate(lifecycleStack.SessionPolicyStore)
	identityAuthorityGate.Set(lifecycleStack.SessionPolicyStore)
	sessionPolicyRenewal.Set(sessionPolicyEvaluator)
	identityProvider := providers.NewIdentityProviderWithPasswordResetAndLockout(identityStore, authSessions, humanVerifier, eventPublisher, passwordResetService, mailOutbox, optionsService, loginLockout).
		WithIdentityRegistryStore(identityReviewStore).
		WithSessionPolicyEvaluator(sessionPolicyEvaluator).
		WithRiskEvaluator(riskEvaluator).
		WithAuthProviderFlow(authProviderFlow).
		WithProfileProviderComposer(profileProviderComposer).
		WithRecoveryProviderFlow(recoveryProviderFlow).
		WithIdentityProviderCatalog(lifecycleStack.IdentityRegistry).
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
	// 搜索：Host 框架 + 默认站内引擎（PG FTS）；外部引擎经 search.provider 插件。
	searchProviders := extensionsruntime.NewSearchProviderRegistry(extensionStore)
	siteEngine := search.NewPostgresSiteEngine(pool)
	searchEngine := extensionsruntime.NewResolvingSearchEngine(searchProviders, extensionRuntime, siteEngine)
	searchIndexer := search.NewIndexer(searchEngine, nil, jobDispatcher)
	forumSettingsResolver := providers.NewForumSettingsResolver(optionsService)
	searchService := search.NewService(searchEngine, forumSettingsResolver)
	// 搜索索引重建：forumStore 提供 ListAllTopicIDs（TopicIDSource），
	// reindexStore 记录运行状态，dispatcher 批量入队 IndexTopicArgs。
	reindexManager := search.NewReindexManager(forumStore, search.NewPostgresReindexStore(pool), jobDispatcher)

	// F3.2：发帖/评论写路径可选 Idempotency-Key；存储复用 shared Redis。
	idempotencyStore := idempotency.NewStore(
		idempotency.NewRedisBackend(sharedRedisClient), idempotency.DefaultTTL,
	).WithRequiredReplayCipher(requiredReplayCipher)
	// D3：公开详情浏览计数（与 worker 刷盘共用同一 Redis）。
	topicViewCounter := forum.NewRedisTopicViewCounter(sharedRedisClient).WithLogger(logger)
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
	).WithIdempotency(idempotencyStore).WithContentPostFilter(forum.ContentRegistryBridge{
		Inner: contentregistry.NewForumPostFilter(lifecycleStack.ContentRegistry),
	}).WithEditorDocumentSchema(forum.EditorRegistrySchemaBridge{
		Registry: lifecycleStack.EditorRegistry,
	}).WithSearchProviderAdmin(searchProviderAdminAdapter{registry: searchProviders}).
		WithViewRecorder(topicViewCounter)
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
		WithCacheInspector(hostCacheRuntime.Registry, hostCacheRuntime.Inspector).
		WithComponentCompositionInspector(lifecycleStack.ComponentRegistry, lifecycleStack.ComponentComposition).
		WithAssetInspector(lifecycleStack.AssetRegistry).
		WithThemeRuntimeInspector(themeRuntime).
		WithEditorRegistry(lifecycleStack.EditorRegistry).
		WithEntityRegistry(lifecycleStack.EntityRegistry).
		WithContentRegistry(lifecycleStack.ContentRegistry).
		WithMediaRegistry(lifecycleStack.MediaRegistry).
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
	// V3 Navigation Registry：与 lifecycle stack 共享同一实例；Core 已在进程启动时发布。
	pageNavigationRegistry := lifecycleStack.NavigationRegistry
	if pageNavigationRegistry == nil {
		return nil, fmt.Errorf("extension lifecycle navigation registry is unavailable")
	}
	// 生产 Navigation Runtime：精确 runtime instance admission；声明式贡献可合成，
	// Handler 渲染未接入前 fail closed（optional replace 回退 / selected replace 关闭）。
	// 使用 lifecycleRuntime（*Manager），保证 Inspect/Available 与发布制品同源。
	pageNavigationRuntime := newProductionNavigationRuntime(lifecycleRuntime)
	pageSiteChromeService := sitechrome.NewService(siteChromeStore).
		WithExtensionNavItems(providers.NewExtensionNavItemProvider(extensionService)).
		WithNavigationRegistry(pageNavigationRegistry).
		WithNavigationRuntime(pageNavigationRuntime, pageNavigationRuntime)
	// 导航检查器复用 SiteChrome 内部 trace ring，保证合成与审计同源。
	extensionsProvider.WithNavigationInspector(pageSiteChromeService.NavigationInspector())
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

	// Readiness：PG 必检；Redis 失败记 degraded 仍 ready。
	// Meilisearch 已拆为可选 search.provider 插件，不再作为 core readiness 组件。
	// F4.3：合并 system.health.checks 贡献（不调用插件 RPC）。
	readyEvaluate := func(ctx context.Context) health.ReadyReport {
		return health.EvaluateWithExtensionContributions(ctx, []health.Checker{
			health.PostgresChecker{Pool: pool},
			health.RedisChecker{Client: sharedRedisClient},
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
	executableTrustService.WithRevocationSink(extensionsruntime.NewExecutableTrustRevocationFence(
		lifecycleStack.RuntimeManager, extensionGuardPolicy,
	))
	if err := extensionGuardPolicy.Refresh(ctx); err != nil {
		stopPluginRuntimeCoordinator()
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		closePluginRuntime()
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("refresh extension guard policy failed: %w", err)
	}
	routeFailureRecorder, err := httpserver.NewRouteFailureRecorder(
		lifecycleStack.RuntimeManager,
		httpserver.NewPostgresRouteRuntimeIncidentStore(pool),
		auditWriter,
		logger,
	)
	if err != nil {
		stopPluginRuntimeCoordinator()
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		closePluginRuntime()
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("create route failure recorder: %w", err)
	}
	closeRouteFailureRecorder := func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if closeErr := routeFailureRecorder.Close(closeCtx); closeErr != nil && logger != nil {
			logger.Warn("route failure recorder stop timed out", "error", closeErr)
		}
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
		Schemas:        httpserver.CatalogRouteSchemaValidator{Catalog: lifecycleStack.RouteSchemas},
		Trace:          routeTraceRing,
		Policies:       lifecycleStack.RouteSchemas,
		Idempotency:    httpserver.NewRequiredRouteIdempotency(idempotencyStore),
		Failures:       routeFailureRecorder,
		StreamFailures: routeFailureRecorder,
	})

	return &apiCoreStack{
		adminOverviewProvider:     adminOverviewProvider,
		apiTokenService:           apiTokenService,
		attachmentsProvider:       attachmentsProvider,
		auditWriter:               auditWriter,
		authSessions:              authSessions,
		closePluginRuntime:        closePluginRuntime,
		closeRouteFailureRecorder: closeRouteFailureRecorder,
		databaseLeaseRegistry:     databaseLeaseRegistry,
		databaseProvider:          databaseProvider,
		entityMetaProvider:        entityMetaProvider,
		extensionGuardPolicy:      extensionGuardPolicy,
		extensionRuntime:          extensionRuntime,
		extensionService:          extensionService,
		extensionStore:            extensionStore,
		extensionsProvider:        extensionsProvider,
		forumProvider:             forumProvider,
		heartbeatStore:            heartbeatStore,
		hostAPIGateway:            hostAPIGateway,
		hostInstallationID:        hostInstallationID,
		identityProvider:          identityProvider,
		identityStore:             identityStore,
		jobClient:                 jobClient,
		jobsProvider:              jobsProvider,
		lifecycleStack:            lifecycleStack,
		mailProvider:              mailProvider,
		moderationProvider:        moderationProvider,
		notificationsProvider:     notificationsProvider,
		optionsProvider:           optionsProvider,
		optionsService:            optionsService,
		pagesProvider:             pagesProvider,
		pluginRuntimeCoordinator:  pluginRuntimeCoordinator,
		pluginRuntimeStopTimeout:  pluginRuntimeStopTimeout,
		pool:                      pool,
		productionSEO:             productionSEO,
		profileProvider:           profileProvider,
		// 将 Query result cache 所有权移交给 finishAPIHTTP（API.close 最终关闭）。
		queryResultCache:             queryCacheOwnership.HandOff(),
		readyEvaluate:                readyEvaluate,
		redisStorage:                 redisStorage,
		routeDispatcher:              routeDispatcher,
		seoProvider:                  seoProvider,
		sharedRedisClient:            sharedRedisClient,
		siteChromeProvider:           siteChromeProvider,
		stopPluginRuntimeCoordinator: stopPluginRuntimeCoordinator,
		webhooksProvider:             webhooksProvider,
	}, nil
}

// finishAPIHTTP：Fiber 应用、主题 watcher、嵌入 worker 与 API 句柄。
func finishAPIHTTP(ctx context.Context, cfg config.Config, logger *slog.Logger, core *apiCoreStack) (*API, error) {
	var err error
	// 承接 wire 阶段移交的 cache：本阶段失败则关闭；成功后由 API.close 关闭。
	queryCacheOwnership := newQueryResultCacheStageHandoff(core.queryResultCache, logger)
	defer queryCacheOwnership.CloseUnlessHandedOff()
	app := httpserver.NewApp(cfg, logger, httpserver.Dependencies{
		RouteProviders:  []httpserver.RouteProvider{core.identityProvider, core.notificationsProvider, core.mailProvider, core.adminOverviewProvider, core.forumProvider, core.profileProvider, core.moderationProvider, core.optionsProvider, core.siteChromeProvider, core.attachmentsProvider, core.seoProvider, core.databaseProvider, core.jobsProvider, core.extensionsProvider, core.webhooksProvider, core.entityMetaProvider, core.pagesProvider},
		RoutePlans:      core.lifecycleStack.RouteProviders,
		RouteDispatcher: core.routeDispatcher,
		RouteActors: func(c fiber.Ctx) (identity.Actor, error) {
			return loadSessionPolicyAwareRouteActor(c, core.authSessions, core.identityStore)
		},
		Options:      core.optionsService,
		Storage:      core.redisStorage,
		Ready:        core.readyEvaluate,
		BearerTokens: httpserver.TokenServiceAdapter{Service: core.apiTokenService},
		Auditor:      core.auditWriter,
	})
	var themeRuntimeWatcher *apiThemeRuntimeWatcherRuntime
	themeRuntimeStopTimeout := normalizedPluginRuntimeCoordinatorStopTimeout(cfg.WorkerShutdownTimeout)
	stopThemeRuntimeWatcher := func() {
		if themeRuntimeWatcher == nil {
			return
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), themeRuntimeStopTimeout)
		defer cancel()
		if stopErr := themeRuntimeWatcher.Stop(shutdownCtx); stopErr != nil && logger != nil {
			logger.Warn("theme runtime watcher stop failed", "error", stopErr)
		}
	}
	themeRuntimeHandedOff := false
	defer func() {
		if !themeRuntimeHandedOff {
			stopThemeRuntimeWatcher()
		}
	}()
	if !cfg.SafeMode {
		themeRuntimeWatcher, err = startAPIThemeRuntimeWatcher(
			ctx, core.extensionStore, core.extensionService, logger, themeRuntimeStopTimeout,
		)
		if err != nil {
			core.closeRouteFailureRecorder()
			core.stopPluginRuntimeCoordinator()
			if stopErr := supportjobs.Stop(ctx, core.jobClient); stopErr != nil {
				logger.Warn("job dispatcher stop failed", "error", stopErr)
			}
			core.closePluginRuntime()
			_ = core.hostAPIGateway.Close()
			core.sharedRedisClient.Close()
			if closeErr := core.redisStorage.Close(); closeErr != nil {
				logger.Warn("redis session storage close failed", "error", closeErr)
			}
			core.pool.Close()
			return nil, fmt.Errorf("start theme runtime watcher failed: %w", err)
		}
	}

	// Worker 心跳：嵌入 worker 时由 API 进程发布；独立 worker 在 NewWorker 内发布。
	// 未嵌入时 overview 仍可读独立 worker 写入的同一 Redis key。
	heartbeatCtx, heartbeatCancel := context.WithCancel(context.Background())

	var embeddedWorker *Worker
	if shouldEmbedWorkerInAPI(cfg) {
		workerQueryInvalidation := newEmbeddedWorkerQueryInvalidationRuntime(cfg, core.hostInstallationID, logger)
		// Embed 时复用 API 已 Reconcile 的 core.extensionRuntime，避免每个后端插件双起子进程。
		// 插件 runtime 复用，但 Query invalidator 独占 Redis client，避免 worker
		// 的 terminal latch 污染 API execution cache。
		embeddedWorker, err = newWorkerWithPool(cfg, core.pool, logger, workerRuntimeDeps{
			ExtensionRuntime:  core.extensionRuntime,
			PluginSchedules:   core.lifecycleStack.Schedules,
			QueryInvalidation: workerQueryInvalidation,
			// 与 API 共用 Redis：flush_view_counts 读 API 写入的 view delta。
			HostCacheRedis: core.sharedRedisClient,
			OwnsRuntime:    false,
		})
		if err != nil {
			core.closeRouteFailureRecorder()
			stopThemeRuntimeWatcher()
			heartbeatCancel()
			core.stopPluginRuntimeCoordinator()
			if stopErr := supportjobs.Stop(ctx, core.jobClient); stopErr != nil {
				logger.Warn("job dispatcher stop failed", "error", stopErr)
			}
			core.closePluginRuntime()
			core.sharedRedisClient.Close()
			if closeErr := core.redisStorage.Close(); closeErr != nil {
				logger.Warn("redis session storage close failed", "error", closeErr)
			}
			core.pool.Close()
			return nil, fmt.Errorf("embedded worker setup failed: %w", err)
		}
		if err := embeddedWorker.Start(ctx); err != nil {
			core.closeRouteFailureRecorder()
			stopThemeRuntimeWatcher()
			heartbeatCancel()
			embeddedWorker.Close()
			core.stopPluginRuntimeCoordinator()
			if stopErr := supportjobs.Stop(ctx, core.jobClient); stopErr != nil {
				logger.Warn("job dispatcher stop failed", "error", stopErr)
			}
			core.closePluginRuntime()
			core.sharedRedisClient.Close()
			if closeErr := core.redisStorage.Close(); closeErr != nil {
				logger.Warn("redis session storage close failed", "error", closeErr)
			}
			core.pool.Close()
			return nil, fmt.Errorf("embedded worker start failed: %w", err)
		}
		go (&health.Publisher{Store: core.heartbeatStore}).Run(heartbeatCtx)
		logger.InfoContext(ctx, "embedded api worker started")
	}
	forumReadPolicyCtx, forumReadPolicyCancel := context.WithCancel(context.Background())
	go core.optionsService.RunForumReadPolicyRefresh(forumReadPolicyCtx, options.RecommendedForumReadPolicyRefreshInterval)
	extensionGuardPolicyCtx, extensionGuardPolicyCancel := context.WithCancel(context.Background())
	go core.extensionGuardPolicy.RunRefresh(extensionGuardPolicyCtx, extensions.RecommendedGuardPolicyRefreshInterval)
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
				if _, err := core.databaseLeaseRegistry.ReapExpiredRuntimeLeases(
					databaseLeaseReaperCtx, extensionsruntime.DefaultExtensionDatabaseRuntimeLeaseReapLimit,
				); err != nil && databaseLeaseReaperCtx.Err() == nil {
					logger.Warn("extension database runtime lease reaper degraded", "error", err)
				}
			}
		}
	}()
	pluginRuntimeFailures, pluginRuntimeMonitorDone := superviseAPIPluginRuntimeCoordinator(
		core.pluginRuntimeCoordinator,
		core.closePluginRuntime,
	)
	apiRuntimeFailures, stopAPIRuntimeFailureMerge := mergeAPIRuntimeFailureSources(
		pluginRuntimeFailures,
		themeRuntimeWatcher.Failures(),
	)

	// development：延迟清理 air 热重载后被 init 收养的 backend plugin 孤儿。
	// 必须在旧 sforum-api 被 stopBin 杀掉之后（kill_delay≈2s）再扫，pre_cmd 单独不够。
	stopOrphanPluginReaper := startDevelopmentOrphanPluginReaper(cfg, logger)

	api := &API{
		App:      app,
		Addr:     apiAddress(cfg),
		SEO:      core.productionSEO.Execution,
		failures: apiRuntimeFailures,
		close: func() {
			stopOrphanPluginReaper()
			core.closeRouteFailureRecorder()
			stopAPIRuntimeFailureMerge()
			core.stopPluginRuntimeCoordinator()
			if pluginRuntimeMonitorDone != nil {
				waitCtx, cancel := context.WithTimeout(context.Background(), core.pluginRuntimeStopTimeout)
				select {
				case <-pluginRuntimeMonitorDone:
				case <-waitCtx.Done():
					if logger != nil {
						logger.Warn("plugin runtime coordinator monitor stop timed out", "error", waitCtx.Err())
					}
				}
				cancel()
			}
			stopThemeRuntimeWatcher()
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
			if err := supportjobs.Stop(context.Background(), core.jobClient); err != nil {
				logger.Warn("job dispatcher stop failed", "error", err)
			}
			core.closePluginRuntime()
			_ = core.hostAPIGateway.Close()
			core.queryResultCache.Close(logger)
			if err := core.redisStorage.Close(); err != nil {
				logger.Warn("redis session storage close failed", "error", err)
			}
			// core.sharedRedisClient 被 humanverify 与 forum 缓存共用，关闭一次即可。
			if err := core.sharedRedisClient.Close(); err != nil {
				logger.Warn("shared redis client close failed", "error", err)
			}
			core.pool.Close()
		},
	}
	// 成功装配：cache 所有权交给 API.close（close 闭包已捕获 core.queryResultCache）。
	_ = queryCacheOwnership.HandOff()
	select {
	case runtimeErr, ok := <-apiRuntimeFailures:
		if ok && runtimeErr != nil {
			api.Close()
			return nil, fmt.Errorf("API runtime failed during startup: %w", runtimeErr)
		}
	default:
	}
	themeRuntimeHandedOff = true
	return api, nil
}

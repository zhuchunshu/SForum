package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/session"

	httpserver "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensionjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Extensions"
	adminoverview "github.com/zhuchunshu/sforum/apps/api/app/Models/AdminOverview"
	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	database "github.com/zhuchunshu/sforum/apps/api/app/Models/Database"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	moderation "github.com/zhuchunshu/sforum/apps/api/app/Models/Moderation"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	profile "github.com/zhuchunshu/sforum/apps/api/app/Models/Profile"
	"github.com/zhuchunshu/sforum/apps/api/app/Providers"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	avatar "github.com/zhuchunshu/sforum/apps/api/app/Support/Avatar"
	cache "github.com/zhuchunshu/sforum/apps/api/app/Support/Cache"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Crypto"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	mail "github.com/zhuchunshu/sforum/apps/api/app/Support/Mail"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Postgres"
	redisplatform "github.com/zhuchunshu/sforum/apps/api/app/Support/Redis"
	search "github.com/zhuchunshu/sforum/apps/api/app/Support/Search"
	themeruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeRuntime"
	webreleasecoordinator "github.com/zhuchunshu/sforum/apps/api/app/Support/WebReleaseCoordinator"
	webreleaseruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/WebReleaseRuntime"
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
	Reconcile(ctx context.Context, items []extensions.Extension)
	Close(ctx context.Context)
}

var newExtensionRuntimeManager = func(store extensions.Store) extensionRuntime {
	return extensionsruntime.NewManager(extensionsruntime.ManagerConfig{
		Starter:       extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{Settings: store}),
		DeliveryStore: store,
	})
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
	optionsService := options.NewServiceWithDefaults(optionStore, optionsDefaultsFromConfig(cfg)).WithCipher(optionCipher)
	if err := optionsService.EnsureDefaults(ctx); err != nil {
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("ensure runtime options defaults failed: %w", err)
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
	databaseStore := database.NewPostgresStore(pool)
	extensionStore := extensions.NewPostgresStore(pool)
	frontendTrustStore := extensions.NewPostgresFrontendTrustStore(pool)
	webReleaseStore := extensions.NewPostgresWebReleaseStore(pool)
	extensionRuntime := newExtensionRuntimeManager(extensionStore)
	jobClient, err := supportjobs.NewInsertOnlyClient(pool, supportjobs.FromAppConfig(cfg))
	if err != nil {
		extensionRuntime.Close(ctx)
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("job dispatcher setup failed: %w", err)
	}
	jobDispatcher := supportjobs.NewDispatcher(jobClient)
	themeDispatcher := extensionjobs.ActivationDispatcherAdapter{Dispatcher: jobDispatcher}
	hostComposition, err := webreleaseruntime.CompositionHost(cfg.WebReleaseWebRoot)
	if err != nil {
		extensionRuntime.Close(ctx)
		sharedRedisClient.Close()
		_ = redisStorage.Close()
		pool.Close()
		return nil, fmt.Errorf("resolve web release host identity: %w", err)
	}
	webReleasePlanner := extensions.NewWebReleasePlanner(extensionStore, frontendTrustStore, hostComposition)
	webReleaseService := extensions.NewWebReleaseService(
		webReleasePlanner, pool, webReleaseStore,
		extensionjobs.WebReleaseBuildDispatcherAdapter{Dispatcher: jobDispatcher},
	)
	frontendService := extensions.NewFrontendService(extensionStore, frontendTrustStore, webReleaseService, webReleaseStore, hostComposition)
	webReleaseAdminService := extensions.NewWebReleaseAdminService(webReleaseStore, webReleaseService)
	// API 进程同步路径（恢复默认主题）也需要写 current.json。
	// 这里构造一个仅用于 WriteCurrent 的 builder，不参与主题构建（构建仍由 worker 完成）。
	themeCurrentWriter := themeruntime.NewBuilder(themeruntime.Config{ReleaseRoot: cfg.ThemeReleaseRoot})
	extensionService := extensions.NewServiceWithThemeActivationWithOptions(
		extensionStore, cfg.ExtensionRoot, cfg.BuiltinExtensionRoot,
		extensionRuntime, nil, themeDispatcher,
		extensions.WithThemeCurrentWriter(themeCurrentWriter),
		extensions.WithWebReleaseLifecycle(frontendService, webReleaseService),
	)
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
	legacyMailValues, err := optionsService.InternalValues(ctx)
	if err != nil {
		return nil, fmt.Errorf("load legacy mail options: %w", err)
	}
	if err := notifications.NewPostgresStore(pool).AdoptLegacyMail(ctx, legacyMailValues); err != nil {
		return nil, fmt.Errorf("adopt legacy mail settings: %w", err)
	}
	// 首次启用统一 runtime 时只排队 canonical release；构建失败不能阻断旧站点启动。
	if err := ensureInitialWebRelease(ctx, webReleaseStore, webReleaseService); err != nil {
		logger.Warn("queue initial web release failed", "error", err)
	}
	if items, err := extensionStore.List(ctx); err == nil {
		extensionRuntime.Reconcile(ctx, items)
	} else {
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
	webReleaseCoordinator := webreleasecoordinator.New(
		webreleasecoordinator.NewPostgresStore(pool, webReleaseStore, extensionStore),
		webreleasecoordinator.NewRuntimeAdapter(extensionStore, extensionRuntime),
		webreleaseruntime.NewPointerStore(cfg.WebReleaseRoot, webReleaseStore),
		postgres.NewAdvisoryLocker(pool),
	)
	if err := webReleaseCoordinator.Start(ctx); err != nil {
		extensionRuntime.Close(ctx)
		sharedRedisClient.Close()
		_ = redisStorage.Close()
		pool.Close()
		return nil, fmt.Errorf("start web release coordinator: %w", err)
	}
	// 邮件服务与密码重置：mail resolver 复用 options.Service（实现 mail.Resolver）。
	mailService := mail.NewService(optionsService, logger)
	siteName, _ := optionsService.SiteName(ctx)
	siteURL, _ := optionsService.WebOption(ctx, "site.url")
	passwordResetService := identity.NewPasswordResetServiceWithPasswordPolicy(identityStore, mailService, identity.PasswordResetConfig{
		SiteName: siteName,
		SiteURL:  siteURL,
	}, optionsService)
	identityProvider := providers.NewIdentityProviderWithPasswordReset(identityStore, authSessions, humanVerifier, extensionRuntime, passwordResetService, mailService, optionsService)
	adminOverviewProvider := providers.NewAdminOverviewProvider(adminOverviewStore, adminoverview.NewRuntimeCollector(time.Now().UTC(), pool), identityStore, authSessions)
	// 搜索：API 进程持有只入队的 indexer（EnqueueIndex/EnqueueDelete）和查询用的 search service。
	// Meilisearch client 不可达时，索引调度静默失败、搜索端点返回 503，主流程不受影响。
	meiliClient := search.NewClientWithTimeout(cfg.MeiliHost, cfg.MeiliMasterKey, cfg.MeiliTimeout)
	searchIndexer := search.NewIndexer(meiliClient, nil, jobDispatcher)
	forumSettingsResolver := providers.NewForumSettingsResolver(optionsService)
	searchService := search.NewService(meiliClient, forumSettingsResolver)
	// 搜索索引重建：forumStore 提供 ListAllTopicIDs（TopicIDSource），
	// reindexStore 记录运行状态，dispatcher 批量入队 IndexTopicArgs。
	reindexManager := search.NewReindexManager(forumStore, search.NewPostgresReindexStore(pool), jobDispatcher)
	forumProvider := providers.NewForumProviderWithSearchTopicActionsAndPublicationPolicy(
		forumCachedStore,
		optionsService,
		identityStore,
		authSessions,
		extensionRuntime,
		searchIndexer,
		searchServiceAdapter{inner: searchService},
		reindexServiceAdapter{inner: reindexManager},
		providers.NewExtensionTopicActionProvider(extensionService),
		providers.NewModerationPublicationPolicy(moderationStore, optionsService),
	)
	avatarAttachmentService := attachments.NewServiceWithEvents(attachmentStore, optionsService, extensionRuntime)
	profileProvider := providers.NewProfileProviderWithAvatar(profileStore, identityStore, authSessions, avatarAttachmentService, optionsService)
	moderationProvider := providers.NewModerationWorkbenchProviderWithIndexer(moderationStore, forumStore, identityStore, authSessions, searchIndexer)
	optionsProvider := providers.NewOptionsProviderWithService(optionsService, identityStore, authSessions)
	attachmentsProvider := providers.NewAttachmentsProviderWithEvents(attachmentStore, optionsService, identityStore, authSessions, extensionRuntime)
	seoProvider := providers.NewSEOProvider(pool, optionsService)
	databaseProvider := providers.NewDatabaseProvider(databaseStore, identityStore, authSessions)
	jobsProvider := providers.NewJobsProvider(pool, jobClient, identityStore, authSessions)
	extensionsProvider := providers.NewExtensionsProviderWithService(extensionService, identityStore, authSessions, extensionRuntime, frontendService, webReleaseAdminService)

	app := httpserver.NewApp(cfg, logger, httpserver.Dependencies{
		RouteProviders: []httpserver.RouteProvider{identityProvider, adminOverviewProvider, forumProvider, profileProvider, moderationProvider, optionsProvider, attachmentsProvider, seoProvider, databaseProvider, jobsProvider, extensionsProvider},
		Options:        optionsService,
		Storage:        redisStorage,
	})

	var embeddedWorker *Worker
	if shouldEmbedWorkerInAPI(cfg) {
		embeddedWorker, err = newWorkerWithPool(cfg, pool, logger)
		if err != nil {
			_ = webReleaseCoordinator.Stop(context.Background())
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
			_ = webReleaseCoordinator.Stop(context.Background())
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
		logger.InfoContext(ctx, "embedded api worker started")
	}

	return &API{
		App:  app,
		Addr: apiAddress(cfg),
		close: func() {
			coordinatorCtx, coordinatorCancel := context.WithTimeout(context.Background(), cfg.WorkerShutdownTimeout)
			if err := webReleaseCoordinator.Stop(coordinatorCtx); err != nil {
				logger.Warn("web release coordinator stop failed", "error", err)
			}
			coordinatorCancel()
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

// shouldUseSecureCookie 决定 session/CSRF cookie 是否带 Secure 标志。
// 生产环境（AppEnv=="production"）强制启用；此外当 APP_URL 是 https 时也启用，
// 避免 staging 等"非 production 但走 HTTPS"的部署因环境字符串漏配而把 cookie 走 HTTP。
func shouldUseSecureCookie(cfg config.Config) bool {
	if strings.EqualFold(cfg.AppEnv, "production") {
		return true
	}
	if parsed, err := url.Parse(strings.TrimSpace(cfg.AppURL)); err == nil {
		return strings.EqualFold(parsed.Scheme, "https")
	}
	return false
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

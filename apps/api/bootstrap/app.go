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
	extensionjobs "github.com/zhuchunshu/sforum/apps/api/app/Jobs/Extensions"
	adminoverview "github.com/zhuchunshu/sforum/apps/api/app/Models/AdminOverview"
	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	database "github.com/zhuchunshu/sforum/apps/api/app/Models/Database"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	moderation "github.com/zhuchunshu/sforum/apps/api/app/Models/Moderation"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	profile "github.com/zhuchunshu/sforum/apps/api/app/Models/Profile"
	"github.com/zhuchunshu/sforum/apps/api/app/Providers"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	cache "github.com/zhuchunshu/sforum/apps/api/app/Support/Cache"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	mail "github.com/zhuchunshu/sforum/apps/api/app/Support/Mail"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Postgres"
	redisplatform "github.com/zhuchunshu/sforum/apps/api/app/Support/Redis"
	search "github.com/zhuchunshu/sforum/apps/api/app/Support/Search"
	themeruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeRuntime"
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
		Starter:       extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{}),
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

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns)
	if err != nil {
		return nil, fmt.Errorf("postgres setup failed: %w", err)
	}

	redisStorage, err := redisplatform.NewStorage(cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("redis session storage setup failed: %w", err)
	}
	humanVerifyStore := humanverify.NewRedisStore(humanverify.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword))
	optionStore := options.NewPostgresStore(pool)
	optionsService := options.NewServiceWithDefaults(optionStore, optionsDefaultsFromConfig(cfg))
	if err := optionsService.EnsureDefaults(ctx); err != nil {
		if closeErr := humanVerifyStore.Close(); closeErr != nil {
			logger.Warn("human verification redis close failed", "error", closeErr)
		}
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("ensure runtime options defaults failed: %w", err)
	}
	humanVerifier := humanverify.NewRuntimeService(optionsService, humanVerifyStore, humanVerifyConfigFromConfig(cfg))

	sessionStore := session.NewStore(session.Config{
		Storage:         redisStorage,
		KeyGenerator:    secureSessionID,
		Extractor:       extractors.FromCookie("sforum_session"),
		IdleTimeout:     cfg.SessionIdleTimeout,
		AbsoluteTimeout: cfg.SessionAbsoluteTimeout,
		CookieHTTPOnly:  true,
		CookieSameSite:  fiber.CookieSameSiteLaxMode,
		CookiePath:      "/",
		CookieSecure:    strings.EqualFold(cfg.AppEnv, "production"),
	})
	authSessions := authsession.NewManager(sessionStore, authsession.Config{
		RenewalInterval: cfg.SessionRenewalInterval,
		HashSecret:      cfg.SessionHashSecret,
	})
	identityStore := identity.NewPostgresStore(pool)
	adminOverviewStore := adminoverview.NewPostgresStore(pool)
	forumStore := forum.NewPostgresStore(pool)
	// 业务读缓存：复用 session 同款 Redis client，避免多套连接。
	// 失败不阻断启动——缓存为可重建的派生数据，降级为直连 PG。
	cacheClient := humanverify.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword)
	forumCachedStore := forum.NewCachedStore(forumStore, cache.NewRedisCache(cacheClient))
	profileStore := profile.NewPostgresStore(pool)
	moderationStore := moderation.NewPostgresStore(pool)
	attachmentStore := attachments.NewPostgresStore(pool)
	databaseStore := database.NewPostgresStore(pool)
	extensionStore := extensions.NewPostgresStore(pool)
	extensionRuntime := newExtensionRuntimeManager(extensionStore)
	jobClient, err := supportjobs.NewInsertOnlyClient(pool, supportjobs.FromAppConfig(cfg))
	if err != nil {
		extensionRuntime.Close(ctx)
		if closeErr := humanVerifyStore.Close(); closeErr != nil {
			logger.Warn("human verification redis close failed", "error", closeErr)
		}
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("job dispatcher setup failed: %w", err)
	}
	jobDispatcher := supportjobs.NewDispatcher(jobClient)
	themeDispatcher := extensionjobs.ActivationDispatcherAdapter{Dispatcher: jobDispatcher}
	// API 进程同步路径（恢复默认主题）也需要写 current.json。
	// 这里构造一个仅用于 WriteCurrent 的 builder，不参与主题构建（构建仍由 worker 完成）。
	themeCurrentWriter := themeruntime.NewBuilder(themeruntime.Config{ReleaseRoot: cfg.ThemeReleaseRoot})
	extensionService := extensions.NewServiceWithThemeActivationWithOptions(
		extensionStore, cfg.ExtensionRoot, cfg.BuiltinExtensionRoot,
		extensionRuntime, nil, themeDispatcher,
		extensions.WithThemeCurrentWriter(themeCurrentWriter),
	)
	if _, err := extensionService.SyncBuiltins(ctx); err != nil {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		extensionRuntime.Close(ctx)
		if closeErr := humanVerifyStore.Close(); closeErr != nil {
			logger.Warn("human verification redis close failed", "error", closeErr)
		}
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("sync builtin extensions failed: %w", err)
	}
	if items, err := extensionStore.List(ctx); err == nil {
		extensionRuntime.Reconcile(ctx, items)
	} else {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		extensionRuntime.Close(ctx)
		if closeErr := humanVerifyStore.Close(); closeErr != nil {
			logger.Warn("human verification redis close failed", "error", closeErr)
		}
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("list extensions for runtime reconciliation failed: %w", err)
	}
	// 邮件服务与密码重置：mail resolver 复用 options.Service（实现 mail.Resolver）。
	mailService := mail.NewService(optionsService, logger)
	siteName, _ := optionsService.SiteName(ctx)
	siteURL, _ := optionsService.WebOption(ctx, "site.url")
	passwordResetService := identity.NewPasswordResetService(identityStore, mailService, identity.PasswordResetConfig{
		SiteName: siteName,
		SiteURL:  siteURL,
	})
	identityProvider := providers.NewIdentityProviderWithPasswordReset(identityStore, authSessions, humanVerifier, extensionRuntime, passwordResetService, mailService, optionsService)
	adminOverviewProvider := providers.NewAdminOverviewProvider(adminOverviewStore, adminoverview.NewRuntimeCollector(time.Now().UTC(), pool), identityStore, authSessions)
	// 搜索：API 进程持有只入队的 indexer（EnqueueIndex/EnqueueDelete）和查询用的 search service。
	// Meilisearch client 不可达时，索引调度静默失败、搜索端点返回 503，主流程不受影响。
	meiliClient := search.NewClient(cfg.MeiliHost, cfg.MeiliMasterKey)
	searchIndexer := search.NewIndexer(meiliClient, nil, jobDispatcher)
	searchService := search.NewService(meiliClient)
	// 搜索索引重建：forumStore 提供 ListAllTopicIDs（TopicIDSource），
	// reindexStore 记录运行状态，dispatcher 批量入队 IndexTopicArgs。
	reindexManager := search.NewReindexManager(forumStore, search.NewPostgresReindexStore(pool), jobDispatcher)
	forumProvider := providers.NewForumProviderWithSearch(forumCachedStore, optionsService, identityStore, authSessions, extensionRuntime, searchIndexer, searchServiceAdapter{inner: searchService}, reindexServiceAdapter{inner: reindexManager})
	profileProvider := providers.NewProfileProvider(profileStore, identityStore, authSessions)
	moderationProvider := providers.NewModerationProvider(moderationStore, forumStore, identityStore, authSessions)
	optionsProvider := providers.NewOptionsProviderWithService(optionsService, identityStore, authSessions)
	attachmentsProvider := providers.NewAttachmentsProviderWithEvents(attachmentStore, optionsService, identityStore, authSessions, extensionRuntime)
	databaseProvider := providers.NewDatabaseProvider(databaseStore, identityStore, authSessions)
	extensionsProvider := providers.NewExtensionsProviderWithRuntimeAndThemeActivation(extensionStore, identityStore, authSessions, cfg.ExtensionRoot, cfg.BuiltinExtensionRoot, extensionRuntime, themeDispatcher, extensions.WithThemeCurrentWriter(themeCurrentWriter))

	app := httpserver.NewApp(cfg, logger, httpserver.Dependencies{
		RouteProviders: []httpserver.RouteProvider{identityProvider, adminOverviewProvider, forumProvider, profileProvider, moderationProvider, optionsProvider, attachmentsProvider, databaseProvider, extensionsProvider},
		Options:        optionsService,
	})

	var embeddedWorker *Worker
	if shouldEmbedWorkerInAPI(cfg) {
		embeddedWorker, err = newWorkerWithPool(cfg, pool)
		if err != nil {
			if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
				logger.Warn("job dispatcher stop failed", "error", stopErr)
			}
			extensionRuntime.Close(ctx)
			if closeErr := humanVerifyStore.Close(); closeErr != nil {
				logger.Warn("human verification redis close failed", "error", closeErr)
			}
			if closeErr := redisStorage.Close(); closeErr != nil {
				logger.Warn("redis session storage close failed", "error", closeErr)
			}
			pool.Close()
			return nil, fmt.Errorf("embedded worker setup failed: %w", err)
		}
		if err := embeddedWorker.Start(ctx); err != nil {
			embeddedWorker.Close()
			if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
				logger.Warn("job dispatcher stop failed", "error", stopErr)
			}
			extensionRuntime.Close(ctx)
			if closeErr := humanVerifyStore.Close(); closeErr != nil {
				logger.Warn("human verification redis close failed", "error", closeErr)
			}
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
			if err := humanVerifyStore.Close(); err != nil {
				logger.Warn("human verification redis close failed", "error", err)
			}
			if err := cacheClient.Close(); err != nil {
				logger.Warn("forum cache redis close failed", "error", err)
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

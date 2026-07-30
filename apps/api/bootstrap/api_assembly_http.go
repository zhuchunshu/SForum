package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	httpserver "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	health "github.com/zhuchunshu/sforum/apps/api/app/Support/Health"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

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
		Recovery:     core.pluginRuntimeRecovery,
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
	if !cfg.SafeMode && !core.pluginRuntimeRecovery.Active() {
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
			core.closeNotificationStore()
			core.pool.Close()
			return nil, fmt.Errorf("start theme runtime watcher failed: %w", err)
		}
	}

	// Worker 心跳：嵌入 worker 时由 API 进程发布；独立 worker 在 NewWorker 内发布。
	// 未嵌入时 overview 仍可读独立 worker 写入的同一 Redis key。
	heartbeatCtx, heartbeatCancel := context.WithCancel(context.Background())

	var embeddedWorker *Worker
	if shouldEmbedWorkerInAPI(cfg) && !core.pluginRuntimeRecovery.Active() {
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
			core.closeNotificationStore()
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
			core.closeNotificationStore()
			core.pool.Close()
			return nil, fmt.Errorf("embedded worker start failed: %w", err)
		}
		go (&health.Publisher{Store: core.heartbeatStore}).Run(heartbeatCtx)
		logger.InfoContext(ctx, "embedded api worker started")
	}
	forumReadPolicyCtx, forumReadPolicyCancel := context.WithCancel(context.Background())
	go core.optionsService.RunForumReadPolicyRefresh(forumReadPolicyCtx, options.RecommendedForumReadPolicyRefreshInterval)
	extensionGuardPolicyCtx, extensionGuardPolicyCancel := context.WithCancel(context.Background())
	if !core.pluginRuntimeRecovery.Active() {
		go core.extensionGuardPolicy.RunRefresh(extensionGuardPolicyCtx, extensions.RecommendedGuardPolicyRefreshInterval)
	}
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
			core.closeNotificationStore()
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

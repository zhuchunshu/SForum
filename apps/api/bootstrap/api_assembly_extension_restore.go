package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	health "github.com/zhuchunshu/sforum/apps/api/app/Support/Health"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	pluginbootstrap "github.com/zhuchunshu/sforum/apps/api/app/Support/PluginBootstrap"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

type apiExtensionRestoreInput struct {
	infrastructure         *apiInfrastructure
	extensionService       *extensions.Service
	frontendService        *extensions.FrontendService
	executableTrustService *extensions.ExecutableTrustService
	databaseCatalogBinder  postgresProtocolV2DatabaseCatalogBinder
	extensionRuntime       extensionRuntime
	lifecycleRuntime       *extensionsruntime.Manager
	lifecycleStack         *productionLifecycleStack
}

type apiExtensionRuntimeState struct {
	pluginRuntimeCoordinator     *pluginRuntimeCoordinatorRuntime
	pluginRuntimeRecovery        *health.RecoveryRequirement
	pluginRuntimeStopTimeout     time.Duration
	stopPluginRuntimeCoordinator func()
	closePluginRuntime           func()
}

func restoreAPIExtensionPlatform(ctx context.Context, cfg config.Config, logger *slog.Logger, input apiExtensionRestoreInput) (*apiExtensionRuntimeState, error) {
	pool := input.infrastructure.pool
	redisStorage := input.infrastructure.redisStorage
	sharedRedisClient := input.infrastructure.sharedRedisClient
	optionsService := input.infrastructure.optionsService
	extensionStore := input.infrastructure.extensionStore
	jobClient := input.infrastructure.jobClient
	extensionService := input.extensionService
	frontendService := input.frontendService
	executableTrustService := input.executableTrustService
	databaseCatalogBinder := input.databaseCatalogBinder
	extensionRuntime := input.extensionRuntime
	lifecycleRuntime := input.lifecycleRuntime
	lifecycleStack := input.lifecycleStack
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
	if err := syncExternalExtensionSources(ctx, logger, extensionService); err != nil {
		if stopErr := supportjobs.Stop(ctx, jobClient); stopErr != nil {
			logger.Warn("job dispatcher stop failed", "error", stopErr)
		}
		extensionRuntime.Close(ctx)
		sharedRedisClient.Close()
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, fmt.Errorf("sync external extension sources failed: %w", err)
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
	var pluginRuntimeRecovery *health.RecoveryRequirement
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
	if err != nil && !cfg.SafeMode {
		convergenceErr := err
		stopPluginRuntimeCoordinator()
		pluginRuntimeCoordinator = nil
		pluginRuntimeRecovery = newPluginRuntimeRecoveryRequirement(ctx, extensionStore, convergenceErr)
		logger.Error(
			"plugin runtime convergence failed; entering Host recovery-only mode",
			"error", convergenceErr,
			"publication_revision", pluginRuntimeRecovery.PublicationRevision,
			"artifacts", len(pluginRuntimeRecovery.Artifacts),
		)
		if restoreErr := extensionService.RestoreSafeModeThemeRegistry(ctx); restoreErr != nil {
			logger.Warn("restore recovery-only default theme registry failed", "error", restoreErr)
		}
		reconciledExtensions, err = reconcileAPIExtensionRuntime(ctx, true, extensionStore, extensionRuntime)
		if err != nil {
			err = fmt.Errorf("enter recovery-only mode after %v: %w", convergenceErr, err)
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
	effectiveSafeMode := cfg.SafeMode || pluginRuntimeRecovery.Active()
	if err := lifecycleStack.Registries.RestoreRoutePublications(ctx, reconciledExtensions, effectiveSafeMode); err != nil {
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
	return &apiExtensionRuntimeState{
		pluginRuntimeCoordinator:     pluginRuntimeCoordinator,
		pluginRuntimeRecovery:        pluginRuntimeRecovery,
		pluginRuntimeStopTimeout:     pluginRuntimeStopTimeout,
		stopPluginRuntimeCoordinator: stopPluginRuntimeCoordinator,
		closePluginRuntime:           closePluginRuntime,
	}, nil
}

func newPluginRuntimeRecoveryRequirement(
	ctx context.Context,
	store *extensions.PostgresStore,
	cause error,
) *health.RecoveryRequirement {
	requirement := &health.RecoveryRequirement{
		Code:      health.PluginRuntimeRecoveryCode,
		Component: "plugin_runtime",
		Message:   pluginRuntimeRecoveryMessage(cause),
	}
	if store == nil {
		return requirement
	}
	publication, err := store.LatestPluginRuntimePublication(ctx)
	if err != nil {
		return requirement
	}
	requirement.PublicationRevision = publication.Revision
	requirement.Artifacts = make([]health.RecoveryArtifact, 0, len(publication.Members))
	for _, member := range publication.Members {
		requirement.Artifacts = append(requirement.Artifacts, health.RecoveryArtifact{
			ExtensionID: member.ExtensionID,
			Version:     member.ExtensionVersion,
			Digest:      member.PackageDigest,
		})
	}
	return requirement
}

func pluginRuntimeRecoveryMessage(err error) string {
	switch {
	case errors.Is(err, pluginbootstrap.ErrBootstrapABIIncompatible):
		return "plugin process bootstrap ABI is incompatible"
	case errors.Is(err, pluginbootstrap.ErrExecutableArchitecture):
		return "plugin executable architecture is incompatible"
	case errors.Is(err, pluginbootstrap.ErrExecutableDependency):
		return "plugin executable dependency is unavailable"
	case errors.Is(err, pluginbootstrap.ErrExecutablePermission):
		return "plugin executable permission was denied"
	case errors.Is(err, pluginbootstrap.ErrProcessStart):
		return "plugin process failed to start"
	default:
		return "plugin runtime convergence failed; inspect API logs for the exact cause"
	}
}

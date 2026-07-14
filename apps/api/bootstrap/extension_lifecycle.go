package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	pages "github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

var errProductionLifecycleDependency = errors.New("bootstrap: production extension lifecycle dependency unavailable")

type productionLifecycleStackConfig struct {
	Pool            *pgxpool.Pool
	Store           *extensions.PostgresStore
	Features        extensions.FeatureFlagSource
	Trust           *extensions.ExecutableTrustService
	Runtime         *extensionsruntime.Manager
	Pages           *pages.Registry
	Services        *hostapi.ServiceRegistry
	River           hostapi.PluginJobLifecycleRiverClient
	ExtensionRoot   string
	MigrationEngine extensionsruntime.LifecycleMigrationEngine
	Database        extensionsruntime.ExtensionDatabaseDisposition
}

// productionLifecycleStack 保留组装后的具体实例，避免 lifecycle 的不同边界
// 意外各自创建 Manager、journal 或 schedule admission registry。
type productionLifecycleStack struct {
	Repository         *extensions.PostgresLifecycleRepository
	RuntimeManager     *extensionsruntime.Manager
	Runtime            *extensionsruntime.ExactLifecycleCoordinatorRuntimeAdapter
	Preflight          *extensionsruntime.ProductionLifecycleBoundaryPreflight
	StaticPreflight    extensions.LifecycleStaticPreflight
	MigrationEngine    extensionsruntime.LifecycleMigrationEngine
	Migrations         *extensionsruntime.ProductionLifecycleBoundaryMigrations
	Schedules          *supportjobs.PluginScheduleAdmissionRegistry
	JobStore           *hostapi.PostgresPluginJobLifecycleStore
	JobCoordinator     *hostapi.PluginJobLifecycleCoordinator
	Jobs               *extensionsruntime.PostgresLifecycleBoundaryJobs
	RouteFoundation    *routes.Registry
	RegistryRepository *extensionsruntime.PostgresLifecycleRegistryPublicationRepository
	Registries         *extensionsruntime.PostgresLifecycleBoundaryRegistries
	State              *extensionsruntime.PostgresLifecycleBoundaryState
	PublicationJournal *extensionsruntime.PostgresLifecycleBoundaryPublicationJournal
	Cleanup            *extensionsruntime.PostgresLifecycleBoundaryCleanup
	Database           extensionsruntime.ExtensionDatabaseDisposition
	CleanupPurger      extensionsruntime.LifecycleBoundaryCleanupPurger
	CleanupFinalizer   *extensionsruntime.PostgresLifecycleBoundaryCleanupFinalizer
	Boundary           *extensionsruntime.ComposedLifecycleHostBoundary
	Host               *extensionsruntime.ExactLifecycleCoordinatorHost
	Coordinator        *extensions.LifecycleCoordinator
}

func requireProductionExtensionRuntime(runtime extensionRuntime) (*extensionsruntime.Manager, error) {
	manager, ok := runtime.(*extensionsruntime.Manager)
	if !ok || manager == nil {
		return nil, fmt.Errorf("%w: exact runtime manager", errProductionLifecycleDependency)
	}
	return manager, nil
}

func newProductionLifecycleStack(config productionLifecycleStackConfig) (*productionLifecycleStack, error) {
	if config.Pool == nil || config.Store == nil || config.Features == nil || config.Trust == nil ||
		config.Runtime == nil || config.Pages == nil || config.Services == nil || config.River == nil ||
		strings.TrimSpace(config.ExtensionRoot) == "" || config.MigrationEngine == nil || config.Database == nil {
		return nil, errProductionLifecycleDependency
	}

	runtime, err := config.Runtime.NewExactLifecycleCoordinatorRuntimeAdapter()
	if err != nil {
		return nil, fmt.Errorf("%w: exact coordinator runtime: %v", errProductionLifecycleDependency, err)
	}
	cleanupPurger, err := newProductionLifecycleCleanupPurger(
		config.Pool,
		config.Runtime,
		config.ExtensionRoot,
		config.Database,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: cleanup purger: %v", errProductionLifecycleDependency, err)
	}
	repository := extensions.NewPostgresLifecycleRepository(config.Pool)
	migrations := extensionsruntime.NewProductionLifecycleBoundaryMigrations(config.Pool, config.MigrationEngine)
	preflight := extensionsruntime.NewProductionLifecycleBoundaryPreflight(
		extensionsruntime.ProductionLifecycleBoundaryPreflightConfig{
			Pool: config.Pool, Inventory: config.Store, Features: config.Features,
			Compatibility: config.Runtime, Trust: config.Trust, Migrations: migrations,
		},
	)
	staticPreflight := func(
		ctx context.Context,
		operation extensions.LifecycleMachineOperation,
		source *extensions.Extension,
		target extensions.Extension,
	) error {
		return preflight.CheckLifecycleStaticPreflight(ctx, extensionsruntime.LifecycleStaticPreflightRequest{
			Operation: operation, SourceExtension: source, TargetExtension: target,
		})
	}

	journal := extensionsruntime.NewPostgresLifecycleBoundaryPublicationJournal(config.Pool)
	// 当前只存在 admission owner，不虚构尚未实现的 production schedule trigger。
	schedules := supportjobs.NewPluginScheduleAdmissionRegistry()
	jobStore := hostapi.NewPostgresPluginJobLifecycleStore(config.Pool, config.River)
	jobCoordinator := &hostapi.PluginJobLifecycleCoordinator{Store: jobStore}
	jobs := extensionsruntime.NewPostgresLifecycleBoundaryJobs(
		extensionsruntime.PostgresLifecycleBoundaryJobsConfig{
			Pool: config.Pool, Runtime: config.Runtime, Schedules: schedules,
			Coordinator: jobCoordinator, Trust: config.Trust, Journal: journal,
		},
	)

	// P4 只建立 publication foundation；P6 才会接入完整 core route catalog 与执行链。
	routeFoundation := routes.NewRegistry()
	registryRepository := extensionsruntime.NewPostgresLifecycleRegistryPublicationRepository(config.Pool)
	registries := extensionsruntime.NewPostgresLifecycleBoundaryRegistries(
		extensionsruntime.LifecycleRegistryBoundaryConfig{
			Repository: registryRepository, Manager: config.Runtime, Pages: config.Pages,
			Routes: routeFoundation, Services: config.Services,
		},
	)
	state := extensionsruntime.NewPostgresLifecycleBoundaryState(config.Store)
	cleanup := extensionsruntime.NewPostgresLifecycleBoundaryCleanup(config.Pool)
	cleanupFinalizer := extensionsruntime.NewPostgresLifecycleBoundaryCleanupFinalizer(
		config.Pool,
		cleanupPurger,
	)
	boundary := extensionsruntime.NewComposedLifecycleHostBoundary(
		extensionsruntime.ComposedLifecycleHostBoundaryDependencies{
			Runtime: config.Runtime, Preflight: preflight, Migrations: migrations,
			Jobs: jobs, Registries: registries, State: state, Journal: journal, Cleanup: cleanup,
		},
	)
	host := extensionsruntime.NewExactLifecycleCoordinatorHost(config.Runtime, boundary)
	coordinator := extensions.NewLifecycleCoordinator(repository, runtime, host)

	return &productionLifecycleStack{
		Repository: repository, RuntimeManager: config.Runtime, Runtime: runtime,
		Preflight: preflight, StaticPreflight: staticPreflight,
		MigrationEngine: config.MigrationEngine, Migrations: migrations,
		Schedules: schedules, JobStore: jobStore, JobCoordinator: jobCoordinator, Jobs: jobs,
		RouteFoundation: routeFoundation, RegistryRepository: registryRepository, Registries: registries,
		State: state, PublicationJournal: journal, Cleanup: cleanup,
		Database: config.Database, CleanupPurger: cleanupPurger,
		CleanupFinalizer: cleanupFinalizer, Boundary: boundary,
		Host: host, Coordinator: coordinator,
	}, nil
}

func (s *productionLifecycleStack) bindService(service *extensions.Service) error {
	if s == nil || service == nil || s.Coordinator == nil || s.StaticPreflight == nil ||
		s.Repository == nil || s.CleanupFinalizer == nil {
		return errProductionLifecycleDependency
	}
	extensions.WithLifecycleCoordinator(s.Coordinator, s.StaticPreflight, s.Repository)(service)
	extensions.WithLifecycleInspectionRepository(s.Repository)(service)
	extensions.WithLifecycleCleanupFinalizer(adaptLifecycleCleanupFinalizer(s.CleanupFinalizer))(service)
	return nil
}

func adaptLifecycleCleanupFinalizer(
	finalizer extensionsruntime.LifecycleBoundaryCleanupFinalizer,
) extensions.LifecycleCleanupFinalizer {
	return func(ctx context.Context, operationID int64) (extensions.LifecycleCleanupFinalization, error) {
		if finalizer == nil {
			return extensions.LifecycleCleanupFinalization{}, errProductionLifecycleDependency
		}
		result, err := finalizer.FinalizeLifecycleHostCleanup(ctx, operationID)
		if err != nil {
			return extensions.LifecycleCleanupFinalization{}, err
		}
		return extensions.LifecycleCleanupFinalization{
			OperationID: result.OperationID, Status: result.Status,
			PhysicalPurgeComplete: result.PhysicalPurgeCompleted,
		}, nil
	}
}

type extensionRuntimeReconciler interface {
	Reconcile(context.Context, []extensions.Extension)
}

func reconcileAPIExtensionRuntime(
	ctx context.Context,
	safeMode bool,
	store extensions.Store,
	runtime extensionRuntimeReconciler,
) error {
	if store == nil || runtime == nil {
		return fmt.Errorf("%w: runtime reconciliation", errProductionLifecycleDependency)
	}
	items, err := store.List(ctx)
	if err != nil {
		return err
	}
	if safeMode {
		items = nil
	}
	runtime.Reconcile(ctx, items)
	return nil
}

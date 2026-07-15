package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	componentcatalog "github.com/zhuchunshu/sforum/apps/api/app/Support/ComponentCatalog"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	pages "github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

type lifecycleFeatureFacts struct{}

func (lifecycleFeatureFacts) MissingRequiredFeatures(context.Context, []string) ([]string, error) {
	return nil, nil
}

type lifecycleRiverClient struct{}

func (lifecycleRiverClient) InsertTx(
	context.Context,
	pgx.Tx,
	river.JobArgs,
	*river.InsertOpts,
) (*rivertype.JobInsertResult, error) {
	return nil, nil
}

func (lifecycleRiverClient) JobCancelTx(context.Context, pgx.Tx, int64) (*rivertype.JobRow, error) {
	return nil, nil
}

type lifecycleMigrationEngine struct{}

func (lifecycleMigrationEngine) ReconcileLifecycleMigration(
	context.Context,
	extensionsruntime.LifecycleMigrationEnginePlan,
) error {
	return nil
}

func (lifecycleMigrationEngine) InspectLifecycleMigration(
	context.Context,
	extensionsruntime.LifecycleMigrationEnginePlan,
) (extensionsruntime.LifecycleMigrationEngineProof, error) {
	return extensionsruntime.LifecycleMigrationEngineProof{}, nil
}

type lifecycleDatabaseDisposition struct{}

func (lifecycleDatabaseDisposition) ApplyLifecycleDataDisposition(
	context.Context,
	extensionsruntime.ExtensionDatabaseDispositionRequest,
) (extensionsruntime.ExtensionDatabaseDispositionReceipt, error) {
	return extensionsruntime.ExtensionDatabaseDispositionReceipt{}, nil
}

func newBootstrapLifecycleStack(t *testing.T) (*productionLifecycleStack, *extensionsruntime.Manager, *extensions.PostgresStore) {
	return newBootstrapLifecycleStackWithSafeMode(t, false)
}

func newBootstrapLifecycleStackWithSafeMode(
	t *testing.T,
	safeMode bool,
) (*productionLifecycleStack, *extensionsruntime.Manager, *extensions.PostgresStore) {
	t.Helper()
	pool := &pgxpool.Pool{}
	store := extensions.NewPostgresStore(pool)
	trust := extensions.NewExecutableTrustService(store, extensions.NewPostgresExecutableTrustStore(pool))
	manager := extensionsruntime.NewManager(extensionsruntime.ManagerConfig{
		Starter: extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{}),
	})
	stack, err := newProductionLifecycleStack(productionLifecycleStackConfig{
		Pool: pool, Store: store, Features: lifecycleFeatureFacts{}, Trust: trust,
		Runtime: manager, Pages: pages.NewRegistry(nil), Services: hostapi.NewServiceRegistry(),
		River: lifecycleRiverClient{}, MigrationEngine: lifecycleMigrationEngine{},
		ExtensionRoot: t.TempDir(), Database: lifecycleDatabaseDisposition{}, SafeMode: safeMode,
	})
	if err != nil {
		t.Fatalf("new production lifecycle stack: %v", err)
	}
	return stack, manager, store
}

func TestProductionLifecycleStackConstructsEveryRequiredDependency(t *testing.T) {
	stack, manager, _ := newBootstrapLifecycleStack(t)

	checks := map[string]bool{
		"repository": stack.Repository != nil, "runtime": stack.Runtime != nil,
		"preflight":        stack.Preflight != nil && stack.StaticPreflight != nil,
		"migration engine": stack.MigrationEngine != nil, "migrations": stack.Migrations != nil,
		"schedules": stack.Schedules != nil,
		"job store": stack.JobStore != nil, "job coordinator": stack.JobCoordinator != nil,
		"jobs": stack.Jobs != nil, "route registry": stack.RouteRegistry != nil,
		"route schemas":       stack.RouteSchemas != nil,
		"component registry":  stack.ComponentRegistry != nil,
		"route providers":     stack.RouteProviders != nil,
		"registry repository": stack.RegistryRepository != nil, "registries": stack.Registries != nil,
		"state": stack.State != nil, "journal": stack.PublicationJournal != nil,
		"cleanup": stack.Cleanup != nil, "cleanup purger": stack.CleanupPurger != nil,
		"database disposition": stack.Database != nil,
		"cleanup finalizer":    stack.CleanupFinalizer != nil, "boundary": stack.Boundary != nil,
		"host": stack.Host != nil, "coordinator": stack.Coordinator != nil,
	}
	for name, ok := range checks {
		if !ok {
			t.Fatalf("production lifecycle dependency %q is nil", name)
		}
	}
	if stack.RuntimeManager != manager {
		t.Fatal("coordinator stack did not retain the exact API runtime Manager")
	}
	if stack.Registries.ComponentRegistry() != stack.ComponentRegistry {
		t.Fatal("lifecycle boundary and production stack use different Component Registry instances")
	}
	snapshot := stack.RouteRegistry.Snapshot()
	if snapshot.Revision != 1 || snapshot.SafeMode || len(snapshot.Routes) != len(routes.CoreRouteCatalog()) ||
		len(snapshot.Conflicts) != 0 {
		t.Fatalf("production core route snapshot = revision %d safe=%t routes=%d conflicts=%#v",
			snapshot.Revision, snapshot.SafeMode, len(snapshot.Routes), snapshot.Conflicts)
	}
	for _, routeID := range []string{"core.route.system.health", "core.route.system.ready"} {
		found := false
		for _, route := range snapshot.Routes {
			if route.ID == routeID && route.Provider.Kind == routes.ProviderCore {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("production core catalog omitted Host route %q", routeID)
		}
	}

	// lifecycle boundary 必须持有 stack 暴露的同一个 Registry，否则激活发布与
	// 请求解析会落到两份不同的快照。
	boundaryRoutes := reflect.ValueOf(stack.Registries).Elem().FieldByName("routes")
	if boundaryRoutes.IsNil() || boundaryRoutes.Pointer() != reflect.ValueOf(stack.RouteRegistry).Pointer() {
		t.Fatal("lifecycle boundary and production stack use different Route Registry instances")
	}
	boundarySchemas := reflect.ValueOf(stack.Registries).Elem().FieldByName("routeSchemas")
	if boundarySchemas.IsNil() || boundarySchemas.Pointer() != reflect.ValueOf(stack.RouteSchemas).Pointer() {
		t.Fatal("lifecycle boundary and production stack use different Route Schema Publication instances")
	}
}

func TestProductionLifecycleStackRestoresComponentsThroughSharedRegistry(t *testing.T) {
	stack, _, _ := newBootstrapLifecycleStack(t)
	extension := bootstrapLifecycleComponentExtension(t, "bootstrap.component.normal")
	if err := stack.Registries.RestoreRoutePublications(
		context.Background(), []extensions.Extension{extension}, false,
	); err != nil {
		t.Fatalf("restore production component publication: %v", err)
	}
	runtime, ok := stack.ComponentRegistry.RuntimeSnapshot(extension.ID)
	if !ok || runtime.Extension.PackageDigest != extension.PackageDigest ||
		!strings.HasPrefix(runtime.InstanceID, "host-component-package:") {
		t.Fatalf("restored production component runtime = %#v, %t", runtime, ok)
	}
	if snapshot := stack.ComponentRegistry.Snapshot(); snapshot.Revision != 1 || len(snapshot.Contributions) != 1 {
		t.Fatalf("restored production component snapshot = %#v", snapshot)
	}
}

func TestProductionLifecycleStackSafeModeKeepsHostRoutesAndSkipsThirdParty(t *testing.T) {
	stack, _, _ := newBootstrapLifecycleStackWithSafeMode(t, true)
	component := bootstrapLifecycleComponentExtension(t, "bootstrap.component.safe-mode")
	if err := stack.ComponentRegistry.ReplaceRuntime(component, "pre-safe-mode-component"); err != nil {
		t.Fatalf("seed pre-Safe-Mode component: %v", err)
	}
	if err := stack.Registries.RestoreRoutePublications(
		context.Background(), []extensions.Extension{component}, true,
	); err != nil {
		t.Fatalf("restore Safe Mode component publication: %v", err)
	}
	if _, ok := stack.ComponentRegistry.RuntimeSnapshot(component.ID); ok {
		t.Fatal("Safe Mode restore retained a third-party component runtime")
	}
	if componentSnapshot := stack.ComponentRegistry.Snapshot(); componentSnapshot.Revision != 2 || len(componentSnapshot.Contributions) != 0 {
		t.Fatalf("Safe Mode component snapshot = %#v", componentSnapshot)
	}
	publication := stack.RouteRegistry.PublicationSnapshot().Publication
	publication.Plugins = []routes.PluginRouteSet{{
		Artifact: routes.PluginArtifact{
			ExtensionID: "third.party", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("a", 64), RuntimeInstanceID: "runtime-safe-mode",
		},
		Routes: []extensionmanifest.ManifestRoute{{
			ID: "third.party.route", ContractVersion: "third.party.route@1",
			Action: extensionmanifest.RouteActionAdd, Path: "/third-party", Methods: []string{"GET"},
			Guard: extensionmanifest.GuardCorePublic, Mode: extensionmanifest.RouteModeHTTP,
			Handler: "routes/third-party", ResponseSchema: "third.party.response@1",
		}},
	}}
	snapshot, err := stack.RouteRegistry.Publish(publication)
	if err != nil {
		t.Fatalf("safe-mode route publication: %v", err)
	}
	if !snapshot.SafeMode || len(snapshot.Routes) != len(routes.CoreRouteCatalog()) ||
		len(stack.RouteRegistry.PublicationSnapshot().Publication.Plugins) != 0 {
		t.Fatalf("safe-mode snapshot = safe %t routes %d publication %#v",
			snapshot.SafeMode, len(snapshot.Routes), stack.RouteRegistry.PublicationSnapshot().Publication)
	}
	if _, err := stack.RouteRegistry.Resolve("GET", "/third-party"); !errors.Is(err, routes.ErrRouteNotFound) {
		t.Fatalf("safe mode exposed third-party route: %v", err)
	}
	for path, routeID := range map[string]string{
		"/api/v1/health": "core.route.system.health",
		"/api/v1/ready":  "core.route.system.ready",
	} {
		match, err := stack.RouteRegistry.Resolve("GET", path)
		if err != nil || match.Route.ID != routeID || match.Route.Provider.Kind != routes.ProviderCore {
			t.Fatalf("safe-mode Host route %s = %#v, %v", path, match, err)
		}
	}
}

func bootstrapLifecycleComponentExtension(t *testing.T, id string) extensions.Extension {
	t.Helper()
	target, ok := componentcatalog.FindCoreComponent("core.component.page.forum.home")
	if !ok {
		t.Fatal("reviewed forum home component target is missing")
	}
	return extensions.Extension{
		ID: id, Version: "1.0.0", Type: extensions.TypePlugin, Status: extensions.StatusEnabled,
		PackageDigest: strings.Repeat("a", 64),
		Manifest: extensions.Manifest{
			ManifestVersion: 3, ID: id, Version: "1.0.0", Type: extensions.TypePlugin,
			Components: []extensions.ManifestComponent{{
				ID: id + ".component.hide-home", ContractVersion: id + ".component.hide-home@1",
				Action:   extensionmanifest.ComponentActionHide,
				TargetID: target.ID, TargetContractVersion: target.ContractVersion,
			}},
		},
	}
}

func TestProductionLifecycleStackFailsClosedWithoutExactManager(t *testing.T) {
	if _, err := requireProductionExtensionRuntime(nil); !errors.Is(err, errProductionLifecycleDependency) {
		t.Fatalf("nil runtime error = %v", err)
	}
	if _, err := requireProductionExtensionRuntime(fakeBootstrapExtensionRuntime{}); !errors.Is(err, errProductionLifecycleDependency) {
		t.Fatalf("non-Manager runtime error = %v", err)
	}

	manager := extensionsruntime.NewManager(extensionsruntime.ManagerConfig{})
	pool := &pgxpool.Pool{}
	_, err := newProductionLifecycleStack(productionLifecycleStackConfig{
		Pool: pool, Store: extensions.NewPostgresStore(pool),
		Features: lifecycleFeatureFacts{},
		Trust: extensions.NewExecutableTrustService(
			extensions.NewPostgresStore(pool),
			extensions.NewPostgresExecutableTrustStore(pool),
		),
		Runtime: manager, Pages: pages.NewRegistry(nil), Services: hostapi.NewServiceRegistry(),
		River: lifecycleRiverClient{}, MigrationEngine: lifecycleMigrationEngine{},
		ExtensionRoot: t.TempDir(), Database: lifecycleDatabaseDisposition{},
	})
	if !errors.Is(err, errProductionLifecycleDependency) {
		t.Fatalf("Manager without protocol-v2 exact runner error = %v", err)
	}
}

func TestProductionLifecycleStackBindsV2AndInspectionOptions(t *testing.T) {
	stack, _, store := newBootstrapLifecycleStack(t)
	service := extensions.NewService(store, t.TempDir())
	if err := stack.bindService(service); err != nil {
		t.Fatalf("bind lifecycle service: %v", err)
	}

	value := reflect.ValueOf(service).Elem()
	for _, field := range []string{
		"lifecycleCoordinator", "lifecyclePreflight", "lifecycleAuthority", "lifecycleInspector",
		"lifecycleFinalizer", "componentRegistry",
	} {
		binding := value.FieldByName(field)
		if !binding.IsValid() || binding.IsNil() {
			t.Fatalf("Service option %q was not wired", field)
		}
	}
	componentBinding := value.FieldByName("componentRegistry")
	if componentBinding.Elem().Pointer() != reflect.ValueOf(stack.ComponentRegistry).Pointer() {
		t.Fatal("theme activation service did not receive the shared production Component Registry")
	}
}

type lifecycleCleanupFinalizer struct {
	result extensionsruntime.LifecycleCleanupFinalizationResult
	err    error
}

func (f lifecycleCleanupFinalizer) FinalizeLifecycleHostCleanup(
	context.Context,
	int64,
) (extensionsruntime.LifecycleCleanupFinalizationResult, error) {
	return f.result, f.err
}

func TestLifecycleCleanupFinalizerAdapterMapsOnlyModelsContract(t *testing.T) {
	want := extensionsruntime.LifecycleCleanupFinalizationResult{
		CleanupID: "internal-cleanup", OperationID: 71, Status: "finalized",
		PhysicalPurgeCompleted: true, PurgeReceiptID: "internal-receipt",
		PurgeProofDigest: "internal-proof",
	}
	got, err := adaptLifecycleCleanupFinalizer(lifecycleCleanupFinalizer{result: want})(
		context.Background(),
		want.OperationID,
	)
	if err != nil {
		t.Fatalf("adapt cleanup finalizer: %v", err)
	}
	if got.OperationID != want.OperationID || got.Status != want.Status || !got.PhysicalPurgeComplete {
		t.Fatalf("adapted cleanup result = %#v", got)
	}
}

type lifecycleReconcileStore struct {
	extensions.Store
	items []extensions.Extension
	err   error
}

func (s lifecycleReconcileStore) List(context.Context) ([]extensions.Extension, error) {
	return append([]extensions.Extension(nil), s.items...), s.err
}

type lifecycleReconcileRuntime struct {
	calls int
	items []extensions.Extension
}

func (r *lifecycleReconcileRuntime) Reconcile(_ context.Context, items []extensions.Extension) {
	r.calls++
	r.items = append([]extensions.Extension(nil), items...)
}

func TestReconcileAPIExtensionRuntimeKeepsSafeModePluginFree(t *testing.T) {
	item := extensions.Extension{ID: "demo.safe-mode", Type: extensions.TypePlugin, Status: extensions.StatusEnabled}
	store := lifecycleReconcileStore{items: []extensions.Extension{item}}

	safeRuntime := &lifecycleReconcileRuntime{}
	if items, err := reconcileAPIExtensionRuntime(context.Background(), true, store, safeRuntime); err != nil || len(items) != 0 {
		t.Fatalf("safe mode reconcile: %v", err)
	}
	if safeRuntime.calls != 1 || len(safeRuntime.items) != 0 {
		t.Fatalf("safe mode reconciled %#v", safeRuntime.items)
	}

	normalRuntime := &lifecycleReconcileRuntime{}
	items, err := reconcileAPIExtensionRuntime(context.Background(), false, store, normalRuntime)
	if err != nil {
		t.Fatalf("normal reconcile: %v", err)
	}
	if normalRuntime.calls != 1 || !reflect.DeepEqual(normalRuntime.items, []extensions.Extension{item}) ||
		!reflect.DeepEqual(items, []extensions.Extension{item}) {
		t.Fatalf("normal mode reconciled %#v", normalRuntime.items)
	}
}

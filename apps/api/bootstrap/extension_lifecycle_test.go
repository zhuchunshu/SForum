package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	pages "github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
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
		ExtensionRoot: t.TempDir(), Database: lifecycleDatabaseDisposition{},
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
		"jobs": stack.Jobs != nil, "route foundation": stack.RouteFoundation != nil,
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
	if stack.RouteFoundation.Revision() != 0 {
		t.Fatal("P4 route foundation must not masquerade as a populated P6 production registry")
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
		"lifecycleFinalizer",
	} {
		binding := value.FieldByName(field)
		if !binding.IsValid() || binding.IsNil() {
			t.Fatalf("Service option %q was not wired", field)
		}
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
	if err := reconcileAPIExtensionRuntime(context.Background(), true, store, safeRuntime); err != nil {
		t.Fatalf("safe mode reconcile: %v", err)
	}
	if safeRuntime.calls != 1 || len(safeRuntime.items) != 0 {
		t.Fatalf("safe mode reconciled %#v", safeRuntime.items)
	}

	normalRuntime := &lifecycleReconcileRuntime{}
	if err := reconcileAPIExtensionRuntime(context.Background(), false, store, normalRuntime); err != nil {
		t.Fatalf("normal reconcile: %v", err)
	}
	if normalRuntime.calls != 1 || !reflect.DeepEqual(normalRuntime.items, []extensions.Extension{item}) {
		t.Fatalf("normal mode reconciled %#v", normalRuntime.items)
	}
}

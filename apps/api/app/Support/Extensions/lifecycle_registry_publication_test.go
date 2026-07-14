package extensionsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	pages "github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
)

func TestLifecycleRegistryPublicationConvergesAcrossRestartWhileTargetStaysDrained(t *testing.T) {
	ctx := context.Background()
	repository := &memoryLifecycleRegistryRepository{phase: LifecycleRegistryPublicationSource}
	manager := NewManager(ManagerConfig{Starter: newManagerStagedStarter()})
	pageRegistry := pages.NewRegistry(nil)
	routeRegistry := routes.NewRegistry()
	serviceRegistry := hostapi.NewServiceRegistry()
	boundary := NewPostgresLifecycleBoundaryRegistries(LifecycleRegistryBoundaryConfig{
		Repository: repository, Manager: manager, Pages: pageRegistry,
		Routes: routeRegistry, Services: serviceRegistry,
	})

	source := lifecycleRegistryTestExtension(t, "1.0.0", strings.Repeat("a", 64), 1, "/registry-source")
	if err := manager.Start(ctx, source); err != nil {
		t.Fatal(err)
	}
	sourceRuntime, err := manager.ActiveRuntimeInstance(source.ID)
	if err != nil {
		t.Fatal(err)
	}
	sourceBinding := lifecycleRegistryBinding(source, sourceRuntime.Identity.InstanceID)
	if _, err := pageRegistry.PublishExtensionIfRevision(lifecyclePageArtifact(source, sourceBinding), lifecyclePageItems(t, source, sourceBinding), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := routeRegistry.Publish(routes.Publication{Plugins: []routes.PluginRouteSet{lifecycleRouteSet(source, sourceBinding)}}); err != nil {
		t.Fatal(err)
	}
	if err := serviceRegistry.ReplaceExtension(source.ID, []hostapi.ServiceRegistration{lifecycleServiceRegistration(source, sourceBinding.RuntimeInstanceID)}); err != nil {
		t.Fatal(err)
	}

	target := lifecycleRegistryTestExtension(t, "2.0.0", strings.Repeat("b", 64), 2, "/registry-target")
	targetRuntime, err := manager.StageRuntimeInstance(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.HealthRuntimeInstance(ctx, targetRuntime.Identity); err != nil {
		t.Fatal(err)
	}
	targetBinding := lifecycleRegistryBinding(target, targetRuntime.Identity.InstanceID)
	request := lifecycleRegistryRequest(source, target, sourceBinding, targetBinding, 1)
	transaction, err := boundary.PrepareLifecycleRegistryPublication(ctx, request, LifecycleBoundaryActivate)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.BeginDrain(sourceRuntime.Identity); err != nil {
		t.Fatal(err)
	}
	if err := manager.WaitDrain(ctx, sourceRuntime.Identity); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.BeginDrain(targetRuntime.Identity); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.PublishDrainedRuntimeInstance(ctx, targetRuntime.Identity); err != nil {
		t.Fatal(err)
	}
	if err := serviceRegistry.ReplaceExtension(target.ID, []hostapi.ServiceRegistration{lifecycleServiceRegistration(target, targetBinding.RuntimeInstanceID)}); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Publish(ctx); err != nil {
		t.Fatal(err)
	}
	assertLifecycleRegistryTargetHidden(t, manager, pageRegistry, routeRegistry, targetRuntime.Identity)

	// A fresh adapter/transaction reconstructs both plans from immutable package
	// material and the durable phase; no reverse closure from attempt 1 is used.
	request.Attempt = 2
	restarted := NewPostgresLifecycleBoundaryRegistries(LifecycleRegistryBoundaryConfig{
		Repository: repository, Manager: manager, Pages: pageRegistry,
		Routes: routeRegistry, Services: serviceRegistry,
	})
	recovered, err := restarted.PrepareLifecycleRegistryPublication(ctx, request, LifecycleBoundaryActivate)
	if err != nil {
		t.Fatal(err)
	}
	if state, err := recovered.Inspect(ctx); err != nil || state != LifecycleBoundaryTransactionTarget {
		t.Fatalf("restart state = %q, %v", state, err)
	}
	if err := recovered.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.PublishDrainedRuntimeInstance(ctx, sourceRuntime.Identity); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ResumeRuntimeInstance(sourceRuntime.Identity); err != nil {
		t.Fatal(err)
	}
	if page, ok := pageRegistry.ResolveAddedPath("/registry-source"); !ok || page.RuntimeInstanceID != sourceBinding.RuntimeInstanceID {
		t.Fatalf("restored source page = %#v, %t", page, ok)
	}
	if _, ok := pageRegistry.ResolveAddedPath("/registry-target"); ok {
		t.Fatal("target page survived source convergence")
	}
	if hook, ok := manager.HookBus().RuntimeSnapshot(source.ID); !ok || hook.InstanceID != sourceBinding.RuntimeInstanceID {
		t.Fatalf("restored hook = %#v, %t", hook, ok)
	}
	if manager.HookBus().UnregisterRuntime(source.ID, targetBinding.RuntimeInstanceID) {
		t.Fatal("stale target removed restored source hooks")
	}
}

func TestLifecycleRegistryPublicationSupportsPackageWithoutThemeJSON(t *testing.T) {
	extension := lifecycleRegistryTestExtension(t, "1.0.0", strings.Repeat("c", 64), 3, "")
	if err := os.Remove(filepath.Join(extension.PackagePath, "theme.json")); err != nil {
		t.Fatal(err)
	}
	binding := lifecycleRegistryBinding(extension, "runtime-empty-pages")
	material, err := buildLifecycleRegistryMaterial(extension, binding)
	if err != nil || len(material.pages) != 0 || material.digest == "" {
		t.Fatalf("empty Page material = %#v, %v", material.pages, err)
	}
}

func TestLifecycleRegistryPublicationRejectsSameArtifactFromDifferentRuntimeInstance(t *testing.T) {
	extension := lifecycleRegistryTestExtension(t, "1.0.0", strings.Repeat("c", 64), 4, "/instance-fence")
	binding := lifecycleRegistryBinding(extension, "expected-runtime")
	material, err := buildLifecycleRegistryMaterial(extension, binding)
	if err != nil {
		t.Fatal(err)
	}

	foreignRoutes := lifecycleRouteSet(extension, binding)
	foreignRoutes.Artifact.RuntimeInstanceID = "foreign-runtime"
	if _, err := replaceLifecycleRouteSet(
		routes.Publication{Plugins: []routes.PluginRouteSet{foreignRoutes}},
		extension.ID,
		nil,
		&material,
		nil,
	); !errors.Is(err, ErrLifecycleRegistryPublicationConflict) {
		t.Fatalf("same package from foreign route runtime was accepted: %v", err)
	}

	foreignPage := lifecyclePageArtifact(extension, binding)
	foreignPage.RuntimeInstanceID = "foreign-runtime"
	if pageArtifactAllowed(foreignPage, &material) {
		t.Fatal("same package from foreign page runtime was accepted")
	}
}

func TestLifecycleRegistryPartialFamilyFailureStaysInvisibleAndFailsClosed(t *testing.T) {
	ctx := context.Background()
	repository := &memoryLifecycleRegistryRepository{phase: LifecycleRegistryPublicationSource}
	manager := NewManager(ManagerConfig{Starter: newManagerStagedStarter()})
	pageRegistry := pages.NewRegistry(nil)
	routeRegistry := routes.NewRegistry()
	serviceRegistry := hostapi.NewServiceRegistry()
	boundary := NewPostgresLifecycleBoundaryRegistries(LifecycleRegistryBoundaryConfig{
		Repository: repository, Manager: manager, Pages: pageRegistry,
		Routes: routeRegistry, Services: serviceRegistry,
	})

	source := lifecycleRegistryTestExtension(t, "1.0.0", strings.Repeat("d", 64), 11, "/partial-source")
	if err := manager.Start(ctx, source); err != nil {
		t.Fatal(err)
	}
	sourceRuntime, err := manager.ActiveRuntimeInstance(source.ID)
	if err != nil {
		t.Fatal(err)
	}
	sourceBinding := lifecycleRegistryBinding(source, sourceRuntime.Identity.InstanceID)
	if _, err := pageRegistry.PublishExtensionIfRevision(
		lifecyclePageArtifact(source, sourceBinding), lifecyclePageItems(t, source, sourceBinding), 0,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := routeRegistry.Publish(routes.Publication{Plugins: []routes.PluginRouteSet{
		lifecycleRouteSet(source, sourceBinding),
	}}); err != nil {
		t.Fatal(err)
	}
	if err := serviceRegistry.ReplaceExtension(source.ID, []hostapi.ServiceRegistration{
		lifecycleServiceRegistration(source, sourceBinding.RuntimeInstanceID),
	}); err != nil {
		t.Fatal(err)
	}

	target := lifecycleRegistryTestExtension(t, "2.0.0", strings.Repeat("e", 64), 12, "/partial-target")
	targetRuntime, err := manager.StageRuntimeInstance(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.HealthRuntimeInstance(ctx, targetRuntime.Identity); err != nil {
		t.Fatal(err)
	}
	targetBinding := lifecycleRegistryBinding(target, targetRuntime.Identity.InstanceID)
	request := lifecycleRegistryRequest(source, target, sourceBinding, targetBinding, 1)
	transaction, err := boundary.PrepareLifecycleRegistryPublication(ctx, request, LifecycleBoundaryActivate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.BeginDrain(sourceRuntime.Identity); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.BeginDrain(targetRuntime.Identity); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.PublishDrainedRuntimeInstance(ctx, targetRuntime.Identity); err != nil {
		t.Fatal(err)
	}
	if err := serviceRegistry.ReplaceExtension(target.ID, []hostapi.ServiceRegistration{
		lifecycleServiceRegistration(target, targetBinding.RuntimeInstanceID),
	}); err != nil {
		t.Fatal(err)
	}

	// Inject a same-extension artifact that is outside the frozen source/target
	// pair. Route reconciliation runs before Page and therefore leaves a partial
	// local snapshot when Page rejects it.
	foreign := pages.RuntimeArtifact{
		ExtensionID: target.ID, ExtensionVersion: "9.9.9",
		PackageDigest: strings.Repeat("f", 64), RuntimeInstanceID: "foreign-runtime",
	}
	if _, err := pageRegistry.PublishExtensionIfRevision(foreign, nil, pageRegistry.Revision()); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Publish(ctx); !errors.Is(err, ErrLifecycleRegistryPublicationConflict) {
		t.Fatalf("partial family publication error = %v", err)
	}
	assertLifecycleRegistryTargetHidden(t, manager, pageRegistry, routeRegistry, targetRuntime.Identity)
	if state, err := transaction.Inspect(ctx); err != nil || state != LifecycleBoundaryTransactionSource {
		t.Fatalf("failed aggregate durable phase = %q, %v", state, err)
	}

	// Recovery cannot silently overwrite the foreign writer. Once that conflict
	// is explicitly removed, the frozen source set converges and may reopen.
	if err := transaction.Restore(ctx); !errors.Is(err, ErrLifecycleRegistryPublicationConflict) {
		t.Fatalf("restore across foreign Page artifact = %v", err)
	}
	if _, err := pageRegistry.RemoveExtensionIfRevision(target.ID, foreign, pageRegistry.Revision()); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.PublishDrainedRuntimeInstance(ctx, sourceRuntime.Identity); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ResumeRuntimeInstance(sourceRuntime.Identity); err != nil {
		t.Fatal(err)
	}
	if page, ok := pageRegistry.ResolveAddedPath("/partial-source"); !ok || page.RuntimeInstanceID != sourceBinding.RuntimeInstanceID {
		t.Fatalf("recovered source page = %#v, %t", page, ok)
	}
}

func assertLifecycleRegistryTargetHidden(
	t *testing.T,
	manager *Manager,
	pageRegistry *pages.Registry,
	routeRegistry *routes.Registry,
	identity RuntimeInstanceIdentity,
) {
	t.Helper()
	if _, ok := pageRegistry.ResolveAddedPath("/registry-target"); ok {
		t.Fatal("drained target page is visible before marker")
	}
	if _, err := routeRegistry.Resolve("GET", "/registry-target/api"); !errors.Is(err, routes.ErrRouteNotFound) {
		t.Fatalf("drained target route = %v", err)
	}
	for _, class := range []RuntimeCallClass{RuntimeCallPage, RuntimeCallRoute, RuntimeCallHook, RuntimeCallService} {
		if _, err := manager.AcquireRuntimeCall(context.Background(), identity, class); !errors.Is(err, ErrRuntimeAdmissionDraining) {
			t.Fatalf("%s target admission = %v", class, err)
		}
	}
}

func lifecycleRegistryTestExtension(t *testing.T, version, digest string, versionID int64, pagePath string) extensions.Extension {
	t.Helper()
	extension := exactCoordinatorTestExtension("registry.demo", version, digest, "registry.demo.lifecycle@1", versionID)
	extension.Status = extensions.StatusEnabled
	extension.PackagePath = t.TempDir()
	extension.Manifest.Events = []extensions.ManifestEvent{{
		ID: "registry.demo.changed", ContractVersion: "registry.demo.changed@1", Name: "registry.demo.changed", Kind: "observe",
	}}
	extension.Manifest.Services = []extensions.ManifestService{{
		ID: "registry.demo.echo", ContractVersion: "registry.demo.echo@1", Action: hostapi.ServiceActionAdd,
		Handler: "echo", RequestSchema: "registry.demo.request@1", ResponseSchema: "registry.demo.response@1",
	}}
	extension.Manifest.Routes = []extensions.ManifestRoute{{
		ID: "registry.demo.read", ContractVersion: "registry.demo.read@1",
		Action: extensionmanifest.RouteActionAdd, Path: pagePath + "/api", Methods: []string{"GET"},
		Guard: extensionmanifest.GuardCorePublic, Mode: extensionmanifest.RouteModeHTTP,
		Fallback: "closed", Handler: "route.read", ResponseSchema: "registry.demo.read.response@1",
	}}
	packageDocument, err := json.Marshal(pages.ThemePackage{Pages: []pages.ThemePageDecl{{
		ID: "registry.demo.page", Action: string(pages.ActionAdd), Path: pagePath,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extension.PackagePath, "theme.json"), packageDocument, 0o600); err != nil {
		t.Fatal(err)
	}
	return extension
}

func lifecycleRegistryRequest(
	source, target extensions.Extension,
	sourceBinding, targetBinding extensions.LifecycleRuntimeBinding,
	attempt int,
) LifecycleBoundaryRequest {
	return LifecycleBoundaryRequest{
		OperationID: 101, Operation: extensions.LifecycleMachineUpgrade, Position: 8,
		StepID: "lifecycle.upgrade.08.host.enabled", Attempt: attempt,
		SourceExtension: &source, TargetExtension: target,
		SourceBinding: sourceBinding, TargetBinding: targetBinding,
		ActorUserID: 42, AuditEventID: 84,
	}
}

func lifecycleRegistryBinding(extension extensions.Extension, instanceID string) extensions.LifecycleRuntimeBinding {
	return extensions.LifecycleRuntimeBinding{
		ExtensionID: extension.ID, ExtensionVersion: extension.Version,
		PackageDigest: extension.PackageDigest, VersionID: extension.ActiveVersionID,
		RuntimeInstanceID: instanceID,
	}
}

func lifecyclePageArtifact(extension extensions.Extension, binding extensions.LifecycleRuntimeBinding) pages.RuntimeArtifact {
	return pages.RuntimeArtifact{
		ExtensionID: extension.ID, ExtensionVersion: extension.Version,
		PackageDigest: extension.PackageDigest, RuntimeInstanceID: binding.RuntimeInstanceID,
	}
}

func lifecyclePageItems(t *testing.T, extension extensions.Extension, binding extensions.LifecycleRuntimeBinding) []pages.PageContribution {
	t.Helper()
	material, err := buildLifecycleRegistryMaterial(extension, binding)
	if err != nil {
		t.Fatal(err)
	}
	return material.pages
}

func lifecycleRouteSet(extension extensions.Extension, binding extensions.LifecycleRuntimeBinding) routes.PluginRouteSet {
	return routes.PluginRouteSet{Artifact: routes.PluginArtifact{
		ExtensionID: extension.ID, ExtensionVersion: extension.Version,
		PackageDigest: extension.PackageDigest, RuntimeInstanceID: binding.RuntimeInstanceID,
	}, Routes: extension.Manifest.Routes}
}

type lifecycleRegistryServiceProvider struct{}

func (lifecycleRegistryServiceProvider) Invoke(
	context.Context, *protocolv2.RequestContext, string, string, string, *protocolv2.TypedDocument,
) (*protocolv2.TypedDocument, *protocolv2.ErrorDetail, error) {
	return &protocolv2.TypedDocument{SchemaId: "registry.demo.response", SchemaVersion: "1"}, nil, nil
}

func lifecycleServiceRegistration(extension extensions.Extension, instanceID string) hostapi.ServiceRegistration {
	return hostapi.ServiceRegistration{
		ExtensionID: extension.ID, InstanceID: instanceID, Action: hostapi.ServiceActionAdd,
		Descriptor: &protocolv2.ServiceDescriptor{
			ServiceId: "registry.demo.echo", Version: "1.0.0",
			RequestSchemaId: "registry.demo.request@1", ResponseSchemaId: "registry.demo.response@1",
		},
		Provider: lifecycleRegistryServiceProvider{},
	}
}

type memoryLifecycleRegistryRepository struct {
	mu        sync.Mutex
	input     PrepareLifecycleRegistryPublicationInput
	phase     LifecycleRegistryPublicationPhase
	attempt   int
	committed bool
}

func (r *memoryLifecycleRegistryRepository) PrepareLifecycleRegistryPublication(
	_ context.Context,
	input PrepareLifecycleRegistryPublicationInput,
) (LifecycleRegistryPublicationRef, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.attempt > input.Fence.Attempt {
		return LifecycleRegistryPublicationRef{}, ErrLifecycleRegistryPublicationConflict
	}
	if r.attempt > 0 && (!lifecycleRegistryFenceMatchesOperation(r.input.Fence, input.Fence) ||
		r.input.SourceDigest != input.SourceDigest || r.input.TargetDigest != input.TargetDigest) {
		return LifecycleRegistryPublicationRef{}, ErrLifecycleRegistryPublicationConflict
	}
	r.input = input
	r.attempt = input.Fence.Attempt
	return lifecycleRegistryRef(input.Fence), nil
}

func (r *memoryLifecycleRegistryRepository) InspectLifecycleRegistryPublication(
	_ context.Context,
	ref LifecycleRegistryPublicationRef,
) (LifecycleRegistryPublicationPhase, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ref.Attempt != r.attempt {
		return "", ErrLifecycleRegistryPublicationConflict
	}
	return r.phase, nil
}

func (r *memoryLifecycleRegistryRepository) MoveLifecycleRegistryPublication(
	_ context.Context,
	ref LifecycleRegistryPublicationRef,
	phase LifecycleRegistryPublicationPhase,
	apply func() error,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ref.Attempt != r.attempt {
		return ErrLifecycleRegistryPublicationConflict
	}
	if phase == LifecycleRegistryPublicationSource && r.committed {
		return ErrLifecycleRegistryPublicationCommitted
	}
	if err := apply(); err != nil {
		return err
	}
	r.phase = phase
	return nil
}

var _ LifecycleRegistryPublicationRepository = (*memoryLifecycleRegistryRepository)(nil)

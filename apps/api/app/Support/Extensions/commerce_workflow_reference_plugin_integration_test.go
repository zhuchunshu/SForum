package extensionsruntime_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	capabilities "github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	cacheregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/CacheRegistry"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

// TestReferenceCommerceWorkflowPluginAndExtenderJoinedGates proves the P13
// commerce/workflow package plus optional extender: routes/database/hooks/jobs/
// cache/openapi/components/services, happy path, failure, and disable fallback.
func TestReferenceCommerceWorkflowPluginAndExtenderJoinedGates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping reference commerce workflow plugin subprocess build in short mode")
	}
	commerce := buildReferenceCommerceExtension(t)
	extender := buildReferenceCommerceExtenderExtension(t)
	assertCommerceManifestSurfaces(t, commerce)
	if len(extender.Manifest.Dependencies) != 1 ||
		extender.Manifest.Dependencies[0].ID != "sforum.commerce-workflow" ||
		extender.Manifest.Dependencies[0].Kind != "required" {
		t.Fatalf("extender dependency = %#v", extender.Manifest.Dependencies)
	}

	gateway, _ := newProtocolV2HostGateway()
	t.Cleanup(func() { _ = gateway.Close() })
	// own_schema 声明要求 Host 在 Start 时签发 exact runtime 数据库租约。
	leases := newCommerceDatabaseLeaseRegistry()
	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Trust: staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{
			TrustGrantID: "commerce-reference", ImpactDigest: commerce.PackageDigest,
		}},
		HostAPI:                        gateway,
		DatabaseLeases:                 leases,
		DatabaseLeaseHeartbeatInterval: 50 * time.Millisecond,
		DatabaseLeaseOperationTimeout:  5 * time.Second,
	})
	manager := extensionsruntime.NewManager(extensionsruntime.ManagerConfig{Starter: starter})
	if err := manager.Start(t.Context(), commerce); err != nil {
		t.Fatalf("start commerce plugin: %v", err)
	}
	if err := manager.Start(t.Context(), extender); err != nil {
		t.Fatalf("start commerce extender: %v", err)
	}
	stopped := false
	t.Cleanup(func() {
		if !stopped {
			_ = manager.Stop(context.Background(), extender)
			_ = manager.Stop(context.Background(), commerce)
		}
	})
	active, err := manager.ActiveRuntimeInstance(commerce.ID)
	if err != nil {
		t.Fatal(err)
	}

	// --- Hooks: commerce evaluate happy path + failure ---
	okResult := starter.InvokeHook(t.Context(), commerce, extensionsruntime.HookInput{
		DeclarationID: "sforum.commerce-workflow.hook.order-evaluate",
		Name:          "sforum.commerce-workflow.order.evaluate",
		Kind:          "filter",
		ContractVersion: "sforum.commerce-workflow.hook.order-evaluate@1",
		Timeout:         time.Second,
		Payload:         map[string]any{"status": "pending", "orderId": "ord-1"},
		PatchFields:     []string{"status"},
	})
	if !okResult.OK || okResult.Patch["status"] != "approved" {
		t.Fatalf("commerce evaluate happy path = %#v", okResult)
	}
	failResult := starter.InvokeHook(t.Context(), commerce, extensionsruntime.HookInput{
		DeclarationID: "sforum.commerce-workflow.hook.order-evaluate",
		Name:          "sforum.commerce-workflow.order.evaluate",
		Kind:          "filter",
		ContractVersion: "sforum.commerce-workflow.hook.order-evaluate@1",
		Timeout:         time.Second,
		Payload:         map[string]any{"status": "fail", "orderId": "ord-2"},
		PatchFields:     []string{"status"},
	})
	if failResult.OK {
		t.Fatalf("commerce evaluate failure must fail closed: %#v", failResult)
	}

	// --- Extender hook targeting commerce evaluate ---
	extResult := starter.InvokeHook(t.Context(), extender, extensionsruntime.HookInput{
		DeclarationID: "sforum.commerce-workflow-ext.hook.order-enrich",
		Name:          "sforum.commerce-workflow.order.evaluate",
		Kind:          "filter",
		ContractVersion: "sforum.commerce-workflow.hook.order-evaluate@1",
		Timeout:         time.Second,
		Payload:         map[string]any{"status": "approved", "orderId": "ord-1"},
		PatchFields:     []string{"status"},
	})
	if !extResult.OK || extResult.Patch["status"] != "audited" {
		t.Fatalf("extender enrich = %#v", extResult)
	}

	// --- Services: commerce + extender discovery ---
	registry := gateway.ProtocolV2ServiceRegistry()
	commerceService, err := registry.ResolveExact("sforum.commerce-workflow.service.orders", "1.0.0")
	if err != nil || commerceService.Winner.ExtensionID != commerce.ID {
		t.Fatalf("commerce service resolve = %#v err=%v", commerceService, err)
	}
	auditService, err := registry.ResolveExact("sforum.commerce-workflow-ext.service.audit", "1.0.0")
	if err != nil || auditService.Winner.ExtensionID != extender.ID {
		t.Fatalf("extender service resolve = %#v err=%v", auditService, err)
	}
	if commerceService.Winner.Provider == nil {
		t.Fatal("commerce service winner missing provider channel")
	}

	// --- Cache publication from Manifest ---
	cachePub := cacheregistry.Publication{Artifact: cacheregistry.Artifact{
		ExtensionID: commerce.ID, ExtensionVersion: commerce.Version,
		PackageDigest: commerce.PackageDigest, VersionID: commerce.ActiveVersionID,
		RuntimeInstanceID: active.Identity.InstanceID,
	}}
	for _, item := range commerce.Manifest.Cache {
		cachePub.Caches = append(cachePub.Caches, cacheregistry.Declaration{
			ID: item.ID, ContractVersion: item.ContractVersion, Namespace: item.Namespace,
			Policy: item.Policy, Tags: append([]string(nil), item.Tags...),
			Provider: item.Provider, Invalidators: append([]string(nil), item.Invalidators...),
		})
	}
	// Resolve 需要 exact runtime admission；停机后必须 fail closed。
	cacheReg := cacheregistry.New().WithPluginAdmission(func(artifact cacheregistry.Artifact) bool {
		return artifact == cachePub.Artifact
	})
	if _, err := cacheReg.Publish(cachePub); err != nil {
		t.Fatalf("publish cache: %v", err)
	}
	if _, err := cacheReg.Resolve("sforum.commerce-workflow.cache.orders"); err != nil {
		t.Fatalf("resolve cache: %v", err)
	}

	// --- Disable: stop plugins, remove cache, services disappear ---
	if err := manager.Stop(t.Context(), extender); err != nil {
		t.Fatal(err)
	}
	if err := manager.Stop(t.Context(), commerce); err != nil {
		t.Fatal(err)
	}
	stopped = true
	if _, removed, err := cacheReg.Remove(cachePub.Artifact); err != nil || !removed {
		t.Fatalf("remove cache: removed=%v err=%v", removed, err)
	}
	if _, err := cacheReg.Resolve("sforum.commerce-workflow.cache.orders"); err != cacheregistry.ErrNotFound {
		t.Fatalf("cache after disable = %v", err)
	}
	// Service discovery must not keep stopped instances.
	if _, err := registry.ResolveExact("sforum.commerce-workflow.service.orders", "1.0.0"); err == nil {
		t.Fatal("stopped commerce service remained discoverable")
	}
	// Subsequent hooks fail closed after runtime stop.
	afterStop := starter.InvokeHook(t.Context(), commerce, extensionsruntime.HookInput{
		DeclarationID: "sforum.commerce-workflow.hook.order-evaluate",
		Name:          "sforum.commerce-workflow.order.evaluate",
		Kind:          "filter",
		ContractVersion: "sforum.commerce-workflow.hook.order-evaluate@1",
		Timeout:         time.Second,
		Payload:         map[string]any{"status": "pending"},
		PatchFields:     []string{"status"},
	})
	if afterStop.OK {
		t.Fatalf("stopped commerce hook must fail closed: %#v", afterStop)
	}
}

func assertCommerceManifestSurfaces(t *testing.T, extension extensions.Extension) {
	t.Helper()
	if extension.ID != "sforum.commerce-workflow" {
		t.Fatalf("id = %s", extension.ID)
	}
	if extension.Manifest.Database == nil ||
		len(extension.Manifest.Database.Grants) == 0 ||
		extension.Manifest.Database.Schema != "sforum_commerce_workflow" {
		t.Fatalf("database = %#v", extension.Manifest.Database)
	}
	if len(extension.Manifest.Routes) < 4 {
		t.Fatalf("routes = %d", len(extension.Manifest.Routes))
	}
	modes := map[string]bool{}
	actions := map[string]bool{}
	for _, route := range extension.Manifest.Routes {
		modes[route.Mode] = true
		actions[route.Action] = true
	}
	for _, want := range []string{"http", "sse"} {
		if !modes[want] {
			t.Fatalf("missing route mode %s in %#v", want, modes)
		}
	}
	for _, want := range []string{"add", "alias", "redirect"} {
		if !actions[want] {
			t.Fatalf("missing route action %s in %#v", want, actions)
		}
	}
	if len(extension.Manifest.Hooks) < 2 || len(extension.Manifest.Jobs) < 1 ||
		len(extension.Manifest.Cache) < 1 || len(extension.Manifest.Services) < 1 ||
		len(extension.Manifest.Components) < 1 || len(extension.Manifest.OpenAPI) < 1 {
		t.Fatalf("commerce surfaces incomplete: hooks=%d jobs=%d cache=%d services=%d components=%d openapi=%d",
			len(extension.Manifest.Hooks), len(extension.Manifest.Jobs), len(extension.Manifest.Cache),
			len(extension.Manifest.Services), len(extension.Manifest.Components), len(extension.Manifest.OpenAPI))
	}
}

// commerceDatabaseLeaseRegistry is a process-local mock for own_schema Start leases.
type commerceDatabaseLeaseRegistry struct {
	mu    sync.Mutex
	lease extensionsruntime.ExtensionDatabaseRuntimeLeaseSnapshot
}

func newCommerceDatabaseLeaseRegistry() *commerceDatabaseLeaseRegistry {
	return &commerceDatabaseLeaseRegistry{}
}

func (r *commerceDatabaseLeaseRegistry) IssueRuntimeLease(
	_ context.Context,
	request extensionsruntime.ExtensionDatabaseRuntimeLeaseIssue,
) (extensionsruntime.ExtensionDatabaseRuntimeCredential, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	r.lease = extensionsruntime.ExtensionDatabaseRuntimeLeaseSnapshot{
		ID: 1, LeaseID: strings.Repeat("c", 64), GrantID: 2,
		Artifact: request.Artifact, RuntimeInstanceID: request.RuntimeInstanceID,
		RoleName: "sforum_ext_l_commerce_ref", Status: extensionsruntime.ExtensionDatabaseLeaseActive,
		IssuerKind: extensionsruntime.ExtensionDatabaseLeaseIssuerHost, IssueAuditEventID: 3,
		IssuedAt: now, LastHeartbeatAt: now, ExpiresAt: now.Add(2 * time.Minute), Revision: 1,
	}
	return extensionsruntime.ExtensionDatabaseRuntimeCredential{
		LeaseID: r.lease.LeaseID, GrantID: r.lease.GrantID, Artifact: request.Artifact,
		RuntimeInstanceID: request.RuntimeInstanceID,
		Powers:            []string{extensionmanifest.DatabaseGrantOwnSchema},
		SchemaName:        "sforum_commerce_workflow",
		OwnerRoleName:     "sforum_ext_o_commerce_ref",
		RoleName:          r.lease.RoleName,
		DatabaseName:      "sforum",
		SearchPath:        "sforum_commerce_workflow, pg_catalog",
		ConnectionURL:     "postgres://lease_role:lease_secret@127.0.0.1:5432/sforum?sslmode=disable",
		Password:          strings.Repeat("A", 43),
		ExpiresAt:         r.lease.ExpiresAt,
		Revision:          r.lease.Revision,
	}, nil
}

func (r *commerceDatabaseLeaseRegistry) HeartbeatRuntimeLease(
	_ context.Context,
	ref extensionsruntime.ExtensionDatabaseRuntimeLeaseRef,
	expectedRevision int64,
) (extensionsruntime.ExtensionDatabaseRuntimeLeaseSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lease.LeaseID != ref.LeaseID || r.lease.Revision != expectedRevision {
		return extensionsruntime.ExtensionDatabaseRuntimeLeaseSnapshot{}, extensionsruntime.ErrExtensionDatabaseRuntimeLeaseConflict
	}
	r.lease.Revision++
	r.lease.LastHeartbeatAt = time.Now().UTC()
	r.lease.ExpiresAt = r.lease.LastHeartbeatAt.Add(2 * time.Minute)
	return r.lease, nil
}

func (r *commerceDatabaseLeaseRegistry) BeginRuntimeLeaseDrain(
	_ context.Context,
	ref extensionsruntime.ExtensionDatabaseRuntimeLeaseRef,
	expectedRevision int64,
) (extensionsruntime.ExtensionDatabaseRuntimeLeaseSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lease.LeaseID != ref.LeaseID || r.lease.Revision != expectedRevision {
		return extensionsruntime.ExtensionDatabaseRuntimeLeaseSnapshot{}, extensionsruntime.ErrExtensionDatabaseRuntimeLeaseConflict
	}
	r.lease.Status = extensionsruntime.ExtensionDatabaseLeaseDraining
	r.lease.Revision++
	now := time.Now().UTC()
	r.lease.DrainingAt = &now
	return r.lease, nil
}

func (r *commerceDatabaseLeaseRegistry) RevokeRuntimeLease(
	_ context.Context,
	ref extensionsruntime.ExtensionDatabaseRuntimeLeaseRef,
	_ extensionsruntime.ExtensionDatabaseLeaseAuthority,
) (extensionsruntime.ExtensionDatabaseRuntimeLeaseSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lease.LeaseID != "" && r.lease.LeaseID != ref.LeaseID {
		return extensionsruntime.ExtensionDatabaseRuntimeLeaseSnapshot{}, extensionsruntime.ErrExtensionDatabaseRuntimeLeaseConflict
	}
	r.lease.Status = extensionsruntime.ExtensionDatabaseLeaseRevoked
	r.lease.Revision++
	return r.lease, nil
}

func (r *commerceDatabaseLeaseRegistry) InspectRuntimeLease(
	_ context.Context,
	ref extensionsruntime.ExtensionDatabaseRuntimeLeaseRef,
) (extensionsruntime.ExtensionDatabaseRuntimeLeaseSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lease.LeaseID != ref.LeaseID {
		return extensionsruntime.ExtensionDatabaseRuntimeLeaseSnapshot{}, extensionsruntime.ErrExtensionDatabaseRuntimeLeaseNotFound
	}
	return r.lease, nil
}

func buildReferenceCommerceExtension(t *testing.T) extensions.Extension {
	t.Helper()
	return buildReferenceFixtureExtension(t, "sforum-commerce-workflow", "sforum.commerce-workflow", 801, map[string]string{
		"__FRONTEND_DIGEST__": "frontend/order-card.mjs",
		"__OPENAPI_DIGEST__":  "openapi/routes.yaml",
	})
}

func buildReferenceCommerceExtenderExtension(t *testing.T) extensions.Extension {
	t.Helper()
	return buildReferenceFixtureExtension(t, "sforum-commerce-workflow-ext", "sforum.commerce-workflow-ext", 802, nil)
}

// buildReferenceFixtureExtension copies a fixtures/plugins package, builds the
// Protocol V2 backend, fills digest tokens, and loads the exact Manifest V3 package.
func buildReferenceFixtureExtension(
	t *testing.T,
	fixtureDir string,
	packageName string,
	versionID int64,
	fileDigests map[string]string,
) extensions.Extension {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	fixtureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../extensions/fixtures/plugins", fixtureDir))
	packageRoot := filepath.Join(t.TempDir(), packageName)
	if err := os.CopyFS(packageRoot, os.DirFS(fixtureRoot)); err != nil {
		t.Fatalf("copy %s: %v", fixtureDir, err)
	}
	repositoryRoot := filepath.Clean(filepath.Join(fixtureRoot, "../../../.."))
	goModPath := filepath.Join(packageRoot, "backend", "go.mod")
	goMod, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	goMod = []byte(strings.ReplaceAll(string(goMod), "../../../../../apps/api", filepath.Join(repositoryRoot, "apps/api")))
	if err := os.WriteFile(goModPath, goMod, 0o600); err != nil {
		t.Fatal(err)
	}
	binaryPath := filepath.Join(packageRoot, "backend", "plugin")
	build := exec.Command("go", "build", "-mod=mod", "-trimpath", "-buildvcs=false", "-o", binaryPath, ".")
	build.Dir = filepath.Join(packageRoot, "backend")
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", fixtureDir, err, output)
	}
	templateBody, err := os.ReadFile(filepath.Join(packageRoot, "sforum.extension.json.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	manifestBody := strings.ReplaceAll(string(templateBody), "__BACKEND_DIGEST__", fileSHA256(t, binaryPath))
	for token, rel := range fileDigests {
		manifestBody = strings.ReplaceAll(manifestBody, token, fileSHA256(t, filepath.Join(packageRoot, rel)))
	}
	if strings.Contains(manifestBody, "__") {
		t.Fatalf("%s manifest still contains digest tokens", fixtureDir)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, extensionmanifest.ManifestFileName), []byte(manifestBody), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := extensionmanifest.LoadPackage(packageRoot)
	if err != nil {
		t.Fatalf("load %s package: %v", fixtureDir, err)
	}
	packageDigest, err := extensionpackage.DigestTree(packageRoot)
	if err != nil {
		t.Fatal(err)
	}
	return extensions.Extension{
		ID: manifest.ID, Name: manifest.Name, Version: manifest.Version, Type: extensions.TypePlugin,
		Status: extensions.StatusEnabled, Source: extensions.SourceUploaded,
		PackagePath: packageRoot, PackageDigest: packageDigest, Manifest: manifest, ActiveVersionID: versionID,
		CapabilityGrants: []extensions.CapabilityGrant{{Key: capabilities.HostAPI, Risk: capabilities.RiskLow}},
	}
}


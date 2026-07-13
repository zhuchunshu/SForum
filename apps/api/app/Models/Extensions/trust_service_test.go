package extensions

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	audit "github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

func TestExecutableTrustChallengeBindsExactImpactAndIsOneUse(t *testing.T) {
	extension := exactTrustExtension(t, "demo.trusted")
	extensions := &fakeExtensionStore{items: map[string]Extension{extension.ID: extension}}
	now := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	trustStore := &memoryExecutableTrustStore{now: func() time.Time { return now }}
	service := NewExecutableTrustService(extensions, trustStore)
	service.now = func() time.Time { return now }
	service.random = bytes.NewReader(bytes.Repeat([]byte{0x2a}, 32))

	challenge, err := service.Challenge(context.Background(), extensionManager(), extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	if challenge.Token == "" || challenge.Impact.Digest == "" || challenge.Impact.PackageDigest != extension.PackageDigest {
		t.Fatalf("incomplete challenge: %#v", challenge)
	}
	if challenge.Impact.ArtifactDigests["backend"] == "" || challenge.Impact.ArtifactDigests["migration:migrations/001.sql"] == "" {
		t.Fatalf("executable digests missing: %#v", challenge.Impact.ArtifactDigests)
	}
	if trustStore.challenge.TokenHash == "" || trustStore.challenge.TokenHash == challenge.Token {
		t.Fatal("store must persist only the token hash")
	}
	if err := service.ConfirmEnable(context.Background(), extensionManager(), extension, ""); !errors.Is(err, ErrTrustChallengeRequired) {
		t.Fatalf("missing token: %v", err)
	}

	otherSuperAdmin := extensionManager()
	otherSuperAdmin.ID = 99
	if err := service.ConfirmEnable(context.Background(), otherSuperAdmin, extension, challenge.Token); !errors.Is(err, ErrTrustChallengeInvalid) {
		t.Fatalf("wrong actor: %v", err)
	}
	if err := service.ConfirmEnable(context.Background(), extensionManager(), extension, challenge.Token); err != nil {
		t.Fatal(err)
	}
	if err := trustStore.RevokeAll(context.Background(), extension.ID, extensionManager().ID, "test"); err != nil {
		t.Fatal(err)
	}
	if err := service.ConfirmEnable(context.Background(), extensionManager(), extension, challenge.Token); !errors.Is(err, ErrTrustChallengeReplayed) {
		t.Fatalf("replayed token: %v", err)
	}
}

func TestExecutableTrustAuditsChallengeDeniedGrantAndRevoke(t *testing.T) {
	extension := exactTrustExtension(t, "demo.audit")
	store := &fakeExtensionStore{items: map[string]Extension{extension.ID: extension}}
	auditor := &recordingAuditor{}
	service := NewExecutableTrustService(store, &memoryExecutableTrustStore{}).WithAuditor(auditor)
	actor := extensionManager()

	challenge, err := service.Challenge(context.Background(), actor, extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	other := actor
	other.ID++
	if err := service.ConfirmEnable(context.Background(), other, extension, challenge.Token); !errors.Is(err, ErrTrustChallengeInvalid) {
		t.Fatalf("wrong actor must be denied, got %v", err)
	}
	if err := service.ConfirmEnable(context.Background(), actor, extension, challenge.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Revoke(context.Background(), actor, extension.ID); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{
		audit.ActionExtensionTrustChallenge,
		audit.ActionExtensionTrustDenied,
		audit.ActionExtensionTrustGrant,
		audit.ActionExtensionTrustRevoke,
	} {
		if !auditor.hasAction(action) {
			t.Fatalf("missing audit action %s", action)
		}
	}
}

func TestExecutableTrustChallengeRejectsExpiredAndChangedImpact(t *testing.T) {
	extension := exactTrustExtension(t, "demo.expiring")
	extensions := &fakeExtensionStore{items: map[string]Extension{extension.ID: extension}}
	now := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	trustStore := &memoryExecutableTrustStore{now: func() time.Time { return now }}
	service := NewExecutableTrustService(extensions, trustStore)
	service.now = func() time.Time { return now }

	challenge, err := service.Challenge(context.Background(), extensionManager(), extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(DefaultTrustChallengeTTL + time.Second)
	if err := service.ConfirmEnable(context.Background(), extensionManager(), extension, challenge.Token); !errors.Is(err, ErrTrustChallengeExpired) {
		t.Fatalf("expired token: %v", err)
	}

	now = now.Add(time.Second)
	challenge, err = service.Challenge(context.Background(), extensionManager(), extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	changed := extension
	changed.Manifest.Routes = append(changed.Manifest.Routes, ManifestRoute{Path: "/changed", Methods: []string{"POST"}, Access: RouteAccessLogin})
	if err := service.ConfirmEnable(context.Background(), extensionManager(), changed, challenge.Token); !errors.Is(err, ErrTrustChallengeStale) {
		t.Fatalf("changed impact: %v", err)
	}
}

func TestExecutableTrustChallengeRejectsEveryExecutableDigestChange(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, *Extension)
		change  func(*testing.T, *Extension)
	}{
		{
			name: "backend bytes and package digest",
			change: func(t *testing.T, extension *Extension) {
				writeTrustFile(t, extension, "backend/plugin", "changed-plugin-binary", 0o755)
				refreshTrustPackageIdentity(t, extension)
			},
		},
		{
			name: "migration bytes",
			change: func(t *testing.T, extension *Extension) {
				writeTrustFile(t, extension, "migrations/001.sql", "SELECT 2;", 0o600)
				refreshTrustPackageIdentity(t, extension)
			},
		},
		{
			name: "migration declaration",
			change: func(t *testing.T, extension *Extension) {
				writeTrustFile(t, extension, "migrations/002.sql", "SELECT 2;", 0o600)
				extension.Manifest.Migrations = append(extension.Manifest.Migrations, ManifestMigration{Path: "migrations/002.sql"})
				refreshTrustPackageIdentity(t, extension)
			},
		},
		{
			name: "admin frontend bytes",
			prepare: func(t *testing.T, extension *Extension) {
				addTrustAdminFrontend(t, extension)
			},
			change: func(t *testing.T, extension *Extension) {
				writeTrustFile(t, extension, "frontend/admin/dist/settings.mjs", "export function mount() { return () => {} }\n", 0o600)
				refreshTrustPackageIdentity(t, extension)
			},
		},
		{
			name: "admin frontend contract",
			prepare: func(t *testing.T, extension *Extension) {
				addTrustAdminFrontend(t, extension)
			},
			change: func(t *testing.T, extension *Extension) {
				extension.Manifest.SettingsDocument.UI.Component.APIVersion = 2
				refreshTrustPackageIdentity(t, extension)
			},
		},
		{
			name: "requested network authority",
			change: func(t *testing.T, extension *Extension) {
				extension.Manifest.Capabilities = nil
				refreshTrustPackageIdentity(t, extension)
			},
		},
		{
			name: "permission declaration",
			change: func(t *testing.T, extension *Extension) {
				extension.Manifest.Permissions = append(extension.Manifest.Permissions, "demo.use")
				refreshTrustPackageIdentity(t, extension)
			},
		},
		{
			name: "required feature declaration",
			change: func(t *testing.T, extension *Extension) {
				extension.Manifest.RequiresFeatures = append(extension.Manifest.RequiresFeatures, "features.demo")
				refreshTrustPackageIdentity(t, extension)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			extension := exactTrustExtension(t, "demo.changed")
			if test.prepare != nil {
				test.prepare(t, &extension)
			}
			store := &fakeExtensionStore{items: map[string]Extension{extension.ID: extension}}
			trustStore := &memoryExecutableTrustStore{}
			service := NewExecutableTrustService(store, trustStore)
			challenge, err := service.Challenge(context.Background(), extensionManager(), extension.ID)
			if err != nil {
				t.Fatal(err)
			}

			test.change(t, &extension)
			if err := service.ConfirmEnable(context.Background(), extensionManager(), extension, challenge.Token); !errors.Is(err, ErrTrustChallengeStale) {
				t.Fatalf("changed exact artifact must be stale, got %v", err)
			}
		})
	}
}

func TestCanonicalTrustImpactDigestBindsAuthorityContractsAndDependencies(t *testing.T) {
	base, err := buildTrustImpact(exactTrustExtension(t, "demo.contracts"), TrustActionEnable)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		change func(*TrustImpact)
	}{
		{name: "raw request authority", change: func(impact *TrustImpact) { impact.RequestedAuthority.RawRequest = true }},
		{name: "raw core database authority", change: func(impact *TrustImpact) { impact.RequestedAuthority.RawCoreDatabase = true }},
		{name: "host contract", change: func(impact *TrustImpact) { impact.Contracts.HostAPI = "sforum.host/v2" }},
		{name: "frontend contract", change: func(impact *TrustImpact) { impact.Contracts.FrontendAPI = "sforum.component/v2" }},
		{name: "dependency", change: func(impact *TrustImpact) {
			impact.Dependencies = []ManifestDependency{{ID: "demo.parent", Version: "^2.0.0", Kind: "required"}}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed := base
			test.change(&changed)
			digest, err := canonicalTrustImpactDigest(changed)
			if err != nil {
				t.Fatal(err)
			}
			if digest == base.Digest {
				t.Fatalf("%s change did not invalidate impact digest", test.name)
			}
		})
	}
}

func TestManifestV3TrustImpactIncludesEveryDeclarationAndExecutableDigest(t *testing.T) {
	extension := completeV3TrustExtension(t, "demo.v3-trust")
	impact, err := buildTrustImpact(extension, TrustActionEnable)
	if err != nil {
		t.Fatal(err)
	}
	if impact.SchemaVersion != TrustImpactSchemaV2 || impact.ManifestContract != "sforum.manifest@3" || impact.Contracts.HostAPI != "sforum.host-api@2" {
		t.Fatalf("incorrect V3 contracts: %#v", impact)
	}
	if impact.ArtifactDigests["guard:demo.v3-trust.guard.raw"] == "" || impact.ArtifactDigests["l2:demo.v3-trust.file.l2"] == "" {
		t.Fatalf("custom guard or L2 digest missing: %#v", impact.ArtifactDigests)
	}
	if !impact.RequestedAuthority.RawRequest || !impact.RequestedAuthority.RawCoreDatabase || !RequiresExecutableTrust(extension) {
		t.Fatalf("high-risk authority was not derived from V3 declarations: %#v", impact.RequestedAuthority)
	}
	if len(impact.GuardDeclarations) != 1 || len(impact.MigrationDeclarations) != 1 || len(impact.Schedules) != 1 ||
		len(impact.RegistryComponents) != 1 || len(impact.Templates) != 1 || len(impact.Assets) != 1 || len(impact.Content) != 1 ||
		impact.Database == nil || len(impact.Cache) != 1 || len(impact.Services) != 1 || len(impact.Commands) != 1 ||
		len(impact.AdminSurfaces) != 1 || len(impact.Queries) != 1 || impact.Identity == nil || len(impact.PermissionDefinitions) != 1 ||
		len(impact.Media) != 1 || len(impact.Navigation) != 1 || len(impact.Regions) != 1 || len(impact.Dependencies) != 1 ||
		impact.Lifecycle == nil || len(impact.OpenAPI) != 1 || len(impact.PackageFiles) != 7 {
		t.Fatalf("incomplete V3 trust impact: %#v", impact)
	}
}

func TestManifestV3EveryDeclarationInvalidatesCanonicalTrustImpact(t *testing.T) {
	base, err := buildTrustImpact(completeV3TrustExtension(t, "demo.v3-digest"), TrustActionEnable)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		change func(*TrustImpact)
	}{
		{name: "manifest contract", change: func(impact *TrustImpact) { impact.ManifestContract = "sforum.manifest@4" }},
		{name: "backend", change: func(impact *TrustImpact) { impact.Backend.HostAPIVersion = "sforum.host-api@3" }},
		{name: "guard declaration", change: func(impact *TrustImpact) { impact.GuardDeclarations = nil }},
		{name: "migration declaration", change: func(impact *TrustImpact) { impact.MigrationDeclarations = nil }},
		{name: "schedule", change: func(impact *TrustImpact) { impact.Schedules = nil }},
		{name: "registry component", change: func(impact *TrustImpact) { impact.RegistryComponents = nil }},
		{name: "template", change: func(impact *TrustImpact) { impact.Templates = nil }},
		{name: "asset", change: func(impact *TrustImpact) { impact.Assets = nil }},
		{name: "content", change: func(impact *TrustImpact) { impact.Content = nil }},
		{name: "database", change: func(impact *TrustImpact) { impact.Database = nil }},
		{name: "cache", change: func(impact *TrustImpact) { impact.Cache = nil }},
		{name: "service", change: func(impact *TrustImpact) { impact.Services = nil }},
		{name: "command", change: func(impact *TrustImpact) { impact.Commands = nil }},
		{name: "admin surface", change: func(impact *TrustImpact) { impact.AdminSurfaces = nil }},
		{name: "query", change: func(impact *TrustImpact) { impact.Queries = nil }},
		{name: "identity", change: func(impact *TrustImpact) { impact.Identity = nil }},
		{name: "permission definition", change: func(impact *TrustImpact) { impact.PermissionDefinitions = nil }},
		{name: "media", change: func(impact *TrustImpact) { impact.Media = nil }},
		{name: "navigation", change: func(impact *TrustImpact) { impact.Navigation = nil }},
		{name: "region", change: func(impact *TrustImpact) { impact.Regions = nil }},
		{name: "dependency", change: func(impact *TrustImpact) { impact.Dependencies = nil }},
		{name: "lifecycle", change: func(impact *TrustImpact) { impact.Lifecycle = nil }},
		{name: "openapi", change: func(impact *TrustImpact) { impact.OpenAPI = nil }},
		{name: "package file", change: func(impact *TrustImpact) { impact.PackageFiles = nil }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed := base
			test.change(&changed)
			digest, err := canonicalTrustImpactDigest(changed)
			if err != nil {
				t.Fatal(err)
			}
			if digest == base.Digest {
				t.Fatalf("%s change did not invalidate impact digest", test.name)
			}
		})
	}
}

func TestManifestV3HighRiskDeclarationsRequireExactArtifactTrust(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Manifest)
	}{
		{name: "custom guard", change: func(manifest *Manifest) { manifest.Guards = []ManifestGuard{{Kind: "custom"}} }},
		{name: "raw route", change: func(manifest *Manifest) { manifest.Routes = []ManifestRoute{{Guard: extensionmanifest.GuardCoreRaw}} }},
		{name: "raw database", change: func(manifest *Manifest) { manifest.Database = &ManifestDatabase{Authority: "raw_core"} }},
		{name: "kernel database", change: func(manifest *Manifest) { manifest.Database = &ManifestDatabase{Authority: "kernel"} }},
		{name: "L2 component", change: func(manifest *Manifest) { manifest.Components = []ManifestComponent{{L2Component: "demo.file"}} }},
		{name: "lifecycle execute", change: func(manifest *Manifest) {
			manifest.Lifecycle = &ManifestLifecycle{Enable: &extensionmanifest.ManifestLifecycleOperation{Execute: "lifecycle.enable"}}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			extension := Extension{Source: SourceUploaded}
			test.change(&extension.Manifest)
			if !RequiresExecutableTrust(extension) {
				t.Fatal("high-risk V3 declaration did not require exact-artifact trust")
			}
		})
	}
}

func TestManifestV3TrustRejectsChangedL2BytesAgainstDeclaration(t *testing.T) {
	extension := completeV3TrustExtension(t, "demo.v3-l2-change")
	writeTrustFile(t, &extension, "frontend/card.mjs", "export function mount() { throw new Error('changed') }\n", 0o600)
	refreshTrustPackageIdentity(t, &extension)
	if _, err := buildTrustImpact(extension, TrustActionEnable); !errors.Is(err, ErrFrontendPackageChanged) {
		t.Fatalf("changed L2 bytes must fail exact-artifact impact generation, got %v", err)
	}
}

func TestServiceV3TrustAllowsAlreadyGrantedDelegatedEnable(t *testing.T) {
	extension := exactTrustExtension(t, "demo.enable")
	store := &fakeExtensionStore{items: map[string]Extension{extension.ID: extension}}
	trustStore := &memoryExecutableTrustStore{now: func() time.Time { return time.Now().UTC() }}
	trust := NewExecutableTrustService(store, trustStore)
	service := NewServiceWithOptions(store, t.TempDir(), "", &fakeRuntimeManager{}, WithExecutableTrust(trust, true))

	if _, err := service.Enable(context.Background(), extensionManager(), extension.ID, EnableInput{}); !errors.Is(err, ErrTrustChallengeRequired) {
		t.Fatalf("enable without challenge: %v", err)
	}
	challenge, err := service.IssueExecutableTrustChallenge(context.Background(), extensionManager(), extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Enable(context.Background(), extensionManager(), extension.ID, EnableInput{ConfirmationToken: challenge.Token}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Disable(context.Background(), techAdminPluginManager(), extension.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Enable(context.Background(), techAdminPluginManager(), extension.ID, EnableInput{}); err != nil {
		t.Fatalf("delegated manager should operate an already trusted digest: %v", err)
	}
}

func TestV3StaticInstallByDelegatedManagerDoesNotExecutePackage(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{}}
	runtime := &countingRuntimeManager{}
	trust := NewExecutableTrustService(store, &memoryExecutableTrustStore{})
	service := NewServiceWithOptions(store, t.TempDir(), "", runtime, WithExecutableTrust(trust, true))
	archive := extensionArchive(t, validManifest("delegated.backend", TypePlugin),
		zipFile{name: "backend/plugin", body: "binary", mode: 0o755},
		zipFile{name: "migrations/001_init.sql", body: "SELECT 1", mode: 0o644},
	)

	result, err := service.InstallOrUpgradeArchive(context.Background(), techAdminPluginManager(), ArchiveInput{FileName: "plugin.zip", Data: archive})
	if err != nil {
		t.Fatal(err)
	}
	if result.Extension.Status != StatusInstalled || runtime.checks != 0 || runtime.starts != 0 {
		t.Fatalf("static install executed package code: result=%#v checks=%d starts=%d", result, runtime.checks, runtime.starts)
	}
	status, err := service.ExecutableTrustStatus(context.Background(), techAdminPluginManager(), result.Extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !status.TrustRequired || status.Trusted || len(status.Impact.Binaries) != 1 || len(status.Impact.Migrations) != 1 ||
		len(status.Impact.Routes) != 1 || len(status.Impact.Jobs) != 1 || status.Impact.ArtifactDigests["backend"] == "" ||
		status.Impact.ArtifactDigests["migration:migrations/001_init.sql"] == "" {
		t.Fatalf("delegated manager received incomplete static impact preview: %#v", status)
	}
	if runtime.checks != 0 || runtime.starts != 0 {
		t.Fatalf("impact preview executed package code: checks=%d starts=%d", runtime.checks, runtime.starts)
	}
}

func completeV3TrustExtension(t *testing.T, id string) Extension {
	t.Helper()
	item := exactTrustExtension(t, id)
	for relative, body := range map[string]string{
		"backend/guard":       "guard-binary",
		"frontend/card.mjs":   "export function mount() {}\n",
		"frontend/card.css":   ".card {}\n",
		"templates/card.html": "<article></article>\n",
		"openapi/routes.yaml": "openapi: 3.1.0\n",
	} {
		writeTrustFile(t, &item, relative, body, 0o600)
	}
	mustDigest := func(relative string) string {
		digest, err := digestInstalledFile(item, relative)
		if err != nil {
			t.Fatal(err)
		}
		return digest
	}
	backendDigest := mustDigest("backend/plugin")
	guardDigest := mustDigest("backend/guard")
	migrationDigest := mustDigest("migrations/001.sql")
	l2Digest := mustDigest("frontend/card.mjs")
	assetDigest := mustDigest("frontend/card.css")
	templateDigest := mustDigest("templates/card.html")
	openAPIDigest := mustDigest("openapi/routes.yaml")

	item.Manifest.ManifestVersion = extensionmanifest.ManifestVersionV3
	item.Manifest.Backend = ManifestBackend{
		Entry: "backend/plugin", RPC: "hashicorp-go-plugin", ProtocolVersion: 2,
		Digest: backendDigest, HostAPIVersion: "sforum.host-api@2",
	}
	item.Manifest.Permissions = []string{id + ".manage"}
	item.Manifest.Migrations = []ManifestMigration{{
		ID: id + ".migration.initial", ContractVersion: id + ".migration.initial@1",
		Path: "migrations/001.sql", Digest: migrationDigest, Transaction: "required",
	}}
	item.Manifest.Guards = []ManifestGuard{{
		ID: id + ".guard.raw", ContractVersion: id + ".guard.raw@1", Kind: "raw_request",
		Entry: "backend/guard", Digest: guardDigest, Permissions: []string{id + ".manage"},
	}}
	item.Manifest.Routes = []ManifestRoute{{
		ID: id + ".route.run", ContractVersion: id + ".route.run@1", Action: "add",
		Path: "/api/demo/run", Methods: []string{"POST"}, Guard: extensionmanifest.GuardCoreRaw,
		Fallback: "closed", Mode: "http", Handler: "route.run", RequestSchema: id + ".route.run.request@1",
		ResponseSchema: id + ".route.run.response@1",
	}}
	item.Manifest.Hooks = []ManifestHook{{
		ID: id + ".hook.before", ContractVersion: id + ".hook.before@1", Name: "topic.before_create",
		Kind: "filter", Handler: "hook.before", InputSchema: id + ".hook.before.input@1", ResultSchema: id + ".hook.before.result@1",
	}}
	item.Manifest.Events = []ManifestEvent{{
		ID: id + ".event.created", ContractVersion: id + ".event.created@1", Name: "topic.created",
		Kind: "observe", Handler: "event.created", InputSchema: id + ".event.created.input@1",
	}}
	item.Manifest.Jobs = []ManifestJob{{
		ID: id + ".job.refresh", ContractVersion: id + ".job.refresh@1", Name: id + ".refresh",
		Handler: "job.refresh", PayloadSchema: id + ".job.refresh.payload@1", RetryPolicy: "bounded",
	}}
	item.Manifest.Schedules = []ManifestSchedule{{
		ID: id + ".schedule.refresh", ContractVersion: id + ".schedule.refresh@1",
		JobID: id + ".job.refresh", Cron: "0 * * * *", Timezone: "UTC",
	}}
	item.Manifest.Providers = []ManifestProvider{{
		ID: id + ".provider.search", ContractVersion: id + ".provider.search@1", Slot: id + ".search",
		Label: "Search", Handler: "provider.search",
	}}
	item.Manifest.Templates = []ManifestTemplate{{
		ID: id + ".template.card", ContractVersion: id + ".template.card@1", Action: "add",
		Path: "templates/card.html", Digest: templateDigest, ViewModelSchema: id + ".template.card.model@1",
	}}
	item.Manifest.Assets = []ManifestAsset{{
		Handle: id + ".asset.card", ContractVersion: id + ".asset.card@1", Type: "style",
		Path: "frontend/card.css", Digest: assetDigest, Loading: "defer",
	}}
	item.Manifest.Components = []ManifestComponent{{
		ID: id + ".component.card", ContractVersion: id + ".component.card@1", Action: "add",
		L2Component: id + ".file.l2", PropsSchema: id + ".component.card.props@1",
		ResultSchema: id + ".component.card.result@1",
	}}
	item.Manifest.Content = []ManifestContent{{
		ID: id + ".content.card", ContractVersion: id + ".content.card@1", Kind: "block",
		Handler: "content.card", Schema: id + ".content.card.schema@1",
	}}
	item.Manifest.Database = &ManifestDatabase{
		ContractVersion: id + ".database@1", Authority: "raw_core", CoreCompatibility: ">=1.0.0 <2.0.0",
		Backup:    extensionmanifest.ManifestBackupPolicy{Required: true, Strategy: "operator_snapshot"},
		Retention: extensionmanifest.ManifestRetention{OnDisable: "retain", OnUninstall: "retain"},
	}
	item.Manifest.Cache = []ManifestCache{{
		ID: id + ".cache.results", ContractVersion: id + ".cache.results@1", Namespace: id + ".results", Policy: "actor",
	}}
	item.Manifest.Services = []ManifestService{{
		ID: id + ".service.lookup", ContractVersion: id + ".service.lookup@1", Action: "add", Handler: "service.lookup",
		RequestSchema: id + ".service.lookup.request@1", ResponseSchema: id + ".service.lookup.response@1",
	}}
	item.Manifest.Commands = []ManifestCommand{{
		ID: id + ".command.write", ContractVersion: id + ".command.write@1", Handler: "command.write",
		Permission: id + ".manage", InputSchema: id + ".command.write.input@1", ResultSchema: id + ".command.write.result@1",
	}}
	item.Manifest.AdminSurfaces = []ManifestAdminSurface{{
		ID: id + ".admin.notice", ContractVersion: id + ".admin.notice@1", Kind: "notice", Action: "add",
		Label: "Demo", Handler: "admin.notice", Permission: id + ".manage",
	}}
	item.Manifest.Queries = []ManifestQuery{{
		ID: id + ".query.items", ContractVersion: id + ".query.items@1", Entity: id + ".item",
		PlanVersion: id + ".query.items.plan@1", Fields: []string{"id"}, Pagination: "cursor",
		ResultSchema: id + ".query.items.result@1", PermissionPolicy: id + ".manage",
	}}
	item.Manifest.Identity = &ManifestIdentity{ContractVersion: id + ".identity@1", SessionPolicy: "core.session.default"}
	item.Manifest.PermissionDefinitions = []ManifestPermissionDefinition{{
		Key: id + ".manage", ContractVersion: id + ".permission.manage@1", Label: "Manage demo",
		Description: "Manage demo.", AssignmentPolicy: "host",
	}}
	item.Manifest.Media = []ManifestMediaPipeline{{
		ID: id + ".media.image", ContractVersion: id + ".media.image@1", Action: "add",
		MIMEs: []string{"image/png"}, Handler: "media.image",
	}}
	item.Manifest.Navigation = []ManifestNavigation{{
		ID: id + ".navigation.link", ContractVersion: id + ".navigation.link@1", Kind: "item",
		Action: "add", Label: "Demo", Href: "/demo",
	}}
	item.Manifest.Regions = []ManifestRegion{{
		ID: id + ".region.sidebar", ContractVersion: id + ".region.sidebar@1", Action: "add",
		Kind: "widget", Label: "Sidebar",
	}}
	item.Manifest.Dependencies = []ManifestDependency{{ID: "demo.base", Version: "^1.0.0", Kind: "required"}}
	item.Manifest.Lifecycle = &ManifestLifecycle{
		ContractVersion: id + ".lifecycle@1",
		Enable: &extensionmanifest.ManifestLifecycleOperation{
			Plan: "lifecycle.enable.plan", Execute: "lifecycle.enable", ProgressSchema: id + ".lifecycle.progress@1",
			CheckpointSchema: id + ".lifecycle.checkpoint@1",
		},
	}
	item.Manifest.OpenAPI = []ManifestOpenAPIFragment{{
		ID: id + ".openapi.routes", ContractVersion: id + ".openapi.routes@1", Path: "openapi/routes.yaml",
		Digest: openAPIDigest, Namespace: id + ".api",
	}}
	item.Manifest.PackageFiles = []ManifestPackageFile{
		{ID: id + ".file.backend", Kind: "executable", Path: "backend/plugin", Digest: backendDigest},
		{ID: id + ".file.guard", Kind: "executable", Path: "backend/guard", Digest: guardDigest},
		{ID: id + ".file.migration", Kind: "migration", Path: "migrations/001.sql", Digest: migrationDigest},
		{ID: id + ".file.l2", Kind: "frontend", Path: "frontend/card.mjs", Digest: l2Digest},
		{ID: id + ".file.asset", Kind: "asset", Path: "frontend/card.css", Digest: assetDigest},
		{ID: id + ".file.template", Kind: "template", Path: "templates/card.html", Digest: templateDigest},
		{ID: id + ".file.openapi", Kind: "openapi", Path: "openapi/routes.yaml", Digest: openAPIDigest},
	}
	refreshTrustPackageIdentity(t, &item)
	return item
}

func exactTrustExtension(t *testing.T, id string) Extension {
	t.Helper()
	item := installedExtension(id, TypePlugin, ManifestBackend{Entry: "backend/plugin", RPC: "hashicorp-go-plugin", ProtocolVersion: 1})
	item.Source = SourceUploaded
	item.IsDeletable = true
	item.Manifest.Routes = []ManifestRoute{{Path: "/run", Methods: []string{"POST"}, Access: RouteAccessPermission, Permission: "topic.create"}}
	item.Manifest.Migrations = []ManifestMigration{{Path: "migrations/001.sql"}}
	item.Manifest.Capabilities = []string{"net.outbound"}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "backend"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "migrations"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "backend/plugin"), []byte("plugin-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "migrations/001.sql"), []byte("SELECT 1;"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeManifest(root, item.Manifest); err != nil {
		t.Fatal(err)
	}
	digest, err := extensionpackage.DigestTree(root)
	if err != nil {
		t.Fatal(err)
	}
	item.PackagePath = root
	item.PackageDigest = digest
	return item
}

func addTrustAdminFrontend(t *testing.T, extension *Extension) {
	t.Helper()
	component := &SettingsComponent{ID: "settings", APIVersion: 1, Entry: "frontend/admin/dist/settings.mjs"}
	extension.Manifest.Settings = []ManifestSetting{{Key: "title", Type: "text", Label: LocalizedText{Default: "Title"}}}
	extension.Manifest.SettingsDocument = SettingsDocument{
		SchemaVersion: 1,
		Explicit:      true,
		UI:            SettingsUI{Mode: "component", Layout: "form", Component: component},
		Fields:        extension.Manifest.Settings,
	}
	writeTrustFile(t, extension, component.Entry, "export function mount() {}\n", 0o600)
	refreshTrustPackageIdentity(t, extension)
}

func writeTrustFile(t *testing.T, extension *Extension, relative, body string, mode os.FileMode) {
	t.Helper()
	target := filepath.Join(extension.PackagePath, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

func refreshTrustPackageIdentity(t *testing.T, extension *Extension) {
	t.Helper()
	if err := writeManifest(extension.PackagePath, extension.Manifest); err != nil {
		t.Fatal(err)
	}
	if component := extension.Manifest.SettingsDocument.UI.Component; component != nil && component.Entry != "" {
		digest, err := ComputeAdminFrontendDigest(extension.Manifest, extension.PackagePath)
		if err != nil {
			t.Fatal(err)
		}
		extension.AdminFrontendDigest = digest
	}
	digest, err := extensionpackage.DigestTree(extension.PackagePath)
	if err != nil {
		t.Fatal(err)
	}
	extension.PackageDigest = digest
}

type memoryExecutableTrustStore struct {
	mu        sync.Mutex
	now       func() time.Time
	challenge TrustChallengeRecord
	consumed  bool
	grants    map[TrustIdentity]bool
}

func (s *memoryExecutableTrustStore) CreateChallenge(_ context.Context, input TrustChallengeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.challenge = input
	s.consumed = false
	return nil
}

func (s *memoryExecutableTrustStore) HasLiveGrant(_ context.Context, identity TrustIdentity) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.grants[identity], nil
}

func (s *memoryExecutableTrustStore) ConsumeChallenge(_ context.Context, input TrustConsumeInput) (TrustGrant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if input.TokenHash == "" || input.TokenHash != s.challenge.TokenHash || input.ActorUserID != s.challenge.ActorUserID {
		return TrustGrant{}, ErrTrustChallengeInvalid
	}
	if s.consumed {
		return TrustGrant{}, ErrTrustChallengeReplayed
	}
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now()
	}
	if !now.Before(s.challenge.ExpiresAt) {
		return TrustGrant{}, ErrTrustChallengeExpired
	}
	if input.Identity != s.challenge.Identity {
		return TrustGrant{}, ErrTrustChallengeStale
	}
	s.consumed = true
	if s.grants == nil {
		s.grants = map[TrustIdentity]bool{}
	}
	s.grants[input.Identity] = true
	return TrustGrant{ExtensionID: input.Identity.ExtensionID, ImpactDigest: input.Identity.ImpactDigest}, nil
}

func (s *memoryExecutableTrustStore) RevokeAll(_ context.Context, extensionID string, _ int64, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for identity := range s.grants {
		if identity.ExtensionID == extensionID {
			delete(s.grants, identity)
		}
	}
	return nil
}

type countingRuntimeManager struct {
	checks int
	starts int
}

func (m *countingRuntimeManager) Check(context.Context, Extension) error {
	m.checks++
	return nil
}

func (m *countingRuntimeManager) Start(context.Context, Extension) error {
	m.starts++
	return nil
}

func (*countingRuntimeManager) Stop(context.Context, Extension) error { return nil }
func (*countingRuntimeManager) Status(context.Context, Extension) RuntimeStatus {
	return RuntimeStatus{State: RuntimeStopped}
}
func (*countingRuntimeManager) EmitHook(context.Context, string, map[string]any) {}

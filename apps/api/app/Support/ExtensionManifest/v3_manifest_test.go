package extensionmanifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func TestManifestV3CompleteApplicationPlugin(t *testing.T) {
	manifest := completeV3Manifest()
	if err := Validate(manifest); err != nil {
		t.Fatalf("complete V3 manifest should validate: %v", err)
	}

	manifest.Routes[0].Methods = []string{"PROPFIND"}
	manifest.Routes[0].Guard = GuardCorePublic
	if err := Validate(manifest); err != nil {
		t.Fatalf("V3 must support arbitrary declared methods and public writes: %v", err)
	}
}

func TestManifestV3ThemePresentationContract(t *testing.T) {
	digest := strings.Repeat("b", 64)
	manifest := versionedTestManifest(ManifestVersionV3)
	manifest.ID = "demo.theme"
	manifest.Type = TypeTheme
	manifest.PackageFiles = []ManifestPackageFile{
		{ID: "demo.theme.file.template", Kind: "template", Path: "templates/home.html", Digest: digest},
	}
	manifest.Templates = []ManifestTemplate{{
		ID: "demo.theme.template.home", ContractVersion: "demo.theme.template.home@1",
		Action: "add", Path: "templates/home.html", Digest: digest,
		ViewModelSchema: "sforum.page.home@1", ThemeOverrideKey: "demo.theme.home",
	}}
	manifest.Components = []ManifestComponent{{
		ID: "demo.theme.component.home", ContractVersion: "demo.theme.component.home@1",
		Action: ComponentActionAdd, SSRTemplate: "demo.theme.template.home",
		PropsSchema: "sforum.page.home@1",
	}}
	if err := Validate(manifest); err != nil {
		t.Fatalf("V3 presentation-only theme should validate: %v", err)
	}
	manifest.Commands = []ManifestCommand{{ID: "demo.theme.command.evil"}}
	if err := Validate(manifest); err == nil {
		t.Fatal("theme business/runtime declarations must be rejected")
	}
}

func TestManifestV3RejectsUnsafeContracts(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Manifest)
	}{
		{name: "unknown guard", change: func(manifest *Manifest) { manifest.Routes[0].Guard = "missing.guard" }},
		{name: "inherited guard without target", change: func(manifest *Manifest) { manifest.Routes[0].Guard = GuardCoreInherit }},
		{name: "duplicate id", change: func(manifest *Manifest) {
			manifest.Components = append(manifest.Components, manifest.Components[0])
		}},
		{name: "missing contract", change: func(manifest *Manifest) { manifest.Commands[0].ContractVersion = "" }},
		{name: "reserved health path", change: func(manifest *Manifest) { manifest.Routes[0].Path = "/health" }},
		{name: "missing digest", change: func(manifest *Manifest) { manifest.Assets[0].Digest = "" }},
		{name: "unsafe package path", change: func(manifest *Manifest) { manifest.OpenAPI[0].Path = "../openapi.yaml" }},
		{name: "plugin self grants permission", change: func(manifest *Manifest) {
			manifest.PermissionDefinitions[0].AssignmentPolicy = "plugin"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := completeV3Manifest()
			test.change(&manifest)
			if err := Validate(manifest); err == nil {
				t.Fatal("invalid V3 declaration must be rejected")
			}
		})
	}
}

func TestLegacyManifestRejectsV3Declarations(t *testing.T) {
	manifest := versionedTestManifest(0)
	manifest.Dependencies = []ManifestDependency{{ID: "other.plugin", Version: "^1.0.0", Kind: "required"}}
	if err := Validate(manifest); err == nil {
		t.Fatal("V3 fields must not be accepted through an implicit V1 upgrade")
	}
}

func TestManifestV3LoadsEveryShardedDeclaration(t *testing.T) {
	manifest := completeV3Manifest()
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		t.Fatal(err)
	}
	fields := []string{
		"guards", "schedules", "components", "templates", "assets", "content",
		"database", "cache", "services", "commands", "adminSurfaces", "queries",
		"identity", "permissionDefinitions", "media", "navigation", "regions",
		"dependencies", "lifecycle", "openapi", "packageFiles",
	}
	files := FileMapFS{}
	includes := map[string]string{}
	for _, field := range fields {
		shard := "manifest/" + field + ".json"
		files[shard] = root[field]
		delete(root, field)
		includes[field] = shard
	}
	for _, file := range manifest.PackageFiles {
		files[file.Path] = v3FixtureBody()
	}
	root["includes"], err = json.Marshal(includes)
	if err != nil {
		t.Fatal(err)
	}
	files[ManifestFileName], err = json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadPackageFS(files)
	if err != nil {
		t.Fatalf("load V3 shards: %v", err)
	}
	if loaded.Database == nil || loaded.Identity == nil || loaded.Lifecycle == nil || len(loaded.Routes) != 1 || len(loaded.Components) != 1 || len(loaded.Dependencies) != 2 {
		t.Fatalf("incomplete sharded manifest: %#v", loaded)
	}
}

func TestManifestV3PackageFileDigestsAreExact(t *testing.T) {
	manifest := completeV3Manifest()
	files := FileMapFS{}
	for _, file := range manifest.PackageFiles {
		files[file.Path] = v3FixtureBody()
	}
	if err := ValidatePackageFiles(manifest, files); err != nil {
		t.Fatalf("exact package files should validate: %v", err)
	}
	files[manifest.PackageFiles[0].Path] = []byte("changed")
	if err := ValidatePackageFiles(manifest, files); err == nil {
		t.Fatal("changed package bytes must be rejected")
	}
	delete(files, manifest.PackageFiles[0].Path)
	if err := ValidatePackageFiles(manifest, files); err == nil {
		t.Fatal("missing package bytes must be rejected")
	}
}

func completeV3Manifest() Manifest {
	digest := v3FixtureDigest()
	manifest := versionedTestManifest(ManifestVersionV3)
	manifest.ID = "demo.v3"
	manifest.Permissions = []string{"demo.v3.manage"}
	manifest.Backend = ManifestBackend{
		Entry: "backend/plugin", RPC: "hashicorp-go-plugin", ProtocolVersion: 2,
		Digest: digest, HostAPIVersion: "sforum.host-api@2",
	}
	manifest.Migrations = []ManifestMigration{{
		ID: "demo.v3.migration.initial", ContractVersion: "demo.v3.migration.initial@1",
		Path: "migrations/001.sql", Digest: digest, Transaction: "required",
	}}
	manifest.Guards = []ManifestGuard{{
		ID: "demo.v3.guard.owner", ContractVersion: "demo.v3.guard.owner@1",
		Kind: "custom", Entry: "backend/guard", Digest: digest, Permissions: []string{"demo.v3.manage"},
	}}
	manifest.Routes = []ManifestRoute{{
		ID: "demo.v3.route.write", ContractVersion: "demo.v3.route.write@1",
		Action: RouteActionAdd, Path: "/api/v3/demo", Methods: []string{"POST"},
		Guard: "demo.v3.guard.owner", Mode: RouteModeHTTP, Handler: "route.write",
		RequestSchema: "demo.v3.route.write.request@1", ResponseSchema: "demo.v3.route.write.response@1",
		Fallback: "closed", TimeoutMS: 1000,
	}}
	manifest.Hooks = []ManifestHook{{
		ID: "demo.v3.hook.listener", ContractVersion: "demo.v3.hook.listener@1",
		Name: "demo.v3.hook.changed", Kind: "action", Handler: "hook.changed",
		InputSchema: "demo.v3.hook.changed.input@1",
	}}
	manifest.Events = []ManifestEvent{{
		ID: "demo.v3.event.listener", ContractVersion: "demo.v3.event.listener@1",
		Name: "demo.v3.event.changed", Kind: "observe", Handler: "event.changed",
		InputSchema: "demo.v3.event.changed.input@1", TimeoutMS: 1000,
	}}
	manifest.Jobs = []ManifestJob{{
		ID: "demo.v3.job.refresh", ContractVersion: "demo.v3.job.refresh@1",
		Name: "demo.v3.refresh", Handler: "job.refresh", PayloadSchema: "demo.v3.job.refresh.payload@1", RetryPolicy: "bounded",
	}}
	manifest.Schedules = []ManifestSchedule{{
		ID: "demo.v3.schedule.refresh", ContractVersion: "demo.v3.schedule.refresh@1",
		JobID: "demo.v3.job.refresh", Cron: "0 * * * *", Timezone: "UTC",
	}}
	manifest.Providers = []ManifestProvider{{
		ID: "demo.v3.provider.search", ContractVersion: "demo.v3.provider.search@1",
		Slot: "demo.v3.provider.slot", Label: "Demo", Handler: "provider.search", TimeoutMS: 1000,
	}}
	manifest.PackageFiles = []ManifestPackageFile{
		{ID: "demo.v3.file.backend", Kind: "executable", Path: "backend/plugin", Digest: digest},
		{ID: "demo.v3.file.guard", Kind: "executable", Path: "backend/guard", Digest: digest},
		{ID: "demo.v3.file.migration", Kind: "migration", Path: "migrations/001.sql", Digest: digest},
		{ID: "demo.v3.file.template", Kind: "template", Path: "templates/card.html", Digest: digest},
		{ID: "demo.v3.file.asset", Kind: "asset", Path: "frontend/card.css", Digest: digest},
		{ID: "demo.v3.file.frontend", Kind: "frontend", Path: "frontend/card.mjs", Digest: digest},
		{ID: "demo.v3.file.openapi", Kind: "openapi", Path: "openapi/routes.yaml", Digest: digest},
		{ID: "demo.v3.file.locale", Kind: "locale", Path: "locales/zh-CN.json", Digest: digest, Locale: "zh-CN"},
		{ID: "demo.v3.file.database.query", Kind: "database_operation", Path: "database/items-query.sql", Digest: digest},
		{ID: "demo.v3.file.database.execute", Kind: "database_operation", Path: "database/items-insert.sql", Digest: digest},
	}
	manifest.Templates = []ManifestTemplate{{
		ID: "demo.v3.template.card", ContractVersion: "demo.v3.template.card@1",
		Action: "add", Path: "templates/card.html", Digest: digest,
		ViewModelSchema: "demo.v3.template.card.model@1", ThemeOverrideKey: "demo.v3.card",
	}}
	manifest.Assets = []ManifestAsset{{
		Handle: "demo.v3.asset.card", ContractVersion: "demo.v3.asset.card@1",
		Type: "style", Path: "frontend/card.css", Digest: digest, Loading: "defer",
	}}
	manifest.Components = []ManifestComponent{{
		ID: "demo.v3.component.card", ContractVersion: "demo.v3.component.card@1",
		Action: ComponentActionAdd, SSRTemplate: "demo.v3.template.card", L2Component: "demo.v3.file.frontend",
		PropsSchema: "demo.v3.component.card.props@1", ResultSchema: "demo.v3.component.card.result@1", ThemeOverrideKey: "demo.v3.card",
	}}
	manifest.Content = []ManifestContent{{
		ID: "demo.v3.content.block", ContractVersion: "demo.v3.content.block@1",
		Kind: "block", Handler: "content.block", Schema: "demo.v3.content.block.schema@1", Renderer: "demo.v3.template.card",
	}}
	manifest.Database = &ManifestDatabase{
		ContractVersion: "demo.v3.database@1", Authority: "own_schema", Schema: "demo_v3", Role: "demo_v3",
		Backup:    ManifestBackupPolicy{Required: true, Strategy: "pg_dump"},
		Retention: ManifestRetention{OnDisable: "retain", OnUninstall: "export", Days: 30},
		Operations: []ManifestDatabaseOperation{
			{
				ID: "demo.v3.database.items.query", StatementVersion: "1", Kind: "query",
				Path: "database/items-query.sql", Digest: digest,
				Parameters: []ManifestDatabaseParameter{{
					Schema: "demo.v3.database.item-id@1", Field: "id", Kind: "int64", MaxBytes: 8,
				}},
				ResultSchema: "demo.v3.database.items.result@1",
				Columns:      []ManifestDatabaseColumn{{Name: "id"}, {Name: "name", Nullable: true}},
				MaxRows:      100,
				TimeoutMS:    3000,
			},
			{
				ID: "demo.v3.database.items.insert", StatementVersion: "1", Kind: "execute",
				Path: "database/items-insert.sql", Digest: digest,
				Parameters: []ManifestDatabaseParameter{{
					Schema: "demo.v3.database.item-name@1", Field: "name", Kind: "string", MaxBytes: 1024,
				}},
				ResultSchema:    "demo.v3.database.items.result@1",
				Columns:         []ManifestDatabaseColumn{{Name: "id"}, {Name: "name"}},
				MaxAffectedRows: 1,
				TimeoutMS:       3000,
			},
		},
	}
	manifest.Cache = []ManifestCache{{
		ID: "demo.v3.cache.results", ContractVersion: "demo.v3.cache.results@1",
		Namespace: "demo.v3.results", Policy: "actor", Tags: []string{"demo.v3.cache.tag"}, Invalidators: []string{"demo.v3.cache.invalidate"},
	}}
	manifest.Services = []ManifestService{{
		ID: "demo.v3.service.lookup", ContractVersion: "demo.v3.service.lookup@1", Action: "add",
		Handler: "service.lookup", RequestSchema: "demo.v3.service.lookup.request@1", ResponseSchema: "demo.v3.service.lookup.response@1",
	}}
	manifest.Commands = []ManifestCommand{{
		ID: "demo.v3.command.write", ContractVersion: "demo.v3.command.write@1", Handler: "command.write",
		Permission: "demo.v3.manage", InputSchema: "demo.v3.command.write.input@1", ResultSchema: "demo.v3.command.write.result@1",
	}}
	manifest.AdminSurfaces = []ManifestAdminSurface{{
		ID: "demo.v3.admin.notice", ContractVersion: "demo.v3.admin.notice@1", Kind: "notice", Action: "add",
		Label: "Demo notice", Schema: "demo.v3.admin.notice.schema@1", Permission: "demo.v3.manage",
	}}
	manifest.Queries = []ManifestQuery{{
		ID: "demo.v3.query.items", ContractVersion: "demo.v3.query.items@1", Entity: "demo.v3.item",
		PlanVersion: "demo.v3.query.items.plan@1", Fields: []string{"id"}, Pagination: "cursor",
		ResultSchema: "demo.v3.query.items.result@1", PermissionPolicy: "demo.v3.manage", CacheTags: []string{"demo.v3.items"},
	}}
	manifest.PermissionDefinitions = []ManifestPermissionDefinition{{
		Key: "demo.v3.manage", ContractVersion: "demo.v3.permission.manage@1", Label: "Manage demo",
		Description: "Manage the demo plugin.", RecommendedRoles: []string{"administrator"}, AssignmentPolicy: "host",
	}}
	manifest.Identity = &ManifestIdentity{
		ContractVersion: "demo.v3.identity@1", SessionPolicy: "core.session.default",
		UserFields: []ManifestIdentityUserField{{
			ID: "demo.v3.identity.field", ContractVersion: "demo.v3.identity.field@1", Type: "string",
			Schema: "demo.v3.identity.field.schema@1", ReadPermission: "demo.v3.manage", WritePermission: "demo.v3.manage",
		}},
		Providers: []ManifestIdentityProvider{{
			ID: "demo.v3.identity.risk", ContractVersion: "demo.v3.identity.risk@1", Kind: "risk", Handler: "identity.risk",
		}},
	}
	manifest.Media = []ManifestMediaPipeline{{
		ID: "demo.v3.media.image", ContractVersion: "demo.v3.media.image@1", Action: "add",
		MIMEs: []string{"image/png"}, Handler: "media.image", Permission: "demo.v3.manage",
		Transforms: []ManifestMediaTransform{{ID: "thumbnail", Variant: "thumb", Width: 320, Height: 240}},
	}}
	manifest.Navigation = []ManifestNavigation{{
		ID: "demo.v3.navigation.item", ContractVersion: "demo.v3.navigation.item@1",
		Kind: "item", Action: "add", Label: "Demo", Href: "/demo", Order: 10,
	}}
	manifest.Regions = []ManifestRegion{{
		ID: "demo.v3.region.sidebar", ContractVersion: "demo.v3.region.sidebar@1",
		Action: "add", Kind: "sidebar", Label: "Demo sidebar", Multiple: true,
	}}
	manifest.Dependencies = []ManifestDependency{
		{ID: "other.plugin", Version: "^1.0.0", Kind: "required"},
		{Capability: "demo.v3.capability", Version: "1.0.0", Kind: "provides"},
	}
	manifest.Lifecycle = &ManifestLifecycle{
		ContractVersion: "demo.v3.lifecycle@1",
		Install: &ManifestLifecycleOperation{
			Plan: "lifecycle.install.plan", Execute: "lifecycle.install.execute",
			ProgressSchema: "demo.v3.lifecycle.progress@1", CheckpointSchema: "demo.v3.lifecycle.checkpoint@1",
		},
	}
	manifest.OpenAPI = []ManifestOpenAPIFragment{{
		ID: "demo.v3.openapi.routes", ContractVersion: "demo.v3.openapi.routes@1",
		Path: "openapi/routes.yaml", Digest: digest, Namespace: "demo.v3.api",
	}}
	return manifest
}

func v3FixtureBody() []byte {
	return []byte("SForum Manifest V3 fixture\n")
}

func v3FixtureDigest() string {
	digest := sha256.Sum256(v3FixtureBody())
	return hex.EncodeToString(digest[:])
}

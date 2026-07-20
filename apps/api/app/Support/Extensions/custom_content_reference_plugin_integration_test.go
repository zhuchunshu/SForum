package extensionsruntime_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	capabilities "github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	contentregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/ContentRegistry"
	editorregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/EditorRegistry"
	entityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/EntityRegistry"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	navigationregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/NavigationRegistry"
)

// TestReferenceCustomContentPluginPublishesEntityContentEditorNavigation proves
// the P13 custom-content package is installable, publishes Host registries from
// Manifest V3, and falls back cleanly after disable without core product edits.
func TestReferenceCustomContentPluginPublishesEntityContentEditorNavigation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping reference custom-content plugin subprocess build in short mode")
	}
	extension := buildReferenceCustomContentExtension(t)
	if extension.ID != "sforum.custom-content" {
		t.Fatalf("extension id = %q", extension.ID)
	}
	// 独立可安装：Manifest 必须自洽且不依赖 showcase Host 捷径。
	if len(extension.Manifest.Entities) < 3 || len(extension.Manifest.Content) < 4 ||
		len(extension.Manifest.Editor) < 3 || len(extension.Manifest.Navigation) < 1 ||
		len(extension.Manifest.Regions) < 1 {
		t.Fatalf("custom-content surfaces incomplete: entities=%d content=%d editor=%d nav=%d regions=%d",
			len(extension.Manifest.Entities), len(extension.Manifest.Content),
			len(extension.Manifest.Editor), len(extension.Manifest.Navigation),
			len(extension.Manifest.Regions))
	}

	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Trust: staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{
			TrustGrantID: "custom-content-reference", ImpactDigest: extension.PackageDigest,
		}},
	})
	manager := extensionsruntime.NewManager(extensionsruntime.ManagerConfig{Starter: starter})
	if err := manager.Start(t.Context(), extension); err != nil {
		t.Fatalf("start custom-content plugin: %v", err)
	}
	stopped := false
	t.Cleanup(func() {
		if !stopped {
			_ = manager.Stop(context.Background(), extension)
		}
	})
	active, err := manager.ActiveRuntimeInstance(extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	artifactBase := func() (contentregistry.Artifact, entityregistry.Artifact, editorregistry.Artifact, navigationregistry.Artifact) {
		contentArt := contentregistry.Artifact{
			ExtensionID: extension.ID, ExtensionVersion: extension.Version,
			PackageDigest: extension.PackageDigest, VersionID: extension.ActiveVersionID,
			RuntimeInstanceID: active.Identity.InstanceID,
		}
		entityArt := entityregistry.Artifact{
			ExtensionID: extension.ID, ExtensionVersion: extension.Version,
			PackageDigest: extension.PackageDigest, VersionID: extension.ActiveVersionID,
		}
		editorArt := editorregistry.Artifact{
			ExtensionID: extension.ID, ExtensionVersion: extension.Version,
			PackageDigest: extension.PackageDigest, VersionID: extension.ActiveVersionID,
		}
		navArt := navigationregistry.Artifact{
			ExtensionID: extension.ID, ExtensionVersion: extension.Version,
			PackageDigest: extension.PackageDigest, ImpactDigest: extension.PackageDigest,
			VersionID: extension.ActiveVersionID, RuntimeInstanceID: active.Identity.InstanceID,
		}
		return contentArt, entityArt, editorArt, navArt
	}
	contentArt, entityArt, editorArt, navArt := artifactBase()

	// --- Entity ---
	entityReg := entityregistry.New()
	entityDecls := make([]entityregistry.Declaration, 0, len(extension.Manifest.Entities))
	for _, item := range extension.Manifest.Entities {
		entityDecls = append(entityDecls, entityregistry.Declaration{
			ID: item.ID, ContractVersion: item.ContractVersion, Kind: item.Kind, Label: item.Label,
			StorageKey: item.StorageKey, PermissionCreate: item.PermissionCreate, PermissionRead: item.PermissionRead,
			PermissionUpdate: item.PermissionUpdate, PermissionDelete: item.PermissionDelete,
			PermissionImport: item.PermissionImport, PermissionExport: item.PermissionExport,
			ImportExportPolicy: item.ImportExportPolicy, DeletionPolicy: item.DeletionPolicy,
			TaxonomyIDs: append([]string(nil), item.TaxonomyIDs...), Hierarchical: item.Hierarchical,
			EntityIDs: append([]string(nil), item.EntityIDs...), PermissionManage: item.PermissionManage,
			PermissionAssign: item.PermissionAssign, EntityID: item.EntityID, Schema: item.Schema,
			UIComponent: item.UIComponent, Required: item.Required, Indexed: item.Indexed,
			IndexKind: item.IndexKind, PermissionFieldRead: item.PermissionFieldRead,
			PermissionFieldWrite: item.PermissionFieldWrite, Order: item.Order, Priority: item.Priority,
		})
	}
	if _, err := entityReg.Publish(entityregistry.Publication{Artifact: entityArt, Entities: entityDecls}); err != nil {
		t.Fatalf("publish entities: %v", err)
	}
	if _, err := entityReg.Resolve("sforum.custom-content.entity.article"); err != nil {
		t.Fatalf("resolve entity: %v", err)
	}
	if _, err := entityReg.Resolve("sforum.custom-content.taxonomy.topic"); err != nil {
		t.Fatalf("resolve taxonomy: %v", err)
	}
	if _, err := entityReg.Resolve("sforum.custom-content.field.summary"); err != nil {
		t.Fatalf("resolve field: %v", err)
	}

	// --- Content ---
	contentReg := contentregistry.New()
	contentDecls := make([]contentregistry.Declaration, 0, len(extension.Manifest.Content))
	for _, item := range extension.Manifest.Content {
		contentDecls = append(contentDecls, contentregistry.Declaration{
			ID: item.ID, ContractVersion: item.ContractVersion, Kind: item.Kind,
			Handler: item.Handler, Schema: item.Schema, Renderer: item.Renderer, Migration: item.Migration,
		})
	}
	if _, err := contentReg.Publish(contentregistry.Publication{Artifact: contentArt, Content: contentDecls}); err != nil {
		t.Fatalf("publish content: %v", err)
	}
	for _, id := range []string{
		"sforum.custom-content.block.vote",
		"sforum.custom-content.block.product-card",
		"sforum.custom-content.embed.media",
		"sforum.custom-content.block.workflow-form",
	} {
		if _, err := contentReg.Resolve(id); err != nil {
			t.Fatalf("resolve content %s: %v", id, err)
		}
	}

	// --- Editor ---
	editorReg := editorregistry.New()
	editorDecls := make([]editorregistry.Declaration, 0, len(extension.Manifest.Editor))
	for _, item := range extension.Manifest.Editor {
		editorDecls = append(editorDecls, editorregistry.Declaration{
			ID: item.ID, ContractVersion: item.ContractVersion, Kind: item.Kind, Schema: item.Schema,
			ExtensionName: item.ExtensionName, L2Module: item.L2Module, L2Digest: item.L2Digest,
			CommandKey: item.CommandKey, CommandID: item.CommandID, Label: item.Label,
			Icon: item.Icon, Group: item.Group, Order: item.Order, Priority: item.Priority,
			Permission: item.Permission,
		})
	}
	if _, err := editorReg.Publish(editorregistry.Publication{Artifact: editorArt, Editor: editorDecls}); err != nil {
		t.Fatalf("publish editor: %v", err)
	}
	if _, err := editorReg.Resolve("sforum.custom-content.editor.node.vote"); err != nil {
		t.Fatalf("resolve editor node: %v", err)
	}

	// --- Navigation / Regions ---
	navReg := navigationregistry.New()
	navDecls := make([]navigationregistry.NavigationDeclaration, 0, len(extension.Manifest.Navigation))
	for _, item := range extension.Manifest.Navigation {
		decl := navigationregistry.NavigationDeclaration{
			ID: item.ID, ContractVersion: item.ContractVersion, Kind: item.Kind, Action: item.Action,
			TargetID: item.TargetID, Label: item.Label, Href: item.Href, Permission: item.Permission,
			Order: item.Order, Visibility: navigationregistry.VisibilityPublic,
		}
		if item.Label != "" {
			decl.Labels = map[string]string{"zh-CN": item.Label}
		}
		navDecls = append(navDecls, decl)
	}
	regionDecls := make([]navigationregistry.RegionDeclaration, 0, len(extension.Manifest.Regions))
	for _, item := range extension.Manifest.Regions {
		decl := navigationregistry.RegionDeclaration{
			ID: item.ID, ContractVersion: item.ContractVersion, Action: item.Action,
			TargetID: item.TargetID, Kind: item.Kind, Label: item.Label, Multiple: item.Multiple,
			Visibility: navigationregistry.VisibilityPublic,
		}
		if item.Label != "" {
			decl.Labels = map[string]string{"zh-CN": item.Label}
		}
		regionDecls = append(regionDecls, decl)
	}
	if _, err := navReg.Publish(navigationregistry.Publication{
		Artifact: navArt, Navigation: navDecls, Regions: regionDecls,
	}); err != nil {
		t.Fatalf("publish navigation: %v", err)
	}
	navPub, found := navReg.SnapshotPublication(extension.ID)
	if !found || len(navPub.Navigation) != 1 || navPub.Navigation[0].ID != "sforum.custom-content.nav.articles" {
		t.Fatalf("navigation publication = %#v found=%v", navPub, found)
	}
	if len(navPub.Regions) != 1 || navPub.Regions[0].ID != "sforum.custom-content.region.sidebar" {
		t.Fatalf("region publication = %#v", navPub.Regions)
	}

	// --- Disable / remove: declarations leave Host graph; no core rewrite required. ---
	if err := manager.Stop(t.Context(), extension); err != nil {
		t.Fatal(err)
	}
	stopped = true
	if _, removed, err := contentReg.Remove(contentArt); err != nil || !removed {
		t.Fatalf("remove content: removed=%v err=%v", removed, err)
	}
	if _, err := contentReg.Resolve("sforum.custom-content.block.vote"); err != contentregistry.ErrNotFound {
		t.Fatalf("content after disable = %v", err)
	}
	if _, removed, err := entityReg.Remove(entityArt); err != nil || !removed {
		t.Fatalf("remove entity: removed=%v err=%v", removed, err)
	}
	if _, err := entityReg.Resolve("sforum.custom-content.entity.article"); err != entityregistry.ErrNotFound {
		t.Fatalf("entity after disable = %v", err)
	}
	if _, removed, err := editorReg.Remove(editorArt); err != nil || !removed {
		t.Fatalf("remove editor: removed=%v err=%v", removed, err)
	}
	if _, removed, err := navReg.Remove(navArt); err != nil || !removed {
		t.Fatalf("remove navigation: removed=%v err=%v", removed, err)
	}
}

func buildReferenceCustomContentExtension(t *testing.T) extensions.Extension {
	t.Helper()
	fixtureRoot := referenceCustomContentFixtureRoot(t)
	packageRoot := filepath.Join(t.TempDir(), "sforum.custom-content")
	if err := os.CopyFS(packageRoot, os.DirFS(fixtureRoot)); err != nil {
		t.Fatalf("copy custom-content plugin: %v", err)
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
		t.Fatalf("build custom-content plugin: %v\n%s", err, output)
	}
	templateBody, err := os.ReadFile(filepath.Join(packageRoot, "sforum.extension.json.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	editorPath := filepath.Join(packageRoot, "frontend", "editor", "vote.mjs")
	schemaPath := filepath.Join(packageRoot, "schemas", "articles.json")
	manifestBody := string(templateBody)
	manifestBody = strings.ReplaceAll(manifestBody, "__BACKEND_DIGEST__", fileSHA256(t, binaryPath))
	manifestBody = strings.ReplaceAll(manifestBody, "__EDITOR_VOTE_DIGEST__", fileSHA256(t, editorPath))
	manifestBody = strings.ReplaceAll(manifestBody, "__ARTICLES_SCHEMA_DIGEST__", fileSHA256(t, schemaPath))
	if strings.Contains(manifestBody, "__") {
		t.Fatal("custom-content manifest still contains digest tokens")
	}
	if err := os.WriteFile(filepath.Join(packageRoot, extensionmanifest.ManifestFileName), []byte(manifestBody), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := extensionmanifest.LoadPackage(packageRoot)
	if err != nil {
		t.Fatalf("load custom-content package: %v", err)
	}
	packageDigest, err := extensionpackage.DigestTree(packageRoot)
	if err != nil {
		t.Fatal(err)
	}
	return extensions.Extension{
		ID: manifest.ID, Name: manifest.Name, Version: manifest.Version, Type: extensions.TypePlugin,
		Status: extensions.StatusEnabled, Source: extensions.SourceUploaded,
		PackagePath: packageRoot, PackageDigest: packageDigest, Manifest: manifest, ActiveVersionID: 601,
		CapabilityGrants: []extensions.CapabilityGrant{{Key: capabilities.HostAPI, Risk: capabilities.RiskLow}},
	}
}

func referenceCustomContentFixtureRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../extensions/fixtures/plugins/sforum-custom-content"))
}

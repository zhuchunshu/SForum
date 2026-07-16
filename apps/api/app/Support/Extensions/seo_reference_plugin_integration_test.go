package extensionsruntime_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	capabilities "github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	seoregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/SEORegistry"
)

func TestReferenceSEOPluginUsesRealProtocolV2AndCoreFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping reference SEO plugin subprocess build in short mode")
	}
	extension := buildReferenceSEOExtension(t)
	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Trust: staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{
			TrustGrantID: "seo-reference", ImpactDigest: extension.PackageDigest,
		}},
	})
	manager := extensionsruntime.NewManager(extensionsruntime.ManagerConfig{Starter: starter})
	if err := manager.Start(t.Context(), extension); err != nil {
		t.Fatalf("start reference SEO plugin: %v", err)
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
	declaration := extension.Manifest.SEO[0]
	registry := seoregistry.New()
	if _, err := registry.Publish(seoregistry.Publication{
		Artifact: seoregistry.Artifact{
			ExtensionID: extension.ID, ExtensionVersion: extension.Version,
			PackageDigest: extension.PackageDigest, ImpactDigest: extension.PackageDigest,
			VersionID: extension.ActiveVersionID, RuntimeInstanceID: active.Identity.InstanceID,
		},
		Contributions: []seoregistry.Declaration{{
			ID: declaration.ID, ContractVersion: declaration.ContractVersion,
			Scope: declaration.Scope, Kind: declaration.Kind, Action: declaration.Action,
			Handler: declaration.Handler, Priority: declaration.Priority,
			FailurePolicy: declaration.FailurePolicy, Timeout: time.Duration(declaration.TimeoutMS) * time.Millisecond,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	policy, err := seoregistry.NewHostFinalPolicy(seoregistry.HostFinalPolicyConfig{
		SiteURL: "https://forum.example", SupportedLocales: []string{"zh-CN", "en-US"},
		AllowIndexing: true, SitemapEnabled: true, StructuredDataEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	trace := seoregistry.NewExecutionTraceRing(8)
	execution, err := seoregistry.NewExecutionRuntime(seoregistry.ExecutionConfig{
		Registry: registry, Resolver: extensionsruntime.NewProtocolV2SEOProviderResolver(manager),
		Admission: extensionsruntime.NewSEOExecutionAdmission(manager), FinalPolicy: policy, Trace: trace,
	})
	if err != nil {
		t.Fatal(err)
	}
	base := referenceSEOBase("Core topic")
	result, err := execution.Execute(t.Context(), seoregistry.ExecuteRequest{Scope: declaration.Scope, Base: base})
	if err != nil || result.Document.Title != "Core topic | SEO Reference" || len(result.Applied) != 1 {
		t.Fatalf("reference SEO result=%#v err=%v", result, err)
	}

	failedBase := referenceSEOBase("reference:fail")
	result, err = execution.Execute(t.Context(), seoregistry.ExecuteRequest{Scope: declaration.Scope, Base: failedBase})
	if err != nil || result.Document.Title != failedBase.Title || len(result.Fallbacks) != 1 ||
		result.Fallbacks[0].Reason != "provider_failed" {
		t.Fatalf("reference failure fallback=%#v err=%v", result, err)
	}
	if err := manager.Stop(t.Context(), extension); err != nil {
		t.Fatal(err)
	}
	stopped = true
	result, err = execution.Execute(t.Context(), seoregistry.ExecuteRequest{Scope: declaration.Scope, Base: base})
	if err != nil || result.Document.Title != base.Title || len(result.Fallbacks) != 1 ||
		result.Fallbacks[0].Reason != "runtime_unavailable" {
		t.Fatalf("reference disable fallback=%#v err=%v", result, err)
	}
	traces := trace.SEOExecutionTraces(3)
	if len(traces) != 3 || traces[0].Fallbacks != 1 || traces[1].Fallbacks != 1 ||
		traces[2].Applied != 1 || traces[2].Calls[0].ExtensionID != extension.ID {
		t.Fatalf("reference SEO traces=%#v", traces)
	}
}

func referenceSEOBase(title string) seoregistry.Document {
	return seoregistry.Document{
		Title: title, CanonicalURL: "https://forum.example/topic/1",
		Robots: seoregistry.RobotsDirectives{Indexing: seoregistry.RobotsIndex, Following: seoregistry.RobotsFollow},
	}
}

func buildReferenceSEOExtension(t *testing.T) extensions.Extension {
	t.Helper()
	fixtureRoot := referenceSEOFixtureRoot(t)
	packageRoot := filepath.Join(t.TempDir(), "sforum.seo-reference")
	if err := os.CopyFS(packageRoot, os.DirFS(fixtureRoot)); err != nil {
		t.Fatalf("copy reference SEO plugin: %v", err)
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
		t.Fatalf("build reference SEO plugin: %v\n%s", err, output)
	}
	templateBody, err := os.ReadFile(filepath.Join(packageRoot, "sforum.extension.json.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	manifestBody := strings.ReplaceAll(string(templateBody), "__BACKEND_DIGEST__", fileSHA256(t, binaryPath))
	if err := os.WriteFile(filepath.Join(packageRoot, extensionmanifest.ManifestFileName), []byte(manifestBody), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := extensionmanifest.LoadPackage(packageRoot)
	if err != nil {
		t.Fatalf("load exact SEO reference package: %v", err)
	}
	packageDigest, err := extensionpackage.DigestTree(packageRoot)
	if err != nil {
		t.Fatal(err)
	}
	return extensions.Extension{
		ID: manifest.ID, Name: manifest.Name, Version: manifest.Version, Type: extensions.TypePlugin,
		Status: extensions.StatusEnabled, Source: extensions.SourceUploaded,
		PackagePath: packageRoot, PackageDigest: packageDigest, Manifest: manifest, ActiveVersionID: 501,
		CapabilityGrants: []extensions.CapabilityGrant{{Key: capabilities.HostAPI, Risk: capabilities.RiskLow}},
	}
}

func referenceSEOFixtureRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../extensions/fixtures/plugins/sforum-seo-reference"))
}

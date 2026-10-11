package extensionopenapi_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionopenapi "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionOpenAPI"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

// TestGamificationRouteSchemaCatalogBuildsFromRealPackage 是 M4 路由 schema
// 引用格式修复的回归守卫：之前路由用 sforum.gamification.route.*@1 聚合 ID
// 但没有声明 openapi fragment，boot 构建 RouteSchemaCatalog 时报
// "route schema is missing" 导致整个 API 崩溃。补 openapi fragment 后，从
// 真实包加载并构建 catalog 必须成功且所有路由方法都有 binding。
func TestGamificationRouteSchemaCatalogBuildsFromRealPackage(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	packageRoot := filepath.Clean(filepath.Join(filepath.Dir(file),
		"../../../../../extensions/builtin/plugins/sforum-gamification"))

	// loadArtifact 读磁盘原始 manifest + Normalize（不做 includes 合并），
	// 传入的 snapshot 必须与之一致，否则 manifestsEqual 判漂移。
	manifestBody, err := os.ReadFile(filepath.Join(packageRoot, extensionmanifest.ManifestFileName))
	if err != nil {
		t.Fatalf("read manifest bytes: %v", err)
	}
	var manifest extensionmanifest.Manifest
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	manifest = extensionmanifest.Normalize(manifest)
	if err := extensionmanifest.Validate(manifest); err != nil {
		t.Fatalf("validate manifest: %v", err)
	}
	digest, err := extensionpackage.DigestTree(packageRoot)
	if err != nil {
		t.Fatalf("compute package digest: %v", err)
	}

	artifact := extensionopenapi.Artifact{
		Root:          packageRoot,
		ExtensionID:   manifest.ID,
		Version:       manifest.Version,
		PackageDigest: digest,
		Manifest:      manifest,
	}

	catalog, err := extensionopenapi.BuildRouteSchemaCatalog(extensionopenapi.BuildInput{Artifacts: []extensionopenapi.Artifact{artifact}})
	if err != nil {
		t.Fatalf("build route schema catalog from real gamification package: %v", err)
	}
	if catalog.Revision() == "" {
		t.Fatal("catalog revision is empty")
	}

	bindings := catalog.Bindings()
	if len(bindings) == 0 {
		t.Fatal("catalog has no schema bindings; routes are not published")
	}

	// 每条 HTTP 路由的每个方法都应该有 request + response binding。
	type methodKey struct{ routeID, method, direction string }
	seen := make(map[methodKey]bool)
	for _, binding := range bindings {
		seen[methodKey{binding.RouteID, binding.Method, string(binding.Direction)}] = true
	}
	for _, route := range manifest.Routes {
		if route.Mode != extensionmanifest.RouteModeHTTP {
			continue
		}
		for _, method := range route.Methods {
			if route.RequestSchema != "" && !seen[methodKey{route.ID, method, "request"}] {
				t.Errorf("route %s %s request schema binding missing", route.ID, method)
			}
			if route.ResponseSchema != "" && !seen[methodKey{route.ID, method, "response"}] {
				t.Errorf("route %s %s response schema binding missing", route.ID, method)
			}
		}
	}
}

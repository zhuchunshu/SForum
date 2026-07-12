package pages

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func fixturePluginRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	// apps/api/app/Support/Pages -> repo root
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../"))
	return filepath.Join(root, "extensions/fixtures/plugins/page-registry-demo")
}

func TestFixturePluginRegistersAddAndReplaceCandidates(t *testing.T) {
	root := fixturePluginRoot(t)
	if _, err := os.Stat(filepath.Join(root, "theme.json")); err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	store := NewMemoryStore()
	reg := NewRegistry(store)
	bridge := NewExtensionBridge(reg)
	ctx := context.Background()

	ext := ThemeExtension{
		ID: "sforum.page-registry-demo", Version: "1.0.0",
		PackagePath: root, PackageDigest: "digest-v1",
	}
	if err := bridge.RegisterPluginPackage(ctx, ext); err != nil {
		t.Fatal(err)
	}

	// add 公开路径
	c, ok := reg.ResolveAddedPath("/demo-docs/hello")
	if !ok || c.ID != "sforum.page-registry-demo.docs" {
		t.Fatalf("add path: %#v ok=%v", c, ok)
	}
	// login 路径
	c2, ok := reg.ResolveAddedPath("/demo-members")
	if !ok || c2.Access != AccessLogin {
		t.Fatalf("login path: %#v", c2)
	}
	// replace 未批准 → core
	r, err := reg.Resolve(ctx, "forum.home")
	if err != nil || r.Provider != ProviderCore {
		t.Fatalf("replace before approve: %#v err=%v", r, err)
	}
	// super_admin 批准后生效
	if err := reg.ApproveReplace(ctx, ProviderBinding{
		PageID: "forum.home", ExtensionID: ext.ID, ContributionID: "sforum.page-registry-demo.home",
		Version: "1.0.0", PackageDigest: "digest-v1", ApprovedBy: 1,
	}); err != nil {
		t.Fatal(err)
	}
	r, _ = reg.Resolve(ctx, "forum.home")
	if r.Provider != ext.ID {
		t.Fatalf("expected plugin provider, got %#v", r)
	}
	// 禁用 → 清贡献 → 回退 core
	bridge.ClearExtension(ext.ID)
	r, _ = reg.Resolve(ctx, "forum.home")
	if r.Provider != ProviderCore {
		t.Fatalf("after disable expected core, got %#v", r)
	}
	if _, ok := reg.ResolveAddedPath("/demo-docs/hello"); ok {
		t.Fatal("add path must be gone after disable")
	}
}

func TestFixturePluginUpgradeInvalidatesDigestApproval(t *testing.T) {
	root := fixturePluginRoot(t)
	store := NewMemoryStore()
	reg := NewRegistry(store)
	bridge := NewExtensionBridge(reg)
	ctx := context.Background()
	ext := ThemeExtension{
		ID: "sforum.page-registry-demo", Version: "1.0.0",
		PackagePath: root, PackageDigest: "digest-v1",
	}
	_ = bridge.RegisterPluginPackage(ctx, ext)
	_ = reg.ApproveReplace(ctx, ProviderBinding{
		PageID: "forum.home", ExtensionID: ext.ID, ContributionID: "sforum.page-registry-demo.home",
		Version: "1.0.0", PackageDigest: "digest-v1", ApprovedBy: 1,
	})
	// 升级：新 digest 重新注册
	ext.Version = "1.0.1"
	ext.PackageDigest = "digest-v2"
	if err := bridge.RegisterPluginPackage(ctx, ext); err != nil {
		t.Fatal(err)
	}
	r, _ := reg.Resolve(ctx, "forum.home")
	if r.Provider != ProviderCore {
		t.Fatalf("upgrade must invalidate old digest approval, got %#v", r)
	}
}

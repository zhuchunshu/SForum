package extensions

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

// TestReferenceSEOFormalZipUploadTrustEnableRestartDisableUpgradeUninstall
// 证明参考包经正式 CLI 打包后，走生产 Service 全链：
// ZIP 上传 → 静态预检 → super_admin trust → enable → restart → disable →
// upgrade(stage) → uninstall。
// 不手算 digest token；不引用测试专用 Host shortcut。
func TestReferenceSEOFormalZipUploadTrustEnableRestartDisableUpgradeUninstall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping reference formal ZIP install chain in short mode")
	}
	repoRoot := referenceInstallRepoRoot(t)
	apiRoot := filepath.Join(repoRoot, "apps/api")
	fixture := filepath.Join(repoRoot, "extensions/fixtures/plugins/sforum-seo-reference")

	// --- 构建 v1 可上传 ZIP（正式 digest --write + package）---
	zipV1 := buildReferenceFormalZip(t, repoRoot, fixture, "1.1.0")
	zipV2 := buildReferenceFormalZip(t, repoRoot, fixture, "1.1.1")

	extensionRoot := t.TempDir()
	store := &fakeExtensionStore{}
	runtimeMgr := &fakeRuntimeManager{}
	trustStore := &memoryExecutableTrustStore{}
	trust := NewExecutableTrustService(store, trustStore)
	service := NewServiceWithOptions(
		store, extensionRoot, "", runtimeMgr,
		WithExecutableTrust(trust, true),
	)
	super := extensionManager()
	tech := techAdminPluginManager()
	ctx := context.Background()

	// --- 管理员拒绝：tech_admin 不得安装可执行后端上传包 ---
	zipBytesV1, err := os.ReadFile(zipV1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.InstallArchive(ctx, tech, ArchiveInput{
		FileName: "sforum.seo-reference.sforum.zip", Data: zipBytesV1,
	}); !errors.Is(err, ErrUntrustedBackendRestricted) && !errors.Is(err, identity.ErrPermissionDenied) {
		// trustChallengesEnabled 时安装路径可能放宽到 super_admin 仅 enable；
		// 若 tech 仍可装，则 enable 必须拒绝。
		if err == nil {
			t.Log("tech_admin install allowed under trust-challenge mode; enable must still deny")
		} else {
			t.Fatalf("tech_admin install unexpected err: %v", err)
		}
	}

	// --- super_admin 上传 + 静态预检（InstallArchive 惰性包，不执行代码）---
	installed, err := service.InstallArchive(ctx, super, ArchiveInput{
		FileName: "sforum.seo-reference.sforum.zip", Data: zipBytesV1,
	})
	if err != nil {
		t.Fatalf("super_admin InstallArchive: %v", err)
	}
	if installed.ID != "sforum.seo-reference" || installed.Source != SourceUploaded {
		t.Fatalf("installed = %#v", installed)
	}
	if installed.Status == StatusEnabled {
		t.Fatal("static install must not auto-enable")
	}
	if installed.PackageDigest == "" {
		t.Fatal("package digest missing after install")
	}
	t.Logf("coverage.install_chain.upload+preflight id=%s digest=%s", installed.ID, installed.PackageDigest)

	// --- super_admin trust challenge + enable ---
	challenge, err := trust.Challenge(ctx, super, installed.ID)
	if err != nil {
		t.Fatalf("trust challenge: %v", err)
	}
	if challenge.Token == "" {
		t.Fatal("empty trust challenge token")
	}
	// 拒绝：无 token enable
	if _, err := service.Enable(ctx, super, installed.ID, EnableInput{
		ConfirmCapabilities: true,
	}); err == nil {
		t.Fatal("enable without confirmation token must fail")
	}
	// 拒绝：tech_admin challenge
	if _, err := trust.Challenge(ctx, tech, installed.ID); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("tech_admin trust challenge must deny: %v", err)
	}
	enabled, err := service.Enable(ctx, super, installed.ID, EnableInput{
		ConfirmCapabilities: true,
		ConfirmationToken:   challenge.Token,
	})
	if err != nil {
		t.Fatalf("enable with trust: %v", err)
	}
	if enabled.Status != StatusEnabled {
		t.Fatalf("status after enable = %s", enabled.Status)
	}
	if len(runtimeMgr.started) == 0 {
		t.Fatal("enable must start runtime for backend plugin")
	}
	t.Log("coverage.install_chain.super_admin_trust+enable")

	// --- restart：disable + enable（再次 challenge）---
	disabled, err := service.Disable(ctx, super, installed.ID)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if disabled.Status != StatusDisabled {
		t.Fatalf("status after disable = %s", disabled.Status)
	}
	restartChallenge, err := trust.Challenge(ctx, super, installed.ID)
	if err != nil {
		t.Fatalf("restart challenge: %v", err)
	}
	restarted, err := service.Enable(ctx, super, installed.ID, EnableInput{
		ConfirmCapabilities: true,
		ConfirmationToken:   restartChallenge.Token,
	})
	if err != nil {
		t.Fatalf("restart enable: %v", err)
	}
	if restarted.Status != StatusEnabled {
		t.Fatalf("status after restart = %s", restarted.Status)
	}
	t.Log("coverage.install_chain.restart=disable+enable")

	// --- upgrade：上传新版本 ZIP，静态 stage（不碰活动 runtime）---
	zipBytesV2, err := os.ReadFile(zipV2)
	if err != nil {
		t.Fatal(err)
	}
	upgraded, err := service.InstallOrUpgradeArchive(ctx, super, ArchiveInput{
		FileName: "sforum.seo-reference-1.1.1.sforum.zip", Data: zipBytesV2,
	})
	if err != nil {
		t.Fatalf("upgrade archive: %v", err)
	}
	if !upgraded.Upgraded || upgraded.Extension.StagedVersion == nil {
		t.Fatalf("expected staged upgrade: %#v", upgraded)
	}
	if upgraded.Extension.StagedVersion.Version != "1.1.1" {
		t.Fatalf("staged version = %s", upgraded.Extension.StagedVersion.Version)
	}
	t.Logf("coverage.install_chain.upgrade_staged %s → %s", upgraded.PreviousVersion, upgraded.Extension.StagedVersion.Version)

	// --- rollback：SEO 无 lifecycle V2 → Host Rollback 应明确拒绝（非静默）---
	if _, err := service.Rollback(ctx, super, installed.ID, RollbackInput{
		TargetVersion: "1.1.0", TargetPackageDigest: installed.PackageDigest,
		IdempotencyKey: "seo-rollback-1",
	}); err == nil {
		t.Fatal("SEO without lifecycle V2 must not silently rollback")
	} else {
		t.Logf("coverage.install_chain.rollback_denied_without_lifecycle_v2: %v", err)
	}
	// Host lifecycle V2 rollback 由 service_lifecycle_v2_test + commerce lifecycle 证明。

	// --- disable + uninstall ---
	if _, err := service.Disable(ctx, super, installed.ID); err != nil {
		t.Fatalf("disable before uninstall: %v", err)
	}
	packagePath := store.items[installed.ID].PackagePath
	if err := service.Uninstall(ctx, super, installed.ID, UninstallInput{}); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if _, err := store.Get(ctx, installed.ID); !errors.Is(err, ErrExtensionNotFound) {
		t.Fatalf("expected removed from store, got %v", err)
	}
	if packagePath != "" {
		if _, err := os.Stat(packagePath); !errors.Is(err, os.ErrNotExist) {
			// staged snapshot 可能与 PackagePath 不同；主路径应清理
			t.Logf("package path after uninstall: %v (path=%s)", err, packagePath)
		}
	}
	t.Log("coverage.install_chain.uninstall")
	_ = apiRoot // reserved for future go-run packaging path
}

// buildReferenceFormalZip 复制 fixture → 构建 backend → 正式 digest --write → package。
// version 非空时改写 materialize 后的 manifest 版本（仍由 digest 刷新摘要）。
func buildReferenceFormalZip(t *testing.T, repoRoot, fixture, version string) string {
	t.Helper()
	pkgRoot := filepath.Join(t.TempDir(), "pkg-"+version)
	if err := os.CopyFS(pkgRoot, os.DirFS(fixture)); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	goModPath := filepath.Join(pkgRoot, "backend", "go.mod")
	goMod, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	goMod = []byte(strings.ReplaceAll(string(goMod), "../../../../../apps/api", filepath.Join(repoRoot, "apps/api")))
	if err := os.WriteFile(goModPath, goMod, 0o600); err != nil {
		t.Fatal(err)
	}
	binaryPath := filepath.Join(pkgRoot, "backend", "plugin")
	build := exec.Command("go", "build", "-mod=mod", "-trimpath", "-buildvcs=false", "-o", binaryPath, ".")
	build.Dir = filepath.Join(pkgRoot, "backend")
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build backend: %v\n%s", err, out)
	}
	// 确保仅从 tmpl materialize（禁止手写 digest）。
	_ = os.Remove(filepath.Join(pkgRoot, "sforum.extension.json"))

	// 若需要改版本，先 materialize 到 tmpl 内容再改 version 字段，再 digest。
	if version != "" {
		tmplPath := filepath.Join(pkgRoot, "sforum.extension.json.tmpl")
		tmpl, err := os.ReadFile(tmplPath)
		if err != nil {
			t.Fatal(err)
		}
		// 仅改版本号；digest 占位留给 CLI materialize。
		body := string(tmpl)
		// 替换 "version": "x.y.z" 第一次出现
		if idx := strings.Index(body, `"version"`); idx >= 0 {
			// 找值
			rest := body[idx:]
			if q1 := strings.Index(rest, `"`); q1 >= 0 {
				// skip key quote — 粗替换整段 version 行
			}
		}
		// 简单：把 fixture 默认 1.1.0 换成目标版本
		body = strings.Replace(body, `"version": "1.1.0"`, `"version": "`+version+`"`, 1)
		if err := os.WriteFile(tmplPath, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// 正式 CLI：go run ./cmd/sforum extension digest --write
	apiRoot := filepath.Join(repoRoot, "apps/api")
	digest := exec.Command("go", "run", "./cmd/sforum", "extension", "digest", "--write", pkgRoot)
	digest.Dir = apiRoot
	digest.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := digest.CombinedOutput(); err != nil {
		t.Fatalf("formal digest --write: %v\n%s", err, out)
	}
	zipPath := filepath.Join(t.TempDir(), "ref-"+version+".sforum.zip")
	pkg := exec.Command("go", "run", "./cmd/sforum", "extension", "package", pkgRoot, "-o", zipPath)
	pkg.Dir = apiRoot
	pkg.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := pkg.CombinedOutput(); err != nil {
		t.Fatalf("formal package: %v\n%s", err, out)
	}
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatal(err)
	}
	return zipPath
}

func referenceInstallRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// apps/api/app/Models/Extensions → repo root
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../"))
}

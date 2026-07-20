package main

import (
	"bytes"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExtensionTestCommandEventsFixture(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
	fixture := filepath.Join(repoRoot, "extensions/fixtures/plugins/sforum-contract-events")

	cmd := newRootCommand()
	cmd.SetArgs([]string{"extension", "test", fixture})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("extension test: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "PASS") {
		t.Fatalf("expected PASS:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "sforum.contract.events") {
		t.Fatalf("expected id:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "event.known") {
		t.Fatalf("expected event.known check:\n%s", out.String())
	}
}

func TestExtensionTestCommandSMTPPackage(t *testing.T) {
	// V3 packageFiles 绑定真实二进制摘要；在临时目录构建并刷新 digest，
	// 避免依赖 gitignored 工作树产物或平台摘要漂移。
	smtpRoot := prepareBuiltinPluginPackage(t, "sforum-smtp")

	cmd := newRootCommand()
	cmd.SetArgs([]string{"extension", "test", smtpRoot})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("extension test smtp: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "sforum.smtp") {
		t.Fatalf("expected smtp id:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "provider.slot_ok") {
		t.Fatalf("expected provider.slot_ok check:\n%s", out.String())
	}
}

// prepareBuiltinPluginPackage 复制受保护内置插件到临时目录，构建 Linux 后端并刷新 V3 digest。
func prepareBuiltinPluginPackage(t *testing.T, pluginDirName string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
	sourceRoot := filepath.Join(repoRoot, "extensions/builtin/plugins", pluginDirName)
	pkgRoot := filepath.Join(t.TempDir(), pluginDirName)
	if err := os.CopyFS(pkgRoot, os.DirFS(sourceRoot)); err != nil {
		t.Fatalf("copy %s package: %v", pluginDirName, err)
	}
	backendRoot := filepath.Join(sourceRoot, "backend")
	binary := filepath.Join(pkgRoot, "backend", "plugin")
	// 删除可能从源树拷入的本地产物，强制使用本次确定性构建。
	_ = os.Remove(binary)
	build := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-o", binary, ".")
	build.Dir = backendRoot
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build %s backend: %v\n%s", pluginDirName, err, output)
	}
	digestCmd := newRootCommand()
	digestCmd.SetArgs([]string{"extension", "digest", "--write", pkgRoot})
	var digestOut strings.Builder
	digestCmd.SetOut(&digestOut)
	digestCmd.SetErr(&digestOut)
	if err := digestCmd.Execute(); err != nil {
		t.Fatalf("refresh %s digest: %v\n%s", pluginDirName, err, digestOut.String())
	}
	return pkgRoot
}

func TestExtensionTestCommandContentPolicyV2Package(t *testing.T) {
	// 额外证明 Linux 后端构建可复现，再走 extension test 契约门禁。
	pkgRoot := prepareBuiltinPluginPackage(t, "sforum-content-policy")
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
	sourceBackend := filepath.Join(repoRoot, "extensions/builtin/plugins/sforum-content-policy/backend")
	firstBody, err := os.ReadFile(filepath.Join(pkgRoot, "backend", "plugin"))
	if err != nil {
		t.Fatal(err)
	}
	secondBinary := filepath.Join(t.TempDir(), "plugin")
	build := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-o", secondBinary, ".")
	build.Dir = sourceBackend
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build reproducible Linux content-policy v2: %v\n%s", err, output)
	}
	secondBody, err := os.ReadFile(secondBinary)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstBody, secondBody) {
		t.Fatalf("content-policy Linux build is not reproducible: %x != %x", sha256.Sum256(firstBody), sha256.Sum256(secondBody))
	}

	cmd := newRootCommand()
	cmd.SetArgs([]string{"extension", "test", pkgRoot})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("extension test content-policy: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "sforum.content-policy") {
		t.Fatalf("expected content-policy id:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "event.known") {
		t.Fatalf("expected event.known checks:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "contribution.point_ok") {
		t.Fatalf("expected contribution checks:\n%s", out.String())
	}
}

func TestDockerBuildsProtectedBuiltinBackendsAndValidatesV3Digest(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dockerfile := filepath.Join(filepath.Dir(file), "../../Dockerfile")
	body, err := os.ReadFile(dockerfile)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	const linuxBuild = "CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false -o plugin ."
	if count := strings.Count(text, linuxBuild); count < 3 {
		t.Errorf("Dockerfile builds only %d protected builtin Linux backends, want at least 3", count)
	}
	const builtinCopy = "COPY --from=build --chown=sforum:sforum /app/extensions/builtin /app/extensions/builtin"
	if count := strings.Count(text, builtinCopy); count != 2 {
		t.Errorf("Dockerfile copies protected builtins into %d final images, want api and worker", count)
	}
	for _, required := range []string{
		"cd /app/extensions/builtin/plugins/sforum-smtp/backend",
		"cd /app/extensions/builtin/plugins/sforum-content-policy/backend",
		"cd /app/extensions/builtin/plugins/sforum-storage-fs/backend",
		// 三个受保护插件均需 digest --write + extension test，避免 Linux 镜像摘要漂移。
		"extension digest --write /app/extensions/builtin/plugins/sforum-smtp",
		"extension test /app/extensions/builtin/plugins/sforum-smtp",
		"extension digest --write /app/extensions/builtin/plugins/sforum-content-policy",
		"extension validate /app/extensions/builtin/plugins/sforum-content-policy",
		"extension test /app/extensions/builtin/plugins/sforum-content-policy",
		"extension digest --write /app/extensions/builtin/plugins/sforum-storage-fs",
		"extension test /app/extensions/builtin/plugins/sforum-storage-fs",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("Dockerfile is missing protected builtin Linux package gate %q", required)
		}
	}
}

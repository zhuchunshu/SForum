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

func TestExtensionTestCommandSMTPWithSkipBinary(t *testing.T) {
	// SMTP 包 backend/plugin 可能是构建产物或缺失；用 skip 做 CLI 冒烟。
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
	smtpRoot := filepath.Join(repoRoot, "extensions/builtin/plugins/sforum-smtp")

	cmd := newRootCommand()
	cmd.SetArgs([]string{"extension", "test", "--skip-backend-binary", smtpRoot})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("extension test smtp: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "sforum.smtp") {
		t.Fatalf("expected smtp id:\n%s", out.String())
	}
}

func TestExtensionTestCommandContentPolicyV2Package(t *testing.T) {
	// V3 packageFiles 必须校验真实字节；临时构建避免测试依赖 gitignored 本地产物。
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
	sourceRoot := filepath.Join(repoRoot, "extensions/builtin/plugins/sforum-content-policy")
	pkgRoot := filepath.Join(t.TempDir(), "sforum-content-policy")
	if err := os.CopyFS(pkgRoot, os.DirFS(sourceRoot)); err != nil {
		t.Fatalf("copy content-policy package: %v", err)
	}
	backendRoot := filepath.Join(sourceRoot, "backend")
	binary := filepath.Join(pkgRoot, "backend", "plugin")
	secondBinary := filepath.Join(t.TempDir(), "plugin")
	for _, outputPath := range []string{binary, secondBinary} {
		build := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-o", outputPath, ".")
		build.Dir = backendRoot
		build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux")
		if output, err := build.CombinedOutput(); err != nil {
			t.Fatalf("build reproducible Linux content-policy v2: %v\n%s", err, output)
		}
	}
	firstBody, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	secondBody, err := os.ReadFile(secondBinary)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstBody, secondBody) {
		t.Fatalf("content-policy Linux build is not reproducible: %x != %x", sha256.Sum256(firstBody), sha256.Sum256(secondBody))
	}

	digestCmd := newRootCommand()
	digestCmd.SetArgs([]string{"extension", "digest", "--write", pkgRoot})
	var digestOut strings.Builder
	digestCmd.SetOut(&digestOut)
	digestCmd.SetErr(&digestOut)
	if err := digestCmd.Execute(); err != nil {
		t.Fatalf("refresh content-policy digest: %v\n%s", err, digestOut.String())
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
		"extension digest --write /app/extensions/builtin/plugins/sforum-content-policy",
		"extension validate /app/extensions/builtin/plugins/sforum-content-policy",
		"extension test /app/extensions/builtin/plugins/sforum-content-policy",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("Dockerfile is missing protected builtin Linux package gate %q", required)
		}
	}
}

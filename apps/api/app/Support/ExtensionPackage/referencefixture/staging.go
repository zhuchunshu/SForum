// Package referencefixture 提供测试专用的参考插件夹具暂存能力。
//
// 仓库内的参考插件夹具不提交 `backend/plugin` 二进制（.gitignore），
// 而 manifest 又按「exact-artifact」契约写死该文件的 SHA-256 摘要——摘要随
// 工具链版本变化，无法在 CI 复现。因此需要二进制的夹具测试必须先把夹具复制到
// 临时目录，现场构建后端，并把 manifest 的 executable 摘要对齐到新二进制。
// 这与 `scripts/ci/reference-packages-gate.sh` 的暂存语义保持一致。
package referencefixture

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const manifestFileName = "sforum.extension.json"

// StageBackendFixture 把 fixtureRoot 复制到临时目录；若 manifest 声明了
// executable 后端且 backend/go.mod 存在，则现场构建并回写摘要，返回暂存根。
// 无可执行后端的夹具原样返回源码树路径（manifest 可直接校验，无需暂存）。
func StageBackendFixture(t *testing.T, repositoryRoot, fixtureRoot string) string {
	t.Helper()

	manifestPath := filepath.Join(fixtureRoot, manifestFileName)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read fixture manifest: %v", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse fixture manifest: %v", err)
	}
	backend, _ := manifest["backend"].(map[string]any)
	entry, _ := backend["entry"].(string)
	if entry == "" || backend["digest"] == nil {
		return fixtureRoot
	}
	if _, err := os.Stat(filepath.Join(fixtureRoot, filepath.Dir(filepath.FromSlash(entry)), "go.mod")); err != nil {
		t.Fatalf("fixture backend missing go.mod: %v", err)
	}

	packageRoot := filepath.Join(t.TempDir(), filepath.Base(fixtureRoot))
	if err := os.CopyFS(packageRoot, os.DirFS(fixtureRoot)); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}

	// 暂存副本不携带构建产物；go.mod 的 SDK replace 改写为仓库内绝对路径。
	binaryPath := filepath.Join(packageRoot, filepath.FromSlash(entry))
	_ = os.Remove(binaryPath)
	goModPath := filepath.Join(packageRoot, filepath.Dir(filepath.FromSlash(entry)), "go.mod")
	goMod, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	goMod = []byte(strings.ReplaceAll(string(goMod), "../../../../../apps/api", filepath.Join(repositoryRoot, "apps/api")))
	if err := os.WriteFile(goModPath, goMod, 0o600); err != nil {
		t.Fatal(err)
	}

	build := exec.Command("go", "build", "-mod=mod", "-trimpath", "-buildvcs=false", "-o", binaryPath, ".")
	build.Dir = filepath.Dir(binaryPath)
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fixture backend: %v\n%s", err, output)
	}
	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(binary)
	digest := hex.EncodeToString(sum[:])
	backend["digest"] = digest

	packageFiles, _ := manifest["packageFiles"].([]any)
	for _, item := range packageFiles {
		file, _ := item.(map[string]any)
		if file["path"] == entry {
			file["digest"] = digest
		}
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, manifestFileName), append(body, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return packageRoot
}

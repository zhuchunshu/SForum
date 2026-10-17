package extensionpackage

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// dev 重建循环的幂等守卫:产物/清单内容不变时,重复构建不得重写文件——
// 否则 dev watcher 会被自己的写入事件喂成「构建 → 写盘 → 事件 → 再构建」
// 的稳态循环。

func TestBuildBackendSkipsIdenticalBinary(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain unavailable")
	}
	root := t.TempDir()
	backendDir := filepath.Join(root, "backend")
	writeDevBuildFixtureFile(t, filepath.Join(backendDir, "go.mod"), "module acme.digest.plugin/backend\n\ngo 1.26\n")
	writeDevBuildFixtureFile(t, filepath.Join(backendDir, "main.go"), "package main\n\nfunc main() {}\n")

	if err := BuildBackend(root); err != nil {
		t.Fatalf("first build: %v", err)
	}
	binary := filepath.Join(backendDir, "plugin")
	first, err := os.Stat(binary)
	if err != nil {
		t.Fatal(err)
	}

	// 源码未变:重复构建必须保留现役二进制(mtime 不动)。
	if err := BuildBackend(root); err != nil {
		t.Fatalf("second build: %v", err)
	}
	second, err := os.Stat(binary)
	if err != nil {
		t.Fatal(err)
	}
	if !second.ModTime().Equal(first.ModTime()) {
		t.Fatalf("identical rebuild must not replace backend/plugin")
	}

	// 源码变化后必须替换。
	writeDevBuildFixtureFile(t, filepath.Join(backendDir, "main.go"),
		"package main\n\nimport \"os\"\n\nfunc main() { os.Exit(2) }\n")
	if err := BuildBackend(root); err != nil {
		t.Fatalf("changed build: %v", err)
	}
	third, err := os.Stat(binary)
	if err != nil {
		t.Fatal(err)
	}
	if third.ModTime().Equal(first.ModTime()) {
		t.Fatalf("changed source must replace backend/plugin")
	}
}

func TestWriteDigestSkipsUnchangedManifest(t *testing.T) {
	root := t.TempDir()
	manifest := `{
  "author": {
    "name": "SForum"
  },
  "description": "Digest idempotency fixture",
  "id": "acme.digest.plugin",
  "manifestVersion": 3,
  "name": "Digest Idempotency Fixture",
  "packageFiles": [
    {
      "id": "acme.digest.plugin.file.data",
      "kind": "asset",
      "path": "data.txt"
    }
  ],
  "sforumVersion": ">=0.1.0",
  "type": "plugin",
  "url": "https://example.com/acme.digest.plugin",
  "version": "1.0.0"
}`
	manifestPath := filepath.Join(root, "sforum.extension.json")
	writeDevBuildFixtureFile(t, manifestPath, manifest)
	writeDevBuildFixtureFile(t, filepath.Join(root, "data.txt"), "hello\n")

	if _, err := WriteDigest(root); err != nil {
		t.Fatalf("first digest: %v", err)
	}
	first, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	// 包内容未变:重复 digest 不得重写 manifest。
	if _, err := WriteDigest(root); err != nil {
		t.Fatalf("second digest: %v", err)
	}
	second, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !second.ModTime().Equal(first.ModTime()) {
		t.Fatalf("unchanged package must not rewrite manifest")
	}

	// 文件变化后必须重写(digest 刷新)。
	writeDevBuildFixtureFile(t, filepath.Join(root, "data.txt"), "changed\n")
	if _, err := WriteDigest(root); err != nil {
		t.Fatalf("changed digest: %v", err)
	}
	third, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if third.ModTime().Equal(first.ModTime()) {
		t.Fatalf("changed package content must refresh manifest")
	}
}

func writeDevBuildFixtureFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

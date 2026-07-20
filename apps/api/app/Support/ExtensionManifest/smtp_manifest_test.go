package extensionmanifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBuiltinSMTPManifestValidatesWithSchemaActions(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// apps/api/app/Support/ExtensionManifest -> repo root
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../"))
	sourceRoot := filepath.Join(root, "extensions/builtin/plugins/sforum-smtp")
	// 临时构建 + digest，避免依赖源码树 gitignored binary 是否与已提交摘要一致。
	packageRoot := filepath.Join(t.TempDir(), "sforum-smtp")
	if err := copyDirForSMTPTest(sourceRoot, packageRoot); err != nil {
		t.Fatalf("copy smtp package: %v", err)
	}
	_ = os.Remove(filepath.Join(packageRoot, "backend", "plugin"))
	binary := filepath.Join(packageRoot, "backend", "plugin")
	build := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-o", binary, ".")
	build.Dir = filepath.Join(sourceRoot, "backend")
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build smtp backend: %v\n%s", err, output)
	}
	body, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	digest := hex.EncodeToString(sum[:])
	manifestPath := filepath.Join(packageRoot, ManifestFileName)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, rewriteExecutableDigests(raw, digest), 0o644); err != nil {
		t.Fatal(err)
	}

	// LoadPackage 支持单文件与 includes 多文件；SMTP 迁移后仍走此路径。
	normalized, err := LoadPackage(packageRoot)
	if err != nil {
		t.Fatalf("load smtp package: %v", err)
	}
	if len(normalized.Settings) < 5 {
		t.Fatalf("expected smtp settings, got %d", len(normalized.Settings))
	}
	host := normalized.Settings[0]
	if host.Key != "host" {
		t.Fatalf("expected host first, got %s", host.Key)
	}
	if ResolveSettingPresentation(host, "zh-CN").Label == "" {
		t.Fatal("zh label empty")
	}
	if ResolveSettingPresentation(host, "en-US").Label == "" {
		t.Fatal("en label empty")
	}
	if normalized.SettingsDocument.UI.Layout != SettingsLayoutTabs || len(normalized.SettingsDocument.Actions) != 1 {
		t.Fatalf("expected tabbed schema + probe action: %#v", normalized.SettingsDocument)
	}
	if normalized.SettingsDocument.Actions[0].Kind != SettingsActionProviderProbe {
		t.Fatalf("unexpected smtp action: %#v", normalized.SettingsDocument.Actions[0])
	}
	// 身份文案来自 manifest/langs/zh-CN.json（按语言分文件）。
	if LocalizedDisplay(normalized, "zh-CN").Description == "" {
		t.Fatal("expected zh-CN identity description from langs include")
	}
	if LocalizedDisplay(normalized, "zh-CN").Name != "SForum SMTP" {
		t.Fatalf("unexpected zh-CN name: %q", LocalizedDisplay(normalized, "zh-CN").Name)
	}
}

func rewriteExecutableDigests(raw []byte, digest string) []byte {
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return raw
	}
	if backend, ok := root["backend"].(map[string]any); ok {
		backend["digest"] = digest
		root["backend"] = backend
	}
	if files, ok := root["packageFiles"].([]any); ok {
		for _, item := range files {
			file, ok := item.(map[string]any)
			if !ok {
				continue
			}
			path, _ := file["path"].(string)
			kind, _ := file["kind"].(string)
			if path == "backend/plugin" || kind == "executable" {
				file["digest"] = digest
			}
		}
		root["packageFiles"] = files
	}
	encoded, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return raw
	}
	return append(encoded, '\n')
}

func copyDirForSMTPTest(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

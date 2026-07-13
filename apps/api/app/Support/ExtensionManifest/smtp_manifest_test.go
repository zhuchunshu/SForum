package extensionmanifest

import (
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
	packageRoot := filepath.Join(root, "extensions/builtin/plugins/sforum-smtp")
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

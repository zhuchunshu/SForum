package pages

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// 作者侧契约（M9 参考包）：已提交的参考包 sforum-page-layout-reference 必须
// 通过真实激活预检，并且三种声明方式得到正确结果：
// 显式 bare → bare、显式 site → site、未声明 → site（平台默认）。
//
// 这里刻意走「包 → 预检」这条路，而不是再测一遍 NormalizeLayout：作者第一步
// 遇到的是包声明，不是内部函数。
func TestPageLayoutReferencePackagePreflightsWithExactLayouts(t *testing.T) {
	dir := layoutReferencePackageDir(t)
	bridge := &ExtensionBridge{}
	contributions, err := bridge.PreflightThemePackage(ThemeExtension{
		ID: "sforum.page-layout-reference", Version: "0.1.0", PackagePath: dir,
	}, "")
	if err != nil {
		t.Fatalf("reference package must preflight: %v", err)
	}
	if len(contributions) != 3 {
		t.Fatalf("contributions = %d; want 3", len(contributions))
	}
	// 预检阶段保留声明原样（未声明就是空值）；归一化到平台默认由
	// prepareContributions 单点完成。两个阶段都要断言，否则「默认 site」这条
	// 作者最关心的语义会漏掉。
	declared := map[string]PageLayout{}
	for _, contribution := range contributions {
		declared[contribution.Path] = contribution.Layout
	}
	for path, want := range map[string]PageLayout{
		"/page-layout-reference/bare":    LayoutBare,
		"/page-layout-reference/site":    LayoutSite,
		"/page-layout-reference/default": "",
	} {
		got, ok := declared[path]
		if !ok {
			t.Fatalf("page %s missing from the preflight result: %#v", path, declared)
		}
		if got != want {
			t.Fatalf("page %s preflight layout = %q; want %q", path, got, want)
		}
	}

	prepared, err := prepareContributions("sforum.page-layout-reference", contributions)
	if err != nil {
		t.Fatalf("prepareContributions: %v", err)
	}
	effective := map[string]PageLayout{}
	for _, contribution := range prepared {
		effective[contribution.Path] = contribution.Layout
	}
	for path, want := range map[string]PageLayout{
		"/page-layout-reference/bare":    LayoutBare,
		"/page-layout-reference/site":    LayoutSite,
		"/page-layout-reference/default": LayoutSite,
	} {
		if effective[path] != want {
			t.Fatalf("page %s effective layout = %q; want %q", path, effective[path], want)
		}
	}
}

// 三方身份契约（AGENTS.md 硬规则 4）：theme.json 的每个模板路径必须恰好对应一条
// templates[] 声明与一条 kind=template 的 packageFiles[] 条目，且摘要与文件一致。
func TestPageLayoutReferencePackageKeepsThreeWayTemplateIdentity(t *testing.T) {
	dir := layoutReferencePackageDir(t)
	manifest := readJSONFile(t, filepath.Join(dir, "sforum.extension.json"))
	theme := readJSONFile(t, filepath.Join(dir, "theme.json"))

	declared := map[string]int{}
	for _, raw := range manifest["templates"].([]any) {
		entry := raw.(map[string]any)
		path := entry["path"].(string)
		declared[path]++
		digest, _ := entry["digest"].(string)
		if digest != fileDigest(t, filepath.Join(dir, path)) {
			t.Fatalf("templates[] digest for %s does not match the file", path)
		}
	}
	packaged := map[string]int{}
	for _, raw := range manifest["packageFiles"].([]any) {
		entry := raw.(map[string]any)
		if entry["kind"] != "template" {
			continue
		}
		path := entry["path"].(string)
		packaged[path]++
		digest, _ := entry["digest"].(string)
		if digest != fileDigest(t, filepath.Join(dir, path)) {
			t.Fatalf("packageFiles[] digest for %s does not match the file", path)
		}
	}

	for _, raw := range theme["pages"].([]any) {
		page := raw.(map[string]any)
		path := page["template"].(string)
		if declared[path] != 1 {
			t.Fatalf("theme.json template %s has %d templates[] declarations; want exactly 1", path, declared[path])
		}
		if packaged[path] != 1 {
			t.Fatalf("theme.json template %s has %d packageFiles[] entries; want exactly 1", path, packaged[path])
		}
	}
}

func layoutReferencePackageDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "..", "..", "extensions", "fixtures", "plugins", "sforum-page-layout-reference")
	absolute, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(absolute, "theme.json")); err != nil {
		t.Fatalf("reference package missing at %s: %v", absolute, err)
	}
	return absolute
}

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func fileDigest(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

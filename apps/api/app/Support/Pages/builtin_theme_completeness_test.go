package pages

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	themecompiler "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeCompiler"
)

// TestBuiltinThemesCoverAllReplaceablePages is the P13 reference-theme gate:
// default and nocturne must declare every replaceable Page Registry id, ship
// the template file, embed the required host body island, and compile.
func TestBuiltinThemesCoverAllReplaceablePages(t *testing.T) {
	repoRoot := themeCompletenessRepoRoot(t)
	for _, themeRel := range []string{
		"extensions/builtin/themes/sforum-default",
		"extensions/builtin/themes/sforum-nocturne",
	} {
		themeRel := themeRel
		t.Run(themeRel, func(t *testing.T) {
			root := filepath.Join(repoRoot, themeRel)
			pkg, err := LoadThemePackage(root)
			if err != nil {
				t.Fatalf("load theme package: %v", err)
			}
			byTarget := make(map[string]ThemePageDecl, len(pkg.Pages))
			for _, page := range pkg.Pages {
				if strings.EqualFold(strings.TrimSpace(page.Action), string(ActionReplace)) {
					byTarget[strings.TrimSpace(page.Target)] = page
				}
			}
			islands := productionThemeIslandBindings()
			compiler := themecompiler.NewCompiler(themecompiler.Limits{})
			bindings := themecompiler.Bindings{
				BindingRevision: strings.Repeat("c", 64),
				Islands:         islands,
				SiteName:        "SForum",
				PageViewModels:  map[string]themecompiler.PageTemplateBinding{},
			}
			// 仅编译 theme.json 声明的页模板；plugin override 不走 Page ViewModel 绑定。
			selected := map[string]struct{}{}
			for _, page := range Catalog() {
				if !page.Replaceable {
					continue
				}
				decl, ok := byTarget[page.ID]
				if !ok {
					t.Fatalf("missing replace declaration for %s", page.ID)
				}
				if strings.TrimSpace(decl.Template) == "" {
					t.Fatalf("%s has empty template path", page.ID)
				}
				if strings.TrimSpace(decl.Contract) != page.ContractVersion {
					t.Fatalf("%s contract=%q want %q", page.ID, decl.Contract, page.ContractVersion)
				}
				absTemplate := filepath.Join(root, filepath.FromSlash(decl.Template))
				raw, err := os.ReadFile(absTemplate)
				if err != nil {
					t.Fatalf("read %s: %v", decl.Template, err)
				}
				tag := RequiredThemeBodyIslandTag(page.ID)
				if tag == "" {
					t.Fatalf("no body island mapping for replaceable page %s", page.ID)
				}
				if !strings.Contains(string(raw), "<"+tag) {
					t.Fatalf("%s template %s missing required island <%s>", page.ID, decl.Template, tag)
				}
				bindings.PageViewModels[decl.Template] = themecompiler.PageTemplateBinding{
					PageID: page.ID, SchemaVersion: page.ContractVersion,
				}
				selected[decl.Template] = struct{}{}
			}
			// Nocturne must ship at least one plugin template override.
			if strings.Contains(themeRel, "nocturne") {
				override := filepath.Join(root, "templates", "plugins", "sforum.seo-reference", "status-badge.html")
				if _, err := os.Stat(override); err != nil {
					t.Fatalf("nocturne missing plugin override template: %v", err)
				}
				if _, err := os.Stat(filepath.Join(root, "manifest", "settings.json")); err != nil {
					t.Fatalf("nocturne missing settings document: %v", err)
				}
			}
			// Default must ship shared layouts/partials for chrome reuse.
			if strings.Contains(themeRel, "sforum-default") {
				for _, rel := range []string{"layouts/base.html", "partials/chrome-start.html", "partials/chrome-end.html"} {
					if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
						t.Fatalf("default missing %s: %v", rel, err)
					}
				}
			}
			digest := strings.Repeat("a", 64)
			snapshot, err := compiler.CompileFS(selectedThemeFS{FS: os.DirFS(root), selected: selected}, digest, bindings)
			if err != nil {
				t.Fatalf("compile theme package: %v", err)
			}
			for _, page := range Catalog() {
				if !page.Replaceable {
					continue
				}
				decl := byTarget[page.ID]
				model, err := BuildCorePageViewModel(CorePageViewModelRequest{
					PageID: page.ID, Locale: "zh-CN", Path: page.PathPattern,
					SEO: themecompiler.PageSEOView{Title: page.ID},
				})
				if err != nil {
					t.Fatalf("viewmodel %s: %v", page.ID, err)
				}
				bound, err := themecompiler.CorePageViewModelRegistry().Bind(
					page.ID, page.ContractVersion, digest, model,
				)
				if err != nil {
					t.Fatalf("bind %s: %v", page.ID, err)
				}
				output, err := snapshot.Render(context.Background(), decl.Template, bound)
				if err != nil || len(output.HTMLSegments()) == 0 {
					t.Fatalf("render %s: segments=%d err=%v", page.ID, len(output.HTMLSegments()), err)
				}
			}
		})
	}
}

func themeCompletenessRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// apps/api/app/Support/Pages → repo root is five levels up
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
}

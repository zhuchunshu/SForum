package pages

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// TestCatalogCoversAllPublicNuxtPages 是页面目录的防漂移审计:
// apps/web/app/pages 下每个非豁免页面文件必须对应一条目录 CoreComponent,
// 反向每条非 Virtual 目录条目必须有页面文件,且页面壳 SFPageOutlet 的 page id 与目录一致。
func TestCatalogCoversAllPublicNuxtPages(t *testing.T) {
	repoRoot := themeCompletenessRepoRoot(t)
	pagesRoot := filepath.Join(repoRoot, "apps", "web", "app", "pages")

	// 豁免:admin 后台(非扩展面)、Registry 驱动的扩展 add 页 catch-all 宿主。
	exemptDirs := []string{"admin", "x"}
	exemptFiles := map[string]bool{"[...sfRegistryPage].vue": true}

	byComponent := map[string]PageDefinition{}
	for _, definition := range Catalog() {
		byComponent[definition.CoreComponent] = definition
	}

	outletPattern := regexp.MustCompile(`<SFPageOutlet\s+page="([^"]+)"`)
	seenComponents := map[string]bool{}

	err := filepath.WalkDir(pagesRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(pagesRoot, path)
		if relErr != nil {
			return relErr
		}
		if entry.IsDir() {
			if slices.Contains(exemptDirs, rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".vue") || exemptFiles[rel] {
			return nil
		}
		component := "pages/" + strings.TrimSuffix(filepath.ToSlash(rel), ".vue")
		definition, registered := byComponent[component]
		if !registered {
			t.Errorf("page file %s has no catalog entry (CoreComponent %q); register it in Support/Pages/catalog.go or add an exemption here", rel, component)
			return nil
		}
		seenComponents[component] = true

		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		match := outletPattern.FindStringSubmatch(string(raw))
		if match == nil {
			t.Errorf("page file %s must render <SFPageOutlet page=%q>", rel, definition.ID)
			return nil
		}
		if match[1] != definition.ID {
			t.Errorf("page file %s declares SFPageOutlet page=%q but catalog id is %q", rel, match[1], definition.ID)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk pages: %v", err)
	}

	for _, definition := range Catalog() {
		if definition.Virtual || !strings.HasPrefix(definition.CoreComponent, "pages/") {
			continue
		}
		if strings.HasPrefix(definition.CoreComponent, "pages/admin/") {
			continue
		}
		if !seenComponents[definition.CoreComponent] {
			t.Errorf("catalog entry %s references missing page file %s.vue", definition.ID, definition.CoreComponent)
		}
	}
}

func TestNuxtParentPagesDoNotShadowNestedRoutes(t *testing.T) {
	repoRoot := themeCompletenessRepoRoot(t)
	pagesRoot := filepath.Join(repoRoot, "apps", "web", "app", "pages")
	pageFiles := map[string]string{}

	err := filepath.WalkDir(pagesRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".vue") {
			return nil
		}
		rel, relErr := filepath.Rel(pagesRoot, path)
		if relErr != nil {
			return relErr
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		pageFiles[filepath.ToSlash(rel)] = string(raw)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for parent, raw := range pageFiles {
		prefix := strings.TrimSuffix(parent, ".vue") + "/"
		for child := range pageFiles {
			if strings.HasPrefix(child, prefix) && !strings.Contains(raw, "<NuxtPage") {
				t.Errorf("Nuxt parent page %s shadows nested route %s; move the index route to %sindex.vue or render <NuxtPage>", parent, child, prefix)
			}
		}
	}
}

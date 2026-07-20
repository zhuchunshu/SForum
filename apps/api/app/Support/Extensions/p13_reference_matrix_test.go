package extensionsruntime_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestP13ReferencePluginPackagesExist is the installable-package inventory gate
// for the five P13 reference classes. Each package must ship a Manifest V3
// template (or manifest) under fixtures and/or dev without requiring core
// product edits.
func TestP13ReferencePluginPackagesExist(t *testing.T) {
	root := p13RepoRoot(t)
	// extensions/dev 被 .gitignore 排除；仓库权威参考包在 fixtures（与 CI 子进程门禁同源）。
	required := map[string][]string{
		"seo": {
			"extensions/fixtures/plugins/sforum-seo-reference/sforum.extension.json.tmpl",
		},
		"identity": {
			"extensions/fixtures/plugins/sforum-membership-reference/sforum.extension.json.tmpl",
		},
		"custom-content": {
			"extensions/fixtures/plugins/sforum-custom-content/sforum.extension.json.tmpl",
		},
		"media": {
			"extensions/fixtures/plugins/sforum-media-optimize/sforum.extension.json.tmpl",
		},
		"commerce": {
			"extensions/fixtures/plugins/sforum-commerce-workflow/sforum.extension.json.tmpl",
			"extensions/fixtures/plugins/sforum-commerce-workflow-ext/sforum.extension.json.tmpl",
		},
	}
	surfaceHints := map[string][]string{
		"seo":            {`"seo"`, `"kind": "title"`, `"kind": "jsonld"`},
		"identity":       {`"identity"`, `"permissionDefinitions"`, `"assignmentPolicy"`},
		"custom-content": {`"entities"`, `"content"`, `"queries"`},
		"media":          {`"media"`},
		"commerce":       {`"routes"`, `"database"`, `"hooks"`, `"jobs"`},
	}
	for class, paths := range required {
		found := false
		var body string
		for _, rel := range paths {
			abs := filepath.Join(root, rel)
			raw, err := os.ReadFile(abs)
			if err != nil {
				continue
			}
			found = true
			body = string(raw)
			// tmpl 不是完整 JSON（含 __BACKEND_DIGEST__）；至少是 object 形。
			if !strings.Contains(body, `"manifestVersion"`) && !strings.Contains(body, `"id"`) {
				t.Fatalf("%s: %s missing manifest markers", class, rel)
			}
			break
		}
		if !found {
			t.Fatalf("P13 %s reference package missing; tried %v", class, paths)
		}
		for _, hint := range surfaceHints[class] {
			if !strings.Contains(body, strings.Trim(hint, `"`)) && !strings.Contains(body, hint) {
				// 宽松：提示串可能带引号
				if !strings.Contains(body, strings.ReplaceAll(hint, `"`, "")) {
					t.Fatalf("P13 %s package missing surface hint %s", class, hint)
				}
			}
		}
	}
	// 主题完整性由 Pages.TestBuiltinThemesCoverAllReplaceablePages 负责；这里只确认包在仓库中。
	for _, theme := range []string{
		"extensions/builtin/themes/sforum-default/theme.json",
		"extensions/builtin/themes/sforum-nocturne/theme.json",
	} {
		raw, err := os.ReadFile(filepath.Join(root, theme))
		if err != nil {
			t.Fatalf("theme package: %v", err)
		}
		var pkg struct {
			Pages []struct {
				Target string `json:"target"`
				Action string `json:"action"`
			} `json:"pages"`
		}
		if err := json.Unmarshal(raw, &pkg); err != nil {
			t.Fatalf("theme json %s: %v", theme, err)
		}
		if len(pkg.Pages) < 20 {
			t.Fatalf("%s must cover replaceable public pages, got %d", theme, len(pkg.Pages))
		}
	}
}

func p13RepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../"))
}

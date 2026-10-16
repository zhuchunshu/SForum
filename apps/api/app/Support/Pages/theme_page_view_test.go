package pages

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// 公开页正文的提供方式：view=template（默认）| view=component。
// 未知 view 必须 fail closed，作者写错时立刻失败而不是静默退化成模板页。
func TestThemePageViewDeclarationContract(t *testing.T) {
	cases := []struct {
		name    string
		page    map[string]any
		wantErr bool
	}{
		{
			name: "default template view",
			page: map[string]any{"id": "demo.page.t", "action": "add", "path": "/demo", "template": "templates/a.html"},
		},
		{
			name: "explicit template view",
			page: map[string]any{"id": "demo.page.t2", "action": "add", "path": "/demo2", "template": "templates/a.html", "view": "template"},
		},
		{
			name: "component view",
			page: map[string]any{"id": "demo.page.c", "action": "add", "path": "/demo3", "view": "component", "component": map[string]any{"id": "demo.component.card"}},
		},
		{
			name: "schema view",
			page: map[string]any{
				"id": "demo.page.s", "action": "add", "path": "/demo-s", "view": "schema",
				"schema": map[string]any{
					"descriptor": "demo.file.page.list", "contract": "demo.schema.page.list@1",
				},
			},
		},
		{
			name:    "schema view without descriptor and contract",
			page:    map[string]any{"id": "demo.page.s2", "action": "add", "path": "/demo-s2", "view": "schema"},
			wantErr: true,
		},
		{
			name: "schema view with template",
			page: map[string]any{
				"id": "demo.page.s3", "action": "add", "path": "/demo-s3", "template": "templates/a.html",
				"view": "schema",
				"schema": map[string]any{
					"descriptor": "demo.file.page.list", "contract": "demo.schema.page.list@1",
				},
			},
			wantErr: true,
		},
		{
			name: "schema view with component",
			page: map[string]any{
				"id": "demo.page.s4", "action": "add", "path": "/demo-s4", "view": "schema",
				"component": map[string]any{"id": "demo.component.card"},
				"schema": map[string]any{
					"descriptor": "demo.file.page.list", "contract": "demo.schema.page.list@1",
				},
			},
			wantErr: true,
		},
		{
			name: "component view with schema descriptor",
			page: map[string]any{
				"id": "demo.page.s5", "action": "add", "path": "/demo-s5", "view": "component",
				"component": map[string]any{"id": "demo.component.card"},
				"schema": map[string]any{
					"descriptor": "demo.file.page.list", "contract": "demo.schema.page.list@1",
				},
			},
			wantErr: true,
		},
		{
			name:    "unknown view fails closed",
			page:    map[string]any{"id": "demo.page.x", "action": "add", "path": "/demo4", "template": "templates/a.html", "view": "vue"},
			wantErr: true,
		},
		{
			name:    "component view without component id",
			page:    map[string]any{"id": "demo.page.c2", "action": "add", "path": "/demo5", "view": "component"},
			wantErr: true,
		},
		{
			name:    "component view with template",
			page:    map[string]any{"id": "demo.page.c3", "action": "add", "path": "/demo6", "template": "templates/a.html", "view": "component", "component": map[string]any{"id": "demo.component.card"}},
			wantErr: true,
		},
		{
			name:    "template view with component",
			page:    map[string]any{"id": "demo.page.c4", "action": "add", "path": "/demo7", "template": "templates/a.html", "component": map[string]any{"id": "demo.component.card"}},
			wantErr: true,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			root := t.TempDir()
			body, err := json.Marshal(map[string]any{"pages": []any{testCase.page}})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "theme.json"), body, 0o644); err != nil {
				t.Fatal(err)
			}
			_, err = LoadThemePackage(root)
			if testCase.wantErr && err == nil {
				t.Fatal("expected the declaration to be rejected")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("declaration rejected: %v", err)
			}
		})
	}
}

// 贡献必须把 view/component 透传下去，否则解析层无从判断正文来源。
func TestContributionsFromThemeCarryPageViewAndComponent(t *testing.T) {
	pkg := ThemePackage{Pages: []ThemePageDecl{
		{ID: "demo.page.c", Action: "add", Path: "/demo", View: "component", Component: &ThemePageComponent{ID: " demo.component.card "}},
		{ID: "demo.page.t", Action: "add", Path: "/demo2", Template: "templates/a.html"},
	}}
	contribs := ContributionsFromTheme("demo.ext", "1.0.0", "digest", pkg)
	if len(contribs) != 2 {
		t.Fatalf("contributions = %d", len(contribs))
	}
	if contribs[0].View != PageViewComponent || contribs[0].ComponentID != "demo.component.card" {
		t.Fatalf("component page lost its view/component: %#v", contribs[0])
	}
	if contribs[1].View != PageViewTemplate || contribs[1].ComponentID != "" {
		t.Fatalf("template page gained a component view: %#v", contribs[1])
	}
}

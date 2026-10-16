package themecompiler

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
)

var iconCommentPattern = regexp.MustCompile(`(?s)<!--.*?-->`)

func TestThemeTemplateIconHelperEmitsHostIconPlaceholder(t *testing.T) {
	snapshot := compileTestSnapshot(t, fstest.MapFS{
		"templates/home.html": &fstest.MapFile{Data: []byte(
			`<h1>{{icon "i-lucide-inbox"}}Inbox</h1>` +
				`<h2>{{icon "i-tabler-brand-github"}}</h2>` +
				`<h3>{{icon "i-mdi-home"}}</h3>` +
				`<h4>{{icon ""}}</h4>`,
		)},
	}, testDigest('i'), Bindings{BindingRevision: testDigest('0')}, Limits{})

	output, err := snapshot.renderPassive(context.Background(), "templates/home.html", map[string]any{})
	if err != nil {
		t.Fatalf("an unsupported icon name must not fail the render: %v", err)
	}

	// 合法名称输出占位元素；SVG 由客户端运行时注入，服务端绝不内联图标集。
	for _, expected := range []string{
		`<span class="sf-host-icon" data-sf-icon="i-lucide-inbox" aria-hidden="true"></span>`,
		`<span class="sf-host-icon" data-sf-icon="i-tabler-brand-github" aria-hidden="true"></span>`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("render missing %q: %s", expected, output)
		}
	}
	// 非法名称保留一条可发现的注释，而不是静默或 500。
	if !strings.Contains(output, `<!-- sforum.icon: unsupported name "i-mdi-home" -->`) {
		t.Fatalf("invalid icon name must leave a diagnosable comment: %s", output)
	}
	// 空名称不产生任何输出。
	if strings.Contains(output, "sforum.icon: unsupported name \"\"") {
		t.Fatalf("empty icon name must produce nothing: %s", output)
	}
}

func TestThemeTemplateIconHelperNeverAllowsCommentBreakout(t *testing.T) {
	// 敌意名称只能来自数据：模板源码里出现 "<script>" 会被静态检查直接拒绝，
	// 这正是主题编译器的第一道防线。这里验证第二道——helper 自身的转义。
	snapshot := compileTestSnapshot(t, fstest.MapFS{
		"templates/home.html": &fstest.MapFile{Data: []byte(`<p>{{icon .Name}}</p>`)},
	}, testDigest('j'), Bindings{BindingRevision: testDigest('0')}, Limits{})

	hostile := []string{
		`i-lucide-x--><script>alert(1)</script>`,
		`i-lucide-<img src=x onerror=alert(1)>`,
		`i-lucide-x" onload="alert(1)`,
		`../../etc/passwd`,
	}
	for _, name := range hostile {
		output, err := snapshot.renderPassive(context.Background(), "templates/home.html", map[string]any{"Name": name})
		if err != nil {
			t.Fatalf("hostile icon name must not fail the render: %v", err)
		}
		// 把注释整体剥离后，敌意名称不得贡献任何标记：剩下的只应是模板自身的 <p></p>。
		stripped := iconCommentPattern.ReplaceAllString(output, "")
		if stripped != "<p></p>" {
			t.Fatalf("hostile icon name %q contributed markup: %s", name, output)
		}
		// 注释必须恰好一对：名称里的 ">" 已被替换，无法提前闭合注释。
		if strings.Count(output, "<!--") != 1 || strings.Count(output, "-->") != 1 {
			t.Fatalf("icon helper allowed comment breakout for %q: %s", name, output)
		}
		if !strings.Contains(output, "sforum.icon: unsupported name") {
			t.Fatalf("expected a diagnosable comment for %q: %s", name, output)
		}
	}
}

func TestThemeTemplateInspectionAllowsTheIconHelper(t *testing.T) {
	// 模板表达式里的函数名受白名单约束；icon 必须在其中，否则主题会被判不安全。
	if _, ok := allowedFunctions["icon"]; !ok {
		t.Fatal("icon helper is missing from the inspected function allowlist")
	}
	var bindings Bindings
	if _, ok := restrictedFuncMap(bindings)["icon"]; !ok {
		t.Fatal("icon helper is missing from the template func map")
	}
}

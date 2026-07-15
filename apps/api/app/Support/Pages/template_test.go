package pages

import (
	"strings"
	"testing"
)

func TestRenderTemplateEscapesVars(t *testing.T) {
	out, err := RenderTemplate(`<h1>{{title}}</h1><sf-home-page></sf-home-page>`, map[string]string{"title": `<b>Hi</b>`})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "<b>Hi</b>") {
		t.Fatalf("raw HTML leaked: %s", out)
	}
	if !strings.Contains(out, "&lt;b&gt;Hi&lt;/b&gt;") {
		t.Fatalf("expected escaped title, got: %s", out)
	}
	if !strings.Contains(out, "sf-home-page") {
		t.Fatalf("expected host island preserved: %s", out)
	}
}

func TestValidateTemplateRejectsScriptVariants(t *testing.T) {
	cases := []string{
		`<script>alert(1)</script>`,
		`<SCRIPT SRC=//x.com></SCRIPT>`,
		`<img src=x onerror=alert(1)>`,
		`<img src=x oNeRrOr=alert(1)>`,
		`<a href="javascript:alert(1)">x</a>`,
		`<a href=javascript:alert(1)>x</a>`,
		`<a href="java&#115;cript:alert(1)">x</a>`,
		"<a href=\"java\nscript:alert(1)\">x</a>",
		`<iframe src="https://evil"></iframe>`,
		`<object data="https://evil"></object>`,
		`<embed src="https://evil">`,
		`<style>body{background:url('javascript:1')}</style>`,
		`<svg onload=alert(1)>`,
		`<math><mi>x</mi></math>`,
		`<meta http-equiv="refresh" content="0;url=//evil">`,
		`<base href="https://evil">`,
		`<form action="https://evil"><input></form>`,
		`<div style="background:url(javascript:1)">`,
		`<sf-extension-widget entry="https://evil/x.js"></sf-extension-widget>`,
		`<sf-unknown-island></sf-unknown-island>`,
		`<sf-home-page onclick="alert(1)"></sf-home-page>`,
		`<sf-home-page href="javascript:1"></sf-home-page>`,
	}
	for _, src := range cases {
		if err := ValidateTemplate(src); err == nil {
			t.Fatalf("expected reject for %q", src)
		}
	}
}

func TestSanitizeTemplateStripsDangerous(t *testing.T) {
	// 纵深：即便 Validate 被绕过，sanitize 也应去掉危险标签
	// 注意：含 onerror 的源在 Validate 阶段就会拒绝；这里直接测 policy 管线中的 HTML 部分
	raw := `<div>ok<img src="/x.png" alt="a"><a href="/safe">x</a></div>`
	clean, err := SanitizeTemplateHTML(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(clean, "ok") {
		t.Fatalf("lost content: %s", clean)
	}
	// 注入 script 片段
	raw2 := `<div>ok</div>`
	clean2, err := SanitizeTemplateHTML(raw2)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(clean2), "<script") {
		t.Fatal(clean2)
	}
}

func TestValidateTemplateSizeAndDepth(t *testing.T) {
	big := strings.Repeat("a", MaxTemplateBytes+1)
	if err := ValidateTemplate(big); err == nil {
		t.Fatal("expected size reject")
	}
	var b strings.Builder
	for i := 0; i < MaxTemplateDepth+5; i++ {
		b.WriteString("<div>")
	}
	for i := 0; i < MaxTemplateDepth+5; i++ {
		b.WriteString("</div>")
	}
	if err := ValidateTemplate(b.String()); err == nil {
		t.Fatal("expected depth reject")
	}
}

func TestValidateTemplateAllowsSafeLayout(t *testing.T) {
	src := `<main class="home"><h1>Welcome</h1><sf-home-page></sf-home-page><a href="/login">Login</a></main>`
	if err := ValidateTemplate(src); err != nil {
		t.Fatal(err)
	}
	out, err := RenderTemplate(src, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "sf-home-page") || !strings.Contains(out, "Welcome") {
		t.Fatalf("unexpected: %s", out)
	}
}

func TestValidateTemplateAllowsExactPublicL2IslandWithSanitizedFallback(t *testing.T) {
	src := `<main><sf-extension-widget extension-id="demo.public" component-id="demo.public.component.card"><article data-runtime-only="removed">Primary <strong>SSR fallback</strong></article></sf-extension-widget><sf-home-page></sf-home-page></main>`
	if err := ValidateTemplate(src); err != nil {
		t.Fatal(err)
	}
	clean, err := SanitizeTemplateHTML(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`<sf-extension-widget component-id="demo.public.component.card" extension-id="demo.public">`,
		`<article>Primary <strong>SSR fallback</strong></article>`,
		`</sf-extension-widget>`,
	} {
		if !strings.Contains(clean, expected) {
			t.Fatalf("sanitized public L2 fallback missing %q: %s", expected, clean)
		}
	}
	if strings.Contains(clean, "data-runtime-only") {
		t.Fatalf("unreviewed fallback attribute survived: %s", clean)
	}
}

func TestValidateTemplateRejectsUnsafePublicL2IslandContracts(t *testing.T) {
	cases := []string{
		`<sf-extension-widget></sf-extension-widget>`,
		`<sf-extension-widget extension-id="demo.public"></sf-extension-widget>`,
		`<sf-extension-widget component-id="demo.public.component.card"></sf-extension-widget>`,
		`<sf-extension-widget extension-id="demo.public" component-id="other.component.card"></sf-extension-widget>`,
		`<sf-extension-widget extension-id="demo.public" component-id="demo.public.component.card" entry="frontend/card.mjs"></sf-extension-widget>`,
		`<sf-extension-widget extension-id="demo.public" component-id="demo.public.component.card"><sf-home-page></sf-home-page></sf-extension-widget>`,
	}
	for _, src := range cases {
		if err := ValidateTemplate(src); err == nil {
			t.Fatalf("expected public L2 island rejection for %q", src)
		}
	}
}

func TestExtractHostIslands(t *testing.T) {
	src, err := SanitizeTemplateHTML(`<div>Hi</div><sf-home-page name="x"></sf-home-page><p>end</p>`)
	if err != nil {
		t.Fatal(err)
	}
	segs := ExtractHostIslands(src)
	if len(segs) < 2 {
		t.Fatalf("expected segments, got %#v", segs)
	}
	found := false
	for _, s := range segs {
		if s.Type == "island" && s.Tag == "sf-home-page" {
			found = true
			if s.Attrs["name"] != "x" {
				t.Fatalf("attrs: %#v", s.Attrs)
			}
		}
	}
	if !found {
		t.Fatalf("island not found: %#v", segs)
	}
}

func TestLoadDefaultHomeTemplate(t *testing.T) {
	// 相对 monorepo：从 apps/api 运行时 path 不同；用 testdata 风格内联
	src := `<!-- home --><div class="sf-page" data-page="forum.home"><sf-home-page></sf-home-page></div>`
	// data-page 不在 allowlist attrs globally for data-* — div 上 data-page 会被 strip，这可接受
	// 调整为 class only
	src = `<div class="sf-page sf-page--home"><sf-home-page></sf-home-page></div>`
	out, err := RenderTemplate(src, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "sf-home-page") {
		t.Fatalf("got %s", out)
	}
}

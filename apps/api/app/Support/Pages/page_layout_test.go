package pages

import (
	"errors"
	"testing"
)

// layout 的默认值是 site：公开页共享同一几何契约，而插件公开页此前是全站唯一没有
// navbar/footer 的页面。实测 5 个插件页模板无一挂载全局 chrome（对照：主题模板 34 个
// 全部挂载），因此默认补上外壳不会造成双层导航。作者要自持 chrome 时显式声明 bare。
func TestNormalizeLayoutDefaultsToSite(t *testing.T) {
	got, err := NormalizeLayout("")
	if err != nil {
		t.Fatalf("NormalizeLayout(\"\") error: %v", err)
	}
	if got != LayoutSite {
		t.Fatalf("NormalizeLayout(\"\") = %q; want %q", got, LayoutSite)
	}
}

func TestNormalizeLayoutAcceptsKnownValuesCaseInsensitively(t *testing.T) {
	cases := []struct {
		raw  string
		want PageLayout
	}{
		{"site", LayoutSite},
		{"SITE", LayoutSite},
		{"  Site  ", LayoutSite},
		{"bare", LayoutBare},
		{"BARE", LayoutBare},
	}
	for _, tc := range cases {
		got, err := NormalizeLayout(tc.raw)
		if err != nil {
			t.Fatalf("NormalizeLayout(%q) error: %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("NormalizeLayout(%q) = %q; want %q", tc.raw, got, tc.want)
		}
	}
}

// 未知非空值必须 fail closed，不得静默按 bare 或 site 放行。
func TestNormalizeLayoutRejectsUnknownValue(t *testing.T) {
	if _, err := NormalizeLayout("fancy"); !errors.Is(err, ErrInvalidLayout) {
		t.Fatalf("NormalizeLayout(\"fancy\") error = %v; want ErrInvalidLayout", err)
	}
}

// 与 ValidAccess 语义保持一致：空值可被规范化，因此视为合法。
func TestValidLayout(t *testing.T) {
	if !ValidLayout(LayoutSite) || !ValidLayout(LayoutBare) || !ValidLayout("") {
		t.Fatalf("known or empty layout reported invalid")
	}
	if ValidLayout("fancy") {
		t.Fatalf("unknown layout reported valid")
	}
}

func TestContributionsFromThemeCarriesLayout(t *testing.T) {
	pkg := ThemePackage{Pages: []ThemePageDecl{
		{ID: "demo.docs", Action: "add", Path: "/demo", Template: "templates/docs.html", Layout: "site"},
		{ID: "demo.members", Action: "add", Path: "/demo-members", Template: "templates/members.html"},
	}}
	out := ContributionsFromTheme("demo.theme", "1.0.0", "digest", pkg)
	if len(out) != 2 {
		t.Fatalf("contributions = %d; want 2", len(out))
	}
	if out[0].Layout != LayoutSite {
		t.Fatalf("declared layout lost: got %q; want %q", out[0].Layout, LayoutSite)
	}
	// 未声明时保留空值，归一化交给 prepareContributionsFor 单点完成。
	if out[1].Layout != "" {
		t.Fatalf("undeclared layout = %q; want empty before normalization", out[1].Layout)
	}
}

func TestPrepareContributionsNormalizesAndRejectsLayout(t *testing.T) {
	base := PageContribution{
		ID:       "demo.docs",
		Action:   ActionAdd,
		Path:     "/demo",
		Template: "templates/docs.html",
		Access:   AccessPublic,
	}

	got, err := prepareContributions("demo.theme", []PageContribution{base})
	if err != nil {
		t.Fatalf("prepareContributions error: %v", err)
	}
	if got[0].Layout != LayoutSite {
		t.Fatalf("undeclared layout = %q; want platform default %q", got[0].Layout, LayoutSite)
	}

	bare := base
	bare.Layout = LayoutBare
	got, err = prepareContributions("demo.theme", []PageContribution{bare})
	if err != nil {
		t.Fatalf("prepareContributions(bare) error: %v", err)
	}
	if got[0].Layout != LayoutBare {
		t.Fatalf("explicit bare layout = %q; want %q", got[0].Layout, LayoutBare)
	}

	bad := base
	bad.Layout = "fancy"
	if _, err := prepareContributions("demo.theme", []PageContribution{bad}); !errors.Is(err, ErrInvalidLayout) {
		t.Fatalf("unknown layout error = %v; want ErrInvalidLayout", err)
	}
}

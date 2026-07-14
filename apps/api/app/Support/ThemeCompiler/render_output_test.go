package themecompiler

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
)

func TestRenderOutputReturnsOrderedTypedSegmentsAndHostSEO(t *testing.T) {
	files := fstest.MapFS{
		"templates/home.html": &fstest.MapFile{Data: []byte(
			`<main>{{.Topics | len}} topics</main><sf-topic-list page="{{.Base.Pagination.Page}}" compact="true" link="/topics"></sf-topic-list><footer>After</footer>`,
		)},
	}
	bindings := Bindings{
		BindingRevision: viewModelBindingRevision,
		PageViewModels: map[string]PageTemplateBinding{
			"templates/home.html": {PageID: "forum.home", SchemaVersion: "sforum.page.home@1"},
		},
		Islands: map[string]IslandBinding{
			"sf-topic-list": {
				ComponentID: "forum.component.topic_list",
				Props: []IslandPropContract{
					{Name: "page", Type: IslandPropInteger, Required: true},
					{Name: "compact", Type: IslandPropBoolean},
					{Name: "link", Type: IslandPropURL},
				},
			},
		},
	}
	snapshot, err := NewCompiler(Limits{}).CompileFS(files, viewModelThemeDigest, bindings)
	if err != nil {
		t.Fatal(err)
	}
	model := validHomeViewModel()
	model.Base.Pagination = &PaginationView{Page: 2, PageSize: 20, Total: 42}
	model.Base.SEO = PageSEOView{
		Title: "Host title", CanonicalURL: "https://example.com/",
		AlternateLinks: []AlternateLink{{Locale: "zh-CN", URL: "https://example.com/zh-CN"}},
		StructuredData: []StructuredDataView{{Kind: "WebSite", Name: "SForum"}},
	}
	bound, err := CorePageViewModelRegistry().Bind("forum.home", "sforum.page.home@1", viewModelThemeDigest, model)
	if err != nil {
		t.Fatal(err)
	}
	model.Base.SEO.Title = "mutated"
	model.Base.SEO.AlternateLinks[0].URL = "https://attacker.invalid"

	output, err := snapshot.Render(context.Background(), "templates/home.html", bound)
	if err != nil {
		t.Fatal(err)
	}
	segments := output.HTMLSegments()
	if len(segments) != 3 {
		t.Fatalf("segments = %#v", segments)
	}
	if segments[0].String() != "<main>1 topics</main>" || segments[2].String() != "<footer>After</footer>" ||
		!strings.Contains(segments[1].String(), `data-sforum-island="forum.component.topic_list:1"`) {
		t.Fatalf("HTML segments = %#v", segments)
	}
	islands := output.Islands()
	if len(islands) != 1 {
		t.Fatalf("islands = %#v", islands)
	}
	island := islands[0]
	if island.ID != "forum.component.topic_list:1" || island.ComponentID != "forum.component.topic_list" || len(island.Props) != 3 {
		t.Fatalf("island = %#v", island)
	}
	if island.Props[0].Name != "compact" || !island.Props[0].BooleanValue ||
		island.Props[1].Name != "link" || island.Props[1].StringValue != "/topics" ||
		island.Props[2].Name != "page" || island.Props[2].IntegerValue != 2 {
		t.Fatalf("typed props = %#v", island.Props)
	}
	islands[0].Props[0].Name = "mutated"
	if output.Islands()[0].Props[0].Name != "compact" {
		t.Fatal("island getter exposed mutable output state")
	}
	seo := output.SEO()
	if seo.Title != "Host title" || seo.AlternateLinks[0].URL != "https://example.com/zh-CN" || seo.StructuredData[0].Kind != "WebSite" {
		t.Fatalf("SEO = %#v", seo)
	}
	seo.AlternateLinks[0].URL = "mutated"
	if output.SEO().AlternateLinks[0].URL != "https://example.com/zh-CN" {
		t.Fatal("SEO getter exposed mutable output state")
	}
}

func TestCompilerAndRenderRejectUndeclaredOrInvalidIslands(t *testing.T) {
	compiler := NewCompiler(Limits{})
	baseBindings := Bindings{
		BindingRevision: viewModelBindingRevision,
		PageViewModels: map[string]PageTemplateBinding{
			"templates/home.html": {PageID: "forum.home", SchemaVersion: "sforum.page.home@1"},
		},
	}
	if _, err := compiler.CompileFS(fstest.MapFS{
		"templates/home.html": &fstest.MapFile{Data: []byte(`<sf-unknown></sf-unknown>`)},
	}, viewModelThemeDigest, baseBindings); !errors.Is(err, ErrUnknownIsland) {
		t.Fatalf("unknown island error = %v", err)
	}

	baseBindings.Islands = map[string]IslandBinding{
		"sf-known": {ComponentID: "forum.component.known", Props: []IslandPropContract{{Name: "session-token", Type: IslandPropString}}},
	}
	if _, err := compiler.CompileFS(fstest.MapFS{
		"templates/home.html": &fstest.MapFile{Data: []byte(`<sf-known session-token="x"></sf-known>`)},
	}, viewModelThemeDigest, baseBindings); !errors.Is(err, ErrInvalidIsland) {
		t.Fatalf("sensitive island prop error = %v", err)
	}

	if _, _, err := segmentRenderedHTML(`<sf-known link="javascript:alert(1)"></sf-known>`, map[string]IslandBinding{
		"sf-known": {ComponentID: "forum.component.known", Props: []IslandPropContract{{Name: "link", Type: IslandPropURL}}},
	}); !errors.Is(err, ErrInvalidIsland) {
		t.Fatalf("unsafe island URL error = %v", err)
	}

	baseBindings.Islands = nil
	if _, err := compiler.CompileFS(fstest.MapFS{
		"templates/home.html": &fstest.MapFile{Data: []byte(`<template data-sforum-island="forged"></template>`)},
	}, viewModelThemeDigest, baseBindings); !errors.Is(err, ErrInvalidIsland) {
		t.Fatalf("reserved placeholder error = %v", err)
	}
}

func TestRenderOutputPreservesNestedIslandDOM(t *testing.T) {
	files := fstest.MapFS{
		"templates/home.html": &fstest.MapFile{Data: []byte(
			`<div class="sf-page"><header>Before</header><sf-home-page></sf-home-page><footer>After</footer></div>`,
		)},
	}
	snapshot, err := NewCompiler(Limits{}).CompileFS(files, viewModelThemeDigest, Bindings{
		BindingRevision: viewModelBindingRevision,
		PageViewModels: map[string]PageTemplateBinding{
			"templates/home.html": {PageID: "forum.home", SchemaVersion: "sforum.page.home@1"},
		},
		Islands: map[string]IslandBinding{
			"sf-home-page": {ComponentID: "forum.component.home_page"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	bound, err := CorePageViewModelRegistry().Bind(
		"forum.home", "sforum.page.home@1", viewModelThemeDigest, validHomeViewModel(),
	)
	if err != nil {
		t.Fatal(err)
	}
	output, err := snapshot.Render(context.Background(), "templates/home.html", bound)
	if err != nil {
		t.Fatal(err)
	}
	segments := output.HTMLSegments()
	if len(segments) != 1 {
		t.Fatalf("nested document split into %d fragments: %#v", len(segments), segments)
	}
	want := `<div class="sf-page"><header>Before</header><template data-sforum-island="forum.component.home_page:1"></template><footer>After</footer></div>`
	if segments[0].String() != want {
		t.Fatalf("nested HTML = %q, want %q", segments[0].String(), want)
	}
	if islands := output.Islands(); len(islands) != 1 || islands[0].ComponentID != "forum.component.home_page" {
		t.Fatalf("islands = %#v", islands)
	}
}

func TestBundledThemeTemplatesCompileAndPreserveNestedHomeIsland(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../../../"))
	for _, themeID := range []string{"sforum-default", "sforum-nocturne"} {
		t.Run(themeID, func(t *testing.T) {
			snapshot, err := NewCompiler(Limits{}).CompileDir(
				filepath.Join(repositoryRoot, "extensions/builtin/themes", themeID),
				viewModelThemeDigest,
				Bindings{
					BindingRevision: viewModelBindingRevision,
					SiteName:        "SForum",
					PageViewModels: map[string]PageTemplateBinding{
						"templates/home.html": {PageID: "forum.home", SchemaVersion: "sforum.page.home@1"},
					},
					Islands: map[string]IslandBinding{
						"sf-home-page": {ComponentID: "forum.component.home_page"},
					},
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			bound, err := CorePageViewModelRegistry().Bind(
				"forum.home", "sforum.page.home@1", viewModelThemeDigest, validHomeViewModel(),
			)
			if err != nil {
				t.Fatal(err)
			}
			output, err := snapshot.Render(context.Background(), "templates/home.html", bound)
			if err != nil {
				t.Fatal(err)
			}
			var rendered strings.Builder
			for _, segment := range output.HTMLSegments() {
				rendered.WriteString(segment.String())
			}
			if !strings.Contains(rendered.String(), `<div class="sf-page`) ||
				!strings.Contains(rendered.String(), `data-sforum-island="forum.component.home_page:1"`) ||
				!strings.Contains(rendered.String(), `</template>`) || !strings.Contains(rendered.String(), `</div>`) {
				t.Fatalf("bundled theme structure was not preserved: %s", rendered.String())
			}
			if themeID == "sforum-nocturne" && !strings.Contains(rendered.String(), ">SForum</p>") {
				t.Fatalf("siteName compatibility binding missing: %s", rendered.String())
			}
		})
	}
}

func TestCompilerRequiresExactSensitivePageIsland(t *testing.T) {
	compiler := NewCompiler(Limits{})
	pageBindings := map[string]PageTemplateBinding{
		"templates/login.html": {PageID: "auth.login", SchemaVersion: "sforum.page.login@1"},
	}
	tests := []struct {
		name    string
		source  string
		islands map[string]IslandBinding
	}{
		{name: "missing", source: `<main>Login</main>`},
		{
			name:   "wrong protected component",
			source: `<sf-register-form></sf-register-form>`,
			islands: map[string]IslandBinding{
				"sf-register-form": {ComponentID: "identity.component.register_form"},
			},
		},
		{
			name:   "duplicate required component",
			source: `<sf-login-form></sf-login-form><sf-login-form></sf-login-form>`,
			islands: map[string]IslandBinding{
				"sf-login-form": {ComponentID: "identity.component.login_form"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := compiler.CompileFS(fstest.MapFS{
				"templates/login.html": &fstest.MapFile{Data: []byte(test.source)},
			}, viewModelThemeDigest, Bindings{
				BindingRevision: viewModelBindingRevision, PageViewModels: pageBindings, Islands: test.islands,
			})
			if !errors.Is(err, ErrRequiredIsland) {
				t.Fatalf("CompileFS() error = %v", err)
			}
		})
	}
}

func TestSnapshotRequiresSensitiveIslandInRenderedBranch(t *testing.T) {
	snapshot, err := NewCompiler(Limits{}).CompileFS(fstest.MapFS{
		"templates/login.html": &fstest.MapFile{Data: []byte(
			`{{if .RegistrationEnabled}}<sf-login-form></sf-login-form>{{end}}`,
		)},
	}, viewModelThemeDigest, Bindings{
		BindingRevision: viewModelBindingRevision,
		PageViewModels: map[string]PageTemplateBinding{
			"templates/login.html": {PageID: "auth.login", SchemaVersion: "sforum.page.login@1"},
		},
		Islands: map[string]IslandBinding{
			"sf-login-form": {ComponentID: "identity.component.login_form"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	model := LoginPageViewModel{
		Base: PageViewModelBase{
			PageID: "auth.login", SchemaVersion: "sforum.page.login@1", Locale: "en-US",
			Route: PageRouteView{Path: "/login"}, SEO: PageSEOView{Title: "Login"},
		},
		Form: HostFormBoundary{
			ComponentID: "identity.component.login_form", ActionRouteIDs: []string{"core.route.identity.login"},
		},
		RegistrationEnabled: false,
	}
	bound, err := CorePageViewModelRegistry().Bind(
		"auth.login", "sforum.page.login@1", viewModelThemeDigest, model,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot.Render(context.Background(), "templates/login.html", bound); !errors.Is(err, ErrRequiredIsland) {
		t.Fatalf("Render() error = %v", err)
	}
}

func TestSafeHTMLCannotMintHostIslandAuthority(t *testing.T) {
	files := fstest.MapFS{
		"templates/topic.html": &fstest.MapFile{Data: []byte(`{{safeHTML .Body}}`)},
	}
	snapshot, err := NewCompiler(Limits{}).CompileFS(files, viewModelThemeDigest, Bindings{
		BindingRevision: viewModelBindingRevision,
		PageViewModels: map[string]PageTemplateBinding{
			"templates/topic.html": {PageID: "forum.topic.show", SchemaVersion: "sforum.page.topic_show@1"},
		},
		Islands: map[string]IslandBinding{"sf-topic-list": {ComponentID: "forum.component.topic_list"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	model := TopicDetailPageViewModel{
		Base: PageViewModelBase{
			PageID: "forum.topic.show", SchemaVersion: "sforum.page.topic_show@1", Locale: "en-US",
			Route: PageRouteView{Path: "/t/example"}, SEO: PageSEOView{Title: "Example"},
		},
		Body: NewSafeHTMLFromSanitized(`<sf-topic-list></sf-topic-list>`),
	}
	bound, err := CorePageViewModelRegistry().Bind("forum.topic.show", "sforum.page.topic_show@1", viewModelThemeDigest, model)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot.Render(context.Background(), "templates/topic.html", bound); !errors.Is(err, ErrInvalidIsland) {
		t.Fatalf("SafeHTML island error = %v", err)
	}
}

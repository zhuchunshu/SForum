package pages

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	themecompiler "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeCompiler"
)

func TestAllCatalogThemeRendersPerformNoFilesystemIOAfterCompilation(t *testing.T) {
	const digest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	requiredTags := map[string]string{
		"forum.topic.create":      "sf-topic-composer",
		"forum.settings.profile":  "sf-profile-settings",
		"forum.settings.security": "sf-security-settings",
		"auth.login":              "sf-login-form",
		"auth.register":           "sf-register-form",
		"auth.forgot_password":    "sf-recovery-request",
		"auth.reset_password":     "sf-recovery-confirm",
	}
	islands := map[string]themecompiler.IslandBinding{
		"sf-topic-composer":    {ComponentID: "forum.component.topic_composer"},
		"sf-profile-settings":  {ComponentID: "profile.component.settings_form"},
		"sf-security-settings": {ComponentID: "identity.component.security_settings"},
		"sf-login-form":        {ComponentID: "identity.component.login_form"},
		"sf-register-form":     {ComponentID: "identity.component.register_form"},
		"sf-recovery-request":  {ComponentID: "identity.component.recovery_request_form"},
		"sf-recovery-confirm":  {ComponentID: "identity.component.recovery_confirm_form"},
	}
	files := fstest.MapFS{}
	bindings := make(map[string]themecompiler.PageTemplateBinding, len(Catalog()))
	for _, page := range Catalog() {
		name := "templates/" + page.ID + ".html"
		body := `<main><h1>{{.Base.SEO.Title}}</h1>`
		if tag := requiredTags[page.ID]; tag != "" {
			body += "<" + tag + "></" + tag + ">"
		}
		body += `</main>`
		files[name] = &fstest.MapFile{Data: []byte(body), Mode: 0o600}
		bindings[name] = themecompiler.PageTemplateBinding{
			PageID: page.ID, SchemaVersion: page.ContractVersion,
		}
	}
	source := &countingThemeRuntimeFS{FS: files}
	snapshot, err := themecompiler.NewCompiler(themecompiler.Limits{}).CompileFS(source, digest, themecompiler.Bindings{
		BindingRevision: strings.Repeat("b", 64),
		PageViewModels:  bindings,
		Islands:         islands,
	})
	if err != nil {
		t.Fatal(err)
	}
	compileOpens := source.opens.Load()
	for _, page := range Catalog() {
		model, err := BuildCorePageViewModel(CorePageViewModelRequest{
			PageID: page.ID, Locale: "zh-CN", Path: page.PathPattern,
			SEO: themecompiler.PageSEOView{Title: page.ID},
		})
		if err != nil {
			t.Fatalf("build %s ViewModel: %v", page.ID, err)
		}
		bound, err := themecompiler.CorePageViewModelRegistry().Bind(
			page.ID, page.ContractVersion, digest, model,
		)
		if err != nil {
			t.Fatalf("bind %s ViewModel: %v", page.ID, err)
		}
		name := "templates/" + page.ID + ".html"
		output, err := snapshot.Render(context.Background(), name, bound)
		if err != nil || len(output.HTMLSegments()) == 0 {
			t.Fatalf("render %s: segments=%d err=%v", page.ID, len(output.HTMLSegments()), err)
		}
	}
	if got := source.opens.Load(); got != compileOpens {
		t.Fatalf("catalog render opened theme files: compile=%d after=%d", compileOpens, got)
	}
}

func TestAllCatalogProviderResolvesPerformNoBindingStoreReads(t *testing.T) {
	store := newCountingBindingStore()
	registry := NewRegistry(store)
	if err := registry.RestoreBindings(t.Context()); err != nil {
		t.Fatal(err)
	}
	for range 100 {
		for _, page := range Catalog() {
			resolved, err := registry.Resolve(t.Context(), page.ID)
			if err != nil || resolved.Provider != ProviderCore {
				t.Fatalf("resolve %s = %#v, %v", page.ID, resolved, err)
			}
		}
	}
	if store.listCalls.Load() != 1 || store.getCalls.Load() != 0 {
		t.Fatalf("catalog hot-path Store reads: list=%d get=%d", store.listCalls.Load(), store.getCalls.Load())
	}
}

type countingThemeRuntimeFS struct {
	fs.FS
	opens atomic.Int64
}

func (s *countingThemeRuntimeFS) Open(name string) (fs.File, error) {
	if s == nil || s.FS == nil {
		return nil, fmt.Errorf("theme runtime test filesystem is unavailable")
	}
	s.opens.Add(1)
	return s.FS.Open(name)
}

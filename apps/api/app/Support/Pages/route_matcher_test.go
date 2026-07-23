package pages

import (
	"fmt"
	"testing"
)

func TestCanonicalRouteSignatureIgnoresParamNames(t *testing.T) {
	a, err := CanonicalRouteSignature("/docs/:slug")
	if err != nil {
		t.Fatal(err)
	}
	b, err := CanonicalRouteSignature("/docs/:id")
	if err != nil {
		t.Fatal(err)
	}
	if a != b || a != "/docs/P" {
		t.Fatalf("expected same signature /docs/P, got %q %q", a, b)
	}
	c, err := CanonicalRouteSignature("/docs/*rest")
	if err != nil {
		t.Fatal(err)
	}
	if c == a {
		t.Fatal("catch-all must differ from param")
	}
	if c != "/docs/C" {
		t.Fatalf("catch-all sig: %q", c)
	}
}

func TestRegisterRejectsParamNameCollision(t *testing.T) {
	reg := NewRegistry(NewMemoryStore())
	if err := reg.RegisterContributions("a", []PageContribution{{
		ID: "a.docs", Action: ActionAdd, Path: "/docs/:slug",
		ExtensionID: "a", Version: "1", PackageDigest: "d", Access: AccessPublic,
	}}); err != nil {
		t.Fatal(err)
	}
	err := reg.RegisterContributions("b", []PageContribution{{
		ID: "b.docs", Action: ActionAdd, Path: "/docs/:id",
		ExtensionID: "b", Version: "1", PackageDigest: "d", Access: AccessPublic,
	}})
	if err == nil {
		t.Fatal("expected conflict between /docs/:slug and /docs/:id")
	}
}

func TestRegisterRejectsPublicVsLoginEquivalentRoutes(t *testing.T) {
	reg := NewRegistry(NewMemoryStore())
	if err := reg.RegisterContributions("a", []PageContribution{{
		ID: "a.x", Action: ActionAdd, Path: "/portal/:id",
		ExtensionID: "a", Version: "1", PackageDigest: "d", Access: AccessPublic,
	}}); err != nil {
		t.Fatal(err)
	}
	err := reg.RegisterContributions("b", []PageContribution{{
		ID: "b.x", Action: ActionAdd, Path: "/portal/:slug",
		ExtensionID: "b", Version: "1", PackageDigest: "d", Access: AccessLogin,
	}})
	if err == nil {
		t.Fatal("equivalent routes with different access must still conflict")
	}
}

func TestStaticBeatsParamBeatsCatchAll(t *testing.T) {
	reg := NewRegistry(NewMemoryStore())
	// 注册顺序故意：先 catch-all，再 param，再 static
	if err := reg.RegisterContributions("p", []PageContribution{
		{ID: "p.all", Action: ActionAdd, Path: "/docs/*rest", ExtensionID: "p", Version: "1", PackageDigest: "d", Access: AccessPublic},
		{ID: "p.slug", Action: ActionAdd, Path: "/docs/:slug", ExtensionID: "p", Version: "1", PackageDigest: "d", Access: AccessPublic},
		{ID: "p.arch", Action: ActionAdd, Path: "/docs/archive", ExtensionID: "p", Version: "1", PackageDigest: "d", Access: AccessPublic},
	}); err != nil {
		t.Fatal(err)
	}
	m, ok := reg.ResolveAddedPathMatch("/docs/archive")
	if !ok || m.Contribution.ID != "p.arch" {
		t.Fatalf("static should win: %#v", m)
	}
	m, ok = reg.ResolveAddedPathMatch("/docs/hello")
	if !ok || m.Contribution.ID != "p.slug" {
		t.Fatalf("param should win over catch-all: %#v", m)
	}
	if m.Params["slug"] != "hello" {
		t.Fatalf("params: %#v", m.Params)
	}
	m, ok = reg.ResolveAddedPathMatch("/docs/a/b/c")
	if !ok || m.Contribution.ID != "p.all" {
		t.Fatalf("catch-all: %#v", m)
	}
	if m.Params["rest"] != "a/b/c" {
		t.Fatalf("catch-all params: %#v", m.Params)
	}
}

func TestLocalePrefixAndTrailingSlash(t *testing.T) {
	reg := NewRegistry(NewMemoryStore())
	_ = reg.RegisterContributions("p", []PageContribution{{
		ID: "p.docs", Action: ActionAdd, Path: "/docs/:slug",
		ExtensionID: "p", Version: "1", PackageDigest: "d", Access: AccessPublic,
	}})
	for _, path := range []string{"/docs/hello", "/docs/hello/", "/en/docs/hello", "/zh-CN/docs/hello/"} {
		m, ok := reg.ResolveAddedPathMatch(path)
		if !ok || m.Params["slug"] != "hello" {
			t.Fatalf("path %s: %#v ok=%v", path, m, ok)
		}
	}
}

func TestResolveAddedPathDeterministic(t *testing.T) {
	reg := NewRegistry(NewMemoryStore())
	_ = reg.RegisterContributions("p", []PageContribution{
		{ID: "p.a", Action: ActionAdd, Path: "/x/:a", ExtensionID: "p", Version: "1", PackageDigest: "d", Access: AccessPublic},
		{ID: "p.b", Action: ActionAdd, Path: "/y/:b", ExtensionID: "p", Version: "1", PackageDigest: "d", Access: AccessPublic},
		{ID: "p.c", Action: ActionAdd, Path: "/z/*c", ExtensionID: "p", Version: "1", PackageDigest: "d", Access: AccessPublic},
	})
	var first string
	for i := 0; i < 50; i++ {
		m, ok := reg.ResolveAddedPathMatch("/x/1")
		if !ok {
			t.Fatal("expected match")
		}
		if i == 0 {
			first = m.Contribution.ID
		} else if m.Contribution.ID != first {
			t.Fatalf("non-deterministic: %s vs %s", first, m.Contribution.ID)
		}
	}
}

func TestUnknownAccessRejectedAtRegister(t *testing.T) {
	reg := NewRegistry(NewMemoryStore())
	err := reg.RegisterContributions("p", []PageContribution{{
		ID: "p.x", Action: ActionAdd, Path: "/x", Access: Access("superuser"),
		ExtensionID: "p", Version: "1", PackageDigest: "d",
	}})
	if err == nil {
		t.Fatal("unknown access must fail closed")
	}
	// 空 access → public
	if err := reg.RegisterContributions("p", []PageContribution{{
		ID: "p.y", Action: ActionAdd, Path: "/y", Access: "",
		ExtensionID: "p", Version: "1", PackageDigest: "d",
	}}); err != nil {
		t.Fatal(err)
	}
	c, ok := reg.ResolveAddedPath("/y")
	if !ok || c.Access != AccessPublic {
		t.Fatalf("empty access should normalize public: %#v", c)
	}
}

func TestPermissionAccessRequiresKey(t *testing.T) {
	reg := NewRegistry(NewMemoryStore())
	err := reg.RegisterContributions("p", []PageContribution{{
		ID: "p.x", Action: ActionAdd, Path: "/secret", Access: AccessPermission,
		ExtensionID: "p", Version: "1", PackageDigest: "d",
	}})
	if err == nil {
		t.Fatal("permission without key must fail")
	}
	if err := reg.RegisterContributions("p", []PageContribution{{
		ID: "p.x", Action: ActionAdd, Path: "/secret", Access: AccessPermission, Permission: "extension.view",
		ExtensionID: "p", Version: "1", PackageDigest: "d",
	}}); err != nil {
		t.Fatal(err)
	}
}

func TestThemeReplaceAllowsSameAddPath(t *testing.T) {
	reg := NewRegistry(NewMemoryStore())
	_ = reg.RegisterContributions("theme.a", []PageContribution{{
		ID: "a.docs", Action: ActionAdd, Path: "/help/:slug",
		ExtensionID: "theme.a", Version: "1", PackageDigest: "d1", Access: AccessPublic,
	}})
	// 新主题相同路径应允许（替换视角）
	if err := reg.PreflightContributionsReplacing("theme.b", []PageContribution{{
		ID: "b.docs", Action: ActionAdd, Path: "/help/:id",
		ExtensionID: "theme.b", Version: "1", PackageDigest: "d2", Access: AccessPublic,
	}}, "theme.b", "theme.a"); err != nil {
		t.Fatalf("theme switch same path should pass: %v", err)
	}
	if err := reg.ReplaceThemeContributions("theme.b", []PageContribution{{
		ID: "b.docs", Action: ActionAdd, Path: "/help/:id",
		ExtensionID: "theme.b", Version: "1", PackageDigest: "d2", Access: AccessPublic,
	}}, "theme.a"); err != nil {
		t.Fatal(err)
	}
	c, ok := reg.ResolveAddedPath("/help/x")
	if !ok || c.ExtensionID != "theme.b" {
		t.Fatalf("expected theme.b: %#v", c)
	}
	if _, ok := reg.ResolveAddedPathMatch("/help/x"); !ok {
		t.Fatal("path should match")
	}
	// 与已启用插件冲突仍失败
	_ = reg.RegisterContributions("plugin.x", []PageContribution{{
		ID: "px", Action: ActionAdd, Path: "/shared/:x",
		ExtensionID: "plugin.x", Version: "1", PackageDigest: "d", Access: AccessPublic,
	}})
	err := reg.ReplaceThemeContributions("theme.c", []PageContribution{{
		ID: "c", Action: ActionAdd, Path: "/shared/:y",
		ExtensionID: "theme.c", Version: "1", PackageDigest: "d", Access: AccessPublic,
	}}, "theme.b")
	if err == nil {
		t.Fatal("theme vs plugin conflict must fail")
	}
	// 失败后旧主题仍在
	c, ok = reg.ResolveAddedPath("/help/x")
	if !ok || c.ExtensionID != "theme.b" {
		t.Fatalf("rollback keep theme.b: %#v", c)
	}
}

func TestApproveIgnoresClientTemplatePath(t *testing.T) {
	store := NewMemoryStore()
	reg := NewRegistry(store)
	ctx := t.Context()
	_ = reg.RegisterContributions("demo.theme", []PageContribution{{
		ID: "demo.home", Action: ActionReplace, Target: "forum.home",
		Template: "templates/real.html", Contract: "sforum.page.home@1",
		ExtensionID: "demo.theme", Version: "1.0.0", PackageDigest: "abc",
	}})
	if err := reg.ApproveReplace(ctx, ProviderBinding{
		PageID: "forum.home", ExtensionID: "demo.theme", ContributionID: "demo.home",
		Version: "1.0.0", PackageDigest: "abc", ContractVersion: "sforum.page.home@1",
		ApprovedBy: 1, TemplatePath: "templates/evil.html",
	}); err != nil {
		t.Fatal(err)
	}
	b, ok, _ := store.GetBinding(ctx, "forum.home")
	if !ok || b.TemplatePath != "templates/real.html" {
		t.Fatalf("must use contribution template, got %#v", b)
	}
	r, _ := reg.Resolve(ctx, "forum.home")
	if r.TemplatePath != "templates/real.html" {
		t.Fatalf("resolve template: %s", r.TemplatePath)
	}
}

func TestMatchCorePagePathAcceptsArbitraryVirtualSystemErrorPath(t *testing.T) {
	for _, pageID := range []string{"system.forbidden", "system.not_found", "system.rate_limited", "system.server_error"} {
		params, ok := MatchCorePagePath(pageID, "/missing/discussion")
		if !ok || len(params) != 0 {
			t.Fatalf("%s should bind the current error path without params: ok=%v params=%#v", pageID, ok, params)
		}
	}
	if _, ok := MatchCorePagePath("forum.home", "/missing/discussion"); ok {
		t.Fatal("ordinary catalog pages must still reject mismatched paths")
	}
}

func TestContractMismatchFallsBack(t *testing.T) {
	store := NewMemoryStore()
	reg := NewRegistry(store)
	ctx := t.Context()
	_ = reg.RegisterContributions("demo.theme", []PageContribution{{
		ID: "demo.home", Action: ActionReplace, Target: "forum.home",
		Template: "templates/home.html", Contract: "sforum.page.home@1",
		ExtensionID: "demo.theme", Version: "1.0.0", PackageDigest: "abc",
	}})
	_ = store.UpsertBinding(ctx, ProviderBinding{
		PageID: "forum.home", ExtensionID: "demo.theme", ContributionID: "demo.home",
		Version: "1.0.0", PackageDigest: "abc", ContractVersion: "sforum.page.home@999",
	})
	if err := reg.RestoreBindings(ctx); err != nil {
		t.Fatal(err)
	}
	r, _ := reg.Resolve(ctx, "forum.home")
	if r.Provider != ProviderCore {
		t.Fatalf("bad contract should fallback core, got %s", r.Provider)
	}
}

func TestNormalizeAccess(t *testing.T) {
	a, err := NormalizeAccess("")
	if err != nil || a != AccessPublic {
		t.Fatalf("%v %v", a, err)
	}
	if _, err := NormalizeAccess("nope"); err == nil {
		t.Fatal("expected error")
	}
	for _, v := range []string{"public", "login", "guest", "moderation", "permission"} {
		if _, err := NormalizeAccess(v); err != nil {
			t.Fatal(v, err)
		}
	}
}

func TestSignaturesStableAcrossRuns(t *testing.T) {
	for i := 0; i < 20; i++ {
		reg := NewRegistry(NewMemoryStore())
		_ = reg.RegisterContributions("p", []PageContribution{
			{ID: "static", Action: ActionAdd, Path: "/docs/archive", ExtensionID: "p", Version: "1", PackageDigest: "d", Access: AccessPublic},
			{ID: "param", Action: ActionAdd, Path: "/docs/:slug", ExtensionID: "p", Version: "1", PackageDigest: "d", Access: AccessPublic},
		})
		m, ok := reg.ResolveAddedPathMatch("/docs/archive")
		if !ok || m.Contribution.ID != "static" {
			t.Fatalf("run %d: %#v", i, m)
		}
	}
}

func TestCompileRouteErrors(t *testing.T) {
	if _, err := CompileRoute("/a/*x/b", PageContribution{}); err == nil {
		t.Fatal("catch-all mid path")
	}
	sig, err := CanonicalRouteSignature("/Docs/:Slug")
	if err != nil {
		t.Fatal(err)
	}
	if sig != "/docs/P" {
		t.Fatalf("case fold static: %s", sig)
	}
	_ = fmt.Sprintf("ok")
}

func TestMatchCorePagePathBindsOnlyTheExactCatalogPage(t *testing.T) {
	params, ok := MatchCorePagePath("forum.category.show", "/zh-CN/c/support")
	if !ok || params["categorySlug"] != "support" {
		t.Fatalf("expected exact localized category match, got ok=%v params=%v", ok, params)
	}
	params, ok = MatchCorePagePath("forum.topic.show", "/t/42/hello-world")
	if !ok || params["path"] != "42/hello-world" {
		t.Fatalf("expected catch-all topic path, got ok=%v params=%v", ok, params)
	}
	if _, ok := MatchCorePagePath("forum.home", "/u/alice"); ok {
		t.Fatal("a profile path must not be accepted for the home ViewModel")
	}
	if _, ok := MatchCorePagePath("plugin.unknown", "/"); ok {
		t.Fatal("unknown page ids must fail closed")
	}
}

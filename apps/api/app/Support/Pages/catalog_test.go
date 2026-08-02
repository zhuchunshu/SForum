package pages

import "testing"

func TestValidateCatalog(t *testing.T) {
	if err := ValidateCatalog(); err != nil {
		t.Fatalf("ValidateCatalog: %v", err)
	}
}

func TestCatalogUniqueAndHasHome(t *testing.T) {
	items := Catalog()
	if len(items) < 10 {
		t.Fatalf("expected substantial catalog, got %d", len(items))
	}
	home, ok := Find("forum.home")
	if !ok || home.PathPattern != "/" || !home.Replaceable {
		t.Fatalf("forum.home missing or invalid: %#v ok=%v", home, ok)
	}
	if _, ok := Find("no.such.page"); ok {
		t.Fatal("unknown id should not be found")
	}
}

func TestNotificationSettingsPageContract(t *testing.T) {
	page, ok := Find("forum.settings.notifications")
	if !ok || page.PathPattern != "/settings/notifications" || page.Access != AccessLogin || !page.Replaceable || page.ContractVersion != "sforum.page.settings_notifications@1" {
		t.Fatalf("notification settings contract missing or invalid: %#v ok=%v", page, ok)
	}
}

func TestAppearanceSettingsPageContract(t *testing.T) {
	page, ok := Find("forum.settings.appearance")
	if !ok || page.PathPattern != "/settings/appearance" || page.Access != AccessLogin || !page.Replaceable || page.ContractVersion != "sforum.page.settings_appearance@1" {
		t.Fatalf("appearance settings contract missing or invalid: %#v ok=%v", page, ok)
	}
	if RequiredThemeBodyIslandTag(page.ID) != "sf-appearance-settings" {
		t.Fatalf("appearance settings body island mismatch: %q", RequiredThemeBodyIslandTag(page.ID))
	}
}

func TestLoginMethodsSettingsPageContract(t *testing.T) {
	page, ok := Find("forum.settings.login_methods")
	if !ok || page.PathPattern != "/settings/login-methods" || page.Access != AccessLogin || !page.Replaceable || page.ContractVersion != "sforum.page.settings_login_methods@1" {
		t.Fatalf("login methods settings contract missing or invalid: %#v ok=%v", page, ok)
	}
	if RequiredThemeBodyIslandTag(page.ID) != "sf-login-methods-settings" {
		t.Fatalf("login methods body island mismatch: %q", RequiredThemeBodyIslandTag(page.ID))
	}
}

func TestExternalAuthContinuationPageContract(t *testing.T) {
	page, ok := Find("auth.external_continuation")
	if !ok || page.PathPattern != "/auth/continue" || page.Access != AccessPublic || !page.Replaceable || page.ContractVersion != "sforum.page.external_auth_continuation@1" {
		t.Fatalf("external auth continuation contract missing or invalid: %#v ok=%v", page, ok)
	}
	if RequiredThemeBodyIslandTag(page.ID) != "sf-external-auth-continuation" {
		t.Fatalf("external auth continuation body island mismatch: %q", RequiredThemeBodyIslandTag(page.ID))
	}
}

func TestEmailVerificationPageContract(t *testing.T) {
	page, ok := Find("auth.email_verification")
	if !ok || page.PathPattern != "/email-verification" || page.Access != AccessLogin || !page.Replaceable || page.ContractVersion != "sforum.page.email_verification@1" {
		t.Fatalf("email verification contract missing or invalid: %#v ok=%v", page, ok)
	}
	if RequiredThemeBodyIslandTag(page.ID) != "sf-email-verification" {
		t.Fatalf("email verification body island mismatch: %q", RequiredThemeBodyIslandTag(page.ID))
	}
}

func TestLocalPasswordSettingsPageContract(t *testing.T) {
	page, ok := Find("forum.settings.password")
	if !ok || page.PathPattern != "/settings/password" || page.Access != AccessLogin || !page.Replaceable || page.ContractVersion != "sforum.page.settings_password@1" {
		t.Fatalf("local password settings contract missing or invalid: %#v ok=%v", page, ok)
	}
	if RequiredThemeBodyIslandTag(page.ID) != "sf-local-password-settings" {
		t.Fatalf("local password body island mismatch: %q", RequiredThemeBodyIslandTag(page.ID))
	}
}

func TestPersonalAccessTokensPageContract(t *testing.T) {
	page, ok := Find("forum.settings.tokens")
	if !ok || page.PathPattern != "/settings/tokens" || page.Access != AccessLogin || !page.Replaceable || page.ContractVersion != "sforum.page.settings_tokens@1" {
		t.Fatalf("personal access tokens contract missing or invalid: %#v ok=%v", page, ok)
	}
	if RequiredThemeBodyIslandTag(page.ID) != "sf-personal-access-tokens" {
		t.Fatalf("personal access tokens body island mismatch: %q", RequiredThemeBodyIslandTag(page.ID))
	}
}

func TestSearchPageContract(t *testing.T) {
	page, ok := Find("forum.search")
	if !ok || page.PathPattern != "/search" || page.Access != AccessPublic || !page.Replaceable || page.ContractVersion != "sforum.page.search@1" {
		t.Fatalf("search page contract missing or invalid: %#v ok=%v", page, ok)
	}
	if RequiredThemeBodyIslandTag(page.ID) != "sf-search-page" {
		t.Fatalf("search page body island mismatch: %q", RequiredThemeBodyIslandTag(page.ID))
	}
}

func TestNotificationDetailPageContract(t *testing.T) {
	page, ok := Find("forum.notification.show")
	if !ok || page.PathPattern != "/notifications/:notificationId" || page.Access != AccessLogin || !page.Replaceable || page.ContractVersion != "sforum.page.notification_show@1" {
		t.Fatalf("notification detail contract missing or invalid: %#v ok=%v", page, ok)
	}
	matched, ok := MatchPath("/notifications/42")
	if !ok || matched.ID != page.ID || RequiredThemeBodyIslandTag(page.ID) != "sf-notification-detail-page" {
		t.Fatalf("notification detail path/island mismatch: %#v ok=%v", matched, ok)
	}
}

func TestModerationPageAllowsOnlyThemePresentation(t *testing.T) {
	page, ok := Find("moderation.review")
	if !ok || page.Replaceable || !page.Themeable || page.Access != AccessModeration {
		t.Fatalf("moderation presentation boundary missing or invalid: %#v ok=%v", page, ok)
	}
	if RequiredThemeBodyIslandTag(page.ID) != "sf-moderation-review" {
		t.Fatalf("moderation body island mismatch: %q", RequiredThemeBodyIslandTag(page.ID))
	}
}

func TestResolveCore(t *testing.T) {
	resolved, err := ResolveCore("forum.home")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Provider != ProviderCore || resolved.Action != "core" {
		t.Fatalf("unexpected resolve: %#v", resolved)
	}
	if _, err := ResolveCore("missing"); err == nil {
		t.Fatal("expected error for missing page")
	}
}

func TestMatchPath(t *testing.T) {
	cases := []struct {
		path string
		id   string
		ok   bool
	}{
		{"/", "forum.home", true},
		{"/en", "forum.home", true},
		{"/search", "forum.search", true},
		{"/en/search", "forum.search", true},
		{"/categories", "forum.category.index", true},
		{"/en/categories", "forum.category.index", true},
		{"/c/general", "forum.category.show", true},
		{"/login", "auth.login", true},
		{"/auth/continue", "auth.external_continuation", true},
		{"/control-panel", "", false},
		{"/api/v1/health", "", false},
	}
	for _, tc := range cases {
		page, ok := MatchPath(tc.path)
		if ok != tc.ok {
			t.Fatalf("path %q: ok=%v want %v", tc.path, ok, tc.ok)
		}
		if tc.ok && page.ID != tc.id {
			t.Fatalf("path %q: got id %q want %q", tc.path, page.ID, tc.id)
		}
	}
}

func TestReservedPaths(t *testing.T) {
	for _, p := range []string{"/admin", "/admin/x", "/control-panel", "/api/v1/x", "/_nuxt/foo", "/_sforum/notifications/sw.js", "/health"} {
		if !IsReservedPath(p) {
			t.Fatalf("expected reserved %q", p)
		}
	}
	for _, p := range []string{"/", "/login", "/t/1/hello", "/docs/guide"} {
		if IsReservedPath(p) {
			t.Fatalf("did not expect reserved %q", p)
		}
	}
}

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
		{"/categories", "forum.category.index", true},
		{"/en/categories", "forum.category.index", true},
		{"/c/general", "forum.category.show", true},
		{"/login", "auth.login", true},
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

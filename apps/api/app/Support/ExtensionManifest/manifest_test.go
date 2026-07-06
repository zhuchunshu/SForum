package extensionmanifest

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestAdminManifestV2NormalizeValidateAndResolveManagePath(t *testing.T) {
	body := []byte(`{
		"id":"demo.plugin",
		"name":"Demo Plugin",
		"description":"Demo plugin.",
		"url":"https://example.com/demo",
		"author":{"name":"Demo Studio"},
		"version":"1.0.0",
		"type":"plugin",
		"sforumVersion":"^1.0.0",
		"admin":{
			"entry":"settings",
			"pages":[
				{"path":"settings","label":"Settings","view":"settings"},
				{"path":"/dashboard","label":"Dashboard","view":"about","menu":true,"icon":"i-lucide-layout-dashboard","order":20}
			]
		}
	}`)

	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	normalized := Normalize(manifest)
	if normalized.Admin.Entry != "/settings" {
		t.Fatalf("expected normalized admin entry, got %q", normalized.Admin.Entry)
	}
	pages := EffectiveAdminPages(normalized)
	if len(pages) != 2 {
		t.Fatalf("expected two effective pages, got %#v", pages)
	}
	if pages[0].Path != "/settings" || pages[0].Menu {
		t.Fatalf("settings page should normalize with menu false: %#v", pages[0])
	}
	if pages[1].Path != "/dashboard" || !pages[1].Menu {
		t.Fatalf("dashboard page should be an explicit menu page: %#v", pages[1])
	}
	if AdminManagePath(normalized) != "/settings" {
		t.Fatalf("expected entry to drive manage path, got %q", AdminManagePath(normalized))
	}
	menuPages := MenuAdminPages(normalized)
	if len(menuPages) != 1 || menuPages[0].Path != "/dashboard" {
		t.Fatalf("expected only explicit menu page, got %#v", menuPages)
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("v2 admin manifest should validate: %v", err)
	}
}

func TestAdminManifestV2RejectsBrokenEntryAndExternalPaths(t *testing.T) {
	base := Manifest{
		ID:            "demo.plugin",
		Name:          "Demo Plugin",
		Description:   "Demo plugin.",
		URL:           "https://example.com/demo",
		Author:        ManifestAuthor{Name: "Demo Studio"},
		Version:       "1.0.0",
		Type:          TypePlugin,
		SForumVersion: "^1.0.0",
	}

	cases := []struct {
		name     string
		manifest Manifest
	}{
		{
			name: "entry must target declared page or about",
			manifest: func() Manifest {
				next := base
				next.Admin.Entry = "/missing"
				next.Admin.Pages = []ManifestAdminPage{{Path: "/settings", Label: "Settings", View: "settings"}}
				return next
			}(),
		},
		{
			name: "entry cannot be external url",
			manifest: func() Manifest {
				next := base
				next.Admin.Entry = "https://example.com/settings"
				next.Admin.Pages = []ManifestAdminPage{{Path: "/settings", Label: "Settings", View: "settings"}}
				return next
			}(),
		},
		{
			name: "page cannot contain traversal",
			manifest: func() Manifest {
				next := base
				next.Admin.Pages = []ManifestAdminPage{{Path: "/../settings", Label: "Settings", View: "settings"}}
				return next
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Validate(tc.manifest); !errors.Is(err, ErrInvalidManifest) {
				t.Fatalf("expected invalid manifest, got %v", err)
			}
		})
	}
}

func TestLegacyAdminPagesRemainCompatible(t *testing.T) {
	manifest := Manifest{
		ID:            "legacy.plugin",
		Name:          "Legacy Plugin",
		Description:   "Legacy plugin.",
		URL:           "https://example.com/legacy",
		Author:        ManifestAuthor{Name: "Demo Studio"},
		Version:       "1.0.0",
		Type:          TypePlugin,
		SForumVersion: "^1.0.0",
		AdminPages: []ManifestAdminPage{
			{Path: "/settings", Label: "Settings", View: "settings", Menu: true},
		},
	}

	if err := Validate(manifest); err != nil {
		t.Fatalf("legacy adminPages should validate: %v", err)
	}
	normalized := Normalize(manifest)
	if AdminManagePath(normalized) != "/settings" {
		t.Fatalf("expected legacy settings page as manage path, got %q", AdminManagePath(normalized))
	}
	menuPages := MenuAdminPages(normalized)
	if len(menuPages) != 1 || menuPages[0].Path != "/settings" {
		t.Fatalf("expected legacy explicit menu page, got %#v", menuPages)
	}
}

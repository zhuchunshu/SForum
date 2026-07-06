package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func TestGeneratePluginScaffoldNonInteractive(t *testing.T) {
	target := filepath.Join(t.TempDir(), "plugin")
	created, err := GenerateExtensionScaffold(makeOptions{
		Kind:          "plugin",
		ID:            "acme.demo",
		Name:          "Acme Demo",
		Description:   "Demo plugin.",
		URL:           "https://example.com/acme-demo",
		AuthorName:    "Acme",
		AuthorURL:     "https://example.com",
		AuthorEmail:   "dev@example.com",
		Out:           target,
		NoInteraction: true,
		Backend:       true,
	})
	if err != nil {
		t.Fatalf("GenerateExtensionScaffold returned error: %v", err)
	}
	if created != target {
		t.Fatalf("expected target %q, got %q", target, created)
	}
	manifest := readGeneratedManifest(t, target)
	if manifest.Type != extensionmanifest.TypePlugin || manifest.Backend.Entry != "backend/plugin" {
		t.Fatalf("unexpected plugin manifest: %#v", manifest)
	}
	if manifest.Admin.Entry != "/settings" || len(manifest.Admin.Pages) != 1 || manifest.Admin.Pages[0].Menu {
		t.Fatalf("expected v2 plugin admin settings page without sidebar menu: %#v", manifest.Admin)
	}
	if len(manifest.AdminPages) != 0 {
		t.Fatalf("new scaffolds should not use legacy adminPages: %#v", manifest.AdminPages)
	}
	if err := extensionmanifest.Validate(manifest); err != nil {
		t.Fatalf("generated plugin manifest should validate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "backend", "plugin")); err != nil {
		t.Fatalf("expected backend stub: %v", err)
	}
}

func TestGenerateThemeScaffoldNonInteractive(t *testing.T) {
	target := filepath.Join(t.TempDir(), "theme")
	_, err := GenerateExtensionScaffold(makeOptions{
		Kind:          "theme",
		ID:            "acme.theme",
		Name:          "Acme Theme",
		Description:   "Demo theme.",
		URL:           "https://example.com/acme-theme",
		AuthorName:    "Acme",
		AuthorURL:     "https://example.com",
		Out:           target,
		NoInteraction: true,
	})
	if err != nil {
		t.Fatalf("GenerateExtensionScaffold returned error: %v", err)
	}
	manifest := readGeneratedManifest(t, target)
	if manifest.Type != extensionmanifest.TypeTheme || manifest.Frontend.Layer != "layer" {
		t.Fatalf("unexpected theme manifest: %#v", manifest)
	}
	if manifest.Admin.Entry != "/settings" || len(manifest.Admin.Pages) != 1 || len(manifest.Settings) == 0 {
		t.Fatalf("expected theme v2 admin page and settings declarations: %#v", manifest)
	}
	if len(manifest.AdminPages) != 0 {
		t.Fatalf("new theme scaffolds should not use legacy adminPages: %#v", manifest.AdminPages)
	}
	if err := extensionmanifest.Validate(manifest); err != nil {
		t.Fatalf("generated theme manifest should validate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "layer", "nuxt.config.ts")); err != nil {
		t.Fatalf("expected theme layer skeleton: %v", err)
	}
}

func readGeneratedManifest(t *testing.T, dir string) extensionmanifest.Manifest {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(dir, extensionmanifest.ManifestFileName))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest extensionmanifest.Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	return manifest
}

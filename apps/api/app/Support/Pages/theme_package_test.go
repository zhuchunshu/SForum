package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkinFromPackageUsesDigestPathForRelativeAssets(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "theme.json"), []byte(`{
		"skin":{"css":["assets/theme.css"],"tokens":"assets/tokens.css"}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	digest := strings.Repeat("a", 64)
	skin, err := SkinFromPackage("demo.theme", "1.0.0", digest, root)
	if err != nil {
		t.Fatal(err)
	}
	prefix := "/_sforum/assets/themes/demo.theme/" + digest + "/assets/"
	if len(skin.CSS) != 1 || skin.CSS[0] != prefix+"theme.css" || skin.Tokens != prefix+"tokens.css" {
		t.Fatalf("skin = %#v", skin)
	}
}

func TestResolveThemeAssetRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "ok.css"), []byte("body{}"), 0o644)
	if _, err := ResolveThemeAsset(root, "../secret"); err == nil {
		t.Fatal("expected traversal reject")
	}
	if _, err := ResolveThemeAsset(root, "/etc/passwd"); err == nil {
		t.Fatal("expected absolute reject")
	}
	if _, err := ResolveThemeAsset(root, "a/../../x"); err == nil {
		t.Fatal("expected .. reject")
	}
	full, err := ResolveThemeAsset(root, "ok.css")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(full) != "ok.css" {
		t.Fatalf("got %s", full)
	}
}

func TestResolveThemeAssetRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.css")
	_ = os.WriteFile(secret, []byte("secret"), 0o644)
	link := filepath.Join(root, "escape.css")
	if err := os.Symlink(secret, link); err != nil {
		t.Skip("symlink not supported:", err)
	}
	if _, err := ResolveThemeAsset(root, "escape.css"); err == nil {
		t.Fatal("expected symlink reject")
	}
}

func TestAllowedThemeAssetExtNoSVG(t *testing.T) {
	if _, ok := AllowedThemeAssetExt[".svg"]; ok {
		t.Fatal("svg must not be allowed for ordinary theme assets")
	}
	if _, ok := AllowedThemeAssetExt[".js"]; ok {
		t.Fatal("js must not be allowed")
	}
}

func TestLoadThemePackageValidatesNavigationLocationBindings(t *testing.T) {
	root := t.TempDir()
	valid := `{"navigationLocations":{"public.topbar.primary":"sf-navbar","public.sidebar.primary":"sf-home-navigation","public.mobile.primary":"sf-navbar","public.footer.primary":"sf-footer"},"skin":{"css":[]}}`
	if err := os.WriteFile(filepath.Join(root, "theme.json"), []byte(valid), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg, err := LoadThemePackage(root)
	if err != nil || len(pkg.NavigationLocations) != 4 {
		t.Fatalf("valid navigation locations: pkg=%#v err=%v", pkg.NavigationLocations, err)
	}

	invalid := `{"navigationLocations":{"public.sidebar.primary":"sf-footer"},"skin":{"css":[]}}`
	if err := os.WriteFile(filepath.Join(root, "theme.json"), []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadThemePackage(root); err == nil {
		t.Fatal("mismatched navigation island must fail closed")
	}
}

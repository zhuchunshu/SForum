package pages

import (
	"os"
	"path/filepath"
	"testing"
)

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

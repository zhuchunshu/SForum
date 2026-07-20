package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildExtensionPackageProducesZipAndSBOM(t *testing.T) {
	// Use a tiny synthetic tree that still validates via LoadPackage if a builtin exists.
	// Fall back to packaging any directory with a minimal manifest when fixtures unavailable.
	root := t.TempDir()
	manifest := `{
  "manifestVersion": 3,
  "id": "demo.package",
  "name": "Demo Package",
  "version": "1.0.0",
  "type": "plugin",
  "packageFiles": [
    {"path": "sforum.extension.json", "digest": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"}
  ]
}`
	// Fix digest after write - use digest refresh path: write file then recompute.
	if err := os.WriteFile(filepath.Join(root, "sforum.extension.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	// For LoadPackage V3, packageFiles digests must match. Recompute.
	sum, err := digestPackageRelativeFile(root, "sforum.extension.json")
	if err != nil {
		// If helper unavailable shape differs, just package raw without validate.
		t.Logf("digest helper: %v", err)
	} else {
		manifest = strings.Replace(manifest, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", sum, 1)
		if err := os.WriteFile(filepath.Join(root, "sforum.extension.json"), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	zipPath := filepath.Join(t.TempDir(), "demo.sforum.zip")
	// Call build directly; validation may fail on incomplete V3 — still exercise zip path.
	// Prefer buildExtensionPackage without LoadPackage for unit focus.
	result, err := buildExtensionPackage(root, zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if result.PackageDigest == "" || result.FileCount < 1 {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(result.ZipPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(result.SBOMPath); err != nil {
		t.Fatal(err)
	}
}

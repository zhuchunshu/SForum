package main

import (
	"archive/zip"
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
	result, err := buildExtensionPackage(root, zipPath, packageBuildOptions{})
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

func TestBuildExtensionPackageExcludeSource(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string, mode os.FileMode) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
	}
	write("sforum.extension.json", `{"id":"demo.release"}`, 0o644)
	write("backend/main.go", "package main\n", 0o644)
	write("backend/go.mod", "module demo\n", 0o644)
	write("backend/plugin", "#!/bin/sh\necho ok\n", 0o755)
	write("frontend/settings.mjs", "export default {}\n", 0o644)
	write("frontend/settings.mjs.map", "{}\n", 0o644)
	write("frontend/Widget.vue", "<template></template>\n", 0o644)
	write("testdata/fixture.json", "{}\n", 0o644)
	write("README.md", "# demo\n", 0o644)

	// 默认：源码一并打入
	fullZip := filepath.Join(t.TempDir(), "full.sforum.zip")
	full, err := buildExtensionPackage(root, fullZip, packageBuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	fullNames := zipEntryNames(t, full.ZipPath)
	for _, want := range []string{"backend/main.go", "backend/go.mod", "frontend/Widget.vue", "testdata/fixture.json"} {
		if !fullNames[want] {
			t.Fatalf("default package missing source file %s; got %#v", want, fullNames)
		}
	}

	// --exclude-source：只保留运行时与说明文件
	releaseZip := filepath.Join(t.TempDir(), "release.sforum.zip")
	release, err := buildExtensionPackage(root, releaseZip, packageBuildOptions{ExcludeSource: true})
	if err != nil {
		t.Fatal(err)
	}
	if release.SkippedCount < 4 {
		t.Fatalf("expected source skips, got %#v", release)
	}
	names := zipEntryNames(t, release.ZipPath)
	for _, want := range []string{"sforum.extension.json", "backend/plugin", "frontend/settings.mjs", "README.md"} {
		if !names[want] {
			t.Fatalf("release package missing %s; got %#v", want, names)
		}
	}
	for _, deny := range []string{"backend/main.go", "backend/go.mod", "frontend/Widget.vue", "frontend/settings.mjs.map", "testdata/fixture.json"} {
		if names[deny] {
			t.Fatalf("release package should omit %s; got %#v", deny, names)
		}
	}
}

func TestIsPackageSourceFile(t *testing.T) {
	cases := map[string]bool{
		"backend/main.go":         true,
		"backend/go.mod":          true,
		"backend/plugin":          false,
		"frontend/settings.mjs":   false,
		"frontend/settings.css":   false,
		"frontend/settings.mjs.map": true,
		"frontend/Widget.vue":     true,
		"manifest/settings.json":  false,
		"sforum.extension.json":   false,
		"README.md":               false,
	}
	for path, want := range cases {
		if got := isPackageSourceFile(path); got != want {
			t.Fatalf("isPackageSourceFile(%q)=%v want %v", path, got, want)
		}
	}
}

func zipEntryNames(t *testing.T, zipPath string) map[string]bool {
	t.Helper()
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	out := make(map[string]bool, len(r.File))
	for _, f := range r.File {
		out[f.Name] = true
	}
	return out
}

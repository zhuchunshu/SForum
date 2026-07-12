package webreleaseruntime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHostSourceExcludedSkipsNestedNodeModules(t *testing.T) {
	cases := []struct {
		path      string
		directory bool
		want      bool
	}{
		{"node_modules", true, true},
		{"node_modules/vue", true, true},
		{"packages/admin-sdk/node_modules", true, true},
		{"packages/admin-sdk/node_modules/.bin", true, true},
		{"packages/admin-sdk/node_modules/.bin/nuxi", false, true},
		{"packages/admin-sdk/src/index.ts", false, false},
		{".nuxt", true, true},
		{"app/pages/index.vue", false, false},
	}
	for _, tc := range cases {
		if got := hostSourceExcluded(tc.path, tc.directory); got != tc.want {
			t.Fatalf("hostSourceExcluded(%q, %v)=%v, want %v", tc.path, tc.directory, got, tc.want)
		}
	}
}

func TestDigestWebSourceIgnoresWorkspacePackageNodeModulesSymlinks(t *testing.T) {
	root := t.TempDir()
	// Minimal host tree: package.json + bun.lock + nested workspace peer symlink.
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"web"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bun.lock"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	sdk := filepath.Join(root, "packages", "admin-sdk")
	if err := os.MkdirAll(filepath.Join(sdk, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sdk, "src", "index.ts"), []byte("export {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(sdk, "node_modules", ".bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "packages", "admin-sdk", "src", "index.ts")
	if err := os.Symlink(target, filepath.Join(bin, "nuxi")); err != nil {
		t.Fatal(err)
	}

	digest, err := digestWebSource(root)
	if err != nil {
		t.Fatalf("digestWebSource: %v", err)
	}
	if digest == "" {
		t.Fatal("expected non-empty digest")
	}

	host, err := CompositionHost(root)
	if err != nil {
		t.Fatalf("CompositionHost: %v", err)
	}
	if host.WebSource != digest {
		t.Fatalf("CompositionHost.WebSource=%q digest=%q", host.WebSource, digest)
	}
}

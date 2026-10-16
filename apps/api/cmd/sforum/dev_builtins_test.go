package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M0 ADR 6.2.5：内置清单必须从清单树发现，脚本里不得再抄一份 id 列表。
func TestBuiltinDiscoveryIsTheOnlyBuiltinList(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	packages, err := discoverBuiltinPackages(filepath.Join(repoRoot, "extensions", "builtin"))
	if err != nil {
		t.Fatal(err)
	}
	withBackend := map[string]string{}
	for _, pkg := range packages {
		if pkg.HasGoBackend {
			withBackend[pkg.ID] = pkg.BackendEntry
		}
		if pkg.Type != "plugin" && pkg.Type != "theme" {
			t.Fatalf("unexpected package type %q for %s", pkg.Type, pkg.ID)
		}
	}
	if len(withBackend) < 8 {
		t.Fatalf("discovered %d built-ins with a Go backend: %#v", len(withBackend), withBackend)
	}
	for id, entry := range withBackend {
		if entry != "backend/plugin" {
			t.Fatalf("built-in %s backend entry = %q", id, entry)
		}
	}
	// 主题没有后端，必须被识别为不可编译而不是被当成插件。
	var themes int
	for _, pkg := range packages {
		if pkg.Type == "theme" {
			themes++
			if pkg.HasGoBackend {
				t.Fatalf("theme %s reported a Go backend", pkg.ID)
			}
		}
	}
	if themes == 0 {
		t.Fatalf("no built-in theme discovered; discovery must cover themes too")
	}

	// 脚本必须消费发现结果，不得再硬编码任何内置 id。
	script, err := os.ReadFile(filepath.Join(repoRoot, "scripts", "build-builtin-plugins.sh"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(script)
	if !strings.Contains(body, "extension builtins") {
		t.Fatalf("build script does not consume discovery")
	}
	for id := range withBackend {
		if strings.Contains(body, id) {
			t.Fatalf("build script still hardcodes built-in id %s", id)
		}
	}
	for _, directory := range []string{"sforum-smtp", "sforum-shortcodes", "sforum-auth-github"} {
		if strings.Contains(body, directory) {
			t.Fatalf("build script still hardcodes built-in directory %s", directory)
		}
	}
}

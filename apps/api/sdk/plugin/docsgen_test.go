package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateCatalogDocsNonEmpty(t *testing.T) {
	docs := GenerateCatalogDocs()
	for _, name := range DocFileNames() {
		body, ok := docs.Files[name]
		if !ok || body == "" {
			t.Fatalf("missing generated doc %s", name)
		}
		if !strings.Contains(body, genMarker) {
			t.Fatalf("%s missing gen marker", name)
		}
		if !strings.Contains(body, "sforum extension docs generate") {
			t.Fatalf("%s missing regenerate instructions", name)
		}
	}
	// 关键目录项应出现在对应页面。
	if !strings.Contains(docs.Files[DocEvents], "topic.created") {
		t.Fatal("events.md should list topic.created")
	}
	if !strings.Contains(docs.Files[DocCapabilities], "host.api") {
		t.Fatal("capabilities.md should list host.api")
	}
	if !strings.Contains(docs.Files[DocContributionPoints], "forum.topic.actions") {
		t.Fatal("contribution-points.md should list forum.topic.actions")
	}
	if !strings.Contains(docs.Files[DocProviderSlots], "mail.provider") {
		t.Fatal("provider-slots.md should list mail.provider")
	}
	if !strings.Contains(docs.Files[DocSchedules], "identity.cleanup_sessions") {
		t.Fatal("schedules.md should list identity.cleanup_sessions")
	}
}

func TestWriteAndCheckCatalogDocs(t *testing.T) {
	dir := t.TempDir()
	if err := WriteCatalogDocs(dir); err != nil {
		t.Fatal(err)
	}
	ok, msg := CheckCatalogDocs(dir)
	if !ok {
		t.Fatalf("fresh write should match: %s", msg)
	}
	// 人为漂移。
	path := filepath.Join(dir, DocEvents)
	if err := os.WriteFile(path, []byte("tampered\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, msg = CheckCatalogDocs(dir)
	if ok {
		t.Fatal("expected drift detection")
	}
	if !strings.Contains(msg, DocEvents) {
		t.Fatalf("drift message should mention events.md: %s", msg)
	}
}

func TestCommittedCatalogDocsInSync(t *testing.T) {
	// 锁定仓库内 docs/extensions/catalogs 与代码目录一致（F4.2 防漂移）。
	root := findRepoRootForTest(t)
	dir := filepath.Join(root, DefaultCatalogDocsDir)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("catalog docs dir missing (run sforum extension docs generate): %v", err)
	}
	ok, msg := CheckCatalogDocs(dir)
	if !ok {
		t.Fatalf("catalog docs out of date:\n%s\n\nRegenerate with:\n  cd apps/api && go run ./cmd/sforum extension docs generate", msg)
	}
}

func findRepoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "extensions")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repo root")
		}
		dir = parent
	}
}

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
	if !strings.Contains(docs.Files[DocFamilies], FamilyHooks) {
		t.Fatal("families.md should list hooks family")
	}
	if !strings.Contains(docs.Files[DocHooks], "topic.before_create") {
		t.Fatal("hooks.md should list host event names")
	}
	if !strings.Contains(docs.Files[DocServices], "ServiceDiscoveryService") {
		t.Fatal("services.md should document discovery RPCs")
	}
	if !strings.Contains(docs.Files[DocJobs], "ExecuteJob") {
		t.Fatal("jobs.md should document ExecuteJob")
	}
	if !strings.Contains(docs.Files[DocCommands], "InvokeCommand") {
		t.Fatal("commands.md should document InvokeCommand")
	}
	// schedules 必须诚实声明：无 List/Trigger helper，Host 未注册 ScheduleService。
	if !strings.Contains(docs.Files[DocSchedules], "ScheduleService") {
		t.Fatal("schedules.md must mention ScheduleService wire boundary")
	}
	if !strings.Contains(docs.Files[DocSchedules], "does not register") &&
		!strings.Contains(docs.Files[DocSchedules], "not callable") {
		t.Fatal("schedules.md must honestly deny a registered/callable ScheduleService path")
	}
	if strings.Contains(docs.Files[DocSchedules], "Host.ScheduleList") ||
		strings.Contains(docs.Files[DocSchedules], "ScheduleService helper methods") {
		t.Fatal("schedules.md must not invent a public ScheduleService helper")
	}
	if !strings.Contains(docs.Files[DocHooks], "resultSchemaID+\".patch@\"+version") &&
		!strings.Contains(docs.Files[DocHooks], `.patch@`) {
		t.Fatal("hooks.md must document Host-derived filter patch schema convention")
	}
	if !strings.Contains(docs.Files[DocProviderSlots], "invoke") ||
		!strings.Contains(docs.Files[DocProviderSlots], "probe") {
		t.Fatal("provider-slots.md must document invoke-only versioned path and legacy probe boundary")
	}
	if strings.Contains(docs.Files[DocProviderSlots], "for example `invoke`, `probe`, `send`") {
		t.Fatal("provider-slots.md must not claim probe/send are accepted by ProviderRegistry")
	}
	for name, body := range map[string]string{
		DocProviderSlots: docs.Files[DocProviderSlots],
		DocServices:      docs.Files[DocServices],
		DocSchedules:     docs.Files[DocSchedules],
	} {
		if !strings.Contains(body, "id@positiveVersion") {
			t.Fatalf("%s must document the contractVersion=id@positiveVersion shape", name)
		}
	}
	if !strings.Contains(docs.Files[DocServices], "ServiceDefinition.Version") ||
		!strings.Contains(docs.Files[DocServices], "strict SemVer") ||
		!strings.Contains(docs.Files[DocServices], "major") ||
		!strings.Contains(docs.Files[DocServices], "@N") {
		t.Fatal("services.md must document the SDK SemVer-to-Manifest contract-major rule")
	}
	fields, _, _, _ := manifestV3SchemaFields()
	if len(fields) < 40 {
		t.Fatalf("Manifest V3 root field catalog is incomplete: %d", len(fields))
	}
	for _, field := range fields {
		if !strings.Contains(docs.Files[DocManifestV3], "`"+field+"`") {
			t.Fatalf("manifest-v3.md should list schema field %s", field)
		}
	}
	for _, phrase := range []string{"extension digest --write", "manifestVersion: 3", "exact-artifact trust review"} {
		if !strings.Contains(docs.Files[DocManifestV3], phrase) {
			t.Fatalf("manifest-v3.md missing authoring contract %q", phrase)
		}
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

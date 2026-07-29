package plugin_test

import (
	"path/filepath"
	"runtime"
	"testing"

	pluginsdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// apps/api/sdk/plugin -> repo root
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
}

func fixtureRoot(t *testing.T, rel string) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "extensions/fixtures", rel)
}

func TestFixtureEventsContract(t *testing.T) {
	root := fixtureRoot(t, "plugins/sforum-contract-events")
	report, err := pluginsdk.LoadAndTest(root, pluginsdk.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK {
		t.Fatalf("expected pass, got errors=%d checks=%#v", report.Errors, report.Checks)
	}
	if report.Manifest.ID != "sforum.contract.events" {
		t.Fatalf("id=%s", report.Manifest.ID)
	}
	// 事件与贡献点应出现在 ok checks 中。
	codes := checkCodes(report)
	for _, want := range []string{"manifest.ok", "event.known", "contribution.point_ok"} {
		if !codes[want] {
			t.Fatalf("missing check %s in %#v", want, report.Checks)
		}
	}
}

func TestFixtureSchedulesContract(t *testing.T) {
	root := fixtureRoot(t, "plugins/sforum-contract-schedules")
	report, err := pluginsdk.LoadAndTest(root, pluginsdk.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK {
		t.Fatalf("expected pass: %#v", report.Checks)
	}
	// 宿主 schedule 目录必须非空；fixture 本身不声明 cron。
	schedules := pluginsdk.CoreSchedules()
	if len(schedules) == 0 {
		t.Fatal("core schedules empty")
	}
	foundIdentity := false
	for _, s := range schedules {
		if s.ID == "identity.cleanup_sessions" {
			foundIdentity = true
			break
		}
	}
	if !foundIdentity {
		t.Fatalf("missing identity.cleanup_sessions in %#v", schedules)
	}
}

func checkCodes(report pluginsdk.Report) map[string]bool {
	out := map[string]bool{}
	for _, c := range report.Checks {
		out[c.Code] = true
	}
	return out
}

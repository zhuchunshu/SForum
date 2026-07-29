package version

import (
	"bytes"
	"strings"
	"testing"
)

func TestGetExposesProjectAndInjectedBuildIdentity(t *testing.T) {
	originalCurrent, originalCommit, originalBuildTime := Current, Commit, BuildTime
	t.Cleanup(func() {
		Current, Commit, BuildTime = originalCurrent, originalCommit, originalBuildTime
	})
	Current = "2.8.0"
	Commit = "0123456789abcdef"
	BuildTime = "2026-07-29T10:00:00+08:00"

	info := Get()
	if info.Name != Name || info.Version != "2.8.0" || info.Commit != Commit || info.BuiltAt != BuildTime {
		t.Fatalf("unexpected build info: %#v", info)
	}
	if info.GoVersion == "" || info.SourceURL != SourceURL {
		t.Fatalf("missing runtime project identity: %#v", info)
	}
}

func TestPrintIfRequestedIsExactAndCompact(t *testing.T) {
	originalCurrent, originalCommit, originalBuildTime := Current, Commit, BuildTime
	t.Cleanup(func() {
		Current, Commit, BuildTime = originalCurrent, originalCommit, originalBuildTime
	})
	Current = "2.8.0"
	Commit = "0123456789abcdef"
	BuildTime = ""

	var output bytes.Buffer
	if !PrintIfRequested(&output, []string{"--version"}) {
		t.Fatal("expected --version to be handled")
	}
	if got := strings.TrimSpace(output.String()); got != "SForum 2.8.0 (0123456789ab)" {
		t.Fatalf("unexpected version output %q", got)
	}
	if PrintIfRequested(&output, []string{"serve", "--version"}) {
		t.Fatal("expected non-exact arguments to continue to the command")
	}
}

func TestDevelopmentVersionIncludesShortCommitOnce(t *testing.T) {
	originalCurrent, originalCommit, originalBuildTime := Current, Commit, BuildTime
	t.Cleanup(func() {
		Current, Commit, BuildTime = originalCurrent, originalCommit, originalBuildTime
	})
	Current = "dev"
	Commit = "0123456789abcdef"
	BuildTime = ""

	info := Get()
	if info.Version != "dev-01234" {
		t.Fatalf("development version = %q", info.Version)
	}
	if summary := Summary(); !strings.HasPrefix(summary, "SForum dev-01234") || strings.Contains(summary, "(0123456789ab)") {
		t.Fatalf("development summary = %q", summary)
	}
}

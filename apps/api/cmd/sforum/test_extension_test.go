package main

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExtensionTestCommandEventsFixture(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
	fixture := filepath.Join(repoRoot, "extensions/fixtures/plugins/sforum-contract-events")

	cmd := newRootCommand()
	cmd.SetArgs([]string{"extension", "test", fixture})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("extension test: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "PASS") {
		t.Fatalf("expected PASS:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "sforum.contract.events") {
		t.Fatalf("expected id:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "event.known") {
		t.Fatalf("expected event.known check:\n%s", out.String())
	}
}

func TestExtensionTestCommandSMTPWithSkipBinary(t *testing.T) {
	// SMTP 包 backend/plugin 可能是构建产物或缺失；用 skip 做 CLI 冒烟。
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
	smtpRoot := filepath.Join(repoRoot, "extensions/builtin/plugins/sforum-smtp")

	cmd := newRootCommand()
	cmd.SetArgs([]string{"extension", "test", "--skip-backend-binary", smtpRoot})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("extension test smtp: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "sforum.smtp") {
		t.Fatalf("expected smtp id:\n%s", out.String())
	}
}

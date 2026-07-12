package main

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExtensionDocsGenerateCheck(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// 确保 CLI --check 与仓库内已提交文档一致。
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
	_ = repoRoot

	cmd := newRootCommand()
	cmd.SetArgs([]string{"extension", "docs", "generate", "--check"})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	// findRepoRoot 从 cwd 向上找；测试 cwd 可能在 apps/api。
	if err := cmd.Execute(); err != nil {
		t.Fatalf("docs generate --check: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "in sync") {
		t.Fatalf("expected in sync message:\n%s", out.String())
	}
}

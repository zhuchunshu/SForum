package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDevCleanupOrphanPluginsCommandRegistered(t *testing.T) {
	root := newRootCommand()
	cmd, _, err := root.Find([]string{"dev:cleanup-orphan-plugins"})
	if err != nil || cmd == nil {
		t.Fatalf("command not registered: %v", err)
	}
	if !strings.Contains(cmd.Short, "orphan") && !strings.Contains(cmd.Use, "cleanup-orphan") {
		t.Fatalf("unexpected command: %#v", cmd)
	}
}

func TestDevCleanupOrphanPluginsDryRunExecutesRealPath(t *testing.T) {
	root := newRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"dev:cleanup-orphan-plugins", "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute dry-run: %v\n%s", err, buf.String())
	}
	out := buf.String()
	// 真实 CleanupOrphanExtensionPlugins 路径：无孤儿时打印 no orphan…；有则 DRY_RUN would stop
	if !strings.Contains(out, "no orphan") && !strings.Contains(out, "DRY_RUN would stop") && !strings.Contains(out, "selected") {
		t.Fatalf("unexpected dry-run output: %q", out)
	}
}

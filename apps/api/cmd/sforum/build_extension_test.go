package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExtensionBuildRunsFrozenFrontendBuildAndPackageGates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake Bun executable uses a POSIX shell")
	}
	root := filepath.Join(t.TempDir(), "plugin")
	_, err := GenerateExtensionScaffold(makeOptions{
		Kind: "plugin", ID: "acme.build", Name: "Acme Build", Description: "Build command fixture.",
		URL: "https://example.com/build", AuthorName: "Acme", Out: root, NoInteraction: true,
		VueAdminPage: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	frontendRoot := filepath.Join(root, "frontend", "admin")
	if err := os.WriteFile(filepath.Join(frontendRoot, "bun.lock"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(t.TempDir(), "bun.log")
	binDir := t.TempDir()
	fakeBun := filepath.Join(binDir, "bun")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$SFORUM_TEST_BUN_LOG\"\n"
	if err := os.WriteFile(fakeBun, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SFORUM_TEST_BUN_LOG", logPath)

	cmd := newRootCommand()
	cmd.SetArgs([]string{"extension", "build", "--allow-scaffold", root})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("extension build: %v\n%s", err, out.String())
	}
	logBody, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(logBody) != "install --frozen-lockfile\nrun build\n" {
		t.Fatalf("unexpected Bun calls: %q", logBody)
	}
	for _, expected := range []string{"refreshing exact package digests", "validating extension package", "running extension contract tests", "PASS"} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("build output missing %q:\n%s", expected, out.String())
		}
	}
}

func TestExtensionBuildSkipInstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake Bun executable uses a POSIX shell")
	}
	root := filepath.Join(t.TempDir(), "plugin")
	_, err := GenerateExtensionScaffold(makeOptions{
		Kind: "plugin", ID: "acme.skip-install", Name: "Acme Skip", Description: "Skip install fixture.",
		URL: "https://example.com/skip", AuthorName: "Acme", Out: root, NoInteraction: true,
		VueAdminPage: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(t.TempDir(), "bun.log")
	binDir := t.TempDir()
	fakeBun := filepath.Join(binDir, "bun")
	if err := os.WriteFile(fakeBun, []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$SFORUM_TEST_BUN_LOG\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SFORUM_TEST_BUN_LOG", logPath)

	cmd := newRootCommand()
	cmd.SetArgs([]string{"extension", "build", "--skip-install", "--allow-scaffold", root})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("extension build --skip-install: %v\n%s", err, out.String())
	}
	logBody, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(logBody) != "run build\n" {
		t.Fatalf("unexpected Bun calls: %q", logBody)
	}
}

func TestExtensionBuildWithoutFrontendStillRunsPackageGates(t *testing.T) {
	root := filepath.Join(t.TempDir(), "plugin")
	_, err := GenerateExtensionScaffold(makeOptions{
		Kind: "plugin", ID: "acme.backendless", Name: "Acme Backendless", Description: "No frontend fixture.",
		URL: "https://example.com/backendless", AuthorName: "Acme", Out: root, NoInteraction: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	cmd := newRootCommand()
	cmd.SetArgs([]string{"extension", "build", root})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("extension build without frontend: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "skipping frontend build") || !strings.Contains(out.String(), "PASS") {
		t.Fatalf("unexpected output:\n%s", out.String())
	}
}

func TestExtensionBuildRejectsSymlinkLockfile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink setup differs on Windows")
	}
	root := filepath.Join(t.TempDir(), "plugin")
	_, err := GenerateExtensionScaffold(makeOptions{
		Kind: "plugin", ID: "acme.symlink-lock", Name: "Acme Symlink", Description: "Symlink lock fixture.",
		URL: "https://example.com/symlink", AuthorName: "Acme", Out: root, NoInteraction: true,
		VueAdminPage: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	frontendRoot := filepath.Join(root, "frontend", "admin")
	lockTarget := filepath.Join(t.TempDir(), "bun.lock")
	if err := os.WriteFile(lockTarget, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(lockTarget, filepath.Join(frontendRoot, "bun.lock")); err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "bun"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cmd := newRootCommand()
	cmd.SetArgs([]string{"extension", "build", root})
	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "lockfile is not a regular file") {
		t.Fatalf("expected symlink lockfile rejection, got %v", err)
	}
}

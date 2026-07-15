package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type publicL2Browser struct {
	command string
	workdir string
	session string
	env     []string
	closed  bool
}

func startPublicL2Browser(t *testing.T, repositoryRoot, baseURL string) *publicL2Browser {
	t.Helper()
	command := publicL2PlaywrightCLI(t)
	workdir := t.TempDir()
	browser := &publicL2Browser{
		command: command,
		workdir: workdir,
		session: fmt.Sprintf("sforum-p9-public-l2-%d", time.Now().UnixNano()),
	}
	browser.env = themeE2EEnvironment(map[string]string{
		"PLAYWRIGHT_CLI_SESSION":    browser.session,
		"PLAYWRIGHT_CLI_OUTPUT_DIR": filepath.Join(workdir, "output"),
	})
	t.Cleanup(func() { browser.close(t) })
	// The health page establishes the isolated random origin without loading L2
	// before the request listener in the browser assertion is installed.
	browser.call(t, 90*time.Second, "open", strings.TrimRight(baseURL, "/")+"/health")
	return browser
}

func (b *publicL2Browser) run(t *testing.T, filename string) {
	t.Helper()
	source, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("public L2 browser assertion %s is unavailable: %v", filename, err)
	}
	output := b.call(t, 90*time.Second, "run-code", string(source))
	if strings.Contains(output, "### Error") {
		t.Fatalf("public L2 browser assertion failed:\n%s", output)
	}
}

func (b *publicL2Browser) close(t *testing.T) {
	t.Helper()
	if b == nil || b.closed {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, b.command, "close")
	command.Dir = b.workdir
	command.Env = b.env
	output, err := command.CombinedOutput()
	if err != nil {
		t.Logf("close isolated Playwright session %s: %v (context: %v)\n%s", b.session, err, ctx.Err(), output)
		return
	}
	b.closed = true
}

func (b *publicL2Browser) call(t *testing.T, timeout time.Duration, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, b.command, args...)
	command.Dir = b.workdir
	command.Env = b.env
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run isolated Playwright session %s (%s): %v (context: %v)\n%s",
			b.session, strings.Join(args, " "), err, ctx.Err(), output)
	}
	return string(output)
}

func publicL2PlaywrightCLI(t *testing.T) string {
	t.Helper()
	if configured := strings.TrimSpace(os.Getenv("SFORUM_PLAYWRIGHT_CLI")); configured != "" {
		path, err := filepath.Abs(configured)
		if err != nil {
			t.Fatal(err)
		}
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("SFORUM_PLAYWRIGHT_CLI=%s is unavailable: %v", configured, err)
		}
		return path
	}
	home, err := os.UserHomeDir()
	if err == nil {
		bundled := filepath.Join(home, ".codex", "skills", "playwright", "scripts", "playwright_cli.sh")
		if info, statErr := os.Stat(bundled); statErr == nil && info.Mode().IsRegular() {
			return bundled
		}
	}
	if command, lookErr := exec.LookPath("playwright-cli"); lookErr == nil {
		return command
	}
	t.Fatal("public L2 E2E requires SFORUM_PLAYWRIGHT_CLI or playwright-cli on PATH")
	return ""
}

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

func writeLocalSDKPackage(t *testing.T, sdkDir, directory, name, version string) {
	t.Helper()
	target := filepath.Join(sdkDir, directory)
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "{\n  \"name\": \"" + name + "\",\n  \"version\": \"" + version + "\"\n}\n"
	if err := os.WriteFile(filepath.Join(target, "package.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fakeBunRecordingPackageJSON 记录每次调用参数，并把工作区的 package.json 抄进日志，
// 这样测试能观察到「安装时依赖确实被改写成本地 SDK」。
func fakeBunRecordingPackageJSON(t *testing.T) string {
	t.Helper()
	logPath := filepath.Join(t.TempDir(), "bun.log")
	binDir := t.TempDir()
	// 同时模拟 bun 写锁文件：SDK 链接后的锁文件也必须被还原，否则作者下一次
	// 普通的 bun install --frozen-lockfile 会因为 file: 记录而失败。
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$SFORUM_TEST_BUN_LOG\"\ncat package.json >> \"$SFORUM_TEST_BUN_LOG\"\nprintf '{\"linked\":true}\\n' > bun.lock\n"
	if err := os.WriteFile(filepath.Join(binDir, "bun"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SFORUM_TEST_BUN_LOG", logPath)
	return logPath
}

func TestExtensionBuildLinksLocalSDKPackagesForDevelopment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake Bun executable uses a POSIX shell")
	}
	t.Setenv("SFORUM_SDK_DIR", "")
	root := filepath.Join(t.TempDir(), "plugin")
	if _, err := GenerateExtensionScaffold(makeOptions{
		Kind: "plugin", ID: "acme.sdk-link", Name: "Acme SDK Link", Description: "Local SDK fixture.",
		URL: "https://example.com/sdk", AuthorName: "Acme", Out: root, NoInteraction: true,
		VueAdminPage: true,
	}); err != nil {
		t.Fatal(err)
	}
	packageJSON := filepath.Join(root, "frontend", "admin", "package.json")
	original, err := os.ReadFile(packageJSON)
	if err != nil {
		t.Fatal(err)
	}

	sdkDir := t.TempDir()
	writeLocalSDKPackage(t, sdkDir, "admin-sdk", "@sforum/admin-sdk", "1.1.0")
	writeLocalSDKPackage(t, sdkDir, "plugin-ui", "@sforum/plugin-ui", "1.1.0")
	writeLocalSDKPackage(t, sdkDir, "unrelated", "@sforum/unrelated", "9.9.9")

	logPath := fakeBunRecordingPackageJSON(t)
	cmd := newRootCommand()
	cmd.SetArgs([]string{"extension", "build", "--allow-scaffold", "--sdk-dir", sdkDir, root})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("extension build --sdk-dir: %v\n%s", err, out.String())
	}

	logBody, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	log := string(logBody)
	for _, expected := range []string{
		filepath.Join(sdkDir, "admin-sdk"),
		filepath.Join(sdkDir, "plugin-ui"),
	} {
		if !strings.Contains(log, `"file:`+expected+`"`) {
			t.Fatalf("install did not see a local SDK link for %s:\n%s", expected, log)
		}
	}
	if strings.Contains(log, "@sforum/unrelated") {
		t.Fatalf("workspace must not gain undeclared packages:\n%s", log)
	}
	if !strings.Contains(out.String(), "linking local SDK packages (development only)") {
		t.Fatalf("missing development link notice:\n%s", out.String())
	}

	// 构建结束后作者的文件必须逐字节恢复，且不留下备份。
	restored, err := os.ReadFile(packageJSON)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != string(original) {
		t.Fatalf("workspace package.json was not restored:\n%s", restored)
	}
	if _, err := os.Stat(packageJSON + ".sforum-sdk-backup"); !os.IsNotExist(err) {
		t.Fatalf("SDK backup should be removed after a successful build, got %v", err)
	}
}

func TestExtensionBuildRejectsUnusableSDKDir(t *testing.T) {
	t.Setenv("SFORUM_SDK_DIR", "")
	root := filepath.Join(t.TempDir(), "plugin")
	if _, err := GenerateExtensionScaffold(makeOptions{
		Kind: "plugin", ID: "acme.sdk-bad", Name: "Acme SDK Bad", Description: "Bad SDK fixture.",
		URL: "https://example.com/badsdk", AuthorName: "Acme", Out: root, NoInteraction: true,
		VueAdminPage: true,
	}); err != nil {
		t.Fatal(err)
	}
	packageJSON := filepath.Join(root, "frontend", "admin", "package.json")
	before, err := os.ReadFile(packageJSON)
	if err != nil {
		t.Fatal(err)
	}

	notADirectory := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(notADirectory, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	noMatchingPackage := t.TempDir()
	writeLocalSDKPackage(t, noMatchingPackage, "other", "@acme/other", "1.0.0")

	for name, sdkDir := range map[string]string{
		"missing directory": filepath.Join(t.TempDir(), "absent"),
		"not a directory":   notADirectory,
		"no match":          noMatchingPackage,
	} {
		out := strings.Builder{}
		cmd := newRootCommand()
		cmd.SetArgs([]string{"extension", "build", "--allow-scaffold", "--sdk-dir", sdkDir, root})
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		if err := cmd.Execute(); err == nil {
			t.Fatalf("%s: expected --sdk-dir rejection", name)
		}
		after, err := os.ReadFile(packageJSON)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != string(before) {
			t.Fatalf("%s: workspace package.json changed on a rejected build:\n%s", name, after)
		}
	}
}

func TestExtensionBuildRestoresLockfileAfterSDKLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake Bun executable uses a POSIX shell")
	}
	t.Setenv("SFORUM_SDK_DIR", "")
	root := filepath.Join(t.TempDir(), "plugin")
	if _, err := GenerateExtensionScaffold(makeOptions{
		Kind: "plugin", ID: "acme.sdk-lock", Name: "Acme SDK Lock", Description: "Lock restore fixture.",
		URL: "https://example.com/sdklock", AuthorName: "Acme", Out: root, NoInteraction: true,
		VueAdminPage: true,
	}); err != nil {
		t.Fatal(err)
	}
	sdkDir := t.TempDir()
	writeLocalSDKPackage(t, sdkDir, "admin-sdk", "@sforum/admin-sdk", "1.1.0")
	writeLocalSDKPackage(t, sdkDir, "plugin-ui", "@sforum/plugin-ui", "1.1.0")

	frontendRoot := filepath.Join(root, "frontend", "admin")
	lockPath := filepath.Join(frontendRoot, "bun.lock")
	fakeBunRecordingPackageJSON(t)

	// 情况一：构建前没有锁文件，构建后也不该留下 bun 写入的那份。
	build := func() {
		t.Helper()
		cmd := newRootCommand()
		cmd.SetArgs([]string{"extension", "build", "--allow-scaffold", "--sdk-dir", sdkDir, root})
		var out strings.Builder
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("extension build --sdk-dir: %v\n%s", err, out.String())
		}
	}
	build()
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("a lockfile created only by the linked build must be removed, got %v", err)
	}

	// 情况二：构建前有锁文件，构建后逐字节还原。
	if err := os.WriteFile(lockPath, []byte("original-lock-content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	build()
	restored, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != "original-lock-content\n" {
		t.Fatalf("lockfile must be restored after an SDK-linked build, got %q", restored)
	}
}

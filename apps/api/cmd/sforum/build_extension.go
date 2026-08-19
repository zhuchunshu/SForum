package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	pluginsdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin"
)

type extensionBuildOptions struct {
	SkipInstall   bool
	AllowScaffold bool
}

func newExtensionBuildCommand() *cobra.Command {
	var opts extensionBuildOptions
	cmd := &cobra.Command{
		Use:   "build [path]",
		Short: "Build author-side frontend assets and run exact package gates",
		Long: `Build frontend/admin with Bun when package.json is present, refresh exact
digests, validate the package, and run Host contract tests. This author command
executes package scripts locally; installation and production runtime never call it.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			return runExtensionBuild(cmd, root, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.SkipInstall, "skip-install", false, "Skip bun install and use existing frontend dependencies")
	cmd.Flags().BoolVar(&opts.AllowScaffold, "allow-scaffold", false, "Allow contract tests before the backend binary exists")
	return cmd
}

func runExtensionBuild(cmd *cobra.Command, root string, opts extensionBuildOptions) error {
	abs, err := resolveExtensionPackageRoot(root)
	if err != nil {
		return err
	}
	rootInfo, err := os.Lstat(abs)
	if err != nil {
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("package root is a symlink: %s", abs)
	}
	if err := buildExtensionAdminFrontend(cmd, abs, opts.SkipInstall); err != nil {
		return err
	}
	cmd.Println("refreshing exact package digests")
	if err := runExtensionDigest(cmd, abs, true); err != nil {
		return err
	}
	cmd.Println("validating extension package")
	if err := runExtensionValidate(cmd, abs, false); err != nil {
		return err
	}
	cmd.Println("running extension contract tests")
	return runExtensionTest(cmd, abs, pluginsdk.Options{SkipBackendBinary: opts.AllowScaffold}, false)
}

func buildExtensionAdminFrontend(cmd *cobra.Command, packageRoot string, skipInstall bool) error {
	frontendRoot := filepath.Join(packageRoot, "frontend", "admin")
	packageJSON := filepath.Join(frontendRoot, "package.json")
	info, err := os.Lstat(packageJSON)
	if errors.Is(err, fs.ErrNotExist) {
		cmd.Println("no frontend/admin/package.json; skipping frontend build")
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect admin frontend: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("admin frontend package is not a regular file: %s", packageJSON)
	}
	bun, err := exec.LookPath("bun")
	if err != nil {
		return errors.New("Bun is required to build frontend/admin; install Bun or build the frontend before running this command")
	}

	if !skipInstall {
		installArgs := []string{"install"}
		hasLock, err := hasBunLock(frontendRoot)
		if err != nil {
			return err
		}
		if hasLock {
			installArgs = append(installArgs, "--frozen-lockfile")
		}
		cmd.Printf("running bun %s in %s\n", strings.Join(installArgs, " "), frontendRoot)
		if err := runExtensionBuildProcess(cmd, bun, frontendRoot, installArgs...); err != nil {
			return err
		}
	} else {
		cmd.Println("skipping frontend dependency installation")
	}
	cmd.Printf("running bun run build in %s\n", frontendRoot)
	return runExtensionBuildProcess(cmd, bun, frontendRoot, "run", "build")
}

func hasBunLock(root string) (bool, error) {
	for _, name := range []string{"bun.lock", "bun.lockb"} {
		path := filepath.Join(root, name)
		info, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return false, fmt.Errorf("inspect Bun lockfile %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("Bun lockfile is not a regular file: %s", path)
		}
		return true, nil
	}
	return false, nil
}

func runExtensionBuildProcess(cmd *cobra.Command, executable, dir string, args ...string) error {
	process := exec.CommandContext(cmd.Context(), executable, args...)
	process.Dir = dir
	process.Stdin = cmd.InOrStdin()
	process.Stdout = cmd.OutOrStdout()
	process.Stderr = cmd.ErrOrStderr()
	if err := process.Run(); err != nil {
		return fmt.Errorf("%s %v failed: %w", filepath.Base(executable), args, err)
	}
	return nil
}

package webreleaseruntime

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

func verifySnapshot(root string, expected string) error {
	actual, err := extensionpackage.DigestTree(root)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("snapshot digest expected %s, got %s", expected, actual)
	}
	return nil
}

func copyTree(source string, target string, excluded func(string, bool) bool) error {
	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	return filepath.WalkDir(source, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, current)
		if err != nil {
			return err
		}
		if relative == "." {
			return os.MkdirAll(target, 0o755)
		}
		if excluded != nil && excluded(filepath.ToSlash(relative), entry.IsDir()) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("symbolic link is not allowed in build input: %s", current)
		}
		destination := filepath.Join(target, relative)
		if info.IsDir() {
			return os.MkdirAll(destination, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular build input: %s", current)
		}
		return copyRegularFile(current, destination, info.Mode())
	})
}

func copyRegularFile(source string, target string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		_ = input.Close()
		return err
	}
	_, copyErr := io.Copy(output, input)
	inputErr, outputErr := input.Close(), output.Close()
	if copyErr != nil {
		return copyErr
	}
	if inputErr != nil {
		return inputErr
	}
	return outputErr
}

func hostSourceExcluded(relative string, directory bool) bool {
	first := strings.Split(relative, "/")[0]
	if directory {
		switch first {
		case "node_modules", ".nuxt", ".nuxt-build", ".nuxt-typecheck", ".output", ".cache", "coverage", "playwright-report", "test-results":
			return true
		}
	}
	base := filepath.Base(relative)
	return base == ".env" || strings.HasPrefix(base, ".env.")
}

func linkHostDependencies(webRoot string, workspace string) error {
	source, err := filepath.Abs(filepath.Join(webRoot, "node_modules"))
	if err != nil {
		return err
	}
	if info, err := os.Stat(source); err != nil || !info.IsDir() {
		return fmt.Errorf("host node_modules is unavailable")
	}
	return os.Symlink(source, filepath.Join(workspace, "node_modules"))
}

func absolutePath(value string) string {
	absolute, err := filepath.Abs(value)
	if err != nil {
		return filepath.Clean(value)
	}
	return absolute
}

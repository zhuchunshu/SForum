package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

// 脚手架 digest 占位符：__BACKEND_DIGEST__ / __OPENAPI_DIGEST__ 等。
var manifestDigestPlaceholder = regexp.MustCompile(`__[A-Z][A-Z0-9_]*__`)

func newExtensionDigestCommand() *cobra.Command {
	var write bool
	cmd := &cobra.Command{
		Use:   "digest [path]",
		Short: "Inspect or refresh exact Manifest V3 package-file digests",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			abs, err := filepath.Abs(root)
			if err != nil {
				return err
			}
			manifestPath := filepath.Join(abs, extensionmanifest.ManifestFileName)
			body, err := os.ReadFile(manifestPath)
			if err != nil {
				// 正式打包路径：仅有 .tmpl 时先 materialize 占位 digest，再由 --write 刷新真实摘要。
				// 测试与作者不得再手算 SHA 替换 token。
				if !os.IsNotExist(err) {
					return err
				}
				tmplPath := filepath.Join(abs, extensionmanifest.ManifestFileName+".tmpl")
				tmplBody, tmplErr := os.ReadFile(tmplPath)
				if tmplErr != nil {
					return fmt.Errorf("read manifest: %w (also no %s: %v)", err, filepath.Base(tmplPath), tmplErr)
				}
				body = []byte(materializeManifestTemplate(string(tmplBody)))
				if writeErr := os.WriteFile(manifestPath, body, 0o600); writeErr != nil {
					return writeErr
				}
				cmd.Printf("materialized %s from template\n", manifestPath)
			}
			var manifest map[string]any
			if err := json.Unmarshal(body, &manifest); err != nil {
				return err
			}
			version, _ := manifest["manifestVersion"].(float64)
			if int(version) != extensionmanifest.ManifestVersionV3 {
				return fmt.Errorf("%s is not an explicit Manifest V3 package", manifestPath)
			}
			files, ok := manifest["packageFiles"].([]any)
			if !ok {
				return fmt.Errorf("Manifest V3 packageFiles must be inline for digest refresh")
			}
			digests := make(map[string]string, len(files))
			for _, raw := range files {
				file, ok := raw.(map[string]any)
				if !ok {
					return fmt.Errorf("invalid packageFiles entry")
				}
				relative, _ := file["path"].(string)
				digest, err := digestPackageRelativeFile(abs, relative)
				if err != nil {
					return err
				}
				digests[relative] = digest
				file["digest"] = digest
			}
			syncInlineDeclarationDigests(manifest, digests)
			paths := make([]string, 0, len(digests))
			for relative := range digests {
				paths = append(paths, relative)
			}
			sort.Strings(paths)
			for _, relative := range paths {
				cmd.Printf("%s\t%s\n", relative, digests[relative])
			}
			if !write {
				return nil
			}
			if err := writeJSON(manifestPath, manifest); err != nil {
				return err
			}
			if _, err := extensionmanifest.LoadPackage(abs); err != nil {
				if restoreErr := os.WriteFile(manifestPath, body, 0o644); restoreErr != nil {
					return fmt.Errorf("refreshed manifest is invalid: %v; restore failed: %w", err, restoreErr)
				}
				return fmt.Errorf("refreshed manifest is invalid: %w", err)
			}
			cmd.Printf("updated %s\n", manifestPath)
			return nil
		},
	}
	cmd.Flags().BoolVar(&write, "write", false, "Write refreshed digests to the root manifest and validate the package")
	return cmd
}

// materializeManifestTemplate 将脚手架占位符（如 __BACKEND_DIGEST__）替换为 64 个 0，
// 仅用于生成可被 digest --write 刷新的合法 JSON；真实摘要必须由 digest 命令写入。
func materializeManifestTemplate(body string) string {
	return manifestDigestPlaceholder.ReplaceAllString(body, strings.Repeat("0", 64))
}

func digestPackageRelativeFile(root string, relative string) (string, error) {
	safe, ok := extensionmanifest.SafeArchivePath(relative)
	if !ok || safe != relative || safe == extensionmanifest.ManifestFileName {
		return "", fmt.Errorf("unsafe package file path %q", relative)
	}
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return "", err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("package root is a symlink: %s", root)
	}
	current := root
	for _, part := range strings.Split(filepath.ToSlash(relative), "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("package file path contains symlink: %s", relative)
		}
	}
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}

func syncInlineDeclarationDigests(manifest map[string]any, digests map[string]string) {
	if backend, ok := manifest["backend"].(map[string]any); ok {
		if entry, _ := backend["entry"].(string); digests[entry] != "" {
			backend["digest"] = digests[entry]
		}
	}
	for family, pathKey := range map[string]string{
		"migrations": "path", "guards": "entry", "templates": "path", "assets": "path", "openapi": "path",
	} {
		items, _ := manifest[family].([]any)
		for _, raw := range items {
			item, _ := raw.(map[string]any)
			path, _ := item[pathKey].(string)
			if digests[path] != "" {
				item["digest"] = digests[path]
			}
		}
	}
}

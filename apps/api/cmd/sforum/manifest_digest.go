package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

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
				return err
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

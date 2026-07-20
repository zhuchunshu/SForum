package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func newExtensionPackageCommand() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "package [path]",
		Short: "Build a zip package with digests and SBOM stub for an extension root",
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
			// Validate package first.
			if _, err := extensionmanifest.LoadPackage(abs); err != nil {
				return fmt.Errorf("package validation failed: %w", err)
			}
			if output == "" {
				base := filepath.Base(abs)
				output = filepath.Join(abs, base+".sforum.zip")
			}
			outAbs, err := filepath.Abs(output)
			if err != nil {
				return err
			}
			result, err := buildExtensionPackage(abs, outAbs)
			if err != nil {
				return err
			}
			cmd.Printf("package\t%s\n", result.ZipPath)
			cmd.Printf("digest\t%s\n", result.PackageDigest)
			cmd.Printf("sbom\t%s\n", result.SBOMPath)
			cmd.Printf("files\t%d\n", result.FileCount)
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "output zip path (default: <root>/<name>.sforum.zip)")
	return cmd
}

type packageBuildResult struct {
	ZipPath       string
	PackageDigest string
	SBOMPath      string
	FileCount     int
}

func buildExtensionPackage(root, zipPath string) (packageBuildResult, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || strings.HasSuffix(name, ".sforum.zip") {
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		// Skip existing zip outputs inside root.
		if strings.HasSuffix(path, ".sforum.zip") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return packageBuildResult{}, err
	}
	sort.Strings(files)

	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		return packageBuildResult{}, err
	}
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return packageBuildResult{}, err
	}
	defer zipFile.Close()

	hasher := sha256.New()
	zipWriter := zip.NewWriter(io.MultiWriter(zipFile, hasher))
	for _, rel := range files {
		abs := filepath.Join(root, rel)
		info, err := os.Stat(abs)
		if err != nil {
			_ = zipWriter.Close()
			return packageBuildResult{}, err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			_ = zipWriter.Close()
			return packageBuildResult{}, err
		}
		header.Name = filepath.ToSlash(rel)
		header.Method = zip.Deflate
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			_ = zipWriter.Close()
			return packageBuildResult{}, err
		}
		f, err := os.Open(abs)
		if err != nil {
			_ = zipWriter.Close()
			return packageBuildResult{}, err
		}
		_, copyErr := io.Copy(writer, f)
		_ = f.Close()
		if copyErr != nil {
			_ = zipWriter.Close()
			return packageBuildResult{}, copyErr
		}
	}
	if err := zipWriter.Close(); err != nil {
		return packageBuildResult{}, err
	}
	digest := hex.EncodeToString(hasher.Sum(nil))

	// SBOM stub (CycloneDX-like minimal JSON) for marketplace provenance display.
	sbomPath := zipPath + ".sbom.json"
	sbom := map[string]any{
		"bomFormat":    "CycloneDX",
		"specVersion":  "1.5",
		"version":      1,
		"serialNumber": "urn:sforum:package:" + digest,
		"metadata": map[string]any{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"component": map[string]any{
				"type":   "application",
				"name":   filepath.Base(root),
				"version": "package",
				"hashes": []map[string]string{{"alg": "SHA-256", "content": digest}},
			},
		},
		"components": []any{},
	}
	raw, err := json.MarshalIndent(sbom, "", "  ")
	if err != nil {
		return packageBuildResult{}, err
	}
	if err := os.WriteFile(sbomPath, append(raw, '\n'), 0o644); err != nil {
		return packageBuildResult{}, err
	}
	return packageBuildResult{
		ZipPath: zipPath, PackageDigest: digest, SBOMPath: sbomPath, FileCount: len(files),
	}, nil
}

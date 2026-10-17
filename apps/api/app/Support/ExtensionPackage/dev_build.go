package extensionpackage

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

// DevBuild 是「dev 自动激活」构建阶段的公共实现（host dev watcher 与
// `sforum extension build/dev` CLI 共用）：
//   - BuildBackend：backend/ 有 Go 源码时执行 go build -trimpath -o backend/plugin ./；
//   - WriteDigest：等价于 `sforum extension digest --write`（含 assets[].integrity
//     的 SRI 重算与 inline 声明摘要同步）。
//
// 放在 Support 层是因为 host 进程不能 import cmd/sforum 的 main 包。
// 环境门（仅 development 使用）由调用方（bootstrap dev 装配）负责。

// BuildBackend 构建包的后端二进制。backend 目录不存在或没有 Go 文件时是无害 no-op。
func BuildBackend(packageRoot string) error {
	backendDir := filepath.Join(packageRoot, "backend")
	entries, err := os.ReadDir(backendDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read backend dir: %w", err)
	}
	hasGo := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			hasGo = true
			break
		}
	}
	if !hasGo {
		return nil
	}
	binary := filepath.Join(backendDir, "plugin")
	cmd := exec.Command("go", "build", "-trimpath", "-o", binary, "./")
	cmd.Dir = backendDir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build backend: %v: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// WriteDigest 刷新包清单的 exact 摘要（等价 `extension digest --write`），
// 并对已声明 integrity 的资产重算 SRI。返回清单刷新后的根摘要。
func WriteDigest(packageRoot string) (string, error) {
	manifestPath := filepath.Join(packageRoot, extensionmanifest.ManifestFileName)
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("read manifest: %w", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(body, &manifest); err != nil {
		return "", err
	}
	version, _ := manifest["manifestVersion"].(float64)
	if int(version) != extensionmanifest.ManifestVersionV3 {
		return "", fmt.Errorf("%s is not an explicit Manifest V3 package", manifestPath)
	}
	if err := refreshAssetIntegrity(packageRoot, manifest); err != nil {
		return "", err
	}
	rawFiles, present := manifest["packageFiles"]
	files, _ := rawFiles.([]any)
	if !present || rawFiles == nil {
		// 无 packageFiles 的合法形态：没有可刷新的摘要，仍校验一遍包。
		if _, err := extensionmanifest.LoadPackage(packageRoot); err != nil {
			return "", fmt.Errorf("package is invalid: %w", err)
		}
		return extensionpackageRootDigest(packageRoot)
	}
	digests := make(map[string]string, len(files))
	for _, raw := range files {
		file, ok := raw.(map[string]any)
		if !ok {
			return "", fmt.Errorf("invalid packageFiles entry")
		}
		relative, _ := file["path"].(string)
		digest, err := digestPackageRelativeFile(packageRoot, relative)
		if err != nil {
			return "", err
		}
		digests[relative] = digest
		file["digest"] = digest
	}
	syncInlineDeclarationDigests(manifest, digests)
	if err := writeJSONFile(manifestPath, manifest); err != nil {
		return "", err
	}
	if _, err := extensionmanifest.LoadPackage(packageRoot); err != nil {
		_ = os.WriteFile(manifestPath, body, 0o644)
		return "", fmt.Errorf("refreshed manifest is invalid: %w", err)
	}
	return extensionpackageRootDigest(packageRoot)
}

func extensionpackageRootDigest(packageRoot string) (string, error) {
	tree, err := DigestTree(packageRoot)
	if err != nil {
		return "", err
	}
	return tree, nil
}

// refreshAssetIntegrity 重算清单里已声明 integrity 的资产 SRI（与 CLI 行为一致）。
func refreshAssetIntegrity(root string, manifest map[string]any) error {
	assets, _ := manifest["assets"].([]any)
	for _, raw := range assets {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		current, _ := item["integrity"].(string)
		if current == "" {
			continue
		}
		relative, _ := item["path"].(string)
		if relative == "" {
			continue
		}
		digest, err := digestPackageRelativeFile(root, relative)
		if err != nil {
			return fmt.Errorf("recompute integrity for %s: %w", relative, err)
		}
		rawDigest, err := hex.DecodeString(digest)
		if err != nil {
			return fmt.Errorf("recompute integrity for %s: %w", relative, err)
		}
		integrity := "sha256-" + base64.StdEncoding.EncodeToString(rawDigest)
		if item["integrity"] != integrity {
			item["integrity"] = integrity
		}
	}
	return nil
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
	items, _ := manifest["editor"].([]any)
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		path, _ := item["l2Module"].(string)
		if digests[path] != "" {
			item["l2Digest"] = digests[path]
		}
	}
}

func writeJSONFile(path string, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(path, body, 0o644)
}

// BuildAndDigestDevPackage 是 dev watcher 的单包构建入口：
// backend 构建（无后端源码时 no-op）→ digest 写回（含 SRI 重算）→ 返回根摘要。
func BuildAndDigestDevPackage(packageRoot string) (string, error) {
	if err := BuildBackend(packageRoot); err != nil {
		return "", err
	}
	return WriteDigest(packageRoot)
}

package extensions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxPrebuiltAdminModuleBytes = 2 * 1024 * 1024
	maxPrebuiltAdminCSSBytes    = 1 * 1024 * 1024
)

type adminFrontendDigestContract struct {
	Settings    *SettingsComponent `json:"settings"`
	EntryDigest string             `json:"entryDigest"`
	CSSDigest   string             `json:"cssDigest,omitempty"`
}

// ComputeAdminFrontendDigest 只覆盖作者预构建的设置组件及其公开契约。
func ComputeAdminFrontendDigest(manifest Manifest, packageRoot string) (string, error) {
	contract := adminFrontendDigestContract{}
	if component := manifest.SettingsDocument.UI.Component; component != nil && component.Entry != "" {
		entryDigest, err := digestAdminAsset(packageRoot, component.Entry)
		if err != nil {
			return "", err
		}
		copy := *component
		contract.Settings = &copy
		contract.EntryDigest = entryDigest
		if component.CSS != "" {
			contract.CSSDigest, err = digestAdminAsset(packageRoot, component.CSS)
			if err != nil {
				return "", err
			}
		}
	}
	if contract.EntryDigest == "" {
		return "", nil
	}
	body, err := json.Marshal(contract)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}

func digestAdminAsset(packageRoot, relative string) (string, error) {
	target, info, err := resolveAdminAsset(packageRoot, relative)
	if err != nil {
		return "", err
	}
	limit := int64(maxPrebuiltAdminModuleBytes)
	if filepath.Ext(target) == ".css" {
		limit = maxPrebuiltAdminCSSBytes
	}
	if info.Size() > limit {
		return "", fmt.Errorf("admin asset %s exceeds %d bytes", relative, limit)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("read admin asset %s: %w", relative, err)
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}

func resolveAdminAsset(packageRoot, relative string) (string, os.FileInfo, error) {
	clean := filepath.Clean(filepath.FromSlash(relative))
	root, err := filepath.Abs(packageRoot)
	if err != nil {
		return "", nil, err
	}
	target, err := filepath.Abs(filepath.Join(root, clean))
	if err != nil || (target != root && !strings.HasPrefix(target, root+string(filepath.Separator))) {
		return "", nil, fmt.Errorf("unsafe admin asset %s", relative)
	}
	info, err := os.Lstat(target)
	if err != nil {
		return "", nil, fmt.Errorf("stat admin asset %s: %w", relative, err)
	}
	if !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("admin asset %s must be a regular file", relative)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", nil, err
	}
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil || (resolvedTarget != resolvedRoot && !strings.HasPrefix(resolvedTarget, resolvedRoot+string(filepath.Separator))) {
		return "", nil, fmt.Errorf("unsafe admin asset %s", relative)
	}
	return target, info, nil
}

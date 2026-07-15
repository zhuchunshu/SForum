package extensions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

type adminFrontendMaterial struct {
	Digest string
	Entry  []byte
	CSS    []byte
}

// ComputeAdminFrontendDigest 只覆盖作者预构建的设置组件及其公开契约。
func ComputeAdminFrontendDigest(manifest Manifest, packageRoot string) (string, error) {
	material, err := readAdminFrontendMaterial(manifest, packageRoot)
	return material.Digest, err
}

// readAdminFrontendMaterial 从同一批稳定、根目录限域的文件句柄构造授权摘要和
// 实际返回字节。调用方不得在验证 Digest 后二次读盘。
func readAdminFrontendMaterial(manifest Manifest, packageRoot string) (adminFrontendMaterial, error) {
	contract := adminFrontendDigestContract{}
	material := adminFrontendMaterial{}
	if component := manifest.SettingsDocument.UI.Component; component != nil && component.Entry != "" {
		entry, err := readAdminAsset(packageRoot, component.Entry, maxPrebuiltAdminModuleBytes)
		if err != nil {
			return adminFrontendMaterial{}, err
		}
		copy := *component
		contract.Settings = &copy
		material.Entry = entry
		contract.EntryDigest = digestAdminAssetBytes(entry)
		if component.CSS != "" {
			material.CSS, err = readAdminAsset(packageRoot, component.CSS, maxPrebuiltAdminCSSBytes)
			if err != nil {
				return adminFrontendMaterial{}, err
			}
			contract.CSSDigest = digestAdminAssetBytes(material.CSS)
		}
	}
	if contract.EntryDigest == "" {
		return material, nil
	}
	body, err := json.Marshal(contract)
	if err != nil {
		return adminFrontendMaterial{}, err
	}
	digest := sha256.Sum256(body)
	material.Digest = hex.EncodeToString(digest[:])
	return material, nil
}

func readAdminAsset(packageRoot, relative string, limit int) ([]byte, error) {
	body, err := readStableExtensionFile(
		Extension{PackagePath: packageRoot}, relative, int64(limit), false,
	)
	if err != nil {
		return nil, fmt.Errorf("read admin asset %s: %w", relative, err)
	}
	return body, nil
}

func digestAdminAssetBytes(body []byte) string {
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

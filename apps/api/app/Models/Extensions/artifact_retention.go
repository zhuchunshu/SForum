package extensions

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

// ErrArtifactUnusable 表示一份冻结制品当前无法被装载。它与「行在、盘缺」是
// 同一族结论：调用方应先放弃，而不是按退避无限重试。
var ErrArtifactUnusable = errors.New("extensions: artifact is not loadable")

// artifactState 判定一个制品路径在磁盘上是否可用。它是制品保留（GC）与
// 缺失制品清理共用的唯一判据来源：任何"行在、盘缺"的结论都必须经过它。
func artifactState(packagePath string) string {
	if strings.TrimSpace(packagePath) == "" {
		return ArtifactMissing
	}
	if _, err := os.Stat(packagePath); err != nil {
		return ArtifactMissing
	}
	return ArtifactAvailable
}

// stagedArtifactUsable 判定一份冻结的候选制品当前是否仍可装载。
//
// dev 自动机在自动恢复失败升级之前必须过这道闸：不可装载的制品每一轮重试都只会
// 产出同样的终态失败（现场曾把同一 operation 推到 attempt=98），此时唯一有意义的
// 动作是把原因写进台账后放弃，等作者产出新摘要。
//
// 判据与运行时装载同构：目录存在（artifactState）、manifest 可解析且身份一致、
// 整树摘要等于冻结摘要（权限位也是摘要输入）、声明的 backend entry 是存在的普通文件。
func stagedArtifactUsable(staged Extension) error {
	if strings.TrimSpace(staged.PackagePath) == "" {
		return fmt.Errorf("%w: %s has no package path", ErrArtifactUnusable, staged.ID)
	}
	if artifactState(staged.PackagePath) != ArtifactAvailable {
		return fmt.Errorf("%w: %s artifact directory %s is missing",
			ErrArtifactUnusable, staged.ID, staged.PackagePath)
	}
	manifest, err := extensionmanifest.LoadPackage(staged.PackagePath)
	if err != nil {
		return fmt.Errorf("%w: %s manifest is unreadable: %v", ErrArtifactUnusable, staged.ID, err)
	}
	if manifest.ID != staged.ID || manifest.Type != staged.Type {
		return fmt.Errorf("%w: %s manifest identity %q/%q does not match the frozen artifact",
			ErrArtifactUnusable, staged.ID, manifest.ID, manifest.Type)
	}
	digest, err := extensionpackage.DigestTree(staged.PackagePath)
	if err != nil {
		return fmt.Errorf("%w: %s content cannot be digested: %v", ErrArtifactUnusable, staged.ID, err)
	}
	if strings.TrimSpace(staged.PackageDigest) != "" && !strings.EqualFold(digest, staged.PackageDigest) {
		return fmt.Errorf("%w: %s content no longer matches the frozen digest", ErrArtifactUnusable, staged.ID)
	}
	if entry := strings.TrimSpace(manifest.Backend.Entry); entry != "" {
		target, ok := installedFilePath(staged, entry)
		if !ok {
			return fmt.Errorf("%w: %s backend entry %s is invalid", ErrArtifactUnusable, staged.ID, entry)
		}
		info, statErr := os.Stat(target)
		if statErr != nil || info.IsDir() {
			return fmt.Errorf("%w: %s backend entry %s is not available", ErrArtifactUnusable, staged.ID, entry)
		}
	}
	return nil
}

func decorateArtifactState(extension Extension) Extension {
	extension.ArtifactState = artifactState(extension.PackagePath)
	if extension.StagedVersion != nil {
		extension.StagedVersion.ArtifactState = artifactState(extension.StagedVersion.PackagePath)
	}
	return extension
}

func requireArtifactAvailable(extension Extension) error {
	if artifactState(extension.PackagePath) == ArtifactMissing {
		return fmt.Errorf("%w: %s", ErrArtifactMissing, extension.ID)
	}
	return nil
}

// normalizeMissingArtifactDataMode 归一化缺失制品清理的数据处置模式。
func normalizeMissingArtifactDataMode(value string) (string, error) {
	switch value {
	case "", MissingArtifactDataPreserve:
		return MissingArtifactDataPreserve, nil
	case MissingArtifactDataDiscardSettings:
		return MissingArtifactDataDiscardSettings, nil
	default:
		return "", fmt.Errorf("%w: invalid data mode", ErrMissingArtifactCleanupInvalid)
	}
}

// normalizedMissingArtifactIDs 去重并排序扩展 id，保证清理与 GC 的批量输入
// 顺序稳定（同序输入产生同序计划与同序审计）。
func normalizedMissingArtifactIDs(values []string) []string {
	unique := make(map[string]struct{}, len(values))
	for _, value := range values {
		if id := normalizeID(value); id != "" {
			unique[id] = struct{}{}
		}
	}
	ids := make([]string, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

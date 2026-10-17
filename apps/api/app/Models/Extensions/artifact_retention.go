package extensions

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

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

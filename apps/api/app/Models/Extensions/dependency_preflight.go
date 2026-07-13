package extensions

import (
	"context"
	"fmt"
	"sort"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

// ResolveInstalledDependencyGraph 返回当前已启用扩展的确定性激活顺序。
func (s *Service) ResolveInstalledDependencyGraph(ctx context.Context) (extensionmanifest.PackageGraph, error) {
	items, err := s.store.List(ctx)
	if err != nil {
		return extensionmanifest.PackageGraph{}, fmt.Errorf("list installed extensions for dependency graph: %w", err)
	}
	manifests := make([]extensionmanifest.Manifest, 0, len(items))
	for _, item := range items {
		if item.Status == StatusEnabled {
			manifests = append(manifests, item.Manifest)
		}
	}
	sort.Slice(manifests, func(i, j int) bool { return manifests[i].ID < manifests[j].ID })
	graph, err := extensionmanifest.ResolvePackageGraph(manifests)
	if err != nil {
		return extensionmanifest.PackageGraph{}, fmt.Errorf("resolve installed extension dependency graph: %w", err)
	}
	return graph, nil
}

// PreflightActivationDependencies 校验启用目标替换同 ID 记录后的完整激活集合。
func (s *Service) PreflightActivationDependencies(ctx context.Context, id string) (extensionmanifest.PackageGraph, error) {
	candidate, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return extensionmanifest.PackageGraph{}, err
	}
	return s.preflightActivationDependencies(ctx, candidate)
}

func (s *Service) preflightActivationDependencies(ctx context.Context, candidate Extension) (extensionmanifest.PackageGraph, error) {
	items, err := s.store.List(ctx)
	if err != nil {
		return extensionmanifest.PackageGraph{}, fmt.Errorf("%w: list activation dependencies for %s: %w", ErrPreflightFailed, candidate.ID, err)
	}

	selected := make(map[string]extensionmanifest.Manifest, len(items)+1)
	for _, item := range items {
		if item.Status == StatusEnabled && item.ID != candidate.ID {
			selected[item.ID] = item.Manifest
		}
	}
	// 候选始终覆盖同 ID 已安装记录，避免升级后用旧版本做依赖判定。
	selected[candidate.ID] = candidate.Manifest

	ids := make([]string, 0, len(selected))
	for id := range selected {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	manifests := make([]extensionmanifest.Manifest, 0, len(ids))
	for _, id := range ids {
		manifests = append(manifests, selected[id])
	}

	graph, err := extensionmanifest.ResolvePackageGraph(manifests)
	if err != nil {
		return extensionmanifest.PackageGraph{}, fmt.Errorf("%w: resolve activation dependency graph for %s: %w", ErrPreflightFailed, candidate.ID, err)
	}
	return graph, nil
}

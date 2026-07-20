package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	contentregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/ContentRegistry"
)

func (b *PostgresLifecycleBoundaryRegistries) ContentRegistry() *contentregistry.Registry {
	if b == nil {
		return nil
	}
	return b.content
}

// buildLifecycleContentPublication freezes Manifest.content into one exact
// Content Registry publication. Impact digest is verified by freeze callers for
// trust authority; the Content artifact itself does not carry ImpactDigest.
func buildLifecycleContentPublication(
	extension extensions.Extension,
	binding extensions.LifecycleRuntimeBinding,
) (*contentregistry.Publication, error) {
	if len(extension.Manifest.Content) == 0 {
		return nil, nil
	}
	if validateExactContentPublicationArtifact(extension, binding) != nil {
		return nil, ErrLifecycleRegistryPublicationInvalid
	}
	publication := contentregistry.Publication{Artifact: contentregistry.Artifact{
		ExtensionID: extension.ID, ExtensionVersion: extension.Version,
		PackageDigest: extension.PackageDigest, VersionID: extension.ActiveVersionID,
		RuntimeInstanceID: binding.RuntimeInstanceID,
	}}
	publication.Content = make([]contentregistry.Declaration, 0, len(extension.Manifest.Content))
	for _, declaration := range extension.Manifest.Content {
		publication.Content = append(publication.Content, contentregistry.Declaration{
			ID: declaration.ID, ContractVersion: declaration.ContractVersion,
			Kind: declaration.Kind, Handler: declaration.Handler,
			Schema: declaration.Schema, Renderer: declaration.Renderer,
			Migration: declaration.Migration,
		})
	}
	// 用探针 Registry 规范化顺序与边界，再冻结进 durable lifecycle digest。
	probe := contentregistry.New()
	if _, err := probe.ReplaceAllIfRevision(0, []contentregistry.Publication{publication}, false); err != nil {
		return nil, fmt.Errorf("build content registry publication: %w", err)
	}
	frozen, found := probe.SnapshotPublication(extension.ID)
	if !found {
		return nil, ErrLifecycleRegistryPublicationInvalid
	}
	return &frozen, nil
}

func validateExactContentPublicationArtifact(
	extension extensions.Extension,
	binding extensions.LifecycleRuntimeBinding,
) error {
	// Content 声明可带 handler 或纯 renderer；带 handler 时要求 exact runtime。
	requiresRuntime := false
	for _, declaration := range extension.Manifest.Content {
		if strings.TrimSpace(declaration.Handler) != "" {
			requiresRuntime = true
			break
		}
	}
	if extension.ID == "" || extension.Version == "" || extension.ActiveVersionID <= 0 ||
		extension.ID != strings.TrimSpace(extension.ID) || extension.Version != strings.TrimSpace(extension.Version) ||
		!validLifecycleCleanupDigest(extension.PackageDigest) ||
		extension.Type != extensions.TypePlugin || extension.Manifest.ID != extension.ID ||
		extension.Manifest.Version != extension.Version || extension.Manifest.Type != extensions.TypePlugin ||
		validateExactCoordinatorBinding("content registry", binding, extension, requiresRuntime) != nil {
		return ErrLifecycleRegistryPublicationInvalid
	}
	if requiresRuntime {
		if extension.Manifest.Backend.ProtocolVersion != 2 || strings.TrimSpace(extension.Manifest.Backend.Entry) == "" {
			return ErrLifecycleRegistryPublicationInvalid
		}
	}
	return nil
}

func (b *PostgresLifecycleBoundaryRegistries) freezeContentMaterials(
	ctx context.Context,
	request LifecycleBoundaryRequest,
	source, target *lifecycleRegistryMaterial,
) error {
	hasSource := source != nil && len(source.extension.Manifest.Content) > 0
	hasTarget := target != nil && len(target.extension.Manifest.Content) > 0
	if !hasSource && !hasTarget {
		return nil
	}
	if b == nil || b.content == nil || b.assetAuthority == nil || ctx == nil {
		return ErrLifecycleRegistryPublicationUnavailable
	}
	// Impact 校验与 SEO/Navigation 一致：source 用 restore，target 用 operation。
	// 不写入 Content Artifact（该 Registry 无 ImpactDigest 字段）。
	if hasSource {
		if _, err := b.assetAuthority.RestoreImpactDigest(ctx, source.extension); err != nil {
			return fmt.Errorf("freeze source content authority: %w", err)
		}
		if err := b.freezeContentMaterial(source); err != nil {
			return err
		}
	}
	if hasTarget {
		if _, err := b.assetAuthority.OperationImpactDigest(ctx, request.OperationID, target.extension); err != nil {
			return fmt.Errorf("freeze target content authority: %w", err)
		}
		if err := b.freezeContentMaterial(target); err != nil {
			return err
		}
	}
	return nil
}

func (b *PostgresLifecycleBoundaryRegistries) freezeContentMaterial(
	material *lifecycleRegistryMaterial,
) error {
	publication, err := buildLifecycleContentPublication(material.extension, material.binding)
	if err != nil {
		return err
	}
	material.contentPublication = publication
	return refreshLifecycleRegistryMaterialDigest(material)
}

func (b *PostgresLifecycleBoundaryRegistries) restoreContentPublications(
	ctx context.Context,
	items []extensions.Extension,
	safeMode bool,
) error {
	if b == nil || ctx == nil {
		return ErrLifecycleRegistryPublicationUnavailable
	}
	// Content Registry 未接线时跳过：兼容尚未注入 P10 Content 的旧边界测试。
	if b.content == nil {
		return nil
	}
	if b.manager == nil {
		return ErrLifecycleRegistryPublicationUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	snapshot := b.content.Snapshot()
	publications := coreLifecycleContentPublications(snapshot.Publications)
	if safeMode {
		if _, err := b.content.ReplaceAllIfRevision(snapshot.Revision, publications, true); err != nil {
			return wrapLifecycleContentError("restore content registry safe mode", err)
		}
		return nil
	}
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return err
		}
		if item.Type != extensions.TypePlugin || item.Status != extensions.StatusEnabled || len(item.Manifest.Content) == 0 {
			continue
		}
		if b.assetAuthority == nil {
			return ErrLifecycleRegistryPublicationUnavailable
		}
		if _, err := b.assetAuthority.RestoreImpactDigest(ctx, item); errors.Is(err, extensions.ErrTrustGrantNotFound) ||
			errors.Is(err, extensions.ErrLifecycleAuthorityNotFound) {
			// 未确认/已撤销可执行包在启动路径保持关闭。
			continue
		} else if err != nil {
			return fmt.Errorf("restore content authority for %s: %w", item.ID, err)
		}
		runtime, err := b.manager.ActiveRuntimeInstance(item.ID)
		if err != nil {
			// 失败或 boot-loop 抑制的进程不得恢复声明。
			// 纯 renderer 内容仍要求进程可用时走同一路径（与 SEO 对齐）。
			continue
		}
		if !runtimeInstanceMatchesExtension(runtime, item) || !b.manager.RuntimeInstanceAvailable(runtime.Identity) {
			return fmt.Errorf("%w: startup content runtime for %s is not exact and available",
				ErrLifecycleRegistryPublicationConflict, item.ID)
		}
		publication, err := buildLifecycleContentPublication(item, extensions.LifecycleRuntimeBinding{
			ExtensionID: item.ID, ExtensionVersion: item.Version, PackageDigest: item.PackageDigest,
			VersionID: item.ActiveVersionID, RuntimeInstanceID: runtime.Identity.InstanceID,
		})
		if err != nil {
			return fmt.Errorf("restore content registry for %s: %w", item.ID, err)
		}
		if publication != nil {
			publications = append(publications, *publication)
		}
	}
	if _, err := b.content.ReplaceAllIfRevision(snapshot.Revision, publications, false); err != nil {
		return wrapLifecycleContentError("restore content registry publication", err)
	}
	return nil
}

func (b *PostgresLifecycleBoundaryRegistries) RestoreContentPublications(
	ctx context.Context,
	items []extensions.Extension,
	safeMode bool,
) error {
	return b.restoreContentPublications(ctx, items, safeMode)
}

func coreLifecycleContentPublications(input []contentregistry.Publication) []contentregistry.Publication {
	result := make([]contentregistry.Publication, 0, len(input))
	for _, publication := range input {
		if publication.Artifact.Core {
			result = append(result, publication)
		}
	}
	return result
}

func (b *PostgresLifecycleBoundaryRegistries) validateContentTransition(
	source, target *lifecycleRegistryMaterial,
) error {
	hasContent := (source != nil && source.contentPublication != nil) ||
		(target != nil && target.contentPublication != nil)
	if !hasContent {
		return nil
	}
	if b == nil || b.content == nil {
		return ErrLifecycleRegistryPublicationUnavailable
	}
	extensionID := lifecycleComponentExtensionID(source, target)
	if extensionID == "" {
		return ErrLifecycleRegistryPublicationInvalid
	}
	snapshot := b.content.Snapshot()
	for _, desired := range []*lifecycleRegistryMaterial{source, target} {
		var publication *contentregistry.Publication
		if desired != nil {
			publication = desired.contentPublication
		}
		graph, err := lifecycleContentGraph(snapshot, extensionID, publication, source, target)
		if err != nil {
			return err
		}
		if _, err := contentregistry.New().ReplaceAllIfRevision(0, graph, snapshot.SafeMode); err != nil {
			return wrapLifecycleContentError("validate content registry publication", err)
		}
	}
	return nil
}

func lifecycleContentGraph(
	snapshot contentregistry.Snapshot,
	extensionID string,
	desired *contentregistry.Publication,
	allowedMaterials ...*lifecycleRegistryMaterial,
) ([]contentregistry.Publication, error) {
	allowed := make(map[contentregistry.Artifact]contentregistry.Publication, len(allowedMaterials))
	for _, material := range allowedMaterials {
		if material == nil || material.contentPublication == nil {
			continue
		}
		artifact := material.contentPublication.Artifact
		if existing, found := allowed[artifact]; found && !reflect.DeepEqual(existing, *material.contentPublication) {
			return nil, ErrLifecycleRegistryPublicationConflict
		}
		allowed[artifact] = *material.contentPublication
	}
	publications := make([]contentregistry.Publication, 0, len(snapshot.Publications)+1)
	for _, publication := range snapshot.Publications {
		if publication.Artifact.ExtensionID != extensionID {
			publications = append(publications, publication)
			continue
		}
		frozen, ok := allowed[publication.Artifact]
		if !ok || !reflect.DeepEqual(frozen, publication) {
			return nil, ErrLifecycleRegistryPublicationConflict
		}
	}
	if desired != nil {
		if snapshot.SafeMode {
			return nil, ErrLifecycleRegistryPublicationConflict
		}
		publications = append(publications, *desired)
	}
	return publications, nil
}

func (b *PostgresLifecycleBoundaryRegistries) reconcileContent(
	ctx context.Context,
	extensionID string,
	source, target, desired *lifecycleRegistryMaterial,
) error {
	hasContent := (source != nil && source.contentPublication != nil) ||
		(target != nil && target.contentPublication != nil) ||
		(desired != nil && desired.contentPublication != nil)
	if !hasContent {
		return nil
	}
	if b == nil || b.content == nil || ctx == nil {
		return ErrLifecycleRegistryPublicationUnavailable
	}
	var desiredPublication *contentregistry.Publication
	if desired != nil {
		desiredPublication = desired.contentPublication
	}
	for attempts := 0; attempts < 16; attempts++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		snapshot := b.content.Snapshot()
		graph, err := lifecycleContentGraph(snapshot, extensionID, desiredPublication, source, target)
		if err != nil {
			return err
		}
		if _, err := b.content.ReplaceAllIfRevision(snapshot.Revision, graph, snapshot.SafeMode); err == nil {
			return nil
		} else if !errors.Is(err, contentregistry.ErrRevisionConflict) {
			return wrapLifecycleContentError("publish content registry graph", err)
		}
	}
	return contentregistry.ErrRevisionConflict
}

func wrapLifecycleContentError(action string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, contentregistry.ErrArtifactConflict) || errors.Is(err, contentregistry.ErrSafeMode) {
		return fmt.Errorf("%w: %s: %v", ErrLifecycleRegistryPublicationConflict, action, err)
	}
	return fmt.Errorf("%s: %w", action, err)
}

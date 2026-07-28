package extensionsruntime

import (
	"errors"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	pages "github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
)

func buildExactPluginPageRuntime(
	extension extensions.Extension,
	binding extensions.LifecycleRuntimeBinding,
	contributions []pages.PageContribution,
	siteName string,
	locales []string,
) (*pages.ThemeRuntimeSnapshot, error) {
	templates := make([]pages.RuntimeTemplateDeclaration, len(extension.Manifest.Templates))
	for index, item := range extension.Manifest.Templates {
		templates[index] = pages.RuntimeTemplateDeclaration{
			ID: item.ID, ContractVersion: item.ContractVersion, Action: item.Action,
			TargetID: item.TargetID, Path: item.Path, Digest: item.Digest,
			ViewModelSchema: item.ViewModelSchema, ThemeOverrideKey: item.ThemeOverrideKey,
		}
	}
	schemas := make([]pages.RuntimeDataSchemaDeclaration, 0, len(extension.Manifest.PackageFiles))
	for _, file := range extension.Manifest.PackageFiles {
		if file.Kind != "schema" {
			continue
		}
		schemas = append(schemas, pages.RuntimeDataSchemaDeclaration{
			ID: file.ID, Version: file.Version, Path: file.Path, Digest: file.Digest,
		})
	}
	snapshot, err := pages.BuildThemeRuntimeSnapshot(pages.ThemeRuntimeBuildInput{
		Artifact: pages.RuntimeArtifact{
			ExtensionID: extension.ID, ExtensionVersion: extension.Version,
			PackageDigest: extension.PackageDigest, RuntimeInstanceID: binding.RuntimeInstanceID,
		},
		PackageRoot: extensions.PackageContentRoot(extension), Contributions: contributions,
		Templates: templates, DataSchemas: schemas, PackageKind: pages.RuntimeTemplatePlugin,
		RequireDeclaredTemplates: extensionmanifest.EffectiveManifestVersion(extension.Manifest) >= extensionmanifest.ManifestVersionV3,
		SiteName:                 siteName, Locales: append([]string(nil), locales...),
	})
	if errors.Is(err, pages.ErrThemeRuntimeMissing) {
		return nil, nil
	}
	return snapshot, err
}

func (b *PostgresLifecycleBoundaryRegistries) removeSupersededPageRuntimes(
	desired *lifecycleRegistryMaterial,
	materials ...*lifecycleRegistryMaterial,
) error {
	if b == nil || b.themeRuntime == nil {
		return nil
	}
	var keep pages.RuntimeArtifact
	if desired != nil {
		keep = pages.RuntimeArtifact{
			ExtensionID: desired.extension.ID, ExtensionVersion: desired.extension.Version,
			PackageDigest: desired.extension.PackageDigest, RuntimeInstanceID: desired.binding.RuntimeInstanceID,
		}
	}
	for _, material := range materials {
		if material == nil {
			continue
		}
		artifact := pages.RuntimeArtifact{
			ExtensionID: material.extension.ID, ExtensionVersion: material.extension.Version,
			PackageDigest: material.extension.PackageDigest, RuntimeInstanceID: material.binding.RuntimeInstanceID,
		}
		if artifact == keep {
			continue
		}
		if _, err := b.themeRuntime.RemoveExact(artifact); err != nil && !errors.Is(err, pages.ErrThemeRuntimeConflict) {
			return err
		}
	}
	return nil
}

func pageArtifactAllowed(artifact pages.RuntimeArtifact, materials ...*lifecycleRegistryMaterial) bool {
	for _, material := range materials {
		if material == nil {
			continue
		}
		if artifact.ExtensionID == material.extension.ID &&
			artifact.ExtensionVersion == material.extension.Version &&
			artifact.PackageDigest == material.extension.PackageDigest &&
			artifact.RuntimeInstanceID == material.binding.RuntimeInstanceID {
			return true
		}
	}
	return false
}

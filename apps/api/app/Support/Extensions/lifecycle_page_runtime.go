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

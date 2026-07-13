package extensionmanifest

import "fmt"

const (
	ManifestVersionV1     = 1
	ManifestVersionV2     = 2
	ManifestVersionV3     = 3
	ManifestVersionLatest = ManifestVersionV3
)

// EffectiveManifestVersion keeps pre-versioned packages byte-compatible while
// exposing one explicit contract version to validators and runtime consumers.
func EffectiveManifestVersion(manifest Manifest) int {
	if manifest.ManifestVersion == 0 {
		return ManifestVersionV1
	}
	return manifest.ManifestVersion
}

func ManifestContract(manifest Manifest) string {
	return fmt.Sprintf("sforum.manifest@%d", EffectiveManifestVersion(manifest))
}

func validateManifestVersion(manifest Manifest) error {
	version := EffectiveManifestVersion(manifest)
	if version < ManifestVersionV1 || version > ManifestVersionLatest {
		return ErrInvalidManifest
	}
	return nil
}

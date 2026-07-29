package extensionmanifest

import "fmt"

const (
	ManifestVersionV3     = 3
	ManifestVersionLatest = ManifestVersionV3
)

// EffectiveManifestVersion exposes the explicitly declared manifest contract.
// LoadPackage and Validate reject zero and every version other than V3.
func EffectiveManifestVersion(manifest Manifest) int {
	return manifest.ManifestVersion
}

func ManifestContract(manifest Manifest) string {
	return fmt.Sprintf("sforum.manifest@%d", EffectiveManifestVersion(manifest))
}

func validateManifestVersion(manifest Manifest) error {
	if EffectiveManifestVersion(manifest) != ManifestVersionLatest {
		return ErrInvalidManifest
	}
	return nil
}

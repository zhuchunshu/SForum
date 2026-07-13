package extensionmanifest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// ValidatePackageFiles verifies exact bytes without executing package code.
func ValidatePackageFiles(manifest Manifest, pkg PackageFS) error {
	if EffectiveManifestVersion(manifest) != ManifestVersionV3 {
		return nil
	}
	for _, file := range manifest.PackageFiles {
		body, err := pkg.ReadFile(file.Path)
		if err != nil {
			return fmt.Errorf("%w: read package file %s: %v", ErrInvalidManifest, file.Path, err)
		}
		digest := sha256.Sum256(body)
		if hex.EncodeToString(digest[:]) != file.Digest {
			return fmt.Errorf("%w: package file digest mismatch: %s", ErrInvalidManifest, file.Path)
		}
	}
	return nil
}

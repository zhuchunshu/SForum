package extensions

import (
	"fmt"
	"os"
	"strings"
)

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

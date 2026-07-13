package extensionmanifest

import (
	"path"
	"regexp"
	"strings"
)

const (
	AdminMicroFrontendAPIVersion = 1

	ContributionPointKindDescriptor = "descriptor"
)

var adminComponentIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,80}$`)

// ValidateWithContributionPoints validates a manifest against the host-owned descriptor catalog.
func ValidateWithContributionPoints(manifest Manifest, points []ContributionPointDefinition) error {
	return validateManifest(manifest, points)
}

func normalizeAdminRelativePath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return ""
	}
	return path.Clean(value)
}

func safeAdminRelativePath(value string) bool {
	if value == "" || value == "." || strings.HasPrefix(value, "/") || strings.ContainsRune(value, '\x00') {
		return false
	}
	if len(value) >= 2 && value[1] == ':' && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) {
		return false
	}
	cleaned := path.Clean(value)
	return cleaned != ".." && !strings.HasPrefix(cleaned, "../")
}

package extensionmanifest

import (
	"encoding/json"
	"path"
	"regexp"
	"strings"
)

const (
	AdminFrontendAPIVersion = 1

	ContributionPointKindDescriptor = "descriptor"
	ContributionPointKindComponent  = "component"
)

var adminComponentIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,80}$`)

type ManifestAdminFrontend struct {
	Root       string            `json:"root"`
	APIVersion int               `json:"apiVersion"`
	Components map[string]string `json:"components"`
	Locales    map[string]string `json:"locales"`
}

type AdminComponentContributionPayload struct {
	Component string `json:"component"`
}

// ValidateWithContributionPoints validates a manifest against the catalog installed by the host.
func ValidateWithContributionPoints(manifest Manifest, points []ContributionPointDefinition) error {
	return validateManifest(manifest, points)
}

func normalizeAdminFrontend(admin *ManifestAdminFrontend) *ManifestAdminFrontend {
	if admin == nil {
		return nil
	}
	normalized := &ManifestAdminFrontend{
		Root:       normalizeAdminRelativePath(admin.Root),
		APIVersion: admin.APIVersion,
		Components: make(map[string]string, len(admin.Components)),
		Locales:    make(map[string]string, len(admin.Locales)),
	}
	for id, modulePath := range admin.Components {
		normalized.Components[id] = normalizeAdminRelativePath(modulePath)
	}
	for locale, localePath := range admin.Locales {
		normalized.Locales[locale] = normalizeAdminRelativePath(localePath)
	}
	return normalized
}

func normalizeAdminRelativePath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return ""
	}
	return path.Clean(value)
}

func validateAdminFrontend(manifest Manifest) error {
	admin := manifest.Frontend.Admin
	if admin == nil {
		return nil
	}
	// 插件与主题均可声明 trusted admin 前端（主题主要用于自定义设置页）。
	if (manifest.Type != TypePlugin && manifest.Type != TypeTheme) || !safeAdminRelativePath(admin.Root) || admin.APIVersion != AdminFrontendAPIVersion {
		return ErrInvalidManifest
	}
	if len(admin.Components) == 0 {
		return ErrInvalidManifest
	}
	for id, modulePath := range admin.Components {
		if !adminComponentIDPattern.MatchString(id) || !safeAdminRelativePath(modulePath) || !supportedAdminModulePath(modulePath) {
			return ErrInvalidManifest
		}
	}
	if len(admin.Locales) < 2 || admin.Locales["zh-CN"] == "" || admin.Locales["en-US"] == "" {
		return ErrInvalidManifest
	}
	for locale, localePath := range admin.Locales {
		if strings.TrimSpace(locale) == "" || !safeAdminRelativePath(localePath) || strings.ToLower(path.Ext(localePath)) != ".json" {
			return ErrInvalidManifest
		}
	}
	return nil
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

func supportedAdminModulePath(value string) bool {
	switch strings.ToLower(path.Ext(value)) {
	case ".vue", ".ts", ".tsx", ".js", ".jsx":
		return true
	default:
		return false
	}
}

func normalizeAdminComponentPayload(raw json.RawMessage) (json.RawMessage, bool) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return raw, false
	}
	componentRaw, ok := payload["component"]
	if !ok {
		return raw, false
	}
	var component string
	if err := json.Unmarshal(componentRaw, &component); err != nil {
		return raw, false
	}
	componentBody, err := json.Marshal(NormalizeID(component))
	if err != nil {
		return raw, false
	}
	payload["component"] = componentBody
	normalized, err := json.Marshal(payload)
	if err != nil {
		return raw, false
	}
	return normalized, true
}

func adminComponentBinding(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", ErrInvalidManifest
	}
	var payload AdminComponentContributionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", ErrInvalidManifest
	}
	component := NormalizeID(payload.Component)
	if !adminComponentIDPattern.MatchString(component) {
		return "", ErrInvalidManifest
	}
	return component, nil
}

func validateAdminComponentReferences(manifest Manifest, references map[string]int) error {
	admin := manifest.Frontend.Admin
	if admin == nil {
		if len(references) != 0 {
			return ErrInvalidManifest
		}
		return nil
	}
	for component := range admin.Components {
		if references[component] != 1 {
			return ErrInvalidManifest
		}
	}
	return nil
}

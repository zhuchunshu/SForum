package extensionmanifest

import (
	"path"
	"regexp"
	"sort"
	"strings"
)

const (
	AdminMicroFrontendAPIVersion = 1

	ContributionPointKindDescriptor = "descriptor"
)

var adminComponentIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,80}$`)

// AdminComponent is one author-prebuilt browser module mounted inside a
// Host-owned admin surface. Settings and ordinary admin pages share this exact
// artifact contract, but expose different bridge capabilities at runtime.
type AdminComponent struct {
	ID         string `json:"id"`
	APIVersion int    `json:"apiVersion"`
	Entry      string `json:"entry,omitempty"`
	CSS        string `json:"css,omitempty"`
}

type AdminComponentBinding struct {
	Surface   string
	PagePath  string
	Component AdminComponent
}

// DeclaredAdminComponents returns every executable admin browser surface in a
// deterministic order for trust impact, digest, and package validation.
func DeclaredAdminComponents(manifest Manifest) []AdminComponentBinding {
	bindings := make([]AdminComponentBinding, 0, len(manifest.Admin.Pages))
	if component := manifest.SettingsDocument.UI.Component; component != nil && component.Entry != "" {
		bindings = append(bindings, AdminComponentBinding{Surface: "settings", Component: *component})
	}
	for _, page := range EffectiveAdminPages(manifest) {
		if page.View != "component" || page.Component == nil || page.Component.Entry == "" {
			continue
		}
		bindings = append(bindings, AdminComponentBinding{
			Surface: "page:" + page.Path, PagePath: page.Path, Component: *page.Component,
		})
	}
	sort.Slice(bindings, func(left, right int) bool {
		if bindings[left].Surface == bindings[right].Surface {
			return bindings[left].Component.ID < bindings[right].Component.ID
		}
		return bindings[left].Surface < bindings[right].Surface
	})
	return bindings
}

func normalizeAdminComponent(component *AdminComponent) {
	if component == nil {
		return
	}
	component.ID = NormalizeID(component.ID)
	component.Entry = normalizeAdminRelativePath(component.Entry)
	component.CSS = normalizeAdminRelativePath(component.CSS)
}

func validAdminComponent(component *AdminComponent) bool {
	if component == nil || !adminComponentIDPattern.MatchString(component.ID) ||
		component.APIVersion != AdminMicroFrontendAPIVersion || component.Entry == "" {
		return false
	}
	if !safeAdminRelativePath(component.Entry) || path.Ext(component.Entry) != ".mjs" ||
		!strings.HasPrefix(component.Entry, "frontend/admin/dist/") {
		return false
	}
	return component.CSS == "" || (safeAdminRelativePath(component.CSS) &&
		path.Ext(component.CSS) == ".css" && strings.HasPrefix(component.CSS, "frontend/admin/dist/"))
}

func normalizeAdminPageSlice(pages []ManifestAdminPage) {
	for index := range pages {
		pages[index].Path = NormalizeRoutePath(pages[index].Path)
		pages[index].Label = strings.TrimSpace(pages[index].Label)
		pages[index].Description = strings.TrimSpace(pages[index].Description)
		pages[index].Icon = strings.TrimSpace(pages[index].Icon)
		pages[index].View = strings.ToLower(strings.TrimSpace(pages[index].View))
		if pages[index].View == "" {
			pages[index].View = "about"
		}
		pages[index].Permission = strings.TrimSpace(pages[index].Permission)
		normalizeAdminComponent(pages[index].Component)
	}
}

func validateAdminDeclaration(manifest Manifest) error {
	pages := EffectiveAdminPages(manifest)
	componentIDs := map[string]struct{}{}
	if component := manifest.SettingsDocument.UI.Component; component != nil && component.Entry != "" {
		componentIDs[component.ID] = struct{}{}
	}
	for _, page := range pages {
		if page.Path == "" || !strings.HasPrefix(page.Path, "/") || strings.Contains(page.Path, "..") || page.Label == "" {
			return ErrInvalidManifest
		}
		if page.View != "" && page.View != "about" && page.View != "settings" && page.View != "component" {
			return ErrInvalidManifest
		}
		if page.View == "component" {
			if !validAdminComponent(page.Component) {
				return ErrInvalidManifest
			}
			if _, exists := componentIDs[page.Component.ID]; exists {
				return ErrInvalidManifest
			}
			componentIDs[page.Component.ID] = struct{}{}
		} else if page.Component != nil {
			return ErrInvalidManifest
		}
		if page.Order < 0 {
			return ErrInvalidManifest
		}
	}
	if manifest.Admin.Entry == "" {
		return nil
	}
	if strings.Contains(manifest.Admin.Entry, "://") || !strings.HasPrefix(manifest.Admin.Entry, "/") || strings.Contains(manifest.Admin.Entry, "..") {
		return ErrInvalidManifest
	}
	if manifest.Admin.Entry == "/about" {
		return nil
	}
	for _, page := range pages {
		if page.Path == manifest.Admin.Entry {
			return nil
		}
	}
	return ErrInvalidManifest
}

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

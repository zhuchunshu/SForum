package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

const (
	WebReleaseTriggerRebuild       = "rebuild"
	WebReleaseTriggerPluginEnable  = "plugin.enable"
	WebReleaseTriggerPluginDisable = "plugin.disable"
	WebReleaseTriggerTrustGrant    = "frontend.trust_grant"
	WebReleaseTriggerTrustRevoke   = "frontend.trust_revoke"
	WebReleaseTriggerRestore       = "frontend.restore_defaults"
	WebReleaseTriggerTheme         = "theme.activate"
	WebReleaseTriggerRollback      = "rollback"
)

var (
	ErrWebReleaseInvalidComposition = errors.New("extensions: invalid web release composition")
	ErrWebReleasePackageChanged     = errors.New("extensions: web release package changed")
)

type WebThemeSnapshot struct {
	ExtensionID   string `json:"extensionId"`
	Version       string `json:"version"`
	LayerPath     string `json:"layerPath"`
	PackageDigest string `json:"packageDigest"`
}

type WebExtensionSnapshot struct {
	ExtensionID     string                 `json:"extensionId"`
	Version         string                 `json:"version"`
	PackageDigest   string                 `json:"packageDigest"`
	FrontendRoot    string                 `json:"frontendRoot"`
	ComponentMap    map[string]string      `json:"componentMap"`
	APIVersion      int                    `json:"apiVersion"`
	Contributions   []ManifestContribution `json:"contributions"`
	LocaleMap       map[string]string      `json:"localeMap"`
	LocaleMapDigest string                 `json:"localeMapDigest"`
	Dependencies    DependencySummary      `json:"dependencies"`
	SortOrder       int                    `json:"sortOrder"`
}

type WebComposition struct {
	Theme      WebThemeSnapshot       `json:"theme"`
	Extensions []WebExtensionSnapshot `json:"extensions"`
	WebSource  string                 `json:"webSource"`
	WebLock    string                 `json:"webLock"`
	SDKVersion int                    `json:"sdkVersion"`
	BunVersion string                 `json:"bunVersion"`
	Contract   int                    `json:"contract"`
}

type WebCompositionHost struct {
	WebSource  string
	WebLock    string
	SDKVersion int
	BunVersion string
	Contract   int
	HostPeers  extensionpackage.HostPeers
}

type PlanWebReleaseInput struct {
	TriggerKind        string
	TriggerExtensionID string
	TargetThemeID      string
	RequestedBy        int64
	ReloadMode         string
}

type PlannedWebRelease struct {
	Composition WebComposition
	Snapshot    json.RawMessage
	Hash        string
	Existing    *WebRelease
}

type WebReleaseExtensionReader interface {
	List(context.Context) ([]Extension, error)
	Get(context.Context, string) (Extension, error)
	ActiveTheme(context.Context) (Extension, error)
}

type WebReleaseGrantReader interface {
	FrontendGrant(context.Context, string, string, string) (FrontendTrustGrant, error)
}

type WebReleasePlanner struct {
	extensions WebReleaseExtensionReader
	grants     WebReleaseGrantReader
	host       WebCompositionHost
}

func NewWebReleasePlanner(extensions WebReleaseExtensionReader, grants WebReleaseGrantReader, host WebCompositionHost) *WebReleasePlanner {
	return &WebReleasePlanner{extensions: extensions, grants: grants, host: host}
}

func (p *WebReleasePlanner) Plan(ctx context.Context, input PlanWebReleaseInput) (PlannedWebRelease, error) {
	if p == nil || p.extensions == nil || p.grants == nil {
		return PlannedWebRelease{}, fmt.Errorf("%w: planner dependencies are missing", ErrWebReleaseInvalidComposition)
	}
	theme, err := p.resolveTheme(ctx, input.TargetThemeID)
	if err != nil {
		return PlannedWebRelease{}, err
	}
	items, err := p.extensions.List(ctx)
	if err != nil {
		return PlannedWebRelease{}, err
	}
	extensions := make([]WebExtensionSnapshot, 0)
	for _, item := range items {
		if item.Type != TypePlugin || item.Manifest.Frontend.Admin == nil || !pluginEnabledForPlan(item, input) {
			continue
		}
		trusted, err := p.isTrusted(ctx, item)
		if err != nil {
			return PlannedWebRelease{}, err
		}
		if !trusted {
			continue
		}
		snapshot, err := p.extensionSnapshot(item)
		if err != nil {
			return PlannedWebRelease{}, err
		}
		extensions = append(extensions, snapshot)
	}
	sort.Slice(extensions, func(i, j int) bool {
		return extensions[i].ExtensionID < extensions[j].ExtensionID
	})
	for index := range extensions {
		extensions[index].SortOrder = index
	}
	composition := WebComposition{
		Theme:      theme,
		Extensions: extensions,
		WebSource:  strings.TrimSpace(p.host.WebSource),
		WebLock:    strings.TrimSpace(p.host.WebLock),
		SDKVersion: p.host.SDKVersion,
		BunVersion: strings.TrimSpace(p.host.BunVersion),
		Contract:   p.host.Contract,
	}
	if composition.WebSource == "" || composition.WebLock == "" || composition.SDKVersion < 1 || composition.BunVersion == "" || composition.Contract < 1 {
		return PlannedWebRelease{}, fmt.Errorf("%w: host build identity is incomplete", ErrWebReleaseInvalidComposition)
	}
	body, err := json.Marshal(composition)
	if err != nil {
		return PlannedWebRelease{}, fmt.Errorf("marshal web release composition: %w", err)
	}
	digest := sha256.Sum256(body)
	return PlannedWebRelease{
		Composition: composition,
		Snapshot:    append(json.RawMessage(nil), body...),
		Hash:        hex.EncodeToString(digest[:]),
	}, nil
}

func (p *WebReleasePlanner) resolveTheme(ctx context.Context, targetID string) (WebThemeSnapshot, error) {
	var theme Extension
	var err error
	if strings.TrimSpace(targetID) == "" {
		theme, err = p.extensions.ActiveTheme(ctx)
	} else {
		theme, err = p.extensions.Get(ctx, strings.TrimSpace(targetID))
	}
	if err != nil {
		return WebThemeSnapshot{}, err
	}
	if theme.Type != TypeTheme || strings.TrimSpace(theme.Manifest.Frontend.Layer) == "" {
		return WebThemeSnapshot{}, fmt.Errorf("%w: extension %s is not a buildable theme", ErrWebReleaseInvalidComposition, theme.ID)
	}
	if err := verifyPlannedPackage(theme); err != nil {
		return WebThemeSnapshot{}, err
	}
	layerPath, ok := InstalledFilePathForRuntime(theme, theme.Manifest.Frontend.Layer)
	if !ok {
		return WebThemeSnapshot{}, fmt.Errorf("%w: theme layer path is invalid", ErrWebReleaseInvalidComposition)
	}
	info, err := os.Stat(layerPath)
	if err != nil || !info.IsDir() {
		return WebThemeSnapshot{}, fmt.Errorf("%w: theme layer is unavailable", ErrWebReleaseInvalidComposition)
	}
	return WebThemeSnapshot{
		ExtensionID:   theme.ID,
		Version:       theme.Version,
		LayerPath:     layerPath,
		PackageDigest: theme.PackageDigest,
	}, nil
}

func (p *WebReleasePlanner) isTrusted(ctx context.Context, extension Extension) (bool, error) {
	if extension.Source == SourceBuiltin {
		if !extension.IsSystem || extension.IsDeletable {
			return false, fmt.Errorf("%w: builtin frontend %s lacks protected source policy", ErrWebReleaseInvalidComposition, extension.ID)
		}
		return true, nil
	}
	if extension.Source != SourceUploaded {
		return false, nil
	}
	grant, err := p.grants.FrontendGrant(ctx, extension.ID, extension.Version, extension.PackageDigest)
	if errors.Is(err, ErrFrontendGrantNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if grant.RevocationRequestedAt != nil || grant.RevokedAt != nil {
		return false, nil
	}
	admin := extension.Manifest.Frontend.Admin
	return grant.APIVersion == admin.APIVersion, nil
}

func (p *WebReleasePlanner) extensionSnapshot(extension Extension) (WebExtensionSnapshot, error) {
	if err := verifyPlannedPackage(extension); err != nil {
		return WebExtensionSnapshot{}, err
	}
	admin := extension.Manifest.Frontend.Admin
	dependencies, err := extensionpackage.InspectAdminFrontend(extensionpackage.FrontendInspectInput{
		PackageRoot: extension.PackagePath,
		Root:        admin.Root,
		Components:  admin.Components,
		Locales:     admin.Locales,
		HostPeers:   p.host.HostPeers,
	})
	if err != nil {
		return WebExtensionSnapshot{}, fmt.Errorf("inspect admin frontend %s: %w", extension.ID, err)
	}
	contributions, err := trustedComponentContributions(extension.Manifest, admin.Components)
	if err != nil {
		return WebExtensionSnapshot{}, err
	}
	componentMap := cloneStringMap(admin.Components)
	localeMap := cloneStringMap(admin.Locales)
	localeBody, err := json.Marshal(localeMap)
	if err != nil {
		return WebExtensionSnapshot{}, err
	}
	localeDigest := sha256.Sum256(localeBody)
	return WebExtensionSnapshot{
		ExtensionID:     extension.ID,
		Version:         extension.Version,
		PackageDigest:   extension.PackageDigest,
		FrontendRoot:    admin.Root,
		ComponentMap:    componentMap,
		APIVersion:      admin.APIVersion,
		Contributions:   contributions,
		LocaleMap:       localeMap,
		LocaleMapDigest: hex.EncodeToString(localeDigest[:]),
		Dependencies:    dependencies,
	}, nil
}

func verifyPlannedPackage(extension Extension) error {
	expected := strings.TrimSpace(extension.PackageDigest)
	if expected == "" || strings.TrimSpace(extension.PackagePath) == "" {
		return fmt.Errorf("%w: extension %s has no immutable package identity", ErrWebReleasePackageChanged, extension.ID)
	}
	actual, err := extensionpackage.DigestTree(extension.PackagePath)
	if err != nil {
		return fmt.Errorf("%w: extension %s: %v", ErrWebReleasePackageChanged, extension.ID, err)
	}
	if actual != expected {
		return fmt.Errorf("%w: extension %s digest expected %s, got %s", ErrWebReleasePackageChanged, extension.ID, expected, actual)
	}
	return nil
}

func pluginEnabledForPlan(extension Extension, input PlanWebReleaseInput) bool {
	if extension.ID == input.TriggerExtensionID {
		switch input.TriggerKind {
		case WebReleaseTriggerPluginEnable:
			return true
		case WebReleaseTriggerPluginDisable:
			return false
		}
	}
	return extension.Status == StatusEnabled
}

func trustedComponentContributions(manifest Manifest, components map[string]string) ([]ManifestContribution, error) {
	result := make([]ManifestContribution, 0)
	for _, contribution := range manifest.Contributions {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(contribution.Payload, &raw); err != nil {
			continue
		}
		if _, ok := raw["component"]; !ok {
			continue
		}
		var payload AdminComponentContributionPayload
		if err := json.Unmarshal(contribution.Payload, &payload); err != nil || components[payload.Component] == "" {
			return nil, fmt.Errorf("%w: invalid component contribution %s/%s", ErrWebReleaseInvalidComposition, manifest.ID, contribution.ID)
		}
		copy := contribution
		copy.Label = cloneStringMap(contribution.Label)
		copy.Payload = append(json.RawMessage(nil), contribution.Payload...)
		result = append(result, copy)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Order != result[j].Order {
			return result[i].Order < result[j].Order
		}
		if result[i].ID != result[j].ID {
			return result[i].ID < result[j].ID
		}
		return result[i].Point < result[j].Point
	})
	return result, nil
}

func cloneStringMap(value map[string]string) map[string]string {
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

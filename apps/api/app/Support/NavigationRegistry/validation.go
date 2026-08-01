package navigationregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	semver "github.com/Masterminds/semver/v3"
)

const (
	maxPublications               = 512
	maxContributions              = 4096
	maxDependenciesPerPublication = 256
	maxKindFilters                = 16
	maxPriority                   = 1_000_000
	maxTargetDepth                = 16
	maxLocalesPerDeclaration      = 32
	maxRuntimeInstanceIDLength    = 160
	maxNavigationHrefRunes        = 2048
)

var (
	idPattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,120}$`)
	digestPattern   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	contractPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$`)
	localePattern   = regexp.MustCompile(`^[a-zA-Z]{2,8}([_-][a-zA-Z0-9]{1,8})*$`)
	opaquePattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)
)

func normalizePublications(input []Publication) ([]Publication, error) {
	if len(input) > maxPublications {
		return nil, ErrInvalid
	}
	result := make([]Publication, 0, len(input))
	seen := map[string]bool{}
	count := 0
	for _, publication := range input {
		normalized, err := normalizePublication(publication)
		if err != nil {
			return nil, err
		}
		if seen[normalized.Artifact.ExtensionID] {
			return nil, fmt.Errorf("%w: duplicate publication %s", ErrConflict, normalized.Artifact.ExtensionID)
		}
		seen[normalized.Artifact.ExtensionID] = true
		count += len(normalized.Navigation) + len(normalized.Regions)
		if count > maxContributions {
			return nil, ErrInvalid
		}
		result = append(result, normalized)
	}
	sort.Slice(result, func(i, j int) bool { return artifactBefore(result[i].Artifact, result[j].Artifact) })
	return result, nil
}

func normalizePublication(input Publication) (Publication, error) {
	artifact, err := normalizeArtifact(input.Artifact)
	if err != nil {
		return Publication{}, ErrInvalid
	}
	result := Publication{Artifact: artifact}
	seen := map[string]bool{}
	if len(input.Dependencies) > maxDependenciesPerPublication {
		return Publication{}, ErrInvalid
	}
	for _, raw := range input.Dependencies {
		dependency, err := normalizeDependency(raw)
		if err != nil {
			return Publication{}, err
		}
		key := dependency.Kind + "\x00" + dependency.ExtensionID + "\x00" + dependency.Capability
		if seen[key] {
			return Publication{}, ErrInvalid
		}
		seen[key] = true
		result.Dependencies = append(result.Dependencies, dependency)
	}
	for _, raw := range input.Navigation {
		declaration, err := normalizeNavigation(artifact, raw)
		if err != nil || seen["contribution\x00"+declaration.ID] {
			return Publication{}, ErrInvalid
		}
		seen["contribution\x00"+declaration.ID] = true
		result.Navigation = append(result.Navigation, declaration)
	}
	for _, raw := range input.Regions {
		declaration, err := normalizeRegion(artifact, raw)
		if err != nil || seen["contribution\x00"+declaration.ID] {
			return Publication{}, ErrInvalid
		}
		seen["contribution\x00"+declaration.ID] = true
		result.Regions = append(result.Regions, declaration)
	}
	sort.Slice(result.Dependencies, func(i, j int) bool {
		left := result.Dependencies[i].Kind + "\x00" + result.Dependencies[i].ExtensionID + "\x00" + result.Dependencies[i].Capability
		right := result.Dependencies[j].Kind + "\x00" + result.Dependencies[j].ExtensionID + "\x00" + result.Dependencies[j].Capability
		return left < right
	})
	sort.Slice(result.Navigation, func(i, j int) bool { return result.Navigation[i].ID < result.Navigation[j].ID })
	sort.Slice(result.Regions, func(i, j int) bool { return result.Regions[i].ID < result.Regions[j].ID })
	return result, nil
}

func normalizeArtifact(input Artifact) (Artifact, error) {
	input.ExtensionID = strings.ToLower(strings.TrimSpace(input.ExtensionID))
	input.ExtensionVersion = strings.TrimSpace(input.ExtensionVersion)
	input.PackageDigest = normalizeDigest(input.PackageDigest)
	input.ImpactDigest = normalizeDigest(input.ImpactDigest)
	input.RuntimeInstanceID = strings.TrimSpace(input.RuntimeInstanceID)
	isCoreNamespace := strings.HasPrefix(input.ExtensionID, "core.")
	isTrustedCore := validCoreArtifactSeal(input)
	if !idPattern.MatchString(input.ExtensionID) || !digestPattern.MatchString(input.PackageDigest) ||
		!digestPattern.MatchString(input.ImpactDigest) || input.ExtensionID == "core" {
		return Artifact{}, ErrInvalid
	}
	if _, err := semver.StrictNewVersion(input.ExtensionVersion); err != nil {
		return Artifact{}, ErrInvalid
	}
	if input.Core {
		if !isCoreNamespace || !isTrustedCore || input.VersionID != 0 || input.RuntimeInstanceID != "" {
			return Artifact{}, ErrInvalid
		}
	} else if input.coreSeal != [32]byte{} || isCoreNamespace || input.VersionID <= 0 ||
		len(input.RuntimeInstanceID) > maxRuntimeInstanceIDLength || !opaquePattern.MatchString(input.RuntimeInstanceID) {
		return Artifact{}, ErrInvalid
	}
	return input, nil
}

// NewCoreArtifact is the Host-only construction boundary for core navigation
// catalogs. Callers must pass Host-owned constants, never extension input.
func NewCoreArtifact(extensionID, extensionVersion, packageDigest, impactDigest string) (Artifact, error) {
	artifact := Artifact{
		ExtensionID: strings.ToLower(strings.TrimSpace(extensionID)), ExtensionVersion: strings.TrimSpace(extensionVersion),
		PackageDigest: normalizeDigest(packageDigest), ImpactDigest: normalizeDigest(impactDigest), Core: true,
	}
	artifact.coreSeal = coreArtifactSeal(artifact)
	return normalizeArtifact(artifact)
}

func coreArtifactSeal(artifact Artifact) [32]byte {
	material := SchemaVersion + "\x00core-artifact\x00" + artifact.ExtensionID + "\x00" +
		artifact.ExtensionVersion + "\x00" + artifact.PackageDigest + "\x00" + artifact.ImpactDigest
	return sha256.Sum256([]byte(material))
}

func validCoreArtifactSeal(artifact Artifact) bool {
	return artifact.Core && artifact.coreSeal != [32]byte{} && artifact.coreSeal == coreArtifactSeal(artifact)
}

func normalizeDependency(input Dependency) (Dependency, error) {
	input.ExtensionID = strings.ToLower(strings.TrimSpace(input.ExtensionID))
	input.Capability = strings.ToLower(strings.TrimSpace(input.Capability))
	input.Version = strings.TrimSpace(input.Version)
	input.Kind = strings.ToLower(strings.TrimSpace(input.Kind))
	switch input.Kind {
	case DependencyRequired, DependencyOptional, DependencyConflict:
		if (input.ExtensionID == "") == (input.Capability == "") ||
			(input.ExtensionID != "" && !idPattern.MatchString(input.ExtensionID)) ||
			(input.Capability != "" && !idPattern.MatchString(input.Capability)) {
			return Dependency{}, ErrInvalid
		}
		if _, err := semver.NewConstraint(input.Version); err != nil {
			return Dependency{}, ErrInvalid
		}
	case DependencyProvides:
		if input.ExtensionID != "" || !idPattern.MatchString(input.Capability) {
			return Dependency{}, ErrInvalid
		}
		if _, err := semver.StrictNewVersion(input.Version); err != nil {
			return Dependency{}, ErrInvalid
		}
	default:
		return Dependency{}, ErrInvalid
	}
	return input, nil
}

func normalizeNavigation(artifact Artifact, input NavigationDeclaration) (NavigationDeclaration, error) {
	input.ID = strings.ToLower(strings.TrimSpace(input.ID))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.Kind = strings.ToLower(strings.TrimSpace(input.Kind))
	input.Action = strings.ToLower(strings.TrimSpace(input.Action))
	input.TargetID = strings.ToLower(strings.TrimSpace(input.TargetID))
	input.Label = strings.TrimSpace(input.Label)
	labels, err := normalizeLabels(input.Labels)
	if err != nil {
		return NavigationDeclaration{}, err
	}
	input.Labels = labels
	input.Href = strings.TrimSpace(input.Href)
	input.Permission = strings.ToLower(strings.TrimSpace(input.Permission))
	input.OwnerResource = strings.ToLower(strings.TrimSpace(input.OwnerResource))
	input.Visibility = normalizeVisibilityPolicy(input.Visibility)
	input.Handler = strings.ToLower(strings.TrimSpace(input.Handler))
	if !validContributionIdentity(artifact, input.ID, input.ContractVersion) || !validAction(input.Action) ||
		!validNavigationKind(input.Kind) || !validLabelSet(input.Label, input.Labels) || input.Order < 0 ||
		input.Order > 1_000_000 || input.Priority < -maxPriority || input.Priority > maxPriority ||
		(input.Action != ActionAdd && input.TargetID == "") ||
		(input.TargetID != "" && !idPattern.MatchString(input.TargetID)) ||
		(input.Permission != "" && !idPattern.MatchString(input.Permission)) ||
		(input.OwnerResource != "" && !idPattern.MatchString(input.OwnerResource)) ||
		!validVisibilityPolicy(input.Visibility) ||
		(input.Handler != "" && (!idPattern.MatchString(input.Handler) || !strings.HasPrefix(input.Handler, artifact.ExtensionID+"."))) ||
		(input.Href != "" && (utf8.RuneCountInString(input.Href) > maxNavigationHrefRunes || !safeHostLinkPath(input.Href))) {
		return NavigationDeclaration{}, ErrInvalid
	}
	return input, nil
}

func normalizeRegion(artifact Artifact, input RegionDeclaration) (RegionDeclaration, error) {
	input.ID = strings.ToLower(strings.TrimSpace(input.ID))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.Action = strings.ToLower(strings.TrimSpace(input.Action))
	input.TargetID = strings.ToLower(strings.TrimSpace(input.TargetID))
	input.Kind = strings.ToLower(strings.TrimSpace(input.Kind))
	input.Label = strings.TrimSpace(input.Label)
	labels, err := normalizeLabels(input.Labels)
	if err != nil {
		return RegionDeclaration{}, err
	}
	input.Labels = labels
	input.Permission = strings.ToLower(strings.TrimSpace(input.Permission))
	input.Visibility = normalizeVisibilityPolicy(input.Visibility)
	input.Handler = strings.ToLower(strings.TrimSpace(input.Handler))
	if !validContributionIdentity(artifact, input.ID, input.ContractVersion) || !validAction(input.Action) ||
		!validRegionKind(input.Kind) || !validLabelSet(input.Label, input.Labels) || input.Order < 0 ||
		input.Order > 1_000_000 || input.Priority < -maxPriority || input.Priority > maxPriority ||
		(input.Action != ActionAdd && input.TargetID == "") ||
		(input.TargetID != "" && !idPattern.MatchString(input.TargetID)) ||
		(input.Permission != "" && !idPattern.MatchString(input.Permission)) ||
		!validVisibilityPolicy(input.Visibility) ||
		(input.Handler != "" && (!idPattern.MatchString(input.Handler) || !strings.HasPrefix(input.Handler, artifact.ExtensionID+"."))) {
		return RegionDeclaration{}, ErrInvalid
	}
	return input, nil
}

func computeGraphDigest(publications []Publication, safeMode bool, selections []ProviderSelection) string {
	if publications == nil {
		publications = []Publication{}
	}
	if selections == nil {
		selections = []ProviderSelection{}
	}
	body, _ := json.Marshal(struct {
		SchemaVersion string              `json:"schemaVersion"`
		SafeMode      bool                `json:"safeMode"`
		Publications  []Publication       `json:"publications"`
		Selections    []ProviderSelection `json:"selections"`
	}{SchemaVersion: SchemaVersion, SafeMode: safeMode, Publications: publications, Selections: selections})
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func normalizeLabels(input map[string]string) (map[string]string, error) {
	if len(input) > maxLocalesPerDeclaration {
		return nil, ErrInvalid
	}
	if len(input) == 0 {
		return nil, nil
	}
	result := make(map[string]string, len(input))
	for rawLocale, rawLabel := range input {
		locale, err := normalizeLocale(rawLocale)
		label := strings.TrimSpace(rawLabel)
		if err != nil || label == "" || len(label) > 1024 || len([]rune(label)) > 256 {
			return nil, ErrInvalid
		}
		if _, duplicate := result[locale]; duplicate {
			return nil, ErrInvalid
		}
		result[locale] = label
	}
	return result, nil
}

func normalizeLocale(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "_", "-"))
	if value == "" {
		return "zh-CN", nil
	}
	if len(value) > 64 || !localePattern.MatchString(value) {
		return "", ErrInvalid
	}
	parts := strings.Split(value, "-")
	parts[0] = strings.ToLower(parts[0])
	for index := 1; index < len(parts); index++ {
		if len(parts[index]) == 2 || len(parts[index]) == 3 {
			parts[index] = strings.ToUpper(parts[index])
		} else {
			parts[index] = strings.ToLower(parts[index])
		}
	}
	return strings.Join(parts, "-"), nil
}

func localizedLabel(label string, labels map[string]string, locale string) string {
	if value := labels[locale]; value != "" {
		return value
	}
	language := strings.Split(locale, "-")[0]
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if strings.Split(key, "-")[0] == language {
			return labels[key]
		}
	}
	if label == "" && len(keys) > 0 {
		return labels[keys[0]]
	}
	return label
}

func validLabelSet(label string, labels map[string]string) bool {
	return (label != "" || len(labels) > 0) && len(label) <= 1024 && len([]rune(label)) <= 256
}

func normalizeVisibilityPolicy(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return VisibilityPublic
	}
	return value
}

func validVisibilityPolicy(value string) bool {
	return value == VisibilityPublic || value == VisibilityAnonymous || value == VisibilityAuthenticated
}

func normalizeDigest(value string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "sha256:")
}

func validContributionIdentity(artifact Artifact, id, contract string) bool {
	return idPattern.MatchString(id) && strings.HasPrefix(id, artifact.ExtensionID+".") &&
		contractPattern.MatchString(contract) && strings.HasPrefix(contract, id+"@")
}

func validAction(value string) bool {
	switch value {
	case ActionAdd, ActionBefore, ActionAfter, ActionWrap, ActionReplace, ActionHide, ActionFilter:
		return true
	default:
		return false
	}
}

func validNavigationKind(value string) bool {
	switch value {
	case NavigationKindMenu, NavigationKindItem, NavigationKindBreadcrumb, NavigationKindHeader, NavigationKindFooter, NavigationKindSidebar:
		return true
	case NavigationKindAccountSettings:
		return true
	default:
		return false
	}
}

func validRegionKind(value string) bool {
	switch value {
	case RegionKindMenu, RegionKindWidget, RegionKindHeader, RegionKindFooter, RegionKindSidebar, RegionKindContent:
		return true
	default:
		return false
	}
}

func safeHostLinkPath(value string) bool {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	return value != "" && strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") &&
		!strings.Contains(value, "://") && !strings.Contains(value, "..") &&
		value != "/api" && !strings.HasPrefix(value, "/api/")
}

func artifactBefore(left, right Artifact) bool {
	if left.Core != right.Core {
		return left.Core
	}
	if left.ExtensionID != right.ExtensionID {
		return left.ExtensionID < right.ExtensionID
	}
	if left.ExtensionVersion != right.ExtensionVersion {
		return left.ExtensionVersion < right.ExtensionVersion
	}
	if left.PackageDigest != right.PackageDigest {
		return left.PackageDigest < right.PackageDigest
	}
	if left.ImpactDigest != right.ImpactDigest {
		return left.ImpactDigest < right.ImpactDigest
	}
	if left.VersionID != right.VersionID {
		return left.VersionID < right.VersionID
	}
	return left.RuntimeInstanceID < right.RuntimeInstanceID
}

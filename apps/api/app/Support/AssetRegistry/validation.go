package assetregistry

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"path"
	"slices"
	"sort"
	"strings"

	semver "github.com/Masterminds/semver/v3"
)

// Limits bound both individual declarations and aggregate graph work. The
// aggregate limits matter because individually valid publications can otherwise
// create an unbounded startup graph or CSP response policy.
const (
	maxRegistryOwners       = 512
	maxRegistryAssets       = 4096
	maxRegistryDependencies = 16384
	maxRegistryScopes       = 16384
	maxRegistryCSP          = 8192

	maxAssetsPerPublication       = 512
	maxDependenciesPerPublication = 4096
	maxScopesPerPublication       = 4096
	maxCSPPerPublication          = 2048

	maxDependenciesPerAsset = 64
	maxScopesPerAsset       = 64
	maxCSPPerAsset          = 32

	maxPlanHandles = maxRegistryAssets
	maxPlanScopes  = maxRegistryScopes
)

func normalizePublication(publication Publication) (Publication, error) {
	artifact, err := normalizeArtifact(publication.Artifact)
	if err != nil || len(publication.Assets) > maxAssetsPerPublication {
		return Publication{}, ErrInvalid
	}
	assets := make([]Declaration, 0, len(publication.Assets))
	seen := map[string]struct{}{}
	dependencies, scopes, csp := 0, 0, 0
	for _, declaration := range publication.Assets {
		normalized, err := normalizeDeclaration(artifact, declaration)
		if err != nil {
			return Publication{}, err
		}
		if _, duplicate := seen[normalized.Handle]; duplicate {
			return Publication{}, ErrConflict
		}
		seen[normalized.Handle] = struct{}{}
		assets = append(assets, normalized)
		dependencies += len(normalized.Dependencies)
		scopes += len(normalized.Scope)
		csp += len(normalized.CSP)
		if dependencies > maxDependenciesPerPublication || scopes > maxScopesPerPublication ||
			csp > maxCSPPerPublication {
			return Publication{}, ErrInvalid
		}
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Handle < assets[j].Handle })
	return Publication{Artifact: artifact, Assets: assets}, nil
}

func normalizeArtifact(artifact Artifact) (Artifact, error) {
	artifact.ExtensionID = strings.ToLower(strings.TrimSpace(artifact.ExtensionID))
	artifact.ExtensionVersion = strings.TrimSpace(artifact.ExtensionVersion)
	artifact.PackageDigest = normalizeDigest(artifact.PackageDigest)
	artifact.ImpactDigest = normalizeDigest(artifact.ImpactDigest)
	if !idPattern.MatchString(artifact.ExtensionID) || artifact.ExtensionID == "core" || len(artifact.ExtensionVersion) > 64 ||
		!digestPattern.MatchString(artifact.PackageDigest) || !digestPattern.MatchString(artifact.ImpactDigest) ||
		artifact.Core != strings.HasPrefix(artifact.ExtensionID, "core.") {
		return Artifact{}, ErrInvalid
	}
	if _, err := semver.StrictNewVersion(artifact.ExtensionVersion); err != nil {
		return Artifact{}, ErrInvalid
	}
	return artifact, nil
}

func normalizeDeclaration(artifact Artifact, input Declaration) (Declaration, error) {
	input.Handle = strings.ToLower(strings.TrimSpace(input.Handle))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.Type = strings.ToLower(strings.TrimSpace(input.Type))
	input.Path = strings.TrimSpace(input.Path)
	input.Digest = normalizeDigest(input.Digest)
	input.Loading = strings.ToLower(strings.TrimSpace(input.Loading))
	input.Integrity = strings.TrimSpace(input.Integrity)
	var err error
	if input.Dependencies, err = normalizedIDs(input.Dependencies, maxDependenciesPerAsset); err != nil {
		return Declaration{}, err
	}
	if input.Scope, err = normalizedIDs(input.Scope, maxScopesPerAsset); err != nil {
		return Declaration{}, err
	}
	if input.CSP, err = normalizedStrings(input.CSP, maxCSPPerAsset); err != nil {
		return Declaration{}, err
	}
	if !validAssetIdentity(artifact, input.Handle, input.ContractVersion) ||
		(input.Type != "script" && input.Type != "style") || !safePackagePath(input.Path) ||
		!validAssetPath(input.Type, input.Path) || !digestPattern.MatchString(input.Digest) ||
		(isReservedCoreHandle(input.Handle) && !isCoreArtifact(artifact)) {
		return Declaration{}, ErrInvalid
	}
	if input.Type == "style" && input.Module {
		return Declaration{}, ErrInvalid
	}
	switch input.Loading {
	case "", "blocking", "defer", "async", "preload", "lazy":
	default:
		return Declaration{}, ErrInvalid
	}
	if slices.Contains(input.Dependencies, input.Handle) {
		return Declaration{}, ErrDependency
	}
	wantIntegrity, err := digestIntegrity(input.Digest)
	if err != nil {
		return Declaration{}, ErrInvalid
	}
	if input.Integrity != "" && input.Integrity != wantIntegrity {
		return Declaration{}, ErrInvalid
	}
	input.Integrity = wantIntegrity
	for _, declaration := range input.CSP {
		if !validCSPDeclaration(declaration) {
			return Declaration{}, ErrInvalid
		}
	}
	return input, nil
}

func validateGraph(assets map[string]Asset) error {
	for _, asset := range assets {
		for _, dependency := range asset.Dependencies {
			// core.asset.* is supplied by the Host outside this plugin registry.
			// Every other dependency names a required active provider and therefore
			// fails publication/removal closed when that provider is absent.
			if _, exists := assets[dependency]; !exists && !isCoreAssetHandle(dependency) {
				return ErrDependency
			}
		}
	}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string) error
	visit = func(handle string) error {
		if visited[handle] {
			return nil
		}
		asset, exists := assets[handle]
		if !exists {
			if isCoreAssetHandle(handle) {
				return nil
			}
			return ErrDependency
		}
		if visiting[handle] {
			return ErrDependency
		}
		visiting[handle] = true
		for _, dependency := range asset.Dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		delete(visiting, handle)
		visited[handle] = true
		return nil
	}
	for handle := range assets {
		if err := visit(handle); err != nil {
			return err
		}
	}
	return nil
}

func isCoreAssetHandle(handle string) bool {
	return strings.HasPrefix(handle, "core.asset.")
}

func isReservedCoreHandle(handle string) bool {
	return handle == "core" || strings.HasPrefix(handle, "core.")
}

func isCoreArtifact(artifact Artifact) bool {
	return artifact.Core
}

func validAssetIdentity(artifact Artifact, handle, contractVersion string) bool {
	if !idPattern.MatchString(handle) || !contractPattern.MatchString(contractVersion) {
		return false
	}
	contractID, _, found := strings.Cut(contractVersion, "@")
	if !found {
		return false
	}
	if artifact.Core {
		// Host-owned stable IDs use core.* while their public contracts use
		// the established sforum.* namespace (for example core.asset.vue ->
		// sforum.asset.vue@1).
		return isCoreAssetHandle(handle) && contractID == "sforum."+strings.TrimPrefix(handle, "core.")
	}
	return strings.HasPrefix(handle, artifact.ExtensionID+".") && contractID == handle
}

func computeGraphDigest(publications []Publication) string {
	if publications == nil {
		publications = []Publication{}
	}
	body, _ := json.Marshal(publications)
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func safePackagePath(value string) bool {
	if value == "" || len(value) > 512 || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") {
		return false
	}
	clean := path.Clean(value)
	return clean == value && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func validAssetPath(assetType, value string) bool {
	extension := strings.ToLower(path.Ext(value))
	if assetType == "style" {
		return extension == ".css"
	}
	return extension == ".js" || extension == ".mjs"
}

func validCSPDeclaration(value string) bool {
	if value == "" || len(value) > 512 || strings.ContainsAny(value, ";\r\n\x00") {
		return false
	}
	fields := strings.Fields(value)
	if len(fields) < 2 {
		return false
	}
	switch fields[0] {
	case "connect-src", "font-src", "img-src", "media-src", "script-src", "style-src", "worker-src":
	default:
		return false
	}
	for _, source := range fields[1:] {
		if len(source) > 240 || strings.ContainsAny(source, "\"<>\\") {
			return false
		}
	}
	return true
}

func normalizeDigest(value string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "sha256:")
}

func digestIntegrity(digest string) (string, error) {
	raw, err := hex.DecodeString(digest)
	if err != nil || len(raw) != 32 {
		return "", ErrInvalid
	}
	return "sha256-" + base64.StdEncoding.EncodeToString(raw), nil
}

func normalizedIDs(values []string, maximum int) ([]string, error) {
	if len(values) > maximum {
		return nil, ErrInvalid
	}
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if !idPattern.MatchString(value) {
			return nil, ErrInvalid
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, ErrInvalid
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func normalizedStrings(values []string, maximum int) ([]string, error) {
	if len(values) > maximum {
		return nil, ErrInvalid
	}
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.Join(strings.Fields(value), " ")
		if value == "" {
			return nil, ErrInvalid
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, ErrInvalid
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func strictPlanIDs(values []string, maximum int) ([]string, error) {
	if len(values) > maximum {
		return nil, ErrInvalid
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		// Plan is a runtime selection boundary. Reject malformed input instead
		// of silently trimming or folding it into a different identity.
		if value != strings.ToLower(strings.TrimSpace(value)) || !idPattern.MatchString(value) {
			return nil, ErrInvalid
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, ErrInvalid
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

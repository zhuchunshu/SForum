package assetregistry

import (
	"encoding/base64"
	"encoding/hex"
	"path"
	"slices"
	"sort"
	"strings"

	semver "github.com/Masterminds/semver/v3"
)

func normalizePublication(publication Publication) (Publication, error) {
	artifact := publication.Artifact
	artifact.ExtensionID = strings.ToLower(strings.TrimSpace(artifact.ExtensionID))
	artifact.ExtensionVersion = strings.TrimSpace(artifact.ExtensionVersion)
	artifact.PackageDigest = normalizeDigest(artifact.PackageDigest)
	artifact.ImpactDigest = normalizeDigest(artifact.ImpactDigest)
	if !idPattern.MatchString(artifact.ExtensionID) || len(artifact.ExtensionVersion) > 64 ||
		!digestPattern.MatchString(artifact.PackageDigest) || !digestPattern.MatchString(artifact.ImpactDigest) ||
		len(publication.Assets) > 512 {
		return Publication{}, ErrInvalid
	}
	if _, err := semver.StrictNewVersion(artifact.ExtensionVersion); err != nil {
		return Publication{}, ErrInvalid
	}
	assets := make([]Declaration, 0, len(publication.Assets))
	seen := map[string]struct{}{}
	for _, declaration := range publication.Assets {
		normalized, err := normalizeDeclaration(declaration)
		if err != nil {
			return Publication{}, err
		}
		if _, duplicate := seen[normalized.Handle]; duplicate {
			return Publication{}, ErrConflict
		}
		seen[normalized.Handle] = struct{}{}
		assets = append(assets, normalized)
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Handle < assets[j].Handle })
	return Publication{Artifact: artifact, Assets: assets}, nil
}

func normalizeDeclaration(input Declaration) (Declaration, error) {
	input.Handle = strings.ToLower(strings.TrimSpace(input.Handle))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.Type = strings.ToLower(strings.TrimSpace(input.Type))
	input.Path = strings.TrimSpace(input.Path)
	input.Digest = normalizeDigest(input.Digest)
	input.Loading = strings.ToLower(strings.TrimSpace(input.Loading))
	input.Integrity = strings.TrimSpace(input.Integrity)
	var err error
	if input.Dependencies, err = normalizedIDs(input.Dependencies, 64); err != nil {
		return Declaration{}, err
	}
	if input.Scope, err = normalizedIDs(input.Scope, 64); err != nil {
		return Declaration{}, err
	}
	if input.CSP, err = normalizedStrings(input.CSP, 32); err != nil {
		return Declaration{}, err
	}
	if !idPattern.MatchString(input.Handle) || !contractPattern.MatchString(input.ContractVersion) ||
		(input.Type != "script" && input.Type != "style") || !safePackagePath(input.Path) ||
		!validAssetPath(input.Type, input.Path) || !digestPattern.MatchString(input.Digest) {
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
			if _, exists := assets[dependency]; !exists && !strings.HasPrefix(dependency, "core.asset.") {
				return ErrDependency
			}
		}
	}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string) error
	visit = func(handle string) error {
		if visited[handle] || strings.HasPrefix(handle, "core.asset.") {
			return nil
		}
		if visiting[handle] {
			return ErrDependency
		}
		visiting[handle] = true
		for _, dependency := range assets[handle].Dependencies {
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

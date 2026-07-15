package assetregistry

import (
	"slices"
	"sort"
)

// ReplaceAll builds one complete immutable snapshot. Startup, restart, and
// multi-node convergence should use this path so publication order cannot pick
// a different winner for conflicting handles.
func (r *Registry) ReplaceAll(publications []Publication) (uint64, error) {
	if r == nil || len(publications) > maxRegistryOwners {
		return 0, ErrInvalid
	}
	normalized, err := normalizePublications(publications)
	if err != nil {
		return r.load().revision, err
	}
	next := make(map[string]Asset)
	for _, publication := range normalized {
		for _, declaration := range publication.Assets {
			if _, exists := next[declaration.Handle]; exists {
				return r.load().revision, ErrConflict
			}
			next[declaration.Handle] = Asset{Declaration: declaration, Artifact: publication.Artifact}
			if len(next) > maxRegistryAssets {
				return r.load().revision, ErrInvalid
			}
		}
	}
	if err := validateGraph(next); err != nil {
		return r.load().revision, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	if equalAssetMaps(current.assets, next) {
		return current.revision, nil
	}
	revision := current.revision + 1
	r.state.Store(&registryState{revision: revision, assets: next})
	return revision, nil
}

func normalizePublications(publications []Publication) ([]Publication, error) {
	normalized := make([]Publication, 0, len(publications))
	owners := make(map[string]struct{}, len(publications))
	for _, publication := range publications {
		item, err := normalizePublication(publication)
		if err != nil {
			return nil, err
		}
		if _, duplicate := owners[item.Artifact.ExtensionID]; duplicate {
			return nil, ErrConflict
		}
		owners[item.Artifact.ExtensionID] = struct{}{}
		normalized = append(normalized, item)
	}
	sort.Slice(normalized, func(i, j int) bool {
		left, right := normalized[i].Artifact, normalized[j].Artifact
		if left.ExtensionID != right.ExtensionID {
			return left.ExtensionID < right.ExtensionID
		}
		if left.ExtensionVersion != right.ExtensionVersion {
			return left.ExtensionVersion < right.ExtensionVersion
		}
		if left.PackageDigest != right.PackageDigest {
			return left.PackageDigest < right.PackageDigest
		}
		return left.ImpactDigest < right.ImpactDigest
	})
	return normalized, nil
}

func equalAssetMaps(left, right map[string]Asset) bool {
	if len(left) != len(right) {
		return false
	}
	for handle, a := range left {
		b, ok := right[handle]
		if !ok || a.Handle != b.Handle || a.ContractVersion != b.ContractVersion || a.Type != b.Type ||
			a.Path != b.Path || a.Digest != b.Digest || a.Module != b.Module || a.Loading != b.Loading ||
			a.Integrity != b.Integrity || a.Artifact != b.Artifact ||
			!slices.Equal(a.Dependencies, b.Dependencies) || !slices.Equal(a.Scope, b.Scope) || !slices.Equal(a.CSP, b.CSP) {
			return false
		}
	}
	return true
}

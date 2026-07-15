package assetregistry

import (
	"reflect"
	"sort"
)

// ReplaceAll builds one complete immutable snapshot. Startup, restart, and
// multi-node convergence should use this path so publication order cannot pick
// a different winner for conflicting handles.
func (r *Registry) ReplaceAll(publications []Publication) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	next, err := buildState(0, publications)
	if err != nil {
		return r.load().revision, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	if equalPublicationMaps(current.publications, next.publications) {
		return current.revision, nil
	}
	next.revision = current.revision + 1
	r.state.Store(next)
	return next.revision, nil
}

func buildState(revision uint64, publications []Publication) (*registryState, error) {
	normalized, err := normalizePublications(publications)
	if err != nil {
		return nil, err
	}
	state := &registryState{
		revision: revision, publications: make(map[string]Publication, len(normalized)), assets: map[string]Asset{},
	}
	state.digest = computeGraphDigest(normalized)
	for _, publication := range normalized {
		state.publications[publication.Artifact.ExtensionID] = clonePublication(publication)
		for _, declaration := range publication.Assets {
			if _, exists := state.assets[declaration.Handle]; exists {
				return nil, ErrConflict
			}
			state.assets[declaration.Handle] = Asset{Declaration: cloneDeclaration(declaration), Artifact: publication.Artifact}
		}
	}
	if err := validateGraph(state.assets); err != nil {
		return nil, err
	}
	return state, nil
}

func normalizePublications(publications []Publication) ([]Publication, error) {
	if len(publications) > maxRegistryOwners {
		return nil, ErrInvalid
	}
	normalized := make([]Publication, 0, len(publications))
	owners := make(map[string]struct{}, len(publications))
	assets, dependencies, scopes, csp := 0, 0, 0, 0
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
		assets += len(item.Assets)
		for _, declaration := range item.Assets {
			dependencies += len(declaration.Dependencies)
			scopes += len(declaration.Scope)
			csp += len(declaration.CSP)
		}
		if assets > maxRegistryAssets || dependencies > maxRegistryDependencies ||
			scopes > maxRegistryScopes || csp > maxRegistryCSP {
			return nil, ErrInvalid
		}
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

func equalPublicationMaps(left, right map[string]Publication) bool {
	return reflect.DeepEqual(left, right)
}

func equalPublications(left, right Publication) bool {
	return reflect.DeepEqual(left, right)
}

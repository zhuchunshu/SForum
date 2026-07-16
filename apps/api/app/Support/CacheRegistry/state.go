package cacheregistry

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
)

type registryState struct {
	revision     uint64
	digest       string
	safeMode     bool
	publications map[string]Publication
	caches       map[string]Contribution
	namespaces   map[string]Contribution
}

// artifactDeclarationIdentity intentionally excludes RuntimeInstanceID. A new
// process for the same immutable package/version row must publish the same
// declarations; rotating the runtime id is not authority to reinterpret bytes.
type artifactDeclarationIdentity struct {
	extensionID      string
	extensionVersion string
	packageDigest    string
	versionID        int64
	core             bool
}

func declarationIdentity(artifact Artifact) artifactDeclarationIdentity {
	return artifactDeclarationIdentity{
		extensionID: artifact.ExtensionID, extensionVersion: artifact.ExtensionVersion,
		packageDigest: artifact.PackageDigest, versionID: artifact.VersionID, core: artifact.Core,
	}
}

func emptyState() *registryState {
	return &registryState{
		digest:       computeGraphDigest(nil, false),
		publications: map[string]Publication{},
		caches:       map[string]Contribution{},
		namespaces:   map[string]Contribution{},
	}
}

func buildState(revision uint64, input []Publication, safeMode bool) (*registryState, error) {
	publications, err := normalizePublications(filterSafeModeInput(input, safeMode))
	if err != nil {
		return nil, err
	}
	state := &registryState{
		revision: revision, safeMode: safeMode, digest: computeGraphDigest(publications, safeMode),
		publications: make(map[string]Publication, len(publications)),
		caches:       map[string]Contribution{}, namespaces: map[string]Contribution{},
	}
	for _, publication := range publications {
		state.publications[publication.Artifact.ExtensionID] = clonePublication(publication)
		for _, declaration := range publication.Caches {
			contribution := Contribution{Declaration: cloneDeclaration(declaration), Artifact: publication.Artifact}
			if existing, duplicate := state.caches[declaration.ID]; duplicate {
				return nil, fmt.Errorf("%w: duplicate cache id %s owned by %s and %s",
					ErrConflict, declaration.ID, existing.Artifact.ExtensionID, publication.Artifact.ExtensionID)
			}
			if existing, duplicate := state.namespaces[declaration.Namespace]; duplicate {
				return nil, fmt.Errorf("%w: duplicate cache namespace %s owned by %s and %s",
					ErrConflict, declaration.Namespace, existing.Artifact.ExtensionID, publication.Artifact.ExtensionID)
			}
			state.caches[declaration.ID] = contribution
			state.namespaces[declaration.Namespace] = contribution
		}
	}
	return state, nil
}

func cloneDeclaration(value Declaration) Declaration {
	value.Tags = slices.Clone(value.Tags)
	value.Invalidators = slices.Clone(value.Invalidators)
	return value
}

func cloneContribution(value Contribution) Contribution {
	value.Declaration = cloneDeclaration(value.Declaration)
	return value
}

func clonePublication(value Publication) Publication {
	value.Caches = slices.Clone(value.Caches)
	for index := range value.Caches {
		value.Caches[index] = cloneDeclaration(value.Caches[index])
	}
	return value
}

func clonePublicationMap(values map[string]Publication) map[string]Publication {
	result := make(map[string]Publication, len(values))
	for extensionID, publication := range values {
		result[extensionID] = clonePublication(publication)
	}
	return result
}

func publicationValues(values map[string]Publication) []Publication {
	result := make([]Publication, 0, len(values))
	for _, publication := range values {
		result = append(result, clonePublication(publication))
	}
	return result
}

func sortedPublications(values map[string]Publication) []Publication {
	result := publicationValues(values)
	sort.Slice(result, func(i, j int) bool { return artifactBefore(result[i].Artifact, result[j].Artifact) })
	return result
}

func sortedContributions(values map[string]Contribution) []Contribution {
	result := make([]Contribution, 0, len(values))
	for _, contribution := range values {
		result = append(result, cloneContribution(contribution))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Namespace != result[j].Namespace {
			return result[i].Namespace < result[j].Namespace
		}
		if result[i].Artifact.ExtensionID != result[j].Artifact.ExtensionID {
			return result[i].Artifact.ExtensionID < result[j].Artifact.ExtensionID
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func snapshotFromState(state *registryState) Snapshot {
	return Snapshot{
		SchemaVersion: SchemaVersion, Revision: state.revision, Digest: state.digest, SafeMode: state.safeMode,
		Publications: sortedPublications(state.publications), Caches: sortedContributions(state.caches),
	}
}

func cloneSnapshot(value Snapshot) Snapshot {
	value.Publications = slices.Clone(value.Publications)
	for index := range value.Publications {
		value.Publications[index] = clonePublication(value.Publications[index])
	}
	value.Caches = slices.Clone(value.Caches)
	for index := range value.Caches {
		value.Caches[index] = cloneContribution(value.Caches[index])
	}
	return value
}

func equalPublicationMaps(left, right map[string]Publication) bool {
	return reflect.DeepEqual(left, right)
}

func equalPublications(left, right Publication) bool {
	return reflect.DeepEqual(left, right)
}

func equalPublicationDeclarations(left, right Publication) bool {
	return reflect.DeepEqual(left.Caches, right.Caches)
}

func validateExactPublicationReplay(current, next map[string]Publication) error {
	for extensionID, active := range current {
		candidate, found := next[extensionID]
		if !found || active.Artifact != candidate.Artifact {
			continue
		}
		if !equalPublications(active, candidate) {
			return fmt.Errorf("%w: exact artifact %s changed declarations", ErrArtifactConflict, extensionID)
		}
	}
	return nil
}

func validatePublicationHistory(history map[artifactDeclarationIdentity]Publication, publications map[string]Publication) error {
	for extensionID, candidate := range publications {
		sealed, found := history[declarationIdentity(candidate.Artifact)]
		if found && !equalPublicationDeclarations(sealed, candidate) {
			return fmt.Errorf("%w: immutable artifact %s changed declarations", ErrArtifactConflict, extensionID)
		}
	}
	return nil
}

func recordPublicationHistory(history map[artifactDeclarationIdentity]Publication, publications map[string]Publication) {
	for _, publication := range publications {
		history[declarationIdentity(publication.Artifact)] = clonePublication(publication)
	}
}

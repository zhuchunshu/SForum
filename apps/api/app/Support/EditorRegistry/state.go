package editorregistry

import (
	"fmt"
	"sort"
)

type registryState struct {
	revision     uint64
	digest       string
	safeMode     bool
	publications map[string]Publication
	editor       map[string]Contribution
}

func emptyState() *registryState {
	state := &registryState{
		publications: map[string]Publication{},
		editor:       map[string]Contribution{},
	}
	state.digest = computeGraphDigest(nil, false)
	return state
}

func buildState(revision uint64, input []Publication, safeMode bool) (*registryState, error) {
	publications, err := normalizePublications(filterSafeModeInput(input, safeMode))
	if err != nil {
		return nil, err
	}
	state := &registryState{
		revision:     revision,
		safeMode:     safeMode,
		publications: make(map[string]Publication, len(publications)),
		editor:       map[string]Contribution{},
	}
	for _, publication := range publications {
		state.publications[publication.Artifact.ExtensionID] = clonePublication(publication)
		for _, declaration := range publication.Editor {
			if existing, duplicate := state.editor[declaration.ID]; duplicate {
				return nil, fmt.Errorf("%w: duplicate editor id %s owned by %s and %s",
					ErrConflict, declaration.ID, existing.Artifact.ExtensionID, publication.Artifact.ExtensionID)
			}
			state.editor[declaration.ID] = Contribution{
				Declaration: cloneDeclaration(declaration),
				Artifact:    publication.Artifact,
			}
		}
	}
	// Cross-publication toolbar→command: command may live in same package only
	// (already enforced in normalizePublication). Graph is ready.
	state.digest = computeGraphDigest(sortedPublications(state.publications), safeMode)
	return state, nil
}

func sortedPublications(publications map[string]Publication) []Publication {
	result := make([]Publication, 0, len(publications))
	for _, publication := range publications {
		result = append(result, clonePublication(publication))
	}
	sort.Slice(result, func(i, j int) bool {
		return artifactBefore(result[i].Artifact, result[j].Artifact)
	})
	return result
}

func publicationValues(publications map[string]Publication) []Publication {
	return sortedPublications(publications)
}

func clonePublicationMap(input map[string]Publication) map[string]Publication {
	result := make(map[string]Publication, len(input))
	for key, value := range input {
		result[key] = clonePublication(value)
	}
	return result
}

func clonePublication(input Publication) Publication {
	result := Publication{Artifact: input.Artifact}
	if len(input.Editor) > 0 {
		result.Editor = make([]Declaration, len(input.Editor))
		for index, declaration := range input.Editor {
			result.Editor[index] = cloneDeclaration(declaration)
		}
	}
	return result
}

func cloneDeclaration(input Declaration) Declaration {
	return input
}

func cloneContribution(input Contribution) Contribution {
	return Contribution{Declaration: cloneDeclaration(input.Declaration), Artifact: input.Artifact}
}

func equalPublicationMaps(left, right map[string]Publication) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		other, ok := right[key]
		if !ok || !equalPublications(value, other) {
			return false
		}
	}
	return true
}

func equalStates(left, right *registryState) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.digest == right.digest && left.safeMode == right.safeMode &&
		equalPublicationMaps(left.publications, right.publications)
}

func snapshotFromState(state *registryState) Snapshot {
	if state == nil {
		state = emptyState()
	}
	snapshot := Snapshot{
		SchemaVersion: SchemaVersion,
		Revision:      state.revision,
		Digest:        state.digest,
		SafeMode:      state.safeMode,
		Publications:  sortedPublications(state.publications),
	}
	for _, contribution := range state.editor {
		snapshot.Editor = append(snapshot.Editor, cloneContribution(contribution))
	}
	sort.Slice(snapshot.Editor, func(i, j int) bool {
		return contributionBefore(snapshot.Editor[i], snapshot.Editor[j])
	})
	return snapshot
}

func cloneSnapshot(input Snapshot) Snapshot {
	result := Snapshot{
		SchemaVersion: input.SchemaVersion,
		Revision:      input.Revision,
		Digest:        input.Digest,
		SafeMode:      input.SafeMode,
	}
	if len(input.Publications) > 0 {
		result.Publications = make([]Publication, len(input.Publications))
		for index, publication := range input.Publications {
			result.Publications[index] = clonePublication(publication)
		}
	}
	if len(input.Editor) > 0 {
		result.Editor = make([]Contribution, len(input.Editor))
		for index, contribution := range input.Editor {
			result.Editor[index] = cloneContribution(contribution)
		}
	}
	return result
}

func validateExactPublicationReplay(current, next map[string]Publication) error {
	for extensionID, active := range current {
		candidate, found := next[extensionID]
		if !found {
			continue
		}
		// Same package digest must replay identical declarations; a different
		// package may replace only when identity fields change together.
		if active.Artifact.PackageDigest == candidate.Artifact.PackageDigest &&
			!equalPublications(active, candidate) {
			return fmt.Errorf("%w: immutable package %s changed editor declarations",
				ErrArtifactConflict, extensionID)
		}
	}
	return nil
}

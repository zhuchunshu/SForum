package seoregistry

import (
	"fmt"
	"reflect"
	"sort"
)

type registryState struct {
	revision uint64
	digest   string
	safeMode bool

	publications  map[string]Publication
	contributions map[string]Contribution
	byScope       map[string]map[string][]Contribution
	conflicts     []Conflict
}

// RuntimeInstanceID is deliberately excluded: restarting the same immutable
// package/version may rotate its process id, but cannot reinterpret declarations.
type artifactDeclarationIdentity struct {
	extensionID      string
	extensionVersion string
	packageDigest    string
	impactDigest     string
	versionID        int64
	core             bool
}

func declarationIdentity(artifact Artifact) artifactDeclarationIdentity {
	return artifactDeclarationIdentity{
		extensionID: artifact.ExtensionID, extensionVersion: artifact.ExtensionVersion,
		packageDigest: artifact.PackageDigest, impactDigest: artifact.ImpactDigest,
		versionID: artifact.VersionID, core: artifact.Core,
	}
}

func emptyState() *registryState {
	return &registryState{
		digest: computeGraphDigest(nil, false), publications: map[string]Publication{},
		contributions: map[string]Contribution{}, byScope: map[string]map[string][]Contribution{},
	}
}

func buildState(revision uint64, input []Publication, safeMode bool) (*registryState, error) {
	state := emptyState()
	state.revision = revision
	state.safeMode = safeMode
	publications, err := normalizePublications(filterSafeModeInput(input, safeMode))
	if err != nil {
		return nil, err
	}
	for _, publication := range publications {
		state.publications[publication.Artifact.ExtensionID] = clonePublication(publication)
		for _, declaration := range publication.Contributions {
			if _, duplicate := state.contributions[declaration.ID]; duplicate {
				return nil, fmt.Errorf("%w: duplicate contribution %s", ErrConflict, declaration.ID)
			}
			contribution := Contribution{Declaration: declaration, Artifact: publication.Artifact}
			state.contributions[contribution.ID] = contribution
			if state.byScope[contribution.Scope] == nil {
				state.byScope[contribution.Scope] = make(map[string][]Contribution)
			}
			state.byScope[contribution.Scope][contribution.Kind] = append(
				state.byScope[contribution.Scope][contribution.Kind], contribution,
			)
		}
	}
	for scope, byKind := range state.byScope {
		for kind, contributions := range byKind {
			sort.Slice(contributions, func(i, j int) bool { return contributionBefore(contributions[i], contributions[j]) })
			state.byScope[scope][kind] = contributions
			state.conflicts = append(state.conflicts, contributionConflicts(scope, kind, contributions)...)
		}
	}
	sort.Slice(state.conflicts, func(i, j int) bool {
		left, right := state.conflicts[i], state.conflicts[j]
		if left.Scope != right.Scope {
			return left.Scope < right.Scope
		}
		if left.Kind != right.Kind {
			return kindIndex(left.Kind) < kindIndex(right.Kind)
		}
		return left.Action < right.Action
	})
	state.digest = computeGraphDigest(publications, safeMode)
	return state, nil
}

func equalPublicationDeclarations(left, right Publication) bool {
	return reflect.DeepEqual(left.Contributions, right.Contributions)
}

func validatePublicationHistory(
	history map[artifactDeclarationIdentity]Publication,
	publications map[string]Publication,
) error {
	for extensionID, candidate := range publications {
		sealed, found := history[declarationIdentity(candidate.Artifact)]
		if found && !equalPublicationDeclarations(sealed, candidate) {
			return fmt.Errorf("%w: immutable artifact %s changed its declarations", ErrArtifactConflict, extensionID)
		}
	}
	return nil
}

func recordPublicationHistory(
	history map[artifactDeclarationIdentity]Publication,
	publications map[string]Publication,
) {
	for _, publication := range publications {
		history[declarationIdentity(publication.Artifact)] = clonePublication(publication)
	}
}

func contributionConflicts(scope, kind string, values []Contribution) []Conflict {
	byAction := map[string][]Contribution{}
	for _, contribution := range values {
		if contribution.Action == ActionReplace || (contribution.Action == ActionAdd && scalarKind(kind)) {
			byAction[contribution.Action] = append(byAction[contribution.Action], contribution)
		}
	}
	result := make([]Conflict, 0, 2)
	for _, action := range []string{ActionAdd, ActionReplace} {
		candidates := byAction[action]
		if len(candidates) > 1 {
			result = append(result, Conflict{
				Scope: scope, Kind: kind, Action: action,
				Candidates: cloneContributions(candidates), Winner: candidates[0],
			})
		}
	}
	return result
}

func scalarKind(kind string) bool {
	return kind == KindTitle || kind == KindCanonical || kind == KindRobots
}

func kindIndex(kind string) int {
	for index, candidate := range executionKindOrder {
		if candidate == kind {
			return index
		}
	}
	return len(executionKindOrder)
}

func sortedContributionValues(values map[string]Contribution) []Contribution {
	result := make([]Contribution, 0, len(values))
	for _, contribution := range values {
		result = append(result, contribution)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func snapshotFromState(state *registryState) Snapshot {
	return Snapshot{
		SchemaVersion: SchemaVersion, Revision: state.revision, Digest: state.digest, SafeMode: state.safeMode,
		Publications: sortedPublications(state.publications), Contributions: sortedContributionValues(state.contributions),
		Conflicts: cloneConflicts(state.conflicts),
	}
}

func contributionsForScope(state *registryState, scope string) []Contribution {
	result := make([]Contribution, 0)
	scopes := []string{scope}
	if scope != GlobalScope {
		scopes = append(scopes, GlobalScope)
	}
	for _, candidateScope := range scopes {
		for _, kind := range executionKindOrder {
			result = append(result, state.byScope[candidateScope][kind]...)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := result[i], result[j]
		if left.Kind != right.Kind {
			return kindIndex(left.Kind) < kindIndex(right.Kind)
		}
		if left.Action != right.Action {
			return actionIndex(left.Action) < actionIndex(right.Action)
		}
		return contributionBefore(left, right)
	})
	return result
}

func conflictsForScope(state *registryState, scope string) []Conflict {
	values := contributionsForScope(state, scope)
	byKind := make(map[string][]Contribution)
	for _, contribution := range values {
		byKind[contribution.Kind] = append(byKind[contribution.Kind], contribution)
	}
	result := make([]Conflict, 0)
	for _, kind := range executionKindOrder {
		for _, conflict := range contributionConflicts(scope, kind, byKind[kind]) {
			result = append(result, conflict)
		}
	}
	return result
}

var executionKindOrder = []string{
	KindTitle, KindMeta, KindCanonical, KindRobots, KindHreflang, KindSitemap, KindJSONLD,
}

func actionIndex(action string) int {
	switch action {
	case ActionAdd:
		return 0
	case ActionReplace:
		return 1
	case ActionFilter:
		return 2
	default:
		return 3
	}
}

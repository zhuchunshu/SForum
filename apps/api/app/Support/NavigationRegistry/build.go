package navigationregistry

import (
	"fmt"
	"sort"
	"strings"
)

func emptyState() *registryState {
	return &registryState{
		digest: computeGraphDigest(nil, false, nil), publications: map[string]Publication{},
		navigation: map[string]NavigationContribution{}, regions: map[string]RegionContribution{},
		navigationTargets: map[string]NavigationContribution{}, regionTargets: map[string]RegionContribution{},
		navigationByTarget: map[string][]NavigationContribution{}, regionsByTarget: map[string][]RegionContribution{},
		providerSelections: map[string]ProviderSelection{},
	}
}

func buildState(revision uint64, input []Publication, safeMode bool, selections map[string]ProviderSelection) (*registryState, error) {
	publications, err := normalizePublications(filterSafeModeInput(input, safeMode))
	if err != nil {
		return nil, err
	}
	dependencies, err := resolveDependencies(publications)
	if err != nil {
		return nil, err
	}

	state := emptyState()
	state.revision = revision
	state.safeMode = safeMode
	for _, publication := range publications {
		state.publications[publication.Artifact.ExtensionID] = clonePublication(publication)
	}

	allNavigation := map[string]NavigationContribution{}
	allRegions := map[string]RegionContribution{}
	for _, publication := range publications {
		for _, declaration := range publication.Navigation {
			if _, duplicate := allNavigation[declaration.ID]; duplicate || allRegions[declaration.ID].ID != "" {
				return nil, fmt.Errorf("%w: duplicate contribution %s", ErrConflict, declaration.ID)
			}
			contribution := NavigationContribution{NavigationDeclaration: declaration, Artifact: publication.Artifact}
			allNavigation[contribution.ID] = contribution
			if contribution.Action == ActionAdd {
				state.navigationTargets[contribution.ID] = contribution
			}
		}
		for _, declaration := range publication.Regions {
			if _, duplicate := allRegions[declaration.ID]; duplicate || allNavigation[declaration.ID].ID != "" {
				return nil, fmt.Errorf("%w: duplicate contribution %s", ErrConflict, declaration.ID)
			}
			contribution := RegionContribution{RegionDeclaration: declaration, Artifact: publication.Artifact}
			allRegions[contribution.ID] = contribution
			if contribution.Action == ActionAdd {
				state.regionTargets[contribution.ID] = contribution
			}
		}
	}

	activeNavigation := allTrueNavigation(allNavigation)
	activeRegions := allTrueRegions(allRegions)
	for {
		changed := false
		for _, contribution := range sortedNavigationValues(allNavigation) {
			if !activeNavigation[contribution.ID] || contribution.TargetID == "" {
				continue
			}
			keep, targetErr := validateNavigationTarget(
				contribution, state, activeNavigation, activeRegions, dependencies,
			)
			if targetErr != nil {
				return nil, targetErr
			}
			if !keep {
				activeNavigation[contribution.ID] = false
				changed = true
			}
		}
		for _, contribution := range sortedRegionValues(allRegions) {
			if !activeRegions[contribution.ID] || contribution.TargetID == "" {
				continue
			}
			keep, targetErr := validateRegionTarget(contribution, state, activeRegions, dependencies)
			if targetErr != nil {
				return nil, targetErr
			}
			if !keep {
				activeRegions[contribution.ID] = false
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	if err := validateTargetCycles(state, activeNavigation, activeRegions); err != nil {
		return nil, err
	}
	if err := validateTargetDepth(state, activeNavigation, activeRegions); err != nil {
		return nil, err
	}

	state.navigationTargets = map[string]NavigationContribution{}
	state.regionTargets = map[string]RegionContribution{}
	for _, contribution := range sortedNavigationValues(allNavigation) {
		if !activeNavigation[contribution.ID] {
			continue
		}
		state.navigation[contribution.ID] = contribution
		if contribution.Action == ActionAdd {
			state.navigationTargets[contribution.ID] = contribution
		} else {
			state.navigationByTarget[contribution.TargetID] = append(state.navigationByTarget[contribution.TargetID], contribution)
		}
	}
	for _, contribution := range sortedRegionValues(allRegions) {
		if !activeRegions[contribution.ID] {
			continue
		}
		state.regions[contribution.ID] = contribution
		if contribution.Action == ActionAdd {
			state.regionTargets[contribution.ID] = contribution
		} else {
			state.regionsByTarget[contribution.TargetID] = append(state.regionsByTarget[contribution.TargetID], contribution)
		}
	}
	for targetID := range state.navigationByTarget {
		sort.Slice(state.navigationByTarget[targetID], func(i, j int) bool {
			return navigationProviderBefore(state.navigationByTarget[targetID][i], state.navigationByTarget[targetID][j])
		})
	}
	for targetID := range state.regionsByTarget {
		sort.Slice(state.regionsByTarget[targetID], func(i, j int) bool {
			return regionProviderBefore(state.regionsByTarget[targetID][i], state.regionsByTarget[targetID][j])
		})
	}
	state.providerSelections = retainValidSelections(state, selections)
	state.digest = computeGraphDigest(sortedPublications(state.publications), safeMode, sortedProviderSelections(state.providerSelections))
	return state, nil
}

func filterSafeModeInput(input []Publication, safeMode bool) []Publication {
	if !safeMode {
		return input
	}
	result := make([]Publication, 0, len(input))
	for _, publication := range input {
		if validCoreArtifactSeal(publication.Artifact) {
			result = append(result, publication)
		}
	}
	return result
}

func validateNavigationTarget(
	contribution NavigationContribution,
	state *registryState,
	activeNavigation, activeRegions map[string]bool,
	dependencies dependencyResolution,
) (bool, error) {
	target, navigationDefined := state.navigationTargets[contribution.TargetID]
	region, regionDefined := state.regionTargets[contribution.TargetID]
	navigationTarget := navigationDefined && activeNavigation[target.ID]
	regionTarget := regionDefined && activeRegions[region.ID]
	if contribution.Action != ActionAdd && navigationDefined && !navigationTarget {
		return false, nil
	}
	if contribution.Action != ActionAdd && !navigationDefined {
		return optionalOrTargetError(contribution.Artifact, contribution.TargetID, dependencies)
	}
	if contribution.Action == ActionAdd && (navigationDefined && !navigationTarget || regionDefined && !regionTarget) {
		return false, nil
	}
	if contribution.Action == ActionAdd && !navigationTarget && !regionTarget {
		return optionalOrTargetError(contribution.Artifact, contribution.TargetID, dependencies)
	}
	owner := target.Artifact
	if regionTarget {
		owner = region.Artifact
	}
	if keep, err := authorizeTarget(contribution.Artifact, owner, contribution.TargetID, dependencies); !keep || err != nil {
		return keep, err
	}
	if contribution.Action != ActionAdd && contribution.Kind != target.Kind {
		return false, fmt.Errorf("%w: navigation %s kind does not match target %s", ErrConflict, contribution.ID, target.ID)
	}
	return true, nil
}

func validateRegionTarget(
	contribution RegionContribution,
	state *registryState,
	activeRegions map[string]bool,
	dependencies dependencyResolution,
) (bool, error) {
	target, defined := state.regionTargets[contribution.TargetID]
	if defined && !activeRegions[target.ID] {
		return false, nil
	}
	if !defined {
		return optionalOrTargetError(contribution.Artifact, contribution.TargetID, dependencies)
	}
	if keep, err := authorizeTarget(contribution.Artifact, target.Artifact, contribution.TargetID, dependencies); !keep || err != nil {
		return keep, err
	}
	if contribution.Action != ActionAdd && contribution.Kind != target.Kind {
		return false, fmt.Errorf("%w: region %s kind does not match target %s", ErrConflict, contribution.ID, target.ID)
	}
	return true, nil
}

func authorizeTarget(consumer, owner Artifact, targetID string, dependencies dependencyResolution) (bool, error) {
	if owner.ExtensionID == consumer.ExtensionID || owner.Core {
		return true, nil
	}
	if dependencies.authorized[consumer.ExtensionID][owner.ExtensionID] {
		return true, nil
	}
	if optionalDependencyMatches(dependencies.optionalByConsumer[consumer.ExtensionID], owner.ExtensionID, targetID) {
		return false, nil
	}
	if dependencies.optionalCapabilityOwners[consumer.ExtensionID][owner.ExtensionID] {
		return false, nil
	}
	return false, fmt.Errorf("%w: %s does not declare target owner %s", ErrDependency, consumer.ExtensionID, owner.ExtensionID)
}

func optionalOrTargetError(
	consumer Artifact,
	targetID string,
	dependencies dependencyResolution,
) (bool, error) {
	if optionalDependencyMatches(dependencies.optionalByConsumer[consumer.ExtensionID], "", targetID) ||
		missingOptionalCapabilityTarget(consumer, targetID, dependencies) {
		return false, nil
	}
	return false, fmt.Errorf("%w: target %s is unavailable", ErrConflict, targetID)
}

// Capability dependencies deliberately do not bind an extension id. When an
// optional provider is absent, its target namespace therefore cannot be
// recovered from the active graph. Only an unknown external namespace may be
// omitted; Core and consumer-owned missing targets remain hard declaration
// errors, and known owners still pass through authorizeTarget's exact check.
func missingOptionalCapabilityTarget(
	consumer Artifact,
	targetID string,
	dependencies dependencyResolution,
) bool {
	if !dependencies.unresolvedOptionalCapability[consumer.ExtensionID] ||
		strings.HasPrefix(targetID, "core.") || strings.HasPrefix(targetID, consumer.ExtensionID+".") {
		return false
	}
	for extensionID := range dependencies.activeExtensionIDs {
		if strings.HasPrefix(targetID, extensionID+".") {
			return false
		}
	}
	return true
}

// optionalDependencyMatches soft-drops only when the consumer declared an
// optional edge that actually covers the missing/incompatible owner. Known
// owners require an exact extension id match so a short optional prefix cannot
// fail-open across a different plugin. Missing targets use the longest matching
// optional id prefix, matching hookDependencyForTarget.
func optionalDependencyMatches(dependencies []Dependency, ownerID, targetID string) bool {
	if ownerID != "" {
		for _, dependency := range dependencies {
			if dependency.ExtensionID == ownerID {
				return true
			}
		}
		return false
	}
	best := ""
	for _, dependency := range dependencies {
		if dependency.ExtensionID == "" || !strings.HasPrefix(targetID, dependency.ExtensionID+".") {
			continue
		}
		if len(dependency.ExtensionID) > len(best) {
			best = dependency.ExtensionID
		}
	}
	return best != ""
}

func validateTargetCycles(state *registryState, activeNavigation, activeRegions map[string]bool) error {
	edges := map[string]string{}
	for id, target := range state.navigationTargets {
		if activeNavigation[id] && target.TargetID != "" {
			edges[id] = target.TargetID
		}
	}
	for id, target := range state.regionTargets {
		if activeRegions[id] && target.TargetID != "" {
			edges[id] = target.TargetID
		}
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) bool
	visit = func(id string) bool {
		if visiting[id] {
			return true
		}
		if visited[id] {
			return false
		}
		visiting[id] = true
		if parent := edges[id]; parent != "" && visit(parent) {
			return true
		}
		delete(visiting, id)
		visited[id] = true
		return false
	}
	for id := range edges {
		if visit(id) {
			return fmt.Errorf("%w: target cycle includes %s", ErrDependency, id)
		}
	}
	return nil
}

func validateTargetDepth(state *registryState, activeNavigation, activeRegions map[string]bool) error {
	parents := map[string]string{}
	for id, target := range state.navigationTargets {
		if activeNavigation[id] {
			parents[id] = target.TargetID
		}
	}
	for id, target := range state.regionTargets {
		if activeRegions[id] {
			parents[id] = target.TargetID
		}
	}
	for id := range parents {
		depth := 0
		for current := id; current != ""; current = parents[current] {
			depth++
			if depth > maxTargetDepth {
				return fmt.Errorf("%w: target depth exceeds %d at %s", ErrLimitExceeded, maxTargetDepth, id)
			}
		}
	}
	return nil
}

func navigationProviderBefore(left, right NavigationContribution) bool {
	if left.Priority != right.Priority {
		return left.Priority > right.Priority
	}
	if left.Order != right.Order {
		return left.Order < right.Order
	}
	if left.Artifact.ExtensionID != right.Artifact.ExtensionID {
		return left.Artifact.ExtensionID < right.Artifact.ExtensionID
	}
	return left.ID < right.ID
}

func regionProviderBefore(left, right RegionContribution) bool {
	if left.Priority != right.Priority {
		return left.Priority > right.Priority
	}
	if left.Order != right.Order {
		return left.Order < right.Order
	}
	if left.Artifact.ExtensionID != right.Artifact.ExtensionID {
		return left.Artifact.ExtensionID < right.Artifact.ExtensionID
	}
	return left.ID < right.ID
}

func allTrueNavigation(values map[string]NavigationContribution) map[string]bool {
	result := make(map[string]bool, len(values))
	for id := range values {
		result[id] = true
	}
	return result
}

func allTrueRegions(values map[string]RegionContribution) map[string]bool {
	result := make(map[string]bool, len(values))
	for id := range values {
		result[id] = true
	}
	return result
}

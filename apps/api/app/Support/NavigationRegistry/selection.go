package navigationregistry

import (
	"sort"
	"strings"
)

func (r *Registry) SelectProvider(request SelectProviderRequest) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	family := strings.ToLower(strings.TrimSpace(request.Family))
	targetID := strings.ToLower(strings.TrimSpace(request.TargetID))
	refs, _, err := normalizeProviderRefs([]ProviderRef{request.Provider})
	if err != nil || !validProviderFamily(family) || !idPattern.MatchString(targetID) {
		return r.load().revision, ErrProviderSelection
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	if current.revision != request.ExpectedRevision {
		return current.revision, ErrRevisionConflict
	}
	selection, err := exactProviderSelection(current, family, targetID, refs[0])
	if err != nil {
		return current.revision, err
	}
	selections := cloneProviderSelections(current.providerSelections)
	key := providerSelectionKey(family, targetID)
	if existing, found := selections[key]; found && existing == selection {
		return current.revision, nil
	}
	selections[key] = selection
	next, err := buildState(current.revision+1, publicationValues(current.publications), current.safeMode, selections)
	if err != nil {
		return current.revision, err
	}
	if _, retained := next.providerSelections[key]; !retained {
		return current.revision, ErrProviderSelection
	}
	r.state.Store(next)
	return next.revision, nil
}

func (r *Registry) ResetProvider(request ResetProviderRequest) (uint64, bool, error) {
	if r == nil {
		return 0, false, ErrInvalid
	}
	family := strings.ToLower(strings.TrimSpace(request.Family))
	targetID := strings.ToLower(strings.TrimSpace(request.TargetID))
	if !validProviderFamily(family) || !idPattern.MatchString(targetID) {
		return r.load().revision, false, ErrProviderSelection
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	if current.revision != request.ExpectedRevision {
		return current.revision, false, ErrRevisionConflict
	}
	selections := cloneProviderSelections(current.providerSelections)
	key := providerSelectionKey(family, targetID)
	if _, found := selections[key]; !found {
		return current.revision, false, nil
	}
	delete(selections, key)
	next, err := buildState(current.revision+1, publicationValues(current.publications), current.safeMode, selections)
	if err != nil {
		return current.revision, false, err
	}
	r.state.Store(next)
	return next.revision, true, nil
}

func exactProviderSelection(state *registryState, family, targetID string, ref ProviderRef) (ProviderSelection, error) {
	switch family {
	case ProviderFamilyNavigation:
		target, found := state.navigationTargets[targetID]
		if !found {
			return ProviderSelection{}, ErrProviderSelection
		}
		for _, candidate := range state.navigationByTarget[targetID] {
			if candidate.Action == ActionReplace && candidate.ID == ref.ContributionID && candidate.Artifact == ref.Artifact {
				return ProviderSelection{
					Family: family, TargetID: targetID, TargetContractVersion: target.ContractVersion,
					ProviderID: candidate.ID, ProviderContractVersion: candidate.ContractVersion, Provider: ref,
				}, nil
			}
		}
	case ProviderFamilyRegion:
		target, found := state.regionTargets[targetID]
		if !found {
			return ProviderSelection{}, ErrProviderSelection
		}
		for _, candidate := range state.regionsByTarget[targetID] {
			if candidate.Action == ActionReplace && candidate.ID == ref.ContributionID && candidate.Artifact == ref.Artifact {
				return ProviderSelection{
					Family: family, TargetID: targetID, TargetContractVersion: target.ContractVersion,
					ProviderID: candidate.ID, ProviderContractVersion: candidate.ContractVersion, Provider: ref,
				}, nil
			}
		}
	}
	return ProviderSelection{}, ErrProviderSelection
}

func retainValidSelections(state *registryState, input map[string]ProviderSelection) map[string]ProviderSelection {
	result := make(map[string]ProviderSelection, len(input))
	for key, selection := range input {
		current, err := exactProviderSelection(state, selection.Family, selection.TargetID, selection.Provider)
		if err == nil && current == selection {
			result[key] = selection
		}
	}
	return result
}

func selectedProvider(state *registryState, family, targetID string) (ProviderSelection, bool) {
	selection, found := state.providerSelections[providerSelectionKey(family, targetID)]
	return selection, found
}

func selectedNavigationCandidate(selection ProviderSelection, candidates []NavigationContribution) (NavigationContribution, bool) {
	for _, candidate := range candidates {
		if candidate.ID == selection.ProviderID && candidate.ContractVersion == selection.ProviderContractVersion &&
			candidate.Artifact == selection.Provider.Artifact {
			return candidate, true
		}
	}
	return NavigationContribution{}, false
}

func selectedRegionCandidate(selection ProviderSelection, candidates []RegionContribution) (RegionContribution, bool) {
	for _, candidate := range candidates {
		if candidate.ID == selection.ProviderID && candidate.ContractVersion == selection.ProviderContractVersion &&
			candidate.Artifact == selection.Provider.Artifact {
			return candidate, true
		}
	}
	return RegionContribution{}, false
}

func cloneProviderSelections(input map[string]ProviderSelection) map[string]ProviderSelection {
	result := make(map[string]ProviderSelection, len(input))
	for key, selection := range input {
		result[key] = selection
	}
	return result
}

func sortedProviderSelections(input map[string]ProviderSelection) []ProviderSelection {
	result := make([]ProviderSelection, 0, len(input))
	for _, selection := range input {
		result = append(result, selection)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := result[i], result[j]
		if left.Family != right.Family {
			return left.Family < right.Family
		}
		return left.TargetID < right.TargetID
	})
	return result
}

func validProviderFamily(value string) bool {
	return value == ProviderFamilyNavigation || value == ProviderFamilyRegion
}

func providerSelectionKey(family, targetID string) string {
	return family + "\x00" + targetID
}

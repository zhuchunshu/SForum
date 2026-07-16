package navigationregistry

import (
	"slices"
	"sort"
)

func clonePublication(value Publication) Publication {
	// 克隆不能改变规范化 publication 的 nil/非 nil 形态或后续等价判断。
	value.Dependencies = slices.Clone(value.Dependencies)
	value.Navigation = slices.Clone(value.Navigation)
	for index := range value.Navigation {
		value.Navigation[index].Labels = cloneStringMap(value.Navigation[index].Labels)
	}
	value.Regions = slices.Clone(value.Regions)
	for index := range value.Regions {
		value.Regions[index].Labels = cloneStringMap(value.Regions[index].Labels)
	}
	return value
}

func clonePublicationMap(values map[string]Publication) map[string]Publication {
	result := make(map[string]Publication, len(values))
	for id, publication := range values {
		result[id] = clonePublication(publication)
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
	sort.Slice(result, func(i, j int) bool {
		return artifactBefore(result[i].Artifact, result[j].Artifact)
	})
	return result
}

func sortedNavigationValues(values map[string]NavigationContribution) []NavigationContribution {
	result := make([]NavigationContribution, 0, len(values))
	for _, value := range values {
		result = append(result, cloneNavigationContribution(value))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func sortedRegionValues(values map[string]RegionContribution) []RegionContribution {
	result := make([]RegionContribution, 0, len(values))
	for _, value := range values {
		result = append(result, cloneRegionContribution(value))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func snapshotFromState(state *registryState) Snapshot {
	result := Snapshot{
		SchemaVersion: SchemaVersion,
		Revision:      state.revision,
		Digest:        state.digest,
		SafeMode:      state.safeMode,
		Publications:  sortedPublications(state.publications),
	}
	result.Navigation = sortedNavigationValues(state.navigation)
	result.Regions = sortedRegionValues(state.regions)
	for _, target := range sortedNavigationTargetValues(state.navigationTargets) {
		candidates := navigationContributionsByAction(state.navigationByTarget[target.ID], ActionReplace)
		if len(candidates) > 1 {
			result.NavigationConflicts = append(result.NavigationConflicts, NavigationProviderConflict{
				TargetID: target.ID, Candidates: candidates, Winner: candidates[0],
			})
		}
	}
	for _, target := range sortedRegionTargetValues(state.regionTargets) {
		candidates := regionContributionsByAction(state.regionsByTarget[target.ID], ActionReplace)
		if len(candidates) > 1 {
			result.RegionConflicts = append(result.RegionConflicts, RegionProviderConflict{
				TargetID: target.ID, Candidates: candidates, Winner: candidates[0],
			})
		}
	}
	result.ProviderSelections = sortedProviderSelections(state.providerSelections)
	return result
}

func navigationContributionsByAction(values []NavigationContribution, action string) []NavigationContribution {
	result := make([]NavigationContribution, 0)
	for _, value := range values {
		if value.Action == action {
			result = append(result, value)
		}
	}
	return result
}

func regionContributionsByAction(values []RegionContribution, action string) []RegionContribution {
	result := make([]RegionContribution, 0)
	for _, value := range values {
		if value.Action == action {
			result = append(result, value)
		}
	}
	return result
}

func cloneSnapshot(value Snapshot) Snapshot {
	value.Publications = slices.Clone(value.Publications)
	for index := range value.Publications {
		value.Publications[index] = clonePublication(value.Publications[index])
	}
	value.Navigation = append([]NavigationContribution(nil), value.Navigation...)
	value.Regions = append([]RegionContribution(nil), value.Regions...)
	value.NavigationConflicts = append([]NavigationProviderConflict(nil), value.NavigationConflicts...)
	for index := range value.NavigationConflicts {
		value.NavigationConflicts[index].Winner = cloneNavigationContribution(value.NavigationConflicts[index].Winner)
		value.NavigationConflicts[index].Candidates = append(
			[]NavigationContribution(nil), value.NavigationConflicts[index].Candidates...,
		)
		for candidate := range value.NavigationConflicts[index].Candidates {
			value.NavigationConflicts[index].Candidates[candidate] = cloneNavigationContribution(value.NavigationConflicts[index].Candidates[candidate])
		}
	}
	value.RegionConflicts = append([]RegionProviderConflict(nil), value.RegionConflicts...)
	for index := range value.RegionConflicts {
		value.RegionConflicts[index].Winner = cloneRegionContribution(value.RegionConflicts[index].Winner)
		value.RegionConflicts[index].Candidates = append(
			[]RegionContribution(nil), value.RegionConflicts[index].Candidates...,
		)
		for candidate := range value.RegionConflicts[index].Candidates {
			value.RegionConflicts[index].Candidates[candidate] = cloneRegionContribution(value.RegionConflicts[index].Candidates[candidate])
		}
	}
	value.ProviderSelections = append([]ProviderSelection(nil), value.ProviderSelections...)
	return value
}

func cloneNavigationResolution(value NavigationResolution) NavigationResolution {
	value.Targets = append([]NavigationTargetPlan(nil), value.Targets...)
	for index := range value.Targets {
		plan := &value.Targets[index]
		plan.Target = cloneNavigationContribution(plan.Target)
		plan.Provider = cloneNavigationContribution(plan.Provider)
		plan.ReplaceCandidates = append([]NavigationContribution(nil), plan.ReplaceCandidates...)
		plan.Before = append([]NavigationContribution(nil), plan.Before...)
		plan.After = append([]NavigationContribution(nil), plan.After...)
		plan.Wrap = append([]NavigationContribution(nil), plan.Wrap...)
		plan.Filters = append([]NavigationContribution(nil), plan.Filters...)
		plan.hides = append([]NavigationContribution(nil), plan.hides...)
		for candidate := range plan.ReplaceCandidates {
			plan.ReplaceCandidates[candidate] = cloneNavigationContribution(plan.ReplaceCandidates[candidate])
		}
		for candidate := range plan.Before {
			plan.Before[candidate] = cloneNavigationContribution(plan.Before[candidate])
		}
		for candidate := range plan.After {
			plan.After[candidate] = cloneNavigationContribution(plan.After[candidate])
		}
		for candidate := range plan.Wrap {
			plan.Wrap[candidate] = cloneNavigationContribution(plan.Wrap[candidate])
		}
		for candidate := range plan.Filters {
			plan.Filters[candidate] = cloneNavigationContribution(plan.Filters[candidate])
		}
		for candidate := range plan.hides {
			plan.hides[candidate] = cloneNavigationContribution(plan.hides[candidate])
		}
	}
	return value
}

func cloneRegionResolution(value RegionResolution) RegionResolution {
	value.Targets = append([]RegionTargetPlan(nil), value.Targets...)
	for index := range value.Targets {
		plan := &value.Targets[index]
		plan.Target = cloneRegionContribution(plan.Target)
		plan.Provider = cloneRegionContribution(plan.Provider)
		plan.ReplaceCandidates = append([]RegionContribution(nil), plan.ReplaceCandidates...)
		plan.Before = append([]RegionContribution(nil), plan.Before...)
		plan.After = append([]RegionContribution(nil), plan.After...)
		plan.Wrap = append([]RegionContribution(nil), plan.Wrap...)
		plan.Filters = append([]RegionContribution(nil), plan.Filters...)
		plan.hides = append([]RegionContribution(nil), plan.hides...)
		for candidate := range plan.ReplaceCandidates {
			plan.ReplaceCandidates[candidate] = cloneRegionContribution(plan.ReplaceCandidates[candidate])
		}
		for candidate := range plan.Before {
			plan.Before[candidate] = cloneRegionContribution(plan.Before[candidate])
		}
		for candidate := range plan.After {
			plan.After[candidate] = cloneRegionContribution(plan.After[candidate])
		}
		for candidate := range plan.Wrap {
			plan.Wrap[candidate] = cloneRegionContribution(plan.Wrap[candidate])
		}
		for candidate := range plan.Filters {
			plan.Filters[candidate] = cloneRegionContribution(plan.Filters[candidate])
		}
		for candidate := range plan.hides {
			plan.hides[candidate] = cloneRegionContribution(plan.hides[candidate])
		}
	}
	return value
}

func cloneNavigationContribution(value NavigationContribution) NavigationContribution {
	value.Labels = cloneStringMap(value.Labels)
	return value
}

func cloneRegionContribution(value RegionContribution) RegionContribution {
	value.Labels = cloneStringMap(value.Labels)
	return value
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

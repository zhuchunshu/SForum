package navigationregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
)

type visibilityState struct {
	permissions map[string]bool
	owned       map[string]bool
	hidden      map[string]bool
	disabled    map[string]bool
	canonical   VisibilityInput
}

func (r *Registry) ResolveNavigation(request NavigationResolveRequest) (NavigationResolution, error) {
	return r.resolveNavigation(request, false)
}

func (r *Registry) resolveNavigation(request NavigationResolveRequest, deferRuntimeHides bool) (NavigationResolution, error) {
	if r == nil {
		return NavigationResolution{}, ErrInvalid
	}
	kinds, err := normalizeKinds(request.Kinds, validNavigationKind)
	if err != nil {
		return NavigationResolution{}, err
	}
	visibility, err := normalizeVisibility(request.Visibility)
	if err != nil {
		return NavigationResolution{}, err
	}
	locale, err := normalizeLocale(request.Locale)
	if err != nil {
		return NavigationResolution{}, err
	}
	state := r.load()
	result := NavigationResolution{
		SchemaVersion: SchemaVersion, Revision: state.revision, Digest: state.digest, SafeMode: state.safeMode, Locale: locale,
		CacheKey: resolutionCacheKey(state, "navigation", locale, kinds, visibility.canonical),
	}
	visibleMemo := map[string]bool{}
	checking := map[string]bool{}
	for _, target := range sortedNavigationTargetValues(state.navigationTargets) {
		if len(kinds) > 0 && !stringSliceContains(kinds, target.Kind) {
			continue
		}
		if !navigationTargetVisible(state, target.ID, visibility, visibleMemo, checking, deferRuntimeHides) {
			continue
		}
		plan := NavigationTargetPlan{Target: target, ParentID: target.TargetID, Provider: target}
		hadReplace := false
		for _, contribution := range state.navigationByTarget[target.ID] {
			if contribution.Action == ActionHide {
				if deferRuntimeHides && !contribution.Artifact.Core && navigationContributionVisible(contribution, visibility, true) {
					plan.hides = append(plan.hides, contribution)
				}
				continue
			}
			if contribution.Action == ActionReplace {
				hadReplace = true
			}
			if !navigationContributionVisible(contribution, visibility, true) {
				continue
			}
			switch contribution.Action {
			case ActionReplace:
				plan.ReplaceCandidates = append(plan.ReplaceCandidates, contribution)
			case ActionBefore:
				plan.Before = append(plan.Before, contribution)
			case ActionAfter:
				plan.After = append(plan.After, contribution)
			case ActionWrap:
				plan.Wrap = append(plan.Wrap, contribution)
			case ActionFilter:
				plan.Filters = append(plan.Filters, contribution)
			}
		}
		if selection, selected := selectedProvider(state, ProviderFamilyNavigation, target.ID); selected {
			plan.SelectionConfigured = true
			if candidate, visible := selectedNavigationCandidate(selection, plan.ReplaceCandidates); visible {
				plan.Provider = candidate
				plan.SelectedProvider = true
			} else {
				plan.UsingFallback = hadReplace
			}
		} else if len(plan.ReplaceCandidates) > 0 {
			plan.Provider = plan.ReplaceCandidates[0]
		} else {
			plan.UsingFallback = hadReplace
		}
		result.Targets = append(result.Targets, plan)
	}
	localizeNavigationResolution(&result, locale)
	return cloneNavigationResolution(result), nil
}

func (r *Registry) ResolveRegions(request RegionResolveRequest) (RegionResolution, error) {
	return r.resolveRegions(request, false)
}

func (r *Registry) resolveRegions(request RegionResolveRequest, deferRuntimeHides bool) (RegionResolution, error) {
	if r == nil {
		return RegionResolution{}, ErrInvalid
	}
	kinds, err := normalizeKinds(request.Kinds, validRegionKind)
	if err != nil {
		return RegionResolution{}, err
	}
	visibility, err := normalizeVisibility(request.Visibility)
	if err != nil {
		return RegionResolution{}, err
	}
	locale, err := normalizeLocale(request.Locale)
	if err != nil {
		return RegionResolution{}, err
	}
	state := r.load()
	result := RegionResolution{
		SchemaVersion: SchemaVersion, Revision: state.revision, Digest: state.digest, SafeMode: state.safeMode, Locale: locale,
		CacheKey: resolutionCacheKey(state, "regions", locale, kinds, visibility.canonical),
	}
	visibleMemo := map[string]bool{}
	checking := map[string]bool{}
	for _, target := range sortedRegionTargetValues(state.regionTargets) {
		if len(kinds) > 0 && !stringSliceContains(kinds, target.Kind) {
			continue
		}
		if !regionTargetVisible(state, target.ID, visibility, visibleMemo, checking, deferRuntimeHides) {
			continue
		}
		plan := RegionTargetPlan{Target: target, ParentID: target.TargetID, Provider: target}
		hadReplace := false
		for _, contribution := range state.regionsByTarget[target.ID] {
			if contribution.Action == ActionHide {
				if deferRuntimeHides && !contribution.Artifact.Core && regionContributionVisible(contribution, visibility, true) {
					plan.hides = append(plan.hides, contribution)
				}
				continue
			}
			if contribution.Action == ActionReplace {
				hadReplace = true
			}
			if !regionContributionVisible(contribution, visibility, true) {
				continue
			}
			switch contribution.Action {
			case ActionReplace:
				plan.ReplaceCandidates = append(plan.ReplaceCandidates, contribution)
			case ActionBefore:
				plan.Before = append(plan.Before, contribution)
			case ActionAfter:
				plan.After = append(plan.After, contribution)
			case ActionWrap:
				plan.Wrap = append(plan.Wrap, contribution)
			case ActionFilter:
				plan.Filters = append(plan.Filters, contribution)
			}
		}
		if selection, selected := selectedProvider(state, ProviderFamilyRegion, target.ID); selected {
			plan.SelectionConfigured = true
			if candidate, visible := selectedRegionCandidate(selection, plan.ReplaceCandidates); visible {
				plan.Provider = candidate
				plan.SelectedProvider = true
			} else {
				plan.UsingFallback = hadReplace
			}
		} else if len(plan.ReplaceCandidates) > 0 {
			plan.Provider = plan.ReplaceCandidates[0]
		} else {
			plan.UsingFallback = hadReplace
		}
		result.Targets = append(result.Targets, plan)
	}
	localizeRegionResolution(&result, locale)
	return cloneRegionResolution(result), nil
}

func localizeNavigationResolution(result *NavigationResolution, locale string) {
	for index := range result.Targets {
		plan := &result.Targets[index]
		plan.Target.Label = localizedLabel(plan.Target.Label, plan.Target.Labels, locale)
		plan.Provider.Label = localizedLabel(plan.Provider.Label, plan.Provider.Labels, locale)
		for _, values := range [][]NavigationContribution{plan.ReplaceCandidates, plan.Before, plan.After, plan.Wrap, plan.Filters} {
			for candidate := range values {
				values[candidate].Label = localizedLabel(values[candidate].Label, values[candidate].Labels, locale)
			}
		}
	}
}

func localizeRegionResolution(result *RegionResolution, locale string) {
	for index := range result.Targets {
		plan := &result.Targets[index]
		plan.Target.Label = localizedLabel(plan.Target.Label, plan.Target.Labels, locale)
		plan.Provider.Label = localizedLabel(plan.Provider.Label, plan.Provider.Labels, locale)
		for _, values := range [][]RegionContribution{plan.ReplaceCandidates, plan.Before, plan.After, plan.Wrap, plan.Filters} {
			for candidate := range values {
				values[candidate].Label = localizedLabel(values[candidate].Label, values[candidate].Labels, locale)
			}
		}
	}
}

func navigationTargetVisible(
	state *registryState,
	targetID string,
	visibility visibilityState,
	memo, checking map[string]bool,
	deferRuntimeHides bool,
) bool {
	if value, found := memo[targetID]; found {
		return value
	}
	if checking[targetID] {
		return false
	}
	checking[targetID] = true
	target, found := state.navigationTargets[targetID]
	visible := found && navigationContributionVisible(target, visibility, true)
	if visible {
		for _, contribution := range state.navigationByTarget[targetID] {
			if contribution.Action == ActionHide && (!deferRuntimeHides || contribution.Artifact.Core) &&
				navigationContributionVisible(contribution, visibility, true) {
				visible = false
				break
			}
		}
	}
	if visible && target.TargetID != "" {
		if _, parentIsNavigation := state.navigationTargets[target.TargetID]; parentIsNavigation {
			visible = navigationTargetVisible(state, target.TargetID, visibility, memo, checking, deferRuntimeHides)
		} else {
			visible = regionTargetVisible(state, target.TargetID, visibility, map[string]bool{}, map[string]bool{}, deferRuntimeHides)
		}
	}
	delete(checking, targetID)
	memo[targetID] = visible
	return visible
}

func regionTargetVisible(
	state *registryState,
	targetID string,
	visibility visibilityState,
	memo, checking map[string]bool,
	deferRuntimeHides bool,
) bool {
	if value, found := memo[targetID]; found {
		return value
	}
	if checking[targetID] {
		return false
	}
	checking[targetID] = true
	target, found := state.regionTargets[targetID]
	visible := found && regionContributionVisible(target, visibility, true)
	if visible {
		for _, contribution := range state.regionsByTarget[targetID] {
			if contribution.Action == ActionHide && (!deferRuntimeHides || contribution.Artifact.Core) &&
				regionContributionVisible(contribution, visibility, true) {
				visible = false
				break
			}
		}
	}
	if visible && target.TargetID != "" {
		visible = regionTargetVisible(state, target.TargetID, visibility, memo, checking, deferRuntimeHides)
	}
	delete(checking, targetID)
	memo[targetID] = visible
	return visible
}

func navigationContributionVisible(contribution NavigationContribution, visibility visibilityState, provider bool) bool {
	if visibility.hidden[contribution.ID] || provider && visibility.disabled[providerRefKey(contribution.ID, contribution.Artifact)] {
		return false
	}
	credentialAllowed := contribution.Permission == "" && contribution.OwnerResource == ""
	credentialAllowed = credentialAllowed || contribution.Permission != "" && visibility.permissions[contribution.Permission]
	credentialAllowed = credentialAllowed || contribution.OwnerResource != "" && visibility.owned[contribution.OwnerResource]
	return actorVisibilityAllows(contribution.Visibility, visibility.canonical.Authenticated) && credentialAllowed
}

func regionContributionVisible(contribution RegionContribution, visibility visibilityState, provider bool) bool {
	return !visibility.hidden[contribution.ID] &&
		(!provider || !visibility.disabled[providerRefKey(contribution.ID, contribution.Artifact)]) &&
		actorVisibilityAllows(contribution.Visibility, visibility.canonical.Authenticated) &&
		(contribution.Permission == "" || visibility.permissions[contribution.Permission])
}

func normalizeVisibility(input VisibilityInput) (visibilityState, error) {
	permissions, err := normalizeIDList(input.Permissions)
	if err != nil {
		return visibilityState{}, err
	}
	owned, err := normalizeIDList(input.OwnedResources)
	if err != nil {
		return visibilityState{}, err
	}
	hidden, err := normalizeIDList(input.HiddenIDs)
	if err != nil {
		return visibilityState{}, err
	}
	disabled, disabledSet, err := normalizeProviderRefs(input.DisabledProviders)
	if err != nil {
		return visibilityState{}, err
	}
	return visibilityState{
		permissions: sliceSet(permissions), owned: sliceSet(owned), hidden: sliceSet(hidden), disabled: disabledSet,
		canonical: VisibilityInput{Authenticated: input.Authenticated, Permissions: permissions, OwnedResources: owned, HiddenIDs: hidden, DisabledProviders: disabled},
	}, nil
}

func actorVisibilityAllows(policy string, authenticated bool) bool {
	switch policy {
	case VisibilityAnonymous:
		return !authenticated
	case VisibilityAuthenticated:
		return authenticated
	default:
		return true
	}
}

func normalizeProviderRefs(input []ProviderRef) ([]ProviderRef, map[string]bool, error) {
	if len(input) > maxContributions {
		return nil, nil, ErrInvalid
	}
	result := make([]ProviderRef, 0, len(input))
	seen := map[string]bool{}
	for _, raw := range input {
		ref := raw
		ref.ContributionID = strings.ToLower(strings.TrimSpace(ref.ContributionID))
		artifact, err := normalizeArtifact(ref.Artifact)
		if err != nil || !idPattern.MatchString(ref.ContributionID) ||
			!strings.HasPrefix(ref.ContributionID, artifact.ExtensionID+".") {
			return nil, nil, ErrInvalid
		}
		ref.Artifact = artifact
		key := providerRefKey(ref.ContributionID, ref.Artifact)
		if !seen[key] {
			seen[key] = true
			result = append(result, ref)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return providerRefKey(result[i].ContributionID, result[i].Artifact) <
			providerRefKey(result[j].ContributionID, result[j].Artifact)
	})
	return result, seen, nil
}

func normalizeKinds(input []string, valid func(string) bool) ([]string, error) {
	if len(input) > maxKindFilters {
		return nil, ErrInvalid
	}
	result := make([]string, 0, len(input))
	seen := map[string]bool{}
	for _, value := range input {
		value = strings.ToLower(strings.TrimSpace(value))
		if !valid(value) {
			return nil, ErrInvalid
		}
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result, nil
}

func normalizeIDList(input []string) ([]string, error) {
	if len(input) > maxContributions {
		return nil, ErrInvalid
	}
	result := make([]string, 0, len(input))
	seen := map[string]bool{}
	for _, value := range input {
		value = strings.ToLower(strings.TrimSpace(value))
		if !idPattern.MatchString(value) {
			return nil, ErrInvalid
		}
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result, nil
}

func resolutionCacheKey(state *registryState, family, locale string, kinds []string, visibility VisibilityInput) string {
	var value strings.Builder
	value.WriteString(SchemaVersion)
	value.WriteByte(0)
	value.WriteString(strconv.FormatUint(state.revision, 10))
	value.WriteByte(0)
	value.WriteString(state.digest)
	value.WriteByte(0)
	value.WriteString(family)
	value.WriteByte(0)
	value.WriteString(locale)
	value.WriteByte(0)
	value.WriteString(strconv.FormatBool(visibility.Authenticated))
	for _, list := range [][]string{kinds, visibility.Permissions, visibility.HiddenIDs} {
		value.WriteByte(0)
		value.WriteString(strings.Join(list, "\x1f"))
	}
	value.WriteByte(0)
	for _, provider := range visibility.DisabledProviders {
		value.WriteString(providerRefKey(provider.ContributionID, provider.Artifact))
		value.WriteByte(0x1f)
	}
	sum := sha256.Sum256([]byte(value.String()))
	return hex.EncodeToString(sum[:])
}

func providerRefKey(contributionID string, artifact Artifact) string {
	return contributionID + "\x00" + artifact.ExtensionID + "\x00" + artifact.ExtensionVersion + "\x00" +
		artifact.PackageDigest + "\x00" + artifact.ImpactDigest + "\x00" + strconv.FormatInt(artifact.VersionID, 10) + "\x00" +
		artifact.RuntimeInstanceID + "\x00" + strconv.FormatBool(artifact.Core)
}

func sliceSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func stringSliceContains(values []string, value string) bool {
	index := sort.SearchStrings(values, value)
	return index < len(values) && values[index] == value
}

func sortedNavigationTargetValues(values map[string]NavigationContribution) []NavigationContribution {
	result := make([]NavigationContribution, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := result[i], result[j]
		if navigationKindRank(left.Kind) != navigationKindRank(right.Kind) {
			return navigationKindRank(left.Kind) < navigationKindRank(right.Kind)
		}
		if left.TargetID != right.TargetID {
			return left.TargetID < right.TargetID
		}
		if left.Order != right.Order {
			return left.Order < right.Order
		}
		if left.Priority != right.Priority {
			return left.Priority > right.Priority
		}
		return left.ID < right.ID
	})
	return result
}

func sortedRegionTargetValues(values map[string]RegionContribution) []RegionContribution {
	result := make([]RegionContribution, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := result[i], result[j]
		if regionKindRank(left.Kind) != regionKindRank(right.Kind) {
			return regionKindRank(left.Kind) < regionKindRank(right.Kind)
		}
		if left.TargetID != right.TargetID {
			return left.TargetID < right.TargetID
		}
		if left.Order != right.Order {
			return left.Order < right.Order
		}
		if left.Priority != right.Priority {
			return left.Priority > right.Priority
		}
		return left.ID < right.ID
	})
	return result
}

func navigationKindRank(kind string) int {
	switch kind {
	case NavigationKindMenu:
		return 0
	case NavigationKindItem:
		return 1
	case NavigationKindBreadcrumb:
		return 2
	case NavigationKindHeader:
		return 3
	case NavigationKindFooter:
		return 4
	default:
		return 5
	}
}

func regionKindRank(kind string) int {
	switch kind {
	case RegionKindMenu:
		return 0
	case RegionKindWidget:
		return 1
	case RegionKindHeader:
		return 2
	case RegionKindFooter:
		return 3
	case RegionKindSidebar:
		return 4
	default:
		return 5
	}
}

package navigationregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type Composer struct {
	registry  *Registry
	admission RuntimeAdmission
	renderer  RuntimeRenderer
	traces    TraceSink
}

func NewComposer(registry *Registry, admission RuntimeAdmission, renderer RuntimeRenderer, traces TraceSink) *Composer {
	return &Composer{registry: registry, admission: admission, renderer: renderer, traces: traces}
}

func (c *Composer) Compose(ctx context.Context, request CompositionRequest) (Composition, error) {
	if c == nil || c.registry == nil || ctx == nil {
		return Composition{}, ErrInvalid
	}
	normalizedVisibility, err := normalizeVisibility(request.Visibility)
	if err != nil {
		return Composition{}, err
	}
	request.Visibility = normalizedVisibility.canonical
	// Pin both families behind the writer mutex for the short plan-resolution
	// window. Runtime calls happen after unlock, while both plans retain one
	// immutable revision/digest even during concurrent selection or lifecycle CAS.
	c.registry.mu.Lock()
	navigation, err := c.registry.resolveNavigation(NavigationResolveRequest{
		Kinds: request.NavigationKinds, Locale: request.Locale, Visibility: request.Visibility,
	}, true)
	if err != nil {
		c.registry.mu.Unlock()
		return Composition{}, err
	}
	regions, err := c.registry.resolveRegions(RegionResolveRequest{
		Kinds: request.RegionKinds, Locale: request.Locale, Visibility: request.Visibility,
	}, true)
	c.registry.mu.Unlock()
	if err != nil {
		return Composition{}, err
	}
	if navigation.Revision != regions.Revision || navigation.Digest != regions.Digest ||
		navigation.SafeMode != regions.SafeMode || navigation.Locale != regions.Locale {
		return Composition{}, ErrRevisionConflict
	}
	base, baseCount, err := normalizeBaseChildren(request.BaseNavigationChildren)
	if err != nil {
		return Composition{}, err
	}

	navigationGroups, err := c.composeNavigationGroups(ctx, navigation, request.Visibility)
	if err != nil {
		return Composition{}, err
	}
	regionGroups, err := c.composeRegionGroups(ctx, regions, request.Visibility)
	if err != nil {
		return Composition{}, err
	}
	budget := compositionBudget{items: baseCount}
	navigationItems, navByRegion, err := buildNavigationTree(navigation.Targets, navigationGroups, base, &budget)
	if err != nil {
		return Composition{}, err
	}
	regionItems, err := buildRegionTree(regions.Targets, regionGroups, navByRegion, &budget)
	if err != nil {
		return Composition{}, err
	}
	result := Composition{
		SchemaVersion: CompositionSchemaVersion, Revision: navigation.Revision, Digest: navigation.Digest,
		SafeMode: navigation.SafeMode, Locale: navigation.Locale,
		Navigation: navigationItems, Regions: regionItems,
	}
	body, err := json.Marshal(result)
	if err != nil || len(body) > maxCompositionBytes {
		return Composition{}, ErrLimitExceeded
	}
	keyMaterial := navigation.CacheKey + "\x00" + regions.CacheKey + "\x00" + string(body)
	sum := sha256.Sum256([]byte(keyMaterial))
	result.CacheKey = hex.EncodeToString(sum[:])
	return cloneComposition(result), nil
}

type navigationGroup struct {
	parentID string
	before   []ComposedItem
	node     ComposedItem
	after    []ComposedItem
	visible  bool
}

type regionGroup struct {
	parentID string
	before   []ComposedItem
	node     ComposedItem
	after    []ComposedItem
	visible  bool
}

func (c *Composer) composeNavigationGroups(
	ctx context.Context,
	resolution NavigationResolution,
	visibility VisibilityInput,
) (map[string]navigationGroup, error) {
	groups := make(map[string]navigationGroup, len(resolution.Targets))
	for _, plan := range resolution.Targets {
		group, err := c.composeNavigationPlan(ctx, resolution, plan, visibility)
		if err != nil {
			return nil, err
		}
		groups[plan.Target.ID] = group
	}
	return groups, nil
}

func (c *Composer) composeNavigationPlan(
	ctx context.Context,
	resolution NavigationResolution,
	plan NavigationTargetPlan,
	visibility VisibilityInput,
) (navigationGroup, error) {
	group := navigationGroup{parentID: plan.ParentID}
	for _, hide := range plan.hides {
		_, hidden, err := c.materializeNavigation(ctx, resolution, plan.Target, hide, ComposedItem{}, visibility)
		if err == nil && hidden {
			return group, nil
		}
	}
	provider, node, hidden, err := c.resolveNavigationProvider(ctx, resolution, plan, visibility)
	if err != nil {
		return group, err
	}
	if hidden || provider.ID == "" {
		return group, nil
	}
	for _, filter := range plan.Filters {
		filtered, filterHidden, filterErr := c.materializeNavigation(
			ctx, resolution, plan.Target, filter, node, visibility,
		)
		if filterErr != nil {
			continue
		}
		if filterHidden {
			return group, nil
		}
		node = filtered
	}
	for _, wrapper := range plan.Wrap {
		item, wrapperHidden, wrapperErr := c.materializeNavigation(
			ctx, resolution, plan.Target, wrapper, ComposedItem{}, visibility,
		)
		if wrapperErr == nil && !wrapperHidden {
			node.Wrappers = append(node.Wrappers, item)
		}
	}
	group.node = node
	group.visible = true
	for _, contribution := range plan.Before {
		item, itemHidden, itemErr := c.materializeNavigation(
			ctx, resolution, plan.Target, contribution, ComposedItem{}, visibility,
		)
		if itemErr == nil && !itemHidden {
			group.before = append(group.before, item)
		}
	}
	for _, contribution := range plan.After {
		item, itemHidden, itemErr := c.materializeNavigation(
			ctx, resolution, plan.Target, contribution, ComposedItem{}, visibility,
		)
		if itemErr == nil && !itemHidden {
			group.after = append(group.after, item)
		}
	}
	return group, nil
}

func (c *Composer) resolveNavigationProvider(
	ctx context.Context,
	resolution NavigationResolution,
	plan NavigationTargetPlan,
	visibility VisibilityInput,
) (NavigationContribution, ComposedItem, bool, error) {
	candidates := plan.ReplaceCandidates
	if plan.SelectionConfigured {
		candidates = nil
	}
	if plan.SelectedProvider {
		candidates = []NavigationContribution{plan.Provider}
	}
	for _, candidate := range candidates {
		node, hidden, err := c.materializeNavigation(ctx, resolution, plan.Target, candidate, ComposedItem{}, visibility)
		if err == nil {
			return candidate, node, hidden, nil
		}
		if plan.SelectedProvider {
			c.appendTrace(TraceEvent{
				Revision: resolution.Revision, Family: ProviderFamilyNavigation, TargetID: plan.Target.ID,
				ContributionID: candidate.ID, ContractVersion: candidate.ContractVersion, Action: ActionReplace,
				Handler: candidate.Handler, Locale: resolution.Locale, Outcome: TraceFailedClosed,
				FallbackReason: "selected_replace_unavailable", Artifact: candidate.Artifact,
			})
			return NavigationContribution{}, ComposedItem{}, false, errors.Join(ErrTrustedReplace, err)
		}
	}
	node, hidden, err := c.materializeNavigation(ctx, resolution, plan.Target, plan.Target, ComposedItem{}, visibility)
	if err != nil && !plan.Target.Artifact.Core {
		return NavigationContribution{}, ComposedItem{}, false, nil
	}
	return plan.Target, node, hidden, err
}

func (c *Composer) materializeNavigation(
	ctx context.Context,
	resolution NavigationResolution,
	target NavigationContribution,
	contribution NavigationContribution,
	current ComposedItem,
	visibility VisibilityInput,
) (item ComposedItem, hidden bool, err error) {
	started := time.Now()
	outcome, reason := TraceSucceeded, ""
	defer func() {
		if err != nil {
			outcome = TraceFallback
			if reason == "" {
				reason = "contribution_failed"
			}
		}
		c.appendTrace(TraceEvent{
			Revision: resolution.Revision, Family: ProviderFamilyNavigation, TargetID: target.ID,
			ContributionID: contribution.ID, ContractVersion: contribution.ContractVersion,
			Action: contribution.Action, Handler: contribution.Handler, Locale: resolution.Locale,
			Outcome: outcome, FallbackReason: reason, Duration: time.Since(started), Artifact: contribution.Artifact,
		})
	}()

	item = navigationItem(target, contribution, current)
	lease, leaseErr := c.acquire(ctx, contribution.Artifact)
	if leaseErr != nil {
		reason = "runtime_unavailable"
		return ComposedItem{}, false, leaseErr
	}
	if lease != nil {
		defer lease.Release()
		ctx = lease.Context()
		defer func() {
			if err == nil && !c.runtimeLeaseCurrent(contribution.Artifact, lease) {
				item, hidden, err = ComposedItem{}, false, ErrRuntimeUnavailable
				reason = "runtime_revoked_before_release"
			}
		}()
	}
	if contribution.Action == ActionHide {
		return item, true, nil
	}
	if contribution.Action == ActionFilter && contribution.Handler == "" {
		return item, false, nil
	}
	if contribution.Handler == "" {
		return item, false, nil
	}
	if c.renderer == nil {
		reason = "renderer_unavailable"
		return ComposedItem{}, false, ErrRuntimeUnavailable
	}
	output, renderErr := c.renderer.RenderNavigation(ctx, RuntimeInvocation{
		Family: ProviderFamilyNavigation, Action: contribution.Action, TargetID: target.ID,
		ContributionID: contribution.ID, ContractVersion: contribution.ContractVersion,
		Handler: contribution.Handler, Locale: resolution.Locale,
		Visibility: runtimeVisibilityFor(target.Permission, contribution.Permission, visibility),
		Current:    cloneComposedItem(item), Artifact: contribution.Artifact,
	})
	if renderErr != nil {
		reason = "renderer_failed"
		return ComposedItem{}, false, renderErr
	}
	item, hidden, err = applyRuntimeOutput(item, output, true)
	if err != nil {
		reason = "output_rejected"
	}
	return item, hidden, err
}

func (c *Composer) composeRegionGroups(
	ctx context.Context,
	resolution RegionResolution,
	visibility VisibilityInput,
) (map[string]regionGroup, error) {
	groups := make(map[string]regionGroup, len(resolution.Targets))
	for _, plan := range resolution.Targets {
		group, err := c.composeRegionPlan(ctx, resolution, plan, visibility)
		if err != nil {
			return nil, err
		}
		groups[plan.Target.ID] = group
	}
	return groups, nil
}

func (c *Composer) composeRegionPlan(
	ctx context.Context,
	resolution RegionResolution,
	plan RegionTargetPlan,
	visibility VisibilityInput,
) (regionGroup, error) {
	group := regionGroup{parentID: plan.ParentID}
	for _, hide := range plan.hides {
		_, hidden, err := c.materializeRegion(ctx, resolution, plan.Target, hide, ComposedItem{}, visibility)
		if err == nil && hidden {
			return group, nil
		}
	}
	provider, node, hidden, err := c.resolveRegionProvider(ctx, resolution, plan, visibility)
	if err != nil {
		return group, err
	}
	if hidden || provider.ID == "" {
		return group, nil
	}
	for _, filter := range plan.Filters {
		filtered, filterHidden, filterErr := c.materializeRegion(ctx, resolution, plan.Target, filter, node, visibility)
		if filterErr != nil {
			continue
		}
		if filterHidden {
			return group, nil
		}
		node = filtered
	}
	for _, wrapper := range plan.Wrap {
		item, wrapperHidden, wrapperErr := c.materializeRegion(ctx, resolution, plan.Target, wrapper, ComposedItem{}, visibility)
		if wrapperErr == nil && !wrapperHidden {
			node.Wrappers = append(node.Wrappers, item)
		}
	}
	group.node = node
	group.visible = true
	for _, contribution := range plan.Before {
		item, itemHidden, itemErr := c.materializeRegion(ctx, resolution, plan.Target, contribution, ComposedItem{}, visibility)
		if itemErr == nil && !itemHidden {
			group.before = append(group.before, item)
		}
	}
	for _, contribution := range plan.After {
		item, itemHidden, itemErr := c.materializeRegion(ctx, resolution, plan.Target, contribution, ComposedItem{}, visibility)
		if itemErr == nil && !itemHidden {
			group.after = append(group.after, item)
		}
	}
	return group, nil
}

func (c *Composer) resolveRegionProvider(
	ctx context.Context,
	resolution RegionResolution,
	plan RegionTargetPlan,
	visibility VisibilityInput,
) (RegionContribution, ComposedItem, bool, error) {
	candidates := plan.ReplaceCandidates
	if plan.SelectionConfigured {
		candidates = nil
	}
	if plan.SelectedProvider {
		candidates = []RegionContribution{plan.Provider}
	}
	for _, candidate := range candidates {
		node, hidden, err := c.materializeRegion(ctx, resolution, plan.Target, candidate, ComposedItem{}, visibility)
		if err == nil {
			return candidate, node, hidden, nil
		}
		if plan.SelectedProvider {
			c.appendTrace(TraceEvent{
				Revision: resolution.Revision, Family: ProviderFamilyRegion, TargetID: plan.Target.ID,
				ContributionID: candidate.ID, ContractVersion: candidate.ContractVersion, Action: ActionReplace,
				Handler: candidate.Handler, Locale: resolution.Locale, Outcome: TraceFailedClosed,
				FallbackReason: "selected_replace_unavailable", Artifact: candidate.Artifact,
			})
			return RegionContribution{}, ComposedItem{}, false, errors.Join(ErrTrustedReplace, err)
		}
	}
	node, hidden, err := c.materializeRegion(ctx, resolution, plan.Target, plan.Target, ComposedItem{}, visibility)
	if err != nil && !plan.Target.Artifact.Core {
		return RegionContribution{}, ComposedItem{}, false, nil
	}
	return plan.Target, node, hidden, err
}

func (c *Composer) materializeRegion(
	ctx context.Context,
	resolution RegionResolution,
	target RegionContribution,
	contribution RegionContribution,
	current ComposedItem,
	visibility VisibilityInput,
) (item ComposedItem, hidden bool, err error) {
	started := time.Now()
	outcome, reason := TraceSucceeded, ""
	defer func() {
		if err != nil {
			outcome = TraceFallback
			if reason == "" {
				reason = "contribution_failed"
			}
		}
		c.appendTrace(TraceEvent{
			Revision: resolution.Revision, Family: ProviderFamilyRegion, TargetID: target.ID,
			ContributionID: contribution.ID, ContractVersion: contribution.ContractVersion,
			Action: contribution.Action, Handler: contribution.Handler, Locale: resolution.Locale,
			Outcome: outcome, FallbackReason: reason, Duration: time.Since(started), Artifact: contribution.Artifact,
		})
	}()

	item = regionItem(target, contribution, current)
	lease, leaseErr := c.acquire(ctx, contribution.Artifact)
	if leaseErr != nil {
		reason = "runtime_unavailable"
		return ComposedItem{}, false, leaseErr
	}
	if lease != nil {
		defer lease.Release()
		ctx = lease.Context()
		defer func() {
			if err == nil && !c.runtimeLeaseCurrent(contribution.Artifact, lease) {
				item, hidden, err = ComposedItem{}, false, ErrRuntimeUnavailable
				reason = "runtime_revoked_before_release"
			}
		}()
	}
	if contribution.Action == ActionHide {
		return item, true, nil
	}
	if contribution.Action == ActionFilter && contribution.Handler == "" {
		return item, false, nil
	}
	if contribution.Handler == "" {
		return item, false, nil
	}
	if c.renderer == nil {
		reason = "renderer_unavailable"
		return ComposedItem{}, false, ErrRuntimeUnavailable
	}
	output, renderErr := c.renderer.RenderRegion(ctx, RuntimeInvocation{
		Family: ProviderFamilyRegion, Action: contribution.Action, TargetID: target.ID,
		ContributionID: contribution.ID, ContractVersion: contribution.ContractVersion,
		Handler: contribution.Handler, Locale: resolution.Locale,
		Visibility: runtimeVisibilityFor(target.Permission, contribution.Permission, visibility),
		Current:    cloneComposedItem(item), Artifact: contribution.Artifact,
	})
	if renderErr != nil {
		reason = "renderer_failed"
		return ComposedItem{}, false, renderErr
	}
	item, hidden, err = applyRuntimeOutput(item, output, false)
	if err != nil {
		reason = "output_rejected"
	}
	return item, hidden, err
}

func (c *Composer) acquire(ctx context.Context, artifact Artifact) (RuntimeLease, error) {
	if artifact.Core {
		if !validCoreArtifactSeal(artifact) {
			return nil, ErrRuntimeUnavailable
		}
		return nil, nil
	}
	if c.admission == nil || !c.admission.Available(artifact) {
		return nil, ErrRuntimeUnavailable
	}
	lease, err := c.admission.Acquire(ctx, artifact)
	if err != nil || lease == nil || lease.Context() == nil || lease.RuntimeInstanceID() != artifact.RuntimeInstanceID {
		if lease != nil {
			lease.Release()
		}
		return nil, ErrRuntimeUnavailable
	}
	return lease, nil
}

func (c *Composer) runtimeLeaseCurrent(artifact Artifact, lease RuntimeLease) bool {
	return c != nil && c.admission != nil && lease != nil && lease.Context() != nil &&
		lease.Context().Err() == nil && lease.RuntimeInstanceID() == artifact.RuntimeInstanceID &&
		c.admission.Available(artifact)
}

func runtimeVisibilityFor(targetPermission, contributionPermission string, visibility VisibilityInput) VisibilityInput {
	result := VisibilityInput{Authenticated: visibility.Authenticated}
	permissions, _ := normalizeIDList(visibility.Permissions)
	granted := sliceSet(permissions)
	for _, permission := range []string{targetPermission, contributionPermission} {
		if permission != "" && granted[permission] &&
			!stringSliceContains(result.Permissions, permission) {
			result.Permissions = append(result.Permissions, permission)
		}
	}
	sort.Strings(result.Permissions)
	return result
}

func (c *Composer) appendTrace(event TraceEvent) {
	if c != nil && c.traces != nil {
		c.traces.AppendNavigationTrace(event)
	}
}

func navigationItem(target, provider NavigationContribution, current ComposedItem) ComposedItem {
	if provider.Action == ActionFilter && current.ID != "" {
		return current
	}
	id, contract := provider.ID, provider.ContractVersion
	if provider.Action == ActionReplace {
		id, contract = target.ID, target.ContractVersion
	}
	return ComposedItem{
		ID: id, ContractVersion: contract, ProviderID: provider.ID,
		ProviderContractVersion: provider.ContractVersion, Kind: provider.Kind,
		Order: provider.Order, Priority: provider.Priority, Label: provider.Label, Href: provider.Href,
		Artifact: provider.Artifact,
	}
}

func regionItem(target, provider RegionContribution, current ComposedItem) ComposedItem {
	if provider.Action == ActionFilter && current.ID != "" {
		return current
	}
	id, contract := provider.ID, provider.ContractVersion
	if provider.Action == ActionReplace {
		id, contract = target.ID, target.ContractVersion
	}
	return ComposedItem{
		ID: id, ContractVersion: contract, ProviderID: provider.ID,
		ProviderContractVersion: provider.ContractVersion, Kind: provider.Kind,
		Order: provider.Order, Priority: provider.Priority, Label: provider.Label,
		Multiple: target.Multiple, Artifact: provider.Artifact,
	}
}

func applyRuntimeOutput(item ComposedItem, output RuntimeOutput, navigation bool) (ComposedItem, bool, error) {
	output.Label = strings.TrimSpace(output.Label)
	output.Href = strings.TrimSpace(output.Href)
	if output.Label != "" {
		if utf8.RuneCountInString(output.Label) > 256 {
			return ComposedItem{}, false, ErrLimitExceeded
		}
		item.Label = output.Label
	}
	if output.Href != "" {
		if !navigation || utf8.RuneCountInString(output.Href) > maxNavigationHrefRunes || !safeHostLinkPath(output.Href) {
			return ComposedItem{}, false, ErrInvalid
		}
		item.Href = output.Href
	}
	if utf8.RuneCountInString(output.Content) > maxRuntimeContentRunes || len(output.Attributes) > maxRuntimeAttributes {
		return ComposedItem{}, false, ErrLimitExceeded
	}
	item.Content = output.Content
	if len(output.Attributes) > 0 {
		item.Attributes = make(map[string]string, len(output.Attributes))
		seen := make(map[string]struct{}, len(output.Attributes))
		for key, value := range output.Attributes {
			key = strings.ToLower(strings.TrimSpace(key))
			value = strings.TrimSpace(value)
			if _, duplicate := seen[key]; duplicate || !validComposedAttribute(key, value) {
				return ComposedItem{}, false, ErrInvalid
			}
			seen[key] = struct{}{}
			item.Attributes[key] = value
		}
	}
	return item, output.Hidden, nil
}

func validComposedAttribute(key, value string) bool {
	if !idPattern.MatchString(key) || utf8.RuneCountInString(value) > maxRuntimeAttributeRunes {
		return false
	}
	switch {
	case key == "open-in-new-tab":
		return value == "true" || value == "false"
	case strings.HasPrefix(key, "aria-"):
		return len(key) > len("aria-")
	case strings.HasPrefix(key, "data-sforum-"):
		return len(key) > len("data-sforum-")
	default:
		return false
	}
}

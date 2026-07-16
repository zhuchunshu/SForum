package navigationregistry

import (
	"net/url"
	"sort"
	"strings"
	"unicode/utf8"
)

type compositionBudget struct {
	items int
}

func (b *compositionBudget) add(count int) error {
	b.items += count
	if b.items > maxComposedItems {
		return ErrLimitExceeded
	}
	return nil
}

func buildNavigationTree(
	plans []NavigationTargetPlan,
	groups map[string]navigationGroup,
	base map[string][]ComposedItem,
	budget *compositionBudget,
) ([]ComposedItem, map[string][]ComposedItem, error) {
	planByID := make(map[string]NavigationTargetPlan, len(plans))
	children := map[string][]string{}
	byRegionIDs := map[string][]string{}
	roots := make([]string, 0)
	for _, plan := range plans {
		planByID[plan.Target.ID] = plan
		if plan.ParentID == "" {
			roots = append(roots, plan.Target.ID)
			continue
		}
		if _, parentIsNavigation := groups[plan.ParentID]; parentIsNavigation {
			children[plan.ParentID] = append(children[plan.ParentID], plan.Target.ID)
		} else {
			byRegionIDs[plan.ParentID] = append(byRegionIDs[plan.ParentID], plan.Target.ID)
		}
	}
	for targetID := range base {
		if _, found := planByID[targetID]; !found {
			return nil, nil, ErrInvalid
		}
	}
	var build func(string, int, map[string]bool) ([]ComposedItem, error)
	build = func(id string, depth int, visiting map[string]bool) ([]ComposedItem, error) {
		if depth > maxTargetDepth || visiting[id] {
			return nil, ErrLimitExceeded
		}
		group := groups[id]
		if !group.visible {
			return nil, nil
		}
		visiting[id] = true
		node := cloneComposedItem(group.node)
		node.Children = append(node.Children, cloneComposedItems(base[id])...)
		for _, childID := range children[id] {
			items, err := build(childID, depth+1, visiting)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, items...)
		}
		delete(visiting, id)
		items := append(cloneComposedItems(group.before), node)
		items = append(items, cloneComposedItems(group.after)...)
		if err := budget.add(len(items) + len(node.Wrappers)); err != nil {
			return nil, err
		}
		return items, nil
	}
	result := make([]ComposedItem, 0)
	for _, root := range roots {
		items, err := build(root, 1, map[string]bool{})
		if err != nil {
			return nil, nil, err
		}
		result = append(result, items...)
	}
	byRegion := make(map[string][]ComposedItem, len(byRegionIDs))
	for regionID, ids := range byRegionIDs {
		sort.SliceStable(ids, func(i, j int) bool {
			left, right := planByID[ids[i]].Target, planByID[ids[j]].Target
			return contributionOrder(left.Order, left.Priority, left.ID, right.Order, right.Priority, right.ID)
		})
		for _, id := range ids {
			items, err := build(id, 1, map[string]bool{})
			if err != nil {
				return nil, nil, err
			}
			byRegion[regionID] = append(byRegion[regionID], items...)
		}
	}
	return result, byRegion, nil
}

func buildRegionTree(
	plans []RegionTargetPlan,
	groups map[string]regionGroup,
	navigationByRegion map[string][]ComposedItem,
	budget *compositionBudget,
) ([]ComposedItem, error) {
	children := map[string][]string{}
	roots := make([]string, 0)
	for _, plan := range plans {
		if plan.ParentID == "" {
			roots = append(roots, plan.Target.ID)
		} else {
			children[plan.ParentID] = append(children[plan.ParentID], plan.Target.ID)
		}
	}
	// Navigation groups attached to a region are composed as typed children. A
	// missing region parent remains invisible rather than leaking as a root.
	var build func(string, int, map[string]bool) ([]ComposedItem, error)
	build = func(id string, depth int, visiting map[string]bool) ([]ComposedItem, error) {
		if depth > maxTargetDepth || visiting[id] {
			return nil, ErrLimitExceeded
		}
		group := groups[id]
		if !group.visible {
			return nil, nil
		}
		visiting[id] = true
		node := cloneComposedItem(group.node)
		for _, childID := range children[id] {
			items, err := build(childID, depth+1, visiting)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, items...)
		}
		node.Children = append(node.Children, cloneComposedItems(navigationByRegion[id])...)
		delete(visiting, id)
		items := append(cloneComposedItems(group.before), node)
		items = append(items, cloneComposedItems(group.after)...)
		if err := budget.add(len(items) + len(node.Wrappers)); err != nil {
			return nil, err
		}
		return items, nil
	}
	result := make([]ComposedItem, 0)
	for _, root := range roots {
		items, err := build(root, 1, map[string]bool{})
		if err != nil {
			return nil, err
		}
		result = append(result, items...)
	}
	return result, nil
}

func normalizeBaseChildren(input map[string][]ComposedItem) (map[string][]ComposedItem, int, error) {
	if len(input) > maxComposedItems {
		return nil, 0, ErrLimitExceeded
	}
	result := make(map[string][]ComposedItem, len(input))
	count := 0
	for rawTarget, rawItems := range input {
		targetID := strings.ToLower(strings.TrimSpace(rawTarget))
		if !idPattern.MatchString(targetID) {
			return nil, 0, ErrInvalid
		}
		items := cloneComposedItems(rawItems)
		for index := range items {
			if err := validateBaseItem(items[index], 1, &count); err != nil {
				return nil, 0, err
			}
		}
		sortComposedItems(items)
		result[targetID] = items
	}
	return result, count, nil
}

func validateBaseItem(item ComposedItem, depth int, count *int) error {
	*count = *count + 1
	if depth > maxTargetDepth || *count > maxComposedItems {
		return ErrLimitExceeded
	}
	if !idPattern.MatchString(item.ID) ||
		!contractPattern.MatchString(item.ContractVersion) || !validNavigationKind(item.Kind) ||
		strings.TrimSpace(item.Label) == "" || utf8.RuneCountInString(item.Label) > 256 ||
		(item.Href != "" && (utf8.RuneCountInString(item.Href) > maxNavigationHrefRunes || !safeComposedHref(item.Href))) {
		return ErrInvalid
	}
	if len(item.Attributes) > maxRuntimeAttributes {
		return ErrLimitExceeded
	}
	for key, value := range item.Attributes {
		if key != strings.ToLower(strings.TrimSpace(key)) || value != strings.TrimSpace(value) ||
			!validComposedAttribute(key, value) {
			return ErrInvalid
		}
	}
	if _, err := normalizeArtifact(item.Artifact); err != nil {
		return ErrInvalid
	}
	for _, child := range item.Children {
		if err := validateBaseItem(child, depth+1, count); err != nil {
			return err
		}
	}
	return nil
}

func safeComposedHref(value string) bool {
	if safeHostLinkPath(value) {
		return true
	}
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed != nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func sortComposedItems(items []ComposedItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return contributionOrder(items[i].Order, items[i].Priority, items[i].ID, items[j].Order, items[j].Priority, items[j].ID)
	})
}

func contributionOrder(leftOrder, leftPriority int, leftID string, rightOrder, rightPriority int, rightID string) bool {
	if leftOrder != rightOrder {
		return leftOrder < rightOrder
	}
	if leftPriority != rightPriority {
		return leftPriority > rightPriority
	}
	return leftID < rightID
}

func cloneComposition(value Composition) Composition {
	value.Navigation = cloneComposedItems(value.Navigation)
	value.Regions = cloneComposedItems(value.Regions)
	return value
}

func cloneComposedItems(values []ComposedItem) []ComposedItem {
	if len(values) == 0 {
		return nil
	}
	result := make([]ComposedItem, len(values))
	for index := range values {
		result[index] = cloneComposedItem(values[index])
	}
	return result
}

func cloneComposedItem(value ComposedItem) ComposedItem {
	value.Attributes = cloneStringMap(value.Attributes)
	value.Wrappers = cloneComposedItems(value.Wrappers)
	value.Children = cloneComposedItems(value.Children)
	return value
}

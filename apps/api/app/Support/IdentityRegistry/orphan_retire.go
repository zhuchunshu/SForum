package identityregistry

import (
	"sort"
	"strings"
)

// ActiveOrphanOwners lists durable identity owners that still have an active
// root tip and/or active declaration tip, but are not among the expected enabled
// identity publishers for this process. Startup uses this to complete incomplete
// uninstalls and force-deleted plugin residue before fail-closed set validation.
func ActiveOrphanOwners(state DurableState, expectedExtensionIDs []string) ([]string, error) {
	if _, err := DurableStateToTombstones(state); err != nil {
		return nil, err
	}
	expected := make(map[string]struct{}, len(expectedExtensionIDs))
	for _, raw := range expectedExtensionIDs {
		id := strings.ToLower(strings.TrimSpace(raw))
		if id == "" || !idPattern.MatchString(id) || strings.HasPrefix(id, "core.") {
			return nil, ErrInvalid
		}
		expected[id] = struct{}{}
	}

	roots, err := durableRootPublications(state)
	if err != nil {
		return nil, err
	}
	orphans := make(map[string]struct{})
	for owner, root := range roots {
		if root.tip.RegistryState != RegistryStateActive {
			continue
		}
		if _, keep := expected[owner]; keep {
			continue
		}
		orphans[owner] = struct{}{}
	}
	for _, tip := range latestDurableDeclarationTips(state.Tips) {
		if tip.RegistryState != RegistryStateActive {
			continue
		}
		owner := strings.ToLower(strings.TrimSpace(tip.OwnerExtensionID))
		if _, keep := expected[owner]; keep {
			continue
		}
		orphans[owner] = struct{}{}
	}

	result := make([]string, 0, len(orphans))
	for owner := range orphans {
		result = append(result, owner)
	}
	sort.Strings(result)
	return result, nil
}

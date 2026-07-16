package extensions

import "strings"

// TransitionPluginRuntimeTrustRevocationMembers removes every executable
// member owned by one revoked extension while preserving the exact unrelated
// desired set. The PostgreSQL producer decides whether an already absent
// member needs a new revision.
func TransitionPluginRuntimeTrustRevocationMembers(
	latest []PluginRuntimeMember,
	extensionID string,
) ([]PluginRuntimeMember, error) {
	if extensionID == "" || extensionID != strings.TrimSpace(extensionID) {
		return nil, ErrPluginRuntimePublicationConflict
	}
	canonical, _, err := canonicalPluginRuntimeMembers(latest)
	if err != nil {
		return nil, ErrPluginRuntimePublicationConflict
	}
	next := make([]PluginRuntimeMember, 0, len(canonical))
	for _, member := range canonical {
		if member.ExtensionID != extensionID {
			next = append(next, member)
		}
	}
	return next, nil
}

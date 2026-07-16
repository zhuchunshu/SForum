package extensions

import "strings"

// TransitionPluginRuntimeTrustRevocationMembers removes every executable
// member owned by one revoked extension while preserving the exact unrelated
// desired set. The caller still publishes an unchanged set when the member is
// already absent so connected nodes receive a durable wake revision.
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

package extensions

import (
	"errors"
	"strings"
	"testing"
)

func TestTransitionPluginRuntimeTrustRevocationMembers(t *testing.T) {
	revoked := PluginRuntimeMember{
		ExtensionID: "revoked.plugin", ExtensionVersionID: 1,
		ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("a", 64),
	}
	unrelated := PluginRuntimeMember{
		ExtensionID: "unrelated.plugin", ExtensionVersionID: 2,
		ExtensionVersion: "2.0.0", PackageDigest: strings.Repeat("b", 64),
	}
	input := []PluginRuntimeMember{unrelated, revoked}
	next, err := TransitionPluginRuntimeTrustRevocationMembers(input, revoked.ExtensionID)
	if err != nil {
		t.Fatal(err)
	}
	assertTransitionMembers(t, next, unrelated)
	if input[0] != unrelated || input[1] != revoked {
		t.Fatalf("trust revocation mutated caller input: %#v", input)
	}

	replayed, err := TransitionPluginRuntimeTrustRevocationMembers(next, revoked.ExtensionID)
	if err != nil {
		t.Fatal(err)
	}
	assertTransitionMembers(t, replayed, unrelated)
}

func TestTransitionPluginRuntimeTrustRevocationMembersRejectsInvalidInput(t *testing.T) {
	valid := PluginRuntimeMember{
		ExtensionID: "valid.plugin", ExtensionVersionID: 1,
		ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("a", 64),
	}
	for _, extensionID := range []string{"", " valid.plugin", "valid.plugin "} {
		if _, err := TransitionPluginRuntimeTrustRevocationMembers([]PluginRuntimeMember{valid}, extensionID); !errors.Is(err, ErrPluginRuntimePublicationConflict) {
			t.Fatalf("extensionID=%q error=%v", extensionID, err)
		}
	}
	if _, err := TransitionPluginRuntimeTrustRevocationMembers([]PluginRuntimeMember{valid, valid}, valid.ExtensionID); !errors.Is(err, ErrPluginRuntimePublicationConflict) {
		t.Fatalf("duplicate desired set error=%v", err)
	}
}

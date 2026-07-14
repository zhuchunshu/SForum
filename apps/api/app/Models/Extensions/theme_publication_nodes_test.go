package extensions

import (
	"strings"
	"testing"
)

func TestThemeRuntimeNodeIdentityBounds(t *testing.T) {
	if !validThemeRuntimeNodeIdentity(ThemeRuntimeNodeIdentity{NodeID: "api-1", BootID: "boot-1"}) {
		t.Fatal("valid node identity rejected")
	}
	for name, identity := range map[string]ThemeRuntimeNodeIdentity{
		"missing node": {BootID: "boot"},
		"missing boot": {NodeID: "node"},
		"long node":    {NodeID: strings.Repeat("n", 129), BootID: "boot"},
		"long boot":    {NodeID: "node", BootID: strings.Repeat("b", 129)},
	} {
		t.Run(name, func(t *testing.T) {
			if validThemeRuntimeNodeIdentity(identity) {
				t.Fatalf("invalid identity accepted: %#v", identity)
			}
		})
	}
}

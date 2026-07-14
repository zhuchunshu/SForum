package bootstrap

import (
	"strings"
	"testing"
)

func TestNormalizeThemeRuntimeNodeIDKeepsStableBoundedIdentity(t *testing.T) {
	if nodeID, err := normalizeThemeRuntimeNodeID("  api-1  "); err != nil || nodeID != "api-1" {
		t.Fatalf("nodeID=%q err=%v", nodeID, err)
	}
	long := strings.Repeat("node", 80)
	first, err := normalizeThemeRuntimeNodeID(long)
	if err != nil || len([]byte(first)) > 128 || !strings.HasPrefix(first, "host-") {
		t.Fatalf("hashed nodeID=%q err=%v", first, err)
	}
	second, err := normalizeThemeRuntimeNodeID(long)
	if err != nil || second != first {
		t.Fatalf("unstable nodeID first=%q second=%q err=%v", first, second, err)
	}
	if _, err := normalizeThemeRuntimeNodeID("   "); err == nil {
		t.Fatal("blank hostname was accepted")
	}
}

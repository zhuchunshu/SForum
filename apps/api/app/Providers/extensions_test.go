package providers

import (
	"testing"

	extensionscontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Extensions"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

func TestPublicFrontendRuntimeMatchesExactAcquiredInstance(t *testing.T) {
	target := extensionsruntime.RuntimeInstanceSnapshot{
		Identity:         extensionsruntime.RuntimeInstanceIdentity{ExtensionID: "demo.plugin", InstanceID: "runtime-1"},
		ExtensionVersion: "1.0.0",
		ArtifactDigest:   "package-v1",
	}
	if !publicFrontendRuntimeMatches(nil, target) {
		t.Fatal("ordinary extension route unexpectedly required a public frontend identity")
	}
	exact := &extensionscontroller.PublicFrontendBridgeIdentity{
		ExtensionID: "demo.plugin", ExtensionVersion: "1.0.0", PackageDigest: "package-v1",
	}
	if !publicFrontendRuntimeMatches(exact, target) {
		t.Fatal("exact public frontend bridge did not match its acquired runtime")
	}
	stale := *exact
	stale.PackageDigest = "package-v2"
	if publicFrontendRuntimeMatches(&stale, target) {
		t.Fatal("stale public frontend package reached a different runtime")
	}
}

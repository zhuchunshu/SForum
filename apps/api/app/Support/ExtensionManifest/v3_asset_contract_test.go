package extensionmanifest

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestManifestV3AssetRuntimeContract(t *testing.T) {
	manifest := completeV3Manifest()
	asset := &manifest.Assets[0]
	asset.Module = false
	asset.Loading = "lazy"
	asset.Dependencies = []string{" CORE.ASSET.VUE "}
	asset.Scope = []string{" DEMO.V3.COMPONENT.CARD "}
	asset.Integrity = ""
	asset.CSP = []string{" connect-src   'self' "}

	normalized := Normalize(manifest)
	asset = &normalized.Assets[0]
	if asset.Dependencies[0] != "core.asset.vue" || asset.Scope[0] != "demo.v3.component.card" ||
		asset.CSP[0] != "connect-src 'self'" {
		t.Fatalf("asset declaration was not normalized: %#v", asset)
	}
	if err := Validate(normalized); err != nil {
		t.Fatalf("valid scoped lazy asset: %v", err)
	}
	body, err := json.Marshal(normalized)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("raw upload schema rejected scoped lazy asset: %v", err)
	}
	asset.Integrity = assetIntegrityForTest(t, asset.Digest)
	if err := Validate(normalized); err != nil {
		t.Fatalf("valid exact SRI: %v", err)
	}
}

func TestManifestV3RejectsUnsafeAssetRuntimeContracts(t *testing.T) {
	tests := []struct {
		name   string
		change func(*ManifestAsset)
	}{
		{name: "style module", change: func(asset *ManifestAsset) { asset.Module = true }},
		{name: "script mime confusion", change: func(asset *ManifestAsset) { asset.Type, asset.Path = "script", "frontend/card.css" }},
		{name: "style mime confusion", change: func(asset *ManifestAsset) { asset.Path = "frontend/card.mjs" }},
		{name: "self dependency", change: func(asset *ManifestAsset) { asset.Dependencies = []string{asset.Handle} }},
		{name: "duplicate dependency", change: func(asset *ManifestAsset) { asset.Dependencies = []string{"core.asset.vue", "core.asset.vue"} }},
		{name: "invalid scope", change: func(asset *ManifestAsset) { asset.Scope = []string{"../page"} }},
		{name: "duplicate scope", change: func(asset *ManifestAsset) { asset.Scope = []string{"forum.home", "forum.home"} }},
		{name: "wrong integrity", change: func(asset *ManifestAsset) { asset.Integrity = "sha256-wrong" }},
		{name: "multiple csp directives", change: func(asset *ManifestAsset) { asset.CSP = []string{"script-src 'self'; default-src *"} }},
		{name: "unknown csp directive", change: func(asset *ManifestAsset) { asset.CSP = []string{"frame-ancestors 'none'"} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := completeV3Manifest()
			test.change(&manifest.Assets[0])
			if err := Validate(manifest); err == nil {
				t.Fatal("unsafe asset declaration must be rejected")
			}
		})
	}
}

func assetIntegrityForTest(t *testing.T, digest string) string {
	t.Helper()
	raw, err := hex.DecodeString(digest)
	if err != nil {
		t.Fatal(err)
	}
	return "sha256-" + base64.StdEncoding.EncodeToString(raw)
}

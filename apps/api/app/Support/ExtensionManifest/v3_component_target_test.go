package extensionmanifest

import "testing"

func TestManifestV3AcceptsReviewedPublicAndAdminComponentTargets(t *testing.T) {
	targets := []struct {
		id       string
		contract string
	}{
		{id: "core.component.page.forum.home", contract: "sforum.component.page.forum.home@1"},
		{id: "core.component.page.admin", contract: "sforum.component.page.admin@1"},
		{id: "core.component.admin.sfadmin_footer", contract: "sforum.component.admin.sfadmin_footer@1"},
		{id: "core.component.shared.sfbutton", contract: "sforum.component.shared.sfbutton@1"},
	}
	for _, target := range targets {
		t.Run(target.id, func(t *testing.T) {
			manifest := completeV3Manifest()
			manifest.Components[0].Action = ComponentActionBefore
			manifest.Components[0].TargetID = target.id
			manifest.Components[0].TargetContractVersion = target.contract
			if err := Validate(manifest); err != nil {
				t.Fatalf("reviewed Host target %q should validate: %v", target.id, err)
			}
		})
	}
}

func TestManifestV3RejectsUnknownCoreComponentTarget(t *testing.T) {
	manifest := completeV3Manifest()
	manifest.Components[0].Action = ComponentActionReplace
	manifest.Components[0].TargetID = "core.component.page.missing"
	manifest.Components[0].TargetContractVersion = "sforum.component.page.missing@1"
	if err := Validate(manifest); err == nil {
		t.Fatal("unknown core.component target must fail closed")
	}
}

func TestManifestV3ValidatesEveryDeclaredComponentTarget(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		contract string
		valid    bool
	}{
		{name: "malformed target", target: "not a component id", contract: "other.plugin.component.card@1"},
		{name: "missing target contract", target: "core.component.page.forum.home"},
		{name: "orphan target contract", contract: "sforum.component.page.forum.home@1"},
		{name: "malformed target contract", target: "core.component.page.forum.home", contract: "not-a-contract"},
		{name: "mismatched Core contract", target: "core.component.page.forum.home", contract: "sforum.component.page.forum.home@2"},
		{name: "unknown Core target on add", target: "core.component.shared.missing", contract: "sforum.component.shared.missing@1"},
		{name: "reserved non-component Core id", target: "core.asset.forum", contract: "sforum.component.asset.forum@1"},
		{name: "external plugin target", target: "other.plugin.component.card", contract: "other.plugin.component.card@1", valid: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := completeV3Manifest()
			manifest.Components[0].Action = ComponentActionAdd
			manifest.Components[0].TargetID = test.target
			manifest.Components[0].TargetContractVersion = test.contract
			err := Validate(manifest)
			if test.valid && err != nil {
				t.Fatalf("external target should remain composable: %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("invalid target must fail closed")
			}
		})
	}
}

func TestManifestV3ThemeCannotTargetAdminOnlyCoreComponent(t *testing.T) {
	tests := []struct {
		target   string
		contract string
		valid    bool
	}{
		{target: "core.component.page.forum.home", contract: "sforum.component.page.forum.home@1", valid: true},
		{target: "core.component.shared.sfbutton", contract: "sforum.component.shared.sfbutton@1", valid: true},
		{target: "core.component.page.admin", contract: "sforum.component.page.admin@1"},
		{target: "core.component.admin.sfadmin_footer", contract: "sforum.component.admin.sfadmin_footer@1"},
	}
	for _, test := range tests {
		t.Run(test.target, func(t *testing.T) {
			manifest := versionedTestManifest(ManifestVersionV3)
			manifest.ID = "demo.theme"
			manifest.Type = TypeTheme
			manifest.Components = []ManifestComponent{{
				ID: "demo.theme.component.target", ContractVersion: "demo.theme.component.target@1",
				Action: ComponentActionHide, TargetID: test.target, TargetContractVersion: test.contract,
			}}
			err := Validate(manifest)
			if test.valid && err != nil {
				t.Fatalf("public theme target should validate: %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("theme must not target an admin-only Core component")
			}
		})
	}
}

func TestManifestV3ComponentTargetJSONSchemaRequiresVersionedPair(t *testing.T) {
	valid := []byte(`[{"id":"demo.component","contractVersion":"demo.component@1","action":"hide","targetId":"core.component.page.forum.home","targetContractVersion":"sforum.component.page.forum.home@1"}]`)
	if err := validateV3JSONSchemaFragment(valid, "components"); err != nil {
		t.Fatalf("versioned target pair should satisfy JSON Schema: %v", err)
	}
	for _, body := range [][]byte{
		[]byte(`[{"id":"demo.component","contractVersion":"demo.component@1","action":"hide","targetId":"core.component.page.forum.home"}]`),
		[]byte(`[{"id":"demo.component","contractVersion":"demo.component@1","action":"add","targetContractVersion":"sforum.component.page.forum.home@1"}]`),
	} {
		if err := validateV3JSONSchemaFragment(body, "components"); err == nil {
			t.Fatal("JSON Schema accepted an incomplete component target pair")
		}
	}
}

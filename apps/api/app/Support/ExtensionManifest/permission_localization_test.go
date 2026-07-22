package extensionmanifest

import (
	"encoding/json"
	"testing"
)

func TestPermissionDefinitionLocalizedTextSupportsMapsAndLegacyStrings(t *testing.T) {
	var localized ManifestPermissionDefinition
	if err := json.Unmarshal([]byte(`{
		"key":"demo.permission.manage",
		"contractVersion":"demo.permission.manage@1",
		"label":{"zh-CN":"管理演示扩展","en-US":"Manage demo"},
		"description":{"zh":"管理演示功能。","en-US":"Manage demo features."},
		"assignmentPolicy":"host"
	}`), &localized); err != nil {
		t.Fatal(err)
	}
	if got := localized.Label.Resolve("zh-CN"); got != "管理演示扩展" {
		t.Fatalf("zh-CN label = %q", got)
	}
	if got := localized.Description.Resolve("zh-TW"); got != "管理演示功能。" {
		t.Fatalf("zh prefix description = %q", got)
	}
	if got := localized.Label.Resolve("fr-FR"); got != "Manage demo" {
		t.Fatalf("default label = %q", got)
	}

	var legacy ManifestPermissionDefinition
	if err := json.Unmarshal([]byte(`{
		"key":"demo.permission.read",
		"contractVersion":"demo.permission.read@1",
		"label":"Read demo",
		"description":"Read demo records.",
		"assignmentPolicy":"host"
	}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.Label.Default != "Read demo" || legacy.Description.Default != "Read demo records." {
		t.Fatalf("legacy strings = %#v", legacy)
	}
}

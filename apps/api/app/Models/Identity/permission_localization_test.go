package identity

import "testing"

func TestLocalizePermissionsUsesExactPrefixAndDefaultFallbacks(t *testing.T) {
	permission := Permission{
		Key:         "demo.permission.manage",
		Module:      "extension",
		Label:       "Manage demo",
		Description: "Manage demo features.",
		LabelLocales: map[string]string{
			"zh-CN": "管理演示扩展",
			"zh":    "管理演示",
		},
		DescriptionLocales: map[string]string{
			"zh": "管理演示功能。",
		},
	}

	localized := localizePermissions([]Permission{permission}, "zh-CN")
	if localized[0].Label != "管理演示扩展" || localized[0].Description != "管理演示功能。" {
		t.Fatalf("zh-CN permission = %#v", localized[0])
	}
	localized = localizePermissions([]Permission{permission}, "fr-FR")
	if localized[0].Label != permission.Label || localized[0].Description != permission.Description {
		t.Fatalf("fallback permission = %#v", localized[0])
	}
}

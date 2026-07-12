package extensionmanifest

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestAdminFrontendNormalizeAndValidateWithComponentCatalog(t *testing.T) {
	manifest := validAdminFrontendManifest()
	manifest.Frontend.Admin.Root = ` frontend\admin `
	manifest.Frontend.Admin.Components["fixture.panel"] = ` components\FixturePanel.vue `
	manifest.Frontend.Admin.Locales["zh-CN"] = ` locales\zh-CN.json `
	manifest.Contributions[0].Payload = json.RawMessage(`{"width":120,"component":" Fixture.Panel "}`)

	normalized := Normalize(manifest)
	if normalized.Frontend.Admin == nil {
		t.Fatal("normalized admin frontend declaration is missing")
	}
	if normalized.Frontend.Admin.Root != "frontend/admin" {
		t.Fatalf("unexpected normalized root: %q", normalized.Frontend.Admin.Root)
	}
	if normalized.Frontend.Admin.Components["fixture.panel"] != "components/FixturePanel.vue" {
		t.Fatalf("unexpected normalized component path: %#v", normalized.Frontend.Admin.Components)
	}
	if normalized.Frontend.Admin.Locales["zh-CN"] != "locales/zh-CN.json" {
		t.Fatalf("unexpected normalized locale path: %#v", normalized.Frontend.Admin.Locales)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(normalized.Contributions[0].Payload, &payload); err != nil {
		t.Fatalf("decode normalized component payload: %v", err)
	}
	var component string
	if err := json.Unmarshal(payload["component"], &component); err != nil {
		t.Fatalf("decode normalized component id: %v", err)
	}
	if component != "fixture.panel" || string(payload["width"]) != "120" {
		t.Fatalf("component payload lost its binding or slot fields: component=%q payload=%s", component, normalized.Contributions[0].Payload)
	}

	if err := ValidateWithContributionPoints(manifest, adminFixtureContributionPoints()); err != nil {
		t.Fatalf("valid admin frontend manifest should pass the test catalog: %v", err)
	}
	if err := Validate(manifest); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("production catalog must not accept the test-only point, got %v", err)
	}
}

func TestThemeMayDeclareAdminFrontendWithSettingsPage(t *testing.T) {
	manifest := validAdminFrontendManifest()
	manifest.Type = TypeTheme
	manifest.Frontend.Layer = "layer"
	manifest.Contributions = []ManifestContribution{{
		Point:   "admin.extension.settings.page",
		ID:      "theme-settings-page",
		Order:   10,
		Label:   map[string]string{"zh-CN": "主题设置", "en-US": "Theme settings"},
		Payload: json.RawMessage(`{"component":"fixture.panel"}`),
	}}
	if err := Validate(manifest); err != nil {
		t.Fatalf("theme with settings page admin frontend should validate: %v", err)
	}
}

func TestThemeRejectsNonSettingsContributionPoints(t *testing.T) {
	manifest := validAdminFrontendManifest()
	manifest.Type = TypeTheme
	manifest.Frontend.Layer = "layer"
	// 主题不可贡献 jobs 等插件向点。
	manifest.Contributions = []ManifestContribution{{
		Point:   "admin.jobs.table.columns",
		ID:      "theme-job-col",
		Order:   10,
		Label:   map[string]string{"zh-CN": "非法", "en-US": "Illegal"},
		Payload: json.RawMessage(`{"component":"fixture.panel"}`),
	}}
	if err := Validate(manifest); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected ErrInvalidManifest for theme jobs contribution, got %v", err)
	}
}

func TestAdminFrontendRejectsInvalidDeclarations(t *testing.T) {
	tests := []struct {
		name   string
		points []ContributionPointDefinition
		mutate func(*Manifest)
	}{
		{
			name: "root must stay in package",
			mutate: func(manifest *Manifest) {
				manifest.Frontend.Admin.Root = "../frontend/admin"
			},
		},
		{
			name: "root cannot be package root",
			mutate: func(manifest *Manifest) {
				manifest.Frontend.Admin.Root = "."
			},
		},
		{
			name: "api version must be positive",
			mutate: func(manifest *Manifest) {
				manifest.Frontend.Admin.APIVersion = 0
			},
		},
		{
			name: "api version must be supported",
			mutate: func(manifest *Manifest) {
				manifest.Frontend.Admin.APIVersion = AdminFrontendAPIVersion + 1
			},
		},
		{
			name: "components are required",
			mutate: func(manifest *Manifest) {
				manifest.Frontend.Admin.Components = nil
			},
		},
		{
			name: "component id must be canonical",
			mutate: func(manifest *Manifest) {
				manifest.Frontend.Admin.Components = map[string]string{"Fixture Panel": "components/FixturePanel.vue"}
			},
		},
		{
			name: "component path cannot escape root",
			mutate: func(manifest *Manifest) {
				manifest.Frontend.Admin.Components["fixture.panel"] = "../FixturePanel.vue"
			},
		},
		{
			name: "component module extension must be supported",
			mutate: func(manifest *Manifest) {
				manifest.Frontend.Admin.Components["fixture.panel"] = "components/FixturePanel.json"
			},
		},
		{
			name: "zh-CN locale is required",
			mutate: func(manifest *Manifest) {
				delete(manifest.Frontend.Admin.Locales, "zh-CN")
			},
		},
		{
			name: "en-US locale is required",
			mutate: func(manifest *Manifest) {
				delete(manifest.Frontend.Admin.Locales, "en-US")
			},
		},
		{
			name: "locale path cannot escape root",
			mutate: func(manifest *Manifest) {
				manifest.Frontend.Admin.Locales["zh-CN"] = "../zh-CN.json"
			},
		},
		{
			name: "locale path must be json",
			mutate: func(manifest *Manifest) {
				manifest.Frontend.Admin.Locales["zh-CN"] = "locales/zh-CN.ts"
			},
		},
		{
			name: "unknown contribution point",
			mutate: func(manifest *Manifest) {
				manifest.Contributions[0].Point = "admin.unknown"
			},
		},
		{
			name: "duplicate contribution id",
			mutate: func(manifest *Manifest) {
				manifest.Frontend.Admin.Components["fixture.secondary"] = "components/Secondary.vue"
				manifest.Contributions = append(manifest.Contributions, adminFixtureContribution("fixture.panel", "fixture.secondary"))
			},
		},
		{
			name: "component binding is required",
			mutate: func(manifest *Manifest) {
				manifest.Contributions[0].Payload = json.RawMessage(`{"width":120}`)
			},
		},
		{
			name: "component binding must exist",
			mutate: func(manifest *Manifest) {
				manifest.Contributions[0].Payload = json.RawMessage(`{"component":"fixture.missing"}`)
			},
		},
		{
			name: "every component must be used",
			mutate: func(manifest *Manifest) {
				manifest.Frontend.Admin.Components["fixture.unused"] = "components/Unused.vue"
			},
		},
		{
			name: "component can be bound only once",
			mutate: func(manifest *Manifest) {
				manifest.Contributions = append(manifest.Contributions, adminFixtureContribution("fixture.second", "fixture.panel"))
			},
		},
		{
			name: "component point requires admin declaration",
			mutate: func(manifest *Manifest) {
				manifest.Frontend.Admin = nil
			},
		},
		{
			name: "descriptor point cannot carry component contract",
			points: []ContributionPointDefinition{{
				ID:          "admin.test.fixture",
				Owner:       "test",
				Kind:        ContributionPointKindDescriptor,
				Description: "Test descriptor fixture.",
				PayloadType: "extensionRoute",
			}},
			mutate: func(*Manifest) {},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := validAdminFrontendManifest()
			test.mutate(&manifest)
			points := test.points
			if points == nil {
				points = adminFixtureContributionPoints()
			}
			if err := ValidateWithContributionPoints(manifest, points); !errors.Is(err, ErrInvalidManifest) {
				t.Fatalf("expected ErrInvalidManifest, got %v", err)
			}
		})
	}
}

func validAdminFrontendManifest() Manifest {
	return Manifest{
		ID:            "fixture.plugin",
		Name:          "Fixture Plugin",
		Description:   "Trusted admin frontend fixture.",
		URL:           "https://example.com/fixture",
		Author:        ManifestAuthor{Name: "SForum Test"},
		Version:       "1.0.0",
		Type:          TypePlugin,
		SForumVersion: "^1.0.0",
		Frontend: ManifestFrontend{Admin: &ManifestAdminFrontend{
			Root:       "frontend/admin",
			APIVersion: AdminFrontendAPIVersion,
			Components: map[string]string{"fixture.panel": "components/FixturePanel.vue"},
			Locales: map[string]string{
				"zh-CN": "locales/zh-CN.json",
				"en-US": "locales/en-US.json",
			},
		}},
		Contributions: []ManifestContribution{adminFixtureContribution("fixture.panel", "fixture.panel")},
	}
}

func adminFixtureContribution(id string, component string) ManifestContribution {
	return ManifestContribution{
		Point:   "admin.test.fixture",
		ID:      id,
		Order:   100,
		Label:   map[string]string{"zh-CN": "测试组件", "en-US": "Test component"},
		Payload: json.RawMessage(`{"component":"` + component + `"}`),
	}
}

func adminFixtureContributionPoints() []ContributionPointDefinition {
	return []ContributionPointDefinition{{
		ID:          "admin.test.fixture",
		Owner:       "test",
		Kind:        ContributionPointKindComponent,
		Description: "Build-only trusted component fixture.",
		PayloadType: "adminComponent",
	}}
}

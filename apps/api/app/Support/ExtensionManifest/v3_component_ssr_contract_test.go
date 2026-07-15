package extensionmanifest

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestManifestV3ComponentSSRTemplateConsistency(t *testing.T) {
	tests := []struct {
		name   string
		valid  bool
		change func(*Manifest)
	}{
		{
			name:  "exact matching schema and override key",
			valid: true,
			change: func(manifest *Manifest) {
				// completeV3Manifest 已满足精确绑定。
			},
		},
		{
			name:  "both override keys empty remains deterministic",
			valid: true,
			change: func(manifest *Manifest) {
				manifest.Components[0].ThemeOverrideKey = ""
				manifest.Templates[0].ThemeOverrideKey = ""
			},
		},
		{
			name:  "l2 only without ssr template remains valid",
			valid: true,
			change: func(manifest *Manifest) {
				manifest.Components[0].SSRTemplate = ""
				manifest.Components[0].ThemeOverrideKey = "fixture.l2.card"
			},
		},
		{
			name: "missing ssr template id",
			change: func(manifest *Manifest) {
				manifest.Components[0].SSRTemplate = "demo.v3.template.missing"
			},
		},
		{
			name: "props schema drifts from view model schema",
			change: func(manifest *Manifest) {
				manifest.Components[0].PropsSchema = "demo.v3.component.card.other@1"
			},
		},
		{
			name: "view model schema drifts from props schema",
			change: func(manifest *Manifest) {
				manifest.Templates[0].ViewModelSchema = "demo.v3.template.card.model@1"
			},
		},
		{
			name: "schema case ambiguity fails closed",
			change: func(manifest *Manifest) {
				manifest.Components[0].PropsSchema = strings.ToUpper(manifest.Components[0].PropsSchema)
			},
		},
		{
			name: "component override key without template key",
			change: func(manifest *Manifest) {
				manifest.Templates[0].ThemeOverrideKey = ""
			},
		},
		{
			name: "template override key without component key",
			change: func(manifest *Manifest) {
				manifest.Components[0].ThemeOverrideKey = ""
			},
		},
		{
			name: "override key mismatch",
			change: func(manifest *Manifest) {
				manifest.Components[0].ThemeOverrideKey = "demo.v3.other"
			},
		},
		{
			name: "replace template is not a component fragment",
			change: func(manifest *Manifest) {
				manifest.Templates[0].Action = "replace"
				manifest.Templates[0].TargetID = "other.plugin.template.card"
			},
		},
		{
			name: "add template must not carry a target id for fragment use",
			change: func(manifest *Manifest) {
				manifest.Templates[0].TargetID = "other.plugin.template.card"
			},
		},
		{
			name: "ssr template points at non-template package file id",
			change: func(manifest *Manifest) {
				manifest.Components[0].SSRTemplate = "demo.v3.file.frontend"
			},
		},
		{
			name: "hide with mismatched ssr binding fails closed",
			change: func(manifest *Manifest) {
				manifest.Components[0].Action = ComponentActionHide
				manifest.Components[0].TargetID = "core.component.page.forum.home"
				manifest.Components[0].TargetContractVersion = "sforum.component.page.forum.home@1"
				manifest.Components[0].PropsSchema = ""
				manifest.Components[0].L2Component = ""
			},
		},
		{
			name:  "hide without ssr binding remains valid",
			valid: true,
			change: func(manifest *Manifest) {
				manifest.Components[0].Action = ComponentActionHide
				manifest.Components[0].TargetID = "core.component.page.forum.home"
				manifest.Components[0].TargetContractVersion = "sforum.component.page.forum.home@1"
				manifest.Components[0].SSRTemplate = ""
				manifest.Components[0].L2Component = ""
				manifest.Components[0].PropsSchema = ""
				manifest.Components[0].ResultSchema = ""
				manifest.Components[0].ThemeOverrideKey = ""
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := completeV3Manifest()
			test.change(&manifest)
			err := Validate(manifest)
			if test.valid && err != nil {
				t.Fatalf("expected valid component/ssr binding: %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("invalid component/ssr binding must fail closed")
			}
		})
	}
}

func TestManifestV3ComponentSSRTemplateJSONSchemaBoundary(t *testing.T) {
	// JSON Schema 只约束结构；跨数组一致性由语义 Validate 在规范化后 fail closed。
	component := []byte(`[{"id":"demo.v3.component.card","contractVersion":"demo.v3.component.card@1","action":"add","ssrTemplate":"demo.v3.template.card","propsSchema":"demo.v3.component.card.props@1","themeOverrideKey":"demo.v3.card"}]`)
	if err := validateV3JSONSchemaFragment(component, "components"); err != nil {
		t.Fatalf("component fragment should satisfy JSON Schema: %v", err)
	}
	template := []byte(`[{"id":"demo.v3.template.card","contractVersion":"demo.v3.template.card@1","action":"add","path":"templates/card.html","digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","viewModelSchema":"demo.v3.component.card.props@1","themeOverrideKey":"demo.v3.card"}]`)
	if err := validateV3JSONSchemaFragment(template, "templates"); err != nil {
		t.Fatalf("template fragment should satisfy JSON Schema: %v", err)
	}

	// 结构合法但语义漂移的 props/viewModel 仍通过 JSON Schema，由 Validate 拒绝。
	drifted := []byte(`[{"id":"demo.v3.template.card","contractVersion":"demo.v3.template.card@1","action":"add","path":"templates/card.html","digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","viewModelSchema":"demo.v3.template.card.model@1","themeOverrideKey":"demo.v3.card"}]`)
	if err := validateV3JSONSchemaFragment(drifted, "templates"); err != nil {
		t.Fatalf("structurally valid drifted template should still pass JSON Schema: %v", err)
	}
	manifest := completeV3Manifest()
	manifest.Templates[0].ViewModelSchema = "demo.v3.template.card.model@1"
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("canonical body with schema drift should still be structurally valid: %v", err)
	}
	if err := Validate(manifest); err == nil {
		t.Fatal("semantic Validate must reject component/template schema drift")
	}
}

func TestManifestV3ComponentSSRTemplateIncludeShards(t *testing.T) {
	manifest := completeV3Manifest()
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		t.Fatal(err)
	}
	files := FileMapFS{}
	for _, file := range manifest.PackageFiles {
		files[file.Path] = v3FixtureBody()
	}
	includes := map[string]string{
		"components": "manifest/components.json",
		"templates":  "manifest/templates.json",
	}
	files["manifest/components.json"] = root["components"]
	files["manifest/templates.json"] = root["templates"]
	delete(root, "components")
	delete(root, "templates")
	root["includes"], err = json.Marshal(includes)
	if err != nil {
		t.Fatal(err)
	}
	files[ManifestFileName], err = json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadPackageFS(files)
	if err != nil {
		t.Fatalf("sharded consistent component/ssr binding should load: %v", err)
	}
	if len(loaded.Components) != 1 || loaded.Components[0].SSRTemplate != "demo.v3.template.card" ||
		loaded.Components[0].PropsSchema != loaded.Templates[0].ViewModelSchema ||
		loaded.Components[0].ThemeOverrideKey != loaded.Templates[0].ThemeOverrideKey {
		t.Fatalf("sharded component/ssr contract = components=%#v templates=%#v", loaded.Components, loaded.Templates)
	}

	// include 分片内 schema 漂移必须在合并规范化后 fail closed。
	var templates []ManifestTemplate
	if err := json.Unmarshal(files["manifest/templates.json"], &templates); err != nil {
		t.Fatal(err)
	}
	templates[0].ViewModelSchema = "demo.v3.template.card.model@1"
	files["manifest/templates.json"], err = json.Marshal(templates)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPackageFS(files); err == nil {
		t.Fatal("include shard schema drift must fail closed")
	}
}

func TestManifestV3ComponentSSRTemplateNormalizationBeforeValidation(t *testing.T) {
	manifest := completeV3Manifest()
	manifest.Components[0].SSRTemplate = " DEMO.V3.TEMPLATE.CARD "
	manifest.Components[0].ThemeOverrideKey = " DEMO.V3.CARD "
	manifest.Templates[0].ThemeOverrideKey = " demo.v3.card "
	manifest.Components[0].PropsSchema = "  demo.v3.component.card.props@1  "
	manifest.Templates[0].ViewModelSchema = "demo.v3.component.card.props@1"
	normalized := Normalize(manifest)
	if normalized.Components[0].SSRTemplate != "demo.v3.template.card" ||
		normalized.Components[0].ThemeOverrideKey != "demo.v3.card" ||
		normalized.Templates[0].ThemeOverrideKey != "demo.v3.card" ||
		normalized.Components[0].PropsSchema != "demo.v3.component.card.props@1" {
		t.Fatalf("component/ssr fields were not normalized: component=%#v template=%#v",
			normalized.Components[0], normalized.Templates[0])
	}
	if err := Validate(normalized); err != nil {
		t.Fatalf("normalized component/ssr binding should validate: %v", err)
	}
}

package extensionmanifest

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func aiToolDeclaration() ManifestAITool {
	return ManifestAITool{
		ID:              "demo.v3.ai.order_lookup",
		ContractVersion: "demo.v3.ai.order_lookup@1",
		Name:            "demo-order-lookup",
		Description:     "Look up an order by its public number.",
		Parameters: json.RawMessage(
			`{"type":"object","properties":{"orderNumber":{"type":"string"}},"required":["orderNumber"]}`,
		),
		Handler:   "demo.v3.ai.order_lookup",
		TimeoutMS: 1000,
	}
}

func aiToolManifest(declarations ...ManifestAITool) Manifest {
	manifest := completeV3Manifest()
	manifest.AITools = declarations
	return manifest
}

// 通过校验的声明必须能被登记表接受：清单侧与运行期共用同一份边界，否则会出现
// 「能声明、不能执行」或「声明被拒、执行却允许」的规则分裂。
func TestManifestV3AIToolsAccepted(t *testing.T) {
	manifest := aiToolManifest(aiToolDeclaration())
	if err := Validate(manifest); err != nil {
		t.Fatalf("declared AI tool should validate: %v", err)
	}
	normalized := Normalize(aiToolManifest(ManifestAITool{
		ID:              " demo.v3.ai.order_lookup ",
		ContractVersion: "demo.v3.ai.order_lookup@1 ",
		Name:            " demo-order-lookup ",
		Description:     " Look up an order. ",
		Parameters:      json.RawMessage(`{"type":"object"}`),
		Handler:         " demo.v3.ai.order_lookup ",
		TimeoutMS:       1000,
	})).AITools[0]
	if normalized.ID != "demo.v3.ai.order_lookup" || normalized.Name != "demo-order-lookup" ||
		normalized.Handler != "demo.v3.ai.order_lookup" || normalized.Description != "Look up an order." {
		t.Fatalf("normalized declaration drifted: %#v", normalized)
	}
}

func TestManifestV3AIToolsRejected(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{"theme cannot contribute tools", func(m *Manifest) { m.Type = TypeTheme }},
		{"protocol must be v2", func(m *Manifest) { m.Backend.ProtocolVersion = 1 }},
		{"backend entry required", func(m *Manifest) { m.Backend.Entry = "" }},
		{"name with dot", func(m *Manifest) { m.AITools[0].Name = "demo.order" }},
		{"uppercase name", func(m *Manifest) { m.AITools[0].Name = "Demo-Order" }},
		{"empty name", func(m *Manifest) { m.AITools[0].Name = "" }},
		{"name too long", func(m *Manifest) { m.AITools[0].Name = strings.Repeat("a", AIToolNameMaximumLength+1) }},
		{"description required", func(m *Manifest) { m.AITools[0].Description = "   " }},
		{"description too long", func(m *Manifest) {
			m.AITools[0].Description = strings.Repeat("x", AIToolDescriptionMaximumBytes+1)
		}},
		{"parameters must be an object", func(m *Manifest) { m.AITools[0].Parameters = json.RawMessage(`["x"]`) }},
		{"parameters must be valid json", func(m *Manifest) { m.AITools[0].Parameters = json.RawMessage(`{"type":`) }},
		{"parameters required", func(m *Manifest) { m.AITools[0].Parameters = nil }},
		{"parameters too large", func(m *Manifest) {
			m.AITools[0].Parameters = json.RawMessage(`{"pad":"` + strings.Repeat("x", AIToolSchemaMaximumBytes) + `"}`)
		}},
		{"timeout required", func(m *Manifest) { m.AITools[0].TimeoutMS = 0 }},
		{"timeout above cap", func(m *Manifest) { m.AITools[0].TimeoutMS = AIToolMaximumTimeoutMS + 1 }},
		{"handler must be owned", func(m *Manifest) { m.AITools[0].Handler = "other.v3.ai.handler" }},
		{"handler must be namespaced", func(m *Manifest) { m.AITools[0].Handler = "ai.handler" }},
		{"id must be owned", func(m *Manifest) { m.AITools[0].ID = "other.v3.ai.order_lookup" }},
		{"contract version must match id", func(m *Manifest) {
			m.AITools[0].ContractVersion = "demo.v3.ai.order_lookup@2"
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			manifest := aiToolManifest(aiToolDeclaration())
			testCase.mutate(&manifest)
			if err := Validate(manifest); err == nil {
				t.Fatalf("%s should be rejected", testCase.name)
			}
		})
	}
}

func TestManifestV3AIToolsRejectsDuplicateNamesAndExcessDeclarations(t *testing.T) {
	first := aiToolDeclaration()
	second := aiToolDeclaration()
	second.ID = "demo.v3.ai.order_export"
	second.ContractVersion = "demo.v3.ai.order_export@1"
	second.Handler = "demo.v3.ai.order_export"
	if err := Validate(aiToolManifest(first, second)); err == nil {
		t.Fatal("duplicate tool names must be rejected: the model cannot disambiguate them")
	}

	declarations := make([]ManifestAITool, 0, AIToolsMaximum+1)
	for index := 0; index <= AIToolsMaximum; index++ {
		declaration := aiToolDeclaration()
		declaration.ID = fmt.Sprintf("demo.v3.ai.tool_%d", index)
		declaration.ContractVersion = declaration.ID + "@1"
		declaration.Name = fmt.Sprintf("demo-tool-%d", index)
		declaration.Handler = declaration.ID
		declarations = append(declarations, declaration)
	}
	if err := Validate(aiToolManifest(declarations...)); err == nil {
		t.Fatalf("more than %d declarations must be rejected", AIToolsMaximum)
	}
}

// 工具超时不做隐式默认：声明里的值是运营者看到的上界，缺省补一个值会让「清单里
// 写的」与「实际生效的」变成两份事实。
func TestManifestV3AIToolTimeoutIsNeverDefaulted(t *testing.T) {
	manifest := aiToolManifest(aiToolDeclaration())
	manifest.AITools[0].TimeoutMS = 0
	if normalized := Normalize(manifest); normalized.AITools[0].TimeoutMS != 0 {
		t.Fatalf("timeout must not be defaulted, got %d", normalized.AITools[0].TimeoutMS)
	}
}

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

// 会话事实是「声明即授权」：缺省只给 locale，viewer 是一次身份披露，必须由
// 作者显式写下；大小写写错应当被拒绝，而不是被宿主悄悄改写。
func TestManifestV3AIToolsContextDeclaration(t *testing.T) {
	declaration := aiToolDeclaration()
	if got := declaration.AIToolContext(); len(got) != 1 || got[0] != AIToolContextLocale {
		t.Fatalf("default context = %#v，缺省必须只含 locale", got)
	}
	if declaration.DeclaresAIToolContext(AIToolContextViewer) {
		t.Fatal("viewer must not be disclosed without an explicit declaration")
	}

	declaration.Context = []string{AIToolContextLocale, AIToolContextTopic}
	if err := Validate(aiToolManifest(declaration)); err != nil {
		t.Fatalf("declared context must validate: %v", err)
	}
	if !declaration.DeclaresAIToolContext(AIToolContextTopic) {
		t.Fatal("declared topic must be reported")
	}

	cases := map[string][]string{
		"unknown fact": {"ip"},
		"wrong case":   {"Locale"},
		"duplicate":    {AIToolContextLocale, AIToolContextLocale},
		"empty string": {""},
	}
	for name, context := range cases {
		t.Run(name, func(t *testing.T) {
			manifest := aiToolManifest(aiToolDeclaration())
			manifest.AITools[0].Context = context
			if err := Validate(manifest); err == nil {
				t.Fatalf("%s must be rejected", name)
			}
		})
	}
}

// aiToolDescriptionI18nAt 返回 count 个已规范化 locale 的展示文案，用来压条数边界。
func aiToolDescriptionI18nAt(count int) map[string]string {
	locales := []string{"en", "en-US", "en-GB", "zh-CN", "zh-TW", "ja", "de", "fr", "es"}
	values := make(map[string]string, count)
	for index := 0; index < count && index < len(locales); index++ {
		values[locales[index]] = fmt.Sprintf("Localized description %d", index)
	}
	return values
}

// descriptionI18n 只影响控制台展示：它不能替换模型可见的 description，也不能
// 突破 schema 与 Go 规则中任何一边的边界。
func TestManifestV3AIToolDescriptionI18nAccepted(t *testing.T) {
	declaration := aiToolDeclaration()
	declaration.DescriptionI18n = map[string]string{
		"en-US": "Look up an order by its public number.",
		"zh-CN": "按公开订单号查询订单。",
		"ja":    "公開注文番号で注文を検索します。",
	}
	manifest := aiToolManifest(declaration)
	if err := Validate(manifest); err != nil {
		t.Fatalf("localized descriptions should validate: %v", err)
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("schema and Go rules must accept the same map: %v", err)
	}
	if manifest.AITools[0].Description != declaration.Description {
		t.Fatal("localized text must not replace the model-facing description")
	}

	atLimit := aiToolDeclaration()
	atLimit.DescriptionI18n = aiToolDescriptionI18nAt(AIToolDescriptionI18nMaximum)
	if err := Validate(aiToolManifest(atLimit)); err != nil {
		t.Fatalf("%d locales should validate: %v", AIToolDescriptionI18nMaximum, err)
	}
}

// 键必须是已规范化的 locale：Normalize 不折叠大小写与分隔符，所以这些形状会
// 原样走到规则上并被拒绝，而不是被悄悄改写。
func TestManifestV3AIToolDescriptionI18nRejected(t *testing.T) {
	cases := map[string]map[string]string{
		"value too long":   {"en-US": strings.Repeat("x", AIToolDescriptionMaximumBytes+1)},
		"too many locales": aiToolDescriptionI18nAt(AIToolDescriptionI18nMaximum + 1),
		"underscore key":   {"zh_cn": "查询订单。"},
		"lowercase region": {"zh-cn": "查询订单。"},
		"padded key":       {" zh-CN ": "查询订单。"},
	}
	for name, values := range cases {
		t.Run(name, func(t *testing.T) {
			manifest := aiToolManifest(aiToolDeclaration())
			manifest.AITools[0].DescriptionI18n = values
			if err := Validate(manifest); err == nil {
				t.Fatalf("%s must be rejected", name)
			}
		})
	}
}

// Normalize 会丢弃空文案条目，因此「空值被拒绝」这条规则只能在规范化之前验证：
// 直接调用校验器，确认规则本身对原始输入成立，而不是依赖调用顺序。
func TestManifestV3AIToolDescriptionI18nRejectsEmptyText(t *testing.T) {
	cases := map[string]map[string]string{
		"empty value":      {"en-US": ""},
		"whitespace value": {"en-US": "   "},
		"blank key":        {"  ": "查询订单。"},
	}
	for name, values := range cases {
		t.Run(name, func(t *testing.T) {
			manifest := aiToolManifest(aiToolDeclaration())
			manifest.AITools[0].DescriptionI18n = values
			validator := v3Validator{manifest: manifest, ids: map[string]string{}}
			err := validator.validateAITools()
			if err == nil {
				t.Fatalf("%s must be rejected before normalization", name)
			}
			if !strings.Contains(err.Error(), "v3 aitools rule #14") {
				t.Fatalf("descriptionI18n must fail rule #14, got %v", err)
			}
		})
	}
}

func TestManifestV3AIToolDescriptionI18nNormalized(t *testing.T) {
	declaration := aiToolDeclaration()
	declaration.DescriptionI18n = map[string]string{
		"  ":    "空白键没有展示意义",
		"en-US": "  Look up an order.  ",
		"zh-CN": "   ",
		"ja":    "注文を検索します。",
	}
	normalized := Normalize(aiToolManifest(declaration)).AITools[0].DescriptionI18n
	if len(normalized) != 2 {
		t.Fatalf("normalized locales = %#v, want 2 entries", normalized)
	}
	if normalized["en-US"] != "Look up an order." || normalized["ja"] != "注文を検索します。" {
		t.Fatalf("normalized locales drifted: %#v", normalized)
	}

	empty := aiToolDeclaration()
	empty.DescriptionI18n = map[string]string{}
	if got := Normalize(aiToolManifest(empty)).AITools[0].DescriptionI18n; got != nil {
		t.Fatalf("empty map must normalize to nil, got %#v", got)
	}
}

// 旧清单（没有 descriptionI18n）的规范形必须逐字节不变：protocolRuntimeManifestDigest
// 直接对 canonical manifest 做 SHA-256，多出一个空字段就会让每个旧插件的运行时
// 摘要漂移。空 map 与空文案同理——它们规范化后与「没写」完全一致。
func TestManifestV3AIToolDescriptionI18nKeepsLegacyCanonicalForm(t *testing.T) {
	legacy := Normalize(aiToolManifest(aiToolDeclaration()))
	if legacy.AITools[0].DescriptionI18n != nil {
		t.Fatalf("legacy declaration must not gain a locale map: %#v", legacy.AITools[0].DescriptionI18n)
	}
	if err := Validate(aiToolManifest(aiToolDeclaration())); err != nil {
		t.Fatalf("legacy declaration must still validate: %v", err)
	}
	body, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "descriptionI18n") {
		t.Fatalf("legacy canonical manifest must not mention descriptionI18n: %s", body)
	}

	blank := aiToolDeclaration()
	blank.DescriptionI18n = map[string]string{"en-US": "   "}
	blankBody, err := json.Marshal(Normalize(aiToolManifest(blank)))
	if err != nil {
		t.Fatal(err)
	}
	if string(blankBody) != string(body) {
		t.Fatalf("blank localized text must canonicalize like an absent field:\n%s\n%s", blankBody, body)
	}
}

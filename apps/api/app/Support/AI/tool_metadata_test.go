package ai

// 参数摘要只在控制台展示，但它是运营者判断「这个工具要传什么」的唯一来源，
// 因此必须是确定性的：排序、截断、非法 schema 的处理都不能依赖 map 迭代顺序。
// summarizeToolParameters 不导出，所以这份用例用同包测试。

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestToolParameterSummarySortsAndFiltersRequired(t *testing.T) {
	summary := summarizeToolParameters(json.RawMessage(`{
		"type": "object",
		"properties": {"query": {"type": "string"}, "limit": {"type": "integer"}, "since": {"type": "string"}},
		"required": ["since", "query", "undeclared"]
	}`))
	if summary == nil {
		t.Fatal("schema with properties must produce a summary")
	}
	if want := []string{"limit", "query", "since"}; !slices.Equal(summary.Names, want) {
		t.Fatalf("names = %#v, want %#v", summary.Names, want)
	}
	// undeclared 在 properties 里不存在：非法 schema 的必填项不进摘要，否则
	// 运营者会看到一个列表里没有的名字。
	if want := []string{"query", "since"}; !slices.Equal(summary.Required, want) {
		t.Fatalf("required = %#v, want %#v", summary.Required, want)
	}
	if summary.Truncated {
		t.Fatal("three parameters must not be marked truncated")
	}
}

func TestToolParameterSummaryOmitsAbsentRequired(t *testing.T) {
	summary := summarizeToolParameters(json.RawMessage(`{"properties":{"query":{"type":"string"}}}`))
	if summary == nil {
		t.Fatal("schema with properties must produce a summary")
	}
	if len(summary.Required) != 0 {
		t.Fatalf("required must be empty when the schema declares none: %#v", summary.Required)
	}
}

// 用常量而不是写死 24：上限调整时这条用例仍覆盖边界，不会静默失去覆盖。
func TestToolParameterSummaryMarksTruncation(t *testing.T) {
	build := func(count int) json.RawMessage {
		properties := make(map[string]json.RawMessage, count)
		for i := 0; i < count; i++ {
			properties[fmt.Sprintf("p%02d", i)] = json.RawMessage(`{"type":"string"}`)
		}
		raw, err := json.Marshal(map[string]any{"type": "object", "properties": properties})
		if err != nil {
			t.Fatalf("marshal schema: %v", err)
		}
		return raw
	}

	over := summarizeToolParameters(build(MaxToolParameterNames + 6))
	if over == nil {
		t.Fatal("oversized schema must still produce a summary")
	}
	if !over.Truncated {
		t.Fatalf("%d parameters must be marked truncated", MaxToolParameterNames+6)
	}
	if len(over.Names) != MaxToolParameterNames {
		t.Fatalf("names = %d, want %d", len(over.Names), MaxToolParameterNames)
	}
	// 保留的是排序后的前 N 个，不是 map 顺序里的随机 N 个。
	if over.Names[0] != "p00" || over.Names[MaxToolParameterNames-1] != fmt.Sprintf("p%02d", MaxToolParameterNames-1) {
		t.Fatalf("truncation must keep the sorted prefix: %#v", over.Names)
	}

	// 恰好等于上限时是完整的，不能标截断。
	exact := summarizeToolParameters(build(MaxToolParameterNames))
	if exact == nil || exact.Truncated || len(exact.Names) != MaxToolParameterNames {
		t.Fatalf("exact-limit schema must not be truncated: %#v", exact)
	}
}

func TestToolParameterSummaryClipsLongNamesByRune(t *testing.T) {
	longName := strings.Repeat("参", MaxToolParameterNameLen+6)
	summary := summarizeToolParameters(json.RawMessage(`{"properties":{"` + longName + `":{"type":"string"}}}`))
	if summary == nil || len(summary.Names) != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	clipped := summary.Names[0]
	if !utf8.ValidString(clipped) {
		t.Fatalf("clipping split a multi-byte character: %q", clipped)
	}
	if got := utf8.RuneCountInString(clipped); got != MaxToolParameterNameLen {
		t.Fatalf("clipped rune count = %d, want %d", got, MaxToolParameterNameLen)
	}
	if !strings.HasSuffix(clipped, "…") {
		t.Fatalf("clipped name must end with an ellipsis marker: %q", clipped)
	}
	if got := strings.Count(clipped, "参"); got != MaxToolParameterNameLen-1 {
		t.Fatalf("clipped name kept %d runes of the original, want %d", got, MaxToolParameterNameLen-1)
	}
}

// 两个长名字可能截断成同一个展示名：摘要按展示名去重，required 也按展示名对齐。
func TestToolParameterSummaryDeduplicatesClippedNames(t *testing.T) {
	prefix := strings.Repeat("a", MaxToolParameterNameLen-1)
	raw := json.RawMessage(`{"properties":{"` + prefix + `-one":{"type":"string"},"` + prefix + `-two":{"type":"string"}},"required":["` + prefix + `-two"]}`)
	summary := summarizeToolParameters(raw)
	if summary == nil {
		t.Fatal("summary must not be nil")
	}
	if len(summary.Names) != 1 {
		t.Fatalf("clipped duplicates must collapse to one name: %#v", summary.Names)
	}
	if !slices.Equal(summary.Required, summary.Names) {
		t.Fatalf("required must reference the displayed name: %#v vs %#v", summary.Required, summary.Names)
	}
}

func TestToolParameterSummaryRejectsUnusableSchemas(t *testing.T) {
	cases := map[string]string{
		"empty":                       "",
		"blank":                       "   \n\t",
		"json null":                   "null",
		"no properties key":           `{"type":"object"}`,
		"empty properties":            `{"properties":{}}`,
		"properties not an object":    `{"properties":[]}`,
		"required without properties": `{"required":["query"]}`,
		"invalid json":                `{"properties":`,
		"blank property names":        `{"properties":{"  ":{"type":"string"}}}`,
	}
	for name, raw := range cases {
		if got := summarizeToolParameters(json.RawMessage(raw)); got != nil {
			t.Fatalf("%s: expected no summary, got %#v", name, got)
		}
	}
}

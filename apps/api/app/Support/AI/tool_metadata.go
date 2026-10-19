package ai

import (
	"encoding/json"
	"sort"
	"strings"
)

// 工具的可枚举元数据：控制台要回答「这个工具是谁提供的、会不会说英文、调用它
// 要传什么、最坏占用多久」。这些事实都来自工具自身或它的清单声明，登记表只负责
// 把它们汇总成一份稳定快照。
//
// 与 ToolOriginProvider 一样，它们都是可选接口：没有实现它们的工具仍然是一个
// 合法工具，只是控制台少显示几行。这样 Core 内置工具、旧版插件工具与新版插件
// 工具可以共用同一条枚举路径。

const (
	// MaxToolParameterNames 是参数摘要里最多列出的参数名数量。它是展示上限，
	// 不是 schema 上限：完整 schema 仍由 MaxToolSchemaBytes 约束。
	MaxToolParameterNames = 24
	// MaxToolParameterNameLen 是单个参数名的展示长度上限（按 rune 计）。
	MaxToolParameterNameLen = 64
)

// ToolParameterSummary 是参数 schema 的展示摘要。它只保留运营者判断「这个工具
// 需要什么输入」所需的两项事实，不下发完整 JSON Schema。
type ToolParameterSummary struct {
	// Names 是顶层 properties 的名字，已排序。
	Names []string `json:"names"`
	// Required 是 required 里确实出现在 properties 中的名字，已排序。
	Required []string `json:"required,omitempty"`
	// Truncated 表示 properties 数量超过 MaxToolParameterNames，列表不是全部。
	Truncated bool `json:"truncated,omitempty"`
}

// ToolDisplayDescriptionProvider 是工具可选的界面本地化声明。
//
// 它只影响控制台展示：模型收到的始终是 Definition().Description（会被
// RenderToolSection 渲染进 system）。把两者分开，翻译文案才不会悄悄改变提示词、
// 缓存键与成本口径。
type ToolDisplayDescriptionProvider interface {
	DisplayDescriptions() map[string]string
}

// ToolTimeoutProvider 是插件工具声明的单次调用超时上限。Core 内置工具是进程内
// 调用，没有这一层声明，因此不实现该接口。
type ToolTimeoutProvider interface {
	TimeoutMS() int
}

// ToolSettingsPathProvider 是插件工具可选的设置页声明。只有真的声明了设置页的
// 扩展才实现它：控制台据此显示跳转图标，而不是让运营者点进一个 404。
type ToolSettingsPathProvider interface {
	SettingsPath() string
}

// summarizeToolParameters 从工具的 JSON Schema 提取展示摘要。
//
// 非法 JSON 或没有顶层 properties 时返回 nil：控制台隐藏参数行，而不是显示一份
// 猜出来的结构。schema 的语义仍归供应商与插件，宿主不做第二次解释。
func summarizeToolParameters(raw json.RawMessage) *ToolParameterSummary {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal([]byte(trimmed), &schema); err != nil {
		return nil
	}
	if len(schema.Properties) == 0 {
		return nil
	}

	rawNames := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		rawNames = append(rawNames, name)
	}
	if len(rawNames) == 0 {
		return nil
	}
	sort.Strings(rawNames)
	truncated := len(rawNames) > MaxToolParameterNames
	if truncated {
		rawNames = rawNames[:MaxToolParameterNames]
	}

	names := make([]string, 0, len(rawNames))
	present := make(map[string]struct{}, len(rawNames))
	for _, name := range rawNames {
		clipped := clipParameterName(name)
		if clipped == "" {
			continue
		}
		if _, exists := present[clipped]; exists {
			continue
		}
		present[clipped] = struct{}{}
		names = append(names, clipped)
	}
	if len(names) == 0 {
		return nil
	}

	required := make([]string, 0, len(schema.Required))
	requiredSeen := make(map[string]struct{}, len(schema.Required))
	for _, name := range schema.Required {
		name = clipParameterName(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		if _, exists := present[name]; !exists {
			// required 里出现未声明的参数是非法 schema；摘要不把它当成必填项，
			// 否则运营者会看到一个列表里不存在的名字。
			continue
		}
		if _, exists := requiredSeen[name]; exists {
			continue
		}
		requiredSeen[name] = struct{}{}
		required = append(required, name)
	}
	sort.Strings(required)

	return &ToolParameterSummary{
		Names:     names,
		Required:  required,
		Truncated: truncated,
	}
}

// clipParameterName 按 rune 截断过长的参数名，避免在多字节字符中间切断。
func clipParameterName(name string) string {
	runes := []rune(name)
	if len(runes) <= MaxToolParameterNameLen {
		return name
	}
	return string(runes[:MaxToolParameterNameLen-1]) + "…"
}

// copyStringMap 返回 map 的副本。登记表把工具的展示文案交给调用方时不能交引用：
// 控制台或测试改到它，会污染下一次枚举。
func copyStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	copied := make(map[string]string, len(source))
	for key, value := range source {
		copied[key] = value
	}
	return copied
}

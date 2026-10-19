package ai

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// ToolRegistry 是 Host 持有的工具登记表。插件贡献的工具同样登记在这里，
// 因此「站内有哪些工具」永远有一个可枚举、可审计的答案。
//
// 登记表是可替换的：插件启用/停用/升级之后，调用方用 Replace 把「Core 工具 +
// 当前已启用插件工具」的新集合整体换进来，正在进行的补全继续用旧表跑完，新的
// 补全看到新表。读写因此必须过锁——单次补全会多次读取登记表（每一步的工具声明
// 与派发都要查名字），替换发生在两次读取之间。
//
// ToolOrigin 标识一个工具来自哪里。控制台用它把 Core 工具与插件工具分开呈现，
// 运营者因此能看出「这个名字是谁提供的」。
type ToolOrigin struct {
	// Kind 是 core 或 plugin。
	Kind string `json:"kind"`
	// ExtensionID / Version 只在 Kind=plugin 时有值。
	ExtensionID string `json:"extensionId,omitempty"`
	Version     string `json:"version,omitempty"`
}

// ToolAvailability 是「站内有哪些 AI 工具」的可枚举条目。它比 ToolDefinition
// 多一个来源：模型只需要名字与描述，运营者还需要知道谁提供的。
type ToolAvailability struct {
	Name string `json:"name"`
	// Description 是模型收到的那份说明，必须与 ToolDefinition.Description 逐字节
	// 相同：它会被 RenderToolSection 渲染进 system，因此任何展示用文案都不能写
	// 进这里。
	Description string `json:"description"`
	// DescriptionI18n 是仅控制台展示的本地化说明（例如 en-US），来自插件清单
	// 或 Core 内置声明。它不进出站请求、不进缓存键、不参与提示词版本，因此新增
	// 或修改它不会改变任何一次模型调用。
	DescriptionI18n map[string]string `json:"descriptionI18n,omitempty"`
	Origin          ToolOrigin        `json:"origin"`
	// TimeoutMS 是插件声明的单次调用超时上限；Core 内置工具没有这一层声明，为 0。
	TimeoutMS int `json:"timeoutMs,omitempty"`
	// SettingsPath 是插件管理设置页的站内路径（不含 /control-panel 前缀）。
	// 只有声明了设置页的插件工具才有值；Core 内置工具为空。
	SettingsPath string `json:"settingsPath,omitempty"`
	// Parameters 是参数摘要，不是完整 JSON Schema：完整 schema 单个可达 8 KiB，
	// 整份下发会把每次控制台加载撑大。
	Parameters *ToolParameterSummary `json:"parameters,omitempty"`
}

// ToolCatalog 是登记表的只读枚举面。服务层与控制器只依赖它，因此不需要知道
// 工具是进程内函数还是插件 RPC。
type ToolCatalog interface {
	Available() []ToolAvailability
}

// ToolOriginProvider 是工具可选实现的来源声明。没有实现它的工具一律视为 Core
// 内置：插件工具必须主动说出自己的扩展身份。
type ToolOriginProvider interface {
	Origin() ToolOrigin
}

type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: map[string]Tool{}}
}

func (r *ToolRegistry) Register(tool Tool) error {
	if r == nil || tool == nil {
		return ErrToolLoopUnavailable
	}
	definition := tool.Definition()
	if !ValidToolName(definition.Name) {
		return fmt.Errorf("%w: invalid tool name %q", ErrToolLoopUnavailable, definition.Name)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.tools == nil {
		r.tools = map[string]Tool{}
	}
	if _, exists := r.tools[definition.Name]; exists {
		return fmt.Errorf("%w: duplicate tool %s", ErrToolLoopUnavailable, definition.Name)
	}
	r.tools[definition.Name] = tool
	return nil
}

// Tools 返回当前登记表的工具快照，按名字排序以保证枚举顺序稳定。
//
// 它是为了让调用方能拿住「Core 工具集」这一份不变的底子，随后用 Replace 反复
// 重建「底子 + 当前插件工具」：Core 工具是进程内对象、无状态，重建时原样复用。
func (r *ToolRegistry) Tools() []Tool {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return sortedTools(r.tools)
}

// Replace 用一批工具整体替换登记表，语义是「这批就是现在的全部工具」。
//
// 全部校验在替换之前完成：任何一条名字非法或重名都不会改动现有登记表，调用方
// 可以在失败后继续用旧表服务请求——「换表失败」与「站内没有工具」是两回事，
// 后者会让每一次回复都退化成纯文本。成功替换是原子的；已经开始的那次补全继续
// 用旧表，因为它拿的是先前那几份 Tool 值，而不是登记表本身。
func (r *ToolRegistry) Replace(tools []Tool) error {
	if r == nil {
		return ErrToolLoopUnavailable
	}
	next := make(map[string]Tool, len(tools))
	for _, tool := range tools {
		if tool == nil {
			return fmt.Errorf("%w: nil tool", ErrToolLoopUnavailable)
		}
		definition := tool.Definition()
		if !ValidToolName(definition.Name) {
			return fmt.Errorf("%w: invalid tool name %q", ErrToolLoopUnavailable, definition.Name)
		}
		if _, exists := next[definition.Name]; exists {
			return fmt.Errorf("%w: duplicate tool %s", ErrToolLoopUnavailable, definition.Name)
		}
		next[definition.Name] = tool
	}
	r.mu.Lock()
	r.tools = next
	r.mu.Unlock()
	return nil
}

// Lookup 按名字取工具；未注册时返回 ok=false。
func (r *ToolRegistry) Lookup(name string) (Tool, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[strings.TrimSpace(name)]
	return tool, ok
}

// Descriptors 返回白名单内已登记工具的声明，顺序与白名单一致，并受
// MaxToolsPerRequest 约束。未登记的名字被跳过而不是报错：插件卸载后，
// 历史白名单仍应该能跑，而不是把整个用途拖死。
func (r *ToolRegistry) Descriptors(names []string) []ToolDefinition {
	if r == nil || len(names) == 0 {
		return nil
	}
	descriptors := make([]ToolDefinition, 0, len(names))
	for _, name := range names {
		if len(descriptors) >= MaxToolsPerRequest {
			break
		}
		tool, ok := r.Lookup(name)
		if !ok {
			continue
		}
		descriptors = append(descriptors, tool.Definition())
	}
	return descriptors
}

// Available 返回全部已登记工具的枚举视图，按名字排序以保证控制台顺序稳定。
//
// 它只回答「站内有哪些工具」，不回答「模型能不能用」：可见性由用途白名单决定
// （RunInput.ToolNames）。两者刻意分开——否则运营者在控制台看到的清单会随着
// 每个用途的覆盖配置变化，而「站内有哪些工具」本该是一个全局事实。
//
// 四项可选元数据（来源、展示文案、超时、参数摘要）都用类型断言按需读取：没有
// 实现某个接口的工具仍然是一个合法工具，只是控制台少显示几行。Description 保持
// 原样，它是模型收到的那一份，任何展示文案都不得覆盖它。
func (r *ToolRegistry) Available() []ToolAvailability {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.tools) == 0 {
		return nil
	}
	names := sortedToolNames(r.tools)
	out := make([]ToolAvailability, 0, len(names))
	for _, name := range names {
		tool := r.tools[name]
		definition := tool.Definition()
		origin := ToolOrigin{Kind: "core"}
		if provider, ok := tool.(ToolOriginProvider); ok {
			if declared := provider.Origin(); declared.Kind != "" {
				origin = declared
			}
		}
		entry := ToolAvailability{
			Name:        definition.Name,
			Description: definition.Description,
			Origin:      origin,
			Parameters:  summarizeToolParameters(definition.Parameters),
		}
		// 展示文案必须拷贝：交出去的是调用方的副本，改它不会污染下一次枚举。
		if provider, ok := tool.(ToolDisplayDescriptionProvider); ok {
			entry.DescriptionI18n = copyStringMap(provider.DisplayDescriptions())
		}
		if provider, ok := tool.(ToolTimeoutProvider); ok {
			entry.TimeoutMS = provider.TimeoutMS()
		}
		if provider, ok := tool.(ToolSettingsPathProvider); ok {
			entry.SettingsPath = strings.TrimSpace(provider.SettingsPath())
		}
		out = append(out, entry)
	}
	return out
}

// sortedToolNames 返回登记表里工具名的稳定顺序（按名字升序）。
func sortedToolNames(tools map[string]Tool) []string {
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// sortedTools 按名字升序返回工具值，供 Replace 的调用方重建集合。
func sortedTools(tools map[string]Tool) []Tool {
	if len(tools) == 0 {
		return nil
	}
	names := sortedToolNames(tools)
	out := make([]Tool, 0, len(names))
	for _, name := range names {
		out = append(out, tools[name])
	}
	return out
}

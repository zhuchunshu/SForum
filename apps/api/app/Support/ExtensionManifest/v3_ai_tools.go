package extensionmanifest

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AIToolMaximumTimeoutMS 与 SDK 侧 pluginv2.HostAIToolMaximumTimeoutMS 对齐：
// 单次插件工具调用的超时上限，声明侧与执行侧共用同一量纲，避免清单允许
// 而执行层拒绝（或反之）的漂移。
//
// 30 秒是一个**声明上限**，不是一次必然的等待：宿主按声明值设置截止时间，插件
// 自己的预算（例如搜索插件的 timeout_ms，默认 4 秒）决定实际等多久。上限存在的
// 意义是给「慢但合理」的工具留出空间（冷启动检索、文档解析），同时保留一条
// 绝对边界——工具调用占的是 AI 回复 worker 的时间，而 AI 队列并发默认只有
// 2（生产）/1（开发），一次过长的调用足以让全站回复排队。
//
// 历史：5000 → 180000 → 30000。5000 是 SDK 传输默认请求超时被直接推上来的值，
// 对冷启动后端过紧；180 秒在没有站点级工具预算闸门时又太大（运营者明确不加闸门，
// 于是「一次回复最多等多久」只剩插件声明值这一个上界），30 秒是折中：
// 比一次模型调用（profile 默认 30 秒）同量级，慢工具够用，又不至于拖垮队列。
const AIToolMaximumTimeoutMS = 30_000

// AIToolsMaximum 是单个插件可贡献的 AI 工具数上限，与一次补全请求的工具数
// 上限量级对齐但语义不同：一个插件最多贡献 16 个工具。
const AIToolsMaximum = 16

// AIToolSchemaMaximumBytes 是单个工具 Parameters JSON Schema 的字节上限，
// 与执行侧 supportai.MaxToolSchemaBytes 保持同一量纲。
const AIToolSchemaMaximumBytes = 8_192

// AIToolDescriptionMaximumBytes 是工具描述的字节上限，与执行侧
// supportai.MaxToolDescriptionBytes 保持同一量纲。
const AIToolDescriptionMaximumBytes = 1_024

// AIToolNameMaximumLength 是工具名的字符上限，与执行侧
// supportai.MaxToolNameLen 保持同一量纲。
const AIToolNameMaximumLength = 64

// AIToolDescriptionI18nMaximum 是单个工具可声明的本地化说明条数上限，与
// schemas/manifest-v3.schema.json 的 maxProperties 保持同一量纲：本地化说明是
// 控制台展示用的短文本，条数上限只需覆盖站点实际支持的语言数量级。
const AIToolDescriptionI18nMaximum = 8

// AI 工具可以声明的会话事实。声明是**白名单**：没列出来的事实不会被传进
// 插件进程，因此「工具能知道现在是谁在问」必须是作者写下的显式决定。
const (
	// AIToolContextLocale 传站点默认语言（BCP-47，例如 zh-CN）。
	AIToolContextLocale = "locale"
	// AIToolContextTopic 传本次对话所在主题 id（0 表示不限定位置）。
	AIToolContextTopic = "topic"
	// AIToolContextViewer 传提问者用户 id。它是一次身份披露，因此默认不传。
	AIToolContextViewer = "viewer"

	// AIToolContextMaximum 与这三项的数量一致：新增事实必须同时改这里。
	AIToolContextMaximum = 3
)

// ManifestAITool 是清单里的一条插件 AI 工具声明。它是静态事实：宿主在插件
// 启动前就能知道有哪些工具，不需要运行期询问插件。
type ManifestAITool struct {
	// ID 是声明的稳定标识，必须落在所属插件的命名空间下。
	ID string `json:"id"`
	// ContractVersion 是声明的契约版本，参数或结果形状变更时递增。
	ContractVersion string `json:"contractVersion"`
	// Name 是模型看到的工具名：只允许 [a-z0-9_-]，且不含点号。
	Name string `json:"name"`
	// Description 决定模型何时调用该工具。它是模型输入，不是界面文案。
	Description string `json:"description"`
	// DescriptionI18n 是仅控制台展示的本地化说明（locale -> 文本）。
	//
	// 它刻意不是 LocalizedText：后者带自定义 MarshalJSON，而 Go 的 omitempty 对
	// struct 字段不生效，零值会被序列化成 ""，让每个旧插件的 canonical manifest
	// 多出一个字段、manifest 摘要随之漂移。map 的 omitempty 正常工作，旧清单的
	// 规范形与摘要因此逐字节不变。
	DescriptionI18n map[string]string `json:"descriptionI18n,omitempty"`
	// Parameters 是 JSON Schema 对象（不是数组、不是裸 true）。
	Parameters json.RawMessage `json:"parameters,omitempty"`
	// Handler 是插件侧的可执行 handler，必须落在所属插件的命名空间下。
	Handler string `json:"handler"`
	// TimeoutMS 是单次调用的超时上限（1..AIToolMaximumTimeoutMS）。
	TimeoutMS int `json:"timeoutMs"`
	// Context 声明该工具需要哪些会话事实（locale / topic / viewer）。缺省表示
	// 只要 locale：按语言检索是最常见、披露最少的一项。
	Context []string `json:"context,omitempty"`
}

// AIToolContext 返回该工具实际生效的会话事实集合。缺省是「只要 locale」，
// 因为「按提问语言检索」是绝大多数工具都需要、且不涉及身份披露的那一项。
func (d ManifestAITool) AIToolContext() []string {
	if len(d.Context) == 0 {
		return []string{AIToolContextLocale}
	}
	return d.Context
}

// DeclaresAIToolContext 报告该工具是否声明了某一项会话事实。
func (d ManifestAITool) DeclaresAIToolContext(fact string) bool {
	for _, declared := range d.AIToolContext() {
		if declared == fact {
			return true
		}
	}
	return false
}

// ValidateAIToolContext 校验声明的会话事实集合：只允许已定义的三项、不得重复、
// 数量不超过上限。空列表合法（等价于缺省）。
//
// 这里刻意不做规范化（不 trim、不转小写）：清单是作者写下的显式事实，一个大小写
// 写错的值应该被拒绝，而不是被宿主悄悄改写成另一个意思。
func ValidateAIToolContext(values []string) error {
	if len(values) > AIToolContextMaximum {
		return fmt.Errorf("%w: v3 aitools rule #13", ErrInvalidManifest)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		switch value {
		case AIToolContextLocale, AIToolContextTopic, AIToolContextViewer:
		default:
			return fmt.Errorf("%w: v3 aitools rule #13", ErrInvalidManifest)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("%w: v3 aitools rule #13", ErrInvalidManifest)
		}
		seen[value] = struct{}{}
	}
	return nil
}

// validateAIToolDescriptionI18n 校验仅用于控制台展示的本地化说明。字段缺失时
// 零影响：旧清单没有这个字段，规范化与摘要都必须逐字节不变。
//
// 键必须是**已规范化**的 locale（等于 normalizeLocaleKey 的结果）。这里刻意不
// 做规范化，与 ValidateAIToolContext 同一条理由，并且多一层：把 " zh-CN " 与
// "zh-CN" 折叠到同一个槽位会让两个值争夺同一个键，而 canonical manifest 的字节
// 与摘要取决于最终留下的那一个——那是一个不确定的摘要。作者写错的事实被拒绝，
// 而不是被宿主悄悄改写成另一个意思。
func validateAIToolDescriptionI18n(values map[string]string) error {
	if len(values) == 0 {
		return nil
	}
	if len(values) > AIToolDescriptionI18nMaximum {
		return fmt.Errorf("%w: v3 aitools rule #14", ErrInvalidManifest)
	}
	for key, value := range values {
		if key == "" || normalizeLocaleKey(key) != key {
			return fmt.Errorf("%w: v3 aitools rule #14", ErrInvalidManifest)
		}
		text := strings.TrimSpace(value)
		if text == "" || len(text) > AIToolDescriptionMaximumBytes {
			return fmt.Errorf("%w: v3 aitools rule #14", ErrInvalidManifest)
		}
	}
	return nil
}

// normalizeAIToolDescriptionI18n 修剪展示文案两侧空白，并丢弃没有展示意义的条目。
//
// 键只判空、不做大小写或分隔符改写：改写会让 " zh-CN " 与 "zh-CN" 折叠成同一个
// 键，两个值中谁留下取决于 map 遍历顺序，规范形因此不再确定。键的形状由
// validateAIToolDescriptionI18n 拒绝，而不是被规范化悄悄修好。
//
// 全空（含「写了空 map」）时返回 nil，让旧清单与空声明的序列化结果一致。
func normalizeAIToolDescriptionI18n(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	normalized := make(map[string]string, len(values))
	for key, value := range values {
		if strings.TrimSpace(key) == "" {
			continue
		}
		text := strings.TrimSpace(value)
		if text == "" {
			continue
		}
		normalized[key] = text
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

// IsOwnedID 判断一个标识是否严格落在 owner 命名空间下：以 `<owner>.` 开头，
// 且不是 owner 本身。清单 ID、handler 等插件命名空间资源统一用它校验，
// 防止一个插件声明另一个插件的资源。
func IsOwnedID(owner, id string) bool {
	owner = strings.TrimSpace(owner)
	id = strings.TrimSpace(id)
	if owner == "" || id == "" || id == owner {
		return false
	}
	return strings.HasPrefix(id, owner+".") && len(id) > len(owner)+1
}

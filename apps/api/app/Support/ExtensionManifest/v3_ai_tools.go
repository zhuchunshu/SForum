package extensionmanifest

import (
	"encoding/json"
	"strings"
)

// AIToolMaximumTimeoutMS 与 SDK 侧 pluginv2.HostAIToolMaximumTimeoutMS 对齐：
// 单次插件工具调用的超时上限，声明侧与执行侧共用同一量纲，避免清单允许
// 而执行层拒绝（或反之）的漂移。
const AIToolMaximumTimeoutMS = 5000

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

// ManifestAITool 是清单里的一条插件 AI 工具声明。它是静态事实：宿主在插件
// 启动前就能知道有哪些工具，不需要运行期询问插件。
type ManifestAITool struct {
	// ID 是声明的稳定标识，必须落在所属插件的命名空间下。
	ID string `json:"id"`
	// ContractVersion 是声明的契约版本，参数或结果形状变更时递增。
	ContractVersion string `json:"contractVersion"`
	// Name 是模型看到的工具名：只允许 [a-z0-9_-]，且不含点号。
	Name string `json:"name"`
	// Description 决定模型何时调用该工具。
	Description string `json:"description"`
	// Parameters 是 JSON Schema 对象（不是数组、不是裸 true）。
	Parameters json.RawMessage `json:"parameters,omitempty"`
	// Handler 是插件侧的可执行 handler，必须落在所属插件的命名空间下。
	Handler string `json:"handler"`
	// TimeoutMS 是单次调用的超时上限（1..AIToolMaximumTimeoutMS）。
	TimeoutMS int `json:"timeoutMs"`
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

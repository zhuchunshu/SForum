// Package ai 是 Host 的 AI 网关契约层：中立补全契约、provider profile、
// 三层配置与成本闸门。供应商调用由插件实现（ai.provider 槽位），Core 只拥有
// 契约、协议翻译、记账、限流与审计。
package ai

// Host 硬天花板。Settings 与 Profile 中的可调数值都必须落在这些边界内，
// 越界直接拒绝写入，避免运维调参意外移除进程安全边界。
// 该约定沿用 Support/ContentRegistry 的 ExecutionLimits 模式。
//
// 这些数值是「运营者最多能把单次调用设多大」的防呆边界，不是产品天花板：
// 防滥用由预算/速率/配额三道闸门负责（见 gate.go），默认值也保持保守。因此
// 上限应当覆盖主流模型的能力（1M 上下文、128k 输出），而不是把模型按住不用。
// 放宽或收紧都必须同步 sdk/plugin/v2 的镜像常量与 gRPC 传输上限，见
// knowledge/decisions/2026-10-19-ai-context-and-budget-ceilings.md。
const (
	// 单次 provider 调用超时。
	MinTimeoutMS     = 1_000
	MaxTimeoutMS     = 120_000
	DefaultTimeoutMS = 30_000

	// 单次补全输出上限。128k 覆盖当前长输出模型（Claude 64k / GPT-5 128k /
	// Gemini 65k）；默认值仍保守，因为默认值直接决定每次调用的成本。
	MaxCompletionTokens        = 128_000
	DefaultMaxCompletionTokens = 4_096

	// 输入规模上限。长度按 UTF-8 字节计，中文按字节数计入。
	// MaxTotalInputBytes 与插件 gRPC 单消息上限同源（sdk/plugin/v2 的
	// DefaultMaxMessageBytes），否则会出现「Host 放行、传输层拒绝」的分裂。
	MaxMessages        = 256
	MaxMessageBytes    = 1 << 20
	MaxSystemBytes     = 1 << 20
	MaxTotalInputBytes = 12 << 20

	// profile 基数与标识长度。
	MaxProfiles     = 32
	MaxProfileIDLen = 64
	MaxModelLen     = 128
	MaxLabelLen     = 64

	// 用途标识长度。用途 id 进入执行记录与用量归属，并与 provider profile 的
	// 命名空间分离，因此允许点号（forum.reply）。
	MaxPurposeIDLen = 64

	// 工具声明上限。工具声明会随每次请求进入供应商输入，因此在契约层设死上限，
	// 避免一个用途把整站上下文成本推高到无法解释。
	MaxToolsPerRequest      = 16
	MaxToolNameLen          = 64
	MaxToolDescriptionBytes = 1_024
	MaxToolSchemaBytes      = 8_192
	MaxToolArgumentsBytes   = 16_384
	MaxToolCallIDLen        = 128

	// 控制台展示文案的上限。toolDisplayDescriptions 只服务界面，但它是整份设置
	// 文档的一部分，仍要有边界：一个失控的字段不该把 GET /admin/ai/settings 撑大。
	MaxToolDisplayTools   = 64
	MaxToolDisplayLocales = 8

	// 单次回复允许的工具调用次数（步数预算）。默认 5 是「搜索 → 读帖 → 再搜索 →
	// 回答」这类多跳组合够用、成本又可控的折中（B1：由 3 放宽，因为第三跳被截断
	// 会让模型凭已有信息编）。上限 12 是运营者不能越过的进程安全边界，抬它需要
	// 单独决策记录，不是一次调参。
	DefaultToolCallsPerReply = 5
	MaxToolCallsPerReply     = 12

	// 工具结果进入提示词的体积上限：单条与单次回复累计。工具返回的是社区内容，
	// 不设上限等于让一次搜索把整站塞进上下文；但 8KB 读不完一篇长帖，上限应当
	// 与输入总预算同量级而不是比它小两个数量级。
	MaxToolResultBytes    = 64_000
	MaxToolResultRunBytes = 256_000

	// ToolTrace 是给人看的记录，不是回放载荷：参数与摘要都按这个上限截断，
	// 原始字节数另存，让「被截了」这件事本身可见。它们不是运营者可调项，
	// 因此不进 LimitsView。
	MaxToolTraceArgumentsBytes = 1_024
	MaxToolTraceDigestBytes    = 256

	// 多模态图片上限。单张 2MB 是「能看清截图/照片」与「一次请求不会把月度预算
	// 打穿」的折中；4 张是一次回复所需的合理上下文上限。两者都是硬边界，运营者
	// 不可调——可调上限意味着运营者要为「多少 token 一张图」这种细节负责。
	MaxImagesPerMessage = 4
	MaxImageBytes       = 2 * 1024 * 1024
	// MaxImageBase64Bytes 是 base64 编码后的长度上限（原始字节的 4/3，向上取整）。
	MaxImageBase64Bytes = (MaxImageBytes+2)/3*4 + 4

	// 温度范围。
	MinTemperature = 0.0
	MaxTemperature = 2.0

	// 闸门天花板。它们只限制「运营者能设多大」，默认值见 RecommendedSettings。
	MaxRateLimitPerMinute  = 60_000
	MaxExtensionDailyQuota = 100_000_000
	MaxUserDailyQuota      = 1_000_000

	// MicroPerUnit 是金额记账单位：1 个货币单位 = 1_000_000 micro。
	MicroPerUnit = int64(1_000_000)
	// MaxMonthlyBudgetMicros 是站点月度预算天花板（10,000,000 个货币单位）。
	// int64 记账在 1e13 micro 处仍有两个数量级余量（上限约 9.2e18）。
	MaxMonthlyBudgetMicros = int64(10_000_000) * MicroPerUnit
)

// LimitsView 是发给控制台的上限快照。前端据此渲染 min/max 与提示文案，
// 不再各自硬编码一份数字——两处常量漂移正是「界面允许、服务端 422」的成因。
type LimitsView struct {
	MinTimeoutMS             int   `json:"minTimeoutMs"`
	MaxTimeoutMS             int   `json:"maxTimeoutMs"`
	MaxCompletionTokens      int   `json:"maxCompletionTokens"`
	DefaultCompletionTokens  int   `json:"defaultCompletionTokens"`
	MaxToolCallsPerReply     int   `json:"maxToolCallsPerReply"`
	DefaultToolCallsPerReply int   `json:"defaultToolCallsPerReply"`
	MaxRateLimitPerMinute    int   `json:"maxRateLimitPerMinute"`
	MaxExtensionDailyQuota   int   `json:"maxExtensionDailyQuota"`
	MaxUserDailyQuota        int   `json:"maxUserDailyQuota"`
	MaxMonthlyBudgetMicros   int64 `json:"maxMonthlyBudgetMicros"`
	MaxTotalInputBytes       int   `json:"maxTotalInputBytes"`
}

// CurrentLimits 返回当前生效的上限。它是 LimitsView 的唯一构造点：以后调天花板
// 只改常量，控制台自动跟随。
func CurrentLimits() LimitsView {
	return LimitsView{
		MinTimeoutMS:             MinTimeoutMS,
		MaxTimeoutMS:             MaxTimeoutMS,
		MaxCompletionTokens:      MaxCompletionTokens,
		DefaultCompletionTokens:  DefaultMaxCompletionTokens,
		MaxToolCallsPerReply:     MaxToolCallsPerReply,
		DefaultToolCallsPerReply: DefaultToolCallsPerReply,
		MaxRateLimitPerMinute:    MaxRateLimitPerMinute,
		MaxExtensionDailyQuota:   MaxExtensionDailyQuota,
		MaxUserDailyQuota:        MaxUserDailyQuota,
		MaxMonthlyBudgetMicros:   MaxMonthlyBudgetMicros,
		MaxTotalInputBytes:       MaxTotalInputBytes,
	}
}

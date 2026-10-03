// Package ai 是 Host 的 AI 网关契约层：中立补全契约、provider profile、
// 三层配置与成本闸门。供应商调用由插件实现（ai.provider 槽位），Core 只拥有
// 契约、协议翻译、记账、限流与审计。
package ai

// Host 硬天花板。Settings 与 Profile 中的可调数值都必须落在这些边界内，
// 越界直接拒绝写入，避免运维调参意外移除进程安全边界。
// 该约定沿用 Support/ContentRegistry 的 ExecutionLimits 模式。
const (
	// 单次 provider 调用超时。
	MinTimeoutMS     = 1_000
	MaxTimeoutMS     = 120_000
	DefaultTimeoutMS = 30_000

	// 单次补全输出上限。
	MaxCompletionTokens        = 32_000
	DefaultMaxCompletionTokens = 2_048

	// 输入规模上限。长度按 UTF-8 字节计，中文按字节数计入。
	MaxMessages        = 64
	MaxMessageBytes    = 100_000
	MaxSystemBytes     = 100_000
	MaxTotalInputBytes = 400_000

	// profile 基数与标识长度。
	MaxProfiles     = 32
	MaxProfileIDLen = 64
	MaxModelLen     = 128
	MaxLabelLen     = 64

	// 工具声明上限。工具声明会随每次请求进入供应商输入，因此在契约层设死上限，
	// 避免一个用途把整站上下文成本推高到无法解释。
	MaxToolsPerRequest      = 16
	MaxToolNameLen          = 64
	MaxToolDescriptionBytes = 1_024
	MaxToolSchemaBytes      = 8_192
	MaxToolArgumentsBytes   = 16_384
	MaxToolCallIDLen        = 128

	// 单次回复允许的工具调用次数（步数预算）。默认 3 是「搜索后读帖再回答」这类
	// 组合够用、成本又可控的折中；上限 5 是硬天花板，运营者不能越过。
	DefaultToolCallsPerReply = 3
	MaxToolCallsPerReply     = 5

	// 工具结果进入提示词的体积上限：单条与单次回复累计。工具返回的是社区内容，
	// 不设上限等于让一次搜索把整站塞进上下文。
	MaxToolResultBytes    = 8_000
	MaxToolResultRunBytes = 16_000

	// 温度范围。
	MinTemperature = 0.0
	MaxTemperature = 2.0

	// 闸门天花板。
	MaxRateLimitPerMinute  = 6_000
	MaxExtensionDailyQuota = 1_000_000
	MaxUserDailyQuota      = 10_000

	// MicroPerUnit 是金额记账单位：1 个货币单位 = 1_000_000 micro。
	MicroPerUnit = int64(1_000_000)
	// MaxMonthlyBudgetMicros 是站点月度预算天花板（100,000 个货币单位）。
	MaxMonthlyBudgetMicros = int64(100_000) * MicroPerUnit
)

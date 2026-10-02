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

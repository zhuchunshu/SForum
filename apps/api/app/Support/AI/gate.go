package ai

// 闸门拒绝原因。它们进入审计与 Admin 提示，因此是稳定字符串。
const (
	GateReasonDisabled       = "gateway_disabled"
	GateReasonBudgetExceeded = "budget_exceeded"
	GateReasonRateLimited    = "rate_limited"
	GateReasonQuotaExceeded  = "quota_exceeded"
)

// 被哪个作用域拒绝。超限只影响该作用域，其他调用方不受牵连。
const (
	GateScopeSite      = "site"
	GateScopeExtension = "extension"
	GateScopeUser      = "user"
)

// UsageSnapshot 是某一作用域在一个时钟窗口内的累计用量。
type UsageSnapshot struct {
	// MinuteCalls 是最近一分钟的调用次数，用于速率限制。
	MinuteCalls int64 `json:"minuteCalls"`
	// DayCalls / DayTokens 是当日累计，用于配额。
	DayCalls  int64 `json:"dayCalls"`
	DayTokens int64 `json:"dayTokens"`
	// MonthSpendMicros 是当月累计花费，用于预算。
	MonthSpendMicros int64 `json:"monthSpendMicros"`
}

// GateInput 汇总一次调用的闸门判定输入。
type GateInput struct {
	Enabled bool
	Gates   GateSettings
	Site    UsageSnapshot
	// Extension 与 User 仅在对应 Bound 为真时参与判定。
	Extension      UsageSnapshot
	ExtensionBound bool
	User           UsageSnapshot
	UserBound      bool
}

// GateDecision 描述闸门结论。Allowed 为 false 时 Reason 与 Scope 说明原因。
type GateDecision struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
	Scope   string `json:"scope,omitempty"`
}

// EvaluateGates 按爆炸半径从大到小判定：站点预算 → 站点速率 → 扩展配额 →
// 用户配额。先判大范围可以避免在整站已降级时继续为单个插件记账。
// 纯函数，不读写外部状态，便于单测与回放。
func EvaluateGates(in GateInput) GateDecision {
	if !in.Enabled {
		return GateDecision{Reason: GateReasonDisabled, Scope: GateScopeSite}
	}
	if in.Gates.MonthlyBudgetMicros > 0 && in.Site.MonthSpendMicros >= in.Gates.MonthlyBudgetMicros {
		return GateDecision{Reason: GateReasonBudgetExceeded, Scope: GateScopeSite}
	}
	if in.Gates.RateLimitPerMinute > 0 && in.Site.MinuteCalls >= int64(in.Gates.RateLimitPerMinute) {
		return GateDecision{Reason: GateReasonRateLimited, Scope: GateScopeSite}
	}
	if in.ExtensionBound && in.Gates.ExtensionDailyQuota > 0 &&
		in.Extension.DayCalls >= int64(in.Gates.ExtensionDailyQuota) {
		return GateDecision{Reason: GateReasonQuotaExceeded, Scope: GateScopeExtension}
	}
	if in.UserBound && in.Gates.UserDailyQuota > 0 &&
		in.User.DayCalls >= int64(in.Gates.UserDailyQuota) {
		return GateDecision{Reason: GateReasonQuotaExceeded, Scope: GateScopeUser}
	}
	return GateDecision{Allowed: true}
}

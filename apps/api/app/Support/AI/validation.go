package ai

import (
	"strconv"
	"strings"
)

// 这个文件承载「配置为什么不合法」的判定：它把 Settings/Profile/GateSettings 的
// 校验拆成可定位的 Issue 列表，供 422 的字段级消息使用。放在单独文件里是为了让
// settings.go 继续只负责文档形状与读写语义。

// 配置问题码。它们是稳定字符串：进入 422 的 data.fields 后会经 Localization
// 映射成各语言文案，因此不要随手改名。
const (
	IssueCodeRequired        = "required"
	IssueCodeTooLong         = "too_long"
	IssueCodeOutOfRange      = "out_of_range"
	IssueCodeInvalidValue    = "invalid_value"
	IssueCodeInvalidURL      = "invalid_url"
	IssueCodeCredentialInURL = "credential_in_url"
	IssueCodeDuplicate       = "duplicate"
	IssueCodeTooMany         = "too_many"
)

// Issue 是一条可定位的配置问题。Field 与 PUT /admin/ai/settings 载荷的字段路径
// 保持一致（例如 gates.rateLimitPerMinute、profiles[0].defaults.maxTokens），
// 控制台据此把文案挂到对应输入框，而不是只显示一个笼统的 422。
type Issue struct {
	Field string
	Code  string
}

// toolNameListIssues 校验工具白名单的形状。语义（这个工具是否真的存在）不在这里
// 判定：登记表会随插件启停变化，配置保存时把它钉死会让「先配策略、后装插件」变成
// 不可能；运行期由登记表跳过未注册的名字。
//
// 它同时服务于旧版 reply.tools 与按用途配置（Purposes[...].tools）：两处都是同一
// 种「名称列表」形状，用同一条规则校验，避免两条路径出现不同口径。
func toolNameListIssues(field string, allowed []string) []Issue {
	if len(allowed) == 0 {
		return nil
	}
	if len(allowed) > MaxToolsPerRequest {
		return []Issue{{Field: field, Code: IssueCodeTooMany}}
	}
	seen := make(map[string]bool, len(allowed))
	for _, name := range allowed {
		name = strings.TrimSpace(name)
		if !ValidToolName(name) {
			return []Issue{{Field: field, Code: IssueCodeInvalidValue}}
		}
		if seen[name] {
			return []Issue{{Field: field, Code: IssueCodeDuplicate}}
		}
		seen[name] = true
	}
	return nil
}

func (s Settings) replyToolIssues() []Issue {
	if s.Reply.Tools == nil {
		return nil
	}
	return toolNameListIssues("reply.tools.allowed", s.Reply.Tools.Allowed)
}

// purposeIssues 校验按用途配置的形状。与工具白名单同理，这里刻意不要求用途
// 「已登记」：用途登记表会随版本与（后续的）插件用途变化，保存时把它钉死会让
// 「先写策略、后上用途」变成不可能；运行期由 resolver 跳过未登记的用途并回退到
// 自身默认值。
func (s Settings) purposeIssues() []Issue {
	if len(s.Purposes) == 0 {
		return nil
	}
	var issues []Issue
	for id, purpose := range s.Purposes {
		id = strings.TrimSpace(id)
		field := "purposes." + id
		if id == "" || len(id) > MaxPurposeIDLen || !validPurposeID(id) {
			issues = append(issues, Issue{Field: "purposes", Code: IssueCodeInvalidValue})
			continue
		}
		if purpose.Tools != nil {
			issues = append(issues, toolNameListIssues(field+".tools.allowed", purpose.Tools.Allowed)...)
		}
		if posture := strings.TrimSpace(purpose.FailurePosture); posture != "" && !ValidFailurePosture(posture) {
			issues = append(issues, Issue{Field: field + ".failurePosture", Code: IssueCodeInvalidValue})
		}
	}
	return issues
}

// Issues 覆盖策略层与受限层，并给出每一处的字段路径。越界值直接拒绝，而不是
// 截断到边界，避免静默改变运营者的意图。
//
// 它是 Validate 的唯一事实来源：Validate 只是「有任意一条即失败」的包装。
func (s Settings) Issues() []Issue {
	issues := s.replyToolIssues()
	issues = append(issues, s.purposeIssues()...)
	if len(s.Profiles) > MaxProfiles {
		issues = append(issues, Issue{Field: "profiles", Code: IssueCodeTooMany})
	}
	seen := make(map[string]bool, len(s.Profiles))
	for index, profile := range s.Profiles {
		// id 是所有引用的键：成本等级绑定与密钥引用都靠它，所以任何时候都必须合法。
		id := strings.TrimSpace(profile.ID)
		if !validProfileID(id) || len(id) > MaxProfileIDLen {
			issues = append(issues, Issue{Field: profileField(index, "id"), Code: IssueCodeInvalidValue})
			continue
		}
		if seen[id] {
			issues = append(issues, Issue{Field: profileField(index, "id"), Code: IssueCodeDuplicate})
			continue
		}
		seen[id] = true
		if !profile.Enabled {
			// 未启用的配置允许不完整：运营者常常先把自定义提供商存成草稿、
			// 补全后再启用，而草稿不参与运行，不该阻止保存。
			continue
		}
		issues = append(issues, profile.Issues(index)...)
	}
	// 同样刻意不拒绝「总开关开启但没有任何可用 profile」：缺供应商的后果由
	// Warnings 提示，运行时会走失败姿态并留下 trace，而不是把人卡在保存上。
	for class, ref := range s.CostClassProfiles {
		if !ValidCostClass(class) {
			issues = append(issues, Issue{Field: "costClassProfiles", Code: IssueCodeInvalidValue})
			continue
		}
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if !seen[ref] {
			issues = append(issues, Issue{Field: "costClassProfiles." + class, Code: IssueCodeInvalidValue})
		}
	}
	for purpose, posture := range s.PurposeFailurePosture {
		if strings.TrimSpace(purpose) == "" || !ValidFailurePosture(posture) {
			issues = append(issues, Issue{Field: "purposeFailurePosture." + purpose, Code: IssueCodeInvalidValue})
		}
	}
	if s.AutoAction.ConfidenceThreshold < 0 || s.AutoAction.ConfidenceThreshold > 1 {
		issues = append(issues, Issue{Field: "autoAction.confidenceThreshold", Code: IssueCodeOutOfRange})
	}
	for _, action := range s.AutoAction.AllowedActions {
		if !ValidAutoAction(action) {
			issues = append(issues, Issue{Field: "autoAction.allowedActions", Code: IssueCodeInvalidValue})
			break
		}
	}
	if s.AutoAction.Enabled && len(s.AutoAction.AllowedActions) == 0 {
		issues = append(issues, Issue{Field: "autoAction.allowedActions", Code: IssueCodeRequired})
	}
	issues = append(issues, s.Gates.Issues()...)
	return issues
}

func profileField(index int, name string) string {
	return "profiles[" + strconv.Itoa(index) + "]." + name
}

// Validate 覆盖策略层与受限层。越界值直接拒绝，而不是截断到边界，避免静默
// 改变运营者的意图。
func (s Settings) Validate() error {
	if len(s.Issues()) > 0 {
		return ErrSettingsInvalid
	}
	return nil
}

// Issues 逐项报告闸门取值的问题，单位与上限见 limits.go。
func (g GateSettings) Issues() []Issue {
	var issues []Issue
	if g.RateLimitPerMinute < 0 || g.RateLimitPerMinute > MaxRateLimitPerMinute {
		issues = append(issues, Issue{Field: "gates.rateLimitPerMinute", Code: IssueCodeOutOfRange})
	}
	if g.ExtensionDailyQuota < 0 || g.ExtensionDailyQuota > MaxExtensionDailyQuota {
		issues = append(issues, Issue{Field: "gates.extensionDailyQuota", Code: IssueCodeOutOfRange})
	}
	if g.UserDailyQuota < 0 || g.UserDailyQuota > MaxUserDailyQuota {
		issues = append(issues, Issue{Field: "gates.userDailyQuota", Code: IssueCodeOutOfRange})
	}
	if g.MonthlyBudgetMicros < 0 || g.MonthlyBudgetMicros > MaxMonthlyBudgetMicros {
		issues = append(issues, Issue{Field: "gates.monthlyBudgetMicros", Code: IssueCodeOutOfRange})
	}
	if g.ToolCallsPerReply != nil && (*g.ToolCallsPerReply < 0 || *g.ToolCallsPerReply > MaxToolCallsPerReply) {
		issues = append(issues, Issue{Field: "gates.toolCallsPerReply", Code: IssueCodeOutOfRange})
	}
	return issues
}

func (g GateSettings) Validate() error {
	if len(g.Issues()) > 0 {
		return ErrSettingsInvalid
	}
	return nil
}

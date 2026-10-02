package ai

import (
	"errors"
	"strings"
	"time"
)

// SettingsSchemaVersion 是站点 AI 配置的稳定身份。
const SettingsSchemaVersion = "sforum.ai.settings@1"

// 失败姿态：AI 不可用时各用途的降级方式。缺省视为 fail_open，即退回规则模式
// 并记录审计标记，而不是拦下用户请求。
const (
	FailurePostureOpen   = "fail_open"
	FailurePostureClosed = "fail_closed"
	FailurePostureSkip   = "skip_ai"
)

// 自动裁决可用的动作。删除类动作不在此列，见 ValidAutoAction。
const (
	ActionHide   = "hide"
	ActionTag    = "tag"
	ActionReject = "reject"
)

var (
	// ErrSettingsInvalid 表示站点 AI 配置不满足约束。
	ErrSettingsInvalid = errors.New("ai: settings are invalid")
)

// AutoActionSettings 控制「高置信度时自动执行动作」这一策略开关。默认关闭；
// 即使开启，动作范围也受 ValidAutoAction 限制。
type AutoActionSettings struct {
	Enabled             bool     `json:"enabled"`
	ConfidenceThreshold float64  `json:"confidenceThreshold"`
	AllowedActions      []string `json:"allowedActions"`
}

// GateSettings 是三道闸门的站点级参数。0 表示该闸门停用。
type GateSettings struct {
	RateLimitPerMinute  int   `json:"rateLimitPerMinute"`
	ExtensionDailyQuota int   `json:"extensionDailyQuota"`
	UserDailyQuota      int   `json:"userDailyQuota"`
	MonthlyBudgetMicros int64 `json:"monthlyBudgetMicros"`
}

// RedactionSettings 控制提交给供应商前的内容脱敏。
type RedactionSettings struct {
	Enabled     bool `json:"enabled"`
	RedactEmail bool `json:"redactEmail"`
	RedactPhone bool `json:"redactPhone"`
	RedactIP    bool `json:"redactIp"`
}

// Settings 是站点 AI 配置。它覆盖策略层（全部可配）与受限层（可配但有硬
// 天花板）。边界层（记账、trace、密钥处理、排序规则、契约版本）不在其中，
// 也不可由插件或后台改变。
type Settings struct {
	SchemaVersion string `json:"schemaVersion"`
	Enabled       bool   `json:"enabled"`
	// Profiles 是 provider 端点清单。密钥只以 Secret Store 引用形式出现。
	Profiles []Profile `json:"profiles"`
	// CostClassProfiles 把成本等级映射到 profile id。用途只声明成本等级，
	// 由配置决定实际使用哪个模型。
	CostClassProfiles map[string]string `json:"costClassProfiles"`
	// PurposeFailurePosture 覆盖单个用途的失败姿态；缺省为 fail_open。
	PurposeFailurePosture map[string]string  `json:"purposeFailurePosture"`
	AutoAction            AutoActionSettings `json:"autoAction"`
	Gates                 GateSettings       `json:"gates"`
	Redaction             RedactionSettings  `json:"redaction"`
	// Revision 每次保存递增，用于在执行记录中快照生效配置版本。
	Revision        int64     `json:"revision"`
	UpdatedByUserID *int64    `json:"updatedByUserId,omitempty"`
	UpdatedAt       time.Time `json:"updatedAt,omitempty"`
}

// RecommendedSettings 是开箱可用且保守的默认值：总开关关闭、不做任何自动
// 裁决、每插件配额很小、站点预算开启。运营者必须主动调高才会产生规模用量。
func RecommendedSettings() Settings {
	return Settings{
		SchemaVersion: SettingsSchemaVersion,
		Enabled:       false,
		Profiles:      BuiltinProfiles(),
		CostClassProfiles: map[string]string{
			CostClassEconomy:  "deepseek",
			CostClassStandard: "deepseek",
			CostClassPremium:  "deepseek",
		},
		PurposeFailurePosture: map[string]string{},
		AutoAction: AutoActionSettings{
			Enabled:             false,
			ConfidenceThreshold: 0.9,
			AllowedActions:      []string{ActionHide, ActionTag},
		},
		Gates: GateSettings{
			RateLimitPerMinute:  60,
			ExtensionDailyQuota: 500,
			UserDailyQuota:      20,
			MonthlyBudgetMicros: 20 * MicroPerUnit,
		},
		Redaction: RedactionSettings{
			Enabled:     true,
			RedactEmail: true,
			RedactPhone: true,
			RedactIP:    true,
		},
	}
}

func ValidFailurePosture(posture string) bool {
	switch posture {
	case FailurePostureOpen, FailurePostureClosed, FailurePostureSkip:
		return true
	default:
		return false
	}
}

// ValidAutoAction 限制自动裁决可用的动作集合。数据销毁类动作（删除）永不进入
// 白名单，无论开关如何配置；这是硬边界，不是可调项。
func ValidAutoAction(action string) bool {
	switch action {
	case ActionHide, ActionTag, ActionReject:
		return true
	default:
		return false
	}
}

// Normalized 去空白并把 nil map 归一为空 map，避免读路径反复判空。
func (s Settings) Normalized() Settings {
	s.SchemaVersion = SettingsSchemaVersion
	if s.Profiles == nil {
		s.Profiles = []Profile{}
	}
	profiles := make([]Profile, 0, len(s.Profiles))
	for _, profile := range s.Profiles {
		profiles = append(profiles, profile.Normalized())
	}
	s.Profiles = profiles
	if s.CostClassProfiles == nil {
		s.CostClassProfiles = map[string]string{}
	}
	if s.PurposeFailurePosture == nil {
		s.PurposeFailurePosture = map[string]string{}
	}
	if s.AutoAction.AllowedActions == nil {
		s.AutoAction.AllowedActions = []string{}
	}
	return s
}

// Validate 覆盖策略层与受限层。越界值直接拒绝，而不是截断到边界，避免静默
// 改变运营者的意图。
func (s Settings) Validate() error {
	if len(s.Profiles) > MaxProfiles {
		return ErrSettingsInvalid
	}
	seen := make(map[string]bool, len(s.Profiles))
	for _, profile := range s.Profiles {
		// id 是所有引用的键：成本等级绑定与密钥引用都靠它，所以任何时候都必须合法。
		id := strings.TrimSpace(profile.ID)
		if !validProfileID(id) || len(id) > MaxProfileIDLen {
			return ErrSettingsInvalid
		}
		if seen[id] {
			return ErrSettingsInvalid
		}
		seen[id] = true
		if !profile.Enabled {
			// 未启用的配置允许不完整：运营者常常先把自定义提供商存成草稿、
			// 补全后再启用，而草稿不参与运行，不该阻止保存。
			continue
		}
		if err := profile.Validate(); err != nil {
			return ErrSettingsInvalid
		}
	}
	// 同样刻意不拒绝「总开关开启但没有任何可用 profile」：缺供应商的后果由
	// Warnings 提示，运行时会走失败姿态并留下 trace，而不是把人卡在保存上。
	for class, ref := range s.CostClassProfiles {
		if !ValidCostClass(class) {
			return ErrSettingsInvalid
		}
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if !seen[ref] {
			return ErrSettingsInvalid
		}
	}
	for purpose, posture := range s.PurposeFailurePosture {
		if strings.TrimSpace(purpose) == "" || !ValidFailurePosture(posture) {
			return ErrSettingsInvalid
		}
	}
	if s.AutoAction.ConfidenceThreshold < 0 || s.AutoAction.ConfidenceThreshold > 1 {
		return ErrSettingsInvalid
	}
	for _, action := range s.AutoAction.AllowedActions {
		if !ValidAutoAction(action) {
			return ErrSettingsInvalid
		}
	}
	if s.AutoAction.Enabled && len(s.AutoAction.AllowedActions) == 0 {
		return ErrSettingsInvalid
	}
	if err := s.Gates.Validate(); err != nil {
		return err
	}
	return nil
}

func (g GateSettings) Validate() error {
	if g.RateLimitPerMinute < 0 || g.RateLimitPerMinute > MaxRateLimitPerMinute {
		return ErrSettingsInvalid
	}
	if g.ExtensionDailyQuota < 0 || g.ExtensionDailyQuota > MaxExtensionDailyQuota {
		return ErrSettingsInvalid
	}
	if g.UserDailyQuota < 0 || g.UserDailyQuota > MaxUserDailyQuota {
		return ErrSettingsInvalid
	}
	if g.MonthlyBudgetMicros < 0 || g.MonthlyBudgetMicros > MaxMonthlyBudgetMicros {
		return ErrSettingsInvalid
	}
	return nil
}

// Profile 按 id 查找。
func (s Settings) Profile(id string) (Profile, bool) {
	id = strings.TrimSpace(id)
	for _, profile := range s.Profiles {
		if strings.TrimSpace(profile.ID) == id {
			return profile, true
		}
	}
	return Profile{}, false
}

// ResolveProfile 选取本次调用使用的 profile。显式 profileRef 优先；否则按成本
// 等级取绑定；两者都不可用时返回 ErrProfileUnavailable，由网关按失败姿态处理。
func (s Settings) ResolveProfile(costClass, profileRef string) (Profile, error) {
	ref := strings.TrimSpace(profileRef)
	if ref == "" {
		ref = strings.TrimSpace(s.CostClassProfiles[strings.TrimSpace(costClass)])
	}
	if ref == "" {
		return Profile{}, ErrProfileUnavailable
	}
	profile, ok := s.Profile(ref)
	if !ok || !profile.Enabled {
		return Profile{}, ErrProfileUnavailable
	}
	return profile, nil
}

// FailurePosture 返回用途的失败姿态，缺省 fail_open。
func (s Settings) FailurePosture(purpose string) string {
	posture := strings.TrimSpace(s.PurposeFailurePosture[strings.TrimSpace(purpose)])
	if ValidFailurePosture(posture) {
		return posture
	}
	return FailurePostureOpen
}

// Warnings 返回非阻断的配置提示，供 Admin 面板显示。它们不阻止保存。
func (s Settings) Warnings() []string {
	out := []string{}
	if !s.Enabled {
		return out
	}
	enabled := 0
	priced := false
	for _, profile := range s.Profiles {
		if !profile.Enabled {
			continue
		}
		enabled++
		if profile.Price.Configured() {
			priced = true
		}
	}
	if enabled == 0 {
		out = append(out, "gateway_enabled_without_enabled_profile:no_call_will_succeed")
	}
	if !priced {
		out = append(out, "no_enabled_profile_has_price:budget_gate_inactive")
	}
	if s.Gates.MonthlyBudgetMicros <= 0 {
		out = append(out, "monthly_budget_is_zero:budget_gate_inactive")
	}
	for _, class := range []string{CostClassEconomy, CostClassStandard, CostClassPremium} {
		if strings.TrimSpace(s.CostClassProfiles[class]) == "" {
			out = append(out, "cost_class_unbound:"+class)
		}
	}
	return out
}

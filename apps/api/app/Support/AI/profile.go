package ai

import (
	"errors"
	"net/url"
	"strings"
)

// 协议标识。供应商按协议而非厂商寻址：DeepSeek、OpenAI、Moonshot、Qwen 以及
// Ollama / vLLM / 兼容网关共用 openai-chat；Claude 使用 anthropic-messages。
// 新增厂商通常只是新增一个 profile，而不是新增一个插件。
const (
	ProtocolOpenAIChat        = "openai-chat"
	ProtocolAnthropicMessages = "anthropic-messages"
)

// SecretReferencePrefix 是 Secret Store 文档引用前缀。provider 密钥只以引用形式
// 进入设置文档，密钥值本身永不写入普通设置。
const SecretReferencePrefix = "sforum.secret://"

var (
	// ErrProfileInvalid 表示 provider profile 不满足约束。
	ErrProfileInvalid = errors.New("ai: provider profile is invalid")
	// ErrProfileUnavailable 表示配置中找不到可用的 profile。
	ErrProfileUnavailable = errors.New("ai: provider profile is unavailable")
)

// Price 是每百万 token 的计价，单位 micro。全 0 表示未配置单价，此时该 profile
// 的调用不计入金额预算，但次数与 token 配额仍然生效。
type Price struct {
	InputPerMillionMicros  int64 `json:"inputPerMillionMicros"`
	OutputPerMillionMicros int64 `json:"outputPerMillionMicros"`
}

func (p Price) Configured() bool {
	return p.InputPerMillionMicros > 0 || p.OutputPerMillionMicros > 0
}

// EstimateSpendMicros 按计价估算一次调用的花费。未配置单价时返回 0。
// 计价单位是「每百万 token 多少 micro」，因此每次调用按 token 数等比折算。
func EstimateSpendMicros(price Price, usage Usage) int64 {
	var cost int64
	if price.InputPerMillionMicros > 0 {
		cost += int64(usage.InputTokens) * price.InputPerMillionMicros / 1_000_000
	}
	if price.OutputPerMillionMicros > 0 {
		cost += int64(usage.OutputTokens) * price.OutputPerMillionMicros / 1_000_000
	}
	return cost
}

type ProfileDefaults struct {
	MaxTokens   int      `json:"maxTokens"`
	Temperature *float64 `json:"temperature,omitempty"`
	TimeoutMS   int      `json:"timeoutMs"`
}

// Profile 是一个可寻址的供应商端点。baseUrl / model / apiKeyRef 均可由运营者
// 修改；修改只改变生效配置，不改变协议适配器行为，也不需要重启进程。
type Profile struct {
	ID        string          `json:"id"`
	Label     string          `json:"label"`
	Protocol  string          `json:"protocol"`
	BaseURL   string          `json:"baseUrl"`
	APIKeyRef string          `json:"apiKeyRef"`
	Model     string          `json:"model"`
	CostClass string          `json:"costClass"`
	Enabled   bool            `json:"enabled"`
	Price     Price           `json:"price"`
	Defaults  ProfileDefaults `json:"defaults"`
	// SupportsTools 为 nil 时按协议默认视为支持。两套受支持协议都定义了工具
	// 调用，但自托管模型与部分兼容网关并不支持，运营者可显式关闭它。
	SupportsTools *bool `json:"supportsTools,omitempty"`
}

// ToolsSupported 报告该 profile 能否承载工具调用。缺省为支持：若默认不支持，
// 每个既有部署都必须先改配置才能用上新能力，而「不支持」应由事实而不是
// 升级时机决定。
func (p Profile) ToolsSupported() bool {
	return p.SupportsTools == nil || *p.SupportsTools
}

func ValidProtocol(protocol string) bool {
	return protocol == ProtocolOpenAIChat || protocol == ProtocolAnthropicMessages
}

func validProfileID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return false
		}
	}
	return true
}

// Normalized 去空白并补齐结构默认值，供存储前使用。
func (p Profile) Normalized() Profile {
	p.ID = strings.TrimSpace(p.ID)
	p.Label = strings.TrimSpace(p.Label)
	p.Protocol = strings.TrimSpace(p.Protocol)
	p.BaseURL = strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	p.APIKeyRef = strings.TrimSpace(p.APIKeyRef)
	p.Model = strings.TrimSpace(p.Model)
	p.CostClass = strings.TrimSpace(p.CostClass)
	if p.Defaults.MaxTokens <= 0 {
		p.Defaults.MaxTokens = DefaultMaxCompletionTokens
	}
	if p.Defaults.TimeoutMS <= 0 {
		p.Defaults.TimeoutMS = DefaultTimeoutMS
	}
	return p
}

// Validate 只校验该 profile 自身。天花板越界一律拒绝，不给「我只想调大一点」
// 留空间；需要更大的值应先修改 Host 常量并记录决策。
func (p Profile) Validate() error {
	if !validProfileID(p.ID) || len(p.ID) > MaxProfileIDLen {
		return ErrProfileInvalid
	}
	label := strings.TrimSpace(p.Label)
	if label == "" || len(label) > MaxLabelLen {
		return ErrProfileInvalid
	}
	if !ValidProtocol(p.Protocol) {
		return ErrProfileInvalid
	}
	parsed, err := url.Parse(strings.TrimSpace(p.BaseURL))
	if err != nil || parsed.Host == "" {
		return ErrProfileInvalid
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return ErrProfileInvalid
	}
	// 凭据不得出现在 URL 中：密钥只能通过 Secret Store 引用提供，否则会随
	// 设置文档、导出与日志一起泄漏。
	if parsed.User != nil {
		return ErrProfileInvalid
	}
	ref := strings.TrimSpace(p.APIKeyRef)
	if !strings.HasPrefix(ref, SecretReferencePrefix) || len(ref) == len(SecretReferencePrefix) {
		return ErrProfileInvalid
	}
	model := strings.TrimSpace(p.Model)
	if model == "" || len(model) > MaxModelLen {
		return ErrProfileInvalid
	}
	if !ValidCostClass(p.CostClass) {
		return ErrProfileInvalid
	}
	if p.Defaults.MaxTokens <= 0 || p.Defaults.MaxTokens > MaxCompletionTokens {
		return ErrProfileInvalid
	}
	if p.Defaults.TimeoutMS < MinTimeoutMS || p.Defaults.TimeoutMS > MaxTimeoutMS {
		return ErrProfileInvalid
	}
	if p.Defaults.Temperature != nil && (*p.Defaults.Temperature < MinTemperature || *p.Defaults.Temperature > MaxTemperature) {
		return ErrProfileInvalid
	}
	if p.Price.InputPerMillionMicros < 0 || p.Price.OutputPerMillionMicros < 0 {
		return ErrProfileInvalid
	}
	return nil
}

// BuiltinProfiles 返回出厂预设。全部默认关闭：未显式启用并配置密钥前，网关不
// 会产生任何出站调用。baseUrl / model / apiKeyRef 均可由运营者覆盖。
func BuiltinProfiles() []Profile {
	return []Profile{
		{
			ID:        "deepseek",
			Label:     "DeepSeek",
			Protocol:  ProtocolOpenAIChat,
			BaseURL:   "https://api.deepseek.com",
			APIKeyRef: SecretReferencePrefix + "core/ai.deepseek.api_key",
			Model:     "deepseek-chat",
			CostClass: CostClassEconomy,
			Defaults:  ProfileDefaults{MaxTokens: DefaultMaxCompletionTokens, TimeoutMS: DefaultTimeoutMS},
		},
		{
			ID:        "openai",
			Label:     "OpenAI",
			Protocol:  ProtocolOpenAIChat,
			BaseURL:   "https://api.openai.com/v1",
			APIKeyRef: SecretReferencePrefix + "core/ai.openai.api_key",
			Model:     "gpt-4o-mini",
			CostClass: CostClassStandard,
			Defaults:  ProfileDefaults{MaxTokens: DefaultMaxCompletionTokens, TimeoutMS: DefaultTimeoutMS},
		},
		{
			ID:        "anthropic",
			Label:     "Claude",
			Protocol:  ProtocolAnthropicMessages,
			BaseURL:   "https://api.anthropic.com/v1",
			APIKeyRef: SecretReferencePrefix + "core/ai.anthropic.api_key",
			Model:     "claude-sonnet-4-20250514",
			CostClass: CostClassPremium,
			Defaults:  ProfileDefaults{MaxTokens: DefaultMaxCompletionTokens, TimeoutMS: DefaultTimeoutMS},
		},
	}
}

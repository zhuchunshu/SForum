package ai

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// CompletionContractVersion 是 Host 中立补全契约的稳定身份。provider 插件按此
// 契约实现 handler，openai-chat / anthropic-messages 的翻译由 Core 适配器负责。
const CompletionContractVersion = "sforum.ai.completion@1"

// 成本等级由用途声明，网关据此选择 provider profile。用途不得自行指定模型，
// 否则运营者无法用配置控制成本与质量。
const (
	CostClassEconomy  = "economy"
	CostClassStandard = "standard"
	CostClassPremium  = "premium"
)

// 响应格式。
const (
	ResponseFormatText = "text"
	ResponseFormatJSON = "json"
)

// 协议无关的停止原因。
const (
	StopReasonEnd      = "end"
	StopReasonLength   = "length"
	StopReasonFiltered = "filtered"
	StopReasonTool     = "tool"
	StopReasonError    = "error"
)

var (
	// ErrRequestInvalid 表示补全请求不满足契约约束。
	ErrRequestInvalid = errors.New("ai: completion request is invalid")
)

// Role 是消息角色。system 是请求级字段，不出现在 Messages 中，避免适配器
// 猜测各协议对 system 位置的约定。
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// PartType 目前只承认文本。多模态输入已由运营者决定排除。
type PartType string

const PartText PartType = "text"

type Part struct {
	Type PartType `json:"type"`
	Text string   `json:"text,omitempty"`
}

type Message struct {
	Role  Role   `json:"role"`
	Parts []Part `json:"parts"`
}

// Metadata 描述调用归属，用于记账、审计与 trace。用途不在这里：它只有
// CompletionRequest.Purpose 一个来源，避免两处不一致。
type Metadata struct {
	CallerExtensionID string `json:"callerExtensionId,omitempty"`
	ResourceType      string `json:"resourceType,omitempty"`
	ResourceID        string `json:"resourceId,omitempty"`
	Locale            string `json:"locale,omitempty"`
}

// CompletionRequest 是中立请求。ProfileRef 为空时由网关按成本等级选择。
type CompletionRequest struct {
	ProfileRef     string    `json:"profileRef,omitempty"`
	Purpose        string    `json:"purpose"`
	CostClass      string    `json:"costClass"`
	System         string    `json:"system,omitempty"`
	Messages       []Message `json:"messages"`
	MaxTokens      int       `json:"maxTokens"`
	Temperature    *float64  `json:"temperature,omitempty"`
	ResponseFormat string    `json:"responseFormat,omitempty"`
	Metadata       Metadata  `json:"metadata"`
	// PromptVersion 由调用方声明，参与缓存键与执行 trace。修改提示词必须递增它，
	// 否则历史决策无法与产出它的提示词对应。
	PromptVersion string `json:"promptVersion,omitempty"`
	// SubjectUserID 是触发本次调用的终端用户，用于用户级配额。0 表示系统发起。
	// 它不进入 Metadata，也不会出现在任何 provider 请求中。
	SubjectUserID int64 `json:"-"`
}

// ProviderArtifact 标识产出结果的确切插件产物，供审计、归因与卸载后的降级展示。
type ProviderArtifact struct {
	ExtensionID      string `json:"extensionId"`
	ExtensionVersion string `json:"extensionVersion"`
	PackageDigest    string `json:"packageDigest"`
	Protocol         string `json:"protocol"`
	Model            string `json:"model"`
}

type Usage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	CachedTokens int `json:"cachedTokens,omitempty"`
}

func (u Usage) Total() int { return u.InputTokens + u.OutputTokens }

// CompletionResult 是中立结果。PromptVersion 与 ConfigRevision 一并记录，
// 使历史决策可解释、可回放；配置后续变更不得反推历史结论。
type CompletionResult struct {
	Text           string           `json:"text"`
	StopReason     string           `json:"stopReason"`
	Usage          Usage            `json:"usage"`
	Provider       ProviderArtifact `json:"provider"`
	PromptVersion  string           `json:"promptVersion,omitempty"`
	ConfigRevision int64            `json:"configRevision,omitempty"`
	LatencyMS      int64            `json:"latencyMs"`
	CacheHit       bool             `json:"cacheHit,omitempty"`
}

func ValidRole(role Role) bool {
	return role == RoleUser || role == RoleAssistant
}

func ValidCostClass(class string) bool {
	switch class {
	case CostClassEconomy, CostClassStandard, CostClassPremium:
		return true
	default:
		return false
	}
}

func ValidResponseFormat(format string) bool {
	switch format {
	case "", ResponseFormatText, ResponseFormatJSON:
		return true
	default:
		return false
	}
}

// Normalized 去除标识类字段的空白，供存储与比较使用。
func (r CompletionRequest) Normalized() CompletionRequest {
	r.ProfileRef = strings.TrimSpace(r.ProfileRef)
	r.Purpose = strings.TrimSpace(r.Purpose)
	r.CostClass = strings.TrimSpace(r.CostClass)
	r.ResponseFormat = strings.TrimSpace(r.ResponseFormat)
	r.Metadata.CallerExtensionID = strings.TrimSpace(r.Metadata.CallerExtensionID)
	r.Metadata.ResourceType = strings.TrimSpace(r.Metadata.ResourceType)
	r.Metadata.ResourceID = strings.TrimSpace(r.Metadata.ResourceID)
	r.Metadata.Locale = strings.TrimSpace(r.Metadata.Locale)
	return r
}

// Validate 在网关入口处执行。它只校验契约自身的约束；profile 与闸门分别由
// Profile.Validate 与 EvaluateGates 负责，避免把三类失败混成一个错误。
func (r CompletionRequest) Validate() error {
	if strings.TrimSpace(r.Purpose) == "" {
		return ErrRequestInvalid
	}
	if !ValidCostClass(r.CostClass) {
		return ErrRequestInvalid
	}
	if !ValidResponseFormat(r.ResponseFormat) {
		return ErrRequestInvalid
	}
	if len(r.Messages) == 0 || len(r.Messages) > MaxMessages {
		return ErrRequestInvalid
	}
	if len(r.System) > MaxSystemBytes {
		return ErrRequestInvalid
	}
	if r.MaxTokens <= 0 || r.MaxTokens > MaxCompletionTokens {
		return ErrRequestInvalid
	}
	if r.Temperature != nil && (*r.Temperature < MinTemperature || *r.Temperature > MaxTemperature) {
		return ErrRequestInvalid
	}
	total := len(r.System)
	for _, message := range r.Messages {
		if !ValidRole(message.Role) {
			return ErrRequestInvalid
		}
		if len(message.Parts) == 0 {
			return ErrRequestInvalid
		}
		for _, part := range message.Parts {
			if part.Type != PartText {
				return ErrRequestInvalid
			}
			if strings.TrimSpace(part.Text) == "" {
				return ErrRequestInvalid
			}
			if len(part.Text) > MaxMessageBytes {
				return ErrRequestInvalid
			}
			total += len(part.Text)
		}
	}
	if total > MaxTotalInputBytes {
		return ErrRequestInvalid
	}
	return nil
}

// InputBytes 返回请求正文的 UTF-8 字节规模，供记账与 trace 使用。
func (r CompletionRequest) InputBytes() int {
	total := len(r.System)
	for _, message := range r.Messages {
		for _, part := range message.Parts {
			total += len(part.Text)
		}
	}
	return total
}

// TruncateBytes 按 UTF-8 字符边界截断，避免截出半个多字节字符。
func TruncateBytes(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	cut := value[:limit]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut
}

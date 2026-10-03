package ai

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

var (
	// ErrCredentialMissing 表示 profile 指向的密钥未被解析出来。
	ErrCredentialMissing = errors.New("ai: provider credential is missing")
	// ErrWire 表示协议层构造或解析失败。
	ErrWire = errors.New("ai: provider wire error")
)

// WireRequest 是适配器产出的协议请求。执行层负责实际发送、SSRF 校验与超时；
// 适配器只做协议翻译，不持有连接，也不决定重试。
type WireRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
}

// SafeHeaders 返回移除鉴权头后的副本，供 trace、审计与日志使用。密钥本身
// 永远不得进入任何持久化记录。
func (r WireRequest) SafeHeaders() map[string]string {
	out := make(map[string]string, len(r.Headers))
	for key, value := range r.Headers {
		switch strings.ToLower(key) {
		case "authorization", "x-api-key", "api-key", "proxy-authorization":
			continue
		default:
			out[key] = value
		}
	}
	return out
}

// WireResponse 是解析后的中立协议响应。
type WireResponse struct {
	Text       string
	StopReason string
	// ToolCalls 非空表示模型要求先执行工具；两套协议都翻译到这个字段。
	ToolCalls []ToolCall
	Usage     Usage
	// RawFinishReason 保留供应商原始值，便于排查映射偏差。
	RawFinishReason string
}

// BuildWireRequest 按 profile 协议构造 HTTP 请求。apiKey 由调用方从 Secret
// Store 解析后传入，本函数不读取任何存储。
func BuildWireRequest(profile Profile, apiKey string, request CompletionRequest) (WireRequest, error) {
	if err := request.Validate(); err != nil {
		return WireRequest{}, err
	}
	if err := profile.Validate(); err != nil {
		return WireRequest{}, ErrProfileInvalid
	}
	if strings.TrimSpace(apiKey) == "" {
		return WireRequest{}, ErrCredentialMissing
	}
	switch profile.Protocol {
	case ProtocolOpenAIChat:
		return buildOpenAIChatRequest(profile, apiKey, request)
	case ProtocolAnthropicMessages:
		return buildAnthropicMessagesRequest(profile, apiKey, request)
	default:
		return WireRequest{}, ErrProfileInvalid
	}
}

// ParseWireResponse 把协议响应解析为中立结果。非 2xx 一律视为失败，并只暴露
// 供应商给出的短错误信息，不带回可能含敏感内容的完整响应体。
func ParseWireResponse(profile Profile, statusCode int, body []byte) (WireResponse, error) {
	switch profile.Protocol {
	case ProtocolOpenAIChat:
		return parseOpenAIChatResponse(statusCode, body)
	case ProtocolAnthropicMessages:
		return parseAnthropicMessagesResponse(statusCode, body)
	default:
		return WireResponse{}, ErrProfileInvalid
	}
}

// joinAPIURL 把 profile 的 baseURL 与协议路径拼接。baseURL 是 API 根，
// 因此 https://api.deepseek.com 与 https://api.openai.com/v1 都能直接使用。
func joinAPIURL(baseURL, path string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "", ErrProfileInvalid
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" {
		return "", ErrProfileInvalid
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return "", ErrProfileInvalid
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + path
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func marshalWireBody(payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, ErrWire
	}
	return body, nil
}

func unmarshalWireBody(body []byte, target any) error {
	if len(body) == 0 {
		return ErrWire
	}
	if err := json.Unmarshal(body, target); err != nil {
		return ErrWire
	}
	return nil
}

// providerErrorSummary 从响应体的错误字段提取简短说明，用于失败原因。
func providerErrorSummary(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	return TruncateBytes(message, 200)
}

// MapOpenAIFinishReason 与 MapAnthropicStopReason 把供应商原始值映射到中立
// 停止原因。未知值归为 end，避免把可解释的结束误报成错误。
func MapOpenAIFinishReason(reason string) string {
	switch strings.TrimSpace(reason) {
	case "stop":
		return StopReasonEnd
	case "length":
		return StopReasonLength
	case "content_filter":
		return StopReasonFiltered
	case "tool_calls", "function_call":
		return StopReasonTool
	case "":
		return StopReasonEnd
	default:
		return StopReasonEnd
	}
}

func MapAnthropicStopReason(reason string) string {
	switch strings.TrimSpace(reason) {
	case "end_turn", "stop_sequence":
		return StopReasonEnd
	case "max_tokens":
		return StopReasonLength
	case "tool_use":
		return StopReasonTool
	case "":
		return StopReasonEnd
	default:
		return StopReasonEnd
	}
}

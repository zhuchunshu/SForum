package ai

import (
	"fmt"
	"net/http"
)

// anthropic-messages 适配器。与 OpenAI 的差异集中在四处：system 是顶层字段、
// 消息内容是 block 数组、鉴权用 x-api-key 加 anthropic-version 头、
// 用量字段名不同（input_tokens / output_tokens）。

// AnthropicAPIVersion 是 Messages API 的必需版本头。
const AnthropicAPIVersion = "2023-06-01"

type anthropicTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicMessage struct {
	Role    string               `json:"role"`
	Content []anthropicTextBlock `json:"content"`
}

type anthropicMessagesPayload struct {
	Model       string             `json:"model"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature *float64           `json:"temperature,omitempty"`
}

type anthropicMessagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens          int `json:"input_tokens"`
		OutputTokens         int `json:"output_tokens"`
		CacheReadInputTokens int `json:"cache_read_input_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func buildAnthropicMessagesRequest(profile Profile, apiKey string, request CompletionRequest) (WireRequest, error) {
	endpoint, err := joinAPIURL(profile.BaseURL, "/messages")
	if err != nil {
		return WireRequest{}, err
	}
	messages := make([]anthropicMessage, 0, len(request.Messages))
	for _, message := range request.Messages {
		blocks := make([]anthropicTextBlock, 0, len(message.Parts))
		for _, part := range message.Parts {
			blocks = append(blocks, anthropicTextBlock{Type: "text", Text: part.Text})
		}
		messages = append(messages, anthropicMessage{Role: string(message.Role), Content: blocks})
	}
	payload := anthropicMessagesPayload{
		Model:       profile.Model,
		System:      request.System,
		Messages:    messages,
		MaxTokens:   request.MaxTokens,
		Temperature: request.Temperature,
	}
	body, err := marshalWireBody(payload)
	if err != nil {
		return WireRequest{}, err
	}
	return WireRequest{
		Method: http.MethodPost,
		URL:    endpoint,
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"Accept":            "application/json",
			"x-api-key":         apiKey,
			"anthropic-version": AnthropicAPIVersion,
		},
		Body: body,
	}, nil
}

func parseAnthropicMessagesResponse(statusCode int, body []byte) (WireResponse, error) {
	var payload anthropicMessagesResponse
	if err := unmarshalWireBody(body, &payload); err != nil {
		return WireResponse{}, err
	}
	if statusCode < 200 || statusCode > 299 {
		summary := ""
		if payload.Error != nil {
			summary = providerErrorSummary(payload.Error.Message)
		}
		return WireResponse{}, fmt.Errorf("%w: anthropic-messages status %d %s", ErrWire, statusCode, summary)
	}
	text := ""
	for _, block := range payload.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	return WireResponse{
		Text:       text,
		StopReason: MapAnthropicStopReason(payload.StopReason),
		Usage: Usage{
			InputTokens:  payload.Usage.InputTokens,
			OutputTokens: payload.Usage.OutputTokens,
			CachedTokens: payload.Usage.CacheReadInputTokens,
		},
		RawFinishReason: payload.StopReason,
	}, nil
}

package ai

import (
	"fmt"
	"net/http"
)

// openai-chat 适配器。它覆盖 ChatGPT、DeepSeek、Moonshot、Qwen、Ollama、vLLM
// 以及任何 OpenAI 兼容网关；差异全部落在 profile 配置里。
//
// 关于 max_tokens：OpenAI 新模型推荐 max_completion_tokens，但 DeepSeek 与
// 绝大多数兼容实现只认 max_tokens。为了「一个适配器覆盖最多端点」，这里使用
// max_tokens，并在 profile 层用 MaxCompletionTokens 约束取值范围。

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequestPayload struct {
	Model          string              `json:"model"`
	Messages       []openAIChatMessage `json:"messages"`
	MaxTokens      int                 `json:"max_tokens"`
	Temperature    *float64            `json:"temperature,omitempty"`
	ResponseFormat map[string]string   `json:"response_format,omitempty"`
}

type openAIChatResponsePayload struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		PromptTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func buildOpenAIChatRequest(profile Profile, apiKey string, request CompletionRequest) (WireRequest, error) {
	endpoint, err := joinAPIURL(profile.BaseURL, "/chat/completions")
	if err != nil {
		return WireRequest{}, err
	}
	messages := make([]openAIChatMessage, 0, len(request.Messages)+1)
	// OpenAI 协议把 system 放在 messages[0]；Anthropic 是顶层字段。
	if system := request.System; system != "" {
		messages = append(messages, openAIChatMessage{Role: "system", Content: system})
	}
	for _, message := range request.Messages {
		text := ""
		for _, part := range message.Parts {
			text += part.Text
		}
		messages = append(messages, openAIChatMessage{Role: string(message.Role), Content: text})
	}
	payload := openAIChatRequestPayload{
		Model:       profile.Model,
		Messages:    messages,
		MaxTokens:   request.MaxTokens,
		Temperature: request.Temperature,
	}
	if request.ResponseFormat == ResponseFormatJSON {
		payload.ResponseFormat = map[string]string{"type": "json_object"}
	}
	body, err := marshalWireBody(payload)
	if err != nil {
		return WireRequest{}, err
	}
	return WireRequest{
		Method: http.MethodPost,
		URL:    endpoint,
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Accept":        "application/json",
			"Authorization": "Bearer " + apiKey,
		},
		Body: body,
	}, nil
}

func parseOpenAIChatResponse(statusCode int, body []byte) (WireResponse, error) {
	var payload openAIChatResponsePayload
	if err := unmarshalWireBody(body, &payload); err != nil {
		return WireResponse{}, err
	}
	if statusCode < 200 || statusCode > 299 {
		summary := ""
		if payload.Error != nil {
			summary = providerErrorSummary(payload.Error.Message)
		}
		return WireResponse{}, fmt.Errorf("%w: openai-chat status %d %s", ErrWire, statusCode, summary)
	}
	if len(payload.Choices) == 0 {
		return WireResponse{}, fmt.Errorf("%w: openai-chat returned no choices", ErrWire)
	}
	choice := payload.Choices[0]
	return WireResponse{
		Text:       choice.Message.Content,
		StopReason: MapOpenAIFinishReason(choice.FinishReason),
		Usage: Usage{
			InputTokens:  payload.Usage.PromptTokens,
			OutputTokens: payload.Usage.CompletionTokens,
			CachedTokens: payload.Usage.PromptTokensDetails.CachedTokens,
		},
		RawFinishReason: choice.FinishReason,
	}, nil
}

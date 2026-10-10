package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// 模型列表探测的边界。
const (
	// MaxModelListEntries 是一次探测回传给控制台的条目上限。某些网关会暴露上千个
	// 模型：截断并显式告知，比让一个下拉框吞掉整个响应更诚实。
	MaxModelListEntries = 500

	// anthropicModelListLimitParam 是 Claude 的 Models API 分页参数。它的默认页只有
	// 20 条，不带上这一页就只能看到最新的一小撮；500 仍在其允许的 1..1000 之内。
	anthropicModelListLimitParam = "limit"
)

var (
	// ErrModelListUnavailable 表示本部署没有装配模型列表探测能力（装配缺失，而非
	// 运营者填错了什么）。
	ErrModelListUnavailable = errors.New("ai: model listing is not wired")
)

// 探测结论的原因码见 diagnose.go 的 Reason* 常量（诊断与模型列表共用一套取值）。

// ModelListProbe 是控制台「获取模型列表」的探测参数。三个字段都可以缺省：缺省时
// 使用已保存的 profile 与 Secret Store 里的密钥。允许带上它们，是为了让运营者在
// 保存之前就能看见列表——那正是这个按钮存在的理由。
type ModelListProbe struct {
	BaseURL  string `json:"baseUrl,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	// APIKey 只在本次请求里存在：不写库、不记录、不回传。写入密钥仍然只走
	// profile credential 端点。
	APIKey string `json:"apiKey,omitempty"`
}

// ModelList 是探测回来的模型集合。顺序保持供应商给出的顺序：多数供应商把最新或
// 最常用的模型排在前面，重排成字典序会把这个信号丢掉。
type ModelList struct {
	Models []string `json:"models"`
	Count  int      `json:"count"`
	// Truncated 为 true 表示供应商还有更多条目没有展示。界面必须说明这一点，
	// 否则下拉框看起来就是该端点全部的能力。
	Truncated bool `json:"truncated,omitempty"`
}

// ModelListResult 是一次模型列表探测的完整结论。供应商层面的失败不是 HTTP 错误，
// 而是一个带原因码的结论：「地址能连上但密钥无效」正是运营者要读到的信息。
type ModelListResult struct {
	ProfileID string `json:"profileId,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	ModelList
	Reason string `json:"reason,omitempty"`
	Error  string `json:"error,omitempty"`
}

// WithProbe 把控制台的探测覆盖值合并进 profile，并只校验探测必需的字段。
//
// model 刻意不参与校验：获取列表恰恰是为了挑选模型。要求先填模型，就等于让这个
// 按钮在最需要它的场景（新建自定义供应商）里不可用。label 与 apiKeyRef 同理：
// 它们不影响一次只读探测，而「保存前先试一下」正是这个流程的价值。
func (p Profile) WithProbe(probe ModelListProbe) (Profile, error) {
	merged := p
	if protocol := strings.TrimSpace(probe.Protocol); protocol != "" {
		merged.Protocol = protocol
	}
	if baseURL := strings.TrimSpace(probe.BaseURL); baseURL != "" {
		merged.BaseURL = strings.TrimRight(baseURL, "/")
	}
	if !ValidProtocol(merged.Protocol) {
		return Profile{}, ErrProfileInvalid
	}
	if err := ValidateEndpointURL(merged.BaseURL); err != nil {
		return Profile{}, ErrProfileInvalid
	}
	// 超时越界或未设置时回落到默认值：探测必须有一个上限，否则一个挂住的端点会
	// 占住管理请求直到客户端放弃。
	if merged.Defaults.TimeoutMS < MinTimeoutMS || merged.Defaults.TimeoutMS > MaxTimeoutMS {
		merged.Defaults.TimeoutMS = DefaultTimeoutMS
	}
	return merged, nil
}

// BuildModelListRequest 按 profile 协议构造列出模型的只读请求。
//
// 路径与补全路径同源：openai-chat 的 /models 与 anthropic-messages 的 /models 都
// 挂在同一个 API 根之下，因此 baseUrl 的语义在两类调用里完全一致。
func BuildModelListRequest(profile Profile, apiKey string) (WireRequest, error) {
	if !ValidProtocol(profile.Protocol) {
		return WireRequest{}, ErrProfileInvalid
	}
	if err := ValidateEndpointURL(profile.BaseURL); err != nil {
		return WireRequest{}, ErrProfileInvalid
	}
	if strings.TrimSpace(apiKey) == "" {
		return WireRequest{}, ErrCredentialMissing
	}
	endpoint, err := joinAPIURL(profile.BaseURL, "/models")
	if err != nil {
		return WireRequest{}, err
	}
	switch profile.Protocol {
	case ProtocolOpenAIChat:
		return WireRequest{
			Method: http.MethodGet,
			URL:    endpoint,
			Headers: map[string]string{
				"Accept":        "application/json",
				"Authorization": "Bearer " + apiKey,
			},
		}, nil
	case ProtocolAnthropicMessages:
		endpoint, err = withQueryValue(endpoint, anthropicModelListLimitParam, strconv.Itoa(MaxModelListEntries))
		if err != nil {
			return WireRequest{}, err
		}
		return WireRequest{
			Method: http.MethodGet,
			URL:    endpoint,
			Headers: map[string]string{
				"Accept":            "application/json",
				"x-api-key":         apiKey,
				"anthropic-version": AnthropicAPIVersion,
			},
		}, nil
	default:
		return WireRequest{}, ErrProfileInvalid
	}
}

// ParseModelListResponse 把模型列表响应解析为模型标识集合。
//
// 两套协议的列表响应共享同一形状（data[].id），因此共用一个解析器；Claude 额外
// 返回的 display_name 不参与——profile 里存的是模型标识，不是展示名。非 2xx 一律
// 视为失败，并只暴露供应商给出的短错误信息。
func ParseModelListResponse(profile Profile, statusCode int, body []byte) (ModelList, error) {
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		HasMore bool `json:"has_more"`
		Error   *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := unmarshalWireBody(body, &payload); err != nil {
		return ModelList{}, err
	}
	if statusCode < 200 || statusCode > 299 {
		summary := ""
		if payload.Error != nil {
			summary = providerErrorSummary(payload.Error.Message)
		}
		return ModelList{}, fmt.Errorf("%w: %s model list status %d %s", ErrWire, profile.Protocol, statusCode, summary)
	}
	ids := make([]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		ids = append(ids, item.ID)
	}
	return normalizeModelList(ids, payload.HasMore), nil
}

// normalizeModelList 去重、去空白并按上限截断，同时报告是否被截断。去重保持首次
// 出现的顺序：重复条目只会让下拉框变长，不会带来新信息。
func normalizeModelList(ids []string, providerHasMore bool) ModelList {
	seen := make(map[string]bool, len(ids))
	models := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		models = append(models, trimmed)
	}
	truncated := providerHasMore
	if len(models) > MaxModelListEntries {
		models = models[:MaxModelListEntries]
		truncated = true
	}
	return ModelList{Models: models, Count: len(models), Truncated: truncated}
}

// HTTPModelLister 用 Host 受控出站路径执行一次模型列表探测。它与补全调用共享同一
// 个 invoker：SSRF 校验、超时与响应体上限只有一处实现，探测不会成为绕过它们的
// 旁路。
type HTTPModelLister struct {
	invoker *HTTPInvoker
}

func NewHTTPModelLister(invoker *HTTPInvoker) *HTTPModelLister {
	return &HTTPModelLister{invoker: invoker}
}

func (l *HTTPModelLister) ListModels(ctx context.Context, profile Profile, apiKey string) (ModelList, error) {
	if l == nil || l.invoker == nil {
		return ModelList{}, ErrModelListUnavailable
	}
	request, err := BuildModelListRequest(profile, apiKey)
	if err != nil {
		return ModelList{}, err
	}
	status, body, err := l.invoker.Execute(ctx, profile, request)
	if err != nil {
		return ModelList{}, err
	}
	return ParseModelListResponse(profile, status, body)
}

// withQueryValue 在已有 URL 上追加一个查询参数。joinAPIURL 会清掉原有查询串，
// 分页参数必须在拼接之后再补上。
func withQueryValue(endpoint, key, value string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", ErrProfileInvalid
	}
	query := parsed.Query()
	query.Set(key, value)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

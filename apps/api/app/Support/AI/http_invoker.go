package ai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	outboundhttp "github.com/zhuchunshu/sforum/apps/api/app/Support/OutboundHTTP"
)

// MaxProviderResponseBytes 限制读取的供应商响应体大小。异常或恶意的超大响应
// 不应把进程内存吃光。
const MaxProviderResponseBytes = 1 << 20

// HTTPInvoker 用 Host 受控出站路径执行协议请求。
//
// 为什么由 Core 而不是插件出站：运营者要求「内置 DeepSeek / OpenAI 并可修改
// baseUrl 与 apiKey」。若供应商行为在插件代码里，改 baseUrl 就等于改插件行为，
// 该要求无法成立。因此 M0 的默认执行路径是 Core 翻译协议 + Host 受控出站；
// ai.provider 槽位留给非标准协议（自研推理服务、Bedrock、gRPC 等）的后续接入。
type HTTPInvoker struct {
	client            *http.Client
	allowHTTP         bool
	skipURLValidation bool
}

type HTTPInvokerOptions struct {
	// AllowHTTP 允许 http:// 目标（仅开发环境；生产应由出站守卫拦截）。
	AllowHTTP bool
	// Timeout 是客户端级超时上限；单次调用仍受 profile.Defaults.TimeoutMS 约束。
	Timeout time.Duration
	// Client 覆盖出站客户端，仅供测试注入。
	Client *http.Client
	// SkipURLValidation 跳过发送前的公网可达性预检，仅供测试注入。
	SkipURLValidation bool
}

func NewHTTPInvoker(opts HTTPInvokerOptions) *HTTPInvoker {
	client := opts.Client
	if client == nil {
		timeout := opts.Timeout
		if timeout <= 0 {
			timeout = DefaultTimeoutMS * time.Millisecond
		}
		client = outboundhttp.NewSafeClient(outboundhttp.Options{
			AllowHTTP: opts.AllowHTTP,
			Timeout:   timeout,
		})
	}
	return &HTTPInvoker{client: client, allowHTTP: opts.AllowHTTP, skipURLValidation: opts.SkipURLValidation}
}

// Invoke 执行一次协议请求。它不做重试：重试属于上层按失败姿态的决策，在供应商
// 已经限流时重试只会放大流量。
func (i *HTTPInvoker) Invoke(ctx context.Context, profile Profile, request WireRequest) (WireResponse, error) {
	if i == nil || i.client == nil {
		return WireResponse{}, fmt.Errorf("%w: outbound client is not wired", ErrWire)
	}
	if !i.skipURLValidation {
		// 配置时校验过一次，发送前再校验：DNS 可能在保存之后变化。
		if err := outboundhttp.ValidatePublicURL(request.URL, outboundhttp.Options{AllowHTTP: i.allowHTTP}); err != nil {
			return WireResponse{}, fmt.Errorf("%w: %v", outboundhttp.ErrUnsafeURL, err)
		}
	}
	timeout := time.Duration(profile.Defaults.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = DefaultTimeoutMS * time.Millisecond
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpRequest, err := http.NewRequestWithContext(callCtx, request.Method, request.URL, bytes.NewReader(request.Body))
	if err != nil {
		return WireResponse{}, fmt.Errorf("%w: %v", ErrWire, err)
	}
	for key, value := range request.Headers {
		httpRequest.Header.Set(key, value)
	}
	httpRequest.Header.Set("User-Agent", "SForum-AI-Gateway/1")
	response, err := i.client.Do(httpRequest)
	if err != nil {
		if callCtx.Err() != nil {
			return WireResponse{}, fmt.Errorf("%w: provider call timed out", ErrWire)
		}
		return WireResponse{}, fmt.Errorf("%w: %s", ErrWire, TruncateBytes(err.Error(), 200))
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, MaxProviderResponseBytes))
	if err != nil {
		return WireResponse{}, fmt.Errorf("%w: read provider response failed", ErrWire)
	}
	return ParseWireResponse(profile, response.StatusCode, body)
}

package pages

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// 页面 data loader 安全边界：仅经 RouteGateway/loopback，不接受任意 URL。
const (
	MaxLoaderResponseBytes = 256 * 1024
	DefaultLoaderTimeout   = 3 * time.Second
	MaxLoaderTimeout       = 10 * time.Second
)

// LoaderRequest 宿主向插件 loopback 发起的页面数据请求。
type LoaderRequest struct {
	ExtensionID string
	// Route 是插件 manifest 内相对路径，如 /docs/data（禁止绝对 URL / 外网）。
	Route   string
	ActorID int64
	Locale  string
	// TargetBase 由 runtime.RouteTarget 提供的 loopback base。
	TargetBase string
	Timeout    time.Duration
}

// LoaderResult 校验后的 JSON 载荷。
type LoaderResult struct {
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
	Status  int             `json:"status,omitempty"`
	Fallback bool           `json:"fallback,omitempty"`
}

// LoaderTransport 抽象 HTTP 调用（测试可注入）。
type LoaderTransport interface {
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}

// PageDataLoader 通过 loopback 拉取插件页面数据。
type PageDataLoader struct {
	Transport LoaderTransport
}

func NewPageDataLoader(transport LoaderTransport) *PageDataLoader {
	if transport == nil {
		transport = httpTransport{}
	}
	return &PageDataLoader{Transport: transport}
}

type httpTransport struct{}

func (httpTransport) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req.WithContext(ctx))
}

// ValidateDataRoute 拒绝任意 URL；仅允许插件内相对路径。
func ValidateDataRoute(route string) error {
	route = strings.TrimSpace(route)
	if route == "" {
		return fmt.Errorf("pages: empty data route")
	}
	if strings.Contains(route, "://") || strings.HasPrefix(route, "//") {
		return fmt.Errorf("pages: data route must not be absolute URL")
	}
	if strings.Contains(route, "..") {
		return fmt.Errorf("pages: data route path traversal")
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	// 禁止指向宿主敏感前缀（即便经插件代理）
	lower := strings.ToLower(route)
	for _, bad := range []string{"/admin", "/api/", "/control-panel"} {
		if lower == bad || strings.HasPrefix(lower, bad+"/") {
			return fmt.Errorf("pages: data route reserved")
		}
	}
	return nil
}

// Fetch 执行 loader：timeout、大小、content-type、JSON 校验。
// 不转发原始 session cookie；仅传最小 actor id。
func (l *PageDataLoader) Fetch(ctx context.Context, in LoaderRequest) LoaderResult {
	if err := ValidateDataRoute(in.Route); err != nil {
		return LoaderResult{Error: err.Error(), Fallback: true, Status: 422}
	}
	base := strings.TrimSpace(in.TargetBase)
	if base == "" || !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		return LoaderResult{Error: "pages: loader target unavailable", Fallback: true, Status: 503}
	}
	// 仅 loopback 目标（纵深；runtime 也应保证）
	if !isLoopbackBase(base) {
		return LoaderResult{Error: "pages: loader target not loopback", Fallback: true, Status: 503}
	}
	route := strings.TrimSpace(in.Route)
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	timeout := in.Timeout
	if timeout <= 0 {
		timeout = DefaultLoaderTimeout
	}
	if timeout > MaxLoaderTimeout {
		timeout = MaxLoaderTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := strings.TrimRight(base, "/") + route
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return LoaderResult{Error: err.Error(), Fallback: true, Status: 500}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-SForum-Extension-ID", in.ExtensionID)
	if in.ActorID > 0 {
		req.Header.Set("X-SForum-Actor-ID", fmt.Sprintf("%d", in.ActorID))
	}
	if in.Locale != "" {
		req.Header.Set("X-SForum-Locale", in.Locale)
	}
	// 明确不设置 Cookie / Authorization

	resp, err := l.Transport.Do(ctx, req)
	if err != nil {
		return LoaderResult{Error: "pages: loader request failed", Fallback: true, Status: 504}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return LoaderResult{Error: "auth.required", Status: 401, Fallback: true}
	}
	if resp.StatusCode == http.StatusForbidden {
		return LoaderResult{Error: "permission.denied", Status: 403, Fallback: true}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return LoaderResult{Error: fmt.Sprintf("pages: loader status %d", resp.StatusCode), Status: resp.StatusCode, Fallback: true}
	}

	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if ct != "" && !strings.Contains(ct, "application/json") && !strings.Contains(ct, "+json") {
		return LoaderResult{Error: "pages: loader content-type must be json", Status: 502, Fallback: true}
	}

	limited := io.LimitReader(resp.Body, MaxLoaderResponseBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return LoaderResult{Error: "pages: loader read failed", Fallback: true, Status: 502}
	}
	if len(body) > MaxLoaderResponseBytes {
		return LoaderResult{Error: "pages: loader response too large", Fallback: true, Status: 502}
	}
	// 必须是 JSON object 或 array
	trim := bytes.TrimSpace(body)
	if len(trim) == 0 || (trim[0] != '{' && trim[0] != '[') {
		return LoaderResult{Error: "pages: loader body not json", Fallback: true, Status: 502}
	}
	if !json.Valid(trim) {
		return LoaderResult{Error: "pages: loader invalid json", Fallback: true, Status: 502}
	}
	// 粗检敏感键（扩展不得通过 loader 回传密钥类字段）
	lowerBody := strings.ToLower(string(trim))
	for _, sens := range []string{`"password"`, `"secret"`, `"private_key"`, `"session_token"`, `"csrf"`} {
		if strings.Contains(lowerBody, sens) {
			return LoaderResult{Error: "pages: loader response contains sensitive keys", Fallback: true, Status: 502}
		}
	}
	return LoaderResult{Data: json.RawMessage(trim), Status: resp.StatusCode}
}

func isLoopbackBase(base string) bool {
	lower := strings.ToLower(base)
	return strings.Contains(lower, "127.0.0.1") ||
		strings.Contains(lower, "localhost") ||
		strings.Contains(lower, "[::1]") ||
		strings.Contains(lower, "0.0.0.0")
}

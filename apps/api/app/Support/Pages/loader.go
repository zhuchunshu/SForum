package pages

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
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
// 不转发 Cookie / Authorization / CSRF / Session。
// ActorID 仅作可选上下文提示；插件不得将其作为授权依据——权限由 Host API 检查。
type LoaderRequest struct {
	ExtensionID string
	// Route 是插件 manifest 内相对路径，如 /docs/data（禁止绝对 URL / 外网）。
	Route string
	// Params 路由参数（来自 Page Registry 匹配）。
	Params map[string]string
	// Locale 请求语言。
	Locale string
	// ActorID 已认证用户 id（0=匿名）；非授权凭证。
	ActorID int64
	// TargetBase 由 runtime.RouteTarget 提供的 loopback base（须经严格校验）。
	TargetBase string
	Timeout    time.Duration
	// SchemaJSON 可选：manifest 声明的 JSON Schema 原文（空则跳过 schema 校验）。
	SchemaJSON string
	// AllowRedirect 默认 false：完全禁用 loader redirect。
	AllowRedirect bool
}

// LoaderResult 校验后的 JSON 载荷。
type LoaderResult struct {
	Data     json.RawMessage `json:"data,omitempty"`
	Error    string          `json:"error,omitempty"`
	Status   int             `json:"status,omitempty"`
	Fallback bool            `json:"fallback,omitempty"`
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
	// 禁用默认 redirect：CheckRedirect 返回错误
	client := &http.Client{
		Timeout: DefaultLoaderTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return fmt.Errorf("pages: loader redirects are disabled")
		},
	}
	return client.Do(req.WithContext(ctx))
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

// ValidateLoaderTargetBase 严格校验 loopback BaseURL（复用 SSRF 规则，不用 contains）。
func ValidateLoaderTargetBase(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("pages: empty loader target")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("pages: invalid loader target: %w", err)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("pages: loader target scheme %q not allowed", parsed.Scheme)
	}
	if parsed.User != nil {
		return fmt.Errorf("pages: loader target userinfo not allowed")
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("pages: loader target empty host")
	}
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return nil
	}
	// 拒绝 0.0.0.0 等未指定地址
	if host == "0.0.0.0" || host == "::" {
		return fmt.Errorf("pages: loader target unspecified address not allowed")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("pages: loader target resolve %q: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("pages: loader target no addresses for %q", host)
	}
	for _, ip := range ips {
		if !ip.IsLoopback() {
			return fmt.Errorf("pages: loader target %q is not loopback (%s)", host, ip)
		}
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("pages: loader target disallowed address %s", ip)
		}
	}
	return nil
}

// Fetch 执行 loader：timeout、大小、content-type、JSON 校验、禁止 redirect。
// 不转发 Cookie / Authorization / CSRF / Session。
func (l *PageDataLoader) Fetch(ctx context.Context, in LoaderRequest) LoaderResult {
	if err := ValidateDataRoute(in.Route); err != nil {
		return LoaderResult{Error: err.Error(), Fallback: true, Status: 422}
	}
	if err := ValidateLoaderTargetBase(in.TargetBase); err != nil {
		return LoaderResult{Error: err.Error(), Fallback: true, Status: 503}
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

	// 附加 route params 与 locale 为 query（最小上下文，非授权）
	u, err := url.Parse(strings.TrimRight(in.TargetBase, "/") + route)
	if err != nil {
		return LoaderResult{Error: err.Error(), Fallback: true, Status: 500}
	}
	q := u.Query()
	for k, v := range in.Params {
		if k != "" {
			q.Set("param."+k, v)
		}
	}
	if in.Locale != "" {
		q.Set("locale", in.Locale)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return LoaderResult{Error: err.Error(), Fallback: true, Status: 500}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-SForum-Extension-ID", in.ExtensionID)
	// 最小 actor 上下文：仅提示 id，插件不得作为授权依据
	if in.ActorID > 0 {
		req.Header.Set("X-SForum-Actor-ID-Hint", fmt.Sprintf("%d", in.ActorID))
	}
	if in.Locale != "" {
		req.Header.Set("X-SForum-Locale", in.Locale)
	}
	// 明确不设置 Cookie / Authorization / X-Csrf-Token

	resp, err := l.Transport.Do(ctx, req)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "redirect") {
			return LoaderResult{Error: "pages: loader redirect forbidden", Fallback: true, Status: 502}
		}
		return LoaderResult{Error: "pages: loader request failed", Fallback: true, Status: 504}
	}
	defer resp.Body.Close()

	// 3xx 一律拒绝（纵深：即便 transport 跟随了 redirect）
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return LoaderResult{Error: "pages: loader redirect forbidden", Status: resp.StatusCode, Fallback: true}
	}
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
	trim := bytes.TrimSpace(body)
	if len(trim) == 0 || (trim[0] != '{' && trim[0] != '[') {
		return LoaderResult{Error: "pages: loader body not json", Fallback: true, Status: 502}
	}
	if !json.Valid(trim) {
		return LoaderResult{Error: "pages: loader invalid json", Fallback: true, Status: 502}
	}
	// 粗检敏感键
	lowerBody := strings.ToLower(string(trim))
	for _, sens := range []string{`"password"`, `"secret"`, `"private_key"`, `"session_token"`, `"csrf"`, `"authorization"`} {
		if strings.Contains(lowerBody, sens) {
			return LoaderResult{Error: "pages: loader response contains sensitive keys", Fallback: true, Status: 502}
		}
	}
	// 可选 JSON Schema（轻量：若声明了 schema 且响应不是 object/array 结构则已在上面拒绝；
	// 完整 schema 校验在 gateway 层用声明文件做 required 字段检查时可扩展）
	if schema := strings.TrimSpace(in.SchemaJSON); schema != "" {
		if err := validateAgainstSimpleSchema(trim, schema); err != nil {
			return LoaderResult{Error: "pages: loader schema validation failed: " + err.Error(), Fallback: true, Status: 502}
		}
	}
	return LoaderResult{Data: json.RawMessage(trim), Status: resp.StatusCode}
}

// validateAgainstSimpleSchema 轻量 schema 校验：支持 type + required 字段列表的 JSON Schema 子集。
// 避免引入重型依赖；复杂 schema 可后续换成熟库。
func validateAgainstSimpleSchema(data []byte, schemaJSON string) error {
	var schema map[string]any
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return fmt.Errorf("invalid schema json")
	}
	typ, _ := schema["type"].(string)
	if typ == "object" {
		var obj map[string]any
		if err := json.Unmarshal(data, &obj); err != nil {
			return fmt.Errorf("expected object")
		}
		if req, ok := schema["required"].([]any); ok {
			for _, r := range req {
				key, _ := r.(string)
				if key == "" {
					continue
				}
				if _, has := obj[key]; !has {
					return fmt.Errorf("missing required field %q", key)
				}
			}
		}
	}
	if typ == "array" {
		var arr []any
		if err := json.Unmarshal(data, &arr); err != nil {
			return fmt.Errorf("expected array")
		}
	}
	return nil
}

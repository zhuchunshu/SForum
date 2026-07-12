// Package outboundhttp 提供出站 HTTP 的 SSRF 防护（webhook 等）。
// 在配置时与连接时双重校验解析后的 IP，并限制重定向。
package outboundhttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultMaxRedirects 出站请求最多跟随的重定向次数。
	DefaultMaxRedirects = 3
	// DefaultTimeout 默认整请求超时。
	DefaultTimeout = 15 * time.Second
)

var (
	// ErrUnsafeURL 目标 URL 不允许（scheme/userinfo/私网等）。对客户端应映射为通用校验错误。
	ErrUnsafeURL = errors.New("outboundhttp: unsafe url")
)

// Resolver 可替换 DNS 解析，便于 DNS rebinding 测试。
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

type defaultResolver struct{}

func (defaultResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	return net.DefaultResolver.LookupIPAddr(ctx, host)
}

// Options 控制允许的 scheme 与解析行为。
type Options struct {
	// AllowHTTP 为 false 时仅允许 https（生产默认）。开发可显式打开。
	AllowHTTP bool
	// Resolver 默认使用系统解析器。
	Resolver Resolver
	// MaxRedirects 默认 DefaultMaxRedirects。
	MaxRedirects int
	// Timeout 默认 DefaultTimeout。
	Timeout time.Duration
}

// ValidatePublicURL 校验配置时的目标 URL：scheme、禁止 userinfo、解析全部 IP 且均为公网。
func ValidatePublicURL(raw string, opts Options) error {
	parsed, err := parseOutboundURL(raw, opts)
	if err != nil {
		return err
	}
	return validateHostPublic(context.Background(), opts.resolver(), parsed.Hostname())
}

// NewSafeClient 返回带 DialContext IP 校验与重定向校验的 http.Client。
// 连接时重新解析主机名，防止保存后 DNS rebinding。
func NewSafeClient(opts Options) *http.Client {
	resolver := opts.resolver()
	maxRedirects := opts.MaxRedirects
	if maxRedirects <= 0 {
		maxRedirects = DefaultMaxRedirects
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:                 nil, // 出站 webhook 不走环境代理，避免绕过 IP 校验
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          32,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrUnsafeURL, err)
			}
			ips, err := resolvePublicIPs(ctx, resolver, host)
			if err != nil {
				return nil, err
			}
			var lastErr error
			for _, ip := range ips {
				var addr string
				if ip.To4() != nil {
					addr = net.JoinHostPort(ip.String(), port)
				} else {
					addr = net.JoinHostPort(ip.String(), port)
				}
				conn, dialErr := dialer.DialContext(ctx, network, addr)
				if dialErr == nil {
					return conn, nil
				}
				lastErr = dialErr
			}
			if lastErr == nil {
				lastErr = fmt.Errorf("%w: no dialable address", ErrUnsafeURL)
			}
			return nil, lastErr
		},
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("%w: too many redirects", ErrUnsafeURL)
			}
			if err := validateRequestURL(req.URL, opts); err != nil {
				return err
			}
			// 跨 origin 不得携带签名与其它 SForum 身份头。
			if len(via) > 0 && !sameOrigin(via[0].URL, req.URL) {
				stripWebhookAuthHeaders(req)
			}
			// 连接时 DialContext 会再校验 IP；此处再解析一次以尽早失败。
			if err := validateHostPublic(req.Context(), resolver, req.URL.Hostname()); err != nil {
				return err
			}
			return nil
		},
	}
}

func (o Options) resolver() Resolver {
	if o.Resolver != nil {
		return o.Resolver
	}
	return defaultResolver{}
}

func parseOutboundURL(raw string, opts Options) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("%w: empty", ErrUnsafeURL)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsafeURL, err)
	}
	if err := validateRequestURL(parsed, opts); err != nil {
		return nil, err
	}
	return parsed, nil
}

func validateRequestURL(parsed *url.URL, opts Options) error {
	if parsed == nil {
		return fmt.Errorf("%w: nil url", ErrUnsafeURL)
	}
	scheme := strings.ToLower(parsed.Scheme)
	switch scheme {
	case "https":
		// ok
	case "http":
		if !opts.AllowHTTP {
			return fmt.Errorf("%w: http not allowed", ErrUnsafeURL)
		}
	default:
		return fmt.Errorf("%w: scheme %q", ErrUnsafeURL, parsed.Scheme)
	}
	if parsed.User != nil {
		return fmt.Errorf("%w: userinfo not allowed", ErrUnsafeURL)
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("%w: empty host", ErrUnsafeURL)
	}
	// 拒绝显式危险端口以外的非常规端口不是目标；默认允许 80/443 及其它，依赖 IP 边界。
	return nil
}

func validateHostPublic(ctx context.Context, resolver Resolver, host string) error {
	_, err := resolvePublicIPs(ctx, resolver, host)
	return err
}

func resolvePublicIPs(ctx context.Context, resolver Resolver, host string) ([]net.IP, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, fmt.Errorf("%w: empty host", ErrUnsafeURL)
	}
	// 字面量 IP：直接判断，不走 DNS。
	if ip := net.ParseIP(host); ip != nil {
		if IsForbiddenIP(ip) {
			return nil, fmt.Errorf("%w: address %s not allowed", ErrUnsafeURL, ip)
		}
		return []net.IP{ip}, nil
	}
	addrs, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve %q: %v", ErrUnsafeURL, host, err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("%w: no addresses for %q", ErrUnsafeURL, host)
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		ip := addr.IP
		if ip == nil {
			continue
		}
		if IsForbiddenIP(ip) {
			return nil, fmt.Errorf("%w: host %q resolves to disallowed address %s", ErrUnsafeURL, host, ip)
		}
		ips = append(ips, ip)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("%w: no usable addresses for %q", ErrUnsafeURL, host)
	}
	return ips, nil
}

// IsForbiddenIP 拒绝 loopback / 私网 / link-local / 组播 / 未指定 / 文档网段 / CGNAT 等。
func IsForbiddenIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsInterfaceLocalMulticast() {
		return true
	}
	// IPv4 特殊用途
	if v4 := ip.To4(); v4 != nil {
		// 0.0.0.0/8 已由 IsUnspecified 部分覆盖；再拦 0.0.0.0/8 其余
		if v4[0] == 0 {
			return true
		}
		// CGNAT 100.64.0.0/10
		if v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
			return true
		}
		// 192.0.0.0/24 IETF Protocol Assignments（含部分特殊）
		if v4[0] == 192 && v4[1] == 0 && v4[2] == 0 {
			return true
		}
		// TEST-NET / documentation
		if v4[0] == 192 && v4[1] == 0 && v4[2] == 2 {
			return true
		}
		if v4[0] == 198 && v4[1] == 51 && v4[2] == 100 {
			return true
		}
		// 203.0.113.0/24 documentation
		if v4[0] == 203 && v4[1] == 0 && v4[2] == 113 {
			return true
		}
		// 基准测试 198.18.0.0/15
		if v4[0] == 198 && (v4[1] == 18 || v4[1] == 19) {
			return true
		}
		// 广播
		if v4[0] == 255 && v4[1] == 255 && v4[2] == 255 && v4[3] == 255 {
			return true
		}
		return false
	}
	// IPv6 documentation 2001:db8::/32
	if len(ip) == net.IPv6len && ip[0] == 0x20 && ip[1] == 0x01 && ip[2] == 0x0d && ip[3] == 0xb8 {
		return true
	}
	return false
}

func sameOrigin(a, b *url.URL) bool {
	if a == nil || b == nil {
		return false
	}
	return strings.EqualFold(a.Scheme, b.Scheme) &&
		strings.EqualFold(a.Hostname(), b.Hostname()) &&
		portOrDefault(a) == portOrDefault(b)
}

func portOrDefault(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return ""
	}
}

// stripWebhookAuthHeaders 跨 origin 重定向时去掉签名与事件身份头。
func stripWebhookAuthHeaders(req *http.Request) {
	if req == nil {
		return
	}
	for _, key := range []string{
		"X-SForum-Signature",
		"X-SForum-Timestamp",
		"X-SForum-Event",
		"X-SForum-Event-Id",
		"X-SForum-Delivery",
		"X-SForum-Correlation-Id",
		"Authorization",
		"Cookie",
	} {
		req.Header.Del(key)
	}
}

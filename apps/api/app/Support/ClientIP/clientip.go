// Package clientip 从 HTTP 请求解析真实客户端 IP。
//
// 设计目标：在 CDN + 反向代理（Caddy/Nginx）+ Docker + Nuxt 反代多层链路下
// 仍能拿到真实 IP，同时防止客户端伪造 X-Forwarded-For。
//
// 规则：
//  1. 仅当 TCP RemoteIP 落在信任代理集合时，才读取转发头；
//  2. 优先单值 CDN 头（CF-Connecting-IP / True-Client-IP / X-Real-IP）；
//  3. 再处理 X-Forwarded-For：从右往左剥掉信任代理，取第一个非信任 IP；
//  4. 全部失败则回退 RemoteIP。
//
// 业务代码应统一走 FromCtx，不要散落 c.IP()。
package clientip

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"
)

// 常见 CDN / 代理写入的客户端 IP 头（单值，优先于 XFF 链）。
var preferredClientHeaders = []string{
	"CF-Connecting-IP", // Cloudflare
	"True-Client-IP",   // Akamai / 部分企业 CDN
	"X-Real-IP",        // Nginx / Caddy 常见
}

const headerXForwardedFor = "X-Forwarded-For"

// Config 控制哪些来源的转发头可信。
type Config struct {
	// Proxies 是显式信任的代理 IP 或 CIDR（如 "10.0.0.1", "172.18.0.0/16"）。
	Proxies []string
	// TrustPrivate 信任 RFC1918 / ULA 私网（Docker 网桥常用）。
	TrustPrivate bool
	// TrustLoopback 信任 127.0.0.0/8 与 ::1。
	TrustLoopback bool
	// TrustLinkLocal 信任链路本地地址（一般关闭）。
	TrustLinkLocal bool
}

// Resolver 根据配置解析客户端 IP。
type Resolver struct {
	exact     map[netip.Addr]struct{}
	nets      []netip.Prefix
	private   bool
	loopback  bool
	linkLocal bool
}

var (
	mu       sync.RWMutex
	defaultR = NewResolver(Config{TrustPrivate: true, TrustLoopback: true})
)

// Configure 设置进程级默认解析器（bootstrap / NewApp 时调用一次即可）。
func Configure(cfg Config) {
	mu.Lock()
	defaultR = NewResolver(cfg)
	mu.Unlock()
}

// Default 返回当前默认解析器。
func Default() *Resolver {
	mu.RLock()
	defer mu.RUnlock()
	return defaultR
}

// NewResolver 编译信任列表；非法 CIDR/IP 会被静默忽略。
func NewResolver(cfg Config) *Resolver {
	r := &Resolver{
		exact:     make(map[netip.Addr]struct{}),
		private:   cfg.TrustPrivate,
		loopback:  cfg.TrustLoopback,
		linkLocal: cfg.TrustLinkLocal,
	}
	for _, raw := range cfg.Proxies {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.Contains(raw, "/") {
			prefix, err := netip.ParsePrefix(raw)
			if err != nil {
				continue
			}
			r.nets = append(r.nets, prefix.Masked())
			continue
		}
		addr, err := netip.ParseAddr(raw)
		if err != nil {
			continue
		}
		r.exact[addr] = struct{}{}
	}
	return r
}

// FromCtx 使用默认解析器从 Fiber 上下文取真实客户端 IP。
func FromCtx(c fiber.Ctx) string {
	return Default().FromCtx(c)
}

// FromCtx 从 Fiber 上下文解析真实客户端 IP。
func (r *Resolver) FromCtx(c fiber.Ctx) string {
	if c == nil {
		return ""
	}
	remote := remoteIPString(c)
	return r.Resolve(remote, func(name string) string {
		return c.Get(name)
	})
}

// Resolve 纯函数入口，便于单测：remote 为 TCP 对端，getHeader 读取请求头。
func (r *Resolver) Resolve(remote string, getHeader func(string) string) string {
	if r == nil {
		r = Default()
	}
	remoteAddr, remoteOK := parseAddr(remote)
	if !remoteOK {
		return ""
	}

	// 对端不是信任代理：绝不采信可伪造的转发头。
	if !r.isTrusted(remoteAddr) {
		return remoteAddr.String()
	}

	if getHeader != nil {
		// 1) CDN / 反代单值头：取第一个合法且非信任代理的地址。
		for _, name := range preferredClientHeaders {
			if ip, ok := parseAddr(getHeader(name)); ok && !r.isTrusted(ip) {
				return ip.String()
			}
		}

		// 2) X-Forwarded-For：从右往左剥信任代理。
		if client, ok := r.clientFromXFF(getHeader(headerXForwardedFor)); ok {
			return client.String()
		}
	}

	return remoteAddr.String()
}

// clientFromXFF 按「右→左」剥离信任代理，返回链中最右侧的非信任 IP。
// 例：client, proxy1, proxy2（最右为最近一跳）→ 剥 proxy2/proxy1 → client。
func (r *Resolver) clientFromXFF(raw string) (netip.Addr, bool) {
	parts := strings.Split(raw, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		ip, ok := parseAddr(parts[i])
		if !ok {
			continue
		}
		if r.isTrusted(ip) {
			continue
		}
		return ip, true
	}
	return netip.Addr{}, false
}

func (r *Resolver) isTrusted(addr netip.Addr) bool {
	if r == nil {
		return false
	}
	if _, ok := r.exact[addr]; ok {
		return true
	}
	for _, prefix := range r.nets {
		if prefix.Contains(addr) {
			return true
		}
	}
	if r.loopback && addr.IsLoopback() {
		return true
	}
	if r.private && addr.IsPrivate() {
		return true
	}
	if r.linkLocal && (addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast()) {
		return true
	}
	return false
}

// Normalize 把任意 IP 文本规范为 netip 字符串形式；非法则返回空串。
func Normalize(ip string) string {
	addr, ok := parseAddr(ip)
	if !ok {
		return ""
	}
	return addr.String()
}

// Mask 生成展示用脱敏前缀。
// IPv4：保留前 3 段（1.2.3.*）；IPv6：保留 /48 前缀（2001:db8:1:*）。
// 非法输入返回空串。
func Mask(ip string) string {
	addr, ok := parseAddr(ip)
	if !ok {
		return ""
	}
	if v4 := addr.AsSlice(); addr.Is4() || addr.Is4In6() {
		if len(v4) == 16 {
			// IPv4-mapped IPv6：取末 4 字节
			v4 = v4[12:]
		}
		if len(v4) != 4 {
			return ""
		}
		return fmt.Sprintf("%d.%d.%d.*", v4[0], v4[1], v4[2])
	}
	// IPv6：展示 /48 网络前缀 + :*
	prefix, err := addr.Prefix(48)
	if err != nil {
		return ""
	}
	// 取前 3 个 hextet（6 字节）
	b := prefix.Addr().As16()
	return fmt.Sprintf("%x:%x:%x:*",
		uint16(b[0])<<8|uint16(b[1]),
		uint16(b[2])<<8|uint16(b[3]),
		uint16(b[4])<<8|uint16(b[5]),
	)
}

func parseAddr(raw string) (netip.Addr, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return netip.Addr{}, false
	}
	// 去掉可能附带的端口（少见，但 XFF 偶发 host:port）
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}
	// 方括号 IPv6
	raw = strings.Trim(raw, "[]")
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr, true
}

func remoteIPString(c fiber.Ctx) string {
	// 直接取 TCP 对端，避免在配置 TrustProxy 前被 Fiber 的 c.IP() 混入未校验头。
	if ip := c.RequestCtx().RemoteIP(); ip != nil {
		return ip.String()
	}
	// 回退：部分测试 mock 可能只填 c.IP()
	return strings.TrimSpace(c.IP())
}

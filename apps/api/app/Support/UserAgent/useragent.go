// Package useragent 提供 User-Agent 与 IP 的展示层解析，供会话目录登记设备信息。
// 仅用于展示（device_name/browser/os/ip_prefix），不参与鉴权决策。
package useragent

import (
	"net"
	"strings"

	"github.com/mileusna/useragent"
)

// DeviceInfo 是从请求 UA/IP 解析出的展示信息。
type DeviceInfo struct {
	DeviceName   string // 组合展示名，如 "Chrome on macOS"
	Browser      string // 浏览器名，如 "Chrome"
	OS           string // 操作系统名，如 "macOS"
	UserAgentRaw string // 截断后的原始 UA（展示/排查用，不暴露给前端）
	IPPrefix     string // 脱敏 IP 前缀，如 "1.2.3.*"
}

// Parse 解析 UA 与 IP 为展示信息。
// ua / ip 为空时各字段为空串（前端展示「未知」），不报错。
// rawUA 超过 512 字节会被截断，避免异常 UA 占满存储。
func Parse(ua string, ip string) DeviceInfo {
	info := DeviceInfo{
		UserAgentRaw: truncate(strings.TrimSpace(ua), 512),
		IPPrefix:     maskIP(strings.TrimSpace(ip)),
	}
	if ua != "" {
		parsed := useragent.Parse(ua)
		info.Browser = parsed.Name
		info.OS = parsed.OS
		info.DeviceName = buildDeviceName(parsed.Name, parsed.OS)
	}
	return info
}

// buildDeviceName 构造组合展示名，如 "Chrome on macOS"。
// browser/os 任一为空时退化为非空的那一个，都空则返回空串。
func buildDeviceName(browser, os string) string {
	browser = strings.TrimSpace(browser)
	os = strings.TrimSpace(os)
	switch {
	case browser != "" && os != "":
		return browser + " on " + os
	case browser != "":
		return browser
	case os != "":
		return os
	default:
		return ""
	}
}

// maskIP 对 IPv4 脱敏：保留前 3 段，末段替换为 *（如 1.2.3.4 -> 1.2.3.*）。
// IPv6 或无法解析的地址返回空串（不暴露原始地址，前端展示「未知」）。
// 这是展示层脱敏：原始 IP 的哈希另由审计/安全流程处理，这里只产出展示用的前缀。
func maskIP(ip string) string {
	if ip == "" {
		return ""
	}
	// 仅对 IPv4 做前缀脱敏；IPv6 结构复杂，统一返回空串避免误暴露。
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() == nil {
		return ""
	}
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ""
	}
	return parts[0] + "." + parts[1] + "." + parts[2] + ".*"
}

func truncate(value string, max int) string {
	if max <= 0 || len([]rune(value)) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[:max])
}

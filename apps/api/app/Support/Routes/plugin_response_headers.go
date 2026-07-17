package routes

import (
	"net/http"
	"strings"
)

// FilterPluginResponseHeaders 统一收口插件终端响应头。Location 由 add/replace
// 终端处理器保留；会话、重放证据和 Host 保留元数据不能由插件写回客户端。
func FilterPluginResponseHeaders(source http.Header) http.Header {
	blocked := pluginResponseConnectionHeaders(source)
	result := make(http.Header)
	for name, values := range source {
		canonical := strings.ToLower(strings.TrimSpace(name))
		if _, denied := blocked[canonical]; denied || !pluginResponseHeaderAllowed(canonical) {
			continue
		}
		for _, value := range values {
			result.Add(name, value)
		}
	}
	return result
}

func pluginResponseHeaderAllowed(canonical string) bool {
	if strings.HasPrefix(canonical, "x-sforum-") {
		return false
	}
	switch canonical {
	case "", "content-length", "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
		"proxy-connection", "set-cookie", "link", "idempotency-replayed", "te", "trailer",
		"transfer-encoding", "upgrade":
		return false
	default:
		return true
	}
}

func pluginResponseConnectionHeaders(headers http.Header) map[string]struct{} {
	blocked := make(map[string]struct{})
	for name, values := range headers {
		if !strings.EqualFold(strings.TrimSpace(name), "Connection") {
			continue
		}
		for _, value := range values {
			for _, token := range strings.Split(value, ",") {
				if canonical := strings.ToLower(strings.TrimSpace(token)); canonical != "" {
					blocked[canonical] = struct{}{}
				}
			}
		}
	}
	return blocked
}

package pages

import "strings"

// ReservedPrefixes 不可被主题/插件 add 页面占用的路径前缀（locale 剥离后）。
var ReservedPrefixes = []string{
	"/control-panel",
	"/admin",
	"/api",
	"/_nuxt",
	"/__nuxt",
	"/__sforum",
	"/_sforum",
	"/health",
}

// IsReservedPath 判断候选路径是否落在保留前缀上。
func IsReservedPath(path string) bool {
	return isReservedPath(path)
}

func isReservedPath(path string) bool {
	path = stripLocalePrefix(path)
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	lower := strings.ToLower(path)
	for _, prefix := range ReservedPrefixes {
		if lower == prefix || strings.HasPrefix(lower, prefix+"/") {
			return true
		}
	}
	// API 鉴权附件等安全路径（即使未来挂到非 /api 前缀也应拒绝覆盖）
	if strings.HasPrefix(lower, "/api/v1/auth/") || strings.HasPrefix(lower, "/api/v1/attachments/") {
		return true
	}
	return false
}

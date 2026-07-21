// Package pages 是宿主 Page Registry 的权威目录与解析入口。
// 每个核心公开页有稳定 id；主题/插件通过贡献替换或新增视图，不能覆盖核心 API。
package pages

import (
	"fmt"
	"sort"
	"strings"
)

// Access 描述页面访问类别（与 manifest route access 对齐，便于 L1 贡献校验）。
// 未知非空值必须在安装/启用预检时失败（fail closed），不得按 public 放行。
type Access string

const (
	AccessPublic     Access = "public"
	AccessLogin      Access = "login"
	AccessGuest      Access = "guest"
	AccessModeration Access = "moderation"
	AccessPermission Access = "permission"
)

// NormalizeAccess 空值 → public；仅允许已知枚举。
func NormalizeAccess(raw string) (Access, error) {
	v := strings.TrimSpace(strings.ToLower(raw))
	if v == "" {
		return AccessPublic, nil
	}
	switch Access(v) {
	case AccessPublic, AccessLogin, AccessGuest, AccessModeration, AccessPermission:
		return Access(v), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidAccess, raw)
	}
}

// ValidAccess 是否为已规范化的合法 access。
func ValidAccess(a Access) bool {
	_, err := NormalizeAccess(string(a))
	return err == nil
}

// ProviderCore 始终是内置回退提供者。
const ProviderCore = "core"

// PageDefinition 是目录中的一条核心页面定义。
type PageDefinition struct {
	ID               string   `json:"id"`
	PathPattern      string   `json:"pathPattern"`
	Access           Access   `json:"access"`
	ContractVersion  string   `json:"contractVersion"`
	CoreComponent    string   `json:"coreComponent"`
	Replaceable      bool     `json:"replaceable"`
	RequiresFeatures []string `json:"requiresFeatures,omitempty"`
	Notes            string   `json:"notes,omitempty"`
}

// ResolvedPage 是解析结果：目录条目 + 当前生效的提供者。
type ResolvedPage struct {
	Page           PageDefinition `json:"page"`
	Provider       string         `json:"provider"`
	ExtensionID    string         `json:"extensionId,omitempty"`
	ContributionID string         `json:"contributionId,omitempty"`
	Action         string         `json:"action,omitempty"` // core | replace | add
	Fallback       bool           `json:"fallback"`
	// L1 渲染载荷（provider 非 core 时由 HTTP 层填充 HTML）。
	TemplatePath string `json:"templatePath,omitempty"`
	TemplateHTML string `json:"templateHtml,omitempty"`
	DataSource   string `json:"dataSource,omitempty"`
	DataRoute    string `json:"dataRoute,omitempty"`
	// DataSchema 为包内相对路径；replace 必须带入 ResolvedPage，否则 LoadForResolved 会丢 schema 校验。
	DataSchema        string `json:"dataSchema,omitempty"`
	Version           string `json:"version,omitempty"`
	PackageDigest     string `json:"packageDigest,omitempty"`
	RuntimeInstanceID string `json:"runtimeInstanceId,omitempty"`
	PackageRoot       string `json:"-"` // 仅服务端内部使用，不序列化
}

var coreCatalog = []PageDefinition{
	{ID: "forum.home", PathPattern: "/", Access: AccessPublic, ContractVersion: "sforum.page.home@1", CoreComponent: "pages/index", Replaceable: true},
	{ID: "forum.category.index", PathPattern: "/categories", Access: AccessPublic, ContractVersion: "sforum.page.category_index@1", CoreComponent: "pages/categories/index", Replaceable: true},
	{ID: "forum.category.show", PathPattern: "/c/:categorySlug", Access: AccessPublic, ContractVersion: "sforum.page.category_show@1", CoreComponent: "pages/c/[categorySlug]", Replaceable: true},
	{ID: "forum.tag.index", PathPattern: "/tags", Access: AccessPublic, ContractVersion: "sforum.page.tag_index@1", CoreComponent: "pages/tags/index", Replaceable: true, Notes: "gated by forum.tags.public_pages"},
	{ID: "forum.tag.show", PathPattern: "/tags/:tagSlug", Access: AccessPublic, ContractVersion: "sforum.page.tag_show@1", CoreComponent: "pages/tags/[tagSlug]", Replaceable: true, Notes: "gated by forum.tags.public_pages"},
	{ID: "forum.topic.show", PathPattern: "/t/:path(.*)", Access: AccessPublic, ContractVersion: "sforum.page.topic_show@1", CoreComponent: "pages/t/[...path]", Replaceable: true, Notes: "seo.topic_url_mode"},
	{ID: "forum.topic.create", PathPattern: "/topics/new", Access: AccessLogin, ContractVersion: "sforum.page.topic_create@1", CoreComponent: "pages/topics/new", Replaceable: true},
	{ID: "forum.profile.show", PathPattern: "/u/:username", Access: AccessPublic, ContractVersion: "sforum.page.profile_show@1", CoreComponent: "pages/u/[username]", Replaceable: true, RequiresFeatures: []string{"features.public_profiles"}},
	{ID: "forum.settings.profile", PathPattern: "/settings/profile", Access: AccessLogin, ContractVersion: "sforum.page.settings_profile@1", CoreComponent: "pages/settings/profile", Replaceable: true},
	{ID: "forum.settings.security", PathPattern: "/settings/security", Access: AccessLogin, ContractVersion: "sforum.page.settings_security@1", CoreComponent: "pages/settings/security", Replaceable: true},
	{ID: "forum.notifications", PathPattern: "/notifications", Access: AccessLogin, ContractVersion: "sforum.page.notifications@1", CoreComponent: "pages/notifications", Replaceable: true},
	{ID: "moderation.review", PathPattern: "/moderation", Access: AccessModeration, ContractVersion: "sforum.page.moderation_review@1", CoreComponent: "pages/moderation/index", Replaceable: false},
	{ID: "auth.login", PathPattern: "/login", Access: AccessGuest, ContractVersion: "sforum.page.login@1", CoreComponent: "pages/login", Replaceable: true, Notes: "replace must embed host login form island"},
	{ID: "auth.register", PathPattern: "/register", Access: AccessGuest, ContractVersion: "sforum.page.register@1", CoreComponent: "pages/register", Replaceable: true, RequiresFeatures: []string{"features.registration"}},
	{ID: "auth.forgot_password", PathPattern: "/forgot-password", Access: AccessPublic, ContractVersion: "sforum.page.forgot_password@1", CoreComponent: "pages/forgot-password", Replaceable: true},
	{ID: "auth.reset_password", PathPattern: "/reset-password", Access: AccessPublic, ContractVersion: "sforum.page.reset_password@1", CoreComponent: "pages/reset-password", Replaceable: true},
	{ID: "site.terms", PathPattern: "/terms", Access: AccessPublic, ContractVersion: "sforum.page.terms@1", CoreComponent: "pages/terms", Replaceable: true},
	{ID: "site.privacy", PathPattern: "/privacy", Access: AccessPublic, ContractVersion: "sforum.page.privacy@1", CoreComponent: "pages/privacy", Replaceable: true},
	{ID: "site.guidelines", PathPattern: "/guidelines", Access: AccessPublic, ContractVersion: "sforum.page.guidelines@1", CoreComponent: "pages/guidelines", Replaceable: true},
	{ID: "system.not_found", PathPattern: "", Access: AccessPublic, ContractVersion: "sforum.page.not_found@1", CoreComponent: "error", Replaceable: true},
	{ID: "dev.components", PathPattern: "/components", Access: AccessPublic, ContractVersion: "sforum.page.dev_components@1", CoreComponent: "pages/components", Replaceable: false, Notes: "dev gallery"},
}

// Catalog 返回核心页面目录的稳定副本（按 id 排序）。
func Catalog() []PageDefinition {
	out := make([]PageDefinition, len(coreCatalog))
	copy(out, coreCatalog)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Find 按 id 查找核心页面定义。
func Find(id string) (PageDefinition, bool) {
	id = strings.TrimSpace(id)
	for _, page := range coreCatalog {
		if page.ID == id {
			return page, true
		}
	}
	return PageDefinition{}, false
}

// ValidateCatalog 检查目录不变量（启动与单测使用）。
func ValidateCatalog() error {
	seenID := map[string]struct{}{}
	seenPath := map[string]struct{}{}
	for _, page := range coreCatalog {
		if page.ID == "" {
			return fmt.Errorf("pages: empty page id")
		}
		if _, ok := seenID[page.ID]; ok {
			return fmt.Errorf("pages: duplicate id %q", page.ID)
		}
		seenID[page.ID] = struct{}{}
		if page.PathPattern != "" {
			if _, ok := seenPath[page.PathPattern]; ok {
				return fmt.Errorf("pages: duplicate path %q", page.PathPattern)
			}
			seenPath[page.PathPattern] = struct{}{}
			if isReservedPath(page.PathPattern) {
				return fmt.Errorf("pages: core path collides with reserved prefix %q", page.PathPattern)
			}
		}
		if page.ContractVersion == "" || page.CoreComponent == "" {
			return fmt.Errorf("pages: %s missing contract or core component", page.ID)
		}
	}
	return nil
}

// ResolveCore 在无扩展贡献时返回始终-core 的解析结果。
func ResolveCore(id string) (ResolvedPage, error) {
	page, ok := Find(id)
	if !ok {
		return ResolvedPage{}, fmt.Errorf("pages: unknown page id %q", id)
	}
	return ResolvedPage{
		Page:     page,
		Provider: ProviderCore,
		Action:   "core",
		Fallback: false,
	}, nil
}

// MatchPath 在去掉可选 locale 前缀后，按目录 path pattern 做简单匹配。
// 仅用于管理/调试列表；请求路由仍由 Nuxt 文件路由负责。
func MatchPath(requestPath string) (PageDefinition, bool) {
	path := stripLocalePrefix(requestPath)
	if path == "" {
		path = "/"
	}
	for _, page := range coreCatalog {
		if page.PathPattern == "" {
			continue
		}
		if pathMatches(page.PathPattern, path) {
			return page, true
		}
	}
	return PageDefinition{}, false
}

func stripLocalePrefix(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// 仅剥离 /en 或 /en-US 这类前缀；默认 locale 无前缀。
	parts := strings.Split(strings.TrimPrefix(p, "/"), "/")
	if len(parts) == 0 {
		return "/"
	}
	first := parts[0]
	if looksLikeLocale(first) {
		rest := strings.Join(parts[1:], "/")
		if rest == "" {
			return "/"
		}
		return "/" + rest
	}
	return p
}

func looksLikeLocale(s string) bool {
	// en, en-US, zh-CN
	if len(s) == 2 {
		return s[0] >= 'a' && s[0] <= 'z' && s[1] >= 'a' && s[1] <= 'z'
	}
	if len(s) == 5 && s[2] == '-' {
		return looksLikeLocale(s[:2]) && s[3] >= 'A' && s[3] <= 'Z' && s[4] >= 'A' && s[4] <= 'Z'
	}
	return false
}

func pathMatches(pattern, path string) bool {
	if pattern == path {
		return true
	}
	// 简化：:param 与 catch-all 分段匹配
	pp := strings.Split(strings.Trim(pattern, "/"), "/")
	ps := strings.Split(strings.Trim(path, "/"), "/")
	if pattern == "/" {
		return path == "/"
	}
	if strings.Contains(pattern, "(.*)") || strings.Contains(pattern, "...") {
		// catch-all：前缀段相等即可
		if len(pp) == 0 {
			return true
		}
		prefix := pp[0]
		if strings.HasPrefix(prefix, ":") {
			return len(ps) >= 1
		}
		return len(ps) >= 1 && ps[0] == prefix
	}
	if len(pp) != len(ps) {
		return false
	}
	for i := range pp {
		if strings.HasPrefix(pp[i], ":") {
			continue
		}
		if pp[i] != ps[i] {
			return false
		}
	}
	return true
}

package pagescontroller

import (
	"errors"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
)

// Controller 暴露 Page Registry 管理与公开解析 API。
type Controller struct {
	registry *pages.Registry
	users    identity.ActorStore
	sessions *authsession.Manager
	themes   extensions.Store
	auditor  audit.Writer
}

func NewController(registry *pages.Registry, users identity.ActorStore, sessions *authsession.Manager) *Controller {
	return &Controller{registry: registry, users: users, sessions: sessions}
}

func NewControllerWithThemes(registry *pages.Registry, users identity.ActorStore, sessions *authsession.Manager, themes extensions.Store) *Controller {
	return &Controller{registry: registry, users: users, sessions: sessions, themes: themes}
}

// WithAuditor 注入 audit_events 写入（批准/恢复必须审计）。
func (h *Controller) WithAuditor(w audit.Writer) *Controller {
	if h != nil {
		h.auditor = w
	}
	return h
}

func (h *Controller) RegisterRoutes(api fiber.Router) {
	// 公开：解析单个页面（前台 outlet / SSR 使用）
	api.Get("/pages/resolve", h.resolve) // ?id=forum.home
	api.Get("/pages/resolve-path", h.resolvePath) // ?path=/docs/x
	api.Get("/pages/catalog", h.publicCatalog)
	api.Get("/site/active-theme/skin", h.activeSkin)
	api.Get("/site/theme-assets/:extensionId/*", h.themeAsset)

	admin := api.Group("/admin/pages")
	admin.Get("", h.adminList)
	admin.Get("/added", h.adminAdded)
	admin.Get("/activate-preview/:extensionId", h.activatePreview)
	admin.Get("/:pageId", h.adminGet)
	admin.Post("/:pageId/approve", h.adminApprove)
	admin.Post("/:pageId/restore-core", h.adminRestore)
}

type resolveResponse struct {
	Page           pages.PageDefinition `json:"page"`
	Provider       string               `json:"provider"`
	ExtensionID    string               `json:"extensionId,omitempty"`
	ContributionID string               `json:"contributionId,omitempty"`
	Action         string               `json:"action"`
	Fallback       bool                 `json:"fallback"`
	TemplatePath   string               `json:"templatePath,omitempty"`
	TemplateHTML   string               `json:"templateHtml,omitempty"`
	DataSource     string               `json:"dataSource,omitempty"`
	DataRoute      string               `json:"dataRoute,omitempty"`
	LoaderData     any                  `json:"loaderData,omitempty"`
	LoaderError    string               `json:"loaderError,omitempty"`
}

func (h *Controller) resolve(c fiber.Ctx) error {
	pageID := strings.TrimSpace(c.Query("id"))
	if pageID == "" {
		pageID = strings.TrimSpace(c.Params("pageId"))
	}
	if pageID == "" {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "pages.id_required")
	}

	var resolved pages.ResolvedPage
	var err error
	if h.registry != nil {
		resolved, err = h.registry.Resolve(c.Context(), pageID)
	} else {
		resolved, err = pages.ResolveCore(pageID)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pages.not_found")
	}
	// L1：非 core 时尽力加载模板 HTML；失败则标记 fallback，前台仍渲染 core slot。
	if resolved.Provider != pages.ProviderCore && resolved.TemplatePath != "" && h.themes != nil {
		extID := resolved.ExtensionID
		if extID == "" {
			extID = resolved.Provider
		}
		if theme, terr := h.themes.Get(c.Context(), extID); terr == nil && strings.TrimSpace(theme.PackagePath) != "" {
			if html, lerr := pages.LoadTemplate(theme.PackagePath, resolved.TemplatePath); lerr == nil {
				// 无 ViewModel 时用空 vars 渲染（仅宿主岛 + 静态 HTML）。
				if rendered, rerr := pages.RenderTemplate(html, map[string]string{}); rerr == nil {
					resolved.TemplateHTML = rendered
				} else {
					resolved.Fallback = true
					resolved.Provider = pages.ProviderCore
					resolved.TemplateHTML = ""
				}
			} else {
				resolved.Fallback = true
				resolved.Provider = pages.ProviderCore
			}
		}
	}
	return apphttp.OK(c, resolveResponse{
		Page:           resolved.Page,
		Provider:       resolved.Provider,
		ExtensionID:    resolved.ExtensionID,
		ContributionID: resolved.ContributionID,
		Action:         resolved.Action,
		Fallback:       resolved.Fallback,
		TemplatePath:   resolved.TemplatePath,
		TemplateHTML:   resolved.TemplateHTML,
		DataSource:     resolved.DataSource,
		DataRoute:      resolved.DataRoute,
	})
}

func (h *Controller) publicCatalog(c fiber.Ctx) error {
	items := pages.Catalog()
	out := make([]pages.PageDefinition, 0, len(items))
	for _, p := range items {
		if p.ID == "dev.components" {
			continue
		}
		out = append(out, p)
	}
	return apphttp.OK(c, out)
}

func (h *Controller) adminList(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if !canViewPages(actor) {
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	}
	if h.registry == nil {
		items := pages.Catalog()
		list := make([]pages.ProviderListItem, 0, len(items))
		for _, p := range items {
			list = append(list, pages.ProviderListItem{Page: p, Provider: pages.ProviderCore})
		}
		return apphttp.OK(c, list)
	}
	list, err := h.registry.ListProviders(c.Context())
	if err != nil {
		return err
	}
	return apphttp.OK(c, list)
}

func (h *Controller) adminGet(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if !canViewPages(actor) {
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	}
	pageID := c.Params("pageId")
	var resolved pages.ResolvedPage
	if h.registry != nil {
		resolved, err = h.registry.Resolve(c.Context(), pageID)
	} else {
		resolved, err = pages.ResolveCore(pageID)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pages.not_found")
	}
	return apphttp.OK(c, resolved)
}

type approveRequest struct {
	ExtensionID     string `json:"extensionId"`
	ContributionID  string `json:"contributionId"`
	Version         string `json:"version"`
	PackageDigest   string `json:"packageDigest"`
	TemplatePath    string `json:"templatePath"`
	ContractVersion string `json:"contractVersion"`
}

func (h *Controller) adminApprove(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	// 核心页替换仅 super_admin；extension.theme.manage 不可绕过。
	if !actor.IsSuperAdmin() {
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	}
	if h.registry == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "pages.registry_unavailable")
	}
	var body approveRequest
	if err := c.Bind().Body(&body); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "pages.invalid_body")
	}
	pageID := c.Params("pageId")
	err = h.registry.ApproveReplace(c.Context(), pages.ProviderBinding{
		PageID:          pageID,
		ExtensionID:     body.ExtensionID,
		ContributionID:  body.ContributionID,
		Version:         body.Version,
		PackageDigest:   body.PackageDigest,
		ContractVersion: body.ContractVersion,
		ApprovedBy:      actor.ID,
		TemplatePath:    body.TemplatePath,
	})
	if err != nil {
		return mapPagesError(err)
	}
	h.appendPageAudit(c, actor, audit.ActionPageReplaceApprove, map[string]any{
		"pageId":          pageID,
		"extensionId":     body.ExtensionID,
		"contributionId":  body.ContributionID,
		"version":         body.Version,
		"packageDigest":   body.PackageDigest,
		"contractVersion": body.ContractVersion,
	})
	return apphttp.OK(c, map[string]any{"pageId": pageID, "provider": body.ExtensionID})
}

func (h *Controller) adminRestore(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	// 恢复核心页同样仅 super_admin，避免 theme.manage 静默撤销替换。
	if !actor.IsSuperAdmin() {
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	}
	if h.registry == nil {
		return apphttp.OK(c, map[string]any{"pageId": c.Params("pageId"), "provider": pages.ProviderCore})
	}
	pageID := c.Params("pageId")
	if err := h.registry.RestoreCore(c.Context(), pageID); err != nil {
		return mapPagesError(err)
	}
	h.appendPageAudit(c, actor, audit.ActionPageRestoreCore, map[string]any{
		"pageId": pageID,
	})
	return apphttp.OK(c, map[string]any{"pageId": pageID, "provider": pages.ProviderCore})
}

func (h *Controller) appendPageAudit(c fiber.Ctx, actor identity.Actor, action string, metadata map[string]any) {
	if h == nil || h.auditor == nil {
		return
	}
	_ = h.auditor.Append(c.Context(), audit.Event{
		ActorUserID: actor.ID,
		Action:      action,
		Metadata:    metadata,
	})
}

// activatePreview 主题激活确认 UI：列出将新增/替换的页面、路径、安全等级与冲突。
func (h *Controller) activatePreview(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if !canViewPages(actor) {
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	}
	if h.themes == nil {
		return fiber.NewError(fiber.StatusNotFound, "pages.extension_not_found")
	}
	extID := c.Params("extensionId")
	theme, err := h.themes.Get(c.Context(), extID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pages.extension_not_found")
	}
	pkg, err := pages.LoadThemePackage(theme.PackagePath)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "pages.theme_package_invalid")
	}
	contribs := pages.ContributionsFromTheme(theme.ID, theme.Version, theme.PackageDigest, pkg)
	type impact struct {
		Contribution pages.PageContribution `json:"contribution"`
		Page         *pages.PageDefinition  `json:"page,omitempty"`
		Conflicts    []pages.PageContribution `json:"conflicts,omitempty"`
		RequiresApproval bool               `json:"requiresApproval"`
	}
	out := make([]impact, 0, len(contribs))
	for _, contrib := range contribs {
		item := impact{Contribution: contrib, RequiresApproval: contrib.Action == pages.ActionReplace}
		if contrib.Action == pages.ActionReplace {
			if page, ok := pages.Find(contrib.Target); ok {
				p := page
				item.Page = &p
			}
			if h.registry != nil {
				if list, _ := h.registry.ListProviders(c.Context()); list != nil {
					for _, p := range list {
						if p.Page.ID == contrib.Target {
							for _, cand := range p.Candidates {
								if cand.ExtensionID != contrib.ExtensionID {
									item.Conflicts = append(item.Conflicts, cand)
								}
							}
						}
					}
				}
			}
		}
		out = append(out, item)
	}
	return apphttp.OK(c, map[string]any{
		"extensionId":   theme.ID,
		"version":       theme.Version,
		"packageDigest": theme.PackageDigest,
		"impacts":       out,
		"note":          "Activation only registers candidates; core page replace requires super_admin approval.",
	})
}

// resolvePath 公开：按请求 path 解析 add 页面（动态公开路由）。
func (h *Controller) resolvePath(c fiber.Ctx) error {
	path := strings.TrimSpace(c.Query("path"))
	if path == "" {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "pages.path_required")
	}
	if pages.IsReservedPath(path) {
		return fiber.NewError(fiber.StatusNotFound, "pages.not_found")
	}
	if h.registry == nil {
		return fiber.NewError(fiber.StatusNotFound, "pages.not_found")
	}
	contrib, ok := h.registry.ResolveAddedPath(path)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "pages.not_found")
	}
	// access 权威检查
	switch contrib.Access {
	case pages.AccessLogin, pages.AccessModeration:
		actor, err := h.actor(c)
		if err != nil || actor.ID == 0 {
			return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
		}
		if contrib.Access == pages.AccessModeration && !actor.Can(identity.PermissionModerationReview) && !actor.IsSuperAdmin() {
			return fiber.NewError(fiber.StatusForbidden, "permission.denied")
		}
	case pages.AccessGuest:
		// guest-only：已登录也可看模板壳；mutation 仍由核心组件执行
	}

	resp := resolveResponse{
		Page: pages.PageDefinition{
			ID:              contrib.ID,
			PathPattern:     contrib.Path,
			Access:          contrib.Access,
			ContractVersion: contrib.Contract,
			Replaceable:     false,
		},
		Provider:       contrib.ExtensionID,
		ExtensionID:    contrib.ExtensionID,
		ContributionID: contrib.ID,
		Action:         string(pages.ActionAdd),
		Fallback:       false,
		TemplatePath:   contrib.Template,
		DataSource:     contrib.DataSource,
		DataRoute:      contrib.DataRoute,
	}
	// 加载模板
	if contrib.Template != "" && h.themes != nil {
		if theme, terr := h.themes.Get(c.Context(), contrib.ExtensionID); terr == nil {
			if html, lerr := pages.LoadTemplate(theme.PackagePath, contrib.Template); lerr == nil {
				if rendered, rerr := pages.RenderTemplate(html, map[string]string{}); rerr == nil {
					resp.TemplateHTML = rendered
				} else {
					resp.Fallback = true
				}
			} else {
				resp.Fallback = true
			}
		}
	}
	return apphttp.OK(c, resp)
}

func (h *Controller) adminAdded(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if !canViewPages(actor) {
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	}
	if h.registry == nil {
		return apphttp.OK(c, []pages.PageContribution{})
	}
	return apphttp.OK(c, h.registry.AddedPages())
}

func (h *Controller) activeSkin(c fiber.Ctx) error {
	if h.themes == nil {
		return apphttp.OK(c, pages.ActiveSkinPublic{CSS: []string{}})
	}
	theme, err := h.themes.ActiveTheme(c.Context())
	if err != nil {
		return apphttp.OK(c, pages.ActiveSkinPublic{CSS: []string{}})
	}
	skin, err := pages.SkinFromPackage(theme.ID, theme.Version, theme.PackageDigest, theme.PackagePath)
	if err != nil {
		return apphttp.OK(c, pages.ActiveSkinPublic{ExtensionID: theme.ID, CSS: []string{}})
	}
	return apphttp.OK(c, skin)
}

func (h *Controller) themeAsset(c fiber.Ctx) error {
	if h.themes == nil {
		return fiber.NewError(fiber.StatusNotFound, "pages.asset_not_found")
	}
	extensionID := c.Params("extensionId")
	rel := strings.TrimPrefix(c.Params("*"), "/")
	// 可选 digest 查询参数：?v=<packageDigest> 用于 immutable cache 与精确版本绑定
	wantDigest := strings.TrimSpace(c.Query("v"))
	if wantDigest == "" {
		wantDigest = strings.TrimSpace(c.Query("digest"))
	}

	// 仅允许当前活动主题的资源（禁止读取任意已安装主题）。
	active, err := h.themes.ActiveTheme(c.Context())
	if err != nil || active.ID != extensionID {
		return fiber.NewError(fiber.StatusNotFound, "pages.asset_not_found")
	}
	if active.Type != extensions.TypeTheme {
		return fiber.NewError(fiber.StatusNotFound, "pages.asset_not_found")
	}
	// 若请求携带 digest，必须与活动主题 package digest 精确匹配。
	if wantDigest != "" && active.PackageDigest != "" && !strings.EqualFold(wantDigest, active.PackageDigest) {
		return fiber.NewError(fiber.StatusNotFound, "pages.asset_digest_mismatch")
	}

	full, err := pages.ResolveThemeAsset(active.PackagePath, rel)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pages.asset_not_found")
	}
	ext := strings.ToLower(filepath.Ext(full))
	ctype, ok := pages.AllowedThemeAssetExt[ext]
	if !ok {
		// 明确禁止 SVG / JS / HTML 等可执行或高风险类型
		return fiber.NewError(fiber.StatusForbidden, "pages.asset_type_forbidden")
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pages.asset_not_found")
	}
	if ext == ".css" {
		if err := pages.ValidateCSS(string(raw)); err != nil {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "pages.css_invalid")
		}
	}
	if ctype == "" {
		ctype = mime.TypeByExtension(ext)
		if ctype == "" {
			ctype = "application/octet-stream"
		}
	}
	c.Set("Content-Type", ctype)
	c.Set("X-Content-Type-Options", "nosniff")
	// 非可执行资源：禁止被当作脚本解析
	c.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src 'self' data:; font-src 'self'")
	c.Set("X-Frame-Options", "DENY")
	if wantDigest != "" && active.PackageDigest != "" {
		// 精确 digest URL 可 immutable 缓存
		c.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		c.Set("Cache-Control", "public, max-age=300")
	}
	return c.Send(raw)
}

func (h *Controller) actor(c fiber.Ctx) (identity.Actor, error) {
	return apphttp.LoadActor(c, h.sessions, h.users)
}

func canViewPages(actor identity.Actor) bool {
	return actor.Can(identity.PermissionExtensionView) ||
		actor.Can(identity.PermissionExtensionThemeManage) ||
		actor.Can(identity.PermissionExtensionManage)
}

func mapPagesError(err error) error {
	switch {
	case errors.Is(err, pages.ErrUnknownPage):
		return fiber.NewError(fiber.StatusNotFound, "pages.not_found")
	case errors.Is(err, pages.ErrNotReplaceable):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "pages.not_replaceable")
	case errors.Is(err, pages.ErrReservedPath):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "pages.reserved_path")
	case errors.Is(err, pages.ErrInvalidContribution):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "pages.invalid_contribution")
	case errors.Is(err, pages.ErrApprovalRequired):
		return fiber.NewError(fiber.StatusForbidden, "pages.approval_required")
	case errors.Is(err, identity.ErrPermissionDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	default:
		return err
	}
}

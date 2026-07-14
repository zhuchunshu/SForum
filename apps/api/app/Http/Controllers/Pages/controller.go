package pagescontroller

import (
	"context"
	"errors"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
	themecompiler "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeCompiler"
)

// ThemePackageStore 页面解析所需的扩展包最小接口（避免测试实现完整 Store）。
type ThemePackageStore interface {
	Get(ctx context.Context, id string) (extensions.Extension, error)
	ActiveTheme(ctx context.Context) (extensions.Extension, error)
}

type pageViewerStore interface {
	GetCurrentUser(context.Context, int64) (identity.CurrentUser, error)
}

// Controller 暴露 Page Registry 管理与公开解析 API。
type Controller struct {
	registry *pages.Registry
	users    identity.ActorStore
	sessions *authsession.Manager
	themes   ThemePackageStore
	auditor  audit.Writer
	loader   *pages.LoaderGateway
	runtime  *pages.ThemeRuntimeRegistry
}

func (h *Controller) WithThemeRuntime(runtime *pages.ThemeRuntimeRegistry) *Controller {
	if h != nil {
		h.runtime = runtime
	}
	return h
}

func NewController(registry *pages.Registry, users identity.ActorStore, sessions *authsession.Manager) *Controller {
	return &Controller{registry: registry, users: users, sessions: sessions}
}

func NewControllerWithThemes(registry *pages.Registry, users identity.ActorStore, sessions *authsession.Manager, themes ThemePackageStore) *Controller {
	return &Controller{registry: registry, users: users, sessions: sessions, themes: themes}
}

// WithAuditor 注入 audit_events 写入（批准/恢复必须审计）。
func (h *Controller) WithAuditor(w audit.Writer) *Controller {
	if h != nil {
		h.auditor = w
	}
	return h
}

// WithLoader 注入受控 PageDataLoader 网关（生产 SSR）。
func (h *Controller) WithLoader(g *pages.LoaderGateway) *Controller {
	if h != nil {
		h.loader = g
	}
	return h
}

func (h *Controller) RegisterRoutes(api fiber.Router) {
	// 公开：解析单个页面（前台 outlet / SSR 使用）
	api.Get("/pages/resolve", h.resolve)          // ?id=forum.home
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
	Page           pages.PageDefinition     `json:"page"`
	Provider       string                   `json:"provider"`
	ExtensionID    string                   `json:"extensionId,omitempty"`
	ContributionID string                   `json:"contributionId,omitempty"`
	Action         string                   `json:"action"`
	Fallback       bool                     `json:"fallback"`
	TemplatePath   string                   `json:"templatePath,omitempty"`
	TemplateHTML   string                   `json:"templateHtml,omitempty"`
	DataSource     string                   `json:"dataSource,omitempty"`
	DataRoute      string                   `json:"dataRoute,omitempty"`
	RouteParams    map[string]string        `json:"routeParams,omitempty"`
	LoaderData     any                      `json:"loaderData,omitempty"`
	LoaderError    string                   `json:"loaderError,omitempty"`
	Contract       string                   `json:"contractVersion,omitempty"`
	RenderOutput   *pages.ThemeRenderedPage `json:"renderOutput,omitempty"`
}

func (h *Controller) resolve(c fiber.Ctx) error {
	pageID := strings.TrimSpace(c.Query("id"))
	if pageID == "" {
		pageID = strings.TrimSpace(string(c.Request().URI().QueryArgs().Peek("id")))
	}
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
		// DEBUG
	} else {
		resolved, err = pages.ResolveCore(pageID)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pages.not_found")
	}

	// 核心页 access：replace 后仍沿用目录 access
	if err := h.enforcePageAccess(c, resolved.Page.Access, ""); err != nil {
		return err
	}

	locale := strings.TrimSpace(c.Get("Accept-Language"))
	if i := strings.Index(locale, ","); i >= 0 {
		locale = locale[:i]
	}
	actorID := h.optionalActorID(c)

	var runtimeOutput *pages.ThemeRenderedPage
	runtimeCovered := false
	if resolved.Provider != pages.ProviderCore && h.runtime != nil {
		artifact := pages.RuntimeArtifact{
			ExtensionID: resolved.ExtensionID, ExtensionVersion: resolved.Version,
			PackageDigest: resolved.PackageDigest, RuntimeInstanceID: resolved.RuntimeInstanceID,
		}
		if snapshot, ok := h.runtime.Resolve(artifact, resolved.Page.ID, resolved.ContributionID); ok {
			runtimeCovered = true
			viewer, viewerErr := h.pageViewer(c)
			if viewerErr == nil {
				output, renderErr := snapshot.Render(c.Context(), pages.CorePageViewModelRequest{
					PageID: resolved.Page.ID, Locale: locale, Path: resolved.Page.PathPattern,
					Viewer: viewer, SEO: themecompiler.PageSEOView{Title: resolved.Page.ID},
				}, resolved.ContributionID)
				if renderErr == nil {
					resolved.TemplateHTML = snapshot.LegacyHTML(output)
					if resolved.TemplateHTML != "" {
						runtimeOutput = &output
						// 模板链回退仍是有效 L1；只有 emergency 才交还核心页面。
						resolved.Fallback = output.Source == pages.ThemeRenderSourceEmergency
					} else {
						resolved.Fallback = true
					}
				} else {
					resolved.Fallback = true
				}
			} else {
				resolved.Fallback = true
			}
		} else if h.runtime.Claims(resolved.ExtensionID, resolved.Page.ID, resolved.ContributionID) {
			// 精确 artifact 不匹配时禁止降级读取旧包文件。
			runtimeCovered = true
			resolved.Fallback = true
		}
	}

	// Unmigrated plugin/add contracts retain the explicit legacy L1 path until
	// P13. An exact compiled snapshot never falls back to request-time IO.
	if resolved.Provider != pages.ProviderCore && resolved.TemplatePath != "" && h.themes != nil {
		if !runtimeCovered {
			extID := resolved.ExtensionID
			if extID == "" {
				extID = resolved.Provider
			}
			if theme, terr := h.themes.Get(c.Context(), extID); terr == nil {
				root := extensions.PackageContentRoot(theme)
				if root != "" {
					if html, lerr := pages.LoadTemplate(root, resolved.TemplatePath); lerr == nil {
						vars := map[string]string{"locale": locale}
						if rendered, rerr := pages.RenderTemplate(html, vars); rerr == nil {
							resolved.TemplateHTML = rendered
						} else {
							resolved.Fallback = true
							resolved.TemplateHTML = ""
						}
					} else {
						resolved.Fallback = true
					}
				} else {
					resolved.Fallback = true
				}
			} else {
				resolved.Fallback = true
			}
		}
	}

	// Fallback 时对外 provider 显示 core，避免前台误用失败模板。
	provider := resolved.Provider
	action := resolved.Action
	if resolved.Fallback {
		provider = pages.ProviderCore
		if action == "" {
			action = "core"
		}
	}

	resp := resolveResponse{
		Page:           resolved.Page,
		Provider:       provider,
		ExtensionID:    resolved.ExtensionID,
		ContributionID: resolved.ContributionID,
		Action:         action,
		Fallback:       resolved.Fallback,
		TemplatePath:   resolved.TemplatePath,
		TemplateHTML:   resolved.TemplateHTML,
		DataSource:     resolved.DataSource,
		DataRoute:      resolved.DataRoute,
		Contract:       resolved.Page.ContractVersion,
		RenderOutput:   runtimeOutput,
	}

	// access 通过后才调用 loader
	if !resolved.Fallback && resolved.DataSource == "plugin" && resolved.DataRoute != "" && h.loader != nil {
		lr := h.loader.LoadForResolved(c.Context(), resolved, locale, actorID)
		if lr.Error != "" {
			resp.LoaderError = lr.Error
		} else if len(lr.Data) > 0 {
			resp.LoaderData = pages.DecodeLoaderData(lr.Data)
		}
	}

	return apphttp.OK(c, resp)
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
			list = append(list, pages.ProviderListItem{Page: p, Provider: pages.ProviderCore, ContractVersion: p.ContractVersion})
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
	ContractVersion string `json:"contractVersion"`
	// TemplatePath 已废弃：服务端从已注册贡献读取，忽略客户端值（防伪造）。
	TemplatePath string `json:"templatePath"`
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
	// Fiber 可能复用 Params 底层缓冲；入库前必须 Clone。
	pageID := strings.Clone(strings.TrimSpace(c.Params("pageId")))
	// 不把客户端 templatePath 写入绑定
	err = h.registry.ApproveReplace(c.Context(), pages.ProviderBinding{
		PageID:          pageID,
		ExtensionID:     body.ExtensionID,
		ContributionID:  body.ContributionID,
		Version:         body.Version,
		PackageDigest:   body.PackageDigest,
		ContractVersion: body.ContractVersion,
		ApprovedBy:      actor.ID,
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
	if !actor.IsSuperAdmin() {
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	}
	if h.registry == nil {
		return apphttp.OK(c, map[string]any{"pageId": c.Params("pageId"), "provider": pages.ProviderCore})
	}
	pageID := strings.Clone(strings.TrimSpace(c.Params("pageId")))
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
	pkg, err := pages.LoadThemePackage(extensions.PackageContentRoot(theme))
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "pages.theme_package_invalid")
	}
	contribs := pages.ContributionsFromTheme(theme.ID, theme.Version, theme.PackageDigest, pkg)
	type impact struct {
		Contribution     pages.PageContribution   `json:"contribution"`
		Page             *pages.PageDefinition    `json:"page,omitempty"`
		Conflicts        []pages.PageContribution `json:"conflicts,omitempty"`
		RequiresApproval bool                     `json:"requiresApproval"`
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
	match, ok := h.registry.ResolveAddedPathMatch(path)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "pages.not_found")
	}
	contrib := match.Contribution

	// access 权威检查（fail closed）
	if err := h.enforcePageAccess(c, contrib.Access, contrib.Permission); err != nil {
		return err
	}

	locale := strings.TrimSpace(c.Query("locale"))
	if locale == "" {
		locale = strings.TrimSpace(c.Get("Accept-Language"))
		if i := strings.Index(locale, ","); i >= 0 {
			locale = locale[:i]
		}
	}
	actorID := h.optionalActorID(c)

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
		RouteParams:    match.Params,
		Contract:       contrib.Contract,
	}

	// 加载模板（注入 route params）
	if contrib.Template != "" && h.themes != nil {
		if theme, terr := h.themes.Get(c.Context(), contrib.ExtensionID); terr == nil {
			root := extensions.PackageContentRoot(theme)
			if root != "" {
				if html, lerr := pages.LoadTemplate(root, contrib.Template); lerr == nil {
					vars := map[string]string{"locale": locale}
					for k, v := range match.Params {
						vars[k] = v
					}
					if rendered, rerr := pages.RenderTemplate(html, vars); rerr == nil {
						resp.TemplateHTML = rendered
					} else {
						resp.Fallback = true
					}
				} else {
					resp.Fallback = true
				}
			}
		}
	}

	// access 已通过 → loader
	if !resp.Fallback && contrib.DataRoute != "" && h.loader != nil {
		lr := h.loader.LoadForContribution(c.Context(), contrib, match.Params, locale, actorID)
		if lr.Error != "" {
			resp.LoaderError = lr.Error
		} else if len(lr.Data) > 0 {
			resp.LoaderData = pages.DecodeLoaderData(lr.Data)
		}
	}

	return apphttp.OK(c, resp)
}

// enforcePageAccess 严格校验 access；未知值拒绝。
func (h *Controller) enforcePageAccess(c fiber.Ctx, access pages.Access, permissionKey string) error {
	normalized, err := pages.NormalizeAccess(string(access))
	if err != nil {
		// 已注册贡献不应含未知 access；fail closed
		return fiber.NewError(fiber.StatusNotFound, "pages.not_found")
	}
	switch normalized {
	case pages.AccessPublic:
		return nil
	case pages.AccessLogin:
		actor, aerr := h.actor(c)
		if aerr != nil || actor.ID == 0 {
			return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
		}
		return nil
	case pages.AccessGuest:
		// 已登录用户：返回冲突，引导离开 guest 页（不模糊放行）
		if actor, aerr := h.optionalActor(c); aerr == nil && actor.ID > 0 {
			return fiber.NewError(fiber.StatusConflict, "pages.guest_only")
		}
		return nil
	case pages.AccessModeration:
		actor, aerr := h.actor(c)
		if aerr != nil || actor.ID == 0 {
			return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
		}
		if !actor.Can(identity.PermissionModerationReview) && !actor.IsSuperAdmin() {
			return fiber.NewError(fiber.StatusForbidden, "permission.denied")
		}
		return nil
	case pages.AccessPermission:
		actor, aerr := h.actor(c)
		if aerr != nil || actor.ID == 0 {
			return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
		}
		key := strings.TrimSpace(permissionKey)
		if key == "" {
			return fiber.NewError(fiber.StatusForbidden, "permission.denied")
		}
		if !actor.Can(key) && !actor.IsSuperAdmin() {
			return fiber.NewError(fiber.StatusForbidden, "permission.denied")
		}
		return nil
	default:
		return fiber.NewError(fiber.StatusNotFound, "pages.not_found")
	}
}

func (h *Controller) optionalActorID(c fiber.Ctx) int64 {
	actor, err := h.optionalActor(c)
	if err != nil {
		return 0
	}
	return actor.ID
}

func (h *Controller) pageViewer(c fiber.Ctx) (themecompiler.PageViewerState, error) {
	actor, err := h.optionalActor(c)
	if err != nil {
		return themecompiler.PageViewerState{}, err
	}
	if actor.ID == 0 {
		return themecompiler.PageViewerState{}, nil
	}
	store, ok := h.users.(pageViewerStore)
	if !ok {
		return themecompiler.PageViewerState{}, errors.New("pages: current user projection unavailable")
	}
	current, err := store.GetCurrentUser(c.Context(), actor.ID)
	if err != nil {
		return themecompiler.PageViewerState{}, err
	}
	return themecompiler.PageViewerState{
		Authenticated: true,
		UserID:        current.ID,
		Username:      current.Username,
		DisplayName:   current.DisplayName,
		AvatarURL:     current.Avatar.URL,
		// Actor 已包含 PAT scope 收窄结果，不能从 CurrentUser 恢复完整权限。
		Permissions: actorPermissionKeys(actor),
	}, nil
}

func actorPermissionKeys(actor identity.Actor) []string {
	permissions := make([]string, 0, len(actor.Permissions))
	for permission, allowed := range actor.Permissions {
		if allowed {
			permissions = append(permissions, permission)
		}
	}
	sort.Strings(permissions)
	return permissions
}

func (h *Controller) optionalActor(c fiber.Ctx) (identity.Actor, error) {
	return apphttp.OptionalActor(c, h.sessions, h.users)
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
	skin, err := pages.SkinFromPackage(theme.ID, theme.Version, theme.PackageDigest, extensions.PackageContentRoot(theme))
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
	wantDigest := strings.TrimSpace(c.Query("v"))
	if wantDigest == "" {
		wantDigest = strings.TrimSpace(c.Query("digest"))
	}

	active, err := h.themes.ActiveTheme(c.Context())
	if err != nil || active.ID != extensionID {
		return fiber.NewError(fiber.StatusNotFound, "pages.asset_not_found")
	}
	if active.Type != extensions.TypeTheme {
		return fiber.NewError(fiber.StatusNotFound, "pages.asset_not_found")
	}
	if wantDigest != "" && active.PackageDigest != "" && !strings.EqualFold(wantDigest, active.PackageDigest) {
		return fiber.NewError(fiber.StatusNotFound, "pages.asset_digest_mismatch")
	}

	full, err := pages.ResolveThemeAsset(extensions.PackageContentRoot(active), rel)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pages.asset_not_found")
	}
	ext := strings.ToLower(filepath.Ext(full))
	ctype, ok := pages.AllowedThemeAssetExt[ext]
	if !ok {
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
	c.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src 'self' data:; font-src 'self'")
	c.Set("X-Frame-Options", "DENY")
	if wantDigest != "" && active.PackageDigest != "" {
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
	case errors.Is(err, pages.ErrContractMismatch):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "pages.contract_mismatch")
	case errors.Is(err, pages.ErrInvalidAccess):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "pages.invalid_access")
	case errors.Is(err, pages.ErrApprovalRequired):
		return fiber.NewError(fiber.StatusForbidden, "pages.approval_required")
	case errors.Is(err, identity.ErrPermissionDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	default:
		return err
	}
}

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
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
)

// Controller 暴露 Page Registry 管理与公开解析 API。
type Controller struct {
	registry *pages.Registry
	users    identity.ActorStore
	sessions *authsession.Manager
	themes   extensions.Store
}

func NewController(registry *pages.Registry, users identity.ActorStore, sessions *authsession.Manager) *Controller {
	return &Controller{registry: registry, users: users, sessions: sessions}
}

func NewControllerWithThemes(registry *pages.Registry, users identity.ActorStore, sessions *authsession.Manager, themes extensions.Store) *Controller {
	return &Controller{registry: registry, users: users, sessions: sessions, themes: themes}
}

func (h *Controller) RegisterRoutes(api fiber.Router) {
	// 公开：解析单个页面（前台 outlet / SSR 使用）
	api.Get("/pages/resolve", h.resolve) // ?id=forum.home
	api.Get("/pages/catalog", h.publicCatalog)
	api.Get("/site/active-theme/skin", h.activeSkin)
	api.Get("/site/theme-assets/:extensionId/*", h.themeAsset)

	admin := api.Group("/admin/pages")
	admin.Get("", h.adminList)
	admin.Get("/added", h.adminAdded)
	admin.Get("/:pageId", h.adminGet)
	admin.Post("/:pageId/approve", h.adminApprove)
	admin.Post("/:pageId/restore-core", h.adminRestore)
}

type resolveResponse struct {
	Page        pages.PageDefinition `json:"page"`
	Provider    string               `json:"provider"`
	ExtensionID string               `json:"extensionId,omitempty"`
	Action      string               `json:"action"`
	Fallback    bool                 `json:"fallback"`
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
	return apphttp.OK(c, resolveResponse{
		Page:        resolved.Page,
		Provider:    resolved.Provider,
		ExtensionID: resolved.ExtensionID,
		Action:      resolved.Action,
		Fallback:    resolved.Fallback,
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
	ExtensionID    string `json:"extensionId"`
	ContributionID string `json:"contributionId"`
	Version        string `json:"version"`
	PackageDigest  string `json:"packageDigest"`
	TemplatePath   string `json:"templatePath"`
}

func (h *Controller) adminApprove(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
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
		PageID:         pageID,
		ExtensionID:    body.ExtensionID,
		ContributionID: body.ContributionID,
		Version:        body.Version,
		PackageDigest:  body.PackageDigest,
		ApprovedBy:     actor.ID,
		TemplatePath:   body.TemplatePath,
	})
	if err != nil {
		return mapPagesError(err)
	}
	return apphttp.OK(c, map[string]any{"pageId": pageID, "provider": body.ExtensionID})
}

func (h *Controller) adminRestore(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if !actor.IsSuperAdmin() && !actor.Can(identity.PermissionExtensionThemeManage) {
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	}
	if h.registry == nil {
		return apphttp.OK(c, map[string]any{"pageId": c.Params("pageId"), "provider": pages.ProviderCore})
	}
	pageID := c.Params("pageId")
	if err := h.registry.RestoreCore(c.Context(), pageID); err != nil {
		return mapPagesError(err)
	}
	return apphttp.OK(c, map[string]any{"pageId": pageID, "provider": pages.ProviderCore})
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
	skin, err := pages.SkinFromPackage(theme.ID, theme.Version, theme.PackagePath)
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
	theme, err := h.themes.Get(c.Context(), extensionID)
	if err != nil {
		// 也允许服务当前激活主题
		active, aerr := h.themes.ActiveTheme(c.Context())
		if aerr != nil || active.ID != extensionID {
			return fiber.NewError(fiber.StatusNotFound, "pages.asset_not_found")
		}
		theme = active
	}
	if theme.Type != extensions.TypeTheme {
		return fiber.NewError(fiber.StatusNotFound, "pages.asset_not_found")
	}
	full, err := pages.ResolveThemeAsset(theme.PackagePath, rel)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pages.asset_not_found")
	}
	// 仅允许 css 与常见静态资源
	ext := strings.ToLower(filepath.Ext(full))
	switch ext {
	case ".css", ".woff", ".woff2", ".ttf", ".otf", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico":
	default:
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
	ctype := mime.TypeByExtension(ext)
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	c.Set("Content-Type", ctype)
	c.Set("Cache-Control", "public, max-age=300")
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

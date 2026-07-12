package identitycontroller

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	apitokens "github.com/zhuchunshu/sforum/apps/api/app/Models/APITokens"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
)

type createAPITokenRequest struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresAt *string  `json:"expiresAt"`
}

func (h *Controller) listAPITokens(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if h.apiTokens == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "service.not_ready")
	}
	// PAT 管理只允许 cookie 会话，禁止用 PAT 创建/列出 PAT（降低窃取面）。
	if apitokens.TokenIDFromContext(c.Context()) > 0 {
		return fiber.NewError(fiber.StatusForbidden, "api_token.cookie_required")
	}
	includeRevoked := c.Query("includeRevoked") == "true"
	items, err := h.apiTokens.List(c.Context(), actor.ID, includeRevoked)
	if err != nil {
		return err
	}
	return apphttp.OK(c, map[string]any{"items": items})
}

func (h *Controller) createAPIToken(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if h.apiTokens == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "service.not_ready")
	}
	if apitokens.TokenIDFromContext(c.Context()) > 0 {
		return fiber.NewError(fiber.StatusForbidden, "api_token.cookie_required")
	}
	var req createAPITokenRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "api_token.invalid")
	}
	var expires *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "api_token.invalid")
		}
		utc := parsed.UTC()
		expires = &utc
	}
	created, err := h.apiTokens.Create(c.Context(), actor, apitokens.CreateInput{
		Name: req.Name, Scopes: req.Scopes, ExpiresAt: expires,
	})
	if err != nil {
		return mapAPITokenError(err)
	}
	return apphttp.Created(c, created)
}

func (h *Controller) revokeAPIToken(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if h.apiTokens == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "service.not_ready")
	}
	if apitokens.TokenIDFromContext(c.Context()) > 0 {
		return fiber.NewError(fiber.StatusForbidden, "api_token.cookie_required")
	}
	id, err := strconv.ParseInt(c.Params("tokenID"), 10, 64)
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusNotFound, "api_token.not_found")
	}
	if err := h.apiTokens.Revoke(c.Context(), actor, id); err != nil {
		return mapAPITokenError(err)
	}
	return apphttp.NoData(c)
}

func (h *Controller) rotateAPIToken(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if h.apiTokens == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "service.not_ready")
	}
	if apitokens.TokenIDFromContext(c.Context()) > 0 {
		return fiber.NewError(fiber.StatusForbidden, "api_token.cookie_required")
	}
	id, err := strconv.ParseInt(c.Params("tokenID"), 10, 64)
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusNotFound, "api_token.not_found")
	}
	created, err := h.apiTokens.Rotate(c.Context(), actor, id)
	if err != nil {
		return mapAPITokenError(err)
	}
	return apphttp.OK(c, created)
}

func mapAPITokenError(err error) error {
	switch {
	case errors.Is(err, apitokens.ErrTokenNotFound):
		return fiber.NewError(fiber.StatusNotFound, "api_token.not_found")
	case errors.Is(err, apitokens.ErrInvalidInput), errors.Is(err, apitokens.ErrScopeNotAllowed):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "api_token.invalid")
	case errors.Is(err, apitokens.ErrTokenRevoked):
		return fiber.NewError(fiber.StatusConflict, "api_token.revoked")
	default:
		return err
	}
}

// auditSensitiveWithToken 在 PAT 调用敏感路由时追加审计（可选 helper）。
func (h *Controller) auditSensitiveWithToken(c fiber.Ctx, action string, meta map[string]any) {
	if h.auditor == nil {
		return
	}
	userID, ok := apitokens.UserIDFromContext(c.Context())
	if !ok {
		return
	}
	if meta == nil {
		meta = map[string]any{}
	}
	meta["tokenId"] = apitokens.TokenIDFromContext(c.Context())
	meta["via"] = "api_token"
	_ = h.auditor.Append(c.Context(), audit.Event{ActorUserID: userID, Action: action, Metadata: meta})
}

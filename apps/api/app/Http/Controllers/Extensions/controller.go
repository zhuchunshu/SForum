package extensionscontroller

import (
	"context"
	"errors"
	"io"
	"strconv"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
)

const maxUploadedArchiveBytes = 60 * 1024 * 1024

type Controller struct {
	service     *extensions.Service
	frontend    TrustedFrontendService
	webReleases WebReleaseAdminService
	users       identity.ActorStore
	sessions    *authsession.Manager
	gateway     RouteGateway
}

type TrustedFrontendService interface {
	Frontend(context.Context, identity.Actor, string) (extensions.FrontendStatus, error)
	Grant(context.Context, identity.Actor, string, extensions.GrantFrontendInput) (extensions.ExtensionOperation, error)
	Revoke(context.Context, identity.Actor, string) (extensions.ExtensionOperation, error)
	RestoreDefaults(context.Context, identity.Actor) (extensions.ExtensionOperation, error)
}

type WebReleaseAdminService interface {
	List(context.Context, identity.Actor, extensions.WebReleaseListInput) (extensions.WebReleasePage, error)
	Detail(context.Context, identity.Actor, int64) (extensions.WebReleaseDetail, error)
	Retry(context.Context, identity.Actor, int64) (extensions.WebReleaseOperation, error)
	Rollback(context.Context, identity.Actor, int64) (extensions.WebReleaseOperation, error)
}

type ProxyInput struct {
	Matched  extensions.MatchedRoute
	Actor    identity.Actor
	HasActor bool
}

type RouteGateway interface {
	Proxy(c fiber.Ctx, input ProxyInput) error
}

type updateSettingsRequest struct {
	Values map[string]string `json:"values"`
}

func NewController(service *extensions.Service, users identity.ActorStore, sessions *authsession.Manager) *Controller {
	return NewControllerWithGateway(service, users, sessions, nil)
}

func NewControllerWithGateway(service *extensions.Service, users identity.ActorStore, sessions *authsession.Manager, gateway RouteGateway) *Controller {
	return &Controller{service: service, users: users, sessions: sessions, gateway: gateway}
}

func (h *Controller) WithTrustedRuntime(frontend TrustedFrontendService, webReleases WebReleaseAdminService) *Controller {
	h.frontend = frontend
	h.webReleases = webReleases
	return h
}

func (h *Controller) list(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.List(c.Context(), actor)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) navigation(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.Navigation(c.Context(), actor)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) install(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, extensions.CodeInvalidArchive)
	}
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxUploadedArchiveBytes+1))
	if err != nil {
		return err
	}
	result, err := h.service.InstallOrUpgradeArchive(c.Context(), actor, extensions.ArchiveInput{
		FileName: fileHeader.Filename,
		Data:     data,
	})
	if err != nil {
		return mapExtensionError(err)
	}
	// 兼容旧客户端：顶层仍是 Extension；升级元数据挂在 data 外层的 InstallResult。
	return apphttp.Created(c, result)
}

func (h *Controller) uninstall(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var input extensions.UninstallInput
	if len(c.Body()) > 0 {
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
		}
	}
	if err := h.service.Uninstall(c.Context(), actor, c.Params("id"), input); err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, map[string]any{"uninstalled": true, "extensionId": c.Params("id")})
}

func (h *Controller) listMigrations(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.ListMigrations(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) applyMigrations(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.ApplyDeclaredMigrations(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) enable(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var input extensions.EnableInput
	// body 可选；空 body 表示未确认 capabilities。
	if len(c.Body()) > 0 {
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
		}
	}
	item, err := h.service.EnableOperation(c.Context(), actor, c.Params("id"), input)
	if err != nil {
		return mapExtensionError(err)
	}
	return extensionOperationResponse(c, item)
}

func (h *Controller) disable(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	item, err := h.service.DisableOperation(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	return extensionOperationResponse(c, item)
}

func (h *Controller) verify(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	item, err := h.service.VerifyExtension(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, item)
}

func (h *Controller) activate(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	item, err := h.service.ActivateThemeOperation(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	return extensionOperationResponse(c, item)
}

func extensionOperationResponse(c fiber.Ctx, operation extensions.ExtensionOperation) error {
	if operation.Queued {
		return apphttp.JSON(c, fiber.StatusAccepted, apphttp.MessageOK, operation)
	}
	return apphttp.OK(c, operation)
}

func (h *Controller) events(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.Events(c.Context(), actor, c.Params("id"), queryInt(c, "limit", 50))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) eventDefinitions(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.EventDefinitions(c.Context(), actor)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) eventDeliveries(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.EventDeliveries(c.Context(), actor, extensions.EventDeliveryListInput{
		ExtensionID: c.Query("extensionId"),
		EventName:   c.Query("eventName"),
		Status:      c.Query("status"),
		Limit:       queryInt(c, "limit", 50),
	})
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) contributionPoints(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.ContributionPoints(c.Context(), actor)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) contributions(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.Contributions(c.Context(), actor)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) settings(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	settings, err := h.service.Settings(c.Context(), actor, c.Params("id"), apphttp.Locale(c))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, settings)
}

func (h *Controller) updateSettings(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req updateSettingsRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}
	settings, err := h.service.UpdateSettings(c.Context(), actor, c.Params("id"), extensions.UpdateSettingsInput{Values: req.Values}, apphttp.Locale(c))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, settings)
}

func (h *Controller) resetSettings(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	settings, err := h.service.ResetSettings(c.Context(), actor, c.Params("id"), apphttp.Locale(c))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, settings)
}

func queryInt(c fiber.Ctx, name string, fallback int) int {
	value, err := strconv.Atoi(c.Query(name))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func (h *Controller) proxyExtensionRoute(c fiber.Ctx) error {
	routePath := "/" + c.Params("*")
	matched, err := h.service.MatchRoute(c.Context(), c.Params("extensionId"), c.Method(), routePath)
	if err != nil {
		return mapExtensionError(err)
	}
	actor, hasActor, err := h.optionalActor(c)
	if err != nil {
		return err
	}
	access := matched.Route.Access
	if access == "" {
		access = extensions.RouteAccessLogin
	}
	switch access {
	case extensions.RouteAccessLogin:
		if !hasActor {
			return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
		}
	case extensions.RouteAccessPermission:
		if !hasActor {
			return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
		}
		if !actor.Can(matched.Route.Permission) {
			return fiber.NewError(fiber.StatusForbidden, "permission.denied")
		}
	}
	if h.gateway == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeRuntimeUnavailable)
	}
	if err := h.gateway.Proxy(c, ProxyInput{Matched: matched, Actor: actor, HasActor: hasActor}); err != nil {
		return mapExtensionError(err)
	}
	return nil
}

func (h *Controller) actor(c fiber.Ctx) (identity.Actor, error) {
	userID, ok, err := h.sessions.CurrentUserID(c)
	if err != nil {
		return identity.Actor{}, err
	}
	if !ok {
		return identity.Actor{}, fiber.NewError(fiber.StatusUnauthorized, "auth.required")
	}
	return h.users.LoadActor(c.Context(), userID)
}

func (h *Controller) optionalActor(c fiber.Ctx) (identity.Actor, bool, error) {
	userID, ok, err := h.sessions.CurrentUserID(c)
	if err != nil || !ok {
		return identity.Actor{}, false, err
	}
	actor, err := h.users.LoadActor(c.Context(), userID)
	if err != nil {
		return identity.Actor{}, false, err
	}
	return actor, true, nil
}

func mapExtensionError(err error) error {
	switch {
	case errors.Is(err, identity.ErrPermissionDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	case errors.Is(err, extensions.ErrInvalidArchive):
		return fiber.NewError(fiber.StatusUnprocessableEntity, extensions.CodeInvalidArchive)
	case errors.Is(err, extensions.ErrInvalidManifest):
		return fiber.NewError(fiber.StatusUnprocessableEntity, extensions.CodeInvalidManifest)
	case errors.Is(err, extensions.ErrExtensionNotFound):
		return fiber.NewError(fiber.StatusNotFound, extensions.CodeNotFound)
	case errors.Is(err, extensions.ErrExtensionDisabled):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeExtensionDisabled)
	case errors.Is(err, extensions.ErrWebReleaseNotFound):
		return fiber.NewError(fiber.StatusNotFound, extensions.CodeWebReleaseNotFound)
	case errors.Is(err, extensions.ErrFrontendGrantNotFound):
		return fiber.NewError(fiber.StatusNotFound, extensions.CodeFrontendTrustNotFound)
	case errors.Is(err, extensions.ErrFrontendTrustUnavailable),
		errors.Is(err, extensions.ErrFrontendGrantConflict),
		errors.Is(err, extensions.ErrFrontendGrantStateConflict),
		errors.Is(err, extensions.ErrWebReleaseRetryIneligible),
		errors.Is(err, extensions.ErrWebReleaseRollbackIneligible),
		errors.Is(err, extensions.ErrWebReleaseStale),
		errors.Is(err, extensions.ErrWebReleaseCompositionMismatch):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeWebReleaseConflict)
	case errors.Is(err, extensions.ErrWebReleasePackageChanged):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeWebReleaseConflict)
	case errors.Is(err, extensions.ErrPreflightFailed):
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodePreflightFailed)
	case errors.Is(err, extensions.ErrBuildFailed):
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeBuildFailed)
	case errors.Is(err, extensions.ErrThemeActivationRequired):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeThemeActivationRequired)
	case errors.Is(err, extensions.ErrThemeRuntimeUnavailable):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeThemeRuntimeUnavailable)
	case errors.Is(err, extensions.ErrRouteNotFound):
		return fiber.NewError(fiber.StatusNotFound, extensions.CodeRouteNotFound)
	case errors.Is(err, extensions.ErrRouteMethodNotAllowed):
		return fiber.NewError(fiber.StatusMethodNotAllowed, extensions.CodeRouteMethodNotAllowed)
	case errors.Is(err, extensions.ErrRuntimeUnavailable):
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeRuntimeUnavailable)
	case errors.Is(err, extensions.ErrRuntimeFailed):
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeRuntimeFailed)
	case errors.Is(err, extensions.ErrCapabilityConfirmationRequired):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeCapabilityConfirmationRequired)
	case errors.Is(err, extensions.ErrCapabilityDenied):
		return fiber.NewError(fiber.StatusForbidden, extensions.CodeCapabilityDenied)
	case errors.Is(err, extensions.ErrNotDeletable):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeNotDeletable)
	case errors.Is(err, extensions.ErrMustDisableFirst):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeMustDisableFirst)
	case errors.Is(err, extensions.ErrMigrationFailed):
		return fiber.NewError(fiber.StatusUnprocessableEntity, extensions.CodeMigrationFailed)
	default:
		return err
	}
}

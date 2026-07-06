package extensionscontroller

import (
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
	service  *extensions.Service
	users    identity.ActorStore
	sessions *authsession.Manager
	gateway  RouteGateway
}

type ProxyInput struct {
	Matched  extensions.MatchedRoute
	Actor    identity.Actor
	HasActor bool
}

type RouteGateway interface {
	Proxy(c fiber.Ctx, input ProxyInput) error
}

func NewController(service *extensions.Service, users identity.ActorStore, sessions *authsession.Manager) *Controller {
	return NewControllerWithGateway(service, users, sessions, nil)
}

func NewControllerWithGateway(service *extensions.Service, users identity.ActorStore, sessions *authsession.Manager, gateway RouteGateway) *Controller {
	return &Controller{service: service, users: users, sessions: sessions, gateway: gateway}
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
	item, err := h.service.InstallArchive(c.Context(), actor, extensions.ArchiveInput{
		FileName: fileHeader.Filename,
		Data:     data,
	})
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.Created(c, item)
}

func (h *Controller) enable(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	item, err := h.service.Enable(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, item)
}

func (h *Controller) disable(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	item, err := h.service.Disable(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, item)
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
	item, err := h.service.ActivateTheme(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, item)
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
	default:
		return err
	}
}

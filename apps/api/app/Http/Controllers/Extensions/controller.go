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
}

func NewController(service *extensions.Service, users identity.ActorStore, sessions *authsession.Manager) *Controller {
	return &Controller{service: service, users: users, sessions: sessions}
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

func queryInt(c fiber.Ctx, name string, fallback int) int {
	value, err := strconv.Atoi(c.Query(name))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func (h *Controller) routeUnavailable(c fiber.Ctx) error {
	_, err := h.actor(c)
	if err != nil {
		return err
	}
	return fiber.NewError(fiber.StatusServiceUnavailable, "extension.route_unavailable")
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
	default:
		return err
	}
}

package extensionscontroller

import (
	"regexp"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

var frontendDigestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (h *Controller) frontendStatus(c fiber.Ctx) error {
	if h.frontend == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeFrontendRuntimeUnavailable)
	}
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	status, err := h.frontend.Frontend(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, status)
}

func (h *Controller) grantFrontendTrust(c fiber.Ctx) error {
	if h.frontend == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeFrontendRuntimeUnavailable)
	}
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var input extensions.GrantFrontendInput
	if err := c.Bind().Body(&input); err != nil || !frontendDigestPattern.MatchString(input.PackageDigest) {
		return fiber.NewError(fiber.StatusUnprocessableEntity, extensions.CodeFrontendDigestInvalid)
	}
	operation, err := h.frontend.Grant(c.Context(), actor, c.Params("id"), input)
	if err != nil {
		return mapExtensionError(err)
	}
	if operation.Queued {
		return apphttp.JSON(c, fiber.StatusAccepted, apphttp.MessageOK, operation)
	}
	return apphttp.OK(c, operation)
}

func (h *Controller) revokeFrontendTrust(c fiber.Ctx) error {
	if h.frontend == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeFrontendRuntimeUnavailable)
	}
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	operation, err := h.frontend.Revoke(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	if operation.Queued {
		return apphttp.JSON(c, fiber.StatusAccepted, apphttp.MessageOK, operation)
	}
	return apphttp.OK(c, operation)
}

func (h *Controller) restoreFrontendDefaults(c fiber.Ctx) error {
	if h.frontend == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeFrontendRuntimeUnavailable)
	}
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	operation, err := h.frontend.RestoreDefaults(c.Context(), actor)
	if err != nil {
		return mapExtensionError(err)
	}
	if operation.Queued {
		return apphttp.JSON(c, fiber.StatusAccepted, apphttp.MessageOK, operation)
	}
	return apphttp.OK(c, operation)
}

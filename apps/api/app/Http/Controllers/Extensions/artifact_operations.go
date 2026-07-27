package extensionscontroller

import (
	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	modelextensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func (h *Controller) cleanupMissingArtifacts(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var input modelextensions.MissingArtifactCleanupInput
	if err := c.Bind().Body(&input); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}
	result, err := h.service.CleanupMissingArtifacts(c.Context(), actor, input)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, result)
}

func (h *Controller) restart(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var input modelextensions.RestartInput
	if len(c.Body()) > 0 {
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
		}
	}
	input.IdempotencyKey = c.Get("Idempotency-Key")
	item, err := h.service.Restart(c.Context(), actor, c.Params("id"), input)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, item)
}

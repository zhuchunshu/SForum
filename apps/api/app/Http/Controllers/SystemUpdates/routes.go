package systemupdatescontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	api.Get("/admin/system-updates", h.status)
	api.Post("/admin/system-updates/check", h.check)
}

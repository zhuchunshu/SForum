package extensionscontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	api.Get("/admin/extensions", h.list)
	api.Post("/admin/extensions", h.install)
	api.Post("/admin/extensions/:id/enable", h.enable)
	api.Post("/admin/extensions/:id/disable", h.disable)
	api.Get("/admin/extensions/:id/events", h.events)

	api.All("/extensions/:extensionId/*", h.routeUnavailable)
}

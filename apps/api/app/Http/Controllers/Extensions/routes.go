package extensionscontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	api.Get("/admin/extensions", h.list)
	api.Get("/admin/extensions/navigation", h.navigation)
	api.Get("/admin/extensions/contribution-points", h.contributionPoints)
	api.Get("/admin/extensions/contributions", h.contributions)
	api.Post("/admin/extensions", h.install)
	api.Post("/admin/extensions/:id/enable", h.enable)
	api.Post("/admin/extensions/:id/disable", h.disable)
	api.Post("/admin/extensions/:id/verify", h.verify)
	api.Post("/admin/extensions/:id/activate", h.activate)
	api.Get("/admin/extensions/:id/events", h.events)
	api.Get("/admin/extensions/:id/settings", h.settings)
	api.Put("/admin/extensions/:id/settings", h.updateSettings)
	api.Post("/admin/extensions/:id/settings/reset", h.resetSettings)
	api.Get("/admin/extensions/event-definitions", h.eventDefinitions)
	api.Get("/admin/extensions/event-deliveries", h.eventDeliveries)

	api.All("/extensions/:extensionId/*", h.proxyExtensionRoute)
}

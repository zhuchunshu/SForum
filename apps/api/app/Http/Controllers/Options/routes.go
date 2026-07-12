package optionscontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	api.Get("/web-options", h.listPublic)
	api.Get("/web-options/:name", h.getPublic)
	api.Put("/web-options", h.update)
	api.Get("/admin/web-options", h.listAdmin)
	api.Put("/admin/web-options", h.updateAdmin)
	// F4.5：站点产品开关（与 RBAC 正交）。
	api.Get("/admin/features", h.listFeatures)
	api.Put("/admin/features", h.updateFeatures)
	api.Post("/admin/features/restore-defaults", h.restoreFeatureDefaults)
}

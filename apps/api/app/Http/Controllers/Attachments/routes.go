package attachmentscontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	api.Post("/attachments", h.upload)
	api.Get("/attachments/:publicId", h.get)
	api.Get("/attachments/:publicId/content", h.content)

	api.Get("/admin/attachment-settings", h.settings)
	api.Put("/admin/attachment-settings", h.updateSettings)
	api.Post("/admin/attachment-settings/test", h.testSettings)
	api.Get("/admin/attachments", h.listAdmin)
	api.Get("/admin/attachments/:id", h.detailAdmin)
	api.Patch("/admin/attachments/:id", h.updateAdmin)
	api.Delete("/admin/attachments/:id", h.deleteAdmin)
	api.Post("/admin/attachments/cleanup", h.cleanupAdmin)
}

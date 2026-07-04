package optionscontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	api.Get("/web-options", h.listPublic)
	api.Get("/web-options/:name", h.getPublic)
	api.Put("/web-options", h.update)
}

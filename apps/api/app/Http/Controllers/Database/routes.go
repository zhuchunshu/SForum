package databasecontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	api.Get("/admin/database/tables", h.listTables)
	api.Get("/admin/database/tables/:schema/:table", h.detail)
	api.Get("/admin/database/tables/:schema/:table/rows", h.rows)
	api.Get("/admin/database/tables/:schema/:table/rows/reveal", h.reveal)
	api.Get("/admin/database/tables/:schema/:table/export.csv", h.exportCSV)
}

package identitycontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	api.Get("/human-verification/challenge", h.humanVerificationChallenge)

	auth := api.Group("/auth")
	auth.Post("/register", h.register)
	auth.Post("/login", h.login)
	auth.Post("/logout", h.logout)
	auth.Get("/session", h.session)

	api.Get("/roles", h.listRoles)
	api.Post("/roles", h.createRole)
	api.Patch("/roles/:roleKey", h.updateRole)
	api.Delete("/roles/:roleKey", h.deleteRole)
	api.Put("/roles/:roleKey/permissions", h.replaceRolePermissions)
}

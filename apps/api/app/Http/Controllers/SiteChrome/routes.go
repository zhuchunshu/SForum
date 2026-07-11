package sitechromecontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	// 公开：前台 navbar / footer / banner。
	api.Get("/site/nav-items", h.publicNavItems)
	api.Get("/site/friend-links", h.publicFriendLinks)
	api.Get("/site/announcements", h.publicAnnouncements)

	// 管理：settings.site.manage。
	admin := api.Group("/admin/site")
	admin.Get("/nav-items", h.adminNavItems)
	admin.Post("/nav-items", h.adminCreateNavItem)
	admin.Patch("/nav-items/:itemID", h.adminUpdateNavItem)
	admin.Delete("/nav-items/:itemID", h.adminDeleteNavItem)

	admin.Get("/friend-links", h.adminFriendLinks)
	admin.Post("/friend-links", h.adminCreateFriendLink)
	admin.Patch("/friend-links/:linkID", h.adminUpdateFriendLink)
	admin.Delete("/friend-links/:linkID", h.adminDeleteFriendLink)

	admin.Get("/announcements", h.adminAnnouncements)
	admin.Post("/announcements", h.adminCreateAnnouncement)
	admin.Patch("/announcements/:announcementID", h.adminUpdateAnnouncement)
	admin.Delete("/announcements/:announcementID", h.adminDeleteAnnouncement)
}

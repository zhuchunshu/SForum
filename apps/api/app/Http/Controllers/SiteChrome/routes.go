package sitechromecontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	// 公开：前台 navbar / footer / banner。
	api.Get("/site/nav-items", h.publicNavItems)
	api.Get("/site/navigation", h.publicNavigation)
	api.Get("/site/account-navigation", h.accountSettingsNavigation)
	api.Get("/site/friend-links", h.publicFriendLinks)
	api.Get("/site/announcements", h.publicAnnouncements)

	// 管理：settings.site.manage。
	admin := api.Group("/admin/site")
	admin.Post("/brand-assets", h.adminUploadBrandAsset)
	admin.Get("/nav-items", h.adminNavItems)
	admin.Post("/nav-items", h.adminCreateNavItem)
	admin.Patch("/nav-items/:itemID", h.adminUpdateNavItem)
	admin.Delete("/nav-items/:itemID", h.adminDeleteNavItem)
	admin.Get("/navigation", h.adminNavigationDocument)
	admin.Post("/navigation/apply", h.adminApplyNavigation)
	admin.Post("/navigation/defaults/preview", h.adminPreviewNavigationDefaults)
	admin.Post("/navigation/defaults/apply", h.adminApplyNavigationPreview)
	admin.Get("/navigation/snapshots", h.adminNavigationSnapshots)
	admin.Get("/navigation/snapshots/:snapshotID", h.adminNavigationSnapshot)
	admin.Post("/navigation/snapshots/:snapshotID/restore", h.adminRestoreNavigationSnapshot)
	admin.Get("/navigation/export", h.adminExportNavigation)
	admin.Post("/navigation/import/preview", h.adminPreviewNavigationImport)
	admin.Post("/navigation/import/apply", h.adminApplyNavigationPreview)

	admin.Get("/friend-links", h.adminFriendLinks)
	admin.Post("/friend-links", h.adminCreateFriendLink)
	admin.Patch("/friend-links/:linkID", h.adminUpdateFriendLink)
	admin.Delete("/friend-links/:linkID", h.adminDeleteFriendLink)

	admin.Get("/announcements", h.adminAnnouncements)
	admin.Post("/announcements", h.adminCreateAnnouncement)
	admin.Patch("/announcements/:announcementID", h.adminUpdateAnnouncement)
	admin.Delete("/announcements/:announcementID", h.adminDeleteAnnouncement)
}

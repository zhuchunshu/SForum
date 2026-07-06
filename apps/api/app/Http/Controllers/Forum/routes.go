package forumcontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	api.Get("/category-groups", h.categoryGroups)
	api.Get("/categories", h.categories)
	api.Get("/tags", h.tags)
	api.Get("/topics", h.topics)
	api.Post("/topics", h.createTopic)
	api.Get("/topics/:topicID", h.topic)
	api.Get("/topics/:topicID/comments", h.comments)
	api.Post("/topics/:topicID/comments", h.createComment)
	api.Get("/comments/:commentID/replies", h.replies)
	api.Patch("/comments/:commentID", h.updateComment)
	api.Delete("/comments/:commentID", h.deleteComment)

	admin := api.Group("/admin/forum")
	admin.Get("/category-groups", h.adminCategoryGroups)
	admin.Post("/category-groups", h.adminCreateCategoryGroup)
	admin.Patch("/category-groups/:groupID", h.adminUpdateCategoryGroup)
	admin.Get("/categories", h.adminCategories)
	admin.Post("/categories", h.adminCreateCategory)
	admin.Patch("/categories/:categoryID", h.adminUpdateCategory)
	admin.Get("/tags", h.adminTags)
	admin.Post("/tags", h.adminCreateTag)
	admin.Patch("/tags/:tagID", h.adminUpdateTag)
	admin.Get("/settings", h.adminSettings)
	admin.Put("/settings", h.adminUpdateSettings)
	admin.Post("/settings/reset", h.adminResetSettings)
}

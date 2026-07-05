package forumcontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	api.Get("/categories", h.categories)
	api.Get("/topics", h.topics)
	api.Post("/topics", h.createTopic)
	api.Get("/topics/:topicID", h.topic)
	api.Get("/topics/:topicID/comments", h.comments)
	api.Post("/topics/:topicID/comments", h.createComment)
	api.Get("/comments/:commentID/replies", h.replies)
	api.Patch("/comments/:commentID", h.updateComment)
	api.Delete("/comments/:commentID", h.deleteComment)
}

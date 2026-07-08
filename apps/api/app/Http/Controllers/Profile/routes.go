package profilecontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	// 公开资料页：任何人都可读。
	api.Get("/profiles/:username", h.publicProfile)
	// 当前用户资料读写：登录后只能操作自己。
	api.Get("/profile", h.myProfile)
	api.Put("/profile", h.updateMyProfile)
	api.Post("/profile/avatar", h.uploadAvatar)
	api.Delete("/profile/avatar", h.deleteAvatar)
}

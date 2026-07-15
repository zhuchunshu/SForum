package profilecontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	// 公开资料包含主题摘要，读取边界跟随 forum.guest.read。
	api.Get("/profiles/:username", h.publicProfile)
	// 当前用户资料读写：登录后只能操作自己。
	api.Get("/profile", h.myProfile)
	api.Put("/profile", h.updateMyProfile)
	api.Post("/profile/avatar", h.uploadAvatar)
	api.Delete("/profile/avatar", h.deleteAvatar)
}

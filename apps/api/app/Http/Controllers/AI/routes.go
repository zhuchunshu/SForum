package aicontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	admin := api.Group("/admin/ai")
	admin.Get("/settings", h.getSettings)
	admin.Put("/settings", h.updateSettings)
	admin.Post("/settings/reset", h.resetSettings)
	admin.Get("/usage", h.getUsage)
	admin.Get("/executions", h.listExecutions)
	// 连通性诊断：发一次真实的最小调用，验证配置是否可用。
	admin.Post("/diagnose", h.diagnose)
	// 密钥录入：明文只在这一次请求里存在，写入 Secret Store 后不再回传。
	admin.Put("/profiles/:profileId/credential", h.putProfileCredential)
}

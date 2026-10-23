package aicontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	admin := api.Group("/admin/ai")
	admin.Get("/settings", h.getSettings)
	admin.Put("/settings", h.updateSettings)
	admin.Post("/settings/reset", h.resetSettings)
	admin.Get("/usage", h.getUsage)
	admin.Get("/executions", h.listExecutions)
	// 回复结果：每一条触发有没有变成评论。与执行记录并列，因为它们是两个问题。
	admin.Get("/reply-outcomes", h.listReplyOutcomes)
	// 轨迹回放：按 run 归属聚合执行记录。资源类型由控制器常量固定，见 handler。
	admin.Get("/runs/:runId", h.getRunTimeline)
	// 连通性诊断：发一次真实的最小调用，验证配置是否可用。
	admin.Post("/diagnose", h.diagnose)
	// 密钥录入：明文只在这一次请求里存在，写入 Secret Store 后不再回传。
	admin.Put("/profiles/:profileId/credential", h.putProfileCredential)
	// 模型列表：一次只读探测，不产生计费调用，也不写执行记录。
	admin.Post("/profiles/:profileId/models", h.listProfileModels)
}

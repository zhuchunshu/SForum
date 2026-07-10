package moderationcontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	// 公开举报入口（登录活跃用户）。
	api.Post("/moderation/reports", h.createReport)
	// 管理员审核队列。
	admin := api.Group("/admin/moderation")
	admin.Get("/reports", h.listReports)
	admin.Patch("/reports/:reportID", h.updateReport)
	admin.Get("/settings", h.getSettings)
	admin.Put("/settings", h.updateSettings)
	admin.Post("/settings/reset", h.resetSettings)
	admin.Get("/decisions", h.listAdminDecisions)

	workbench := api.Group("/moderation/workbench")
	workbench.Get("/counts", h.queueCounts)
	workbench.Get("/pending", h.listPending)
	workbench.Get("/reports", h.listReportItems)
	workbench.Get("/history", h.listWorkbenchHistory)
	workbench.Get("/context/:targetType/:targetID", h.getReviewContext)
	workbench.Post("/decisions", h.submitDecision)
}

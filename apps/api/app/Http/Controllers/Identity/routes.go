package identitycontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	api.Get("/human-verification/challenge", h.humanVerificationChallenge)

	auth := api.Group("/auth")
	auth.Get("/registration-status", h.registrationStatus)
	auth.Post("/register", h.register)
	auth.Post("/login", h.login)
	auth.Post("/logout", h.logout)
	auth.Get("/session", h.session)
	auth.Post("/password-reset/request", h.passwordResetRequest)
	auth.Post("/password-reset/confirm", h.passwordResetConfirm)

	// 账号安全 / 登录设备管理：自服务，仅需已登录（auth.required），
	// 越权由 store 层 user_id 过滤保证。
	sessions := auth.Group("/sessions")
	sessions.Get("", h.listSessions)
	sessions.Post("/revoke-others", h.revokeOtherSessions)
	sessions.Delete("/:sessionId", h.revokeSession)

	// 个人访问令牌（F3.4）：仅 cookie 会话可管理；Bearer 不可创建/列出。
	tokens := auth.Group("/tokens")
	tokens.Get("", h.listAPITokens)
	tokens.Post("", h.createAPIToken)
	tokens.Delete("/:tokenID", h.revokeAPIToken)
	tokens.Post("/:tokenID/rotate", h.rotateAPIToken)

	// 邮件测试：管理员验证 SMTP 配置是否生效。
	api.Post("/admin/mail/test", h.adminMailTest)

	api.Get("/permissions", h.listPermissions)
	api.Get("/permissions/matrix", h.permissionMatrix)

	api.Get("/roles", h.listRoles)
	api.Post("/roles", h.createRole)
	api.Patch("/roles/:roleKey", h.updateRole)
	api.Delete("/roles/:roleKey", h.deleteRole)
	api.Put("/roles/:roleKey/permissions", h.replaceRolePermissions)

	api.Get("/users", h.listUsers)
	api.Get("/users/:userID", h.getUser)
	api.Put("/users/:userID/roles", h.replaceUserRoles)
	api.Put("/users/:userID/permission-overrides", h.replaceUserPermissionOverrides)
	// 管理员强制下线目标用户的全部设备（user.manage）。
	api.Post("/users/:userID/sessions/revoke", h.adminRevokeUserSessions)
	// 管理员清空目标用户相关真实 IP（user.manage；隐私合规）。
	api.Post("/users/:userID/client-ips/clear", h.adminClearUserClientIPs)
}

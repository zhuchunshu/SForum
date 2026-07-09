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
}

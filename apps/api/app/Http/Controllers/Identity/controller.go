package identitycontroller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	apitokens "github.com/zhuchunshu/sforum/apps/api/app/Models/APITokens"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	clientip "github.com/zhuchunshu/sforum/apps/api/app/Support/ClientIP"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
	useragent "github.com/zhuchunshu/sforum/apps/api/app/Support/UserAgent"
)

// 确保 options.Service 满足 optionsResolver 接口。
var _ optionsResolver = (*options.Service)(nil)

type Controller struct {
	service       *identity.Service
	authSessions  *authsession.Manager
	verifier      humanverify.Verifier
	passwordReset *identity.PasswordResetService
	mailQueue     adminMailQueue
	options       optionsResolver
	// apiTokens 可选：个人访问令牌（F3.4）。
	apiTokens *apitokens.Service
	auditor   audit.Writer
	// sessionPolicy 在签发浏览器会话前做 selected session.evaluate。
	// 为 nil 时保持历史 Core 默认（测试与未接线启动路径）。
	sessionPolicy *identity.SessionPolicyEvaluator
	// riskPolicy 在密码校验成功后、会话签发前组合全部活跃 risk 提供方。
	// 为 nil 时跳过（无插件 risk 面时的 Core 默认）。
	riskPolicy *identity.RiskEvaluator
	// authFlow / profileComposer / recoveryFlow 是 Host 拥有的 Identity 提供方消费者。
	// 为 nil 时外部 auth/profile/recovery 路径保持不可用。
	authFlow        *identity.AuthProviderFlow
	profileComposer *identity.ProfileProviderComposer
	recoveryFlow    *identity.RecoveryProviderFlow
	// providerCatalog 仅用于红acted 可执行提供方列表。
	providerCatalog *identityregistry.Registry
}

// optionsResolver 只暴露密码策略、mail-test 需要的站点名/管理员邮箱，避免全量依赖 options.Service。
type optionsResolver interface {
	SiteName(ctx context.Context) (string, error)
	AdminEmail(ctx context.Context) (string, error)
	WebOption(ctx context.Context, name string) (string, error)
	PasswordPolicy(ctx context.Context) (identity.PasswordPolicy, error)
}

func NewController(service *identity.Service, sessions *session.Store) *Controller {
	return NewControllerWithVerifier(service, sessions, humanverify.NewDisabledService())
}

func NewControllerWithVerifier(service *identity.Service, sessions *session.Store, verifier humanverify.Verifier) *Controller {
	return NewControllerWithAuthSessions(service, authsession.NewManager(sessions, authsession.Config{}), verifier)
}

func NewControllerWithAuthSessions(service *identity.Service, sessions *authsession.Manager, verifier humanverify.Verifier) *Controller {
	return NewControllerWithPasswordReset(service, sessions, verifier, nil, nil, nil)
}

// NewControllerWithPasswordReset 注入密码重置与邮件服务。
type adminMailQueue interface {
	QueueMail(context.Context, notifications.QueueMailInput) (notifications.MailDelivery, error)
}

func NewControllerWithPasswordReset(service *identity.Service, sessions *authsession.Manager, verifier humanverify.Verifier, passwordReset *identity.PasswordResetService, mailQueue adminMailQueue, options optionsResolver) *Controller {
	if verifier == nil {
		verifier = humanverify.NewDisabledService()
	}
	return &Controller{
		service:       service,
		authSessions:  sessions,
		verifier:      verifier,
		passwordReset: passwordReset,
		mailQueue:     mailQueue,
		options:       options,
	}
}

// WithAPITokens 注入 PAT 服务（账号安全页创建/轮换/撤销）。
func (h *Controller) WithAPITokens(tokens *apitokens.Service) *Controller {
	if h != nil {
		h.apiTokens = tokens
	}
	return h
}

func (h *Controller) WithAuditor(w audit.Writer) *Controller {
	if h != nil {
		h.auditor = w
	}
	return h
}

// WithSessionPolicyEvaluator injects Host-owned selected session policy
// evaluation before browser session issue. Renewal is wired on AuthSession.
func (h *Controller) WithSessionPolicyEvaluator(evaluator *identity.SessionPolicyEvaluator) *Controller {
	if h != nil {
		h.sessionPolicy = evaluator
	}
	return h
}

// WithRiskEvaluator injects Host-owned risk composition before session issue.
func (h *Controller) WithRiskEvaluator(evaluator *identity.RiskEvaluator) *Controller {
	if h != nil {
		h.riskPolicy = evaluator
	}
	return h
}

// WithAuthProviderFlow injects Host-owned external auth start/complete consumers.
func (h *Controller) WithAuthProviderFlow(flow *identity.AuthProviderFlow) *Controller {
	if h != nil {
		h.authFlow = flow
	}
	return h
}

// WithProfileProviderComposer injects Host-owned profile section composition.
func (h *Controller) WithProfileProviderComposer(composer *identity.ProfileProviderComposer) *Controller {
	if h != nil {
		h.profileComposer = composer
	}
	return h
}

// WithRecoveryProviderFlow injects Host-owned recovery start/complete consumers.
func (h *Controller) WithRecoveryProviderFlow(flow *identity.RecoveryProviderFlow) *Controller {
	if h != nil {
		h.recoveryFlow = flow
	}
	return h
}

// WithIdentityProviderCatalog injects the live Identity Registry for provider listing.
func (h *Controller) WithIdentityProviderCatalog(registry *identityregistry.Registry) *Controller {
	if h != nil {
		h.providerCatalog = registry
	}
	return h
}

type registerRequest struct {
	Username          string                   `json:"username"`
	Email             string                   `json:"email"`
	Password          string                   `json:"password"`
	DisplayName       string                   `json:"displayName"`
	Locale            string                   `json:"locale"`
	HumanVerification humanVerificationRequest `json:"humanVerification"`
	// StepUpEvidence is Host-minted one-use session policy evidence.
	StepUpEvidence string `json:"stepUpEvidence"`
}

type loginRequest struct {
	Login             string                   `json:"login"`
	Password          string                   `json:"password"`
	HumanVerification humanVerificationRequest `json:"humanVerification"`
	// StepUpEvidence is Host-minted one-use session policy evidence.
	StepUpEvidence string `json:"stepUpEvidence"`
}

type humanVerificationRequest struct {
	Provider string `json:"provider"`
	Token    string `json:"token"`
}

type roleRequest struct {
	Key         string `json:"key"`
	Alias       string `json:"alias"`
	Description string `json:"description"`
}

type replaceRolePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

type replaceUserRolesRequest struct {
	RoleKeys []string `json:"roleKeys"`
}

type replaceUserPermissionOverridesRequest struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

// updateUserRequest 管理员 PATCH 用户；字段可选，未传则不改。
type updateUserRequest struct {
	Username    *string `json:"username"`
	Email       *string `json:"email"`
	DisplayName *string `json:"displayName"`
	Locale      *string `json:"locale"`
	Status      *string `json:"status"`
	Bio         *string `json:"bio"`
	Signature   *string `json:"signature"`
	Location    *string `json:"location"`
	WebsiteURL  *string `json:"websiteUrl"`
}

func (h *Controller) register(c fiber.Ctx) error {
	var req registerRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	input := identity.RegisterInput{
		Username:    req.Username,
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		Locale:      req.Locale,
	}
	if err := h.service.ValidateRegister(c.Context(), input); err != nil {
		return mapIdentityError(err)
	}

	if err := h.verifier.Verify(c.Context(), humanverify.VerifyRequest{
		Provider: req.HumanVerification.Provider,
		Purpose:  humanverify.PurposeRegister,
		Token:    req.HumanVerification.Token,
		IP:       clientip.FromCtx(c),
	}); err != nil {
		return mapHumanVerificationError(err)
	}

	current, err := h.service.Register(c.Context(), input)
	if err != nil {
		return mapIdentityError(err)
	}

	var pendingSession *authsession.Pending
	if err := h.runSessionIssue(c.Context(), current.ID, current.CurrentTokenVersion, req.StepUpEvidence, func(effectCtx context.Context) error {
		var beginErr error
		pendingSession, beginErr = h.beginSessionIssue(c, effectCtx, current.ID, current.CurrentTokenVersion)
		if beginErr != nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, identity.CodeSessionUnavailable)
		}
		h.applySessionDeviceInfo(c, current.ID, pendingSession)
		if err := h.auditLogin(effectCtx, c, current.ID, identity.AuditActionRegister, pendingSession.Info().Hash); err != nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, identity.CodeSessionUnavailable)
		}
		if err := pendingSession.SaveContext(effectCtx); err != nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, identity.CodeSessionUnavailable)
		}
		return nil
	}); err != nil {
		return mapIdentityError(err)
	}
	// 登录成功后强制最大活跃设备数，best-effort 踢出最旧设备（失败不阻塞登录）。
	// 传入本次登录的 sid，确保当前设备永不被踢。
	h.enforceMaxSessions(c, current.ID, pendingSession.Info().SID)

	return apphttp.Created(c, current)
}

func (h *Controller) registrationStatus(c fiber.Ctx) error {
	status, err := h.service.RegistrationStatus(c.Context())
	if err != nil {
		return err
	}
	return apphttp.OK(c, status)
}

func (h *Controller) login(c fiber.Ctx) error {
	var req loginRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	loginInput := identity.LoginInput{
		Login:    req.Login,
		Password: req.Password,
		ClientIP: clientip.FromCtx(c),
	}
	current, err := h.service.Login(c.Context(), loginInput)
	if errors.Is(err, identity.ErrLoginVerificationRequired) {
		if verifyErr := h.verifier.Verify(c.Context(), humanverify.VerifyRequest{
			Provider: req.HumanVerification.Provider,
			Purpose:  humanverify.PurposeLoginRisk,
			Token:    req.HumanVerification.Token,
			IP:       loginInput.ClientIP,
		}); verifyErr != nil {
			return mapHumanVerificationError(verifyErr)
		}
		loginInput.HumanVerified = true
		current, err = h.service.Login(c.Context(), loginInput)
	}
	if err != nil {
		return mapIdentityError(err)
	}

	if err := h.runRiskEvaluation(c.Context(), current.ID, "login"); err != nil {
		return mapIdentityError(err)
	}

	var pendingSession *authsession.Pending
	if err := h.runSessionIssue(c.Context(), current.ID, current.CurrentTokenVersion, req.StepUpEvidence, func(effectCtx context.Context) error {
		var beginErr error
		pendingSession, beginErr = h.beginSessionIssue(c, effectCtx, current.ID, current.CurrentTokenVersion)
		if beginErr != nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, identity.CodeSessionUnavailable)
		}
		h.applySessionDeviceInfo(c, current.ID, pendingSession)
		if err := h.auditLogin(effectCtx, c, current.ID, identity.AuditActionLogin, pendingSession.Info().Hash); err != nil {
			return err
		}
		return pendingSession.SaveContext(effectCtx)
	}); err != nil {
		return mapIdentityError(err)
	}
	// 登录成功后强制最大活跃设备数，best-effort 踢出最旧设备（失败不阻塞登录）。
	// 传入本次登录的 sid，确保当前设备永不被踢。
	h.enforceMaxSessions(c, current.ID, pendingSession.Info().SID)

	return apphttp.OK(c, current)
}

func (h *Controller) humanVerificationChallenge(c fiber.Ctx) error {
	purpose, ok := parseHumanVerificationPurpose(c.Query("purpose"))
	if !ok {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	challenge, err := h.verifier.Challenge(c.Context(), purpose, humanverify.Subject{IP: clientip.FromCtx(c)})
	if err != nil {
		return mapHumanVerificationError(err)
	}
	// ALTCHA widget 直接消费该端点，成功响应必须保持 ALTCHA 原始协议形状。
	return c.Status(fiber.StatusOK).JSON(challenge.Payload)
}

func (h *Controller) logout(c fiber.Ctx) error {
	if err := h.authSessions.Destroy(c); err != nil {
		return err
	}
	return apphttp.NoData(c)
}

func (h *Controller) session(c fiber.Ctx) error {
	userID, ok, err := h.sessionUserID(c)
	if err != nil {
		return err
	}
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
	}

	current, err := h.service.CurrentUser(c.Context(), userID)
	if err != nil {
		return mapIdentityError(err)
	}
	return apphttp.OK(c, current)
}

// passwordResetRequest 发起密码重置。响应始终为成功，不暴露邮箱是否存在。
func (h *Controller) passwordResetRequest(c fiber.Ctx) error {
	if h.passwordReset == nil {
		// 未配置密码重置服务时返回通用成功，避免暴露能力差异。
		return apphttp.OK(c, map[string]any{"sent": true})
	}
	var req passwordResetRequestPayload
	_ = c.Bind().Body(&req)
	// 可选人机验证：当 runtime options 启用 password_reset purpose 时校验。
	// 直接读取首次绑定的 req.HumanVerification，避免对同一 body 二次绑定导致嵌套 token 丢失。
	if err := h.verifier.Verify(c.Context(), humanverify.VerifyRequest{
		Provider: req.HumanVerification.Provider,
		Token:    req.HumanVerification.Token,
		Purpose:  humanverify.PurposePasswordReset,
		IP:       clientip.FromCtx(c),
	}); err != nil {
		return mapHumanVerificationError(err)
	}
	ip := clientip.FromCtx(c)
	_ = h.passwordReset.RequestPasswordReset(c.Context(), identity.RequestPasswordResetInput{
		// 规范化邮箱：trim 后传给 service，与 register/login 路径的输入处理保持一致。
		Email: strings.TrimSpace(req.Email),
		IP:    ip,
	})
	return apphttp.OK(c, map[string]any{"sent": true})
}

// passwordResetConfirm 校验令牌并更新密码。
func (h *Controller) passwordResetConfirm(c fiber.Ctx) error {
	if h.passwordReset == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "auth.password_reset_unavailable")
	}
	var req passwordResetConfirmPayload
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "auth.password_reset_invalid")
	}
	if err := h.passwordReset.ConfirmPasswordReset(c.Context(), identity.ConfirmPasswordResetInput{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	}); err != nil {
		return mapIdentityError(err)
	}
	return apphttp.OK(c, map[string]any{"reset": true})
}

// adminMailTest 向指定收件人或站点管理员邮箱发送测试邮件。
// 收件人优先级：请求 body.recipient → site.admin_email → 422。
func (h *Controller) adminMailTest(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if !actor.Can(identity.PermissionSettingsMailManage) {
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	}
	if h.mailQueue == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "mail.unavailable")
	}
	var req mailTestRequest
	_ = c.Bind().Body(&req)
	adminEmail := ""
	if h.options != nil {
		if value, err := h.options.AdminEmail(c.Context()); err == nil {
			adminEmail = value
		}
	}
	// 收件人优先级：请求 body → site.admin_email（与 SMTP From 无关）。
	recipient, validRecipient := resolveMailTestRecipient(req.Recipient, adminEmail)
	if !validRecipient {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "mail.test_recipient_required")
	}
	siteName := "SForum"
	if h.options != nil {
		if name, err := h.options.SiteName(c.Context()); err == nil && name != "" {
			siteName = name
		}
	}
	data, _ := json.Marshal(map[string]string{"subject": "[" + siteName + "] 测试邮件 / Test mail", "textBody": "这是一封来自 " + siteName + " 的测试邮件，用于验证邮件投递配置是否生效。"})
	delivery, err := h.mailQueue.QueueMail(c.Context(), notifications.QueueMailInput{Recipient: recipient, TemplateKey: "admin.test", TemplateData: data, IdempotencyKey: fmt.Sprintf("admin_test:%d:%d", actor.ID, time.Now().UnixNano())})
	if err != nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "mail.test_failed")
	}
	return apphttp.JSON(c, fiber.StatusAccepted, apphttp.MessageOK, map[string]any{"queued": true, "deliveryId": delivery.ID, "recipient": recipient})
}

func normalizeTestRecipient(value string) (string, bool) {
	value = strings.TrimSpace(value)
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value {
		return "", false
	}
	return value, true
}

// resolveMailTestRecipient 解析测试邮件收件人：显式 recipient 优先，否则 site.admin_email。
func resolveMailTestRecipient(explicit, adminEmail string) (string, bool) {
	if recipient, ok := normalizeTestRecipient(explicit); ok {
		return recipient, true
	}
	return normalizeTestRecipient(adminEmail)
}

type passwordResetRequestPayload struct {
	Email             string                   `json:"email"`
	HumanVerification humanVerificationRequest `json:"humanVerification"`
}

type passwordResetConfirmPayload struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

type mailTestRequest struct {
	Recipient string `json:"recipient"`
}

func (h *Controller) listPermissions(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	permissions, err := h.service.ListPermissions(c.Context(), actor)
	if err != nil {
		return mapIdentityError(err)
	}
	return apphttp.OK(c, permissions)
}

func (h *Controller) permissionMatrix(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	matrix, err := h.service.ListPermissionMatrix(c.Context(), actor)
	if err != nil {
		return mapIdentityError(err)
	}
	return apphttp.OK(c, matrix)
}

func (h *Controller) listRoles(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	roles, err := h.service.ListRoles(c.Context(), actor)
	if err != nil {
		return mapIdentityError(err)
	}
	return apphttp.OK(c, roles)
}

func (h *Controller) listUsers(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	users, err := h.service.ListUsers(c.Context(), actor, identity.UserListInput{
		Page:    queryInt(c, "page"),
		PerPage: queryInt(c, "perPage"),
		Query:   c.Query("query"),
		Status:  c.Query("status"),
		RoleKey: c.Query("roleKey"),
	})
	if err != nil {
		return mapIdentityError(err)
	}
	return apphttp.OK(c, users)
}

func (h *Controller) getUser(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	userID, err := paramInt64(c, "userID")
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}
	user, err := h.service.GetAdminUser(c.Context(), actor, userID)
	if err != nil {
		return mapIdentityError(err)
	}
	return apphttp.OK(c, user)
}

func (h *Controller) updateUser(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	userID, err := paramInt64(c, "userID")
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	var req updateUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	input := identity.AdminUpdateUserInput{
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Locale:      req.Locale,
		Bio:         req.Bio,
		Signature:   req.Signature,
		Location:    req.Location,
		WebsiteURL:  req.WebsiteURL,
	}
	if req.Status != nil {
		status := identity.UserStatus(strings.TrimSpace(*req.Status))
		input.Status = &status
	}

	user, err := h.service.UpdateAdminUser(c.Context(), actor, userID, input)
	if err != nil {
		return mapIdentityError(err)
	}
	return apphttp.OK(c, user)
}

func (h *Controller) createRole(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	var req roleRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	role, err := h.service.CreateRole(c.Context(), actor, identity.RoleInput{
		Key:         req.Key,
		Alias:       req.Alias,
		Description: req.Description,
	})
	if err != nil {
		return mapIdentityError(err)
	}
	return apphttp.Created(c, role)
}

func (h *Controller) updateRole(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	var req roleRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	role, err := h.service.UpdateRole(c.Context(), actor, c.Params("roleKey"), identity.RoleInput{
		Alias:       req.Alias,
		Description: req.Description,
	})
	if err != nil {
		return mapIdentityError(err)
	}
	return apphttp.OK(c, role)
}

func (h *Controller) deleteRole(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	if err := h.service.DeleteRole(c.Context(), actor, c.Params("roleKey")); err != nil {
		return mapIdentityError(err)
	}
	return apphttp.NoData(c)
}

func (h *Controller) replaceRolePermissions(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	var req replaceRolePermissionsRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	if err := h.service.ReplaceRolePermissions(c.Context(), actor, c.Params("roleKey"), req.Permissions); err != nil {
		return mapIdentityError(err)
	}
	return apphttp.NoData(c)
}

func (h *Controller) replaceUserRoles(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	userID, err := paramInt64(c, "userID")
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	var req replaceUserRolesRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	user, err := h.service.ReplaceUserRoles(c.Context(), actor, userID, req.RoleKeys)
	if err != nil {
		return mapIdentityError(err)
	}
	return apphttp.OK(c, user)
}

func (h *Controller) replaceUserPermissionOverrides(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	userID, err := paramInt64(c, "userID")
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	var req replaceUserPermissionOverridesRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	user, err := h.service.ReplaceUserPermissionOverrides(c.Context(), actor, userID, identity.PermissionOverrides{
		Allow: req.Allow,
		Deny:  req.Deny,
	})
	if err != nil {
		return mapIdentityError(err)
	}
	return apphttp.OK(c, user)
}

func (h *Controller) sessionUserID(c fiber.Ctx) (int64, bool, error) {
	// Bearer PAT 优先于 cookie session（机器调用场景）。
	if userID, ok := apitokens.UserIDFromContext(c.Context()); ok {
		return userID, true, nil
	}
	return h.authSessions.CurrentUserID(c)
}

func (h *Controller) sessionUserIDWithoutRenewal(c fiber.Ctx) (int64, bool, error) {
	if userID, ok := apitokens.UserIDFromContext(c.Context()); ok {
		return userID, true, nil
	}
	return h.authSessions.CurrentUserIDWithoutRenewal(c)
}

// runRiskEvaluation 组合活跃 risk 提供方；无接线时保持 Core 默认放行。
func (h *Controller) runRiskEvaluation(ctx context.Context, userID int64, purpose string) error {
	if h == nil || h.riskPolicy == nil {
		return nil
	}
	_, err := h.riskPolicy.RequireAllow(ctx, identity.RiskEvaluationInput{
		UserID: userID, Purpose: purpose,
	})
	return err
}

// runSessionIssue keeps the Host issue mutation inside exact policy admission.
// Nil evaluator preserves the Core default and executes the effect directly.
func (h *Controller) runSessionIssue(
	ctx context.Context,
	userID int64,
	tokenVersion int64,
	stepUpEvidence string,
	effect identity.SessionPolicyHostEffect,
) error {
	if effect == nil {
		return identity.ErrSessionPolicyEvaluationInvalid
	}
	if h == nil || h.sessionPolicy == nil {
		return effect(ctx)
	}
	result, err := h.sessionPolicy.RequireAllowAndRun(
		ctx,
		identity.SessionEvaluationInput{
			UserID: userID, TokenVersion: tokenVersion, Purpose: identity.SessionEvaluationPurposeIssue,
			StepUpEvidenceToken: stepUpEvidence,
		},
		effect,
	)
	if errors.Is(err, identity.ErrSessionPolicyEvaluationStepUp) && result.StepUpToken != "" {
		return &identity.SessionPolicyStepUpRequiredError{
			Token: result.StepUpToken, ExpiresAt: result.StepUpExpiresAt,
		}
	}
	return err
}

func (h *Controller) beginSessionIssue(
	c fiber.Ctx,
	ctx context.Context,
	userID int64,
	tokenVersion int64,
) (*authsession.Pending, error) {
	if h.sessionPolicy == nil {
		return h.authSessions.BeginWithContext(c, ctx, userID)
	}
	return h.authSessions.BeginWithAuthorityVersion(c, ctx, userID, tokenVersion)
}

func (h *Controller) actor(c fiber.Ctx) (identity.Actor, error) {
	userID, ok, err := h.sessionUserID(c)
	return h.actorForUserID(c, userID, ok, err)
}

func (h *Controller) actorWithoutSessionRenewal(c fiber.Ctx) (identity.Actor, error) {
	userID, ok, err := h.sessionUserIDWithoutRenewal(c)
	return h.actorForUserID(c, userID, ok, err)
}

func (h *Controller) actorForUserID(c fiber.Ctx, userID int64, ok bool, err error) (identity.Actor, error) {
	if err != nil {
		return identity.Actor{}, err
	}
	if !ok {
		return identity.Actor{}, fiber.NewError(fiber.StatusUnauthorized, "auth.required")
	}

	actor, err := h.service.Actor(c.Context(), userID)
	if err != nil {
		return identity.Actor{}, mapIdentityError(err)
	}
	// PAT：权限收窄到 scopes，避免令牌等价于完整 cookie 会话。
	if scopes := apitokens.ScopesFromContext(c.Context()); len(scopes) > 0 {
		actor = apitokens.RestrictActor(actor, scopes)
	}
	return actor, nil
}

func (h *Controller) auditLogin(
	ctx context.Context,
	c fiber.Ctx,
	userID int64,
	action string,
	sessionHash string,
) error {
	return h.service.RecordLoginAudit(ctx, identity.LoginAudit{
		UserID:      userID,
		Action:      action,
		IPAddress:   clientip.FromCtx(c),
		UserAgent:   c.Get(fiber.HeaderUserAgent),
		SessionHash: sessionHash,
	})
}

// applySessionDeviceInfo 解析请求 UA/IP 并设置到 pending 会话，Save 时写入会话目录。
// 真实 IP 全文进 ip_address，脱敏前缀进 ip_prefix（用户端设备列表）。
func (h *Controller) applySessionDeviceInfo(c fiber.Ctx, userID int64, pending *authsession.Pending) {
	if pending == nil {
		return
	}
	info := useragent.Parse(c.Get(fiber.HeaderUserAgent), clientip.FromCtx(c))
	pending.SetDeviceInfo(authsession.SessionRecordInput{
		UserID:       userID,
		DeviceName:   info.DeviceName,
		Browser:      info.Browser,
		OS:           info.OS,
		UserAgentRaw: info.UserAgentRaw,
		IPAddress:    info.IPAddress,
		IPPrefix:     info.IPPrefix,
	})
}

// enforceMaxSessions 读取 max_devices 配置并在登录后踢出超额设备。best-effort。
// currentSID 是本次登录的会话标识，一定不会被踢，保证刚登录的设备立即可用。
func (h *Controller) enforceMaxSessions(c fiber.Ctx, userID int64, currentSID string) {
	maxDevices := identity.RecommendedMaxDevices
	if h.options != nil {
		if raw, err := h.options.WebOption(c.Context(), identity.NameSessionsMaxDevices); err == nil && raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil {
				maxDevices = parsed
			}
		}
	}
	_, _ = h.service.EnforceMaxSessions(c.Context(), userID, currentSID, maxDevices)
}

// listSessions 返回当前用户的活跃设备列表（含 isCurrent 标记）。
// 自服务：actor 只能看自己的会话；越权由 store 层 user_id 过滤保证。
func (h *Controller) listSessions(c fiber.Ctx) error {
	userID, ok, err := h.sessionUserID(c)
	if err != nil {
		return err
	}
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
	}
	currentSID, _ := h.authSessions.CurrentSID(c)
	// includeHistory=true 含已下线的历史记录；与 OpenAPI 契约一致，仅接受该参数名。
	includeHistory := c.Query("includeHistory") == "true"

	result, err := h.service.ListSessions(c.Context(), userID, currentSID, includeHistory, queryInt(c, "page"), queryInt(c, "perPage"))
	if err != nil {
		return mapIdentityError(err)
	}
	return apphttp.OK(c, result)
}

// revokeSession 下线当前用户的单个设备（由 path 传入 sid）。
// 越权保护：传别人的 sid 会因 store 层 user_id 不匹配返回 ErrSessionNotFound → 404，
// 不泄漏该 sid 是否属于他人。
func (h *Controller) revokeSession(c fiber.Ctx) error {
	userID, ok, err := h.sessionUserIDWithoutRenewal(c)
	if err != nil {
		return err
	}
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
	}
	sid := strings.TrimSpace(c.Params("sessionId"))
	if sid == "" {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}
	if err := h.service.RevokeSession(c.Context(), userID, sid); err != nil {
		return mapIdentityError(err)
	}
	return apphttp.NoData(c)
}

// revokeOtherSessions 下线除当前设备外的所有其他设备。
func (h *Controller) revokeOtherSessions(c fiber.Ctx) error {
	userID, ok, err := h.sessionUserIDWithoutRenewal(c)
	if err != nil {
		return err
	}
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
	}
	currentSID, _ := h.authSessions.CurrentSID(c)
	count, err := h.service.RevokeOtherSessions(c.Context(), userID, currentSID)
	if err != nil {
		return mapIdentityError(err)
	}
	return apphttp.OK(c, map[string]any{"revoked": count})
}

// adminRevokeUserSessions 管理员强制下线目标用户的全部设备。
// 权限：user.manage；禁止对自己操作（下线自己请用 logout）。
// 鉴权在 service 层为权威，此处 actor 解析失败返回 401/403。
func (h *Controller) adminRevokeUserSessions(c fiber.Ctx) error {
	actor, err := h.actorWithoutSessionRenewal(c)
	if err != nil {
		return err
	}
	targetUserID, err := paramInt64(c, "userID")
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}
	count, err := h.service.AdminRevokeUserSessions(c.Context(), actor, targetUserID)
	if err != nil {
		return mapIdentityError(err)
	}
	return apphttp.OK(c, map[string]any{"revoked": count})
}

// adminClearUserClientIPs 管理员清空目标用户相关真实客户端 IP（隐私合规）。
// 权限：user.manage。删号/封禁产品化后应在状态变更路径内自动调用。
func (h *Controller) adminClearUserClientIPs(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	targetUserID, err := paramInt64(c, "userID")
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}
	result, err := h.service.ClearUserClientIPs(c.Context(), actor, targetUserID)
	if err != nil {
		return mapIdentityError(err)
	}
	return apphttp.OK(c, result)
}

func queryInt(c fiber.Ctx, name string) int {
	value, err := strconv.Atoi(c.Query(name))
	if err != nil {
		return 0
	}
	return value
}

func paramInt64(c fiber.Ctx, name string) (int64, error) {
	return strconv.ParseInt(c.Params(name), 10, 64)
}

func mapIdentityError(err error) error {
	var registerErr *identity.RegisterInvalidError
	var rejected *appevents.RejectedError
	switch {
	case errors.As(err, &registerErr):
		return apphttp.NewErrorWithFields(fiber.StatusUnprocessableEntity, identity.CodeRegisterInvalid, registerErr.Fields)
	// 插件 user.before_register 等同步拒绝：422 + 稳定 reason，不暴露堆栈。
	case errors.As(err, &rejected):
		return fiber.NewError(fiber.StatusUnprocessableEntity, rejected.Reason)
	case errors.Is(err, identity.ErrInvalidCredentials):
		return fiber.NewError(fiber.StatusUnauthorized, "auth.invalid_credentials")
	case errors.Is(err, identity.ErrLoginLocked):
		return fiber.NewError(fiber.StatusTooManyRequests, identity.CodeLoginLocked)
	case errors.Is(err, identity.ErrLoginVerificationRequired):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "human_verification.required")
	case errors.Is(err, identity.ErrSessionPolicyEvaluationStepUp):
		// Session policy step-up reuses the login-risk human verification surface
		// and returns Host-minted one-use evidence when available.
		var stepUp *identity.SessionPolicyStepUpRequiredError
		if errors.As(err, &stepUp) && stepUp.Token != "" {
			return apphttp.NewErrorWithFields(
				fiber.StatusUnprocessableEntity,
				"human_verification.required",
				map[string][]string{"stepUpEvidence": {stepUp.Token}},
			)
		}
		return fiber.NewError(fiber.StatusUnprocessableEntity, "human_verification.required")
	case errors.Is(err, identity.ErrSessionPolicyStepUpInvalid),
		errors.Is(err, identity.ErrSessionPolicyStepUpExpired),
		errors.Is(err, identity.ErrSessionPolicyStepUpReplayed),
		errors.Is(err, identity.ErrSessionPolicyStepUpStale):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "auth.session_policy_step_up_invalid")
	case errors.Is(err, identity.ErrSessionPolicyEvaluationDenied),
		errors.Is(err, identity.ErrRiskEvaluationDenied):
		return fiber.NewError(fiber.StatusForbidden, identity.CodeSessionPolicyDenied)
	case errors.Is(err, identity.ErrRiskEvaluationStepUp):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "human_verification.required")
	case errors.Is(err, identity.ErrRiskEvaluationUnavailable),
		errors.Is(err, identity.ErrRiskEvaluationInvalid):
		return fiber.NewError(fiber.StatusServiceUnavailable, identity.CodeSessionUnavailable)
	case errors.Is(err, identity.ErrSessionPolicyEvaluationUnavailable),
		errors.Is(err, identity.ErrSessionPolicyEvaluationStale),
		errors.Is(err, identity.ErrSessionPolicyEvaluationInvalid),
		errors.Is(err, identity.ErrIdentitySessionPolicyDeclarationStale),
		errors.Is(err, identity.ErrIdentitySessionPolicyStoreUnavailable):
		return fiber.NewError(fiber.StatusServiceUnavailable, identity.CodeSessionUnavailable)
	case errors.Is(err, identity.ErrRegistrationDisabled):
		// 关闭开放注册：403 + 稳定错误码，便于前端展示“注册已关闭”。
		return fiber.NewError(fiber.StatusForbidden, identity.CodeRegisterDisabled)
	case errors.Is(err, identity.ErrPermissionDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	case errors.Is(err, identity.ErrInvalidPermission):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "permission.invalid")
	case errors.Is(err, identity.ErrInvalidRole):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "role.invalid")
	case errors.Is(err, identity.ErrInvalidRoleInput):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "role.invalid_input")
	case errors.Is(err, identity.ErrPermissionOverrideConflict):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "permission.override_conflict")
	case errors.Is(err, identity.ErrSystemRoleLocked):
		return fiber.NewError(fiber.StatusConflict, "role.system_role_locked")
	case errors.Is(err, identity.ErrDefaultRoleLocked):
		return fiber.NewError(fiber.StatusConflict, "role.default_role_locked")
	case errors.Is(err, identity.ErrInitialSuperAdminLocked):
		return fiber.NewError(fiber.StatusConflict, "user.initial_super_admin_locked")
	case errors.Is(err, identity.ErrSuperAdminOverridesLocked):
		return fiber.NewError(fiber.StatusConflict, "user.super_admin_permissions_locked")
	case errors.Is(err, identity.ErrSelfRoleChange):
		return fiber.NewError(fiber.StatusForbidden, "user.cannot_change_self_roles")
	case errors.Is(err, identity.ErrSuperAdminGrantRestricted):
		return fiber.NewError(fiber.StatusForbidden, "user.super_admin_grant_restricted")
	case errors.Is(err, identity.ErrPasswordDoesNotMeetPolicy):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "auth.password_policy")
	case errors.Is(err, identity.ErrPasswordResetTokenNotFound):
		// 无效/过期/已消费令牌：稳定 422，避免落到 500 internal_error。
		return fiber.NewError(fiber.StatusUnprocessableEntity, "auth.password_reset_invalid")
	case errors.Is(err, identity.ErrPasswordResetRateLimited):
		return fiber.NewError(fiber.StatusTooManyRequests, "auth.password_reset_rate_limited")
	case errors.Is(err, identity.ErrSessionNotFound):
		return fiber.NewError(fiber.StatusNotFound, "auth.session_not_found")
	case errors.Is(err, identity.ErrSelfSessionRevoke):
		return fiber.NewError(fiber.StatusBadRequest, "auth.cannot_revoke_own_sessions")
	case errors.Is(err, identity.ErrSuperAdminSessionLocked):
		return fiber.NewError(fiber.StatusForbidden, "auth.super_admin_session_locked")
	case errors.Is(err, identity.ErrUserNotFound):
		return fiber.NewError(fiber.StatusNotFound, "user.not_found")
	case errors.Is(err, identity.ErrInvalidUserUpdate):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "user.invalid_update")
	case errors.Is(err, identity.ErrSelfStatusChange):
		return fiber.NewError(fiber.StatusForbidden, "user.cannot_change_own_status")
	case errors.Is(err, identity.ErrUsernameOrEmailNotUnique):
		return fiber.NewError(fiber.StatusConflict, "user.username_or_email_taken")
	default:
		return err
	}
}

func mapHumanVerificationError(err error) error {
	switch {
	case errors.Is(err, humanverify.ErrRateLimited):
		return apphttp.NewErrorWithFields(fiber.StatusTooManyRequests, humanverify.CodeRateLimited, mapHumanVerificationField(humanverify.CodeRateLimited))
	case errors.Is(err, humanverify.ErrRequired):
		return apphttp.NewErrorWithFields(fiber.StatusUnprocessableEntity, humanverify.CodeRequired, mapHumanVerificationField(humanverify.CodeRequired))
	case errors.Is(err, humanverify.ErrExpired):
		return apphttp.NewErrorWithFields(fiber.StatusUnprocessableEntity, humanverify.CodeExpired, mapHumanVerificationField(humanverify.CodeExpired))
	case errors.Is(err, humanverify.ErrReplayed):
		return apphttp.NewErrorWithFields(fiber.StatusUnprocessableEntity, humanverify.CodeReplayed, mapHumanVerificationField(humanverify.CodeReplayed))
	case errors.Is(err, humanverify.ErrInvalid):
		return apphttp.NewErrorWithFields(fiber.StatusUnprocessableEntity, humanverify.CodeInvalid, mapHumanVerificationField(humanverify.CodeInvalid))
	default:
		return err
	}
}

func mapHumanVerificationField(message string) map[string][]string {
	return map[string][]string{
		identity.FieldHumanVerification: {message},
	}
}

func parseHumanVerificationPurpose(value string) (humanverify.Purpose, bool) {
	switch humanverify.Purpose(value) {
	case humanverify.PurposeRegister, humanverify.PurposePasswordReset, humanverify.PurposeLoginRisk, humanverify.PurposePostRisk:
		return humanverify.Purpose(value), true
	default:
		return "", false
	}
}

package identitycontroller

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
)

type Controller struct {
	service      *identity.Service
	authSessions *authsession.Manager
	verifier     humanverify.Verifier
}

func NewController(service *identity.Service, sessions *session.Store) *Controller {
	return NewControllerWithVerifier(service, sessions, humanverify.NewDisabledService())
}

func NewControllerWithVerifier(service *identity.Service, sessions *session.Store, verifier humanverify.Verifier) *Controller {
	return NewControllerWithAuthSessions(service, authsession.NewManager(sessions, authsession.Config{}), verifier)
}

func NewControllerWithAuthSessions(service *identity.Service, sessions *authsession.Manager, verifier humanverify.Verifier) *Controller {
	if verifier == nil {
		verifier = humanverify.NewDisabledService()
	}
	return &Controller{service: service, authSessions: sessions, verifier: verifier}
}

type registerRequest struct {
	Username          string                   `json:"username"`
	Email             string                   `json:"email"`
	Password          string                   `json:"password"`
	DisplayName       string                   `json:"displayName"`
	Locale            string                   `json:"locale"`
	HumanVerification humanVerificationRequest `json:"humanVerification"`
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
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
		IP:       c.IP(),
	}); err != nil {
		return mapHumanVerificationError(err)
	}

	current, err := h.service.Register(c.Context(), input)
	if err != nil {
		return mapIdentityError(err)
	}

	pendingSession, err := h.authSessions.Begin(c, current.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, identity.CodeSessionUnavailable)
	}
	if err := h.auditLogin(c, current.ID, identity.AuditActionRegister, pendingSession.Info().Hash); err != nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, identity.CodeSessionUnavailable)
	}
	if err := pendingSession.Save(); err != nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, identity.CodeSessionUnavailable)
	}

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

	current, err := h.service.Login(c.Context(), identity.LoginInput{Login: req.Login, Password: req.Password})
	if err != nil {
		return mapIdentityError(err)
	}

	pendingSession, err := h.authSessions.Begin(c, current.ID)
	if err != nil {
		return err
	}
	if err := h.auditLogin(c, current.ID, identity.AuditActionLogin, pendingSession.Info().Hash); err != nil {
		return err
	}
	if err := pendingSession.Save(); err != nil {
		return err
	}

	return apphttp.OK(c, current)
}

func (h *Controller) humanVerificationChallenge(c fiber.Ctx) error {
	purpose, ok := parseHumanVerificationPurpose(c.Query("purpose"))
	if !ok {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	challenge, err := h.verifier.Challenge(c.Context(), purpose, humanverify.Subject{IP: c.IP()})
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
	return h.authSessions.CurrentUserID(c)
}

func (h *Controller) actor(c fiber.Ctx) (identity.Actor, error) {
	userID, ok, err := h.sessionUserID(c)
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
	return actor, nil
}

func (h *Controller) auditLogin(c fiber.Ctx, userID int64, action string, sessionHash string) error {
	return h.service.RecordLoginAudit(c.Context(), identity.LoginAudit{
		UserID:      userID,
		Action:      action,
		IPAddress:   c.IP(),
		UserAgent:   c.Get(fiber.HeaderUserAgent),
		SessionHash: sessionHash,
	})
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
	switch {
	case errors.As(err, &registerErr):
		return apphttp.NewErrorWithFields(fiber.StatusUnprocessableEntity, identity.CodeRegisterInvalid, registerErr.Fields)
	case errors.Is(err, identity.ErrInvalidCredentials):
		return fiber.NewError(fiber.StatusUnauthorized, "auth.invalid_credentials")
	case errors.Is(err, identity.ErrPermissionDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	case errors.Is(err, identity.ErrInvalidPermission):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "permission.invalid")
	case errors.Is(err, identity.ErrInvalidRole):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "role.invalid")
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
	case errors.Is(err, identity.ErrPasswordDoesNotMeetPolicy):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "auth.password_policy")
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

package identitycontroller

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
)

const sessionUserIDKey = "user_id"

type Controller struct {
	service  *identity.Service
	sessions *session.Store
	verifier *humanverify.Service
}

func NewController(service *identity.Service, sessions *session.Store) *Controller {
	return NewControllerWithVerifier(service, sessions, humanverify.NewDisabledService())
}

func NewControllerWithVerifier(service *identity.Service, sessions *session.Store, verifier *humanverify.Service) *Controller {
	if verifier == nil {
		verifier = humanverify.NewDisabledService()
	}
	return &Controller{service: service, sessions: sessions, verifier: verifier}
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

func (h *Controller) register(c fiber.Ctx) error {
	var req registerRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	if err := h.verifier.Verify(c.Context(), humanverify.VerifyRequest{
		Provider: req.HumanVerification.Provider,
		Purpose:  humanverify.PurposeRegister,
		Token:    req.HumanVerification.Token,
		IP:       c.IP(),
	}); err != nil {
		return mapHumanVerificationError(err)
	}

	current, err := h.service.Register(c.Context(), identity.RegisterInput{
		Username:    req.Username,
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		Locale:      req.Locale,
	})
	if err != nil {
		return mapIdentityError(err)
	}

	if err := h.saveSessionUserID(c, current.ID); err != nil {
		return err
	}

	return apphttp.Created(c, current)
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

	if err := h.saveSessionUserID(c, current.ID); err != nil {
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
	return apphttp.OK(c, challenge.Payload)
}

func (h *Controller) logout(c fiber.Ctx) error {
	sess, err := h.sessions.Get(c)
	if err != nil {
		return err
	}
	if err := sess.Destroy(); err != nil {
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

func (h *Controller) saveSessionUserID(c fiber.Ctx, userID int64) error {
	sess, err := h.sessions.Get(c)
	if err != nil {
		return err
	}
	sess.Set(sessionUserIDKey, userID)
	return sess.Save()
}

func (h *Controller) sessionUserID(c fiber.Ctx) (int64, bool, error) {
	sess, err := h.sessions.Get(c)
	if err != nil {
		return 0, false, err
	}

	switch value := sess.Get(sessionUserIDKey).(type) {
	case int64:
		return value, value != 0, nil
	case int:
		return int64(value), value != 0, nil
	default:
		return 0, false, nil
	}
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

func mapIdentityError(err error) error {
	switch {
	case errors.Is(err, identity.ErrInvalidCredentials):
		return fiber.NewError(fiber.StatusUnauthorized, "auth.invalid_credentials")
	case errors.Is(err, identity.ErrPermissionDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	case errors.Is(err, identity.ErrSystemRoleLocked):
		return fiber.NewError(fiber.StatusConflict, "role.system_role_locked")
	case errors.Is(err, identity.ErrDefaultRoleLocked):
		return fiber.NewError(fiber.StatusConflict, "role.default_role_locked")
	case errors.Is(err, identity.ErrInitialSuperAdminLocked):
		return fiber.NewError(fiber.StatusConflict, "user.initial_super_admin_locked")
	case errors.Is(err, identity.ErrPasswordDoesNotMeetPolicy):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "auth.password_policy")
	default:
		return err
	}
}

func mapHumanVerificationError(err error) error {
	switch {
	case errors.Is(err, humanverify.ErrRateLimited):
		return fiber.NewError(fiber.StatusTooManyRequests, humanverify.CodeRateLimited)
	case errors.Is(err, humanverify.ErrRequired):
		return fiber.NewError(fiber.StatusUnprocessableEntity, humanverify.CodeRequired)
	case errors.Is(err, humanverify.ErrExpired):
		return fiber.NewError(fiber.StatusUnprocessableEntity, humanverify.CodeExpired)
	case errors.Is(err, humanverify.ErrReplayed):
		return fiber.NewError(fiber.StatusUnprocessableEntity, humanverify.CodeReplayed)
	case errors.Is(err, humanverify.ErrInvalid):
		return fiber.NewError(fiber.StatusUnprocessableEntity, humanverify.CodeInvalid)
	default:
		return err
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

package identity

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

const sessionUserIDKey = "user_id"

type Handler struct {
	service  *Service
	sessions *session.Store
}

func NewHandler(service *Service, sessions *session.Store) *Handler {
	return &Handler{service: service, sessions: sessions}
}

func (h *Handler) RegisterRoutes(api fiber.Router) {
	auth := api.Group("/auth")
	auth.Post("/register", h.register)
	auth.Post("/login", h.login)
	auth.Post("/logout", h.logout)
	auth.Get("/session", h.session)

	api.Get("/roles", h.listRoles)
	api.Post("/roles", h.createRole)
	api.Patch("/roles/:roleKey", h.updateRole)
	api.Delete("/roles/:roleKey", h.deleteRole)
	api.Put("/roles/:roleKey/permissions", h.replaceRolePermissions)
}

type registerRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
	Locale      string `json:"locale"`
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type roleRequest struct {
	Key         string `json:"key"`
	Alias       string `json:"alias"`
	Description string `json:"description"`
}

type replaceRolePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

func (h *Handler) register(c fiber.Ctx) error {
	var req registerRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	current, err := h.service.Register(c.Context(), RegisterInput{
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

	return c.Status(fiber.StatusCreated).JSON(current)
}

func (h *Handler) login(c fiber.Ctx) error {
	var req loginRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	current, err := h.service.Login(c.Context(), LoginInput{Login: req.Login, Password: req.Password})
	if err != nil {
		return mapIdentityError(err)
	}

	if err := h.saveSessionUserID(c, current.ID); err != nil {
		return err
	}

	return c.JSON(current)
}

func (h *Handler) logout(c fiber.Ctx) error {
	sess, err := h.sessions.Get(c)
	if err != nil {
		return err
	}
	if err := sess.Destroy(); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) session(c fiber.Ctx) error {
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
	return c.JSON(current)
}

func (h *Handler) listRoles(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	roles, err := h.service.ListRoles(c.Context(), actor)
	if err != nil {
		return mapIdentityError(err)
	}
	return c.JSON(roles)
}

func (h *Handler) createRole(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	var req roleRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	role, err := h.service.CreateRole(c.Context(), actor, RoleInput{
		Key:         req.Key,
		Alias:       req.Alias,
		Description: req.Description,
	})
	if err != nil {
		return mapIdentityError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(role)
}

func (h *Handler) updateRole(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	var req roleRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	role, err := h.service.UpdateRole(c.Context(), actor, c.Params("roleKey"), RoleInput{
		Alias:       req.Alias,
		Description: req.Description,
	})
	if err != nil {
		return mapIdentityError(err)
	}
	return c.JSON(role)
}

func (h *Handler) deleteRole(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	if err := h.service.DeleteRole(c.Context(), actor, c.Params("roleKey")); err != nil {
		return mapIdentityError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) replaceRolePermissions(c fiber.Ctx) error {
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
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) saveSessionUserID(c fiber.Ctx, userID int64) error {
	sess, err := h.sessions.Get(c)
	if err != nil {
		return err
	}
	sess.Set(sessionUserIDKey, userID)
	return sess.Save()
}

func (h *Handler) sessionUserID(c fiber.Ctx) (int64, bool, error) {
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

func (h *Handler) actor(c fiber.Ctx) (Actor, error) {
	userID, ok, err := h.sessionUserID(c)
	if err != nil {
		return Actor{}, err
	}
	if !ok {
		return Actor{}, fiber.NewError(fiber.StatusUnauthorized, "auth.required")
	}

	actor, err := h.service.Actor(c.Context(), userID)
	if err != nil {
		return Actor{}, mapIdentityError(err)
	}
	return actor, nil
}

func mapIdentityError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		return fiber.NewError(fiber.StatusUnauthorized, "auth.invalid_credentials")
	case errors.Is(err, ErrPermissionDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	case errors.Is(err, ErrSystemRoleLocked):
		return fiber.NewError(fiber.StatusConflict, "role.system_role_locked")
	case errors.Is(err, ErrDefaultRoleLocked):
		return fiber.NewError(fiber.StatusConflict, "role.default_role_locked")
	case errors.Is(err, ErrInitialSuperAdminLocked):
		return fiber.NewError(fiber.StatusConflict, "user.initial_super_admin_locked")
	case errors.Is(err, ErrPasswordDoesNotMeetPolicy):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "auth.password_policy")
	default:
		return err
	}
}

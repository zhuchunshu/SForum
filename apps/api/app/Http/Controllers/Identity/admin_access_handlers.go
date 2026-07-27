package identitycontroller

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func (h *Controller) listPermissions(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}

	permissions, err := h.service.ListPermissions(c.Context(), actor, apphttp.Locale(c))
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

	matrix, err := h.service.ListPermissionMatrix(c.Context(), actor, apphttp.Locale(c))
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

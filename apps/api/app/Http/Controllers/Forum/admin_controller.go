package forumcontroller

import (
	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
)

type categoryGroupRequest struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	Position    int    `json:"position"`
}

type updateCategoryGroupRequest struct {
	Slug        *string `json:"slug"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Visibility  *string `json:"visibility"`
	Position    *int    `json:"position"`
}

type categoryRequest struct {
	GroupID     int64  `json:"groupId"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	Position    int    `json:"position"`
	DefaultSort string `json:"defaultSort"`
}

type updateCategoryRequest struct {
	GroupID     *int64  `json:"groupId"`
	Slug        *string `json:"slug"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Visibility  *string `json:"visibility"`
	Position    *int    `json:"position"`
	DefaultSort *string `json:"defaultSort"`
}

type tagRequest struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type updateTagRequest struct {
	Slug        *string `json:"slug"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

type updateForumSettingsRequest struct {
	DefaultCategorySlug *string `json:"defaultCategorySlug"`
	TagCreationMode     *string `json:"tagCreationMode"`
	TagPublicPages      *bool   `json:"tagPublicPages"`
	TagMaxPerTopic      *int    `json:"tagMaxPerTopic"`
}

func (h *Controller) adminCategoryGroups(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if _, err := h.service.ForumSettings(c.Context(), actor); err != nil {
		return mapForumError(err)
	}
	items, err := h.service.ListCategoryGroups(c.Context())
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) adminCreateCategoryGroup(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req categoryGroupRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTopic)
	}
	item, err := h.service.CreateCategoryGroup(c.Context(), actor, forum.CreateCategoryGroupInput{
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
		Position:    req.Position,
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.Created(c, item)
}

func (h *Controller) adminUpdateCategoryGroup(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req updateCategoryGroupRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTopic)
	}
	item, err := h.service.UpdateCategoryGroup(c.Context(), actor, forum.UpdateCategoryGroupInput{
		ID:          int64(paramInt(c, "groupID")),
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
		Position:    req.Position,
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, item)
}

func (h *Controller) adminCategories(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if _, err := h.service.ForumSettings(c.Context(), actor); err != nil {
		return mapForumError(err)
	}
	items, err := h.service.ListCategories(c.Context())
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) adminCreateCategory(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req categoryRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTopic)
	}
	item, err := h.service.CreateCategory(c.Context(), actor, forum.CreateCategoryInput{
		GroupID:     req.GroupID,
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
		Position:    req.Position,
		DefaultSort: req.DefaultSort,
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.Created(c, item)
}

func (h *Controller) adminUpdateCategory(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req updateCategoryRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTopic)
	}
	item, err := h.service.UpdateCategory(c.Context(), actor, forum.UpdateCategoryInput{
		ID:          int64(paramInt(c, "categoryID")),
		GroupID:     req.GroupID,
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
		Position:    req.Position,
		DefaultSort: req.DefaultSort,
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, item)
}

func (h *Controller) adminTags(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if _, err := h.service.ForumSettings(c.Context(), actor); err != nil {
		return mapForumError(err)
	}
	items, err := h.service.ListTags(c.Context(), true)
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) adminCreateTag(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req tagRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTag)
	}
	item, err := h.service.CreateTag(c.Context(), actor, forum.CreateTagInput{
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.Created(c, item)
}

func (h *Controller) adminUpdateTag(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req updateTagRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTag)
	}
	item, err := h.service.UpdateTag(c.Context(), actor, forum.UpdateTagInput{
		ID:          int64(paramInt(c, "tagID")),
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, item)
}

func (h *Controller) adminSettings(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	settings, err := h.service.ForumSettings(c.Context(), actor)
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, settings)
}

func (h *Controller) adminUpdateSettings(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req updateForumSettingsRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidSettings)
	}
	settings, err := h.service.UpdateForumSettings(c.Context(), actor, forum.UpdateForumSettingsInput{
		DefaultCategorySlug: req.DefaultCategorySlug,
		TagCreationMode:     req.TagCreationMode,
		TagPublicPages:      req.TagPublicPages,
		TagMaxPerTopic:      req.TagMaxPerTopic,
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, settings)
}

func (h *Controller) adminResetSettings(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	settings, err := h.service.ResetForumSettings(c.Context(), actor)
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, settings)
}

package forumcontroller

import (
	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
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
	Icon        string `json:"icon"`
	IconColor   string `json:"iconColor"`
	Visibility  string `json:"visibility"`
	Position    int    `json:"position"`
	DefaultSort string `json:"defaultSort"`
}

type updateCategoryRequest struct {
	GroupID     *int64  `json:"groupId"`
	Slug        *string `json:"slug"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	IconColor   *string `json:"iconColor"`
	Visibility  *string `json:"visibility"`
	Position    *int    `json:"position"`
	DefaultSort *string `json:"defaultSort"`
}

type tagRequest struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	IconColor   string `json:"iconColor"`
	Status      string `json:"status"`
}

type updateTagRequest struct {
	Slug        *string `json:"slug"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	IconColor   *string `json:"iconColor"`
	Status      *string `json:"status"`
}

type updateForumSettingsRequest struct {
	DefaultCategorySlug *string `json:"defaultCategorySlug"`
	TagCreationMode     *string `json:"tagCreationMode"`
	TagPublicPages      *bool   `json:"tagPublicPages"`
	TagMinPerTopic      *int    `json:"tagMinPerTopic"`
	TagMaxPerTopic      *int    `json:"tagMaxPerTopic"`
	TopicsPerPage       *int    `json:"topicsPerPage"`
	CommentsPerPage     *int    `json:"commentsPerPage"`

	TopicTitleMinRunes       *int `json:"topicTitleMinRunes"`
	TopicTitleMaxRunes       *int `json:"topicTitleMaxRunes"`
	TopicContentMinRunes     *int `json:"topicContentMinRunes"`
	TopicContentMaxRunes     *int `json:"topicContentMaxRunes"`
	TopicEditWindowMinutes   *int `json:"topicEditWindowMinutes"`
	TopicCooldownSeconds      *int `json:"topicCooldownSeconds"`
	DailyTopicLimit          *int `json:"dailyTopicLimit"`
	CommentMinRunes          *int `json:"commentMinRunes"`
	CommentMaxRunes          *int `json:"commentMaxRunes"`
	CommentMaxNestingDepth   *int `json:"commentMaxNestingDepth"`
	CommentEditWindowMinutes *int `json:"commentEditWindowMinutes"`
	CommentCooldownSeconds     *int `json:"commentCooldownSeconds"`
	DailyCommentLimit        *int `json:"dailyCommentLimit"`
	ExcerptRuneLimit         *int `json:"excerptRuneLimit"`
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
		Icon:        req.Icon,
		IconColor:   req.IconColor,
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
		Icon:        req.Icon,
		IconColor:   req.IconColor,
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
		Icon:        req.Icon,
		IconColor:   req.IconColor,
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
		Icon:        req.Icon,
		IconColor:   req.IconColor,
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
		DefaultCategorySlug:      req.DefaultCategorySlug,
		TagCreationMode:          req.TagCreationMode,
		TagPublicPages:           req.TagPublicPages,
		TagMinPerTopic:           req.TagMinPerTopic,
		TagMaxPerTopic:           req.TagMaxPerTopic,
		TopicsPerPage:            req.TopicsPerPage,
		CommentsPerPage:          req.CommentsPerPage,
		TopicTitleMinRunes:       req.TopicTitleMinRunes,
		TopicTitleMaxRunes:       req.TopicTitleMaxRunes,
		TopicContentMinRunes:     req.TopicContentMinRunes,
		TopicContentMaxRunes:     req.TopicContentMaxRunes,
		TopicEditWindowMinutes:   req.TopicEditWindowMinutes,
		TopicCooldownSeconds:      req.TopicCooldownSeconds,
		DailyTopicLimit:          req.DailyTopicLimit,
		CommentMinRunes:          req.CommentMinRunes,
		CommentMaxRunes:          req.CommentMaxRunes,
		CommentMaxNestingDepth:   req.CommentMaxNestingDepth,
		CommentEditWindowMinutes: req.CommentEditWindowMinutes,
		CommentCooldownSeconds:     req.CommentCooldownSeconds,
		DailyCommentLimit:        req.DailyCommentLimit,
		ExcerptRuneLimit:         req.ExcerptRuneLimit,
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

// adminReindexSearch 触发一次搜索索引全量重建。需 search.manage 权限。
func (h *Controller) adminReindexSearch(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if !actor.Can(identity.PermissionSearchManage) {
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	}
	if h.reindexer == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "forum.search_unavailable")
	}
	run, err := h.reindexer.Reindex(c.Context(), actor.ID)
	if err != nil {
		return mapReindexError(err)
	}
	return apphttp.OK(c, run)
}

// adminReindexStatus 返回当前重建的实时进度。需 search.manage 权限。
func (h *Controller) adminReindexStatus(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if !actor.Can(identity.PermissionSearchManage) {
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	}
	if h.reindexer == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "forum.search_unavailable")
	}
	status, err := h.reindexer.ReindexStatus(c.Context())
	if err != nil {
		return mapReindexError(err)
	}
	return apphttp.OK(c, status)
}

// adminReindexRuns 返回重建历史。需 search.manage 权限。
func (h *Controller) adminReindexRuns(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if !actor.Can(identity.PermissionSearchManage) {
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	}
	if h.reindexer == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "forum.search_unavailable")
	}
	runs, err := h.reindexer.ListReindexRuns(c.Context())
	if err != nil {
		return mapReindexError(err)
	}
	return apphttp.OK(c, runs)
}

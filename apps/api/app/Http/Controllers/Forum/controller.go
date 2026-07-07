package forumcontroller

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

type Controller struct {
	service  *forum.Service
	users    identity.ActorStore
	sessions *authsession.Manager
}

func NewController(service *forum.Service, users identity.ActorStore, sessions *authsession.Manager) *Controller {
	return &Controller{service: service, users: users, sessions: sessions}
}

type createTopicRequest struct {
	CategorySlug string             `json:"categorySlug"`
	Title        string             `json:"title"`
	TagSlugs     []string           `json:"tagSlugs"`
	Content      forum.ContentInput `json:"content"`
}

// updateTopicRequest: 所有字段均可选，nil 表示不改。categorySlug/tagSlugs 为空切片表示清空标签。
type updateTopicRequest struct {
	CategorySlug *string             `json:"categorySlug"`
	Title        *string             `json:"title"`
	TagSlugs     []string            `json:"tagSlugs"`
	Content      *forum.ContentInput `json:"content"`
}

type createCommentRequest struct {
	ParentID *int64             `json:"parentId"`
	Content  forum.ContentInput `json:"content"`
}

type updateCommentRequest struct {
	Content forum.ContentInput `json:"content"`
}

func (h *Controller) categories(c fiber.Ctx) error {
	items, err := h.service.ListCategories(c.Context())
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) categoryGroups(c fiber.Ctx) error {
	items, err := h.service.ListCategoryGroups(c.Context())
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) tags(c fiber.Ctx) error {
	items, err := h.service.ListTags(c.Context(), false)
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) topics(c fiber.Ctx) error {
	list, err := h.service.ListTopics(c.Context(), forum.TopicListInput{
		Page:         queryInt(c, "page"),
		PerPage:      queryInt(c, "perPage"),
		CategorySlug: c.Query("categorySlug"),
		TagSlug:      c.Query("tagSlug"),
		Query:        c.Query("query"),
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, list)
}

func (h *Controller) createTopic(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req createTopicRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTopic)
	}
	topic, err := h.service.CreateTopic(c.Context(), actor, forum.CreateTopicInput{
		CategorySlug: req.CategorySlug,
		Title:        req.Title,
		TagSlugs:     req.TagSlugs,
		Content:      req.Content,
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.Created(c, topic)
}

func (h *Controller) topic(c fiber.Ctx) error {
	topic, err := h.service.GetTopic(c.Context(), int64(paramInt(c, "topicID")))
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, topic)
}

func (h *Controller) updateTopic(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req updateTopicRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTopic)
	}
	// 区分 tagSlugs 字段"缺失"与"显式空数组"：仅当请求体里出现了 tagSlugs 才替换标签。
	hasTagSlugs, err := bodyHasKey(c, "tagSlugs")
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTopic)
	}
	input := forum.UpdateTopicInput{
		TopicID:      int64(paramInt(c, "topicID")),
		CategorySlug: req.CategorySlug,
		Title:        req.Title,
		Content:      req.Content,
	}
	if hasTagSlugs {
		input.TagSlugs = req.TagSlugs
	}
	topic, err := h.service.UpdateTopic(c.Context(), actor, input)
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, topic)
}

// bodyHasKey 判断 JSON 请求体是否包含指定顶层字段，用于区分"未提供"与"显式 null/空"。
func bodyHasKey(c fiber.Ctx, key string) (bool, error) {
	body := c.Body()
	if len(body) == 0 {
		return false, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return false, err
	}
	_, ok := raw[key]
	return ok, nil
}

func (h *Controller) deleteTopic(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	topic, err := h.service.DeleteTopic(c.Context(), actor, int64(paramInt(c, "topicID")))
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, topic)
}

// topicAction 是统一的主题生命周期处理入口，对应 hide/restore/lock/unlock/pin/unpin。
func (h *Controller) topicAction(c fiber.Ctx, action string) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	result, err := h.service.ApplyTopicAction(c.Context(), actor, forum.TopicLifecycleInput{
		TopicID: int64(paramInt(c, "topicID")),
		Action:  action,
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, result)
}

func (h *Controller) hideTopic(c fiber.Ctx) error {
	return h.topicAction(c, forum.TopicActionHide)
}

func (h *Controller) restoreTopic(c fiber.Ctx) error {
	return h.topicAction(c, forum.TopicActionRestore)
}

func (h *Controller) lockTopic(c fiber.Ctx) error {
	return h.topicAction(c, forum.TopicActionLock)
}

func (h *Controller) unlockTopic(c fiber.Ctx) error {
	return h.topicAction(c, forum.TopicActionUnlock)
}

func (h *Controller) pinTopic(c fiber.Ctx) error {
	return h.topicAction(c, forum.TopicActionPin)
}

func (h *Controller) unpinTopic(c fiber.Ctx) error {
	return h.topicAction(c, forum.TopicActionUnpin)
}

func (h *Controller) comments(c fiber.Ctx) error {
	list, err := h.service.ListComments(c.Context(), forum.CommentListInput{
		TopicID: int64(paramInt(c, "topicID")),
		View:    c.Query("view", "tree"),
		Page:    queryInt(c, "page"),
		PerPage: queryInt(c, "perPage"),
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, list)
}

func (h *Controller) createComment(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req createCommentRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidContent)
	}
	comment, err := h.service.CreateComment(c.Context(), actor, forum.CreateCommentInput{
		TopicID:  int64(paramInt(c, "topicID")),
		ParentID: req.ParentID,
		Content:  req.Content,
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.Created(c, comment)
}

func (h *Controller) replies(c fiber.Ctx) error {
	items, err := h.service.ListCommentReplies(c.Context(), int64(paramInt(c, "commentID")))
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) updateComment(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req updateCommentRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidContent)
	}
	comment, err := h.service.UpdateComment(c.Context(), actor, forum.UpdateCommentInput{
		CommentID: int64(paramInt(c, "commentID")),
		Content:   req.Content,
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, comment)
}

func (h *Controller) deleteComment(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	comment, err := h.service.DeleteComment(c.Context(), actor, int64(paramInt(c, "commentID")))
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, comment)
}

func (h *Controller) actor(c fiber.Ctx) (identity.Actor, error) {
	userID, ok, err := h.sessions.CurrentUserID(c)
	if err != nil {
		return identity.Actor{}, err
	}
	if !ok {
		return identity.Actor{}, fiber.NewError(fiber.StatusUnauthorized, "auth.required")
	}
	return h.users.LoadActor(c.Context(), userID)
}

func mapForumError(err error) error {
	var rejected *appevents.RejectedError
	switch {
	case errors.As(err, &rejected):
		return fiber.NewError(fiber.StatusUnprocessableEntity, rejected.Reason)
	case errors.Is(err, identity.ErrPermissionDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	case errors.Is(err, forum.ErrInvalidContent):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidContent)
	case errors.Is(err, forum.ErrInvalidTopic):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTopic)
	case errors.Is(err, forum.ErrTopicNotFound):
		return fiber.NewError(fiber.StatusNotFound, forum.CodeTopicNotFound)
	case errors.Is(err, forum.ErrCommentNotFound):
		return fiber.NewError(fiber.StatusNotFound, forum.CodeCommentNotFound)
	case errors.Is(err, forum.ErrTopicClosed):
		return fiber.NewError(fiber.StatusConflict, forum.CodeTopicClosed)
	case errors.Is(err, forum.ErrInvalidTag):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTag)
	case errors.Is(err, forum.ErrTagNotFound):
		return fiber.NewError(fiber.StatusNotFound, forum.CodeTagNotFound)
	case errors.Is(err, forum.ErrInvalidSettings):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidSettings)
	case errors.Is(err, forum.ErrInvalidAction):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidAction)
	default:
		return err
	}
}

func queryInt(c fiber.Ctx, key string) int {
	value, _ := strconv.Atoi(c.Query(key))
	return value
}

func paramInt(c fiber.Ctx, key string) int {
	value, _ := strconv.Atoi(c.Params(key))
	return value
}

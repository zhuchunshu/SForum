package forumcontroller

import (
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
	Content      forum.ContentInput `json:"content"`
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

func (h *Controller) topics(c fiber.Ctx) error {
	list, err := h.service.ListTopics(c.Context(), forum.TopicListInput{
		Page:         queryInt(c, "page"),
		PerPage:      queryInt(c, "perPage"),
		CategorySlug: c.Query("categorySlug"),
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

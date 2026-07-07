package moderationcontroller

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	moderation "github.com/zhuchunshu/sforum/apps/api/app/Models/Moderation"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
)

type Controller struct {
	service  *moderation.Service
	users    identity.ActorStore
	sessions *authsession.Manager
}

func NewController(service *moderation.Service, users identity.ActorStore, sessions *authsession.Manager) *Controller {
	return &Controller{service: service, users: users, sessions: sessions}
}

type createReportRequest struct {
	TargetType string `json:"targetType"`
	TargetID   int64  `json:"targetId"`
	ReasonCode string `json:"reasonCode"`
	Body       string `json:"body"`
}

type updateReportRequest struct {
	Status     string `json:"status"`
	ReviewNote string `json:"reviewNote"`
}

// createReport 公开举报入口（登录活跃用户）。
func (h *Controller) createReport(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req createReportRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, moderation.CodeReportInvalid)
	}
	report, err := h.service.CreateReport(c.Context(), actor, moderation.CreateReportInput{
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		ReasonCode: req.ReasonCode,
		Body:       req.Body,
	})
	if err != nil {
		return mapModerationError(err)
	}
	return apphttp.Created(c, report)
}

// listReports 管理员审核队列。
func (h *Controller) listReports(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	list, err := h.service.ListReports(c.Context(), actor, moderation.ReportListInput{
		Status:     c.Query("status"),
		TargetType: c.Query("targetType"),
		ReporterID: queryInt64(c, "reporterId"),
		Page:       queryInt(c, "page"),
		PerPage:    queryInt(c, "perPage"),
	})
	if err != nil {
		return mapModerationError(err)
	}
	return apphttp.OK(c, list)
}

// updateReport 审核员更新举报状态。
func (h *Controller) updateReport(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req updateReportRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, moderation.CodeReportInvalid)
	}
	report, err := h.service.UpdateReport(c.Context(), actor, moderation.UpdateReportInput{
		ReportID:   int64(paramInt(c, "reportID")),
		Status:     req.Status,
		ReviewNote: req.ReviewNote,
	})
	if err != nil {
		return mapModerationError(err)
	}
	return apphttp.OK(c, report)
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

func mapModerationError(err error) error {
	switch {
	case errors.Is(err, identity.ErrPermissionDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	case errors.Is(err, moderation.ErrReportInvalid):
		return fiber.NewError(fiber.StatusUnprocessableEntity, moderation.CodeReportInvalid)
	case errors.Is(err, moderation.ErrReportNotFound):
		return fiber.NewError(fiber.StatusNotFound, moderation.CodeReportNotFound)
	case errors.Is(err, moderation.ErrReportDuplicate):
		return fiber.NewError(fiber.StatusConflict, moderation.CodeReportDuplicate)
	case errors.Is(err, moderation.ErrReportTargetInvalid):
		return fiber.NewError(fiber.StatusUnprocessableEntity, moderation.CodeReportTargetInvalid)
	default:
		return err
	}
}

func queryInt(c fiber.Ctx, key string) int {
	value, _ := strconv.Atoi(c.Query(key))
	return value
}

func queryInt64(c fiber.Ctx, key string) int64 {
	value, _ := strconv.ParseInt(c.Query(key), 10, 64)
	return value
}

func paramInt(c fiber.Ctx, key string) int {
	value, _ := strconv.Atoi(c.Params(key))
	return value
}

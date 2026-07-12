package notificationscontroller

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
)

type Controller struct {
	store    notifications.Store
	sessions *authsession.Manager
	users    identity.ActorStore
	creator  interface {
		Create(context.Context, notifications.CreateInput) (notifications.Notification, error)
	}
}

func NewController(store notifications.Store, sessions *authsession.Manager, users identity.ActorStore, creator interface {
	Create(context.Context, notifications.CreateInput) (notifications.Notification, error)
}) *Controller {
	return &Controller{store: store, sessions: sessions, users: users, creator: creator}
}
func (h *Controller) RegisterRoutes(api fiber.Router) {
	group := api.Group("/notifications")
	group.Get("", h.list)
	group.Get("/unread-count", h.unreadCount)
	group.Patch("/:id/read", h.markRead)
	group.Post("/read-all", h.markAllRead)
	api.Post("/admin/notifications/test", h.adminTest)
}

func (h *Controller) adminTest(c fiber.Ctx) error {
	actor, err := apphttp.LoadActor(c, h.sessions, h.users)
	if err != nil {
		return err
	}
	if !actor.Can(identity.PermissionSettingsMailManage) {
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	}
	item, err := h.creator.Create(c.Context(), notifications.CreateInput{
		RecipientUserID: actor.ID,
		Type:            notifications.TypeAdminTest,
		TargetType:      "system",
		Payload:         []byte(`{}`),
		DedupeKey:       fmt.Sprintf("admin_test:%d:%d", actor.ID, time.Now().UnixNano()),
	})
	if err != nil {
		return err
	}
	return apphttp.JSON(c, fiber.StatusCreated, apphttp.MessageOK, item)
}
func (h *Controller) userID(c fiber.Ctx) (int64, error) {
	id, ok, err := apphttp.ResolveUserID(c, h.sessions)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "auth.required")
	}
	return id, nil
}
func (h *Controller) list(c fiber.Ctx) error {
	userID, err := h.userID(c)
	if err != nil {
		return err
	}
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	beforeID, _ := strconv.ParseInt(c.Query("beforeId", "0"), 10, 64)
	page, err := h.store.List(c.Context(), notifications.ListInput{RecipientUserID: userID, Limit: limit, BeforeID: beforeID})
	if err != nil {
		return err
	}
	return apphttp.OK(c, page)
}
func (h *Controller) unreadCount(c fiber.Ctx) error {
	userID, err := h.userID(c)
	if err != nil {
		return err
	}
	count, err := h.store.UnreadCount(c.Context(), userID)
	if err != nil {
		return err
	}
	return apphttp.OK(c, map[string]int64{"count": count})
}
func (h *Controller) markRead(c fiber.Ctx) error {
	userID, err := h.userID(c)
	if err != nil {
		return err
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusNotFound, "notification.not_found")
	}
	if err := h.store.MarkRead(c.Context(), userID, id); err != nil {
		if errors.Is(err, notifications.ErrNotificationNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "notification.not_found")
		}
		return err
	}
	return apphttp.OK(c, map[string]bool{"read": true})
}
func (h *Controller) markAllRead(c fiber.Ctx) error {
	userID, err := h.userID(c)
	if err != nil {
		return err
	}
	count, err := h.store.MarkAllRead(c.Context(), userID)
	if err != nil {
		return err
	}
	return apphttp.OK(c, map[string]int64{"updated": count})
}

package providers

import (
	"github.com/gofiber/fiber/v3"
	notificationscontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Notifications"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
)

type NotificationsProvider struct {
	controller *notificationscontroller.Controller
}

func NewNotificationsProvider(store *notifications.PostgresStore, users identity.ActorStore, sessions *authsession.Manager) *NotificationsProvider {
	return &NotificationsProvider{controller: notificationscontroller.NewController(store, sessions, users, store)}
}
func (p *NotificationsProvider) RegisterRoutes(api fiber.Router) { p.controller.RegisterRoutes(api) }

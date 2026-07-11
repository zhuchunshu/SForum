package providers

import (
	"github.com/gofiber/fiber/v3"
	mailcontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Mail"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

type MailProvider struct{ controller *mailcontroller.Controller }

func NewMailProvider(extensionStore extensions.Store, deliveries notifications.Store, registry *extensionsruntime.MailProviderRegistry, users identity.ActorStore, sessions *authsession.Manager, optionService *options.Service) *MailProvider {
	return &MailProvider{controller: mailcontroller.NewController(extensionStore, deliveries, registry, users, sessions, optionService)}
}
func (p *MailProvider) RegisterRoutes(api fiber.Router) { p.controller.RegisterRoutes(api) }

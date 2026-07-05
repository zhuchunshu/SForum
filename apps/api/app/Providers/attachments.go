package providers

import (
	"github.com/gofiber/fiber/v3"

	attachmentscontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Attachments"
	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
)

type AttachmentsProvider struct {
	controller *attachmentscontroller.Controller
}

func NewAttachmentsProvider(store attachments.Store, optionsService *options.Service, users identity.ActorStore, sessions *authsession.Manager) *AttachmentsProvider {
	return &AttachmentsProvider{
		controller: attachmentscontroller.NewController(attachments.NewService(store, optionsService), users, sessions),
	}
}

func (p *AttachmentsProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

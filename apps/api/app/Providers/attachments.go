package providers

import (
	"github.com/gofiber/fiber/v3"

	attachmentscontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Attachments"
	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

type AttachmentsProvider struct {
	controller *attachmentscontroller.Controller
}

func NewAttachmentsProvider(store attachments.Store, optionsService *options.Service, users identity.ActorStore, sessions *authsession.Manager) *AttachmentsProvider {
	return NewAttachmentsProviderWithEvents(store, optionsService, users, sessions, nil)
}

func NewAttachmentsProviderWithEvents(store attachments.Store, optionsService *options.Service, users identity.ActorStore, sessions *authsession.Manager, publisher appevents.Publisher) *AttachmentsProvider {
	return &AttachmentsProvider{
		controller: attachmentscontroller.NewController(attachments.NewServiceWithEvents(store, optionsService, publisher), users, sessions),
	}
}

func (p *AttachmentsProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

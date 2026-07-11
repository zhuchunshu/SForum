package providers

import (
	"github.com/gofiber/fiber/v3"

	attachmentscontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Attachments"
	seocontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/SEO"
	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	seo "github.com/zhuchunshu/sforum/apps/api/app/Models/SEO"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

type AttachmentsProvider struct {
	controller    *attachmentscontroller.Controller
	seoController *seocontroller.Controller
}

func NewAttachmentsProvider(store attachments.Store, optionsService *options.Service, users identity.ActorStore, sessions *authsession.Manager) *AttachmentsProvider {
	return NewAttachmentsProviderWithEvents(store, optionsService, users, sessions, nil)
}

func NewAttachmentsProviderWithEvents(store attachments.Store, optionsService *options.Service, users identity.ActorStore, sessions *authsession.Manager, publisher appevents.Publisher) *AttachmentsProvider {
	attachmentService := attachments.NewServiceWithEvents(store, optionsService, publisher)
	provider := &AttachmentsProvider{controller: attachmentscontroller.NewController(attachmentService, users, sessions)}
	if referenceStore, ok := store.(seo.AssetReferenceStore); ok {
		provider.seoController = seocontroller.NewController(seo.NewAssetService(attachmentService, referenceStore), users, sessions)
	}
	return provider
}

func (p *AttachmentsProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
	if p.seoController != nil {
		p.seoController.RegisterRoutes(api)
	}
}

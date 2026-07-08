package providers

import (
	"github.com/gofiber/fiber/v3"

	profilecontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Profile"
	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	profile "github.com/zhuchunshu/sforum/apps/api/app/Models/Profile"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
)

type ProfileProvider struct {
	controller *profilecontroller.Controller
}

func NewProfileProvider(store profile.Store, users identity.ActorStore, sessions *authsession.Manager) *ProfileProvider {
	return &ProfileProvider{
		controller: profilecontroller.NewController(profile.NewService(store), users, sessions),
	}
}

func NewProfileProviderWithAvatar(store profile.Store, users identity.ActorStore, sessions *authsession.Manager, uploader *attachments.Service, optionsService *options.Service) *ProfileProvider {
	return &ProfileProvider{
		controller: profilecontroller.NewController(profile.NewServiceWithAvatar(store, uploader, optionsService), users, sessions),
	}
}

func (p *ProfileProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

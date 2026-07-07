package providers

import (
	"github.com/gofiber/fiber/v3"

	profilecontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Profile"
	profile "github.com/zhuchunshu/sforum/apps/api/app/Models/Profile"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
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

func (p *ProfileProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

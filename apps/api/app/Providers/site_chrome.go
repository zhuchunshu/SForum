package providers

import (
	"github.com/gofiber/fiber/v3"

	sitechromecontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/SiteChrome"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	sitechrome "github.com/zhuchunshu/sforum/apps/api/app/Models/SiteChrome"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
)

type SiteChromeProvider struct {
	controller *sitechromecontroller.Controller
}

func NewSiteChromeProvider(store sitechrome.Store, users identity.ActorStore, sessions *authsession.Manager) *SiteChromeProvider {
	return &SiteChromeProvider{
		controller: sitechromecontroller.NewController(sitechrome.NewService(store), users, sessions),
	}
}

func (p *SiteChromeProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

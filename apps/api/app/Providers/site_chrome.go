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
	return NewSiteChromeProviderWithExtensionNav(store, users, sessions, nil)
}

// NewSiteChromeProviderWithExtensionNav 注入 forum.nav.items 解析（E2.3）。
func NewSiteChromeProviderWithExtensionNav(store sitechrome.Store, users identity.ActorStore, sessions *authsession.Manager, nav sitechrome.ExtensionNavItemProvider) *SiteChromeProvider {
	service := sitechrome.NewService(store)
	if nav != nil {
		service.WithExtensionNavItems(nav)
	}
	return &SiteChromeProvider{
		controller: sitechromecontroller.NewController(service, users, sessions),
	}
}

func (p *SiteChromeProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

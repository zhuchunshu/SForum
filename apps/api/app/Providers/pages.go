package providers

import (
	"github.com/gofiber/fiber/v3"

	pagescontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Pages"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
)

// PagesProvider 注册 Page Registry HTTP 路由。
type PagesProvider struct {
	controller *pagescontroller.Controller
}

func NewPagesProvider(registry *pages.Registry, users identity.ActorStore, sessions *authsession.Manager) *PagesProvider {
	return &PagesProvider{
		controller: pagescontroller.NewController(registry, users, sessions),
	}
}

func NewPagesProviderWithThemes(registry *pages.Registry, users identity.ActorStore, sessions *authsession.Manager, themes extensions.Store) *PagesProvider {
	return &PagesProvider{
		controller: pagescontroller.NewControllerWithThemes(registry, users, sessions, themes),
	}
}

// WithAuditor 注入页面批准/恢复审计。
func (p *PagesProvider) WithAuditor(w audit.Writer) *PagesProvider {
	if p != nil && p.controller != nil {
		p.controller.WithAuditor(w)
	}
	return p
}

func (p *PagesProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

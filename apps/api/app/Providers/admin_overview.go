package providers

import (
	"github.com/gofiber/fiber/v3"

	adminoverviewcontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/AdminOverview"
	adminoverview "github.com/zhuchunshu/sforum/apps/api/app/Models/AdminOverview"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
)

type AdminOverviewProvider struct {
	controller *adminoverviewcontroller.Controller
}

func NewAdminOverviewProvider(store adminoverview.Store, runtime adminoverview.RuntimeProvider, users identity.ActorStore, sessions *authsession.Manager) *AdminOverviewProvider {
	return &AdminOverviewProvider{
		controller: adminoverviewcontroller.NewController(adminoverview.NewService(store, runtime), users, sessions),
	}
}

func (p *AdminOverviewProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

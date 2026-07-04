package providers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	optionscontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Options"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
)

type OptionsProvider struct {
	controller *optionscontroller.Controller
}

func NewOptionsProvider(store options.Store, users identity.ActorStore, sessions *session.Store) *OptionsProvider {
	return &OptionsProvider{
		controller: optionscontroller.NewController(options.NewService(store), users, sessions),
	}
}

func (p *OptionsProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

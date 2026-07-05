package providers

import (
	"github.com/gofiber/fiber/v3"

	extensionscontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Extensions"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
)

type ExtensionsProvider struct {
	controller *extensionscontroller.Controller
}

func NewExtensionsProvider(store extensions.Store, users identity.ActorStore, sessions *authsession.Manager, extensionRoot string, builtinRoot string) *ExtensionsProvider {
	return &ExtensionsProvider{
		controller: extensionscontroller.NewController(extensions.NewServiceWithBuiltins(store, extensionRoot, builtinRoot), users, sessions),
	}
}

func (p *ExtensionsProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

package providers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	extensionscontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Extensions"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

type ExtensionsProvider struct {
	controller *extensionscontroller.Controller
}

type extensionRuntime interface {
	extensions.RuntimeManager
	RouteTarget(extensionID string) (extensionsruntime.RouteTarget, bool)
}

func NewExtensionsProvider(store extensions.Store, users identity.ActorStore, sessions *authsession.Manager, extensionRoot string, builtinRoot string) *ExtensionsProvider {
	return NewExtensionsProviderWithRuntime(store, users, sessions, extensionRoot, builtinRoot, nil)
}

func NewExtensionsProviderWithRuntime(store extensions.Store, users identity.ActorStore, sessions *authsession.Manager, extensionRoot string, builtinRoot string, runtime extensionRuntime) *ExtensionsProvider {
	return NewExtensionsProviderWithRuntimeAndThemeActivation(store, users, sessions, extensionRoot, builtinRoot, runtime, nil)
}

func NewExtensionsProviderWithRuntimeAndThemeActivation(store extensions.Store, users identity.ActorStore, sessions *authsession.Manager, extensionRoot string, builtinRoot string, runtime extensionRuntime, dispatcher extensions.ThemeActivationDispatcher) *ExtensionsProvider {
	service := extensions.NewServiceWithBuiltins(store, extensionRoot, builtinRoot)
	if dispatcher != nil {
		service = extensions.NewServiceWithThemeActivation(store, extensionRoot, builtinRoot, nil, nil, dispatcher)
	}
	var gateway extensionscontroller.RouteGateway
	if runtime != nil {
		service = extensions.NewServiceWithThemeActivation(store, extensionRoot, builtinRoot, runtime, nil, dispatcher)
		gateway = extensionRouteGateway{runtime: runtime, gateway: extensionsruntime.NewRouteGateway()}
	}
	return &ExtensionsProvider{
		controller: extensionscontroller.NewControllerWithGateway(service, users, sessions, gateway),
	}
}

func (p *ExtensionsProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

type extensionRouteGateway struct {
	runtime extensionRuntime
	gateway *extensionsruntime.RouteGateway
}

func (g extensionRouteGateway) Proxy(c fiber.Ctx, input extensionscontroller.ProxyInput) error {
	target, ok := g.runtime.RouteTarget(input.Matched.Extension.ID)
	if !ok || target.BaseURL == "" {
		return extensions.ErrRuntimeUnavailable
	}
	timeout := 3 * time.Second
	if input.Matched.Route.TimeoutMS > 0 {
		timeout = time.Duration(input.Matched.Route.TimeoutMS) * time.Millisecond
	}
	targetPath := input.Matched.Path
	if query := c.Request().URI().QueryString(); len(query) > 0 {
		targetPath += "?" + string(query)
	}
	actorID := ""
	if input.HasActor {
		actorID = strconv.FormatInt(input.Actor.ID, 10)
	}
	return g.gateway.Proxy(&extensionsruntime.ProxyInput{
		Request:     c.Request(),
		Response:    c.Response(),
		ExtensionID: input.Matched.Extension.ID,
		ActorID:     actorID,
		Locale:      c.Get("Accept-Language"),
		TargetBase:  target.BaseURL,
		TargetPath:  targetPath,
		Timeout:     timeout,
	})
}

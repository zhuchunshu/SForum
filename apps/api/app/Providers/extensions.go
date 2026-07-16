package providers

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	extensionscontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Extensions"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	cacheregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/CacheRegistry"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

type ExtensionsProvider struct {
	controller *extensionscontroller.Controller
}

type extensionRuntime interface {
	extensions.RuntimeManager
	RouteTarget(extensionID string) (extensionsruntime.RouteTarget, bool)
	AcquireActiveRuntimeCall(context.Context, string, extensionsruntime.RuntimeCallClass) (extensionsruntime.RuntimeInstanceSnapshot, *extensionsruntime.RuntimeAdmissionLease, error)
}

func NewExtensionsProvider(store extensions.Store, users identity.ActorStore, sessions *authsession.Manager, extensionRoot string, builtinRoot string) *ExtensionsProvider {
	return NewExtensionsProviderWithRuntime(store, users, sessions, extensionRoot, builtinRoot, nil)
}

// NewExtensionsProviderWithRuntime 构造扩展 API；公开主题始终使用运行时 Page Registry。
func NewExtensionsProviderWithRuntime(store extensions.Store, users identity.ActorStore, sessions *authsession.Manager, extensionRoot string, builtinRoot string, runtime extensionRuntime, options ...extensions.ServiceOption) *ExtensionsProvider {
	service := extensions.NewServiceWithBuiltins(store, extensionRoot, builtinRoot)
	var gateway extensionscontroller.RouteGateway
	if runtime != nil {
		service = extensions.NewServiceWithOptions(store, extensionRoot, builtinRoot, runtime, options...)
		gateway = extensionRouteGateway{runtime: runtime, gateway: extensionsruntime.NewRouteGateway()}
	}
	return &ExtensionsProvider{
		controller: extensionscontroller.NewControllerWithGateway(service, users, sessions, gateway),
	}
}

func NewExtensionsProviderWithService(
	service *extensions.Service,
	users identity.ActorStore,
	sessions *authsession.Manager,
	runtime extensionRuntime,
	frontend extensionscontroller.TrustedFrontendService,
) *ExtensionsProvider {
	var gateway extensionscontroller.RouteGateway
	if runtime != nil {
		gateway = extensionRouteGateway{runtime: runtime, gateway: extensionsruntime.NewRouteGateway()}
	}
	controller := extensionscontroller.NewControllerWithGateway(service, users, sessions, gateway)
	controller.WithTrustedRuntime(frontend)
	return &ExtensionsProvider{controller: controller}
}

func (p *ExtensionsProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

func (p *ExtensionsProvider) WithRouteProviderSelection(
	api *routes.ProviderSelectionAPI,
	auditor audit.IDWriter,
) *ExtensionsProvider {
	if p != nil && p.controller != nil {
		p.controller.WithRouteProviderSelection(api, auditor)
	}
	return p
}

func (p *ExtensionsProvider) WithProviderSlotSelection(
	api *extensionsruntime.ProviderSlotSelectionAPI,
	prober extensionscontroller.ProviderSlotProber,
	auditor audit.IDWriter,
) *ExtensionsProvider {
	if p != nil && p.controller != nil {
		p.controller.WithProviderSlotSelection(api, prober, auditor)
	}
	return p
}

func (p *ExtensionsProvider) WithRouteInspector(inspector *routes.Inspector) *ExtensionsProvider {
	if p != nil && p.controller != nil {
		p.controller.WithRouteInspector(inspector)
	}
	return p
}

func (p *ExtensionsProvider) WithCacheInspector(
	registry *cacheregistry.Registry,
	inspector *hostapi.HostCacheInspector,
) *ExtensionsProvider {
	if p != nil && p.controller != nil {
		p.controller.WithCacheInspector(registry, inspector)
	}
	return p
}

func (p *ExtensionsProvider) WithRouteContractCatalog(
	catalog extensionscontroller.RouteContractCatalog,
) *ExtensionsProvider {
	if p != nil && p.controller != nil {
		p.controller.WithRouteContractCatalog(catalog)
	}
	return p
}

func (p *ExtensionsProvider) WithAdminSurfaces(
	runtime extensionscontroller.AdminSurfaceRuntime,
	auditor audit.Writer,
) *ExtensionsProvider {
	if p != nil && p.controller != nil {
		p.controller.WithAdminSurfaces(runtime, auditor)
	}
	return p
}

type extensionRouteGateway struct {
	runtime extensionRuntime
	gateway *extensionsruntime.RouteGateway
}

func (g extensionRouteGateway) Proxy(c fiber.Ctx, input extensionscontroller.ProxyInput) error {
	target, admission, err := g.runtime.AcquireActiveRuntimeCall(c.Context(), input.Matched.Extension.ID, extensionsruntime.RuntimeCallRoute)
	if err != nil {
		return errors.Join(extensions.ErrRuntimeUnavailable, err)
	}
	defer admission.Release()
	if !publicFrontendRuntimeMatches(input.PublicFrontendExact, target) {
		return extensionscontroller.ErrPublicFrontendBridgeStale
	}
	if target.Target.BaseURL == "" {
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
		Context:     admission.Context,
		Request:     c.Request(),
		Response:    c.Response(),
		ExtensionID: input.Matched.Extension.ID,
		ActorID:     actorID,
		Locale:      c.Get("Accept-Language"),
		TargetBase:  target.Target.BaseURL,
		TargetPath:  targetPath,
		Timeout:     timeout,
	})
}

func publicFrontendRuntimeMatches(
	exact *extensionscontroller.PublicFrontendBridgeIdentity,
	target extensionsruntime.RuntimeInstanceSnapshot,
) bool {
	if exact == nil {
		return true
	}
	return target.Identity.ExtensionID == exact.ExtensionID &&
		target.ExtensionVersion == exact.ExtensionVersion &&
		target.ArtifactDigest == exact.PackageDigest
}

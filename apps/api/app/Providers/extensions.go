package providers

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensionscontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Extensions"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	toolplugin "github.com/zhuchunshu/sforum/apps/api/app/Support/AI/toolplugin"
	assetregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/AssetRegistry"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	cacheregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/CacheRegistry"
	contentregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/ContentRegistry"
	editorregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/EditorRegistry"
	entityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/EntityRegistry"
	extensioncomposition "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionComposition"
	extensionprotocol "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionProtocol"
	extensionruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionRuntime"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	mediaregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/MediaRegistry"
	navigationregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/NavigationRegistry"
	pages "github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

type ExtensionsProvider struct {
	controller *extensionscontroller.Controller
}

type extensionRuntime interface {
	extensions.RuntimeManager
	RouteTarget(extensionID string) (extensionprotocol.RouteTarget, bool)
	AcquireActiveRuntimeCall(context.Context, string, extensionruntime.RuntimeCallClass) (extensionruntime.RuntimeInstanceSnapshot, *extensionruntime.RuntimeAdmissionLease, error)
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

// WithRouteIncidentReader 透传崩溃循环隔离持久状态的只读边界。
func (p *ExtensionsProvider) WithRouteIncidentReader(reader apphttp.RouteRuntimeIncidentReader) *ExtensionsProvider {
	if p != nil && p.controller != nil {
		p.controller.WithRouteIncidentReader(reader)
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

// WithComponentCompositionInspector wires P9 composition registry traces.
func (p *ExtensionsProvider) WithComponentCompositionInspector(
	inspector extensioncomposition.Inspector,
) *ExtensionsProvider {
	if p != nil && p.controller != nil {
		p.controller.WithComponentCompositionInspector(inspector)
	}
	return p
}

// WithNavigationInspector wires P9 navigation/region inspector traces.
func (p *ExtensionsProvider) WithNavigationInspector(
	inspector *navigationregistry.Inspector,
) *ExtensionsProvider {
	if p != nil && p.controller != nil {
		p.controller.WithNavigationInspector(inspector)
	}
	return p
}

// WithAssetInspector wires P9 Asset Registry redacted admin inspection.
func (p *ExtensionsProvider) WithAssetInspector(
	registry *assetregistry.Registry,
) *ExtensionsProvider {
	if p != nil && p.controller != nil {
		p.controller.WithAssetInspector(registry)
	}
	return p
}

// WithThemeRuntimeInspector wires P9 Theme Runtime redacted admin inspection.
func (p *ExtensionsProvider) WithThemeRuntimeInspector(
	registry *pages.ThemeRuntimeRegistry,
) *ExtensionsProvider {
	if p != nil && p.controller != nil {
		p.controller.WithThemeRuntimeInspector(registry)
	}
	return p
}

// WithEditorRegistry wires P10 Editor Registry for public editor catalog.
func (p *ExtensionsProvider) WithEditorRegistry(
	registry *editorregistry.Registry,
) *ExtensionsProvider {
	if p != nil && p.controller != nil {
		p.controller.WithEditorRegistry(registry)
	}
	return p
}

// WithEntityRegistry wires P10 Entity Registry for public entity catalog.
func (p *ExtensionsProvider) WithEntityRegistry(
	registry *entityregistry.Registry,
) *ExtensionsProvider {
	if p != nil && p.controller != nil {
		p.controller.WithEntityRegistry(registry)
	}
	return p
}

// WithContentRegistry wires P10 Content Registry for public content catalog.
func (p *ExtensionsProvider) WithContentRegistry(
	registry *contentregistry.Registry,
) *ExtensionsProvider {
	if p != nil && p.controller != nil {
		p.controller.WithContentRegistry(registry)
	}
	return p
}

// WithMediaRegistry wires P10 Media Registry for public media catalog.
func (p *ExtensionsProvider) WithMediaRegistry(
	registry *mediaregistry.Registry,
) *ExtensionsProvider {
	if p != nil && p.controller != nil {
		p.controller.WithMediaRegistry(registry)
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
	target, admission, err := g.runtime.AcquireActiveRuntimeCall(c.Context(), input.Matched.Extension.ID, extensionruntime.RuntimeCallRoute)
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
	return g.gateway.Proxy(&extensionprotocol.ProxyInput{
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

// aiToolRuntimeInvoker 把 Support/AI 的本地派发类型翻译成扩展运行时的传输请求。
//
// AI 包不能 import 扩展运行时（后者反向依赖命令层，会成环），因此边界上做一次
// 显式翻译。翻译是逐字段的，没有隐式转换：任何一侧新增字段都必须同时改这里，
// 否则「宿主以为发出去的」与「插件实际收到的」会悄悄分叉。
//
// 它放在这里（而不是装配代码旁边）是为了让「模型侧工具接口 → 真实子进程」这条
// 链路只有一个翻译实现：bootstrap 装配与子进程集成测试用的是同一个适配器。
type aiToolRuntimeInvoker struct {
	runtime *extensionsruntime.AIToolRuntimeInvoker
}

// NewAIToolInvoker 把当前进程的插件运行时接到 Support/AI 的插件工具派发面上。
// runtime 为 nil 时返回 nil，调用方必须显式处理「没有运行时可派发」。
func NewAIToolInvoker(runtime *extensionsruntime.Manager) toolplugin.PluginToolInvoker {
	if runtime == nil {
		return nil
	}
	return &aiToolRuntimeInvoker{runtime: extensionsruntime.NewAIToolRuntimeInvoker(runtime)}
}

func (i *aiToolRuntimeInvoker) InvokeVersionedAITool(
	ctx context.Context,
	extension extensions.Extension,
	call toolplugin.PluginToolCall,
) (toolplugin.PluginToolResult, error) {
	if i == nil || i.runtime == nil {
		return toolplugin.PluginToolResult{}, errors.New("ai tool runtime is unavailable")
	}
	response, err := i.runtime.InvokeVersionedAITool(ctx, extension, extensionsruntime.VersionedAIToolRequest{
		DeclarationID:   call.DeclarationID,
		ContractVersion: call.ContractVersion,
		Handler:         call.Handler,
		Timeout:         call.Timeout,
		Name:            call.Name,
		Arguments:       call.Arguments,
	})
	if err != nil {
		return toolplugin.PluginToolResult{}, err
	}
	return toolplugin.PluginToolResult{Content: response.Content, IsError: response.IsError}, nil
}

var _ toolplugin.PluginToolInvoker = (*aiToolRuntimeInvoker)(nil)

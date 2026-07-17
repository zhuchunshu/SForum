package extensionscontroller

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	cacheregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/CacheRegistry"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

const maxUploadedArchiveBytes = 60 * 1024 * 1024

const (
	PublicFrontendHeaderExtensionID      = "X-SForum-Public-Extension-ID"
	PublicFrontendHeaderExtensionVersion = "X-SForum-Public-Extension-Version"
	PublicFrontendHeaderPackageDigest    = "X-SForum-Public-Package-Digest"
	PublicFrontendHeaderImpactDigest     = "X-SForum-Public-Impact-Digest"
	PublicFrontendHeaderComponentID      = "X-SForum-Public-Component-ID"
	CodePublicFrontendBridgeStale        = "extension.public_frontend_bridge_stale"
)

var ErrPublicFrontendBridgeStale = errors.New("extensions: public frontend bridge identity is stale")

type Controller struct {
	service         *extensions.Service
	frontend        TrustedFrontendService
	users           identity.ActorStore
	sessions        *authsession.Manager
	gateway         RouteGateway
	routeProviders  *routes.ProviderSelectionAPI
	routeInspector  *routes.Inspector
	cacheRegistry   *cacheregistry.Registry
	cacheInspect    func(*cacheregistry.Registry, int) (hostapi.HostCacheInspectionSnapshot, error)
	routeContracts  RouteContractCatalog
	routeAuditor    audit.IDWriter
	providerSlots   *extensionsruntime.ProviderSlotSelectionAPI
	providerProber  ProviderSlotProber
	providerAuditor audit.IDWriter
	adminSurfaces   AdminSurfaceRuntime
	adminAuditor    audit.Writer
}

type ProviderSlotProber interface {
	ProbeProviderSlotCandidate(context.Context, string, string) (extensionsruntime.ProviderSlotProbeResult, error)
}

type TrustedFrontendService interface {
	Frontend(context.Context, identity.Actor, string) (extensions.FrontendStatus, error)
	Grant(context.Context, identity.Actor, string, extensions.GrantFrontendInput) (extensions.FrontendStatus, error)
	Revoke(context.Context, identity.Actor, string) (extensions.FrontendStatus, error)
}

type TrustedFrontendAssetService interface {
	Asset(context.Context, identity.Actor, string, string, string) (extensions.FrontendAsset, error)
}

type TrustedFrontendChallengeService interface {
	Challenge(context.Context, identity.Actor, string) (extensions.FrontendTrustChallenge, error)
}

type PublicFrontendRuntimeService interface {
	PublicComponent(context.Context, string, string) (extensions.PublicFrontendComponent, error)
	PublicAsset(context.Context, string, string, string, string) (extensions.FrontendAsset, error)
	PublicPackageAsset(context.Context, string, string, string) (extensions.FrontendAsset, error)
}

type ProxyInput struct {
	Matched             extensions.MatchedRoute
	Actor               identity.Actor
	HasActor            bool
	PublicFrontendExact *PublicFrontendBridgeIdentity
}

type PublicFrontendBridgeIdentity struct {
	ExtensionID      string
	ExtensionVersion string
	PackageDigest    string
	ImpactDigest     string
	ComponentID      string
}

type RouteGateway interface {
	Proxy(c fiber.Ctx, input ProxyInput) error
}

type updateSettingsRequest struct {
	Values map[string]string `json:"values"`
}

type executeSettingsActionRequest struct {
	Values  map[string]string                               `json:"values"`
	Secrets map[string]extensions.SettingsActionSecretInput `json:"secrets"`
}

func NewController(service *extensions.Service, users identity.ActorStore, sessions *authsession.Manager) *Controller {
	return NewControllerWithGateway(service, users, sessions, nil)
}

func NewControllerWithGateway(service *extensions.Service, users identity.ActorStore, sessions *authsession.Manager, gateway RouteGateway) *Controller {
	return &Controller{service: service, users: users, sessions: sessions, gateway: gateway}
}

func (h *Controller) WithTrustedRuntime(frontend TrustedFrontendService) *Controller {
	h.frontend = frontend
	return h
}

func (h *Controller) WithRouteProviderSelection(
	api *routes.ProviderSelectionAPI,
	auditor audit.IDWriter,
) *Controller {
	h.routeProviders = api
	h.routeInspector = routes.NewProviderSelectionInspector(api, nil)
	h.routeAuditor = auditor
	return h
}

func (h *Controller) WithRouteInspector(inspector *routes.Inspector) *Controller {
	h.routeInspector = inspector
	return h
}

func (h *Controller) WithCacheInspector(
	registry *cacheregistry.Registry,
	inspector *hostapi.HostCacheInspector,
) *Controller {
	h.cacheRegistry = registry
	h.cacheInspect = nil
	if inspector != nil {
		h.cacheInspect = inspector.Inspect
	}
	return h
}

func (h *Controller) WithProviderSlotSelection(
	api *extensionsruntime.ProviderSlotSelectionAPI,
	prober ProviderSlotProber,
	auditor audit.IDWriter,
) *Controller {
	h.providerSlots = api
	h.providerProber = prober
	h.providerAuditor = auditor
	return h
}

func (h *Controller) list(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if extensionID := strings.TrimSpace(c.Query("id")); extensionID != "" {
		item, err := h.service.Detail(c.Context(), actor, extensionID)
		if errors.Is(err, extensions.ErrExtensionNotFound) {
			return apphttp.OK(c, []extensions.Extension{})
		}
		if err != nil {
			return mapExtensionError(err)
		}
		return apphttp.OK(c, []extensions.Extension{item})
	}
	items, err := h.service.List(c.Context(), actor)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) navigation(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.Navigation(c.Context(), actor)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) install(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, extensions.CodeInvalidArchive)
	}
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxUploadedArchiveBytes+1))
	if err != nil {
		return err
	}
	result, err := h.service.InstallOrUpgradeArchive(c.Context(), actor, extensions.ArchiveInput{
		FileName: fileHeader.Filename,
		Data:     data,
	})
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.Created(c, result)
}

func (h *Controller) uninstall(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var input extensions.UninstallInput
	if len(c.Body()) > 0 {
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
		}
	}
	input.IdempotencyKey = c.Get("Idempotency-Key")
	result, err := h.service.UninstallWithResult(c.Context(), actor, c.Params("id"), input)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, result)
}

func (h *Controller) listMigrations(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.ListMigrations(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) applyMigrations(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.ApplyDeclaredMigrations(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) enable(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var input extensions.EnableInput
	// body 可选；空 body 表示未确认 capabilities。
	if len(c.Body()) > 0 {
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
		}
	}
	input.IdempotencyKey = c.Get("Idempotency-Key")
	item, err := h.service.Enable(c.Context(), actor, c.Params("id"), input)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, item)
}

func (h *Controller) executableTrustStatus(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	status, err := h.service.ExecutableTrustStatus(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, status)
}

func (h *Controller) issueExecutableTrustChallenge(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	challenge, err := h.service.IssueExecutableTrustChallenge(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, challenge)
}

func (h *Controller) revokeExecutableTrust(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	status, err := h.service.RevokeExecutableTrust(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, status)
}

func (h *Controller) disable(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	item, err := h.service.DisableWithInput(c.Context(), actor, c.Params("id"), extensions.LifecycleRequestInput{
		IdempotencyKey: c.Get("Idempotency-Key"),
	})
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, item)
}

func (h *Controller) upgrade(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var input extensions.UpgradeInput
	if len(c.Body()) > 0 {
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
		}
	}
	input.IdempotencyKey = c.Get("Idempotency-Key")
	item, err := h.service.Upgrade(c.Context(), actor, c.Params("id"), input)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, item)
}

func (h *Controller) rollback(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var input extensions.RollbackInput
	if err := c.Bind().Body(&input); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}
	input.IdempotencyKey = c.Get("Idempotency-Key")
	item, err := h.service.Rollback(c.Context(), actor, c.Params("id"), input)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, item)
}

func (h *Controller) verify(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	item, err := h.service.VerifyExtension(c.Context(), actor, c.Params("id"))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, item)
}

func (h *Controller) activate(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var input extensions.ThemeActivationInput
	if err := c.Bind().Body(&input); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}
	item, err := h.service.ActivateThemeFromPreview(c.Context(), actor, c.Params("id"), input)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, item)
}

func (h *Controller) events(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.Events(c.Context(), actor, c.Params("id"), queryInt(c, "limit", 50))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) eventDefinitions(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.EventDefinitions(c.Context(), actor)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) eventDeliveries(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.EventDeliveries(c.Context(), actor, extensions.EventDeliveryListInput{
		ExtensionID: c.Query("extensionId"),
		EventName:   c.Query("eventName"),
		Status:      c.Query("status"),
		Limit:       queryInt(c, "limit", 50),
	})
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) contributionPoints(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.ContributionPoints(c.Context(), actor)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) contributions(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.Contributions(c.Context(), actor)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) publicActiveThemeSettings(c fiber.Ctx) error {
	settings, err := h.service.PublicActiveThemeSettings(c.Context())
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, settings)
}

func (h *Controller) pageBootstrap(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	result, err := h.service.AdminPageBootstrap(c.Context(), actor, c.Params("id"), c.Query("path"), apphttp.Locale(c))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, result)
}

func (h *Controller) settings(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	settings, err := h.service.Settings(c.Context(), actor, c.Params("id"), apphttp.Locale(c))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, settings)
}

func (h *Controller) updateSettings(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req updateSettingsRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}
	settings, err := h.service.UpdateSettings(c.Context(), actor, c.Params("id"), extensions.UpdateSettingsInput{Values: req.Values}, apphttp.Locale(c))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, settings)
}

func (h *Controller) resetSettings(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	settings, err := h.service.ResetSettings(c.Context(), actor, c.Params("id"), apphttp.Locale(c))
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, settings)
}

func (h *Controller) executeSettingsAction(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req executeSettingsActionRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}
	result, err := h.service.ExecuteSettingsAction(c.Context(), actor, c.Params("id"), c.Params("actionId"), extensions.ExecuteSettingsActionInput{Values: req.Values, Secrets: req.Secrets})
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, result)
}

func queryInt(c fiber.Ctx, name string, fallback int) int {
	value, err := strconv.Atoi(c.Query(name))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func (h *Controller) proxyExtensionRoute(c fiber.Ctx) error {
	exact, err := h.publicFrontendBridgeIdentity(c)
	if err != nil {
		return mapExtensionError(err)
	}
	routePath := "/" + c.Params("*")
	matched, err := h.service.MatchRoute(c.Context(), c.Params("extensionId"), c.Method(), routePath)
	if err != nil {
		return mapExtensionError(err)
	}
	if exact != nil && (matched.Extension.ID != exact.ExtensionID ||
		matched.Extension.Version != exact.ExtensionVersion ||
		matched.Extension.PackageDigest != exact.PackageDigest) {
		return mapExtensionError(ErrPublicFrontendBridgeStale)
	}
	actor, hasActor, err := h.optionalActor(c)
	if err != nil {
		return err
	}
	access := matched.Route.Access
	if access == "" {
		access = extensions.RouteAccessLogin
	}
	switch access {
	case extensions.RouteAccessLogin:
		if !hasActor {
			return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
		}
	case extensions.RouteAccessPermission:
		if !hasActor {
			return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
		}
		if !actor.Can(matched.Route.Permission) {
			return fiber.NewError(fiber.StatusForbidden, "permission.denied")
		}
	}
	if h.gateway == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeRuntimeUnavailable)
	}
	if err := h.gateway.Proxy(c, ProxyInput{
		Matched: matched, Actor: actor, HasActor: hasActor, PublicFrontendExact: exact,
	}); err != nil {
		return mapExtensionError(err)
	}
	return nil
}

func (h *Controller) publicFrontendBridgeIdentity(c fiber.Ctx) (*PublicFrontendBridgeIdentity, error) {
	identity := PublicFrontendBridgeIdentity{
		ExtensionID:      strings.TrimSpace(c.Get(PublicFrontendHeaderExtensionID)),
		ExtensionVersion: strings.TrimSpace(c.Get(PublicFrontendHeaderExtensionVersion)),
		PackageDigest:    strings.TrimSpace(c.Get(PublicFrontendHeaderPackageDigest)),
		ImpactDigest:     strings.TrimSpace(c.Get(PublicFrontendHeaderImpactDigest)),
		ComponentID:      strings.TrimSpace(c.Get(PublicFrontendHeaderComponentID)),
	}
	if identity == (PublicFrontendBridgeIdentity{}) {
		return nil, nil
	}
	if identity.ExtensionID == "" || identity.ExtensionVersion == "" || identity.ComponentID == "" ||
		!frontendDigestPattern.MatchString(identity.PackageDigest) ||
		!frontendDigestPattern.MatchString(identity.ImpactDigest) ||
		identity.ExtensionID != c.Params("extensionId") {
		return nil, ErrPublicFrontendBridgeStale
	}
	runtime, ok := h.frontend.(PublicFrontendRuntimeService)
	if h.frontend == nil || !ok {
		return nil, ErrPublicFrontendBridgeStale
	}
	descriptor, err := runtime.PublicComponent(c.Context(), identity.ExtensionID, identity.ComponentID)
	if err != nil || descriptor.ExtensionID != identity.ExtensionID ||
		descriptor.ExtensionVersion != identity.ExtensionVersion ||
		descriptor.PackageDigest != identity.PackageDigest ||
		descriptor.ImpactDigest != identity.ImpactDigest ||
		descriptor.ComponentID != identity.ComponentID {
		return nil, ErrPublicFrontendBridgeStale
	}
	return &identity, nil
}

func (h *Controller) actor(c fiber.Ctx) (identity.Actor, error) {
	return apphttp.LoadActor(c, h.sessions, h.users)
}

func (h *Controller) optionalActor(c fiber.Ctx) (identity.Actor, bool, error) {
	actor, err := apphttp.OptionalActor(c, h.sessions, h.users)
	if err != nil {
		return identity.Actor{}, false, err
	}
	if actor.ID == 0 {
		return identity.Actor{}, false, nil
	}
	return actor, true, nil
}

func mapExtensionError(err error) error {
	if mapped := mapLifecycleInspectionError(err); mapped != nil {
		return mapped
	}
	switch {
	case errors.Is(err, ErrPublicFrontendBridgeStale):
		return fiber.NewError(fiber.StatusPreconditionFailed, CodePublicFrontendBridgeStale)
	case errors.Is(err, identity.ErrPermissionDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	case errors.Is(err, extensions.ErrUntrustedBackendRestricted):
		return fiber.NewError(fiber.StatusForbidden, extensions.CodeUntrustedBackendRestricted)
	case errors.Is(err, extensions.ErrInvalidArchive):
		return fiber.NewError(fiber.StatusUnprocessableEntity, extensions.CodeInvalidArchive)
	case errors.Is(err, extensions.ErrInvalidManifest):
		return fiber.NewError(fiber.StatusUnprocessableEntity, extensions.CodeInvalidManifest)
	case errors.Is(err, extensions.ErrExtensionNotFound):
		return fiber.NewError(fiber.StatusNotFound, extensions.CodeNotFound)
	case errors.Is(err, extensions.ErrExtensionDisabled):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeExtensionDisabled)
	case errors.Is(err, extensions.ErrSettingsRollbackFailed):
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeSettingsRollbackFailed)
	case errors.Is(err, extensions.ErrSettingsActionInvalid):
		return fiber.NewError(fiber.StatusUnprocessableEntity, extensions.CodeSettingsActionInvalid)
	case errors.Is(err, extensions.ErrSettingsActionUnavailable):
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeSettingsActionUnavailable)
	case errors.Is(err, extensions.ErrFrontendGrantNotFound):
		return fiber.NewError(fiber.StatusNotFound, extensions.CodeFrontendTrustNotFound)
	case errors.Is(err, extensions.ErrFrontendTrustUnavailable),
		errors.Is(err, extensions.ErrFrontendGrantConflict),
		errors.Is(err, extensions.ErrFrontendGrantStateConflict):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeFrontendTrustNotFound)
	case errors.Is(err, extensions.ErrFrontendPackageChanged):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeFrontendPackageChanged)
	case errors.Is(err, extensions.ErrPreflightFailed):
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodePreflightFailed)
	case errors.Is(err, extensions.ErrBuildFailed):
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeBuildFailed)
	case errors.Is(err, extensions.ErrThemeActivationRequired):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeThemeActivationRequired)
	case errors.Is(err, extensions.ErrThemeRuntimeUnavailable):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeThemeRuntimeUnavailable)
	case errors.Is(err, extensions.ErrThemePreviewStale):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeThemePreviewStale)
	case errors.Is(err, extensions.ErrRouteNotFound):
		return fiber.NewError(fiber.StatusNotFound, extensions.CodeRouteNotFound)
	case errors.Is(err, extensions.ErrRouteMethodNotAllowed):
		return fiber.NewError(fiber.StatusMethodNotAllowed, extensions.CodeRouteMethodNotAllowed)
	case errors.Is(err, extensions.ErrRuntimeUnavailable):
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeRuntimeUnavailable)
	case errors.Is(err, extensions.ErrRuntimeFailed):
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeRuntimeFailed)
	case errors.Is(err, extensions.ErrCapabilityConfirmationRequired):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeCapabilityConfirmationRequired)
	case errors.Is(err, extensions.ErrTrustChallengeRequired):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeTrustChallengeRequired)
	case errors.Is(err, extensions.ErrTrustChallengeInvalid):
		return fiber.NewError(fiber.StatusForbidden, extensions.CodeTrustChallengeInvalid)
	case errors.Is(err, extensions.ErrTrustChallengeExpired):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeTrustChallengeExpired)
	case errors.Is(err, extensions.ErrTrustChallengeReplayed):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeTrustChallengeReplayed)
	case errors.Is(err, extensions.ErrTrustChallengeStale):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeTrustChallengeStale)
	case errors.Is(err, extensions.ErrTrustNotRequired):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeTrustNotRequired)
	case errors.Is(err, extensions.ErrCapabilityDenied):
		return fiber.NewError(fiber.StatusForbidden, extensions.CodeCapabilityDenied)
	case errors.Is(err, extensions.ErrFeaturesRequired):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeFeaturesRequired)
	case errors.Is(err, extensions.ErrNotDeletable):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeNotDeletable)
	case errors.Is(err, extensions.ErrMustDisableFirst):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeMustDisableFirst)
	case errors.Is(err, extensions.ErrMigrationFailed):
		return fiber.NewError(fiber.StatusUnprocessableEntity, extensions.CodeMigrationFailed)
	case errors.Is(err, extensions.ErrSafeModeActive):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeSafeModeActive)
	case errors.Is(err, extensions.ErrLifecycleCoordinatorInvalid),
		errors.Is(err, extensions.ErrExtensionVersionInvalid),
		errors.Is(err, extensions.ErrStagedVersionInvalid):
		return fiber.NewError(fiber.StatusUnprocessableEntity, extensions.CodeLifecycleInvalid)
	case errors.Is(err, extensions.ErrLifecycleCoordinatorUnavailable):
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeLifecycleUnavailable)
	case errors.Is(err, extensions.ErrLifecycleCoordinatorActionFailed):
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeLifecycleActionFailed)
	case errors.Is(err, extensions.ErrLifecycleFingerprintConflict),
		errors.Is(err, extensions.ErrLifecycleOperationInProgress),
		errors.Is(err, extensions.ErrLifecycleCoordinatorRetryRequired),
		errors.Is(err, extensions.ErrLifecycleNotRecoverable),
		errors.Is(err, extensions.ErrLifecycleRevisionConflict),
		errors.Is(err, extensions.ErrLifecycleOperationClosed),
		errors.Is(err, extensions.ErrLifecycleStateTransitionDenied),
		errors.Is(err, extensions.ErrExtensionVersionConflict),
		errors.Is(err, extensions.ErrStagedVersionConflict):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeLifecycleConflict)
	case errors.Is(err, extensions.ErrLifecycleCleanupFinalization),
		errors.Is(err, extensions.ErrLifecycleCleanupNotFinalized):
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeLifecycleCleanupFailed)
	case errors.Is(err, extensions.ErrLifecycleAuthorityNotFound):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeLifecycleAuthorityGone)
	case errors.Is(err, extensions.ErrStagedVersionNotFound):
		return fiber.NewError(fiber.StatusConflict, extensions.CodeStagedVersionNotFound)
	case errors.Is(err, extensions.ErrExtensionVersionNotFound):
		return fiber.NewError(fiber.StatusNotFound, extensions.CodeVersionNotFound)
	default:
		return err
	}
}

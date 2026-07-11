package mailcontroller

import (
	"github.com/gofiber/fiber/v3"
	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

type Controller struct {
	extensions extensions.Store
	deliveries notifications.Store
	providers  *extensionsruntime.MailProviderRegistry
	users      identity.ActorStore
	sessions   *authsession.Manager
	options    *options.Service
}

func NewController(extensionStore extensions.Store, deliveries notifications.Store, providers *extensionsruntime.MailProviderRegistry, users identity.ActorStore, sessions *authsession.Manager, optionService *options.Service) *Controller {
	return &Controller{extensions: extensionStore, deliveries: deliveries, providers: providers, users: users, sessions: sessions, options: optionService}
}
func (h *Controller) RegisterRoutes(api fiber.Router) {
	group := api.Group("/admin/mail")
	group.Get("/providers", h.listProviders)
	group.Put("/provider", h.selectProvider)
	group.Post("/provider/reset", h.resetProvider)
	group.Get("/deliveries", h.listDeliveries)
	group.Get("/policy", h.getPolicy)
	group.Put("/policy", h.updatePolicy)
	group.Post("/policy/restore", h.restorePolicy)
}
func (h *Controller) actor(c fiber.Ctx) (identity.Actor, error) {
	id, ok, err := h.sessions.CurrentUserID(c)
	if err != nil {
		return identity.Actor{}, err
	}
	if !ok {
		return identity.Actor{}, fiber.NewError(fiber.StatusUnauthorized, "auth.required")
	}
	actor, err := h.users.LoadActor(c.Context(), id)
	if err != nil {
		return identity.Actor{}, err
	}
	if !actor.Can(identity.PermissionSettingsMailManage) {
		return identity.Actor{}, fiber.NewError(fiber.StatusForbidden, "permission.denied")
	}
	return actor, nil
}
func (h *Controller) authorize(c fiber.Ctx) error { _, err := h.actor(c); return err }
func (h *Controller) listProviders(c fiber.Ctx) error {
	if err := h.authorize(c); err != nil {
		return err
	}
	items, err := h.extensions.List(c.Context())
	if err != nil {
		return err
	}
	selected, ok, err := h.providers.Selected(c.Context())
	if err != nil {
		return err
	}
	providers := []map[string]any{}
	for _, item := range items {
		if item.Status != extensions.StatusEnabled {
			continue
		}
		for _, provider := range item.Manifest.Providers {
			if provider.Slot == extensionsruntime.MailProviderSlot {
				providers = append(providers, map[string]any{"extensionId": item.ID, "label": provider.Label, "healthy": item.Runtime == nil || item.Runtime.State == extensions.RuntimeRunning})
			}
		}
	}
	return apphttp.OK(c, map[string]any{"items": providers, "selected": selected, "configured": ok})
}

func (h *Controller) getPolicy(c fiber.Ctx) error {
	if _, err := h.actor(c); err != nil {
		return err
	}
	policy, err := h.options.NotificationPolicy(c.Context())
	if err != nil {
		return err
	}
	return apphttp.OK(c, policy)
}

func (h *Controller) updatePolicy(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var policy options.NotificationPolicy
	if err := c.Bind().Body(&policy); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "notification.policy_invalid")
	}
	if _, err := h.options.UpdateMany(c.Context(), actor, options.NotificationPolicyUpdateInputs(policy)); err != nil {
		return err
	}
	return apphttp.OK(c, policy)
}

func (h *Controller) restorePolicy(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if _, err := h.options.UpdateMany(c.Context(), actor, options.NotificationPolicyRecommendedInputs()); err != nil {
		return err
	}
	policy, err := h.options.NotificationPolicy(c.Context())
	if err != nil {
		return err
	}
	return apphttp.OK(c, policy)
}

type selectRequest struct {
	ExtensionID string `json:"extensionId"`
}

func (h *Controller) selectProvider(c fiber.Ctx) error {
	if err := h.authorize(c); err != nil {
		return err
	}
	var req selectRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "mail.provider_invalid")
	}
	if err := h.providers.Select(c.Context(), req.ExtensionID); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "mail.provider_unavailable")
	}
	return apphttp.OK(c, map[string]bool{"selected": true})
}
func (h *Controller) resetProvider(c fiber.Ctx) error {
	if err := h.authorize(c); err != nil {
		return err
	}
	if err := h.providers.RestoreDefault(c.Context()); err != nil {
		return err
	}
	return apphttp.OK(c, map[string]any{"configured": false, "secretsPreserved": true})
}
func (h *Controller) listDeliveries(c fiber.Ctx) error {
	if err := h.authorize(c); err != nil {
		return err
	}
	items, err := h.deliveries.ListDeliveries(c.Context(), 50)
	if err != nil {
		return err
	}
	return apphttp.OK(c, map[string]any{"items": items})
}

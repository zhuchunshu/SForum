package identitycontroller

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
)

// listAuthProviders 返回可执行 auth/recovery 提供方的红acted 列表。
// 不含 artifact 路径、Schema 摘要或 handler 内部细节。
func (h *Controller) listAuthProviders(c fiber.Ctx) error {
	if h.providerCatalog == nil {
		return apphttp.OK(c, []authProviderListItem{})
	}
	items := make([]authProviderListItem, 0)
	for _, kind := range []string{identityregistry.ProviderKindAuth, identityregistry.ProviderKindRecovery} {
		for _, provider := range h.providerCatalog.Providers(kind) {
			if len(provider.Operations) == 0 || provider.Artifact.Core {
				continue
			}
			ops := make([]string, 0, len(provider.Operations))
			for _, operation := range provider.Operations {
				ops = append(ops, operation.Name)
			}
			items = append(items, authProviderListItem{
				ID:              provider.ID,
				Kind:            provider.Kind,
				ContractVersion: provider.ContractVersion,
				Priority:        provider.Priority,
				Operations:      ops,
			})
		}
	}
	return apphttp.OK(c, items)
}

type authProviderListItem struct {
	ID              string   `json:"id"`
	Kind            string   `json:"kind"`
	ContractVersion string   `json:"contractVersion"`
	Priority        int      `json:"priority"`
	Operations      []string `json:"operations"`
}

type authProviderStartRequest struct {
	CorrelationID     string `json:"correlationId"`
	DeviceFingerprint string `json:"deviceFingerprint"`
	ClientClass       string `json:"clientClass"`
	RedirectHint      string `json:"redirectHint"`
	AccountHint       string `json:"accountHint"`
}

type authProviderCompleteRequest struct {
	CorrelationID     string `json:"correlationId"`
	CompletionToken   string `json:"completionToken"`
	DeviceFingerprint string `json:"deviceFingerprint"`
	ClientClass       string `json:"clientClass"`
	IdempotencyKey    string `json:"idempotencyKey"`
}

func (h *Controller) authProviderStart(c fiber.Ctx) error {
	providerID := strings.ToLower(strings.TrimSpace(c.Params("providerId")))
	kind := strings.ToLower(strings.TrimSpace(c.Params("operation")))
	operation, err := authProviderStartOperation(kind)
	if err != nil {
		return err
	}
	var req authProviderStartRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "invalid request body")
	}
	// recovery.start 走 recovery flow。
	if operation == identity.RecoveryOperationStart {
		if h.recoveryFlow == nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "auth.provider_unavailable")
		}
		result, err := h.recoveryFlow.Start(c.Context(), identity.RecoveryProviderStartInput{
			ProviderID: providerID, CorrelationID: req.CorrelationID,
			DeviceFingerprint: req.DeviceFingerprint, ClientClass: req.ClientClass,
			AccountHint: req.AccountHint,
		})
		if err != nil {
			return mapAuthProviderHTTPError(err)
		}
		return apphttp.OK(c, map[string]any{
			"providerId": result.ProviderID, "operation": operation,
			"status": result.Status, "correlationId": result.CorrelationID,
			"continueToken": result.ContinueToken, "redirectUrl": result.RedirectURL,
			"challengeKind": result.ChallengeKind,
		})
	}
	if h.authFlow == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "auth.provider_unavailable")
	}
	actorUserID := int64(0)
	if operation == identity.AuthOperationLinkStart {
		actor, actorErr := h.actor(c)
		if actorErr != nil {
			return actorErr
		}
		actorUserID = actor.ID
	}
	result, err := h.authFlow.Start(c.Context(), identity.AuthProviderStartInput{
		ProviderID: providerID, Operation: operation, ActorUserID: actorUserID,
		CorrelationID: req.CorrelationID, DeviceFingerprint: req.DeviceFingerprint,
		ClientClass: req.ClientClass, RedirectHint: req.RedirectHint,
	})
	if err != nil {
		return mapAuthProviderHTTPError(err)
	}
	return apphttp.OK(c, map[string]any{
		"providerId": result.ProviderID, "operation": result.Operation,
		"status": result.Status, "correlationId": result.CorrelationID,
		"continueToken": result.ContinueToken, "redirectUrl": result.RedirectURL,
		"challengeKind": result.ChallengeKind,
	})
}

func (h *Controller) authProviderComplete(c fiber.Ctx) error {
	providerID := strings.ToLower(strings.TrimSpace(c.Params("providerId")))
	kind := strings.ToLower(strings.TrimSpace(c.Params("operation")))
	operation, err := authProviderCompleteOperation(kind)
	if err != nil {
		return err
	}
	var req authProviderCompleteRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "invalid request body")
	}
	if operation == identity.RecoveryOperationComplete {
		if h.recoveryFlow == nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "auth.provider_unavailable")
		}
		result, err := h.recoveryFlow.Complete(c.Context(), identity.RecoveryProviderCompleteInput{
			ProviderID: providerID, CorrelationID: req.CorrelationID,
			CompletionToken: req.CompletionToken, DeviceFingerprint: req.DeviceFingerprint,
			ClientClass: req.ClientClass,
		})
		if err != nil {
			return mapAuthProviderHTTPError(err)
		}
		return apphttp.OK(c, map[string]any{
			"providerId": result.ProviderID, "operation": operation,
			"providerSubjectDigest": result.SubjectDigest, "userHintId": result.UserHintID,
		})
	}
	if h.authFlow == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "auth.provider_unavailable")
	}
	actorUserID := int64(0)
	targetUserID := int64(0)
	if operation == identity.AuthOperationLinkComplete {
		actor, actorErr := h.actor(c)
		if actorErr != nil {
			return actorErr
		}
		actorUserID = actor.ID
		targetUserID = actor.ID
	}
	result, err := h.authFlow.Complete(c.Context(), identity.AuthProviderCompleteInput{
		ProviderID: providerID, Operation: operation,
		ActorUserID: actorUserID, TargetUserID: targetUserID,
		CorrelationID: req.CorrelationID, CompletionToken: req.CompletionToken,
		DeviceFingerprint: req.DeviceFingerprint, ClientClass: req.ClientClass,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return mapAuthProviderHTTPError(err)
	}
	response := map[string]any{
		"providerId": result.ProviderID, "operation": result.Operation,
		"providerSubjectDigest": result.SubjectDigest,
		"displayName":           result.DisplayName,
		// emailHint 仅作展示提示，从不用于自动链接。
		"emailHint": result.EmailHint,
	}
	if result.Link != nil {
		response["linkId"] = result.Link.Link.ID
		response["linkStatus"] = result.Link.Link.Status
	}
	return apphttp.OK(c, response)
}

func authProviderStartOperation(kind string) (string, error) {
	switch kind {
	case "registration":
		return identity.AuthOperationRegistrationStart, nil
	case "login":
		return identity.AuthOperationLoginStart, nil
	case "link":
		return identity.AuthOperationLinkStart, nil
	case "recovery":
		return identity.RecoveryOperationStart, nil
	default:
		return "", fiber.NewError(fiber.StatusNotFound, "auth.provider_operation_not_found")
	}
}

func authProviderCompleteOperation(kind string) (string, error) {
	switch kind {
	case "registration":
		return identity.AuthOperationRegistrationComplete, nil
	case "login":
		return identity.AuthOperationLoginComplete, nil
	case "link":
		return identity.AuthOperationLinkComplete, nil
	case "recovery":
		return identity.RecoveryOperationComplete, nil
	default:
		return "", fiber.NewError(fiber.StatusNotFound, "auth.provider_operation_not_found")
	}
}

func (h *Controller) listProfileSections(c fiber.Ctx) error {
	if h.profileComposer == nil {
		return apphttp.OK(c, []identity.ProfileSection{})
	}
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	sections, err := h.profileComposer.ListSections(c.Context(), actor.ID, actor.ID)
	if err != nil {
		return mapAuthProviderHTTPError(err)
	}
	if sections == nil {
		sections = []identity.ProfileSection{}
	}
	return apphttp.OK(c, sections)
}

type profileSectionUpdateRequest struct {
	ProviderID string         `json:"providerId"`
	Fields     map[string]any `json:"fields"`
}

func (h *Controller) updateProfileSection(c fiber.Ctx) error {
	if h.profileComposer == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "auth.provider_unavailable")
	}
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	sectionID := strings.TrimSpace(c.Params("sectionId"))
	var req profileSectionUpdateRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "invalid request body")
	}
	providerID := strings.ToLower(strings.TrimSpace(req.ProviderID))
	if providerID == "" {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "providerId is required")
	}
	section, err := h.profileComposer.UpdateSection(
		c.Context(), providerID, sectionID, actor.ID, actor.ID, req.Fields,
	)
	if err != nil {
		return mapAuthProviderHTTPError(err)
	}
	return apphttp.OK(c, section)
}

func mapAuthProviderHTTPError(err error) error {
	switch {
	case errors.Is(err, identity.ErrAuthProviderNotFound),
		errors.Is(err, identity.ErrProfileProviderNotFound),
		errors.Is(err, identity.ErrRecoveryProviderNotFound):
		return fiber.NewError(fiber.StatusNotFound, "auth.provider_not_found")
	case errors.Is(err, identity.ErrAuthProviderFlowInvalid),
		errors.Is(err, identity.ErrProfileProviderInvalid),
		errors.Is(err, identity.ErrRecoveryProviderInvalid):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "auth.provider_input_invalid")
	case errors.Is(err, identity.ErrAuthProviderFlowUnavailable),
		errors.Is(err, identity.ErrProfileProviderUnavailable),
		errors.Is(err, identity.ErrRecoveryProviderUnavailable):
		return fiber.NewError(fiber.StatusServiceUnavailable, "auth.provider_unavailable")
	case errors.Is(err, identity.ErrExternalIdentitySubjectConflict):
		return fiber.NewError(fiber.StatusConflict, "auth.external_subject_conflict")
	case errors.Is(err, identity.ErrExternalIdentityLinkStateConflict),
		errors.Is(err, identity.ErrExternalIdentityLinkIdempotencyConflict):
		return fiber.NewError(fiber.StatusConflict, "auth.external_link_conflict")
	default:
		return fiber.NewError(fiber.StatusServiceUnavailable, "auth.provider_unavailable")
	}
}

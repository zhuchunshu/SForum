package identitycontroller

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

// createExternalRegistrationContinuation is the only ticket issuer for both
// explicit provider registration and an unlinked provider login.
func (h *Controller) createExternalRegistrationContinuation(
	ctx context.Context,
	assertion identity.ExternalAuthAssertion,
	returnPath string,
) (string, error) {
	if h.externalAuthService == nil || h.registrationTicketStore == nil {
		return "", identity.ErrExternalAuthProviderUnavailable
	}
	if err := h.externalAuthService.ValidateRegistrationContinuation(ctx, assertion); err != nil {
		return "", err
	}
	token, err := identity.GenerateOpaqueToken()
	if err != nil {
		return "", identity.ErrExternalAuthProviderUnavailable
	}
	now := time.Now()
	ticket := identity.RegistrationTicket{
		Token:                   token,
		ProviderID:              assertion.ProviderID,
		ProviderContractVersion: assertion.ProviderContractVersion,
		OwnerExtensionID:        assertion.OwnerExtensionID,
		OwnerExtensionVersion:   assertion.OwnerExtensionVersion,
		OwnerPackageDigest:      assertion.OwnerPackageDigest,
		Operation:               identity.ExternalAuthOperationRegistration,
		SourceOperation:         assertion.Operation,
		ProviderSubject:         assertion.ProviderSubject,
		SubjectDigest:           assertion.SubjectDigest,
		UsernameHint:            assertion.UsernameHint,
		DisplayName:             assertion.DisplayName,
		EmailHint:               assertion.EmailHint,
		EmailVerified:           assertion.EmailVerified,
		CorrelationID:           assertion.CorrelationID,
		CreatedAt:               now,
		ExpiresAt:               now.Add(identity.RegistrationTicketDefaultTTL),
	}
	if err := h.registrationTicketStore.Save(ctx, ticket); err != nil {
		return "", identity.ErrExternalAuthProviderUnavailable
	}
	return identity.ExternalRegistrationContinuationPath(token, returnPath), nil
}

// externalRegistrationPreparation returns only editable, redacted form hints.
// Inspect does not consume the ticket; submission remains the one-use boundary.
func (h *Controller) externalRegistrationPreparation(c fiber.Ctx) error {
	if h.externalAuthService == nil || h.registrationTicketStore == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "auth.provider_unavailable")
	}
	var req externalRegistrationPreparationRequest
	if err := c.Bind().Body(&req); err != nil || strings.TrimSpace(req.Ticket) == "" {
		return fiber.NewError(fiber.StatusNotFound, "auth.external_registration_ticket_invalid")
	}
	ticket, err := h.registrationTicketStore.Inspect(c.Context(), strings.TrimSpace(req.Ticket))
	if err != nil {
		return mapRegistrationTicketError(err)
	}
	preparation, err := h.externalAuthService.PrepareExternalRegistration(c.Context(), ticket)
	if err != nil {
		return mapExternalAuthRegistrationError(err)
	}
	return apphttp.OK(c, preparation)
}

type externalRegistrationPreparationRequest struct {
	Ticket string `json:"ticket"`
}

func mapRegistrationContinuationReason(err error) string {
	switch {
	case errors.Is(err, identity.ErrExternalAuthBootstrapRequired):
		return "auth.external_bootstrap_required"
	case errors.Is(err, identity.ErrRegistrationDisabled):
		return "auth.registration_disabled"
	case errors.Is(err, identity.ErrExternalAuthOperationNotActivated),
		errors.Is(err, identity.ErrAuthProviderNotFound):
		return "auth.provider_not_enabled"
	default:
		return mapExternalAuthReason(err)
	}
}

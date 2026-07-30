package identity

import (
	"context"
	"errors"
	"strings"
	"time"
)

// ExternalAuthContinuationPreparation exposes only presentation-safe hints and
// the two Host-authorized terminal choices. It never includes the subject,
// artifact identity, OAuth material, or browser-binding digest.
type ExternalAuthContinuationPreparation struct {
	ProviderID      string `json:"providerId"`
	UsernameHint    string `json:"usernameHint"`
	DisplayName     string `json:"displayName"`
	EmailHint       string `json:"emailHint"`
	EmailVerified   bool   `json:"emailVerified"`
	CanLinkExisting bool   `json:"canLinkExisting"`
	CanRegister     bool   `json:"canRegister"`
}

type ExternalAuthContinuationAvailability struct {
	CanLinkExisting bool
	CanRegister     bool
}

// ValidateRegistrationContinuation authorizes an assertion-only provider result
// to enter the single Host external-registration flow. A login assertion may
// continue only while both login and registration remain effectively active.
func (s *ExternalAuthService) ValidateRegistrationContinuation(ctx context.Context, assertion ExternalAuthAssertion) error {
	sourceOperation := assertion.Operation
	if sourceOperation != ExternalAuthOperationLogin && sourceOperation != ExternalAuthOperationRegistration {
		return ErrExternalAuthOperationMismatch
	}
	if err := s.ensureExternalRegistrationPolicy(ctx); err != nil {
		return err
	}
	if err := s.RequireActivated(ctx, assertion.ProviderID, sourceOperation); err != nil {
		return err
	}
	if sourceOperation != ExternalAuthOperationRegistration {
		if err := s.RequireActivated(ctx, assertion.ProviderID, ExternalAuthOperationRegistration); err != nil {
			return err
		}
	}
	live, err := s.providerContribution(assertion.ProviderID)
	if err != nil {
		return ErrExternalAuthProviderUnavailable
	}
	if !assertion.MatchesLiveContribution(live) ||
		!authProviderHasOperation(live, externalOpToCompleteName(sourceOperation)) ||
		!authProviderHasOperation(live, AuthOperationRegistrationComplete) {
		return ErrExternalAuthArtifactMismatch
	}
	return nil
}

// ValidateExistingAccountContinuation authorizes a login assertion for binding
// to a recently authenticated local account. The separate link activation stays
// authoritative even though no second provider OAuth round-trip is required.
func (s *ExternalAuthService) ValidateExistingAccountContinuation(ctx context.Context, assertion ExternalAuthAssertion) error {
	if assertion.Operation != ExternalAuthOperationLogin {
		return ErrExternalAuthOperationMismatch
	}
	if err := s.RequireActivated(ctx, assertion.ProviderID, ExternalAuthOperationLogin); err != nil {
		return err
	}
	if err := s.RequireActivated(ctx, assertion.ProviderID, ExternalAuthOperationLink); err != nil {
		return err
	}
	live, err := s.providerContribution(assertion.ProviderID)
	if err != nil {
		return ErrExternalAuthProviderUnavailable
	}
	if !assertion.MatchesLiveContribution(live) ||
		!authProviderHasOperation(live, AuthOperationLoginComplete) ||
		!authProviderHasOperation(live, AuthOperationLinkComplete) {
		return ErrExternalAuthArtifactMismatch
	}
	return nil
}

// ValidateLoginContinuation resolves the two independently controlled terminal
// effects for one unlinked login assertion.
func (s *ExternalAuthService) ValidateLoginContinuation(ctx context.Context, assertion ExternalAuthAssertion) (ExternalAuthContinuationAvailability, error) {
	if assertion.Operation != ExternalAuthOperationLogin {
		return ExternalAuthContinuationAvailability{}, ErrExternalAuthOperationMismatch
	}
	linkErr := s.ValidateExistingAccountContinuation(ctx, assertion)
	registerErr := s.ValidateRegistrationContinuation(ctx, assertion)
	availability := ExternalAuthContinuationAvailability{
		CanLinkExisting: linkErr == nil,
		CanRegister:     registerErr == nil,
	}
	if availability.CanLinkExisting || availability.CanRegister {
		return availability, nil
	}
	if errors.Is(linkErr, ErrExternalAuthArtifactMismatch) || errors.Is(registerErr, ErrExternalAuthArtifactMismatch) {
		return ExternalAuthContinuationAvailability{}, ErrExternalAuthArtifactMismatch
	}
	if errors.Is(linkErr, ErrExternalAuthProviderUnavailable) || errors.Is(registerErr, ErrExternalAuthProviderUnavailable) {
		return ExternalAuthContinuationAvailability{}, ErrExternalAuthProviderUnavailable
	}
	return ExternalAuthContinuationAvailability{}, linkErr
}

// PrepareExternalAuthContinuation rechecks both choices independently. Closed
// registration does not suppress existing-account binding, and a disabled link
// operation does not suppress registration when that path remains authorized.
func (s *ExternalAuthService) PrepareExternalAuthContinuation(ctx context.Context, ticket RegistrationTicket) (ExternalAuthContinuationPreparation, error) {
	if err := ticket.ValidateBinding(); err != nil {
		return ExternalAuthContinuationPreparation{}, err
	}
	if ticket.IsExpired(time.Now()) {
		return ExternalAuthContinuationPreparation{}, ErrRegistrationTicketExpired
	}
	assertion := externalAuthAssertionFromContinuationTicket(ticket)
	if assertion.Operation != ExternalAuthOperationLogin {
		return ExternalAuthContinuationPreparation{}, ErrExternalAuthOperationMismatch
	}

	availability, err := s.ValidateLoginContinuation(ctx, assertion)
	if err != nil {
		return ExternalAuthContinuationPreparation{}, err
	}

	preparation := ExternalAuthContinuationPreparation{
		ProviderID:      ticket.ProviderID,
		UsernameHint:    strings.TrimSpace(ticket.UsernameHint),
		DisplayName:     strings.TrimSpace(ticket.DisplayName),
		EmailVerified:   ticket.EmailVerified && strings.TrimSpace(ticket.EmailHint) != "",
		CanLinkExisting: availability.CanLinkExisting,
		CanRegister:     availability.CanRegister,
	}
	if preparation.EmailVerified {
		preparation.EmailHint = strings.ToLower(strings.TrimSpace(ticket.EmailHint))
	}
	return preparation, nil
}

// PrepareExternalRegistration rechecks a non-consuming ticket before returning
// editable hints. Unverified provider email is never returned for autofill.
func (s *ExternalAuthService) PrepareExternalRegistration(ctx context.Context, ticket RegistrationTicket) (ExternalRegistrationPreparation, error) {
	if err := ticket.ValidateBinding(); err != nil {
		return ExternalRegistrationPreparation{}, err
	}
	if ticket.IsExpired(time.Now()) {
		return ExternalRegistrationPreparation{}, ErrRegistrationTicketExpired
	}
	assertion := externalAuthAssertionFromContinuationTicket(ticket)
	if err := s.ValidateRegistrationContinuation(ctx, assertion); err != nil {
		return ExternalRegistrationPreparation{}, err
	}
	preparation := ExternalRegistrationPreparation{
		UsernameHint:  strings.TrimSpace(ticket.UsernameHint),
		DisplayName:   strings.TrimSpace(ticket.DisplayName),
		EmailVerified: ticket.EmailVerified && strings.TrimSpace(ticket.EmailHint) != "",
	}
	if preparation.EmailVerified {
		preparation.EmailHint = strings.ToLower(strings.TrimSpace(ticket.EmailHint))
	}
	return preparation, nil
}

func externalAuthAssertionFromContinuationTicket(ticket RegistrationTicket) ExternalAuthAssertion {
	return ExternalAuthAssertion{
		ProviderID:              ticket.ProviderID,
		ProviderContractVersion: ticket.ProviderContractVersion,
		OwnerExtensionID:        ticket.OwnerExtensionID,
		OwnerExtensionVersion:   ticket.OwnerExtensionVersion,
		OwnerPackageDigest:      ticket.OwnerPackageDigest,
		Operation:               ticket.SourceOperation,
		SourceOperation:         ticket.SourceOperation,
		ProviderSubject:         ticket.ProviderSubject,
		SubjectDigest:           ticket.SubjectDigest,
		UsernameHint:            ticket.UsernameHint,
		DisplayName:             ticket.DisplayName,
		EmailHint:               ticket.EmailHint,
		EmailVerified:           ticket.EmailVerified,
		CorrelationID:           ticket.CorrelationID,
	}
}

// CompleteAuthenticatedContinuation links the login assertion to the current
// active user after a session-bound recent authentication. The persisted audit
// truthfully records login.complete as the assertion source and link as the Host
// effect; it never pretends the plugin performed a second link.complete call.
func (s *ExternalAuthService) CompleteAuthenticatedContinuation(
	ctx context.Context,
	assertion ExternalAuthAssertion,
	actorUserID int64,
	sessionFingerprint string,
) (ExternalAuthLinkResult, error) {
	if s == nil || s.deps.LinkStore == nil {
		return ExternalAuthLinkResult{}, ErrExternalAuthProviderUnavailable
	}
	current, err := s.AuthorizeExistingAccountContinuation(ctx, assertion, actorUserID, sessionFingerprint)
	if err != nil {
		return ExternalAuthLinkResult{}, err
	}
	digest, err := assertion.resolvedDigest()
	if err != nil {
		return ExternalAuthLinkResult{}, err
	}
	existing, err := s.deps.LinkStore.FindActive(ctx, assertion.ProviderID, digest)
	if err == nil {
		if existing.UserID != actorUserID {
			return ExternalAuthLinkResult{}, ErrExternalIdentitySubjectConflict
		}
		return ExternalAuthLinkResult{User: current, ProviderID: assertion.ProviderID, LinkID: existing.ID}, nil
	}
	if !errors.Is(err, ErrExternalIdentityLinkNotFound) {
		return ExternalAuthLinkResult{}, err
	}

	contribution, err := s.providerContribution(assertion.ProviderID)
	if err != nil {
		return ExternalAuthLinkResult{}, ErrExternalAuthProviderUnavailable
	}
	mutation, err := s.deps.LinkStore.Link(ctx, LinkExternalIdentityInput{
		UserID:                actorUserID,
		Provider:              contribution,
		ProviderOperation:     AuthOperationLoginComplete,
		ProviderSubjectDigest: digest,
		ActorUserID:           actorUserID,
		IdempotencyKey:        "continuation:" + assertion.CorrelationID,
	}, func() error {
		return s.ValidateExistingAccountContinuation(ctx, assertion)
	})
	if err != nil {
		return ExternalAuthLinkResult{}, err
	}
	return ExternalAuthLinkResult{
		User: current, ProviderID: assertion.ProviderID, LinkID: mutation.Link.ID,
	}, nil
}

// AuthorizeExistingAccountContinuation is the non-consuming admission check
// used before a controller consumes the one-use ticket. Complete reruns it so
// activation, recent-auth, and actor status cannot drift between the two steps.
func (s *ExternalAuthService) AuthorizeExistingAccountContinuation(
	ctx context.Context,
	assertion ExternalAuthAssertion,
	actorUserID int64,
	sessionFingerprint string,
) (CurrentUser, error) {
	if actorUserID <= 0 {
		return CurrentUser{}, ErrExternalAuthActorRequired
	}
	if err := s.ValidateExistingAccountContinuation(ctx, assertion); err != nil {
		return CurrentUser{}, err
	}
	recent, err := s.isSessionRecentlyAuthenticated(ctx, actorUserID, sessionFingerprint)
	if err != nil {
		return CurrentUser{}, err
	}
	if !recent {
		return CurrentUser{}, ErrExternalAuthRecentAuthRequired
	}
	current, err := s.loadCurrentUser(ctx, actorUserID)
	if err != nil {
		return CurrentUser{}, err
	}
	if current.Status != UserStatusActive {
		return CurrentUser{}, ErrExternalAuthActorInactive
	}
	return current, nil
}

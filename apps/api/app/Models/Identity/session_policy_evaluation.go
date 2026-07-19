package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"

	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
)

const (
	// Session evaluation only runs before Host issue or renew. Revocation is
	// Host-local and never consults a provider.
	SessionEvaluationPurposeIssue = "issue"
	SessionEvaluationPurposeRenew = "renew"

	SessionPolicyDispositionAllow  = "allow"
	SessionPolicyDispositionDeny   = "deny"
	SessionPolicyDispositionStepUp = "step_up"

	sessionEvaluateOperation = "session.evaluate"
)

var (
	ErrSessionPolicyEvaluationInvalid     = errors.New("identity: session policy evaluation input is invalid")
	ErrSessionPolicyEvaluationDenied      = errors.New("identity: session policy denied the session effect")
	ErrSessionPolicyEvaluationStepUp      = errors.New("identity: session policy requires step-up verification")
	ErrSessionPolicyEvaluationUnavailable = errors.New("identity: session policy evaluation is unavailable")
	ErrSessionPolicyEvaluationStale       = errors.New("identity: session policy evaluation became stale")
)

// SessionEvaluationInput is the Host-owned workflow claim for issue/renew.
// Plugins never receive passwords, raw cookies, session ids, or CSRF tokens.
type SessionEvaluationInput struct {
	UserID  int64
	Purpose string
	// CorrelationID is an opaque Host correlation token, never a session secret.
	CorrelationID string
	// DeviceFingerprint is a Host-derived opaque device class, never a raw UA.
	DeviceFingerprint string
}

// SessionEvaluationResult is the Host disposition after exact resolution and
// optional plugin evaluation. It is never authorization by itself: the Host
// still owns session creation, renewal, and revocation.
type SessionEvaluationResult struct {
	Disposition      string
	PolicyID         string
	Source           string
	Selection        *IdentitySessionPolicySelection
	Provider         *identityregistry.ProviderContribution
	RegistryRevision uint64
	RegistryDigest   string
	// SelectionRevision is the durable selection tip observed at resolution.
	// Host rechecks it immediately before the session effect.
	SelectionRevision int64
	Output            map[string]any
}

// SessionPolicyProviderResolution freezes one coherent selected policy claim
// for InvokeExact. It is not final execution authority.
type SessionPolicyProviderResolution struct {
	IdentitySessionPolicyResolution
	// Operation is always session.evaluate for plugin sources; empty for Core.
	Operation string
}

// SessionPolicyEvaluateInvoker invokes one exact active session provider.
// Accept must call the returned fence exactly once before the Host session
// effect commits; the invoker retains exact runtime admission through Accept.
type SessionPolicyEvaluateInvoker interface {
	InvokeExact(
		ctx context.Context,
		provider identityregistry.ProviderContribution,
		operation string,
		actorUserID int64,
		input map[string]any,
		accept func(context.Context, map[string]any, func() error) error,
	) error
}

// SessionPolicyEvaluator resolves the selected policy and, when required,
// invokes the exact session.evaluate provider before Host issue/renew.
type SessionPolicyEvaluator struct {
	store   IdentitySessionPolicyStore
	invoker SessionPolicyEvaluateInvoker
}

func NewSessionPolicyEvaluator(
	store IdentitySessionPolicyStore,
	invoker SessionPolicyEvaluateInvoker,
) (*SessionPolicyEvaluator, error) {
	if store == nil {
		return nil, ErrIdentitySessionPolicyStoreUnavailable
	}
	return &SessionPolicyEvaluator{store: store, invoker: invoker}, nil
}

// ProviderResolution freezes the effective selected session policy and its
// exact executable provider claim. Core and Safe Mode resolve without a plugin.
func (e *SessionPolicyEvaluator) ProviderResolution(
	ctx context.Context,
) (SessionPolicyProviderResolution, error) {
	if e == nil || e.store == nil || ctx == nil {
		return SessionPolicyProviderResolution{}, ErrSessionPolicyEvaluationInvalid
	}
	if err := ctx.Err(); err != nil {
		return SessionPolicyProviderResolution{}, err
	}
	resolved, err := e.store.Resolve(ctx)
	if err != nil {
		return SessionPolicyProviderResolution{}, err
	}
	return freezeSessionPolicyProviderResolution(resolved)
}

// InvokeExact evaluates one already-resolved policy claim. Core and Safe Mode
// return allow without calling a plugin. Plugin sources require an invoker and
// fail closed on missing, malformed, timed-out, or non-allow dispositions.
func (e *SessionPolicyEvaluator) InvokeExact(
	ctx context.Context,
	resolution SessionPolicyProviderResolution,
	input SessionEvaluationInput,
) (SessionEvaluationResult, error) {
	if e == nil || e.store == nil || ctx == nil {
		return SessionEvaluationResult{}, ErrSessionPolicyEvaluationInvalid
	}
	if err := ctx.Err(); err != nil {
		return SessionEvaluationResult{}, err
	}
	prepared, err := prepareSessionEvaluationInput(input)
	if err != nil {
		return SessionEvaluationResult{}, err
	}
	if err := validateSessionPolicyProviderResolution(resolution); err != nil {
		return SessionEvaluationResult{}, err
	}

	switch resolution.Source {
	case IdentitySessionPolicySourceCore, IdentitySessionPolicySourceSafeMode:
		return coreSessionEvaluationResult(resolution, prepared), nil
	case IdentitySessionPolicySourcePlugin:
		return e.invokePluginSessionEvaluate(ctx, resolution, prepared)
	default:
		return SessionEvaluationResult{}, ErrSessionPolicyEvaluationUnavailable
	}
}

// Evaluate is the atomic production path: ProviderResolution, InvokeExact, and
// a selection-revision recheck immediately before the Host effect boundary.
// Callers must still keep session issue/renew Host-local and never route
// revocation through this path.
func (e *SessionPolicyEvaluator) Evaluate(
	ctx context.Context,
	input SessionEvaluationInput,
) (SessionEvaluationResult, error) {
	resolution, err := e.ProviderResolution(ctx)
	if err != nil {
		return SessionEvaluationResult{}, err
	}
	result, err := e.InvokeExact(ctx, resolution, input)
	if err != nil {
		return SessionEvaluationResult{}, err
	}
	if err := e.recheckSelectionBeforeEffect(ctx, result); err != nil {
		return SessionEvaluationResult{}, err
	}
	return result, nil
}

// RequireAllow evaluates and returns only when the disposition is allow.
// Deny and step-up map to stable typed errors for Host issue/renew mapping.
func (e *SessionPolicyEvaluator) RequireAllow(
	ctx context.Context,
	input SessionEvaluationInput,
) (SessionEvaluationResult, error) {
	result, err := e.Evaluate(ctx, input)
	if err != nil {
		return SessionEvaluationResult{}, err
	}
	switch result.Disposition {
	case SessionPolicyDispositionAllow:
		return result, nil
	case SessionPolicyDispositionDeny:
		return result, ErrSessionPolicyEvaluationDenied
	case SessionPolicyDispositionStepUp:
		return result, ErrSessionPolicyEvaluationStepUp
	default:
		return SessionEvaluationResult{}, ErrSessionPolicyEvaluationUnavailable
	}
}

func (e *SessionPolicyEvaluator) invokePluginSessionEvaluate(
	ctx context.Context,
	resolution SessionPolicyProviderResolution,
	input preparedSessionEvaluationInput,
) (SessionEvaluationResult, error) {
	if e.invoker == nil || resolution.Provider == nil {
		return SessionEvaluationResult{}, ErrSessionPolicyEvaluationUnavailable
	}
	provider := *resolution.Provider
	if !identitySessionPolicyProviderHasEvaluate(provider) {
		return SessionEvaluationResult{}, ErrSessionPolicyEvaluationUnavailable
	}

	requestInput := map[string]any{
		"purpose": input.Purpose,
		"userId":  input.UserID,
	}
	if input.CorrelationID != "" {
		requestInput["correlationId"] = input.CorrelationID
	}
	if input.DeviceFingerprint != "" {
		requestInput["deviceFingerprint"] = input.DeviceFingerprint
	}

	var output map[string]any
	err := e.invoker.InvokeExact(
		ctx,
		provider,
		sessionEvaluateOperation,
		input.UserID,
		requestInput,
		func(_ context.Context, proposal map[string]any, fence func() error) error {
			if fence == nil {
				return ErrSessionPolicyEvaluationInvalid
			}
			// Fence proves exact runtime admission is still held; the Host
			// session effect rechecks selection revision separately.
			if err := fence(); err != nil {
				return err
			}
			cloned, err := cloneSessionEvaluationDocument(proposal)
			if err != nil {
				return err
			}
			output = cloned
			return nil
		},
	)
	if err != nil {
		return SessionEvaluationResult{}, errors.Join(ErrSessionPolicyEvaluationUnavailable, err)
	}
	disposition, err := parseSessionPolicyDisposition(output)
	if err != nil {
		return SessionEvaluationResult{}, err
	}
	return SessionEvaluationResult{
		Disposition:       disposition,
		PolicyID:          resolution.PolicyID,
		Source:            resolution.Source,
		Selection:         cloneSessionPolicySelection(resolution.Selection),
		Provider:          cloneSessionPolicyProvider(resolution.Provider),
		RegistryRevision:  resolution.RegistryRevision,
		RegistryDigest:    resolution.RegistryDigest,
		SelectionRevision: selectionRevision(resolution.Selection),
		Output:            output,
	}, nil
}

func (e *SessionPolicyEvaluator) recheckSelectionBeforeEffect(
	ctx context.Context,
	result SessionEvaluationResult,
) error {
	// Core/Safe Mode have no durable plugin selection tip to race against when
	// Safe Mode omitted Selection; still re-resolve to catch Safe Mode flips
	// and Core tip changes.
	current, err := e.store.Resolve(ctx)
	if err != nil {
		return errors.Join(ErrSessionPolicyEvaluationStale, err)
	}
	if current.PolicyID != result.PolicyID || current.Source != result.Source ||
		current.RegistryRevision != result.RegistryRevision ||
		current.RegistryDigest != result.RegistryDigest {
		return ErrSessionPolicyEvaluationStale
	}
	switch result.Source {
	case IdentitySessionPolicySourceCore, IdentitySessionPolicySourceSafeMode:
		if current.Provider != nil {
			return ErrSessionPolicyEvaluationStale
		}
		if result.Selection != nil {
			if current.Selection == nil ||
				current.Selection.Revision != result.SelectionRevision ||
				current.Selection.PolicyID != result.Selection.PolicyID {
				return ErrSessionPolicyEvaluationStale
			}
		}
		return nil
	case IdentitySessionPolicySourcePlugin:
		if current.Selection == nil || result.Selection == nil ||
			current.Selection.Revision != result.SelectionRevision ||
			current.Selection.IdentitySessionPolicyEvidence != result.Selection.IdentitySessionPolicyEvidence ||
			current.Provider == nil || result.Provider == nil ||
			!identitySessionPolicyProviderMatches(*current.Provider, *result.Provider) {
			return ErrSessionPolicyEvaluationStale
		}
		return nil
	default:
		return ErrSessionPolicyEvaluationStale
	}
}

type preparedSessionEvaluationInput struct {
	UserID            int64
	Purpose           string
	CorrelationID     string
	DeviceFingerprint string
}

func prepareSessionEvaluationInput(
	input SessionEvaluationInput,
) (preparedSessionEvaluationInput, error) {
	prepared := preparedSessionEvaluationInput{
		UserID:            input.UserID,
		Purpose:           strings.ToLower(strings.TrimSpace(input.Purpose)),
		CorrelationID:     strings.TrimSpace(input.CorrelationID),
		DeviceFingerprint: strings.TrimSpace(input.DeviceFingerprint),
	}
	if prepared.UserID <= 0 {
		return preparedSessionEvaluationInput{}, ErrSessionPolicyEvaluationInvalid
	}
	switch prepared.Purpose {
	case SessionEvaluationPurposeIssue, SessionEvaluationPurposeRenew:
	default:
		return preparedSessionEvaluationInput{}, ErrSessionPolicyEvaluationInvalid
	}
	if len(prepared.CorrelationID) > 128 || len(prepared.DeviceFingerprint) > 128 {
		return preparedSessionEvaluationInput{}, ErrSessionPolicyEvaluationInvalid
	}
	return prepared, nil
}

func freezeSessionPolicyProviderResolution(
	resolved IdentitySessionPolicyResolution,
) (SessionPolicyProviderResolution, error) {
	switch resolved.Source {
	case IdentitySessionPolicySourceCore, IdentitySessionPolicySourceSafeMode:
		if resolved.PolicyID != IdentitySessionPolicyCoreDefault || resolved.Provider != nil {
			return SessionPolicyProviderResolution{}, ErrSessionPolicyEvaluationUnavailable
		}
		return SessionPolicyProviderResolution{
			IdentitySessionPolicyResolution: resolved,
		}, nil
	case IdentitySessionPolicySourcePlugin:
		if resolved.PolicyID == "" || resolved.PolicyID == IdentitySessionPolicyCoreDefault ||
			resolved.Provider == nil || resolved.Selection == nil ||
			!identitySessionPolicyProviderHasEvaluate(*resolved.Provider) {
			return SessionPolicyProviderResolution{}, ErrSessionPolicyEvaluationUnavailable
		}
		return SessionPolicyProviderResolution{
			IdentitySessionPolicyResolution: resolved,
			Operation:                       sessionEvaluateOperation,
		}, nil
	default:
		return SessionPolicyProviderResolution{}, ErrSessionPolicyEvaluationUnavailable
	}
}

func validateSessionPolicyProviderResolution(
	resolution SessionPolicyProviderResolution,
) error {
	switch resolution.Source {
	case IdentitySessionPolicySourceCore, IdentitySessionPolicySourceSafeMode:
		if resolution.PolicyID != IdentitySessionPolicyCoreDefault ||
			resolution.Provider != nil || resolution.Operation != "" {
			return ErrSessionPolicyEvaluationUnavailable
		}
		return nil
	case IdentitySessionPolicySourcePlugin:
		if resolution.Operation != sessionEvaluateOperation ||
			resolution.Provider == nil || resolution.Selection == nil ||
			!identitySessionPolicyProviderHasEvaluate(*resolution.Provider) {
			return ErrSessionPolicyEvaluationUnavailable
		}
		return nil
	default:
		return ErrSessionPolicyEvaluationUnavailable
	}
}

func coreSessionEvaluationResult(
	resolution SessionPolicyProviderResolution,
	input preparedSessionEvaluationInput,
) SessionEvaluationResult {
	_ = input
	return SessionEvaluationResult{
		Disposition:       SessionPolicyDispositionAllow,
		PolicyID:          resolution.PolicyID,
		Source:            resolution.Source,
		Selection:         cloneSessionPolicySelection(resolution.Selection),
		RegistryRevision:  resolution.RegistryRevision,
		RegistryDigest:    resolution.RegistryDigest,
		SelectionRevision: selectionRevision(resolution.Selection),
		Output: map[string]any{
			"disposition": SessionPolicyDispositionAllow,
			"source":      resolution.Source,
		},
	}
}

func parseSessionPolicyDisposition(output map[string]any) (string, error) {
	if output == nil {
		return "", ErrSessionPolicyEvaluationUnavailable
	}
	raw, ok := output["disposition"].(string)
	if !ok {
		return "", ErrSessionPolicyEvaluationUnavailable
	}
	disposition := strings.ToLower(strings.TrimSpace(raw))
	switch disposition {
	case SessionPolicyDispositionAllow, SessionPolicyDispositionDeny, SessionPolicyDispositionStepUp:
		return disposition, nil
	default:
		return "", fmt.Errorf("%w: disposition %q", ErrSessionPolicyEvaluationUnavailable, disposition)
	}
}

func selectionRevision(selection *IdentitySessionPolicySelection) int64 {
	if selection == nil {
		return 0
	}
	return selection.Revision
}

func cloneSessionPolicySelection(
	input *IdentitySessionPolicySelection,
) *IdentitySessionPolicySelection {
	if input == nil {
		return nil
	}
	cloned := *input
	return &cloned
}

func cloneSessionPolicyProvider(
	input *identityregistry.ProviderContribution,
) *identityregistry.ProviderContribution {
	if input == nil {
		return nil
	}
	cloned := *input
	cloned.Operations = append([]identityregistry.ProviderOperation(nil), input.Operations...)
	return &cloned
}

func cloneSessionEvaluationDocument(input map[string]any) (map[string]any, error) {
	if input == nil {
		return map[string]any{}, nil
	}
	// Shallow-safe copy of the Host-validated provider proposal. Nested maps are
	// not authority; only the disposition string is interpreted.
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result, nil
}

package bootstrap

import (
	"context"
	"sync"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
)

// sessionPolicyRenewalGate is installed on AuthSession before the lifecycle
// stack exists, then bound to the Host evaluator once registries are ready.
// Unbound evaluation is Core-allow equivalent and runs the Host effect directly,
// so early requests during boot never call a plugin. After bind, renew fails
// closed on deny/unavailable.
type sessionPolicyRenewalGate struct {
	mu        sync.RWMutex
	evaluator *identity.SessionPolicyEvaluator
}

func (g *sessionPolicyRenewalGate) Set(evaluator *identity.SessionPolicyEvaluator) {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.evaluator = evaluator
	g.mu.Unlock()
}

func (g *sessionPolicyRenewalGate) Evaluate(
	ctx context.Context,
	userID int64,
	tokenVersion int64,
	effect authsession.RenewalEffect,
) error {
	if effect == nil {
		return identity.ErrSessionPolicyEvaluationInvalid
	}
	if g == nil {
		return effect(ctx)
	}
	g.mu.RLock()
	evaluator := g.evaluator
	g.mu.RUnlock()
	if evaluator == nil {
		return effect(ctx)
	}
	_, err := evaluator.RequireAllowAndRun(
		ctx,
		identity.SessionEvaluationInput{
			UserID: userID, TokenVersion: tokenVersion, Purpose: identity.SessionEvaluationPurposeRenew,
		},
		identity.SessionPolicyHostEffect(effect),
	)
	return err
}

// newSessionPolicyEvaluator builds the Host evaluator from the production
// Identity Registry, Session Policy Store, and exact Manager runtime.
func newSessionPolicyEvaluator(
	manager *extensionsruntime.Manager,
	registry *identityregistry.Registry,
	store identity.IdentitySessionPolicyStore,
	stepUp identity.SessionPolicyStepUpStore,
) (*identity.SessionPolicyEvaluator, error) {
	if manager == nil || registry == nil || store == nil {
		return nil, identity.ErrIdentitySessionPolicyStoreUnavailable
	}
	runtime, err := extensionsruntime.NewIdentityProviderRuntime(manager, registry)
	if err != nil {
		return nil, err
	}
	invoker, err := extensionsruntime.NewIdentitySessionEvaluateInvoker(runtime)
	if err != nil {
		return nil, err
	}
	evaluator, err := identity.NewSessionPolicyEvaluator(store, invoker)
	if err != nil {
		return nil, err
	}
	if stepUp != nil {
		evaluator.WithStepUpStore(stepUp)
	}
	return evaluator, nil
}

// newRiskEvaluator builds Host risk composition over the same exact identity
// runtime used by session.evaluate.
func newRiskEvaluator(
	manager *extensionsruntime.Manager,
	registry *identityregistry.Registry,
) (*identity.RiskEvaluator, error) {
	if manager == nil || registry == nil {
		return nil, identity.ErrRiskEvaluationUnavailable
	}
	runtime, err := extensionsruntime.NewIdentityProviderRuntime(manager, registry)
	if err != nil {
		return nil, err
	}
	invoker, err := extensionsruntime.NewIdentitySessionEvaluateInvoker(runtime)
	if err != nil {
		return nil, err
	}
	return identity.NewRiskEvaluator(identity.RegistryRiskProviderSource{Registry: registry}, invoker)
}

package extensionsruntime

import (
	"context"
	"errors"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
)

// IdentitySessionEvaluateInvoker adapts IdentityProviderRuntime to the Host
// session-policy evaluation boundary. It only ever invokes the exact provider
// claim supplied by ProviderResolution; callers cannot substitute Core or a
// generic provider slot.
type IdentitySessionEvaluateInvoker struct {
	runtime *IdentityProviderRuntime
}

func NewIdentitySessionEvaluateInvoker(
	runtime *IdentityProviderRuntime,
) (*IdentitySessionEvaluateInvoker, error) {
	if runtime == nil {
		return nil, ErrIdentityProviderInvocationInvalid
	}
	return &IdentitySessionEvaluateInvoker{runtime: runtime}, nil
}

// InvokeExact holds exact runtime admission through Accept and requires the
// Host accept callback to call the commit fence exactly once.
func (i *IdentitySessionEvaluateInvoker) InvokeExact(
	ctx context.Context,
	provider identityregistry.ProviderContribution,
	operation string,
	actorUserID int64,
	input map[string]any,
	accept func(context.Context, map[string]any, func() error) error,
) error {
	if i == nil || i.runtime == nil || accept == nil {
		return ErrIdentityProviderInvocationInvalid
	}
	if provider.ID == "" || operation == "" {
		return ErrIdentityProviderInvocationInvalid
	}
	_, err := i.runtime.Invoke(
		ctx,
		IdentityProviderInvocation{
			ProviderID:  provider.ID,
			Operation:   operation,
			ActorUserID: actorUserID,
			Input:       input,
		},
		func(callCtx context.Context, proposal IdentityProviderInvocationResult, fence IdentityProviderCommitFence) error {
			// Defense in depth: refuse a different exact artifact than the
			// resolution claim even if Registry ids collide across revisions.
			if !identitySessionProviderMatches(proposal.Provider, provider) ||
				proposal.Operation.Name != operation {
				return errors.Join(ErrIdentityProviderStale, ErrIdentityProviderAcceptFailed)
			}
			return accept(callCtx, proposal.Output, func() error {
				if fence == nil {
					return ErrIdentityProviderInvocationInvalid
				}
				return fence()
			})
		},
	)
	return err
}

func identitySessionProviderMatches(
	left identityregistry.ProviderContribution,
	right identityregistry.ProviderContribution,
) bool {
	if left.Artifact != right.Artifact ||
		left.ID != right.ID ||
		left.ContractVersion != right.ContractVersion ||
		left.Kind != right.Kind ||
		left.Handler != right.Handler ||
		left.Priority != right.Priority ||
		len(left.Operations) != len(right.Operations) {
		return false
	}
	for index := range left.Operations {
		leftOp := left.Operations[index]
		rightOp := right.Operations[index]
		// Compare only public contract fields; bound Schema pointers are
		// Host-private and may differ across deep clones of the same claim.
		if leftOp.Name != rightOp.Name ||
			leftOp.InputSchema != rightOp.InputSchema ||
			leftOp.InputSchemaWireReference != rightOp.InputSchemaWireReference ||
			leftOp.InputSchemaDigest != rightOp.InputSchemaDigest ||
			leftOp.OutputSchema != rightOp.OutputSchema ||
			leftOp.OutputSchemaWireReference != rightOp.OutputSchemaWireReference ||
			leftOp.OutputSchemaDigest != rightOp.OutputSchemaDigest ||
			leftOp.TimeoutMS != rightOp.TimeoutMS ||
			leftOp.FailurePolicy != rightOp.FailurePolicy {
			return false
		}
	}
	return true
}

var _ identity.SessionPolicyEvaluateInvoker = (*IdentitySessionEvaluateInvoker)(nil)

package bootstrap

import (
	"context"
	"errors"
	"testing"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestSessionPolicyRenewalGateUnboundIsNoop(t *testing.T) {
	gate := &sessionPolicyRenewalGate{}
	if err := gate.Evaluate(context.Background(), 42); err != nil {
		t.Fatalf("unbound gate err=%v", err)
	}
}

func TestSessionPolicyRenewalGateUsesBoundEvaluator(t *testing.T) {
	store := &renewalGateTestStore{
		resolve: identity.IdentitySessionPolicyResolution{
			PolicyID: identity.IdentitySessionPolicyCoreDefault,
			Source:   identity.IdentitySessionPolicySourceCore,
			Selection: &identity.IdentitySessionPolicySelection{
				IdentitySessionPolicyEvidence: identity.IdentitySessionPolicyEvidence{
					PolicyID: identity.IdentitySessionPolicyCoreDefault,
				},
			},
			RegistryRevision: 1,
			RegistryDigest:   "core",
		},
	}
	evaluator, err := identity.NewSessionPolicyEvaluator(store, nil)
	if err != nil {
		t.Fatal(err)
	}
	gate := &sessionPolicyRenewalGate{}
	gate.Set(evaluator)
	if err := gate.Evaluate(context.Background(), 7); err != nil {
		t.Fatalf("core renew err=%v", err)
	}

	store.resolve = identity.IdentitySessionPolicyResolution{
		PolicyID: "plugin.session.policy",
		Source:   identity.IdentitySessionPolicySourcePlugin,
	}
	if err := gate.Evaluate(context.Background(), 7); !errors.Is(err, identity.ErrSessionPolicyEvaluationUnavailable) {
		t.Fatalf("plugin without provider err=%v", err)
	}
}

func TestNewSessionPolicyEvaluatorRejectsNilDeps(t *testing.T) {
	if _, err := newSessionPolicyEvaluator(nil, nil, nil); !errors.Is(err, identity.ErrIdentitySessionPolicyStoreUnavailable) {
		t.Fatalf("nil deps err=%v", err)
	}
}

type renewalGateTestStore struct {
	resolve identity.IdentitySessionPolicyResolution
}

func (s *renewalGateTestStore) Current(context.Context) (identity.IdentitySessionPolicySelection, error) {
	return identity.IdentitySessionPolicySelection{}, errors.New("unused")
}
func (s *renewalGateTestStore) Candidate(context.Context, string) (identity.IdentitySessionPolicyEvidence, error) {
	return identity.IdentitySessionPolicyEvidence{}, errors.New("unused")
}
func (s *renewalGateTestStore) Resolve(context.Context) (identity.IdentitySessionPolicyResolution, error) {
	return s.resolve, nil
}
func (s *renewalGateTestStore) Select(context.Context, identity.SelectIdentitySessionPolicyInput) (identity.IdentitySessionPolicyMutation, error) {
	return identity.IdentitySessionPolicyMutation{}, errors.New("unused")
}
func (s *renewalGateTestStore) Reset(context.Context, identity.ResetIdentitySessionPolicyInput) (identity.IdentitySessionPolicyMutation, error) {
	return identity.IdentitySessionPolicyMutation{}, errors.New("unused")
}
func (s *renewalGateTestStore) ListEvents(context.Context, int) ([]identity.IdentitySessionPolicyEvent, error) {
	return nil, errors.New("unused")
}

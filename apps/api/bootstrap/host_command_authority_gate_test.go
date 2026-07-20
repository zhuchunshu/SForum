package bootstrap

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

type recordingIdentityAuthorityGate struct {
	calls atomic.Int32
	err   error
}

func (g *recordingIdentityAuthorityGate) RunSessionPolicyMutation(
	ctx context.Context,
	mutation func() error,
) error {
	g.calls.Add(1)
	if g.err != nil {
		return g.err
	}
	if mutation == nil {
		return errors.New("missing mutation")
	}
	return mutation()
}

func TestDeferredIdentityAuthorityMutationGateBootAndAdopt(t *testing.T) {
	t.Parallel()

	gate := &deferredIdentityAuthorityMutationGate{}
	var ran atomic.Int32
	if err := gate.RunSessionPolicyMutation(t.Context(), func() error {
		ran.Add(1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if ran.Load() != 1 {
		t.Fatalf("unbound gate ran mutation %d times", ran.Load())
	}

	inner := &recordingIdentityAuthorityGate{}
	gate.Set(inner)
	if err := gate.RunSessionPolicyMutation(t.Context(), func() error {
		ran.Add(1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if inner.calls.Load() != 1 || ran.Load() != 2 {
		t.Fatalf("adopted gate calls=%d mutations=%d", inner.calls.Load(), ran.Load())
	}

	inner.err = errors.New("blocked")
	if err := gate.RunSessionPolicyMutation(t.Context(), func() error {
		t.Fatal("mutation must not run when gate fails")
		return nil
	}); !errors.Is(err, inner.err) {
		t.Fatalf("error=%v", err)
	}
}

func TestDeferredIdentityAuthorityMutationGateRejectsNilMutation(t *testing.T) {
	t.Parallel()
	gate := &deferredIdentityAuthorityMutationGate{}
	if err := gate.RunSessionPolicyMutation(t.Context(), nil); err == nil {
		t.Fatal("expected nil mutation error")
	}
}

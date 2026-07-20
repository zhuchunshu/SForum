package identity

import (
	"context"
	"errors"
	"testing"
)

type identityAuthorityMutationGateFunc func(context.Context, func() error) error

func (f identityAuthorityMutationGateFunc) RunSessionPolicyMutation(
	ctx context.Context,
	mutation func() error,
) error {
	return f(ctx, mutation)
}

func TestUpdateAdminUserConsultsAuthorityGateBeforePool(t *testing.T) {
	wantErr := errors.New("session effect still active")
	callbackCalls := 0
	store := (&PostgresStore{}).WithAuthorityMutationGate(identityAuthorityMutationGateFunc(
		func(context.Context, func() error) error { return wantErr },
	))
	_, err := store.UpdateAdminUser(t.Context(), 1, 2, AdminUpdateUserInput{})
	if !errors.Is(err, wantErr) || callbackCalls != 0 {
		t.Fatalf("callbackCalls=%d err=%v", callbackCalls, err)
	}

	store.authorityMutationGate = identityAuthorityMutationGateFunc(
		func(_ context.Context, mutation func() error) error {
			callbackCalls++
			return mutation()
		},
	)
	callbackErr := errors.New("mutation failed")
	err = store.runIdentityAuthorityMutation(t.Context(), func() error { return callbackErr })
	if !errors.Is(err, callbackErr) || callbackCalls != 1 {
		t.Fatalf("callbackCalls=%d err=%v", callbackCalls, err)
	}
}

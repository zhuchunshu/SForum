package identityregistry

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type sessionPolicyMutationGateFunc func(context.Context, func() error) error

func (f sessionPolicyMutationGateFunc) RunSessionPolicyMutation(ctx context.Context, run func() error) error {
	return f(ctx, run)
}

func TestSessionPolicyMutationGateRequiresOneSynchronousCallback(t *testing.T) {
	wantState := DurableState{Tips: []DurableDeclarationTip{{StableID: "fixture.identity.session"}}}
	t.Run("exactly once", func(t *testing.T) {
		calls := 0
		state, err := runSessionPolicyMutationGate(
			t.Context(),
			sessionPolicyMutationGateFunc(func(_ context.Context, run func() error) error { return run() }),
			func() (DurableState, error) { calls++; return wantState, nil },
		)
		if err != nil || calls != 1 || len(state.Tips) != 1 || state.Tips[0].StableID != wantState.Tips[0].StableID {
			t.Fatalf("state=%#v calls=%d err=%v", state, calls, err)
		}
	})

	t.Run("zero call", func(t *testing.T) {
		calls := 0
		_, err := runSessionPolicyMutationGate(
			t.Context(),
			sessionPolicyMutationGateFunc(func(context.Context, func() error) error { return nil }),
			func() (DurableState, error) { calls++; return wantState, nil },
		)
		if !errors.Is(err, errIdentityRegistryStore) || calls != 0 {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	})

	t.Run("deny without callback", func(t *testing.T) {
		wantErr := context.Canceled
		_, err := runSessionPolicyMutationGate(
			t.Context(),
			sessionPolicyMutationGateFunc(func(context.Context, func() error) error { return wantErr }),
			func() (DurableState, error) { t.Fatal("denied gate ran callback"); return DurableState{}, nil },
		)
		if !errors.Is(err, wantErr) {
			t.Fatalf("deny error = %v", err)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		var calls atomic.Int32
		var duplicateErr error
		_, err := runSessionPolicyMutationGate(
			t.Context(),
			sessionPolicyMutationGateFunc(func(_ context.Context, run func() error) error {
				if err := run(); err != nil {
					return err
				}
				duplicateErr = run()
				return nil
			}),
			func() (DurableState, error) { calls.Add(1); return wantState, nil },
		)
		if err != nil || !errors.Is(duplicateErr, errIdentityRegistryStore) || calls.Load() != 1 {
			t.Fatalf("calls=%d duplicate=%v err=%v", calls.Load(), duplicateErr, err)
		}
	})

	t.Run("callback error wins", func(t *testing.T) {
		callbackErr := errors.New("reconcile failed")
		gateErr := errors.New("gate failed")
		_, err := runSessionPolicyMutationGate(
			t.Context(),
			sessionPolicyMutationGateFunc(func(_ context.Context, run func() error) error { _ = run(); return gateErr }),
			func() (DurableState, error) { return wantState, callbackErr },
		)
		if !errors.Is(err, callbackErr) || errors.Is(err, gateErr) {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestSessionPolicyMutationGateRejectsEscapedAndInflightCallbacks(t *testing.T) {
	t.Run("late", func(t *testing.T) {
		var captured func() error
		var calls atomic.Int32
		_, err := runSessionPolicyMutationGate(
			t.Context(),
			sessionPolicyMutationGateFunc(func(_ context.Context, run func() error) error { captured = run; return nil }),
			func() (DurableState, error) { calls.Add(1); return DurableState{}, nil },
		)
		if !errors.Is(err, errIdentityRegistryStore) || captured == nil {
			t.Fatalf("outer error = %v", err)
		}
		if lateErr := captured(); !errors.Is(lateErr, errIdentityRegistryStore) || calls.Load() != 0 {
			t.Fatalf("late calls=%d err=%v", calls.Load(), lateErr)
		}
	})

	t.Run("inflight", func(t *testing.T) {
		entered := make(chan struct{})
		release := make(chan struct{})
		var releaseOnce sync.Once
		releaseCallback := func() { releaseOnce.Do(func() { close(release) }) }
		t.Cleanup(releaseCallback)
		result := make(chan error, 1)
		go func() {
			_, err := runSessionPolicyMutationGate(
				t.Context(),
				sessionPolicyMutationGateFunc(func(_ context.Context, run func() error) error {
					go func() { _ = run() }()
					select {
					case <-entered:
						return nil
					case <-time.After(5 * time.Second):
						return errors.New("callback did not start")
					}
				}),
				func() (DurableState, error) {
					close(entered)
					<-release
					return DurableState{}, nil
				},
			)
			result <- err
		}()
		awaitMutationGateSignal(t, entered, "inflight callback")
		select {
		case err := <-result:
			t.Fatalf("gate returned before inflight callback drained: %v", err)
		default:
		}
		releaseCallback()
		if err := awaitMutationGateResult(t, result); err != nil {
			t.Fatalf("inflight error = %v", err)
		}
	})
}

func TestSessionPolicyMutationGatePreservesPanics(t *testing.T) {
	panicValue := "session policy mutation panic"
	t.Run("callback panic swallowed by gate", func(t *testing.T) {
		var callbackErr error
		defer func() {
			if recovered := recover(); recovered != panicValue {
				t.Fatalf("recovered panic = %#v", recovered)
			}
			if !errors.Is(callbackErr, errIdentityRegistryStore) {
				t.Fatalf("callback err = %v", callbackErr)
			}
		}()
		_, _ = runSessionPolicyMutationGate(
			t.Context(),
			sessionPolicyMutationGateFunc(func(_ context.Context, run func() error) error {
				callbackErr = run()
				return nil
			}),
			func() (DurableState, error) { panic(panicValue) },
		)
	})

	t.Run("gate panic", func(t *testing.T) {
		defer func() {
			if recovered := recover(); recovered != panicValue {
				t.Fatalf("recovered panic = %#v", recovered)
			}
		}()
		_, _ = runSessionPolicyMutationGate(
			t.Context(),
			sessionPolicyMutationGateFunc(func(context.Context, func() error) error { panic(panicValue) }),
			func() (DurableState, error) { t.Fatal("gate panic ran callback"); return DurableState{}, nil },
		)
	})
}

func TestSessionPolicyMutationGateTransfersAsyncCallbackPanic(t *testing.T) {
	panicValue := "async session policy mutation panic"
	entered := make(chan struct{})
	defer func() {
		if recovered := recover(); recovered != panicValue {
			t.Fatalf("recovered panic = %#v", recovered)
		}
	}()
	_, _ = runSessionPolicyMutationGate(
		t.Context(),
		sessionPolicyMutationGateFunc(func(_ context.Context, run func() error) error {
			go func() { _ = run() }()
			<-entered
			return nil
		}),
		func() (DurableState, error) {
			close(entered)
			panic(panicValue)
		},
	)
}

func TestSessionPolicyMutationGateRejectsCanceledAdmission(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var calls atomic.Int32
	_, err := runSessionPolicyMutationGate(
		ctx,
		sessionPolicyMutationGateFunc(func(_ context.Context, run func() error) error { return run() }),
		func() (DurableState, error) {
			calls.Add(1)
			return DurableState{}, nil
		},
	)
	if !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatalf("calls=%d err=%v", calls.Load(), err)
	}
}

func awaitMutationGateSignal(t *testing.T, signal <-chan struct{}, label string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", label)
	}
}

func awaitMutationGateResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for mutation gate result")
		return nil
	}
}

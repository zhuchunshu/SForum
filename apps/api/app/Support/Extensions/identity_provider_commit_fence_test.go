package extensionsruntime

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestIdentityProviderCommitFenceRejectsInflightCompletionAfterClose(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	state := newIdentityProviderCommitFenceState(func() error {
		close(entered)
		<-release
		return nil
	})
	fenceResult := make(chan error, 1)
	go func() { fenceResult <- state.Run() }()
	<-entered
	if !state.closeAcceptance() {
		t.Fatal("inflight fence was not marked started")
	}
	releaseOnce.Do(func() { close(release) })
	if err := <-fenceResult; !errors.Is(err, ErrIdentityProviderInvocationInvalid) {
		t.Fatalf("inflight fence err=%v", err)
	}
	outcome := state.Finish()
	if !errors.Is(outcome.result, ErrIdentityProviderInvocationInvalid) ||
		!errors.Is(outcome.violation, ErrIdentityProviderInvocationInvalid) {
		t.Fatalf("inflight outcome=%#v", outcome)
	}
}

func TestIdentityProviderCommitFenceClosesAtAcceptUnwindBoundary(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	state := newIdentityProviderCommitFenceState(func() error {
		close(entered)
		<-release
		return nil
	})
	fenceResult := make(chan error, 1)
	acceptErr, acceptPanic := runIdentityProviderAccept(
		func(_ context.Context, _ IdentityProviderInvocationResult, fence IdentityProviderCommitFence) error {
			go func() { fenceResult <- fence() }()
			<-entered
			return nil
		},
		context.Background(), IdentityProviderInvocationResult{}, state,
	)
	if acceptErr != nil || acceptPanic != nil {
		t.Fatalf("accept err=%v panic=%#v", acceptErr, acceptPanic)
	}
	releaseOnce.Do(func() { close(release) })
	if err := <-fenceResult; !errors.Is(err, ErrIdentityProviderInvocationInvalid) {
		t.Fatalf("boundary fence err=%v", err)
	}
	outcome := state.Finish()
	if !errors.Is(outcome.result, ErrIdentityProviderInvocationInvalid) ||
		!errors.Is(outcome.violation, ErrIdentityProviderInvocationInvalid) {
		t.Fatalf("boundary outcome=%#v", outcome)
	}
}

func TestIdentityProviderCommitFenceConcurrentDuplicateExecutesOnce(t *testing.T) {
	const calls = 32
	var validationCalls atomic.Int32
	state := newIdentityProviderCommitFenceState(func() error {
		validationCalls.Add(1)
		return nil
	})
	start := make(chan struct{})
	results := make(chan error, calls)
	for range calls {
		go func() {
			<-start
			results <- state.Run()
		}()
	}
	close(start)
	successes := 0
	invalid := 0
	for range calls {
		err := <-results
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrIdentityProviderInvocationInvalid):
			invalid++
		default:
			t.Fatalf("duplicate fence err=%v", err)
		}
	}
	state.closeAcceptance()
	outcome := state.Finish()
	if successes != 1 || invalid != calls-1 || validationCalls.Load() != 1 ||
		!errors.Is(outcome.violation, ErrIdentityProviderInvocationInvalid) {
		t.Fatalf(
			"success=%d invalid=%d validation=%d outcome=%#v",
			successes, invalid, validationCalls.Load(), outcome,
		)
	}
}

func TestIdentityProviderCommitFenceEscapedAndPanicPathsClose(t *testing.T) {
	t.Run("escaped", func(t *testing.T) {
		state := newIdentityProviderCommitFenceState(func() error { return nil })
		state.closeAcceptance()
		outcome := state.Finish()
		if !errors.Is(outcome.violation, ErrIdentityProviderInvocationInvalid) ||
			!errors.Is(state.Run(), ErrIdentityProviderInvocationInvalid) {
			t.Fatalf("escaped outcome=%#v", outcome)
		}
	})

	t.Run("panic", func(t *testing.T) {
		panicValue := &struct{ label string }{label: "fence panic"}
		state := newIdentityProviderCommitFenceState(func() error { panic(panicValue) })
		if err := state.Run(); !errors.Is(err, ErrIdentityProviderInvocationInvalid) {
			t.Fatalf("panic fence err=%v", err)
		}
		state.closeAcceptance()
		outcome := state.Finish()
		if outcome.panicValue != panicValue || !errors.Is(outcome.result, ErrIdentityProviderInvocationInvalid) {
			t.Fatalf("outcome=%#v", outcome)
		}
	})
}

func TestIdentityProviderCommitFenceTransfersAsyncValidationPanic(t *testing.T) {
	panicValue := &struct{ label string }{label: "async fence panic"}
	state := newIdentityProviderCommitFenceState(func() error { panic(panicValue) })
	fenceResult := make(chan error, 1)
	acceptErr, acceptPanic := runIdentityProviderAccept(
		func(_ context.Context, _ IdentityProviderInvocationResult, fence IdentityProviderCommitFence) error {
			go func() { fenceResult <- fence() }()
			return <-fenceResult
		},
		context.Background(), IdentityProviderInvocationResult{}, state,
	)
	outcome := state.Finish()
	if acceptPanic != nil || !errors.Is(acceptErr, ErrIdentityProviderInvocationInvalid) ||
		outcome.panicValue != panicValue || !errors.Is(outcome.result, ErrIdentityProviderInvocationInvalid) {
		t.Fatalf("accept err=%v panic=%#v outcome=%#v", acceptErr, acceptPanic, outcome)
	}
}

func TestManagerIdentityProviderCancellationBeforeFenceFailsClosed(t *testing.T) {
	testCases := []struct {
		name   string
		cancel func(managerIdentityTransportFixture, context.CancelCauseFunc, error) error
	}{
		{
			name: "caller cancellation",
			cancel: func(_ managerIdentityTransportFixture, cancel context.CancelCauseFunc, cause error) error {
				cancel(cause)
				return nil
			},
		},
		{
			name: "force drain",
			cancel: func(fixture managerIdentityTransportFixture, _ context.CancelCauseFunc, cause error) error {
				_, err := fixture.manager.ForceDrain(RuntimeInstanceIdentity{
					ExtensionID: fixture.extension.ID,
					InstanceID:  fixture.publication.Artifact.RuntimeInstanceID,
				}, cause)
				return err
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newManagerIdentityTransportFixture(t, strings.Repeat("a", 64))
			ctx, cancel := context.WithCancelCause(t.Context())
			t.Cleanup(func() { cancel(nil) })
			cause := errors.New(testCase.name)
			committed := false
			_, err := fixture.runtime.Invoke(
				ctx, fixture.invocation(),
				func(_ context.Context, _ IdentityProviderInvocationResult, fence IdentityProviderCommitFence) error {
					if cancelErr := testCase.cancel(fixture, cancel, cause); cancelErr != nil {
						return cancelErr
					}
					if fenceErr := fence(); fenceErr == nil {
						committed = true
					}
					return nil
				},
			)
			if !errors.Is(err, ErrIdentityProviderAcceptFailed) || !errors.Is(err, cause) || committed {
				t.Fatalf("committed=%t err=%v", committed, err)
			}
			snapshot, inspectErr := fixture.manager.ActiveRuntimeInstance(fixture.extension.ID)
			if inspectErr != nil || snapshot.Admission.ActiveTotal != 0 {
				t.Fatalf("snapshot=%#v err=%v", snapshot, inspectErr)
			}
		})
	}
}

func TestManagerIdentityProviderAcceptPanicPreservesPanicAndReleasesLease(t *testing.T) {
	for _, afterFence := range []bool{false, true} {
		t.Run(map[bool]string{false: "before fence", true: "after fence"}[afterFence], func(t *testing.T) {
			fixture := newManagerIdentityTransportFixture(t, strings.Repeat("a", 64))
			panicValue := &struct{ afterFence bool }{afterFence: afterFence}
			var recovered any
			func() {
				defer func() { recovered = recover() }()
				_, _ = fixture.runtime.Invoke(
					t.Context(), fixture.invocation(),
					func(_ context.Context, _ IdentityProviderInvocationResult, fence IdentityProviderCommitFence) error {
						if afterFence {
							if err := fence(); err != nil {
								return err
							}
						}
						panic(panicValue)
					},
				)
			}()
			if recovered != panicValue {
				t.Fatalf("recovered=%#v", recovered)
			}
			snapshot, err := fixture.manager.ActiveRuntimeInstance(fixture.extension.ID)
			if err != nil || snapshot.Admission.ActiveTotal != 0 {
				t.Fatalf("snapshot=%#v err=%v", snapshot, err)
			}
		})
	}
}

func TestManagerIdentityProviderTransfersAsyncFencePanicAndReleasesLease(t *testing.T) {
	fixture := newManagerIdentityTransportFixture(t, strings.Repeat("a", 64))
	identity := RuntimeInstanceIdentity{
		ExtensionID: fixture.extension.ID,
		InstanceID:  fixture.publication.Artifact.RuntimeInstanceID,
	}
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		_, _ = fixture.runtime.Invoke(
			t.Context(), fixture.invocation(),
			func(_ context.Context, _ IdentityProviderInvocationResult, fence IdentityProviderCommitFence) error {
				manager := fixture.runtime.manager
				fixture.runtime.manager = nil
				fenceResult := make(chan error, 1)
				go func() { fenceResult <- fence() }()
				err := <-fenceResult
				fixture.runtime.manager = manager
				return err
			},
		)
	}()
	if recovered == nil {
		t.Fatal("async fence validation panic was not transferred")
	}
	snapshot, err := fixture.manager.InspectRuntimeInstance(identity)
	if err != nil || snapshot.Admission.ActiveTotal != 0 {
		t.Fatalf("snapshot=%#v err=%v", snapshot, err)
	}
}

func TestManagerIdentityProviderAuthorityDriftAfterFenceKeepsTerminalSuccess(t *testing.T) {
	fixture := newManagerIdentityTransportFixture(t, strings.Repeat("a", 64))
	cause := errors.New("post-fence drain")
	committed := false
	result, err := fixture.runtime.Invoke(
		t.Context(), fixture.invocation(),
		func(_ context.Context, _ IdentityProviderInvocationResult, fence IdentityProviderCommitFence) error {
			if err := fence(); err != nil {
				return err
			}
			if _, err := fixture.manager.ForceDrain(RuntimeInstanceIdentity{
				ExtensionID: fixture.extension.ID,
				InstanceID:  fixture.publication.Artifact.RuntimeInstanceID,
			}, cause); err != nil {
				return err
			}
			if _, removed, err := fixture.registry.Remove(fixture.publication.Artifact); err != nil || !removed {
				return errors.Join(ErrIdentityProviderStale, err)
			}
			committed = true
			return nil
		},
	)
	if err != nil || !committed || result.Provider.ID != fixture.provider.ID {
		t.Fatalf("result=%#v committed=%t err=%v", result, committed, err)
	}
}

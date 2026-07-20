package authsession

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunRenewalEffectGateContractMatrix(t *testing.T) {
	t.Parallel()

	effectFailure := errors.New("effect failed")
	gateFailure := errors.New("gate failed")
	tests := []struct {
		name        string
		gate        RenewalEffectGate
		effect      RenewalEffect
		wantEffect  int32
		wantReject  bool
		wantFailure error
	}{
		{
			name: "exactly once",
			gate: func(ctx context.Context, _ int64, _ int64, effect RenewalEffect) error {
				return effect(ctx)
			},
			wantEffect: 1,
		},
		{
			name: "missing callback",
			gate: func(context.Context, int64, int64, RenewalEffect) error {
				return nil
			},
			wantReject: true,
		},
		{
			name: "duplicate callback",
			gate: func(ctx context.Context, _ int64, _ int64, effect RenewalEffect) error {
				_ = effect(ctx)
				_ = effect(ctx)
				return nil
			},
			wantEffect: 1,
		},
		{
			name: "effect error is preserved",
			gate: func(ctx context.Context, _ int64, _ int64, effect RenewalEffect) error {
				_ = effect(ctx)
				return nil
			},
			effect:      func(context.Context) error { return effectFailure },
			wantEffect:  1,
			wantFailure: effectFailure,
		},
		{
			name: "effect error precedes gate error",
			gate: func(ctx context.Context, _ int64, _ int64, effect RenewalEffect) error {
				_ = effect(ctx)
				return gateFailure
			},
			effect:      func(context.Context) error { return effectFailure },
			wantEffect:  1,
			wantFailure: effectFailure,
		},
		{
			name: "gate error rejects",
			gate: func(ctx context.Context, _ int64, _ int64, effect RenewalEffect) error {
				if err := effect(ctx); err != nil {
					return err
				}
				return gateFailure
			},
			wantEffect: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var effectCalls atomic.Int32
			effect := tt.effect
			if effect == nil {
				effect = func(context.Context) error { return nil }
			}
			manager := &Manager{renewalEffectGate: tt.gate}
			err := manager.runRenewalEffectGate(context.Background(), 42, 0, func(ctx context.Context) error {
				effectCalls.Add(1)
				return effect(ctx)
			})
			if effectCalls.Load() != tt.wantEffect {
				t.Fatalf("effect calls=%d want=%d", effectCalls.Load(), tt.wantEffect)
			}
			if tt.wantReject != errors.Is(err, ErrRenewalRejected) {
				t.Fatalf("err=%v rejected=%t", err, errors.Is(err, ErrRenewalRejected))
			}
			if tt.wantFailure != nil && !errors.Is(err, tt.wantFailure) {
				t.Fatalf("err=%v want failure=%v", err, tt.wantFailure)
			}
		})
	}
}

func TestRunRenewalEffectGateConcurrentCallbackExecutesAtMostOnce(t *testing.T) {
	t.Parallel()

	var effectCalls atomic.Int32
	manager := &Manager{renewalEffectGate: func(ctx context.Context, _ int64, _ int64, effect RenewalEffect) error {
		start := make(chan struct{})
		results := make(chan error, 2)
		for range 2 {
			go func() {
				<-start
				results <- effect(ctx)
			}()
		}
		close(start)
		<-results
		<-results
		return nil
	}}
	err := manager.runRenewalEffectGate(context.Background(), 42, 0, func(context.Context) error {
		effectCalls.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if effectCalls.Load() != 1 {
		t.Fatalf("effect calls=%d", effectCalls.Load())
	}
}

func TestRunRenewalEffectGateRejectsEscapedCallback(t *testing.T) {
	t.Parallel()

	escaped := make(chan RenewalEffect, 1)
	manager := &Manager{renewalEffectGate: func(_ context.Context, _ int64, _ int64, effect RenewalEffect) error {
		escaped <- effect
		return nil
	}}
	var effectCalls atomic.Int32
	err := manager.runRenewalEffectGate(context.Background(), 42, 0, func(context.Context) error {
		effectCalls.Add(1)
		return nil
	})
	if !errors.Is(err, ErrRenewalRejected) {
		t.Fatalf("gate err=%v", err)
	}
	if callbackErr := (<-escaped)(context.Background()); !errors.Is(callbackErr, ErrRenewalRejected) {
		t.Fatalf("escaped callback err=%v", callbackErr)
	}
	if effectCalls.Load() != 0 {
		t.Fatalf("effect calls=%d", effectCalls.Load())
	}
}

func TestRunRenewalEffectGateWaitsForInflightCallback(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	release := make(chan struct{})
	callbackDone := make(chan error, 1)
	var releaseOnce sync.Once
	releaseEffect := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseEffect)

	manager := &Manager{renewalEffectGate: func(ctx context.Context, _ int64, _ int64, effect RenewalEffect) error {
		go func() { callbackDone <- effect(ctx) }()
		<-started
		return nil
	}}
	result := make(chan error, 1)
	go func() {
		result <- manager.runRenewalEffectGate(context.Background(), 42, 0, func(context.Context) error {
			close(started)
			<-release
			return nil
		})
	}()
	select {
	case err := <-result:
		t.Fatalf("gate returned before inflight callback completed: %v", err)
	case <-started:
	}
	releaseEffect()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("gate result=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("gate did not return after inflight callback completed")
	}
	select {
	case callbackErr := <-callbackDone:
		if callbackErr != nil {
			t.Fatalf("callback err=%v", callbackErr)
		}
	case <-time.After(time.Second):
		t.Fatal("callback did not finish after release")
	}
}

func TestRunRenewalEffectGatePropagatesAcceptedContext(t *testing.T) {
	t.Parallel()

	type contextKey struct{}
	accepted := context.WithValue(context.Background(), contextKey{}, "accepted")
	manager := &Manager{renewalEffectGate: func(_ context.Context, _ int64, _ int64, effect RenewalEffect) error {
		return effect(accepted)
	}}
	err := manager.runRenewalEffectGate(context.Background(), 42, 0, func(ctx context.Context) error {
		if got := ctx.Value(contextKey{}); got != "accepted" {
			t.Fatalf("effect context marker=%v", got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunRenewalEffectGatePreservesCallbackPanic(t *testing.T) {
	panicValue := "renewal effect panic"
	var callbackErr error
	manager := &Manager{renewalEffectGate: func(ctx context.Context, _ int64, _ int64, effect RenewalEffect) error {
		callbackErr = effect(ctx)
		return nil
	}}
	defer func() {
		if recovered := recover(); recovered != panicValue {
			t.Fatalf("recovered=%#v", recovered)
		}
		if !errors.Is(callbackErr, ErrRenewalRejected) {
			t.Fatalf("callback err=%v", callbackErr)
		}
	}()
	_ = manager.runRenewalEffectGate(context.Background(), 42, 0, func(context.Context) error {
		panic(panicValue)
	})
}

func TestRunRenewalEffectGateTransfersAsyncCallbackPanic(t *testing.T) {
	panicValue := "async renewal callback panic"
	entered := make(chan struct{})
	manager := &Manager{renewalEffectGate: func(_ context.Context, _ int64, _ int64, effect RenewalEffect) error {
		go func() { _ = effect(context.Background()) }()
		<-entered
		return nil
	}}
	defer func() {
		if recovered := recover(); recovered != panicValue {
			t.Fatalf("recovered=%#v", recovered)
		}
	}()
	_ = manager.runRenewalEffectGate(t.Context(), 42, 0, func(context.Context) error {
		close(entered)
		panic(panicValue)
	})
}

func TestRunRenewalEffectGateRejectsCanceledAdmission(t *testing.T) {
	var calls atomic.Int32
	manager := &Manager{renewalEffectGate: func(_ context.Context, _ int64, _ int64, effect RenewalEffect) error {
		return effect(context.Background())
	}}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := manager.runRenewalEffectGate(ctx, 42, 0, func(context.Context) error {
		calls.Add(1)
		return nil
	}); !errors.Is(err, ErrRenewalRejected) || calls.Load() != 0 {
		t.Fatalf("calls=%d err=%v", calls.Load(), err)
	}
}

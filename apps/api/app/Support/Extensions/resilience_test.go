package extensionsruntime

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGateOpensCircuitAfterFailures(t *testing.T) {
	hub := newResilienceHub(ResilienceConfig{
		MaxConcurrent:    2,
		FailureThreshold: 3,
		CircuitOpenFor:   time.Minute,
	})

	for i := 0; i < 3; i++ {
		release, reason := hub.tryEnter(context.Background(), "demo.plugin")
		if reason != "" {
			t.Fatalf("enter %d rejected: %s", i, reason)
		}
		release(false, "extension.hook_failed")
	}

	_, reason := hub.tryEnter(context.Background(), "demo.plugin")
	if reason != "extension.circuit_open" {
		t.Fatalf("expected circuit open, got %q", reason)
	}
	snap := hub.snapshot("demo.plugin")
	if !snap.CircuitOpen || snap.ConsecutiveFailures < 3 {
		t.Fatalf("snapshot: %#v", snap)
	}
}

func TestGateSuccessResetsFailures(t *testing.T) {
	hub := newResilienceHub(ResilienceConfig{FailureThreshold: 3, CircuitOpenFor: time.Minute})
	for i := 0; i < 2; i++ {
		release, _ := hub.tryEnter(context.Background(), "demo.plugin")
		release(false, "boom")
	}
	release, _ := hub.tryEnter(context.Background(), "demo.plugin")
	release(true, "")
	if hub.snapshot("demo.plugin").ConsecutiveFailures != 0 {
		t.Fatalf("expected reset, got %#v", hub.snapshot("demo.plugin"))
	}
}

func TestGateConcurrencyLimit(t *testing.T) {
	hub := newResilienceHub(ResilienceConfig{MaxConcurrent: 1, FailureThreshold: 100})
	release1, reason := hub.tryEnter(context.Background(), "demo.plugin")
	if reason != "" {
		t.Fatalf("first enter: %s", reason)
	}
	defer release1(true, "")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, reason = hub.tryEnter(ctx, "demo.plugin")
	if reason != "extension.hook_timeout" {
		t.Fatalf("expected timeout waiting for slot, got %q", reason)
	}
}

func TestGateHalfOpenAfterCooldown(t *testing.T) {
	hub := newResilienceHub(ResilienceConfig{
		FailureThreshold: 1,
		CircuitOpenFor:   20 * time.Millisecond,
	})
	release, _ := hub.tryEnter(context.Background(), "demo.plugin")
	release(false, "boom")
	if _, reason := hub.tryEnter(context.Background(), "demo.plugin"); reason != "extension.circuit_open" {
		t.Fatalf("expected open, got %q", reason)
	}
	time.Sleep(30 * time.Millisecond)
	release2, reason := hub.tryEnter(context.Background(), "demo.plugin")
	if reason != "" {
		t.Fatalf("expected half-open allow, got %q", reason)
	}
	release2(true, "")
}

func TestGateConcurrentEnter(t *testing.T) {
	hub := newResilienceHub(ResilienceConfig{MaxConcurrent: 4, FailureThreshold: 100})
	var wg sync.WaitGroup
	var okCount atomic.Int32
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			release, reason := hub.tryEnter(ctx, "demo.plugin")
			if reason != "" {
				return
			}
			okCount.Add(1)
			time.Sleep(20 * time.Millisecond)
			release(true, "")
		}()
	}
	wg.Wait()
	if okCount.Load() < 4 {
		t.Fatalf("expected at least 4 successful enters, got %d", okCount.Load())
	}
}

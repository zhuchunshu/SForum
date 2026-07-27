package extensionruntime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRuntimeAdmissionGateDrainBoundaryAndCleanupExemption(t *testing.T) {
	gate := newRuntimeAdmissionTestGate(t)
	route := acquireRuntimeAdmission(t, gate, RuntimeCallRoute)
	host := acquireRuntimeAdmission(t, gate, RuntimeCallHost)

	snapshot := gate.BeginDrain()
	if !snapshot.Draining || snapshot.Forced || snapshot.ActiveTotal != 2 ||
		snapshot.ActiveByClass[RuntimeCallRoute] != 1 || snapshot.ActiveByClass[RuntimeCallHost] != 1 {
		t.Fatalf("drain snapshot = %#v", snapshot)
	}
	for _, class := range []RuntimeCallClass{RuntimeCallRoute, RuntimeCallHost, RuntimeCallJob, "future_ordinary"} {
		if _, err := gate.Acquire(context.Background(), class); !errors.Is(err, ErrRuntimeAdmissionDraining) {
			t.Fatalf("class %q drain error = %v", class, err)
		}
	}

	cleanup := acquireRuntimeAdmission(t, gate, RuntimeCallLifecycleCleanup)
	if got := gate.Snapshot(); got.ActiveTotal != 3 || got.ActiveByClass[RuntimeCallLifecycleCleanup] != 1 {
		t.Fatalf("cleanup snapshot = %#v", got)
	}

	route.Release()
	route.Release()
	host.Release()
	cleanup.Release()
	cleanup.Release()
	if err := gate.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := gate.Snapshot(); got.ActiveTotal != 0 || !got.Draining || got.Forced {
		t.Fatalf("idle drain snapshot = %#v", got)
	}
}

func TestRuntimeAdmissionGateWaitHonorsContextWithoutDroppingInflight(t *testing.T) {
	gate := newRuntimeAdmissionTestGate(t)
	lease := acquireRuntimeAdmission(t, gate, RuntimeCallJob)
	gate.BeginDrain()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := gate.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wait error = %v", err)
	}
	if got := gate.Snapshot(); got.ActiveTotal != 1 || got.ActiveByClass[RuntimeCallJob] != 1 {
		t.Fatalf("timeout changed inflight state: %#v", got)
	}

	lease.Release()
	if err := gate.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeAdmissionGateAcquirePreservesCustomParentCause(t *testing.T) {
	gate := newRuntimeAdmissionTestGate(t)
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("caller cancelled with domain cause")
	cancel(cause)
	if _, err := gate.Acquire(ctx, RuntimeCallProvider); !errors.Is(err, cause) || errors.Is(err, context.Canceled) {
		t.Fatalf("custom parent cancellation error=%v", err)
	}
}

func TestRuntimeAdmissionGateForceCancelsInflightAndClosesCleanup(t *testing.T) {
	gate := newRuntimeAdmissionTestGate(t)
	route := acquireRuntimeAdmission(t, gate, RuntimeCallRoute)
	gate.BeginDrain()
	cleanup := acquireRuntimeAdmission(t, gate, RuntimeCallLifecycleCleanup)
	cause := errors.New("forced uninstall timeout")

	snapshot := gate.ForceCancel(cause)
	if !snapshot.Draining || !snapshot.Forced || !errors.Is(snapshot.ForceCause, cause) || snapshot.ActiveTotal != 2 {
		t.Fatalf("forced snapshot = %#v", snapshot)
	}
	for _, lease := range []*RuntimeAdmissionLease{route, cleanup} {
		select {
		case <-lease.Context.Done():
		case <-time.After(time.Second):
			t.Fatal("forced call context was not cancelled")
		}
		if !errors.Is(context.Cause(lease.Context), cause) ||
			!errors.Is(context.Cause(lease.Context), ErrRuntimeAdmissionForced) {
			t.Fatalf("context cause = %v", context.Cause(lease.Context))
		}
	}
	if _, err := gate.Acquire(context.Background(), RuntimeCallLifecycleCleanup); !errors.Is(err, ErrRuntimeAdmissionForced) || !errors.Is(err, cause) {
		t.Fatalf("post-force cleanup error = %v", err)
	}

	// ForceCancel 只发取消信号，调用未 Release 前 Wait 仍必须超时。
	waitCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := gate.Wait(waitCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("forced wait error = %v", err)
	}
	route.Release()
	cleanup.Release()
	if err := gate.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}

	second := gate.ForceCancel(errors.New("later cause"))
	if !errors.Is(second.ForceCause, cause) {
		t.Fatalf("repeated force replaced first cause: %v", second.ForceCause)
	}
}

func TestRuntimeAdmissionGateDoesNotPublishForceBeforeLeaseCancellation(t *testing.T) {
	gate := newRuntimeAdmissionTestGate(t)
	lease := acquireRuntimeAdmission(t, gate, RuntimeCallProvider)

	entered := make(chan struct{})
	resume := make(chan struct{})
	gate.mu.Lock()
	call := gate.active[lease.id]
	originalCancel := call.cancel
	call.cancel = func(cause error) {
		close(entered)
		<-resume
		originalCancel(cause)
	}
	gate.active[lease.id] = call
	gate.mu.Unlock()

	forceCause := errors.New("linearized ForceDrain")
	forced := make(chan RuntimeAdmissionSnapshot, 1)
	go func() { forced <- gate.ForceCancel(forceCause) }()
	<-entered
	forceVisibleBeforeCancel := gate.mu.TryLock()
	if forceVisibleBeforeCancel {
		gate.mu.Unlock()
	}
	close(resume)
	snapshot := <-forced
	if forceVisibleBeforeCancel {
		t.Fatal("forced state became visible before the existing lease was cancelled")
	}
	if !snapshot.Forced || !errors.Is(context.Cause(lease.Context), ErrRuntimeAdmissionForced) ||
		!errors.Is(context.Cause(lease.Context), forceCause) {
		t.Fatalf("forced snapshot=%#v cause=%v", snapshot, context.Cause(lease.Context))
	}
	lease.Release()
}

func TestRuntimeAdmissionGateConcurrentAcquireAndDrainHasExactBoundary(t *testing.T) {
	gate := newRuntimeAdmissionTestGate(t)
	const callers = 128
	start := make(chan struct{})
	results := make(chan runtimeAdmissionAcquireResult, callers)
	var wait sync.WaitGroup
	for index := range callers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			lease, err := gate.Acquire(context.Background(), RuntimeCallClass(fmt.Sprintf("ordinary-%d", index%4)))
			results <- runtimeAdmissionAcquireResult{lease: lease, err: err}
		}()
	}
	close(start)
	gate.BeginDrain()
	wait.Wait()
	close(results)

	leases := make([]*RuntimeAdmissionLease, 0, callers)
	for result := range results {
		switch {
		case result.err == nil:
			leases = append(leases, result.lease)
		case errors.Is(result.err, ErrRuntimeAdmissionDraining):
		default:
			t.Fatalf("concurrent acquire error = %v", result.err)
		}
	}
	if got := gate.Snapshot(); got.ActiveTotal != len(leases) {
		t.Fatalf("active=%d successful=%d", got.ActiveTotal, len(leases))
	}
	// BeginDrain 返回后启动的 ordinary 调用必须全部落在关闭边界之后。
	for range callers {
		if _, err := gate.Acquire(context.Background(), RuntimeCallRoute); !errors.Is(err, ErrRuntimeAdmissionDraining) {
			t.Fatalf("post-drain acquire error = %v", err)
		}
	}

	var releases sync.WaitGroup
	for _, lease := range leases {
		releases.Add(1)
		go func() {
			defer releases.Done()
			lease.Release()
			lease.Release()
		}()
	}
	releases.Wait()
	if err := gate.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeAdmissionGateValidatesAndPreservesInstanceIdentity(t *testing.T) {
	if _, err := NewRuntimeAdmissionGate(RuntimeInstanceIdentity{ExtensionID: "demo.plugin"}); !errors.Is(err, ErrRuntimeAdmissionInvalid) {
		t.Fatalf("missing instance error = %v", err)
	}
	gate, err := NewRuntimeAdmissionGate(RuntimeInstanceIdentity{ExtensionID: " demo.plugin ", InstanceID: " runtime-1 "})
	if err != nil {
		t.Fatal(err)
	}
	if got := gate.Identity(); got.ExtensionID != "demo.plugin" || got.InstanceID != "runtime-1" {
		t.Fatalf("identity = %#v", got)
	}
	if _, err := gate.Acquire(nil, RuntimeCallRoute); !errors.Is(err, ErrRuntimeAdmissionInvalid) {
		t.Fatalf("nil context error = %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := gate.Acquire(cancelled, RuntimeCallRoute); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled context error = %v", err)
	}
}

type runtimeAdmissionAcquireResult struct {
	lease *RuntimeAdmissionLease
	err   error
}

func newRuntimeAdmissionTestGate(t *testing.T) *RuntimeAdmissionGate {
	t.Helper()
	gate, err := NewRuntimeAdmissionGate(RuntimeInstanceIdentity{
		ExtensionID: "demo.plugin",
		InstanceID:  "runtime-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return gate
}

func acquireRuntimeAdmission(t *testing.T, gate *RuntimeAdmissionGate, class RuntimeCallClass) *RuntimeAdmissionLease {
	t.Helper()
	lease, err := gate.Acquire(context.Background(), class)
	if err != nil {
		t.Fatal(err)
	}
	return lease
}

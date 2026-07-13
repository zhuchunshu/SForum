package extensions

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestLifecycleCoordinatorActionLeaseRejectsConcurrentExecution(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	release := make(chan struct{})
	runtime := &lifecycleCoordinatorLeaseTestRuntime{started: make(chan struct{}), release: release}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)

	first := make(chan lifecycleCoordinatorTestRun, 1)
	go func() {
		result, err := coordinator.Run(context.Background(), input)
		first <- lifecycleCoordinatorTestRun{result: result, err: err}
	}()
	<-runtime.started
	blocked, err := coordinator.Run(context.Background(), input)
	if !errors.Is(err, ErrLifecycleStepLeaseConflict) || blocked.Operation.CompletedAt != nil || len(runtime.requestsSnapshot()) != 1 {
		t.Fatalf("concurrent action = %#v, %v requests=%#v", blocked, err, runtime.requestsSnapshot())
	}
	close(release)
	completed := <-first
	if completed.err != nil || completed.result.Operation.TerminalResult != LifecycleTerminalSucceeded {
		t.Fatalf("first action = %#v, %v", completed.result, completed.err)
	}
}

func TestLifecycleCoordinatorHostLeaseRejectsConcurrentExecution(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	release := make(chan struct{})
	host := &lifecycleCoordinatorLeaseTestHost{
		target: LifecycleMachineStarting, started: make(chan struct{}), release: release,
	}
	runtime := &lifecycleCoordinatorTestRuntime{}
	coordinator := NewLifecycleCoordinator(repository, runtime, host)
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)

	first := make(chan lifecycleCoordinatorTestRun, 1)
	go func() {
		result, err := coordinator.Run(context.Background(), input)
		first <- lifecycleCoordinatorTestRun{result: result, err: err}
	}()
	<-host.started
	blocked, err := coordinator.Run(context.Background(), input)
	if !errors.Is(err, ErrLifecycleStepLeaseConflict) || blocked.Operation.CompletedAt != nil ||
		host.count(LifecycleMachineStarting) != 1 || len(runtime.actionNames()) != 0 {
		t.Fatalf("concurrent Host gate = %#v, %v host=%d actions=%#v",
			blocked, err, host.count(LifecycleMachineStarting), runtime.actionNames())
	}
	close(release)
	completed := <-first
	if completed.err != nil || completed.result.Operation.TerminalResult != LifecycleTerminalSucceeded {
		t.Fatalf("first Host gate = %#v, %v", completed.result, completed.err)
	}
}

func TestLifecycleCoordinatorHeartbeatsLeaseAcrossProgressAndTerminal(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	repository.leaseHeartbeatNotify = make(chan struct{}, 1)
	runtime := &lifecycleCoordinatorLeaseTestRuntime{
		started: make(chan struct{}), waitForHeartbeat: repository.leaseHeartbeatNotify, emitProgress: true,
	}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})
	coordinator.leaseDuration = 100 * time.Millisecond
	coordinator.leaseHeartbeatInterval = time.Millisecond
	result, err := coordinator.Run(context.Background(), lifecycleCoordinatorTestInput(LifecycleMachineEnable, false))
	if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded || repository.heartbeatCount() == 0 {
		t.Fatalf("heartbeat result = %#v, %v count=%d", result, err, repository.heartbeatCount())
	}
	attempt, err := repository.LatestStepAttempt(context.Background(), result.Operation.ID, "lifecycle.enable.02.enable")
	progressRevision, completeRevision := repository.actionLeaseRevisions()
	if err != nil || attempt.Status != LifecycleStepSucceeded || attempt.Checkpoint != "done" ||
		attempt.LeaseOwnerToken != "" || attempt.LeaseExpiresAt != nil || attempt.LeaseHeartbeatAt != nil || attempt.LeaseRevision < 3 {
		t.Fatalf("heartbeat attempt = %#v, %v", attempt, err)
	}
	if progressRevision < 2 || completeRevision < progressRevision {
		t.Fatalf("lease revisions progress=%d complete=%d", progressRevision, completeRevision)
	}
}

func TestLifecycleCoordinatorExpiredLeaseTakesOverFromCheckpoint(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	repository.failAfterLeaseClaimAction = string(LifecycleMachineEnableAction)
	runtime := &lifecycleCoordinatorLeaseTestRuntime{started: make(chan struct{})}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)

	interrupted, err := coordinator.Run(context.Background(), input)
	if !errors.Is(err, errLifecycleCoordinatorTestCrash) || interrupted.Operation.CompletedAt != nil || len(runtime.requestsSnapshot()) != 0 {
		t.Fatalf("claim acknowledgement loss = %#v, %v", interrupted, err)
	}
	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, ErrLifecycleStepLeaseConflict) {
		t.Fatalf("active lease retry = %v", err)
	}
	repository.expireOpenLease("resume-expired")
	recovered, err := coordinator.Run(context.Background(), input)
	requests := runtime.requestsSnapshot()
	if err != nil || recovered.Operation.TerminalResult != LifecycleTerminalSucceeded || len(requests) != 1 ||
		requests[0].Checkpoint != "resume-expired" || requests[0].Attempt != 1 {
		t.Fatalf("expired takeover = %#v, %v requests=%#v", recovered, err, requests)
	}
}

func TestLifecycleCoordinatorHeartbeatFailureRemainsInfrastructureError(t *testing.T) {
	sentinel := errors.New("lease heartbeat database unavailable")
	repository := newLifecycleCoordinatorTestRepository()
	repository.failHeartbeatAction = string(LifecycleMachineEnableAction)
	repository.failHeartbeatOnce = sentinel
	runtime := &lifecycleCoordinatorLeaseTestRuntime{started: make(chan struct{}), waitForContext: true}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})
	coordinator.leaseDuration = 100 * time.Millisecond
	coordinator.leaseHeartbeatInterval = time.Millisecond
	result, err := coordinator.Run(context.Background(), lifecycleCoordinatorTestInput(LifecycleMachineEnable, false))
	if !errors.Is(err, sentinel) || result.Operation.CompletedAt != nil || result.Operation.TerminalResult != "" {
		t.Fatalf("heartbeat failure = %#v, %v", result, err)
	}
	attempt, latestErr := repository.LatestStepAttempt(context.Background(), result.Operation.ID, "lifecycle.enable.02.enable")
	if latestErr != nil || lifecycleStepTerminal(attempt.Status) {
		t.Fatalf("heartbeat failure terminalized step = %#v, %v", attempt, latestErr)
	}
}

func TestLifecycleCoordinatorStopCancelsInFlightHeartbeat(t *testing.T) {
	base := newLifecycleCoordinatorTestRepository()
	repository := &lifecycleCoordinatorBlockingHeartbeatRepository{
		lifecycleCoordinatorTestRepository: base,
		started:                            make(chan struct{}),
	}
	runtime := &lifecycleCoordinatorLeaseTestRuntime{waitForHeartbeat: repository.started}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})
	coordinator.leaseDuration = 100 * time.Millisecond
	coordinator.leaseHeartbeatInterval = time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan lifecycleCoordinatorTestRun, 1)
	go func() {
		result, err := coordinator.Run(ctx, lifecycleCoordinatorTestInput(LifecycleMachineEnable, false))
		done <- lifecycleCoordinatorTestRun{result: result, err: err}
	}()
	select {
	case outcome := <-done:
		if outcome.err != nil || outcome.result.Operation.TerminalResult != LifecycleTerminalSucceeded || ctx.Err() != nil {
			t.Fatalf("heartbeat stop = %#v, %v caller=%v", outcome.result, outcome.err, ctx.Err())
		}
	case <-time.After(time.Second):
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
		}
		t.Fatal("coordinator did not cancel the in-flight heartbeat")
	}
}

func TestLifecycleCoordinatorReplaysPersistedSuccessfulHostGate(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	host := &lifecycleCoordinatorLeaseTestHost{target: LifecycleMachineStarting, afterTarget: repository.failNextTransition}
	coordinator := NewLifecycleCoordinator(repository, &lifecycleCoordinatorTestRuntime{}, host)
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)
	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, errLifecycleCoordinatorTestCrash) {
		t.Fatalf("Host completion crash = %v", err)
	}
	result, err := coordinator.Run(context.Background(), input)
	if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded || host.count(LifecycleMachineStarting) != 1 {
		t.Fatalf("Host replay = %#v, %v calls=%d", result, err, host.count(LifecycleMachineStarting))
	}
}

type lifecycleCoordinatorTestRun struct {
	result LifecycleCoordinatorRunResult
	err    error
}

type lifecycleCoordinatorLeaseTestRuntime struct {
	mu               sync.Mutex
	requests         []LifecycleCoordinatorActionRequest
	started          chan struct{}
	startedOnce      sync.Once
	release          <-chan struct{}
	waitForHeartbeat <-chan struct{}
	waitForContext   bool
	emitProgress     bool
}

type lifecycleCoordinatorBlockingHeartbeatRepository struct {
	*lifecycleCoordinatorTestRepository
	started chan struct{}
	once    sync.Once
}

func (r *lifecycleCoordinatorBlockingHeartbeatRepository) HeartbeatStepLease(
	ctx context.Context,
	_ HeartbeatLifecycleStepLeaseInput,
) (LifecycleStepAttempt, error) {
	r.once.Do(func() { close(r.started) })
	<-ctx.Done()
	return LifecycleStepAttempt{}, ctx.Err()
}

func (r *lifecycleCoordinatorLeaseTestRuntime) RunLifecycleAction(
	ctx context.Context,
	request LifecycleCoordinatorActionRequest,
	onProgress func(LifecycleCoordinatorActionProgress) error,
) (LifecycleCoordinatorActionResult, error) {
	r.mu.Lock()
	r.requests = append(r.requests, request)
	r.mu.Unlock()
	r.startedOnce.Do(func() {
		if r.started != nil {
			close(r.started)
		}
	})
	if r.waitForHeartbeat != nil {
		select {
		case <-r.waitForHeartbeat:
		case <-ctx.Done():
			return LifecycleCoordinatorActionResult{}, ctx.Err()
		}
	}
	if r.release != nil {
		select {
		case <-r.release:
		case <-ctx.Done():
			return LifecycleCoordinatorActionResult{}, ctx.Err()
		}
	}
	if r.waitForContext {
		<-ctx.Done()
		return LifecycleCoordinatorActionResult{}, ctx.Err()
	}
	if r.emitProgress {
		if err := onProgress(LifecycleCoordinatorActionProgress{
			Status: LifecycleStepRunning, Checkpoint: "heartbeat-progress", CompletedUnits: 1, TotalUnits: 2,
		}); err != nil {
			return LifecycleCoordinatorActionResult{}, err
		}
	}
	return LifecycleCoordinatorActionResult{
		Status: LifecycleStepSucceeded, Checkpoint: "done", CompletedUnits: 2, TotalUnits: 2,
	}, nil
}

func (r *lifecycleCoordinatorLeaseTestRuntime) requestsSnapshot() []LifecycleCoordinatorActionRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.requests)
}

type lifecycleCoordinatorLeaseTestHost struct {
	mu          sync.Mutex
	target      LifecycleMachineState
	calls       map[LifecycleMachineState]int
	started     chan struct{}
	startedOnce sync.Once
	release     <-chan struct{}
	afterTarget func()
}

func (h *lifecycleCoordinatorLeaseTestHost) RunLifecycleHostGate(ctx context.Context, request LifecycleCoordinatorGateRequest) error {
	h.mu.Lock()
	if h.calls == nil {
		h.calls = make(map[LifecycleMachineState]int)
	}
	h.calls[request.State]++
	h.mu.Unlock()
	if request.State != h.target {
		return nil
	}
	h.startedOnce.Do(func() {
		if h.started != nil {
			close(h.started)
		}
	})
	if h.release != nil {
		select {
		case <-h.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if h.afterTarget != nil {
		h.afterTarget()
		h.afterTarget = nil
	}
	return nil
}

func (h *lifecycleCoordinatorLeaseTestHost) count(state LifecycleMachineState) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.calls[state]
}

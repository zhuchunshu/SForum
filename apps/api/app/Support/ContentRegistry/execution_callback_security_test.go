package contentregistry

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type blockingExecutionReleaseLease struct {
	ctx      context.Context
	entered  chan struct{}
	release  chan struct{}
	finished chan struct{}
	once     sync.Once
}

func (l *blockingExecutionReleaseLease) CallContext() context.Context { return l.ctx }
func (l *blockingExecutionReleaseLease) Release() {
	l.once.Do(func() { close(l.entered) })
	<-l.release
	close(l.finished)
}

type panickingContentTraceSink struct{}

func (panickingContentTraceSink) AppendContentTrace(ContentTraceEvent) { panic("trace detail") }

type countedExecutionLease struct {
	ctx      context.Context
	releases atomic.Int64
}

func (l *countedExecutionLease) CallContext() context.Context { return l.ctx }
func (l *countedExecutionLease) Release()                     { l.releases.Add(1) }

type blockingContentTraceSink struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
	calls   atomic.Int64
}

func (s *blockingContentTraceSink) AppendContentTrace(ContentTraceEvent) {
	s.calls.Add(1)
	s.once.Do(func() { close(s.entered) })
	<-s.release
}

type statefulPanickingExecutionContext struct {
	context.Context
	errCalls atomic.Int64
}

func (c *statefulPanickingExecutionContext) Err() error {
	if c.errCalls.Add(1) > 1 {
		panic("late context detail")
	}
	return nil
}

func TestExecutorNonCooperativeAdmissionIsBoundedAndQuarantined(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "admissionhang.content.block.card", ContractVersion: "admissionhang.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "admissionhang.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("never")})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	blocked := make(chan struct{})
	var closeBlocked sync.Once
	var active atomic.Int64
	t.Cleanup(func() { closeBlocked.Do(func() { close(blocked) }) })
	admission := &executionTestAdmission{acquire: func(context.Context, AdmissionRequest) (AdmissionLease, error) {
		active.Add(1)
		defer active.Add(-1)
		<-blocked
		return nil, errors.New("late admission failure")
	}}
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, admission, acceptingExecutionSchema,
		ExecutionLimits{CallTimeout: 15 * time.Millisecond, MaxConcurrentCalls: 1})
	started := time.Now()
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if !errors.Is(err, ErrExecutionTimeout) || !errors.Is(err, ErrRuntimeQuarantined) ||
		!reflect.DeepEqual(result, ExecutionResult{}) || time.Since(started) > 500*time.Millisecond {
		t.Fatalf("non-cooperative admission result=%#v error=%v elapsed=%s", result, err, time.Since(started))
	}
	if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrRuntimeQuarantined) || len(admission.snapshot()) != 1 || active.Load() != 1 {
		t.Fatalf("quarantine did not stop admission: error=%v requests=%#v active=%d", err, admission.snapshot(), active.Load())
	}
	closeBlocked.Do(func() { close(blocked) })
	deadline := time.Now().Add(time.Second)
	for active.Load() != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if active.Load() != 0 {
		t.Fatal("late admission owner did not exit")
	}
}

func TestExecutorCancelledAdmissionNeverInvokesProvider(t *testing.T) {
	previousProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previousProcs)

	registry, target := executionRegistry(t, false,
		Declaration{ID: "canceladmission.content.block.card", ContractVersion: "canceladmission.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "canceladmission.content.schema@1"},
	)
	var providerCalls atomic.Int64
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: RendererProviderFunc(
		func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
			providerCalls.Add(1)
			return executionRender(request.Target, "<p>must-not-run</p>"), nil
		},
	)})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	parent, cancel := context.WithCancel(t.Context())
	defer cancel()
	lease := &countedExecutionLease{ctx: context.Background()}
	admission := &executionTestAdmission{acquire: func(context.Context, AdmissionRequest) (AdmissionLease, error) {
		cancel()
		return lease, nil
	}}
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, admission, acceptingExecutionSchema,
		ExecutionLimits{CallTimeout: time.Second, MaxConcurrentCalls: 1})
	result, err := executor.Execute(parent, executionRequest(target, "actor"))
	if err == nil || !reflect.DeepEqual(result, ExecutionResult{}) {
		t.Fatalf("cancelled admission result=%#v error=%v", result, err)
	}
	if providerCalls.Load() != 0 || lease.releases.Load() != 1 {
		t.Fatalf("cancelled admission provider calls=%d releases=%d", providerCalls.Load(), lease.releases.Load())
	}
}

func TestExecutorNonCooperativeHostCallbacksAreBounded(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "hosthang.content.block.card", ContractVersion: "hosthang.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "hosthang.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("never")})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed

	t.Run("permission", func(t *testing.T) {
		blocked := make(chan struct{})
		var closeBlocked sync.Once
		t.Cleanup(func() { closeBlocked.Do(func() { close(blocked) }) })
		executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema,
			ExecutionLimits{CallTimeout: 15 * time.Millisecond, MaxConcurrentCalls: 1})
		request := executionRequest(target, "actor")
		request.Permission.Recheck = PermissionRecheckFunc(func(context.Context, PermissionClaim) error {
			<-blocked
			return nil
		})
		started := time.Now()
		if _, err := executor.Execute(t.Context(), request); !errors.Is(err, ErrExecutionTimeout) || time.Since(started) > 500*time.Millisecond {
			t.Fatalf("non-cooperative permission error=%v elapsed=%s", err, time.Since(started))
		}
		closeBlocked.Do(func() { close(blocked) })
	})

	t.Run("schema", func(t *testing.T) {
		blocked := make(chan struct{})
		var closeBlocked sync.Once
		t.Cleanup(func() { closeBlocked.Do(func() { close(blocked) }) })
		schemas := SchemaValidatorFunc(func(context.Context, SchemaValidationRequest) error {
			<-blocked
			return nil
		})
		executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, schemas,
			ExecutionLimits{CallTimeout: 15 * time.Millisecond, MaxConcurrentCalls: 1})
		started := time.Now()
		if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrExecutionTimeout) || time.Since(started) > 500*time.Millisecond {
			t.Fatalf("non-cooperative schema error=%v elapsed=%s", err, time.Since(started))
		}
		closeBlocked.Do(func() { close(blocked) })
	})
}

func TestExecutorFinalReleaseRetainsSlotAndQuarantinesUntilCompletion(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "releasehang.content.block.card", ContractVersion: "releasehang.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "releasehang.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("private")})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	blocked := make(chan struct{})
	entered := make(chan struct{})
	finished := make(chan struct{})
	var closeBlocked sync.Once
	t.Cleanup(func() { closeBlocked.Do(func() { close(blocked) }) })
	admission := &executionTestAdmission{acquire: func(ctx context.Context, request AdmissionRequest) (AdmissionLease, error) {
		if request.Operation == OperationRelease {
			return &blockingExecutionReleaseLease{ctx: ctx, entered: entered, release: blocked, finished: finished}, nil
		}
		return &executionTestLease{ctx: ctx}, nil
	}}
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, admission, acceptingExecutionSchema,
		ExecutionLimits{CallTimeout: 20 * time.Millisecond, MaxConcurrentCalls: 1})
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if !errors.Is(err, ErrExecutionTimeout) || !errors.Is(err, ErrRuntimeQuarantined) ||
		!errors.Is(err, ErrRuntimeUnavailable) || !reflect.DeepEqual(result, ExecutionResult{}) {
		t.Fatalf("non-cooperative final release result=%#v error=%v", result, err)
	}
	select {
	case <-entered:
	default:
		t.Fatal("final release was not entered")
	}
	if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrRuntimeQuarantined) {
		t.Fatalf("final release quarantine = %v", err)
	}
	closeBlocked.Do(func() { close(blocked) })
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("final release owner did not finish")
	}
}

func TestExecutorStatefulLeaseContextPanicReleasesOnceAndQuarantines(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "contextpanic.content.block.card", ContractVersion: "contextpanic.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "contextpanic.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("private")})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	lease := &countedExecutionLease{ctx: &statefulPanickingExecutionContext{Context: context.Background()}}
	var admissions atomic.Int64
	admission := &executionTestAdmission{acquire: func(context.Context, AdmissionRequest) (AdmissionLease, error) {
		admissions.Add(1)
		return lease, nil
	}}
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, admission, acceptingExecutionSchema,
		ExecutionLimits{CallTimeout: 50 * time.Millisecond, MaxConcurrentCalls: 1})
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if !errors.Is(err, ErrRuntimeUnavailable) || !errors.Is(err, ErrRuntimeQuarantined) ||
		!reflect.DeepEqual(result, ExecutionResult{}) || lease.releases.Load() != 1 {
		t.Fatalf("stateful context result=%#v error=%v releases=%d", result, err, lease.releases.Load())
	}
	if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrRuntimeQuarantined) || admissions.Load() != 1 {
		t.Fatalf("stateful context quarantine error=%v admissions=%d", err, admissions.Load())
	}
}

func TestExecutorSnapshotsSourceBeforePermissionCallbacks(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "snapshotinput.content.block.card", ContractVersion: "snapshotinput.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "snapshotinput.content.schema@1"},
	)
	seen := make(chan string, 1)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: RendererProviderFunc(
		func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
			seen <- string(request.Document.Value)
			return executionRender(request.Target, "<p>safe</p>"), nil
		},
	)})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	request := executionRequest(target, "actor")
	request.Document.Value = []byte(`{"value":"original"}`)
	permissionEntered := make(chan struct{})
	permissionRelease := make(chan struct{})
	var once sync.Once
	request.Permission.Recheck = PermissionRecheckFunc(func(context.Context, PermissionClaim) error {
		once.Do(func() {
			close(permissionEntered)
			<-permissionRelease
		})
		return nil
	})
	done := make(chan error, 1)
	go func() {
		_, err := executor.Execute(t.Context(), request)
		done <- err
	}()
	<-permissionEntered
	copy(request.Document.Value, []byte(`{"value":"mutated!"}`))
	close(permissionRelease)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := <-seen; got != `{"value":"original"}` {
		t.Fatalf("provider observed aliased source = %s", got)
	}
}

func TestExecutorCacheIdentityIncludesRequestInvalidationTags(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "cachetags.content.block.card", ContractVersion: "cachetags.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "cachetags.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("same")})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	first, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if err != nil {
		t.Fatal(err)
	}
	request := executionRequest(target, "actor")
	request.CacheTags = []string{"resource:topic:42"}
	second, err := executor.Execute(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.CacheKey == second.CacheKey || reflect.DeepEqual(first.CacheTags, second.CacheTags) {
		t.Fatalf("cache tag identity first=%#v second=%#v", first, second)
	}
}

func TestExecutorTraceSinkPanicCannotChangeReleasedResult(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "tracepanic.content.block.card", ContractVersion: "tracepanic.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "tracepanic.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("safe")})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema,
		ExecutionLimits{}, WithContentTraceSink(panickingContentTraceSink{}))
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if err != nil || !strings.Contains(renderHTML(result.Render), "safe") {
		t.Fatalf("trace sink changed result=%#v error=%v", result, err)
	}
}

func TestExecutorBlockingTraceSinkIsBoundedAndHasExplicitLifecycle(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "traceblocked.content.block.card", ContractVersion: "traceblocked.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "traceblocked.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("safe")})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	sink := &blockingContentTraceSink{entered: make(chan struct{}), release: make(chan struct{})}
	var releaseSink sync.Once
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema,
		ExecutionLimits{CallTimeout: 100 * time.Millisecond}, WithContentTraceSink(sink))
	t.Cleanup(func() { releaseSink.Do(func() { close(sink.release) }) })

	done := make(chan error, 1)
	go func() {
		_, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("blocking trace sink delayed Execute")
	}
	select {
	case <-sink.entered:
	case <-time.After(time.Second):
		t.Fatal("trace worker did not receive the event")
	}
	if executor.traceDispatch == nil || cap(executor.traceDispatch.events) != contentTraceQueueCapacity {
		t.Fatalf("trace dispatcher = %#v", executor.traceDispatch)
	}
	for index := 0; index < contentTraceQueueCapacity*2; index++ {
		executor.appendTrace(ContentTraceEvent{})
	}
	if len(executor.traceDispatch.events) > contentTraceQueueCapacity || executor.traceDispatch.dropped.Load() == 0 || sink.calls.Load() != 1 {
		t.Fatalf("trace queue len=%d dropped=%d sink calls=%d", len(executor.traceDispatch.events), executor.traceDispatch.dropped.Load(), sink.calls.Load())
	}
	closed := make(chan struct{})
	go func() {
		executor.Close()
		close(closed)
	}()
	select {
	case <-closed:
		t.Fatal("Close returned while the single trace worker was still inside the sink")
	case <-time.After(10 * time.Millisecond):
	}
	releaseSink.Do(func() { close(sink.release) })
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("trace worker did not stop after its sink returned")
	}
	select {
	case <-executor.traceDispatch.done:
	default:
		t.Fatal("trace worker lifecycle did not finish")
	}
}

func TestContentTraceRingRejectsUnexpectedFallbackMetadata(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "tracefallback.content.block.card", ContractVersion: "tracefallback.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "tracefallback.content.schema@1"},
	)
	ring := NewContentTraceRing(4)
	event := ContentTraceEvent{
		Revision: registry.Revision(), TargetID: target.ID, ContentID: target.ID,
		ContractVersion: target.ContractVersion, Action: ActionAdd, Operation: OperationRenderer,
		Artifact: target.Artifact, Outcome: TraceSucceeded, Fallback: strings.Repeat("private", 100),
	}
	ring.AppendContentTrace(event)
	if records := ring.ContentTraces(0); len(records) != 0 {
		t.Fatalf("unexpected fallback trace = %#v", records)
	}
	event.Outcome, event.Fallback = TraceFallback, FallbackPreserveSource
	ring.AppendContentTrace(event)
	if records := ring.ContentTraces(0); len(records) != 1 || records[0].Fallback != FallbackPreserveSource {
		t.Fatalf("valid fallback trace = %#v", records)
	}
}

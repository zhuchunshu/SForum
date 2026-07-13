package hostapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

type pluginJobResolverStub struct {
	target PluginJobRuntimeContract
	err    error
}

func (s pluginJobResolverStub) ResolvePluginJobRuntime(context.Context, string, string) (PluginJobRuntimeContract, error) {
	return s.target, s.err
}

type pluginJobExecutorStub struct {
	calls      int
	invocation supportjobs.PluginJobInvocation
	err        error
}

func (s *pluginJobExecutorStub) ExecutePluginJob(_ context.Context, invocation supportjobs.PluginJobInvocation) error {
	s.calls++
	s.invocation = invocation
	return s.err
}

func TestPluginJobWorkerExecutesExactContract(t *testing.T) {
	args := pluginJobArgsFixture()
	executor := &pluginJobExecutorStub{}
	worker := &PluginJobWorker{
		Resolver: pluginJobResolverStub{target: PluginJobRuntimeContract{Contract: args.Contract(), TrustGrantID: args.TrustGrantID}},
		Executor: executor,
	}
	err := worker.Work(context.Background(), &river.Job[PluginJobArgs]{
		JobRow: &rivertype.JobRow{ID: 41, Attempt: 2}, Args: args,
	})
	if err != nil || executor.calls != 1 {
		t.Fatalf("work = %v, calls = %d", err, executor.calls)
	}
	if executor.invocation.JobID != 41 || executor.invocation.Attempt != 2 || executor.invocation.Contract != args.Contract() ||
		executor.invocation.TrustGrantID != args.TrustGrantID || executor.invocation.Payload["page"] != 1 {
		t.Fatalf("invocation = %#v", executor.invocation)
	}
}

func TestPluginJobWorkerNeverExecutesIncompatibleContract(t *testing.T) {
	args := pluginJobArgsFixture()
	target := args.Contract()
	target.PayloadSchemaVersion = "2"
	executor := &pluginJobExecutorStub{}
	worker := &PluginJobWorker{
		Resolver: pluginJobResolverStub{target: PluginJobRuntimeContract{Contract: target, TrustGrantID: args.TrustGrantID}},
		Executor: executor,
	}
	err := worker.Work(context.Background(), &river.Job[PluginJobArgs]{Args: args})
	assertPluginJobCancelled(t, err, supportjobs.PluginJobReasonIncompatible)
	if executor.calls != 0 {
		t.Fatalf("incompatible job reached executor %d times", executor.calls)
	}
}

func TestPluginJobWorkerCancelsLegacyAndStaleTrustRows(t *testing.T) {
	legacyJSON := []byte(`{"extensionId":"demo.plugin","kind":"demo.sync","payload":{"page":1}}`)
	var legacy PluginJobArgs
	if err := json.Unmarshal(legacyJSON, &legacy); err != nil {
		t.Fatal(err)
	}
	worker := &PluginJobWorker{}
	assertPluginJobCancelled(t, worker.Work(context.Background(), &river.Job[PluginJobArgs]{Args: legacy}), supportjobs.PluginJobReasonEnvelopeInvalid)

	args := pluginJobArgsFixture()
	worker.Resolver = pluginJobResolverStub{target: PluginJobRuntimeContract{Contract: args.Contract(), TrustGrantID: "new-grant"}}
	worker.Executor = &pluginJobExecutorStub{}
	assertPluginJobCancelled(t, worker.Work(context.Background(), &river.Job[PluginJobArgs]{Args: args}), supportjobs.PluginJobReasonTrustGrantStale)
}

func TestPluginJobWorkerCancelsRuntimeChangedDuringResolutionOrDispatch(t *testing.T) {
	args := pluginJobArgsFixture()
	worker := &PluginJobWorker{Resolver: pluginJobResolverStub{err: supportjobs.ErrPluginJobRuntimeStale}}
	assertPluginJobCancelled(t, worker.Work(context.Background(), &river.Job[PluginJobArgs]{Args: args}), supportjobs.PluginJobReasonRuntimeChanged)

	executor := &pluginJobExecutorStub{err: supportjobs.ErrPluginJobRuntimeStale}
	worker = &PluginJobWorker{
		Resolver: pluginJobResolverStub{target: PluginJobRuntimeContract{Contract: args.Contract(), TrustGrantID: args.TrustGrantID}},
		Executor: executor,
	}
	assertPluginJobCancelled(t, worker.Work(context.Background(), &river.Job[PluginJobArgs]{Args: args}), supportjobs.PluginJobReasonRuntimeChanged)
	if executor.calls != 1 {
		t.Fatalf("dispatch race calls = %d", executor.calls)
	}
}

func TestPluginJobWorkerRetriesResolverAndExecutorFailures(t *testing.T) {
	args := pluginJobArgsFixture()
	temporary := errors.New("runtime temporarily unavailable")
	worker := &PluginJobWorker{Resolver: pluginJobResolverStub{err: temporary}}
	if err := worker.Work(context.Background(), &river.Job[PluginJobArgs]{Args: args}); !errors.Is(err, temporary) {
		t.Fatalf("resolver error = %v", err)
	}
	executor := &pluginJobExecutorStub{err: temporary}
	worker = &PluginJobWorker{
		Resolver: pluginJobResolverStub{target: PluginJobRuntimeContract{Contract: args.Contract(), TrustGrantID: args.TrustGrantID}},
		Executor: executor,
	}
	if err := worker.Work(context.Background(), &river.Job[PluginJobArgs]{Args: args}); !errors.Is(err, temporary) {
		t.Fatalf("executor error = %v", err)
	}
}

func assertPluginJobCancelled(t *testing.T, err error, reason string) {
	t.Helper()
	var cancelErr *river.JobCancelError
	if !errors.As(err, &cancelErr) {
		t.Fatalf("expected River cancellation, got %v", err)
	}
	var compatibility *PluginJobCompatibilityError
	if !errors.As(err, &compatibility) || compatibility.Decision.Reason != reason {
		t.Fatalf("compatibility error = %#v", compatibility)
	}
}

func pluginJobArgsFixture() PluginJobArgs {
	return PluginJobArgs{
		EnvelopeVersion: supportjobs.PluginJobEnvelopeVersion,
		ExtensionID:     "demo.plugin", ExtensionVersion: "1.0.0", ArtifactDigest: "digest-v1", TrustGrantID: "grant-1",
		JobName: "demo.sync", JobContractVersion: "demo.job.sync@1",
		PayloadSchemaID: "demo.payload", PayloadSchemaVersion: "1",
		Payload: map[string]any{"page": 1}, EnqueuedAt: time.Now().UTC(),
	}
}

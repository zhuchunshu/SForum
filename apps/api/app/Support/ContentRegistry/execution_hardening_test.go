package contentregistry

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type executionPanickingLease struct {
	ctx context.Context
}

func (l *executionPanickingLease) CallContext() context.Context { return l.ctx }
func (l *executionPanickingLease) Release()                     { panic("lease detail") }

type executionPanickingContextLease struct {
	released atomic.Bool
}

func (l *executionPanickingContextLease) CallContext() context.Context { panic("context detail") }
func (l *executionPanickingContextLease) Release()                     { l.released.Store(true) }

func TestExecutorVisibleResultNeverExposesSourceOrSerializedPayload(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "wire.content.block.card", ContractVersion: "wire.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "wire.content.schema@1"},
	)
	providers := ProviderSet{
		Serializer: SerializerProviderFunc(func(_ context.Context, request SerializerProviderRequest) (SerializedContent, error) {
			return SerializedContent{
				SchemaVersion: SerializedSchemaVersion, ContentID: request.Target.ID,
				ContractVersion: request.Target.ContractVersion, StorageVersion: request.Document.StorageVersion,
				MediaType: "application/json", Data: []byte(`{"private":"serialized-only-secret"}`),
			}, nil
		}),
		Renderer: RendererProviderFunc(func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
			return executionRender(request.Target, "<p>public output</p>"), nil
		}),
	}
	binding := executionBinding(target, target.ID, ActionAdd, 0, providers)
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	request := executionRequest(target, "actor")
	request.Document.Value = []byte(`{"private":"editor-only-secret"}`)
	result, err := executor.Execute(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["document"] != nil || wire["serialized"] != nil ||
		strings.Contains(string(body), "editor-only-secret") || strings.Contains(string(body), "serialized-only-secret") ||
		!strings.Contains(string(body), "public output") {
		t.Fatalf("public result boundary = %s", body)
	}
}

func TestExecutorProviderCannotSpoofHostTimeout(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "spoof.content.block.card", ContractVersion: "spoof.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "spoof.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: RendererProviderFunc(
		func(context.Context, RendererProviderRequest) (RenderSegments, error) {
			return RenderSegments{}, ErrExecutionTimeout
		},
	)})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if !errors.Is(err, ErrProviderFailed) || errors.Is(err, ErrExecutionTimeout) ||
		!reflect.DeepEqual(result, ExecutionResult{}) {
		t.Fatalf("spoofed timeout result=%#v error=%v", result, err)
	}
}

func TestExecutorHostCallbackPanicsFailClosed(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "hostpanic.content.block.card", ContractVersion: "hostpanic.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "hostpanic.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("private")})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	t.Run("permission", func(t *testing.T) {
		executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
		request := executionRequest(target, "actor")
		request.Permission.Recheck = PermissionRecheckFunc(func(context.Context, PermissionClaim) error {
			panic("permission detail")
		})
		if _, err := executor.Execute(t.Context(), request); !errors.Is(err, ErrExecutionDenied) {
			t.Fatalf("permission panic = %v", err)
		}
	})
	t.Run("schema", func(t *testing.T) {
		schemas := SchemaValidatorFunc(func(context.Context, SchemaValidationRequest) error {
			panic("schema detail")
		})
		executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, schemas, ExecutionLimits{})
		if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrSchemaRejected) {
			t.Fatalf("schema panic = %v", err)
		}
	})
	t.Run("admission", func(t *testing.T) {
		admission := &executionTestAdmission{acquire: func(context.Context, AdmissionRequest) (AdmissionLease, error) {
			panic("admission detail")
		}}
		executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, admission, acceptingExecutionSchema, ExecutionLimits{})
		if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrRuntimeUnavailable) {
			t.Fatalf("admission panic = %v", err)
		}
	})
	t.Run("lease_context", func(t *testing.T) {
		lease := &executionPanickingContextLease{}
		admission := &executionTestAdmission{acquire: func(context.Context, AdmissionRequest) (AdmissionLease, error) {
			return lease, nil
		}}
		executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, admission, acceptingExecutionSchema, ExecutionLimits{})
		if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrRuntimeUnavailable) {
			t.Fatalf("lease context panic = %v", err)
		}
		if !lease.released.Load() {
			t.Fatal("panicking lease context was not released")
		}
	})
}

func TestExecutorTotalDeadlineBoundsSequentialProviders(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "deadline.content.block.base", ContractVersion: "deadline.content.block.base@1", Kind: KindBlock, Handler: "base", Schema: "deadline.content.schema@1"},
		Declaration{ID: "deadline.content.block.before", ContractVersion: "deadline.content.block.before@1", Kind: KindBlock, Handler: "before", Schema: "deadline.content.schema@1"},
		Declaration{ID: "deadline.content.block.after", ContractVersion: "deadline.content.block.after@1", Kind: KindBlock, Handler: "after", Schema: "deadline.content.schema@1"},
	)
	contributions := map[string]Contribution{}
	for _, contribution := range registry.List("") {
		contributions[contribution.ID] = contribution
	}
	var calls atomic.Int64
	slow := RendererProviderFunc(func(ctx context.Context, request RendererProviderRequest) (RenderSegments, error) {
		calls.Add(1)
		select {
		case <-time.After(30 * time.Millisecond):
			return executionRender(request.Target, "<p>step</p>"), nil
		case <-ctx.Done():
			return RenderSegments{}, ctx.Err()
		}
	})
	bindings := []ExecutionBinding{
		executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: slow}),
		executionBinding(target, "deadline.content.block.before", ActionBefore, 0, ProviderSet{Renderer: slow}),
		executionBinding(target, "deadline.content.block.after", ActionAfter, 0, ProviderSet{Renderer: slow}),
	}
	for index := range bindings {
		contribution := contributions[bindings[index].DeclarationID]
		bindings[index].ContractVersion, bindings[index].Artifact = contribution.ContractVersion, contribution.Artifact
		bindings[index].Fallback = FallbackClosed
	}
	executor := newExecutionTestExecutor(t, registry, bindings, &executionTestAdmission{}, acceptingExecutionSchema,
		ExecutionLimits{CallTimeout: 70 * time.Millisecond})
	started := time.Now()
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if !errors.Is(err, ErrExecutionTimeout) || !reflect.DeepEqual(result, ExecutionResult{}) ||
		time.Since(started) > 500*time.Millisecond || calls.Load() < 2 {
		t.Fatalf("deadline result=%#v error=%v calls=%d elapsed=%s", result, err, calls.Load(), time.Since(started))
	}
}

func TestExecutorLeaseReleasePanicFailsClosed(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "leasepanic.content.block.card", ContractVersion: "leasepanic.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "leasepanic.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("private")})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	admission := &executionTestAdmission{acquire: func(ctx context.Context, _ AdmissionRequest) (AdmissionLease, error) {
		return &executionPanickingLease{ctx: ctx}, nil
	}}
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, admission, acceptingExecutionSchema, ExecutionLimits{})
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if !errors.Is(err, ErrRuntimeUnavailable) || !errors.Is(err, ErrRuntimeQuarantined) ||
		!reflect.DeepEqual(result, ExecutionResult{}) {
		t.Fatalf("release panic result=%#v error=%v", result, err)
	}
	if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrRuntimeQuarantined) || len(admission.snapshot()) != 1 {
		t.Fatalf("release panic did not quarantine runtime: error=%v requests=%#v", err, admission.snapshot())
	}
}

func TestExecutorRuntimeLeaseCancellationCannotUseProviderFallback(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "leasecancel.content.block.card", ContractVersion: "leasecancel.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "leasecancel.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: RendererProviderFunc(
		func(ctx context.Context, _ RendererProviderRequest) (RenderSegments, error) {
			<-ctx.Done()
			return RenderSegments{}, errors.New("runtime stopped")
		},
	)})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackPreserveSource
	admission := &executionTestAdmission{acquire: func(ctx context.Context, _ AdmissionRequest) (AdmissionLease, error) {
		leaseCtx, cancel := context.WithCancel(ctx)
		time.AfterFunc(time.Millisecond, cancel)
		return &executionTestLease{ctx: leaseCtx}, nil
	}}
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, admission, acceptingExecutionSchema,
		ExecutionLimits{CallTimeout: time.Second})
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if !errors.Is(err, ErrRuntimeUnavailable) || !reflect.DeepEqual(result, ExecutionResult{}) {
		t.Fatalf("runtime lease cancellation result=%#v error=%v", result, err)
	}
}

func TestExecutorFinalFenceLeaseReleasePanicFailsClosed(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "fencepanic.content.block.card", ContractVersion: "fencepanic.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "fencepanic.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("private")})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	admission := &executionTestAdmission{acquire: func(ctx context.Context, request AdmissionRequest) (AdmissionLease, error) {
		if request.Operation == OperationRelease {
			return &executionPanickingLease{ctx: ctx}, nil
		}
		return &executionTestLease{ctx: ctx}, nil
	}}
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, admission, acceptingExecutionSchema, ExecutionLimits{})
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if !errors.Is(err, ErrRuntimeUnavailable) || !reflect.DeepEqual(result, ExecutionResult{}) {
		t.Fatalf("final release panic result=%#v error=%v", result, err)
	}
}

func TestExecutorFinalPermissionFenceRunsAfterReleaseSchemaValidation(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "finalpermission.content.block.card", ContractVersion: "finalpermission.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "finalpermission.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("private")})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	var revoked atomic.Bool
	schemas := SchemaValidatorFunc(func(_ context.Context, request SchemaValidationRequest) error {
		if request.Phase == SchemaPhaseOutput {
			revoked.Store(true)
		}
		return nil
	})
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, schemas, ExecutionLimits{})
	request := executionRequest(target, "actor")
	request.Permission.Recheck = PermissionRecheckFunc(func(_ context.Context, claim PermissionClaim) error {
		if claim.Operation == OperationRelease && revoked.Load() {
			return ErrExecutionDenied
		}
		return nil
	})
	result, err := executor.Execute(t.Context(), request)
	if !errors.Is(err, ErrExecutionDenied) || !reflect.DeepEqual(result, ExecutionResult{}) {
		t.Fatalf("final permission result=%#v error=%v", result, err)
	}
}

func TestExecutorPreserveSourceFallbackUsesOnlyHostSafeOutput(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "preserve.content.block.card", ContractVersion: "preserve.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "preserve.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: RendererProviderFunc(
		func(context.Context, RendererProviderRequest) (RenderSegments, error) {
			return RenderSegments{}, errors.New("ordinary provider failure")
		},
	)})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackPreserveSource
	admission := &executionTestAdmission{}
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, admission, acceptingExecutionSchema, ExecutionLimits{})
	request := executionRequest(target, "actor")
	request.Document.Value = []byte(`{"secret":"preserved-but-private"}`)
	result, err := executor.Execute(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(result)
	if !result.FallbackUsed || !result.SourcePreserved || len(result.Attribution) != 0 ||
		len(result.Render.Segments) != 1 || result.Render.Segments[0].Kind != SegmentUnsupported ||
		strings.Contains(string(body), "preserved-but-private") || len(admission.snapshot()) != 1 {
		t.Fatalf("preserve fallback result=%#v admissions=%#v wire=%s", result, admission.snapshot(), body)
	}
}

func TestExecutorNestedBaseAuthorityFailureCannotReachPreserveFallback(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "nestedfallback.content.block.base", ContractVersion: "nestedfallback.content.block.base@1", Kind: KindBlock, Handler: "base", Schema: "nestedfallback.content.schema@1"},
		Declaration{ID: "nestedfallback.content.block.replace", ContractVersion: "nestedfallback.content.block.replace@1", Kind: KindBlock, Handler: "replace", Schema: "nestedfallback.content.schema@1"},
	)
	replacement, _ := registry.Resolve("nestedfallback.content.block.replace")
	bindings := []ExecutionBinding{
		executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: RendererProviderFunc(
			func(context.Context, RendererProviderRequest) (RenderSegments, error) { panic("base panic") },
		)}),
		executionBinding(target, replacement.ID, ActionReplace, 10, ProviderSet{Renderer: RendererProviderFunc(
			func(context.Context, RendererProviderRequest) (RenderSegments, error) {
				return RenderSegments{}, errors.New("ordinary replacement failure")
			},
		)}),
	}
	bindings[0].ContractVersion, bindings[0].Artifact = target.ContractVersion, target.Artifact
	bindings[1].ContractVersion, bindings[1].Artifact = replacement.ContractVersion, replacement.Artifact
	executor := newExecutionTestExecutor(t, registry, bindings, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if !errors.Is(err, ErrProviderPanic) || !reflect.DeepEqual(result, ExecutionResult{}) {
		t.Fatalf("nested fallback result=%#v error=%v", result, err)
	}
}

func TestExecutorCacheIdentityIncludesMediaAndRenderedOutput(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "cachecontract.content.block.card", ContractVersion: "cachecontract.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "cachecontract.content.schema@1"},
	)
	build := func(mediaType, rendered string) *Executor {
		providers := ProviderSet{
			Serializer: SerializerProviderFunc(func(_ context.Context, request SerializerProviderRequest) (SerializedContent, error) {
				return SerializedContent{
					SchemaVersion: SerializedSchemaVersion, ContentID: request.Target.ID,
					ContractVersion: request.Target.ContractVersion, StorageVersion: request.Document.StorageVersion,
					MediaType: mediaType, Data: append([]byte(nil), request.Document.Value...),
				}, nil
			}),
			Renderer: RendererProviderFunc(func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
				return executionRender(request.Target, "<p>"+rendered+"</p>"), nil
			}),
		}
		binding := executionBinding(target, target.ID, ActionAdd, 0, providers)
		binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
		return newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	}
	jsonResult, err := build("application/json", "same").Execute(t.Context(), executionRequest(target, "actor"))
	if err != nil {
		t.Fatal(err)
	}
	vendorResult, err := build("application/vnd.sforum.editor+json", "same").Execute(t.Context(), executionRequest(target, "actor"))
	if err != nil {
		t.Fatal(err)
	}
	renderResult, err := build("application/json", "different").Execute(t.Context(), executionRequest(target, "actor"))
	if err != nil {
		t.Fatal(err)
	}
	if jsonResult.CacheKey == vendorResult.CacheKey || jsonResult.CacheKey == renderResult.CacheKey {
		t.Fatalf("cache keys collided: json=%s vendor=%s render=%s",
			jsonResult.CacheKey, vendorResult.CacheKey, renderResult.CacheKey)
	}
}

func TestExecutorRejectsUnsupportedStorageVersion(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "storageversion.content.block.card", ContractVersion: "storageversion.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "storageversion.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("unused")})
	binding.ContractVersion, binding.Artifact = target.ContractVersion, target.Artifact
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	request := executionRequest(target, "actor")
	request.Document.StorageVersion = "2"
	if _, err := executor.Execute(t.Context(), request); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("unsupported storage version = %v", err)
	}
}

func TestExecutorRejectsOversizedIgnoredProviderFields(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "preflight.content.block.card", ContractVersion: "preflight.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "preflight.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: RendererProviderFunc(
		func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
			result := executionRender(request.Target, "<p>small</p>")
			result.PlainText = strings.Repeat("ignored", 1024)
			return result, nil
		},
	)})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema,
		ExecutionLimits{MaxOutputBytes: 1024})
	if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrExecutionLimit) {
		t.Fatalf("oversized ignored provider field = %v", err)
	}
}

func TestExecutorRejectedOutputTraceIsNeverSucceeded(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "traceinvalid.content.block.card", ContractVersion: "traceinvalid.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "traceinvalid.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: RendererProviderFunc(
		func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
			result := executionRender(request.Target, "<p>bad</p>")
			result.ContractVersion = "traceinvalid.content.block.card@2"
			return result, nil
		},
	)})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	trace := NewContentTraceRing(16)
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema,
		ExecutionLimits{}, WithContentTraceSink(trace))
	if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrContractStale) {
		t.Fatalf("invalid output = %v", err)
	}
	records := trace.ContentTracesForTarget(target.ID, 10)
	if len(records) != 1 || records[0].Outcome != TraceStale {
		t.Fatalf("invalid output traces = %#v", records)
	}
}

func TestExecutorInspectorUsesTargetedTraceWindow(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "targettrace.content.block.card", ContractVersion: "targettrace.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "targettrace.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("target")})
	binding.ContractVersion, binding.Artifact = target.ContractVersion, target.Artifact
	trace := NewContentTraceRing(512)
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema,
		ExecutionLimits{}, WithContentTraceSink(trace))
	if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 150; index++ {
		trace.AppendContentTrace(ContentTraceEvent{
			Revision: registry.Snapshot().Revision, TargetID: "noise.content.block.card",
			ContentID: "noise.content.block.card", ContractVersion: "noise.content.block.card@1",
			Action: ActionAdd, Operation: OperationRenderer, Artifact: target.Artifact,
			Outcome: TraceSucceeded, Duration: time.Microsecond,
		})
	}
	inspection, err := executor.Inspect(target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspection.Traces) == 0 || inspection.Traces[0].TargetID != target.ID {
		t.Fatalf("targeted inspector traces = %#v", inspection.Traces)
	}
}

func TestExecutorRejectsStaleTargetContractBinding(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "targetcontract.content.block.card", ContractVersion: "targetcontract.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "targetcontract.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("unused")})
	binding.TargetContractVersion = "targetcontract.content.block.card@2"
	binding.ContractVersion, binding.Artifact = target.ContractVersion, target.Artifact
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrCompositionInvalid) {
		t.Fatalf("stale target binding = %v", err)
	}
}

func TestExecutorRejectsRendererOnlyDeclarationAsBackendCallback(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "staticrender.content.block.card", ContractVersion: "staticrender.content.block.card@1", Kind: KindBlock,
			Renderer: "staticrender.card", Schema: "staticrender.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("must-not-run")})
	binding.ContractVersion, binding.Artifact = target.ContractVersion, target.Artifact
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrContractInsufficient) {
		t.Fatalf("renderer-only backend callback = %v", err)
	}
}

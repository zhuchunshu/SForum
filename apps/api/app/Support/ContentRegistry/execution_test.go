package contentregistry

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type executionTestAdmission struct {
	mu       sync.Mutex
	requests []AdmissionRequest
	acquire  func(context.Context, AdmissionRequest) (AdmissionLease, error)
}

func (a *executionTestAdmission) AcquireContentExecution(ctx context.Context, request AdmissionRequest) (AdmissionLease, error) {
	a.mu.Lock()
	a.requests = append(a.requests, request)
	a.mu.Unlock()
	if a.acquire != nil {
		return a.acquire(ctx, request)
	}
	return &executionTestLease{ctx: ctx}, nil
}

func (a *executionTestAdmission) snapshot() []AdmissionRequest {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]AdmissionRequest(nil), a.requests...)
}

type executionTestLease struct {
	ctx context.Context
}

func (l *executionTestLease) CallContext() context.Context { return l.ctx }
func (l *executionTestLease) Release()                     {}

type executionTrackedLease struct {
	ctx      context.Context
	once     sync.Once
	released chan struct{}
}

func (l *executionTrackedLease) CallContext() context.Context { return l.ctx }
func (l *executionTrackedLease) Release() {
	l.once.Do(func() { close(l.released) })
}

func TestExecutorRunsTypedPipelineAndStableContracts(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "typed.content.block.card", ContractVersion: "typed.content.block.card@1", Kind: KindBlock,
			Handler: "typed.card", Schema: "typed.content.block.card.schema@1"},
	)
	var calls []string
	var mu sync.Mutex
	record := func(value string) {
		mu.Lock()
		calls = append(calls, value)
		mu.Unlock()
	}
	providers := ProviderSet{
		Editor: EditorProviderFunc(func(_ context.Context, request EditorProviderRequest) (EditorDocument, error) {
			record(OperationEditor)
			request.Document.Value = []byte(`{"type":"doc","text":"normalized"}`)
			return request.Document, nil
		}),
		Validator: ValidatorProviderFunc(func(_ context.Context, request ValidatorProviderRequest) error {
			record(OperationValidator)
			if !strings.Contains(string(request.Document.Value), "normalized") {
				t.Fatal("validator did not receive editor output")
			}
			return nil
		}),
		Serializer: SerializerProviderFunc(func(_ context.Context, request SerializerProviderRequest) (SerializedContent, error) {
			record(OperationSerializer)
			return SerializedContent{
				SchemaVersion: SerializedSchemaVersion, ContentID: request.Target.ID,
				ContractVersion: request.Target.ContractVersion, StorageVersion: request.Document.StorageVersion,
				MediaType: "application/json", Data: append([]byte(nil), request.Document.Value...),
			}, nil
		}),
		Renderer: RendererProviderFunc(func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
			record(OperationRenderer)
			return executionRender(request.Target, `<p>normalized</p>`), nil
		}),
	}
	admission := &executionTestAdmission{}
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{{
		TargetID: target.ID, TargetContractVersion: target.ContractVersion,
		DeclarationID: target.ID, ContractVersion: target.ContractVersion,
		Artifact: target.Artifact, Action: ActionAdd, Fallback: FallbackClosed, Providers: providers,
	}}, admission, acceptingExecutionSchema, ExecutionLimits{})
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor-1"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{OperationEditor, OperationValidator, OperationSerializer, OperationRenderer}) {
		t.Fatalf("typed call order = %#v", calls)
	}
	if result.SchemaVersion != ExecutionSchemaVersion || result.Render.SchemaVersion != RenderSegmentsSchemaVersion ||
		result.Render.PlainText != "normalized" || len(result.Attribution) != 1 {
		t.Fatalf("stable result = %#v", result)
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["document"] != nil || wire["serialized"] != nil || strings.Contains(string(body), "normalized") && wire["render"] == nil {
		t.Fatalf("visible JSON exposed storage payloads: %s", body)
	}
	operations := map[string]int{}
	for _, request := range admission.snapshot() {
		operations[request.Operation]++
		if request.Artifact != target.Artifact || request.ContentID != target.ID || request.ContractVersion != target.ContractVersion {
			t.Fatalf("non-exact admission = %#v", request)
		}
	}
	for _, operation := range []string{OperationEditor, OperationValidator, OperationSerializer, OperationRenderer, OperationRelease} {
		if operations[operation] != 1 {
			t.Fatalf("admission %s count = %d", operation, operations[operation])
		}
	}
}

func TestExecutorCompositionIsDeterministicAndInspectable(t *testing.T) {
	declarations := []Declaration{
		{ID: "compose.content.block.base", ContractVersion: "compose.content.block.base@1", Kind: KindBlock, Handler: "base", Schema: "compose.content.schema@1"},
		{ID: "compose.content.block.before", ContractVersion: "compose.content.block.before@1", Kind: KindBlock, Handler: "before", Schema: "compose.content.schema@1"},
		{ID: "compose.content.block.after", ContractVersion: "compose.content.block.after@1", Kind: KindBlock, Handler: "after", Schema: "compose.content.schema@1"},
		{ID: "compose.content.block.wrap", ContractVersion: "compose.content.block.wrap@1", Kind: KindBlock, Handler: "wrap", Schema: "compose.content.schema@1"},
		{ID: "compose.content.block.replace.a", ContractVersion: "compose.content.block.replace.a@1", Kind: KindBlock, Handler: "replace.a", Schema: "compose.content.schema@1"},
		{ID: "compose.content.block.replace.b", ContractVersion: "compose.content.block.replace.b@1", Kind: KindBlock, Handler: "replace.b", Schema: "compose.content.schema@1"},
		{ID: "compose.content.filter.final", ContractVersion: "compose.content.filter.final@1", Kind: KindRenderFilter, Handler: "filter", Schema: "compose.content.schema@1"},
	}
	registry, target := executionRegistry(t, false, declarations...)
	contributions := map[string]Contribution{}
	for _, item := range registry.List("") {
		contributions[item.ID] = item
	}
	renderer := func(label string) RendererProvider {
		return RendererProviderFunc(func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
			if request.Action == ActionWrap {
				return executionRender(request.Target, `<section data-wrap="`+label+`">`+renderHTML(request.Inner)+`</section>`), nil
			}
			return executionRender(request.Target, `<span>`+label+`</span>`), nil
		})
	}
	filter := FilterProviderFunc(func(_ context.Context, request FilterProviderRequest) (RenderSegments, error) {
		request.Render.Segments = append(request.Render.Segments, RenderSegment{Kind: SegmentHTML, HTML: `<i>filtered</i>`})
		return request.Render, nil
	})
	bindings := []ExecutionBinding{
		executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: renderer("base")}),
		executionBinding(target, "compose.content.block.before", ActionBefore, 20, ProviderSet{Renderer: renderer("before")}),
		executionBinding(target, "compose.content.block.after", ActionAfter, 20, ProviderSet{Renderer: renderer("after")}),
		executionBinding(target, "compose.content.block.wrap", ActionWrap, 20, ProviderSet{Renderer: renderer("wrap")}),
		executionBinding(target, "compose.content.block.replace.b", ActionReplace, 100, ProviderSet{Renderer: renderer("replace-b")}),
		executionBinding(target, "compose.content.block.replace.a", ActionReplace, 100, ProviderSet{Renderer: renderer("replace-a")}),
		executionBinding(target, "compose.content.filter.final", ActionFilter, 10, ProviderSet{Filter: filter}),
	}
	for index := range bindings {
		contribution := contributions[bindings[index].DeclarationID]
		bindings[index].ContractVersion = contribution.ContractVersion
		bindings[index].Artifact = contribution.Artifact
		bindings[index].CacheTags = []string{"compose:" + strings.TrimPrefix(contribution.ID, "compose.content.")}
	}
	trace := NewContentTraceRing(32)
	executor := newExecutionTestExecutor(t, registry, bindings, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{}, WithContentTraceSink(trace))
	first, err := executor.Execute(t.Context(), executionRequest(target, "actor-1"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := executor.Execute(t.Context(), executionRequest(target, "actor-1"))
	if err != nil {
		t.Fatal(err)
	}
	html := renderHTML(first.Render)
	if !strings.Contains(html, "before") || !strings.Contains(html, "replace-a") || strings.Contains(html, "replace-b") ||
		!strings.Contains(html, "<section>") || !strings.Contains(html, "after") || !strings.Contains(html, "filtered") {
		t.Fatalf("composed html = %s", html)
	}
	if first.CacheKey != second.CacheKey || !reflect.DeepEqual(first.Attribution, second.Attribution) {
		t.Fatal("same composition did not produce a stable identity")
	}
	inspection, err := executor.Inspect(target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspection.Conflicts) != 1 || len(inspection.Conflicts[0].Candidates) != 2 ||
		inspection.Conflicts[0].Candidates[0].ContentID != "compose.content.block.replace.a" ||
		len(inspection.Traces) == 0 {
		t.Fatalf("inspection = %#v", inspection)
	}
}

func TestExecutorOrdinaryProviderFailureUsesDeclaredBaseFallback(t *testing.T) {
	replacer := RendererProviderFunc(func(context.Context, RendererProviderRequest) (RenderSegments, error) {
		return RenderSegments{}, errors.New("provider detail must not escape")
	})
	registry, target := executionRegistry(t, false,
		Declaration{ID: "fallback.content.block.base", ContractVersion: "fallback.content.block.base@1", Kind: KindBlock, Handler: "base", Schema: "fallback.content.schema@1"},
		Declaration{ID: "fallback.content.block.replace", ContractVersion: "fallback.content.block.replace@1", Kind: KindBlock, Handler: "replace", Schema: "fallback.content.schema@1"},
	)
	replacement, _ := registry.Resolve("fallback.content.block.replace")
	bindings := []ExecutionBinding{
		executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("base")}),
		executionBinding(target, replacement.ID, ActionReplace, 100, ProviderSet{Renderer: replacer}),
	}
	bindings[0].ContractVersion, bindings[0].Artifact = target.ContractVersion, target.Artifact
	bindings[1].ContractVersion, bindings[1].Artifact = replacement.ContractVersion, replacement.Artifact
	executor := newExecutionTestExecutor(t, registry, bindings, &executionTestAdmission{}, acceptingExecutionSchema,
		ExecutionLimits{CallTimeout: 200 * time.Millisecond})
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor-1"))
	if err != nil {
		t.Fatal(err)
	}
	if !result.FallbackUsed || !strings.Contains(renderHTML(result.Render), "base") ||
		len(result.Attribution) != 1 || result.Attribution[0].ContentID != target.ID {
		t.Fatalf("fallback result = %#v", result)
	}
}

func TestExecutorProviderPanicFailsClosed(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "panic.content.block.base", ContractVersion: "panic.content.block.base@1", Kind: KindBlock, Handler: "base", Schema: "panic.content.schema@1"},
		Declaration{ID: "panic.content.block.replace", ContractVersion: "panic.content.block.replace@1", Kind: KindBlock, Handler: "replace", Schema: "panic.content.schema@1"},
	)
	replacement, _ := registry.Resolve("panic.content.block.replace")
	bindings := []ExecutionBinding{
		executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("base")}),
		executionBinding(target, replacement.ID, ActionReplace, 100, ProviderSet{Renderer: RendererProviderFunc(func(context.Context, RendererProviderRequest) (RenderSegments, error) {
			panic("provider secret")
		})}),
	}
	bindings[0].ContractVersion, bindings[0].Artifact = target.ContractVersion, target.Artifact
	bindings[1].ContractVersion, bindings[1].Artifact = replacement.ContractVersion, replacement.Artifact
	executor := newExecutionTestExecutor(t, registry, bindings, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if !errors.Is(err, ErrProviderPanic) || !reflect.DeepEqual(result, ExecutionResult{}) {
		t.Fatalf("panic result=%#v error=%v", result, err)
	}
}

func TestExecutorHideWinsSelectedTerminalWithoutRendering(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "hide.content.block.base", ContractVersion: "hide.content.block.base@1", Kind: KindBlock, Handler: "base", Schema: "hide.content.schema@1"},
		Declaration{ID: "hide.content.block.hide", ContractVersion: "hide.content.block.hide@1", Kind: KindBlock, Handler: "hide", Schema: "hide.content.schema@1"},
		Declaration{ID: "hide.content.block.replace", ContractVersion: "hide.content.block.replace@1", Kind: KindBlock, Handler: "replace", Schema: "hide.content.schema@1"},
	)
	hide, _ := registry.Resolve("hide.content.block.hide")
	replacement, _ := registry.Resolve("hide.content.block.replace")
	var rendererCalls atomic.Int64
	renderer := RendererProviderFunc(func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
		rendererCalls.Add(1)
		return executionRender(request.Target, "<p>visible</p>"), nil
	})
	bindings := []ExecutionBinding{
		{TargetID: target.ID, TargetContractVersion: target.ContractVersion, DeclarationID: target.ID, ContractVersion: target.ContractVersion, Artifact: target.Artifact,
			Action: ActionAdd, Providers: ProviderSet{Renderer: renderer}},
		{TargetID: target.ID, TargetContractVersion: target.ContractVersion, DeclarationID: replacement.ID, ContractVersion: replacement.ContractVersion, Artifact: replacement.Artifact,
			Action: ActionReplace, Priority: 50, Providers: ProviderSet{Renderer: renderer}},
		{TargetID: target.ID, TargetContractVersion: target.ContractVersion, DeclarationID: hide.ID, ContractVersion: hide.ContractVersion, Artifact: hide.Artifact,
			Action: ActionHide, Priority: 100},
	}
	admission := &executionTestAdmission{}
	executor := newExecutionTestExecutor(t, registry, bindings, admission, acceptingExecutionSchema, ExecutionLimits{})
	request := executionRequest(target, "actor")
	request.Document.Value = []byte(`{"type":"doc","secret":"hidden-editor-secret"}`)
	result, err := executor.Execute(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Hidden || len(result.Render.Segments) != 0 ||
		result.Render.TextEncoding != RenderTextEncodingHTMLEscaped || rendererCalls.Load() != 0 ||
		len(result.Attribution) != 1 || result.Attribution[0].ContentID != hide.ID {
		t.Fatalf("hidden result = %#v renderer calls=%d", result, rendererCalls.Load())
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	if _, exposed := wire["document"]; exposed {
		t.Fatalf("hidden JSON exposed document: %s", body)
	}
	if _, exposed := wire["serialized"]; exposed {
		t.Fatalf("hidden JSON exposed serialized content: %s", body)
	}
	if strings.Contains(string(body), "hidden-editor-secret") || strings.Contains(string(body), "hidden-serialized-secret") ||
		wire["render"] == nil || wire["attribution"] == nil {
		t.Fatalf("hidden JSON payload/evidence = %s", body)
	}
	requests := admission.snapshot()
	if len(requests) != 2 || requests[0].Operation != OperationHide || requests[1].Operation != OperationRelease {
		t.Fatalf("hide admissions = %#v", requests)
	}
}

func TestExecutorCacheIdentityIsActorAndPolicyIsolated(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "cache.content.block.card", ContractVersion: "cache.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "cache.content.schema@1"},
	)
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: staticExecutionRenderer("cache")})
	binding.ContractVersion, binding.Artifact = target.ContractVersion, target.Artifact
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	first, err := executor.Execute(t.Context(), executionRequest(target, "actor-1"))
	if err != nil {
		t.Fatal(err)
	}
	secondRequest := executionRequest(target, "actor-2")
	second, err := executor.Execute(t.Context(), secondRequest)
	if err != nil {
		t.Fatal(err)
	}
	policyRequest := executionRequest(target, "actor-1")
	policyRequest.Permission.PolicyFingerprint = "policy-v2"
	third, err := executor.Execute(t.Context(), policyRequest)
	if err != nil {
		t.Fatal(err)
	}
	if first.CacheKey == second.CacheKey || first.CacheKey == third.CacheKey ||
		!reflect.DeepEqual(first.CacheTags, []string{"content:" + target.ID}) {
		t.Fatalf("cache isolation first=%#v second=%#v third=%#v", first, second, third)
	}
}

func TestExecutorSafeModeUsesOnlyCoreBindings(t *testing.T) {
	core := publication("core.safecontent", true, 'a')
	core.Content = []Declaration{{
		ID: "core.safecontent.block.card", ContractVersion: "core.safecontent.block.card@1",
		Kind: KindBlock, Handler: "core.card", Schema: "core.safecontent.schema@1",
	}}
	plugin := publication("safe.plugin", false, 'b')
	plugin.Content = []Declaration{{
		ID: "safe.plugin.block.before", ContractVersion: "safe.plugin.block.before@1",
		Kind: KindBlock, Handler: "plugin.before", Schema: "safe.plugin.schema@1",
	}}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{plugin, core}, true); err != nil {
		t.Fatal(err)
	}
	target, _ := registry.Resolve(core.Content[0].ID)
	bindings := []ExecutionBinding{
		{TargetID: target.ID, TargetContractVersion: target.ContractVersion, DeclarationID: target.ID, ContractVersion: target.ContractVersion, Artifact: target.Artifact,
			Action: ActionAdd, Providers: ProviderSet{Renderer: staticExecutionRenderer("core")}},
		{TargetID: target.ID, TargetContractVersion: target.ContractVersion, DeclarationID: plugin.Content[0].ID, ContractVersion: plugin.Content[0].ContractVersion,
			Artifact: plugin.Artifact, Action: ActionBefore, Providers: ProviderSet{Renderer: staticExecutionRenderer("plugin")}},
	}
	executor := newExecutionTestExecutor(t, registry, bindings, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	result, err := executor.Execute(t.Context(), executionRequest(target, "anonymous"))
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := executor.Inspect(target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(renderHTML(result.Render), "plugin") || !inspection.SafeMode || len(inspection.Stale) != 1 {
		t.Fatalf("safe mode result=%#v inspection=%#v", result, inspection)
	}
}

func executionRegistry(t *testing.T, core bool, declarations ...Declaration) (*Registry, Contribution) {
	t.Helper()
	extensionID := strings.Split(declarations[0].ID, ".block.")[0]
	if !strings.Contains(declarations[0].ID, ".block.") {
		extensionID = strings.Join(strings.Split(declarations[0].ID, ".")[:2], ".")
	}
	item := publication(extensionID, core, 'a')
	item.Content = append([]Declaration(nil), declarations...)
	registry := New()
	if _, err := registry.Publish(item); err != nil {
		t.Fatal(err)
	}
	target, err := registry.Resolve(declarations[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	return registry, target
}

func newExecutionTestExecutor(
	t *testing.T,
	registry *Registry,
	bindings []ExecutionBinding,
	admission RuntimeAdmission,
	schemas SchemaValidator,
	limits ExecutionLimits,
	options ...ExecutorOption,
) *Executor {
	t.Helper()
	executor, err := NewExecutor(registry, bindings, admission, schemas, limits, options...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(executor.Close)
	return executor
}

var acceptingExecutionSchema = SchemaValidatorFunc(func(context.Context, SchemaValidationRequest) error { return nil })

func executionRequest(target Contribution, actor string) ExecutionRequest {
	return ExecutionRequest{
		TargetID: target.ID, ContractVersion: target.ContractVersion,
		Document: EditorDocument{
			SchemaVersion: EditorDocumentSchemaVersion, ContentID: target.ID,
			ContractVersion: target.ContractVersion, Schema: target.Schema,
			StorageVersion: "1", Value: []byte(`{"type":"doc","content":[]}`),
		},
		Permission: PermissionInput{
			ActorFingerprint: actor, PolicyFingerprint: "policy-v1",
			Recheck: PermissionRecheckFunc(func(context.Context, PermissionClaim) error { return nil }),
		},
		ResourceID: "topic:42", Locale: "zh-CN", Scope: "public",
	}
}

func executionBinding(target Contribution, declarationID, action string, priority int, providers ProviderSet) ExecutionBinding {
	return ExecutionBinding{
		TargetID: target.ID, TargetContractVersion: target.ContractVersion,
		DeclarationID: declarationID, Action: action,
		Priority: priority, Providers: providers,
	}
}

func staticExecutionRenderer(label string) RendererProvider {
	return RendererProviderFunc(func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
		return executionRender(request.Target, `<p>`+label+`</p>`), nil
	})
}

func executionRender(target Contribution, html string) RenderSegments {
	return RenderSegments{
		SchemaVersion: RenderSegmentsSchemaVersion, ContentID: target.ID,
		ContractVersion: target.ContractVersion,
		Segments:        []RenderSegment{{Kind: SegmentHTML, HTML: html}},
	}
}

func renderHTML(render RenderSegments) string {
	var result strings.Builder
	for _, segment := range render.Segments {
		result.WriteString(segment.HTML)
		result.WriteString(segment.Text)
	}
	return result.String()
}

func TestExecutionNonCooperativeTimeoutQuarantinesExactRuntimeAndRetainsLease(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "bounded.content.block.card", ContractVersion: "bounded.content.block.card@1", Kind: KindBlock,
			Handler: "card", Schema: "bounded.content.schema@1"},
	)
	var active atomic.Int64
	block := make(chan struct{})
	var closeBlock sync.Once
	t.Cleanup(func() { closeBlock.Do(func() { close(block) }) })
	renderer := RendererProviderFunc(func(context.Context, RendererProviderRequest) (RenderSegments, error) {
		active.Add(1)
		defer active.Add(-1)
		<-block
		return executionRender(target, "<p>late</p>"), nil
	})
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: renderer})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	released := make(chan struct{})
	admission := &executionTestAdmission{acquire: func(ctx context.Context, _ AdmissionRequest) (AdmissionLease, error) {
		return &executionTrackedLease{ctx: ctx, released: released}, nil
	}}
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, admission, acceptingExecutionSchema,
		ExecutionLimits{CallTimeout: 10 * time.Millisecond, MaxConcurrentCalls: 2})
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if !errors.Is(err, ErrExecutionTimeout) || !errors.Is(err, ErrRuntimeQuarantined) ||
		!reflect.DeepEqual(result, ExecutionResult{}) {
		t.Fatalf("timed out call result=%#v error=%v", result, err)
	}
	select {
	case <-released:
		t.Fatal("runtime lease released while provider code was still executing")
	default:
	}
	if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrRuntimeUnavailable) || !errors.Is(err, ErrRuntimeQuarantined) {
		t.Fatalf("quarantined exact runtime call = %v", err)
	}
	if active.Load() != 1 || len(admission.snapshot()) != 1 {
		t.Fatalf("active ignored-context calls=%d admissions=%#v", active.Load(), admission.snapshot())
	}
	closeBlock.Do(func() { close(block) })
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("runtime lease was not released after provider code exited")
	}
	if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrRuntimeQuarantined) {
		t.Fatalf("late completion reopened quarantined runtime = %v", err)
	}
}

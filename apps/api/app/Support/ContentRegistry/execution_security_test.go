package contentregistry

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

func TestExecutorSanitizesRendererAndFilterXSS(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "xss.content.block.card", ContractVersion: "xss.content.block.card@1", Kind: KindBlock, Handler: "card", Schema: "xss.content.schema@1"},
		Declaration{ID: "xss.content.filter.final", ContractVersion: "xss.content.filter.final@1", Kind: KindRenderFilter, Handler: "filter", Schema: "xss.content.filter.schema@1"},
	)
	filter, _ := registry.Resolve("xss.content.filter.final")
	renderer := RendererProviderFunc(func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
		return executionRender(request.Target, `<p onclick="alert(1)">ok<script>alert(2)</script><img src="x" onerror="alert(3)"><a href="javascript:alert(4)">link</a></p>`), nil
	})
	filterProvider := FilterProviderFunc(func(_ context.Context, request FilterProviderRequest) (RenderSegments, error) {
		request.Render.Segments = append(request.Render.Segments, RenderSegment{
			Kind: SegmentHTML, HTML: `<svg onload="alert(5)"></svg><iframe srcdoc="bad"></iframe><strong>kept</strong>`,
		})
		return request.Render, nil
	})
	bindings := []ExecutionBinding{
		{TargetID: target.ID, TargetContractVersion: target.ContractVersion, DeclarationID: target.ID, ContractVersion: target.ContractVersion, Artifact: target.Artifact,
			Action: ActionAdd, Fallback: FallbackClosed, Providers: ProviderSet{Renderer: renderer}},
		{TargetID: target.ID, TargetContractVersion: target.ContractVersion, DeclarationID: filter.ID, ContractVersion: filter.ContractVersion, Artifact: filter.Artifact,
			Action: ActionFilter, Fallback: FallbackClosed, Providers: ProviderSet{Filter: filterProvider}},
	}
	executor := newExecutionTestExecutor(t, registry, bindings, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if err != nil {
		t.Fatal(err)
	}
	html := strings.ToLower(renderHTML(result.Render))
	for _, attack := range []string{"<script", "onclick", "onerror", "javascript:", "<svg", "<iframe", "srcdoc"} {
		if strings.Contains(html, attack) {
			t.Fatalf("sanitized HTML retained %q: %s", attack, html)
		}
	}
	if !strings.Contains(html, "<strong>kept</strong>") || result.Render.PlainText != "ok link kept" {
		t.Fatalf("safe output/plain text = %#v", result.Render)
	}
}

func TestExecutorEscapesTextSegmentsExactlyOnceForSSR(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "textxss.content.block.card", ContractVersion: "textxss.content.block.card@1", Kind: KindBlock, Handler: "card", Schema: "textxss.content.schema@1"},
		Declaration{ID: "textxss.content.filter.final", ContractVersion: "textxss.content.filter.final@1", Kind: KindRenderFilter, Handler: "filter", Schema: "textxss.content.filter.schema@1"},
	)
	filter, _ := registry.Resolve("textxss.content.filter.final")
	renderer := RendererProviderFunc(func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
		return RenderSegments{
			SchemaVersion: RenderSegmentsSchemaVersion, ContentID: request.Target.ID,
			ContractVersion: request.Target.ContractVersion,
			Segments: []RenderSegment{
				{Kind: SegmentText, Text: `<script>alert(1)</script>&"'`},
				{Kind: SegmentHTML, HTML: `<strong>safe</strong><script>alert(2)</script>`},
			},
		}, nil
	})
	filterProvider := FilterProviderFunc(func(_ context.Context, request FilterProviderRequest) (RenderSegments, error) {
		request.Render.Segments = append(request.Render.Segments,
			RenderSegment{Kind: SegmentUnsupported, Text: `<img src=x onerror=alert(3)>`},
			RenderSegment{Kind: SegmentText, Text: `&lt;already&gt;`},
		)
		return request.Render, nil
	})
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{
		{TargetID: target.ID, TargetContractVersion: target.ContractVersion, DeclarationID: target.ID, ContractVersion: target.ContractVersion, Artifact: target.Artifact,
			Action: ActionAdd, Fallback: FallbackClosed, Providers: ProviderSet{Renderer: renderer}},
		{TargetID: target.ID, TargetContractVersion: target.ContractVersion, DeclarationID: filter.ID, ContractVersion: filter.ContractVersion, Artifact: filter.Artifact,
			Action: ActionFilter, Fallback: FallbackClosed, Providers: ProviderSet{Filter: filterProvider}},
	}, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	result, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`&lt;script&gt;alert(1)&lt;/script&gt;&amp;&#34;&#39;`,
		"",
		`&lt;img src=x onerror=alert(3)&gt;`,
		`&amp;lt;already&amp;gt;`,
	}
	if result.Render.TextEncoding != RenderTextEncodingHTMLEscaped || len(result.Render.Segments) != len(want) {
		t.Fatalf("SSR text contract = %#v", result.Render)
	}
	for index, segment := range result.Render.Segments {
		if segment.Text != want[index] {
			t.Fatalf("segment %d escaped text = %q, want %q", index, segment.Text, want[index])
		}
	}
	if result.Render.Segments[1].HTML != `<strong>safe</strong>` ||
		result.Render.PlainText != `<script>alert(1)</script>&"' safe <img src=x onerror=alert(3)> &lt;already&gt;` {
		t.Fatalf("sanitized HTML/plain extraction = %#v", result.Render)
	}
	joined := renderHTML(result.Render)
	if strings.Contains(joined, "<script") || strings.Contains(joined, "<img") ||
		strings.Contains(joined, `&amp;lt;script`) {
		t.Fatalf("SSR concatenation is unsafe or double escaped: %s", joined)
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ExecutionResult
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Render.TextEncoding != RenderTextEncodingHTMLEscaped || decoded.Render.Segments[0].Text != want[0] {
		t.Fatalf("JSON round trip changed text encoding: %s", body)
	}
}

func TestExecutorPermissionDenialPrecedesAdmissionAndProvider(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "denied.content.block.card", ContractVersion: "denied.content.block.card@1", Kind: KindBlock, Handler: "card", Schema: "denied.content.schema@1"},
	)
	var providerCalls atomic.Int64
	binding := ExecutionBinding{
		TargetID: target.ID, TargetContractVersion: target.ContractVersion,
		DeclarationID: target.ID, ContractVersion: target.ContractVersion,
		Artifact: target.Artifact, Action: ActionAdd, Fallback: FallbackClosed,
		Providers: ProviderSet{Renderer: RendererProviderFunc(func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
			providerCalls.Add(1)
			return executionRender(request.Target, "<p>secret</p>"), nil
		})},
	}
	admission := &executionTestAdmission{}
	var schemaCalls atomic.Int64
	schemas := SchemaValidatorFunc(func(context.Context, SchemaValidationRequest) error {
		schemaCalls.Add(1)
		return nil
	})
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, admission, schemas, ExecutionLimits{})
	request := executionRequest(target, "unauthorized")
	request.Document.Value = []byte(`not-json`)
	var deniedClaim PermissionClaim
	request.Permission.Recheck = PermissionRecheckFunc(func(_ context.Context, claim PermissionClaim) error {
		deniedClaim = claim
		return ErrExecutionDenied
	})
	if _, err := executor.Execute(t.Context(), request); !errors.Is(err, ErrExecutionDenied) {
		t.Fatalf("denied execution = %v", err)
	}
	if providerCalls.Load() != 0 || schemaCalls.Load() != 0 || len(admission.snapshot()) != 0 {
		t.Fatalf("denied provider calls=%d schema calls=%d admissions=%#v",
			providerCalls.Load(), schemaCalls.Load(), admission.snapshot())
	}
	if deniedClaim.Operation != OperationSource || deniedClaim.TargetID != target.ID ||
		deniedClaim.TargetContractVersion != target.ContractVersion || deniedClaim.TargetSchema != target.Schema ||
		deniedClaim.TargetArtifact != target.Artifact {
		t.Fatalf("denied exact target claim = %#v", deniedClaim)
	}
}

func TestExecutorRejectsProviderContractMismatch(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "mismatch.content.block.card", ContractVersion: "mismatch.content.block.card@1", Kind: KindBlock, Handler: "card", Schema: "mismatch.content.schema@1"},
	)
	binding := ExecutionBinding{
		TargetID: target.ID, TargetContractVersion: target.ContractVersion,
		DeclarationID: target.ID, ContractVersion: target.ContractVersion,
		Artifact: target.Artifact, Action: ActionAdd, Fallback: FallbackClosed,
		Providers: ProviderSet{Renderer: RendererProviderFunc(func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
			result := executionRender(request.Target, "<p>wrong contract</p>")
			result.ContractVersion = "mismatch.content.block.card@2"
			return result, nil
		})},
	}
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrContractStale) {
		t.Fatalf("contract mismatch = %v", err)
	}
}

func TestExecutorStructurallyRevalidatesEveryFilterResult(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "revalidate.content.block.card", ContractVersion: "revalidate.content.block.card@1", Kind: KindBlock, Handler: "card", Schema: "revalidate.content.schema@1"},
		Declaration{ID: "revalidate.content.filter.final", ContractVersion: "revalidate.content.filter.final@1", Kind: KindRenderFilter, Handler: "filter", Schema: "revalidate.content.filter.schema@1"},
	)
	filter, _ := registry.Resolve("revalidate.content.filter.final")
	bindings := []ExecutionBinding{
		{TargetID: target.ID, TargetContractVersion: target.ContractVersion, DeclarationID: target.ID, ContractVersion: target.ContractVersion, Artifact: target.Artifact,
			Action: ActionAdd, Fallback: FallbackClosed, Providers: ProviderSet{Renderer: staticExecutionRenderer("base")}},
		{TargetID: target.ID, TargetContractVersion: target.ContractVersion, DeclarationID: filter.ID, ContractVersion: filter.ContractVersion, Artifact: filter.Artifact,
			Action: ActionFilter, Fallback: FallbackClosed, Providers: ProviderSet{Filter: FilterProviderFunc(func(_ context.Context, request FilterProviderRequest) (RenderSegments, error) {
				request.Render.Segments = append(request.Render.Segments, RenderSegment{Kind: SegmentText, Text: "invalid-filter-output"})
				request.Render.ContractVersion = "revalidate.content.block.card@2"
				return request.Render, nil
			})}},
	}
	executor := newExecutionTestExecutor(t, registry, bindings, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrContractStale) {
		t.Fatalf("filter structural rejection = %v", err)
	}
}

func TestExecutorRejectsStaleRuntimeAtResultRelease(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "release.content.block.card", ContractVersion: "release.content.block.card@1", Kind: KindBlock, Handler: "card", Schema: "release.content.schema@1"},
	)
	binding := ExecutionBinding{
		TargetID: target.ID, TargetContractVersion: target.ContractVersion,
		DeclarationID: target.ID, ContractVersion: target.ContractVersion,
		Artifact: target.Artifact, Action: ActionAdd, Fallback: FallbackClosed,
		Providers: ProviderSet{Renderer: staticExecutionRenderer("must-not-release")},
	}
	admission := &executionTestAdmission{acquire: func(ctx context.Context, request AdmissionRequest) (AdmissionLease, error) {
		if request.Operation == OperationRelease {
			return nil, errors.New("runtime replaced")
		}
		return &executionTestLease{ctx: ctx}, nil
	}}
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, admission, acceptingExecutionSchema, ExecutionLimits{})
	if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrRuntimeUnavailable) {
		t.Fatalf("stale release = %v", err)
	}
	requests := admission.snapshot()
	if len(requests) != 2 || requests[0].Operation != OperationRenderer || requests[1].Operation != OperationRelease {
		t.Fatalf("admission sequence = %#v", requests)
	}
}

func TestExecutorEnforcesInputDepthAndOutputBounds(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "limits.content.block.card", ContractVersion: "limits.content.block.card@1", Kind: KindBlock, Handler: "card", Schema: "limits.content.schema@1"},
	)
	binding := ExecutionBinding{
		TargetID: target.ID, TargetContractVersion: target.ContractVersion,
		DeclarationID: target.ID, ContractVersion: target.ContractVersion,
		Artifact: target.Artifact, Action: ActionAdd, Fallback: FallbackClosed,
		Providers: ProviderSet{Renderer: RendererProviderFunc(func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
			return executionRender(request.Target, strings.Repeat("x", 2048)), nil
		})},
	}
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema,
		ExecutionLimits{MaxJSONDepth: 4, MaxOutputBytes: 1024})
	deep := executionRequest(target, "actor")
	deep.Document.Value = []byte(`{"a":{"b":{"c":{"d":{"e":1}}}}}`)
	if _, err := executor.Execute(t.Context(), deep); !errors.Is(err, ErrExecutionLimit) {
		t.Fatalf("deep input = %v", err)
	}
	if _, err := executor.Execute(t.Context(), executionRequest(target, "actor")); !errors.Is(err, ErrExecutionLimit) {
		t.Fatalf("large output = %v", err)
	}
}

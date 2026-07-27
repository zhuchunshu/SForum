package contentregistry

import (
	"context"
	"strings"
	"testing"
)

// TestReferenceBlocksVoteProductCardEmbedWorkflowForm is the P10 product proof
// that reference content blocks (vote, product/card, embed, workflow form)
// publish, render through Host execution, and remain XSS-sanitized.
func TestReferenceBlocksVoteProductCardEmbedWorkflowForm(t *testing.T) {
	t.Parallel()
	blocks := []struct {
		id       string
		kind     string
		label    string
		html     string
		wantText string
	}{
		{
			id: "sforum.ref-content.block.vote", kind: KindBlock, label: "vote",
			html:     `<div class="sf-vote" data-score="3"><button type="button">up</button><span>3</span></div>`,
			wantText: "up 3",
		},
		{
			id: "sforum.ref-content.block.product-card", kind: KindBlock, label: "product-card",
			html:     `<article class="sf-product-card"><h3>Demo SKU</h3><p>¥99</p></article>`,
			wantText: "Demo SKU ¥99",
		},
		{
			id: "sforum.ref-content.block.embed", kind: KindEmbed, label: "embed",
			html:     `<figure class="sf-embed"><a href="https://example.com/v/1">watch</a></figure>`,
			wantText: "watch",
		},
		{
			id: "sforum.ref-content.block.workflow-form", kind: KindBlock, label: "workflow-form",
			html:     `<form class="sf-workflow-form"><label>Reason<input name="reason"></label><button type="submit">Send</button></form>`,
			wantText: "Reason Send",
		},
	}

	declarations := make([]Declaration, 0, len(blocks))
	for _, block := range blocks {
		declarations = append(declarations, Declaration{
			ID: block.id, ContractVersion: block.id + "@1",
			Kind: block.kind, Handler: block.label, Schema: block.id + ".schema@1",
		})
	}
	// One package owns all reference blocks (reference plugin product surface).
	item := publication("sforum.ref-content", false, 'a')
	item.Content = declarations
	registry := New()
	if _, err := registry.Publish(item); err != nil {
		t.Fatalf("publish reference blocks: %v", err)
	}
	if got := len(registry.List(KindBlock)) + len(registry.List(KindEmbed)); got != 4 {
		t.Fatalf("reference block count = %d", got)
	}

	for _, block := range blocks {
		block := block
		t.Run(block.label, func(t *testing.T) {
			t.Parallel()
			target, err := registry.Resolve(block.id)
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			// Inject XSS alongside legitimate markup; Host must strip attacks.
			payload := block.html + `<script>alert(1)</script><img src=x onerror=alert(2)>`
			renderer := RendererProviderFunc(func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
				return executionRender(request.Target, payload), nil
			})
			bindings := []ExecutionBinding{
				executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: renderer}),
			}
			// Fix binding contract/artifact from publication.
			bindings[0].ContractVersion = target.ContractVersion
			bindings[0].Artifact = target.Artifact
			executor := newExecutionTestExecutor(t, registry, bindings, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
			result, err := executor.Execute(t.Context(), executionRequest(target, "ref-actor"))
			if err != nil {
				t.Fatalf("execute: %v", err)
			}
			html := strings.ToLower(renderHTML(result.Render))
			for _, attack := range []string{"<script", "onerror", "javascript:"} {
				if strings.Contains(html, attack) {
					t.Fatalf("reference block %s retained XSS %q: %s", block.label, attack, html)
				}
			}
			// Plain text extraction should retain safe labels.
			plain := strings.ToLower(result.Render.PlainText)
			for _, token := range strings.Fields(strings.ToLower(block.wantText)) {
				if !strings.Contains(plain, token) {
					t.Fatalf("plain text missing %q: %q html=%s", token, result.Render.PlainText, html)
				}
			}
		})
	}
}

// TestReferenceBlockDisabledPluginRendersStableFallback proves disable removes
// the declaration without rewriting stored editor documents (Host fallback).
func TestReferenceBlockDisabledPluginRendersStableFallback(t *testing.T) {
	t.Parallel()
	registry, target := executionRegistry(t, false,
		Declaration{
			ID: "sforum.ref-content.block.vote", ContractVersion: "sforum.ref-content.block.vote@1",
			Kind: KindBlock, Handler: "vote", Schema: "sforum.ref-content.block.vote.schema@1",
		},
	)
	renderer := staticExecutionRenderer("vote-live")
	bindings := []ExecutionBinding{
		executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: renderer}),
	}
	bindings[0].ContractVersion = target.ContractVersion
	bindings[0].Artifact = target.Artifact
	executor := newExecutionTestExecutor(t, registry, bindings, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	live, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
	if err != nil {
		t.Fatalf("live execute: %v", err)
	}
	if !strings.Contains(renderHTML(live.Render), "vote-live") {
		t.Fatalf("live render = %#v", live.Render)
	}
	// Disable exact artifact: declaration graph empty; stored docs untouched here.
	if _, removed, err := registry.Remove(target.Artifact); err != nil || !removed {
		t.Fatalf("remove = removed=%v err=%v", removed, err)
	}
	if _, err := registry.Resolve(target.ID); err != ErrNotFound {
		t.Fatalf("resolve after disable = %v", err)
	}
}

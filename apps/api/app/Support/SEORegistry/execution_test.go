package seoregistry

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestExecutionAppliesEveryTypedSEOFamilyWithExplicitFallback(t *testing.T) {
	publication := testPublication("plugin.complete", 'a')
	scope := "core.page.topic"
	declarations := []Declaration{
		testDeclaration(publication, "title_add", scope, KindTitle, ActionAdd, FailurePolicyFailClosed, 20),
		testDeclaration(publication, "title_filter", scope, KindTitle, ActionFilter, FailurePolicyFailClosed, 10),
		testDeclaration(publication, "meta_add", scope, KindMeta, ActionAdd, FailurePolicyFailClosed, 10),
		testDeclaration(publication, "canonical_primary", scope, KindCanonical, ActionReplace, FailurePolicyFallback, 20),
		testDeclaration(publication, "canonical_fallback", scope, KindCanonical, ActionReplace, FailurePolicyFailClosed, 10),
		testDeclaration(publication, "robots_add", scope, KindRobots, ActionAdd, FailurePolicyFailClosed, 10),
		testDeclaration(publication, "hreflang_add", scope, KindHreflang, ActionAdd, FailurePolicyFailClosed, 10),
		testDeclaration(publication, "sitemap_add", scope, KindSitemap, ActionAdd, FailurePolicyFailClosed, 10),
		testDeclaration(publication, "jsonld_add", scope, KindJSONLD, ActionAdd, FailurePolicyFailClosed, 10),
	}
	publication.Contributions = declarations
	registry, contributions := publishForExecution(t, publication)
	admission := newTestAdmission()
	trace := NewExecutionTraceRing(8)
	providers := make([]ProviderBinding, 0, len(declarations))
	for _, declaration := range declarations {
		declaration := declaration
		contribution := contributionByID(t, contributions, declaration.ID)
		provider := ProviderFunc(func(_ context.Context, request ProviderRequest) (ProviderResult, error) {
			if !admission.active(publication.Artifact) {
				t.Fatal("provider executed without active exact-artifact lease")
			}
			document := request.Current
			switch declaration.ID {
			case declarations[0].ID:
				document.Title = "Extension title"
			case declarations[1].ID:
				document.Title += " | SForum"
			case declarations[2].ID:
				document.Meta = append(document.Meta, MetaTag{Attribute: "property", Key: "og:type", Content: "article"})
			case declarations[3].ID:
				return ProviderResult{}, errors.New("primary unavailable")
			case declarations[4].ID:
				document.CanonicalURL = "https://forum.example/topics/42"
			case declarations[5].ID:
				document.Robots = RobotsDirectives{Indexing: RobotsIndex, Following: RobotsFollow}
			case declarations[6].ID:
				document.Hreflang = append(document.Hreflang,
					HreflangLink{Locale: "zh-CN", URL: "https://forum.example/zh-CN/topics/42"},
					HreflangLink{Locale: "en-US", URL: "https://forum.example/en-US/topics/42"},
				)
			case declarations[7].ID:
				priority := 0.8
				document.Sitemap = append(document.Sitemap, SitemapEntry{
					URL: "https://forum.example/topics/42", LastModified: "2026-07-16T00:00:00Z",
					ChangeFrequency: SitemapWeekly, Priority: &priority,
				})
			case declarations[8].ID:
				document.JSONLD = append(document.JSONLD, JSONLDDocument{
					Context: "https://schema.org", Type: "DiscussionForumPosting",
					ID: "https://forum.example/topics/42#post", URL: "https://forum.example/topics/42",
					Headline: "Extension title", DatePublished: "2026-07-16T00:00:00Z",
					Author: []JSONLDParty{{Type: "Person", Name: "Alice", URL: "https://forum.example/users/alice"}},
				})
			}
			return ProviderResult{Document: document}, nil
		})
		providers = append(providers, testBinding(publication, contribution.Declaration, provider))
	}
	runtime := mustRuntime(t, registry, admission, providers, trace)
	result, err := runtime.Execute(context.Background(), ExecuteRequest{Scope: scope})
	if err != nil {
		t.Fatal(err)
	}
	if result.Document.Title != "Extension title | SForum" ||
		result.Document.CanonicalURL != "https://forum.example/topics/42" ||
		len(result.Document.Meta) != 1 || len(result.Document.Hreflang) != 2 ||
		len(result.Document.Sitemap) != 1 || len(result.Document.JSONLD) != 1 ||
		result.Document.Robots.Indexing != RobotsIndex {
		t.Fatalf("typed result=%#v", result.Document)
	}
	if len(result.Applied) != 8 || len(result.Fallbacks) != 1 || result.Fallbacks[0].Reason != "provider_failed" {
		t.Fatalf("execution evidence applied=%#v fallbacks=%#v", result.Applied, result.Fallbacks)
	}
	if admission.calls.Load() != 1 || !admission.released(publication.Artifact) {
		t.Fatalf("lease calls=%d released=%t", admission.calls.Load(), admission.released(publication.Artifact))
	}
	inspection, err := runtime.Inspect(scope)
	if err != nil || len(inspection.Providers) != len(declarations) {
		t.Fatalf("runtime inspection=%#v err=%v", inspection, err)
	}
	for _, provider := range inspection.Providers {
		if !provider.Bound || provider.ProviderDigest == "" {
			t.Fatalf("unbound inspection provider=%#v", provider)
		}
	}
	traces := trace.SEOExecutionTraces(1)
	if len(traces) != 1 || traces[0].Outcome != TraceOutcomeApplied || len(traces[0].Calls) != 9 ||
		traces[0].Calls[0].ArtifactDigest != publication.Artifact.PackageDigest {
		t.Fatalf("execution trace=%#v", traces)
	}
}

func TestExecutionProviderDeadlineUsesOnlyDeclaredFallback(t *testing.T) {
	publication := testPublication("plugin.timeout", 'a')
	primary := testDeclaration(publication, "primary", "core.page.topic", KindCanonical, ActionReplace, FailurePolicyFallback, 20)
	primary.Timeout = 5 * time.Millisecond
	fallback := testDeclaration(publication, "fallback", "core.page.topic", KindCanonical, ActionReplace, FailurePolicyFailClosed, 10)
	publication.Contributions = []Declaration{primary, fallback}
	registry, contributions := publishForExecution(t, publication)
	bindings := []ProviderBinding{
		testBinding(publication, contributionByID(t, contributions, primary.ID).Declaration,
			ProviderFunc(func(ctx context.Context, _ ProviderRequest) (ProviderResult, error) {
				<-ctx.Done()
				return ProviderResult{}, ctx.Err()
			})),
		testBinding(publication, contributionByID(t, contributions, fallback.ID).Declaration,
			ProviderFunc(func(_ context.Context, request ProviderRequest) (ProviderResult, error) {
				request.Current.CanonicalURL = "https://forum.example/fallback"
				return ProviderResult{Document: request.Current}, nil
			})),
	}
	runtime := mustRuntime(t, registry, newTestAdmission(), bindings, nil)
	result, err := runtime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Document.CanonicalURL != "https://forum.example/fallback" || len(result.Fallbacks) != 1 ||
		result.Fallbacks[0].Reason != "deadline" {
		t.Fatalf("deadline fallback=%#v", result)
	}

	primary.FailurePolicy = FailurePolicyFailClosed
	publication.Contributions = []Declaration{primary, fallback}
	closedRegistry, closedContributions := publishForExecution(t, publication)
	bindings[0] = testBinding(publication, contributionByID(t, closedContributions, primary.ID).Declaration, bindings[0].Provider)
	bindings[1] = testBinding(publication, contributionByID(t, closedContributions, fallback.ID).Declaration, bindings[1].Provider)
	closedRuntime := mustRuntime(t, closedRegistry, newTestAdmission(), bindings, nil)
	closed, err := closedRuntime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic"})
	if !errors.Is(err, ErrProviderDeadline) || !zeroExecuteResult(closed) {
		t.Fatalf("fail-closed deadline result=%#v err=%v", closed, err)
	}
}

func TestExecutionRequiresLeaseAndFinalExactSnapshotFence(t *testing.T) {
	publication := testPublication("plugin.fence", 'a')
	declaration := testDeclaration(publication, "title", "core.page.topic", KindTitle, ActionAdd, FailurePolicyFailClosed, 0)
	publication.Contributions = []Declaration{declaration}
	registry, contributions := publishForExecution(t, publication)
	contribution := contributionByID(t, contributions, declaration.ID)
	providerCalled := false
	binding := testBinding(publication, contribution.Declaration, ProviderFunc(func(_ context.Context, request ProviderRequest) (ProviderResult, error) {
		providerCalled = true
		request.Current.Title = "must not run"
		return ProviderResult{Document: request.Current}, nil
	}))
	missingLease := ExecutionAdmissionFunc(func(context.Context, Artifact) (AdmissionLease, error) { return nil, nil })
	runtime := mustRuntime(t, registry, missingLease, []ProviderBinding{binding}, nil)
	if result, err := runtime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic"}); !errors.Is(err, ErrArtifactUnavailable) || !zeroExecuteResult(result) || providerCalled {
		t.Fatalf("missing lease result=%#v called=%t err=%v", result, providerCalled, err)
	}

	replacement := testPublication("plugin.fence", 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Artifact.VersionID = 2
	replacement.Artifact.RuntimeInstanceID = "runtime-plugin-fence-v2"
	replacement.Contributions = []Declaration{
		testDeclaration(replacement, "title", "core.page.topic", KindTitle, ActionAdd, FailurePolicyFailClosed, 0),
	}
	admission := newTestAdmission()
	binding = testBinding(publication, contribution.Declaration, ProviderFunc(func(_ context.Context, request ProviderRequest) (ProviderResult, error) {
		if !admission.active(publication.Artifact) {
			t.Fatal("lease was not active inside provider")
		}
		if _, err := registry.PublishIfArtifact(publication.Artifact, replacement); err != nil {
			t.Fatal(err)
		}
		request.Current.Title = "stale"
		return ProviderResult{Document: request.Current}, nil
	}))
	runtime = mustRuntime(t, registry, admission, []ProviderBinding{binding}, nil)
	result, err := runtime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic"})
	if !errors.Is(err, ErrSnapshotStale) || !zeroExecuteResult(result) || !admission.released(publication.Artifact) {
		t.Fatalf("snapshot fence result=%#v released=%t err=%v", result, admission.released(publication.Artifact), err)
	}
}

func TestExecutionSafeModeAndNoContributionReturnDetachedBase(t *testing.T) {
	registry := New()
	runtime := mustRuntime(t, registry, newTestAdmission(), nil, nil)
	base := Document{Meta: []MetaTag{{Attribute: "name", Key: "description", Content: "base"}}}
	result, err := runtime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic", Base: base})
	if err != nil {
		t.Fatal(err)
	}
	result.Document.Meta[0].Content = "mutated"
	if base.Meta[0].Content != "base" {
		t.Fatal("result aliases caller base")
	}
	if _, err := registry.ReplaceAll([]Publication{{}}, true); err != nil {
		t.Fatal(err)
	}
	safe, err := runtime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic", Base: base})
	if err != nil || safe.SchemaVersion != SchemaVersion || safe.Document.Meta[0].Content != "base" {
		t.Fatalf("safe base=%#v err=%v", safe, err)
	}
}

func TestExecutionSafeModeRunsOnlySealedCoreSEO(t *testing.T) {
	artifact, err := NewCoreArtifact(
		"core.seo", "1.0.0", strings.Repeat("c", 64), strings.Repeat("d", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	publication := Publication{Artifact: artifact}
	declaration := testDeclaration(
		publication, "safe-title", "core.page.topic", KindTitle, ActionAdd, FailurePolicyFailClosed, 0,
	)
	publication.Contributions = []Declaration{declaration}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{publication, {}}, true); err != nil {
		t.Fatal(err)
	}
	inspection, err := registry.Inspect("core.page.topic")
	if err != nil || len(inspection.Contributions) != 1 {
		t.Fatalf("safe mode core inspection=%#v err=%v", inspection, err)
	}
	binding := testBinding(publication, declaration, ProviderFunc(
		func(_ context.Context, request ProviderRequest) (ProviderResult, error) {
			request.Current.Title = "Core safe title"
			return ProviderResult{Document: request.Current}, nil
		},
	))
	runtime := mustRuntime(t, registry, newTestAdmission(), []ProviderBinding{binding}, nil)
	result, err := runtime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic"})
	if err != nil || result.Document.Title != "Core safe title" || len(result.Applied) != 1 {
		t.Fatalf("safe mode core result=%#v err=%v", result, err)
	}
}

func TestExecutionRejectsOversizedTypedOutput(t *testing.T) {
	publication := testPublication("plugin.large", 'a')
	declaration := testDeclaration(publication, "meta", "core.page.topic", KindMeta, ActionAdd, FailurePolicyFailClosed, 0)
	publication.Contributions = []Declaration{declaration}
	registry, contributions := publishForExecution(t, publication)
	binding := testBinding(publication, contributionByID(t, contributions, declaration.ID).Declaration,
		ProviderFunc(func(_ context.Context, request ProviderRequest) (ProviderResult, error) {
			request.Current.Meta = append(request.Current.Meta, MetaTag{
				Attribute: "name", Key: "description", Content: strings.Repeat("x", 2000),
			})
			return ProviderResult{Document: request.Current}, nil
		}))
	runtime, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Admission: newTestAdmission(), FinalPolicy: allowTestSEOFinalPolicy(),
		Providers: []ProviderBinding{binding}, MaximumBytes: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := runtime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic"})
	if !errors.Is(err, ErrOutputTooLarge) || !zeroExecuteResult(result) {
		t.Fatalf("oversized output result=%#v err=%v", result, err)
	}
}

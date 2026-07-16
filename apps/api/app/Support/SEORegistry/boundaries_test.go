package seoregistry

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestRegistryEnforcesPublicationAndContributionBounds(t *testing.T) {
	publications := make([]Publication, 0, maxPublications+1)
	for index := 0; index < maxPublications+1; index++ {
		publications = append(publications, testPublication(fmt.Sprintf("plugin.bound.%03d", index), 'a'))
	}
	if _, err := New().ReplaceAll(publications[:maxPublications], false); err != nil {
		t.Fatalf("maximum publications rejected: %v", err)
	}
	if _, err := New().ReplaceAll(publications, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("publication overflow=%v", err)
	}

	publication := testPublication("plugin.contribution.bound", 'b')
	for index := 0; index < maxContributionsPerPackage+1; index++ {
		publication.Contributions = append(publication.Contributions, testDeclaration(
			publication, fmt.Sprintf("meta_%03d", index), "core.page.topic", KindMeta, ActionFilter,
			FailurePolicyFailClosed, index,
		))
	}
	maximum := publication
	maximum.Contributions = maximum.Contributions[:maxContributionsPerPackage]
	if _, err := New().Publish(maximum); err != nil {
		t.Fatalf("maximum contributions rejected: %v", err)
	}
	if _, err := New().Publish(publication); !errors.Is(err, ErrInvalid) {
		t.Fatalf("contribution overflow=%v", err)
	}
}

func TestExecutionAcquiresEveryExactArtifactBeforeCallbacks(t *testing.T) {
	alpha := testPublication("plugin.lease.alpha", 'a')
	alphaDeclaration := testDeclaration(alpha, "title", "core.page.topic", KindTitle, ActionAdd, FailurePolicyFailClosed, 0)
	alpha.Contributions = []Declaration{alphaDeclaration}
	beta := testPublication("plugin.lease.beta", 'b')
	betaDeclaration := testDeclaration(beta, "meta", "core.page.topic", KindMeta, ActionAdd, FailurePolicyFailClosed, 0)
	beta.Contributions = []Declaration{betaDeclaration}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{beta, alpha}, false); err != nil {
		t.Fatal(err)
	}
	inspection, err := registry.Inspect("core.page.topic")
	if err != nil {
		t.Fatal(err)
	}

	type orderedAdmission struct {
		mu     sync.Mutex
		order  []Artifact
		leases map[Artifact]*testLease
	}
	admission := &orderedAdmission{leases: make(map[Artifact]*testLease)}
	acquire := ExecutionAdmissionFunc(func(ctx context.Context, artifact Artifact) (AdmissionLease, error) {
		lease := &testLease{ctx: ctx}
		admission.mu.Lock()
		admission.order = append(admission.order, artifact)
		admission.leases[artifact] = lease
		admission.mu.Unlock()
		return lease, nil
	})
	allActive := func() bool {
		admission.mu.Lock()
		defer admission.mu.Unlock()
		return len(admission.leases) == 2 && !admission.leases[alpha.Artifact].released.Load() &&
			!admission.leases[beta.Artifact].released.Load()
	}
	provider := func(kind string) ProviderFunc {
		return func(_ context.Context, request ProviderRequest) (ProviderResult, error) {
			if !allActive() {
				t.Fatal("callback began before the complete deterministic lease set was active")
			}
			if kind == KindTitle {
				request.Current.Title = "Leased title"
			} else {
				request.Current.Meta = append(request.Current.Meta, MetaTag{Attribute: "name", Key: "description", Content: "Leased"})
			}
			return ProviderResult{Document: request.Current}, nil
		}
	}
	runtime := mustRuntime(t, registry, acquire, []ProviderBinding{
		testBinding(alpha, contributionByID(t, inspection.Contributions, alphaDeclaration.ID).Declaration, provider(KindTitle)),
		testBinding(beta, contributionByID(t, inspection.Contributions, betaDeclaration.ID).Declaration, provider(KindMeta)),
	}, nil)
	result, err := runtime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic"})
	if err != nil || result.Document.Title != "Leased title" || len(result.Document.Meta) != 1 {
		t.Fatalf("leased execution=%#v err=%v", result, err)
	}
	admission.mu.Lock()
	defer admission.mu.Unlock()
	if !reflect.DeepEqual(admission.order, []Artifact{alpha.Artifact, beta.Artifact}) ||
		!admission.leases[alpha.Artifact].released.Load() || !admission.leases[beta.Artifact].released.Load() {
		t.Fatalf("admission order=%#v leases=%#v", admission.order, admission.leases)
	}
}

func TestExecutionRecoversProviderPanicOnlyThroughExplicitFallback(t *testing.T) {
	publication := testPublication("plugin.panic", 'a')
	primary := testDeclaration(publication, "primary", "core.page.topic", KindCanonical, ActionReplace, FailurePolicyFallback, 20)
	fallback := testDeclaration(publication, "fallback", "core.page.topic", KindCanonical, ActionReplace, FailurePolicyFailClosed, 10)
	publication.Contributions = []Declaration{primary, fallback}
	registry, contributions := publishForExecution(t, publication)
	bindings := []ProviderBinding{
		testBinding(publication, contributionByID(t, contributions, primary.ID).Declaration,
			ProviderFunc(func(context.Context, ProviderRequest) (ProviderResult, error) { panic("provider secret") })),
		testBinding(publication, contributionByID(t, contributions, fallback.ID).Declaration,
			ProviderFunc(func(_ context.Context, request ProviderRequest) (ProviderResult, error) {
				request.Current.CanonicalURL = "https://forum.example/recovered"
				return ProviderResult{Document: request.Current}, nil
			})),
	}
	runtime := mustRuntime(t, registry, newTestAdmission(), bindings, nil)
	result, err := runtime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic"})
	if err != nil || result.Document.CanonicalURL != "https://forum.example/recovered" ||
		len(result.Fallbacks) != 1 || result.Fallbacks[0].Reason != "provider_failed" {
		t.Fatalf("panic fallback=%#v err=%v", result, err)
	}

	primary.FailurePolicy = FailurePolicyFailClosed
	publication.Contributions = []Declaration{primary, fallback}
	closedRegistry, closedContributions := publishForExecution(t, publication)
	bindings[0] = testBinding(publication, contributionByID(t, closedContributions, primary.ID).Declaration, bindings[0].Provider)
	bindings[1] = testBinding(publication, contributionByID(t, closedContributions, fallback.ID).Declaration, bindings[1].Provider)
	closedRuntime := mustRuntime(t, closedRegistry, newTestAdmission(), bindings, nil)
	if closed, err := closedRuntime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic"}); !errors.Is(err, ErrProviderFailed) || !zeroExecuteResult(closed) || strings.Contains(err.Error(), "provider secret") {
		t.Fatalf("panic fail-closed result=%#v err=%v", closed, err)
	}
}

func TestExecutionRequiresHostFinalPolicyAndKeepsPolicyInputDetached(t *testing.T) {
	publication := testPublication("plugin.final-policy", 'a')
	declaration := testDeclaration(
		publication, "title", "core.page.topic", KindTitle, ActionAdd, FailurePolicyFailClosed, 0,
	)
	publication.Contributions = []Declaration{declaration}
	registry, contributions := publishForExecution(t, publication)
	binding := testBinding(publication, contributionByID(t, contributions, declaration.ID).Declaration,
		ProviderFunc(func(_ context.Context, request ProviderRequest) (ProviderResult, error) {
			request.Current.Title = "extension title"
			return ProviderResult{Document: request.Current}, nil
		}))
	if _, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Admission: newTestAdmission(), Providers: []ProviderBinding{binding},
	}); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("missing Host final policy=%v", err)
	}

	admission := newTestAdmission()
	runtime, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Admission: admission, Providers: []ProviderBinding{binding},
		FinalPolicy: FinalPolicyFunc(func(_ context.Context, request FinalPolicyRequest) error {
			if request.Scope != "core.page.topic" || request.Document.Title != "extension title" {
				return errors.New("unexpected final policy input")
			}
			request.Document.Title = "forged by policy"
			return nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := runtime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic"})
	if err != nil || result.Document.Title != "extension title" || !admission.released(publication.Artifact) {
		t.Fatalf("Host final policy result=%#v released=%t err=%v", result, admission.released(publication.Artifact), err)
	}

	trace := NewExecutionTraceRing(1)
	denied, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Admission: newTestAdmission(), Providers: []ProviderBinding{binding},
		Trace: trace,
		FinalPolicy: FinalPolicyFunc(func(context.Context, FinalPolicyRequest) error {
			return errors.New("foreign canonical")
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result, err := denied.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic"}); !errors.Is(err, ErrPolicyDenied) || !zeroExecuteResult(result) {
		t.Fatalf("Host final policy denial result=%#v err=%v", result, err)
	}
	if traces := trace.SEOExecutionTraces(1); len(traces) != 1 || traces[0].Outcome != TraceOutcomePolicyDenied {
		t.Fatalf("Host final policy trace=%#v", traces)
	}
}

func TestExecutionDetachesNestedProviderOutput(t *testing.T) {
	publication := testPublication("plugin.detach", 'a')
	declaration := testDeclaration(publication, "jsonld", "core.page.topic", KindJSONLD, ActionAdd, FailurePolicyFailClosed, 0)
	publication.Contributions = []Declaration{declaration}
	registry, contributions := publishForExecution(t, publication)
	emitted := Document{}
	binding := testBinding(publication, contributionByID(t, contributions, declaration.ID).Declaration,
		ProviderFunc(func(_ context.Context, request ProviderRequest) (ProviderResult, error) {
			publisher := &JSONLDParty{Type: "Organization", Name: "SForum", URL: "https://forum.example/about"}
			request.Current.JSONLD = append(request.Current.JSONLD, JSONLDDocument{
				Context: "https://schema.org", Type: "WebPage", URL: "https://forum.example/topic/1",
				Publisher: publisher, ImageURLs: []string{"https://forum.example/assets/topic.png"},
			})
			emitted = request.Current
			return ProviderResult{Document: request.Current}, nil
		}))
	runtime := mustRuntime(t, registry, newTestAdmission(), []ProviderBinding{binding}, nil)
	result, err := runtime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic"})
	if err != nil {
		t.Fatal(err)
	}
	emitted.JSONLD[0].Publisher.Name = "forged"
	emitted.JSONLD[0].ImageURLs[0] = "https://evil.example/"
	if result.Document.JSONLD[0].Publisher.Name != "SForum" ||
		result.Document.JSONLD[0].ImageURLs[0] != "https://forum.example/assets/topic.png" {
		t.Fatalf("provider output aliases released result=%#v", result.Document.JSONLD)
	}
}

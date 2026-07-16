package seoregistry

import (
	"context"
	"errors"
	"math"
	"testing"
)

func TestDocumentValidationRejectsUnsafeCanonicalAndTypedJSONLD(t *testing.T) {
	valid := Document{
		CanonicalURL: "https://forum.example/topic/1",
		Robots:       RobotsDirectives{Indexing: RobotsIndex, Following: RobotsFollow},
		Hreflang:     []HreflangLink{{Locale: "en-US", URL: "https://forum.example/en-US/topic/1"}},
		JSONLD: []JSONLDDocument{{
			Context: "https://schema.org", Type: "DiscussionForumPosting",
			ID: "https://forum.example/topic/1#post", URL: "https://forum.example/topic/1",
		}},
	}
	if err := validateDocument(valid); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		change func(*Document)
	}{
		{name: "javascript canonical", change: func(value *Document) { value.CanonicalURL = "javascript:alert(1)" }},
		{name: "credentials", change: func(value *Document) { value.CanonicalURL = "https://user:secret@forum.example/topic/1" }},
		{name: "fragment", change: func(value *Document) { value.CanonicalURL = "https://forum.example/topic/1#fragment" }},
		{name: "noncanonical root", change: func(value *Document) { value.CanonicalURL = "https://forum.example" }},
		{name: "uppercase host", change: func(value *Document) { value.CanonicalURL = "https://Forum.example/topic/1" }},
		{name: "bad context", change: func(value *Document) { value.JSONLD[0].Context = "https://evil.example" }},
		{name: "bad type", change: func(value *Document) { value.JSONLD[0].Type = "<script>" }},
		{name: "bad date", change: func(value *Document) { value.JSONLD[0].DatePublished = "today" }},
		{name: "bad breadcrumb", change: func(value *Document) {
			value.JSONLD[0].Breadcrumbs = []JSONLDBreadcrumb{{Type: "ListItem", Position: 2, Name: "Topic", URL: "https://forum.example/topic/1"}}
		}},
		{name: "duplicate hreflang", change: func(value *Document) {
			value.Hreflang = append(value.Hreflang, value.Hreflang[0])
		}},
		{name: "noncanonical hreflang", change: func(value *Document) { value.Hreflang[0].Locale = "en-us" }},
		{name: "invalid robots", change: func(value *Document) { value.Robots.Indexing = "maybe" }},
		{name: "http equiv meta", change: func(value *Document) {
			value.Meta = []MetaTag{{Attribute: "http-equiv", Key: "refresh", Content: "0;url=https://evil.example/"}}
		}},
		{name: "control text", change: func(value *Document) { value.Title = "safe\nunsafe" }},
		{name: "invalid dns label", change: func(value *Document) { value.CanonicalURL = "https://bad.-label.example/topic/1" }},
		{name: "invalid port", change: func(value *Document) { value.CanonicalURL = "https://forum.example:70000/topic/1" }},
		{name: "sitemap nan", change: func(value *Document) {
			priority := math.NaN()
			value.Sitemap = []SitemapEntry{{URL: "https://forum.example/topic/1", Priority: &priority}}
		}},
		{name: "sitemap date", change: func(value *Document) {
			value.Sitemap = []SitemapEntry{{URL: "https://forum.example/topic/1", LastModified: "2026/07/16"}}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := cloneDocument(valid)
			test.change(&candidate)
			if err := validateDocument(candidate); !errors.Is(err, ErrOutputInvalid) {
				t.Fatalf("validation error=%v", err)
			}
		})
	}
}

func TestExecutionRejectsCrossKindMutationAndDoesNotReleasePartialOutput(t *testing.T) {
	publication := testPublication("plugin.mutation", 'a')
	declaration := testDeclaration(publication, "title", "core.page.topic", KindTitle, ActionFilter, FailurePolicyFailClosed, 0)
	publication.Contributions = []Declaration{declaration}
	registry, contributions := publishForExecution(t, publication)
	binding := testBinding(publication, contributionByID(t, contributions, declaration.ID).Declaration,
		ProviderFunc(func(_ context.Context, request ProviderRequest) (ProviderResult, error) {
			request.Current.Title = "changed"
			request.Current.CanonicalURL = "https://forum.example/forbidden"
			return ProviderResult{Document: request.Current}, nil
		}))
	runtime := mustRuntime(t, registry, newTestAdmission(), []ProviderBinding{binding}, nil)
	result, err := runtime.Execute(context.Background(), ExecuteRequest{
		Scope: "core.page.topic", Base: Document{Title: "base", CanonicalURL: "https://forum.example/original"},
	})
	if !errors.Is(err, ErrMutationDenied) || !zeroExecuteResult(result) {
		t.Fatalf("cross-kind result=%#v err=%v", result, err)
	}
}

func TestExecutionRejectsAddThatRewritesExistingList(t *testing.T) {
	publication := testPublication("plugin.rewrite", 'a')
	declaration := testDeclaration(publication, "meta", "core.page.topic", KindMeta, ActionAdd, FailurePolicyFailClosed, 0)
	publication.Contributions = []Declaration{declaration}
	registry, contributions := publishForExecution(t, publication)
	binding := testBinding(publication, contributionByID(t, contributions, declaration.ID).Declaration,
		ProviderFunc(func(_ context.Context, request ProviderRequest) (ProviderResult, error) {
			request.Current.Meta[0].Content = "rewritten"
			request.Current.Meta = append(request.Current.Meta, MetaTag{Attribute: "name", Key: "author", Content: "Alice"})
			return ProviderResult{Document: request.Current}, nil
		}))
	runtime := mustRuntime(t, registry, newTestAdmission(), []ProviderBinding{binding}, nil)
	base := Document{Meta: []MetaTag{{Attribute: "name", Key: "description", Content: "original"}}}
	result, err := runtime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic", Base: base})
	if !errors.Is(err, ErrMutationDenied) || !zeroExecuteResult(result) || base.Meta[0].Content != "original" {
		t.Fatalf("rewrite result=%#v base=%#v err=%v", result, base, err)
	}
}

func TestProviderBindingMustMatchExactContributionArtifactAndHandler(t *testing.T) {
	publication := testPublication("plugin.binding", 'a')
	declaration := testDeclaration(publication, "title", "core.page.topic", KindTitle, ActionAdd, FailurePolicyFailClosed, 0)
	publication.Contributions = []Declaration{declaration}
	registry, contributions := publishForExecution(t, publication)
	contribution := contributionByID(t, contributions, declaration.ID)
	binding := testBinding(publication, contribution.Declaration,
		ProviderFunc(func(_ context.Context, request ProviderRequest) (ProviderResult, error) {
			request.Current.Title = "unexpected"
			return ProviderResult{Document: request.Current}, nil
		}))
	binding.Artifact.RuntimeInstanceID = "different-runtime"
	runtime := mustRuntime(t, registry, newTestAdmission(), []ProviderBinding{binding}, nil)
	result, err := runtime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic"})
	if !errors.Is(err, ErrProviderUnavailable) || !zeroExecuteResult(result) {
		t.Fatalf("mismatched exact binding result=%#v err=%v", result, err)
	}
}

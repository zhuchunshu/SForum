package seoregistry

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSEOProductJSDisabledSourceAndPluginFailure(t *testing.T) {
	base := Document{
		Title:        "帖子标题 | SForum",
		CanonicalURL: "https://forum.example/t/1",
		Meta: []MetaTag{
			{Attribute: "name", Key: "description", Content: "正文摘要，无需 JS 即可被爬虫读取。"},
		},
		Robots:   RobotsDirectives{Indexing: RobotsIndex, Following: RobotsFollow},
		Hreflang: []HreflangLink{{Locale: "en-US", URL: "https://forum.example/en/t/1"}},
		JSONLD: []JSONLDDocument{{
			Context: "https://schema.org", Type: "DiscussionForumPosting",
			Name: "帖子标题", Description: "正文摘要，无需 JS 即可被爬虫读取。",
			URL: "https://forum.example/t/1",
		}},
	}
	view := ToPageSEOView(base)
	if view.Title == "" || view.Description == "" || view.CanonicalURL == "" || view.Robots == "" {
		t.Fatalf("JS-disabled SEO source incomplete: %#v", view)
	}
	if len(view.AlternateLinks) != 1 || len(view.StructuredData) != 1 {
		t.Fatalf("hreflang/jsonld missing from SSR view: %#v", view)
	}
	blob := view.Title + view.Description + view.CanonicalURL + view.Robots
	for _, bad := range []string{"<script", "javascript:", "onerror=", "onload="} {
		if strings.Contains(strings.ToLower(blob), bad) {
			t.Fatalf("unsafe SEO content for JS-disabled consumers: %q in %q", bad, blob)
		}
	}

	// Reuse the proven fixture shape from execution_test (plugin.complete / 'a').
	publication := testPublication("plugin.complete", 'a')
	primary := testDeclaration(publication, "primary", "core.page.topic", KindCanonical, ActionReplace, FailurePolicyFallback, 20)
	fallback := testDeclaration(publication, "fallback", "core.page.topic", KindCanonical, ActionReplace, FailurePolicyFailClosed, 10)
	publication.Contributions = []Declaration{primary, fallback}
	registry, contributions := publishForExecution(t, publication)
	admission := newTestAdmission()
	bindings := []ProviderBinding{
		testBinding(publication, contributionByID(t, contributions, primary.ID).Declaration,
			ProviderFunc(func(context.Context, ProviderRequest) (ProviderResult, error) {
				return ProviderResult{}, errors.New("plugin process crashed")
			})),
		testBinding(publication, contributionByID(t, contributions, fallback.ID).Declaration,
			ProviderFunc(func(_ context.Context, request ProviderRequest) (ProviderResult, error) {
				out := request.Current
				if out.Title == "" {
					out.Title = base.Title
				}
				if out.CanonicalURL == "" {
					out.CanonicalURL = base.CanonicalURL
				}
				return ProviderResult{Document: out}, nil
			})),
	}
	runtime := mustRuntime(t, registry, admission, bindings, nil)
	result, err := runtime.Execute(context.Background(), ExecuteRequest{
		Scope: "core.page.topic",
		Base:  base,
	})
	if err != nil {
		t.Fatalf("fallback policy should recover: %v", err)
	}
	if result.Document.Title != base.Title {
		t.Fatalf("failing plugin overwrote base title: got %q want %q", result.Document.Title, base.Title)
	}
	if len(result.Fallbacks) < 1 {
		t.Fatalf("expected fallback evidence, got %#v", result.Fallbacks)
	}
	surviving := ToPageSEOView(result.Document)
	if surviving.Title == "" || surviving.CanonicalURL == "" {
		t.Fatalf("post-failure SSR SEO incomplete: %#v", surviving)
	}
}

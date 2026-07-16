package seoregistry

import (
	"context"
	"errors"
	"testing"
)

func TestHostFinalPolicyKeepsCorePrivacyAndSiteOwnershipAuthoritative(t *testing.T) {
	policy, err := NewHostFinalPolicy(HostFinalPolicyConfig{
		SiteURL: "https://forum.example", SupportedLocales: []string{"zh-CN", "en-US"},
		AllowIndexing: true, SitemapEnabled: true, StructuredDataEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	base := Document{Robots: RobotsDirectives{
		Indexing: RobotsNoIndex, Following: RobotsNoFollow, NoArchive: true,
	}}
	valid := cloneDocument(base)
	valid.CanonicalURL = "https://forum.example/topic/1"
	valid.Hreflang = []HreflangLink{
		{Locale: "zh-CN", URL: "https://forum.example/zh-CN/topic/1"},
		{Locale: "x-default", URL: "https://forum.example/topic/1"},
	}
	if err := policy.ValidateSEO(context.Background(), FinalPolicyRequest{
		Scope: "core.page.topic", Base: base, Document: valid,
	}); err != nil {
		t.Fatalf("strict same-origin output rejected: %v", err)
	}
	tests := []struct {
		name   string
		change func(*Document)
	}{
		{name: "weaken noindex", change: func(value *Document) { value.Robots.Indexing = RobotsIndex }},
		{name: "weaken nofollow", change: func(value *Document) { value.Robots.Following = RobotsFollow }},
		{name: "weaken noarchive", change: func(value *Document) { value.Robots.NoArchive = false }},
		{name: "foreign canonical", change: func(value *Document) { value.CanonicalURL = "https://evil.example/topic/1" }},
		{name: "foreign hreflang", change: func(value *Document) { value.Hreflang[0].URL = "https://evil.example/topic/1" }},
		{name: "unknown locale", change: func(value *Document) { value.Hreflang[0].Locale = "fr-FR" }},
		{name: "sitemap while noindex", change: func(value *Document) {
			value.Sitemap = []SitemapEntry{{URL: "https://forum.example/topic/1"}}
		}},
		{name: "robots meta bypass", change: func(value *Document) {
			value.Meta = []MetaTag{{Attribute: "name", Key: "robots", Content: "index,follow"}}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := cloneDocument(valid)
			test.change(&candidate)
			if err := policy.ValidateSEO(context.Background(), FinalPolicyRequest{
				Scope: "core.page.topic", Base: base, Document: candidate,
			}); !errors.Is(err, ErrPolicyDenied) {
				t.Fatalf("policy error=%v", err)
			}
		})
	}
}

func TestHostFinalPolicyUsesExistingIndexSitemapAndSchemaSwitches(t *testing.T) {
	policy, err := NewHostFinalPolicy(HostFinalPolicyConfig{
		SiteURL: "https://forum.example", SupportedLocales: []string{"zh-CN"},
	})
	if err != nil {
		t.Fatal(err)
	}
	document := Document{
		Robots:  RobotsDirectives{Indexing: RobotsIndex, Following: RobotsFollow},
		Sitemap: []SitemapEntry{{URL: "https://forum.example/topic/1"}},
		JSONLD:  []JSONLDDocument{{Context: "https://schema.org", Type: "WebPage"}},
	}
	if err := policy.ValidateSEO(context.Background(), FinalPolicyRequest{
		Scope: "core.page.topic", Document: document,
	}); !errors.Is(err, ErrPolicyDenied) {
		t.Fatalf("disabled Host options error=%v", err)
	}
	document.Sitemap, document.JSONLD = nil, nil
	if err := policy.ValidateSEO(context.Background(), FinalPolicyRequest{
		Scope: "core.page.topic", Document: document,
	}); !errors.Is(err, ErrPolicyDenied) {
		t.Fatalf("global noindex error=%v", err)
	}
	document.Robots.Indexing = RobotsNoIndex
	if err := policy.ValidateSEO(context.Background(), FinalPolicyRequest{
		Scope: "core.page.topic", Document: document,
	}); err != nil {
		t.Fatalf("strict disabled output rejected: %v", err)
	}
}

func TestExecutionAlwaysAppliesHostPolicyWithoutPluginContributions(t *testing.T) {
	policy, err := NewHostFinalPolicy(HostFinalPolicyConfig{
		SiteURL: "https://forum.example", SupportedLocales: []string{"zh-CN"},
	})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewExecutionRuntime(ExecutionConfig{
		Registry: New(), Admission: newTestAdmission(), FinalPolicy: policy,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := runtime.Execute(context.Background(), ExecuteRequest{
		Scope: "core.page.topic",
		Base:  Document{Robots: RobotsDirectives{Indexing: RobotsIndex, Following: RobotsFollow}},
	})
	if !errors.Is(err, ErrPolicyDenied) || !zeroExecuteResult(result) {
		t.Fatalf("Core-only final fence result=%#v err=%v", result, err)
	}
}

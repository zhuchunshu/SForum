package options

import "testing"

func TestSEOOptionsV2RecommendedDefaults(t *testing.T) {
	settings := seoOptionsFromValues(map[string]string{
		NameSiteName: "SForum",
	})

	if !settings.InheritSiteName || settings.EffectiveSiteName != "SForum" {
		t.Fatalf("unexpected site identity: %#v", settings)
	}
	if settings.HomeTitle != "SForum" || settings.HomeDescription == "" {
		t.Fatalf("unexpected homepage defaults: %#v", settings)
	}
	if settings.PageTitleTemplate != "{pageTitle} | {seoSiteName}" {
		t.Fatalf("unexpected title template %q", settings.PageTitleTemplate)
	}
	if seoRecommendedDefaults()[NameSEOSitemapIncludeForumContent] != "enabled" {
		t.Fatal("forum sitemap must be enabled by the v2 recommended defaults")
	}
}

func TestSEOOptionsV2UsesIndependentSiteIdentity(t *testing.T) {
	settings := seoOptionsFromValues(map[string]string{
		NameSiteName:                  "SForum",
		NameSEOSiteInheritSiteName:    "disabled",
		NameSEOSiteName:               "SForum Developers",
		NameSEOHomeTitle:              "Developer Q&A",
		NameSEOHomeDescription:        "Questions and open source discussions.",
		NameSEOPageTitleTemplate:      "{pageTitle} - {seoSiteName}",
		NameSEOPageDefaultDescription: "Community discussions.",
	})

	if settings.EffectiveSiteName != "SForum Developers" {
		t.Fatalf("expected independent SEO site name, got %q", settings.EffectiveSiteName)
	}
	if settings.HomeTitle != "Developer Q&A" {
		t.Fatalf("expected independent homepage title, got %q", settings.HomeTitle)
	}
}

func TestSEOOptionsV2RejectUnknownTemplateVariable(t *testing.T) {
	if _, ok := normalizeSEOOption(NameSEOPageTitleTemplate, "{title} | {unknown}"); ok {
		t.Fatal("unknown SEO template variable must be rejected")
	}
	if got, ok := normalizeSEOOption(NameSEOContentTopicTitleTemplate, " {topicTitle} | {seoSiteName} "); !ok || got != "{topicTitle} | {seoSiteName}" {
		t.Fatalf("expected registered topic variables, got %q, %t", got, ok)
	}
}

func TestSEOOptionsV2NormalizesContentPolicyEnums(t *testing.T) {
	if got, ok := normalizeSEOOption(NameSEOContentTopicIndexMode, "INDEX"); !ok || got != "index" {
		t.Fatalf("expected normalized index mode, got %q, %t", got, ok)
	}
	if _, ok := normalizeSEOOption(NameSEOContentTopicIndexMode, "sometimes"); ok {
		t.Fatal("invalid index mode must be rejected")
	}
	if got, ok := normalizeSEOOption(NameSEOContentCategoryDescriptionSource, " category_description, site_default "); !ok || got != "category_description,site_default" {
		t.Fatalf("expected normalized description sources, got %q, %t", got, ok)
	}
}

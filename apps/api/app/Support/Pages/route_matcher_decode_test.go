package pages

import "testing"

func TestCorePagePathDecodesDynamicParametersBySegment(t *testing.T) {
	params, ok := MatchCorePagePath("forum.profile.show", "/u/%E6%9E%97%E7%9F%A5%E5%A4%8F")
	if !ok || params["username"] != "林知夏" {
		t.Fatalf("profile params = %#v, ok=%v", params, ok)
	}

	params, ok = MatchCorePagePath("forum.topic.show", "/t/42/%E4%B8%BB%E9%A2%98")
	if !ok || params["path"] != "42/主题" {
		t.Fatalf("topic params = %#v, ok=%v", params, ok)
	}
}

func TestAddedPagePathDecodesDynamicParameters(t *testing.T) {
	registry := NewRegistry(NewMemoryStore())
	if err := registry.RegisterContributions("docs.plugin", []PageContribution{{
		ID: "docs.show", Action: ActionAdd, Path: "/docs/:slug", Access: AccessPublic,
		ExtensionID: "docs.plugin", Version: "1.0.0", PackageDigest: "digest",
	}}); err != nil {
		t.Fatal(err)
	}

	match, ok := registry.ResolveAddedPathMatch("/docs/%E5%85%A5%E9%97%A8")
	if !ok || match.Params["slug"] != "入门" {
		t.Fatalf("added params = %#v, ok=%v", match.Params, ok)
	}
}

func TestDynamicPathRejectsInvalidEscapesAndEncodedSegmentSeparators(t *testing.T) {
	for _, path := range []string{"/u/%zz", "/u/%FF", "/u/alice%2Fadmin"} {
		if params, ok := MatchCorePagePath("forum.profile.show", path); ok {
			t.Fatalf("path %q unexpectedly matched with %#v", path, params)
		}
	}
	if params, ok := MatchCorePagePath("forum.topic.show", "/t/42/%zz"); ok {
		t.Fatalf("invalid catch-all unexpectedly matched with %#v", params)
	}
}

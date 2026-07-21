package main

import (
	"reflect"
	"testing"
)

func TestParseKeywords(t *testing.T) {
	got := parseKeywords("spam, promo\n# comment\n广告\nspam\n")
	want := []string{"spam", "promo", "广告"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseKeywords: got %#v want %#v", got, want)
	}
	if parseKeywords("") != nil && len(parseKeywords("")) != 0 {
		t.Fatalf("empty should be empty")
	}
}

func TestFindKeywordCaseInsensitive(t *testing.T) {
	if got := findKeyword("Hello SPAM world", []string{"spam"}, false); got != "spam" {
		t.Fatalf("expected spam, got %q", got)
	}
	if got := findKeyword("Hello SPAM world", []string{"spam"}, true); got != "" {
		t.Fatalf("case-sensitive should miss, got %q", got)
	}
}

func TestEvaluateDisabledOrEmpty(t *testing.T) {
	cfg := policyConfig{Enabled: false, Keywords: []string{"x"}, MatchContent: true}
	if d := evaluateContent(cfg, "comment.before_create", "", "x"); !d.OK {
		t.Fatal("disabled must pass")
	}
	cfg = policyConfig{Enabled: true, Keywords: nil, MatchContent: true}
	if d := evaluateContent(cfg, "comment.before_create", "", "x"); !d.OK {
		t.Fatal("empty keywords must pass")
	}
}

func TestEvaluateRejectComment(t *testing.T) {
	cfg := policyConfig{
		Enabled: true, Keywords: []string{"bad"}, Mode: modeReject,
		MatchContent: true,
	}
	d := evaluateContent(cfg, "comment.before_create", "", "this is bad word")
	if d.OK || d.Reason != "content_policy.keyword_blocked" {
		t.Fatalf("expected reject, got %#v", d)
	}
}

func TestEvaluateTagModeTopic(t *testing.T) {
	cfg := policyConfig{
		Enabled: true, Keywords: []string{"promo"}, Mode: modeTag,
		ForceTag: "needs-review", MatchTitle: true, MatchContent: true,
	}
	d := evaluateContent(cfg, "topic.before_create", "promo title", "ok body")
	if !d.OK || d.PatchTag != "needs-review" {
		t.Fatalf("expected tag patch, got %#v", d)
	}
	// 评论在 tag 模式仍拒绝
	d2 := evaluateContent(cfg, "comment.before_create", "", "promo reply")
	if d2.OK || d2.Reason != "content_policy.keyword_blocked" {
		t.Fatalf("comment must reject in tag mode, got %#v", d2)
	}
}

func TestEvaluateTagModeMissingForceTag(t *testing.T) {
	cfg := policyConfig{
		Enabled: true, Keywords: []string{"x"}, Mode: modeTag,
		ForceTag: "", MatchContent: true,
	}
	d := evaluateContent(cfg, "topic.before_create", "", "x")
	if d.OK || d.Reason != "content_policy.force_tag_missing" {
		t.Fatalf("expected force_tag_missing, got %#v", d)
	}
}

func TestMergeTagSlugs(t *testing.T) {
	got := mergeTagSlugs([]any{"a", "b"}, "b")
	if !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("dedupe: %#v", got)
	}
	got = mergeTagSlugs([]string{"a"}, "needs-review")
	if !reflect.DeepEqual(got, []string{"a", "needs-review"}) {
		t.Fatalf("append: %#v", got)
	}
}

func TestNormalizeMode(t *testing.T) {
	if normalizeMode("tag") != modeTag {
		t.Fatal("tag")
	}
	if normalizeMode("weird") != modeReject {
		t.Fatal("default reject")
	}
}

func TestPayloadStringContentObject(t *testing.T) {
	// Protocol V2 规范化后 content 是 ContentInput 形状的 object。
	payload := map[string]any{
		"content": map[string]any{
			"rawContent":   "<p>blocked word</p>",
			"sourceFormat": "html",
			"plainText":    "blocked word",
		},
		"title": "hello",
	}
	if got := payloadString(payload, "title"); got != "hello" {
		t.Fatalf("title string: got %q", got)
	}
	if got := payloadString(payload, "content"); got != "blocked word" {
		t.Fatalf("prefer plainText: got %q", got)
	}
	// 无 plainText 时回退 rawContent。
	payload["content"] = map[string]any{"rawContent": "raw only"}
	if got := payloadString(payload, "content"); got != "raw only" {
		t.Fatalf("rawContent fallback: got %q", got)
	}
	// 字符串形态仍兼容旧 payload / 单测。
	if got := payloadString(map[string]any{"content": "plain"}, "content"); got != "plain" {
		t.Fatalf("string content: got %q", got)
	}
}

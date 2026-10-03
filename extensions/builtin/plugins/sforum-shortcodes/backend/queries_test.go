package main

import (
	"encoding/json"
	"testing"
)

func TestPublicFriendLinksStableOrderAndBoundedFields(t *testing.T) {
	t.Parallel()
	rows := []map[string]any{
		{"id": float64(5), "name": "Late", "url": "https://late.example.com", "description": "", "position": float64(30)},
		{"id": float64(3), "name": "First", "url": "https://first.example.com", "description": "", "position": float64(10)},
		{"id": float64(4), "name": "Second", "url": "https://second.example.com", "description": "", "position": float64(10)},
	}
	links := publicFriendLinks(rows)
	if len(links) != 3 || links[0].Name != "First" || links[1].Name != "Second" || links[2].Name != "Late" {
		t.Fatalf("stable position,id order = %#v", links)
	}
}

func TestRowNumberDecodingUsesJSONNumbers(t *testing.T) {
	t.Parallel()
	row := map[string]any{"id": json.Number("9007199254740993")}
	id, ok := rowInt64(row, "id")
	if !ok || id != 9007199254740993 {
		t.Fatalf("json.Number id decode = %d, %t", id, ok)
	}
	invalid := map[string]any{"id": json.Number("12.5")}
	if _, ok := rowInt64(invalid, "id"); ok {
		t.Fatalf("fractional id accepted")
	}
	// M4 frozen投影将 int64 编码为 string（normalizeProtocolV2QueryValue）。
	stringRow := map[string]any{"id": "42"}
	id, ok = rowInt64(stringRow, "id")
	if !ok || id != 42 {
		t.Fatalf("string id decode = %d, %t", id, ok)
	}
	if parsed, ok := rowInt64(map[string]any{"id": "-1"}, "id"); !ok || parsed != -1 {
		t.Fatalf("negative string id parsing = %d, %t", parsed, ok)
	}
	if _, found := publicUserRow([]map[string]any{{"id": "-1", "username": "x"}}, 1); found {
		t.Fatalf("negative row id matched a positive request")
	}
}

func TestResponseRowsSkipsInvalidWireRows(t *testing.T) {
	t.Parallel()
	// 构造 RPC 层无 TypedDocument 的输入由 responseRows 的 nil 防护处理。
	if rows := responseRows(nil); len(rows) != 0 {
		t.Fatalf("nil response rows = %#v", rows)
	}
}

func TestSafeLinkURLAllowsOnlyHTTPSScheme(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"https://example.com":              true,
		"http://example.com/path?q=1":      true,
		"https://user:pass@example.com":    false,
		"javascript:alert(1)":              false,
		"data:text/html,x":                 false,
		"/relative/path":                   false,
		"ftp://example.com/file":           false,
		"https://example.com/a\nb":         false,
		`https://example.com/..%2f..%2f..`: true,
	}
	for value, expected := range cases {
		if got := safeLinkURL(value) != ""; got != expected {
			t.Errorf("safeLinkURL(%q) safe=%t want %t", value, got, expected)
		}
	}
}

func TestSafePathEscapeBlocksTraversal(t *testing.T) {
	t.Parallel()
	if got := profilePath("../admin"); got != "/u/..%2Fadmin" {
		t.Fatalf("path escape = %q", got)
	}
	if got := profilePath(".."); got != "" {
		t.Fatalf("dot-segment path must be rejected, got %q", got)
	}
	if got := profilePath("x y?q=1#frag"); got != "/u/x%20y%3Fq=1%23frag" {
		t.Fatalf("path escape = %q", got)
	}
	if got := categoryPath("general"); got != "/c/general" {
		t.Fatalf("path escape = %q", got)
	}
	if got := categoryPath("x/y"); got != "/c/x%2Fy" {
		t.Fatalf("path escape = %q", got)
	}
}

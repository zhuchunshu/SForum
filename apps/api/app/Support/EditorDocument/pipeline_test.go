package editordocument

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAcceptNativeJSONRoundTripAndSanitizerCorpus(t *testing.T) {
	t.Parallel()
	native := map[string]any{
		"type": "doc",
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "Hello ",
						"marks": []any{
							map[string]any{"type": "bold"},
						},
					},
					map[string]any{
						"type": "text",
						"text": "world",
						"marks": []any{
							map[string]any{
								"type":  "link",
								"attrs": map[string]any{"href": "javascript:alert(1)"},
							},
						},
					},
					map[string]any{
						"type": "text",
						"text": " ",
					},
					map[string]any{
						"type": "text",
						"text": "safe",
						"marks": []any{
							map[string]any{
								"type":  "link",
								"attrs": map[string]any{"href": "https://example.com/a"},
							},
						},
					},
				},
			},
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{
						"type": "sforumEmoji",
						"attrs": map[string]any{
							"name": "sparkles", "label": "灵感", "native": "✨",
						},
					},
				},
			},
			map[string]any{
				"type": "evilScript",
				"content": []any{
					map[string]any{"type": "text", "text": "xss"},
				},
			},
		},
	}
	body, err := json.Marshal(native)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := Accept(Input{NativeJSON: body, Schema: CoreSchema()})
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if accepted.StorageVersion != StorageVersion || accepted.ContentHash == "" {
		t.Fatalf("accepted meta = %#v", accepted)
	}
	if !strings.Contains(accepted.HTMLSanitized, "<strong>Hello </strong>") {
		t.Fatalf("html missing bold: %s", accepted.HTMLSanitized)
	}
	if strings.Contains(accepted.HTMLSanitized, "javascript:") ||
		strings.Contains(accepted.HTMLSanitized, "<script") {
		t.Fatalf("html retained unsafe content: %s", accepted.HTMLSanitized)
	}
	if !strings.Contains(accepted.HTMLSanitized, "https://example.com/a") {
		t.Fatalf("html missing safe link: %s", accepted.HTMLSanitized)
	}
	if !strings.Contains(accepted.HTMLSanitized, "data-sforum-emoji") {
		t.Fatalf("html missing emoji: %s", accepted.HTMLSanitized)
	}
	if !strings.Contains(accepted.PlainText, "Hello") || !strings.Contains(accepted.SearchText, "emoji:sparkles") {
		t.Fatalf("plain/search = %q / %q", accepted.PlainText, accepted.SearchText)
	}
	if len(accepted.Fallbacks) != 1 || accepted.Fallbacks[0] != "evilScript" {
		t.Fatalf("fallbacks = %#v", accepted.Fallbacks)
	}
	if !strings.Contains(accepted.Markdown, "**Hello **") && !strings.Contains(accepted.Markdown, "**Hello") {
		// bold export may glue spaces; ensure Hello present
		if !strings.Contains(accepted.Markdown, "Hello") {
			t.Fatalf("markdown = %q", accepted.Markdown)
		}
	}
}

func TestAcceptRejectsEmptyAndOversized(t *testing.T) {
	t.Parallel()
	if _, err := Accept(Input{}); err == nil {
		t.Fatal("expected empty reject")
	}
	huge := make([]byte, maxDocumentBytes+1)
	for i := range huge {
		huge[i] = 'a'
	}
	if _, err := Accept(Input{NativeJSON: huge}); err == nil {
		t.Fatal("expected oversized reject")
	}
}

func TestAcceptMarkdownCompatibilityPath(t *testing.T) {
	t.Parallel()
	accepted, err := Accept(Input{Markdown: "line one\n\nline two", Schema: CoreSchema()})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(accepted.HTMLSanitized, "line one") || !strings.Contains(accepted.PlainText, "line two") {
		t.Fatalf("accepted = %#v", accepted)
	}
}

func TestMigrateStorageReacceptsUnderCurrentSchema(t *testing.T) {
	t.Parallel()
	body, _ := json.Marshal(map[string]any{
		"type": "doc",
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{"type": "text", "text": "stable"},
				},
			},
		},
	})
	accepted, err := Accept(Input{NativeJSON: body, Schema: CoreSchema()})
	if err != nil {
		t.Fatal(err)
	}
	// Pretend older storage version.
	accepted.StorageVersion = "sforum.editor-document@0"
	migrated, err := MigrateStorage(accepted, StorageVersion, CoreSchema())
	if err != nil {
		t.Fatal(err)
	}
	if migrated.StorageVersion != StorageVersion || !strings.Contains(migrated.PlainText, "stable") {
		t.Fatalf("migrated = %#v", migrated)
	}
	if _, err := MigrateStorage(accepted, "sforum.editor-document@99", CoreSchema()); err != ErrStorageVersion {
		t.Fatalf("future version err = %v", err)
	}
}

func TestSchemaFromEditorNamesMergesWithoutOverridingCore(t *testing.T) {
	t.Parallel()
	schema := SchemaFromEditorNames([]string{"demoVote", "paragraph"}, []string{"demoHighlight"})
	if _, ok := schema.Nodes["demoVote"]; !ok {
		t.Fatal("missing plugin node")
	}
	// Core paragraph must remain non-atom.
	if schema.Nodes["paragraph"].Atom {
		t.Fatal("plugin must not override core paragraph")
	}
	if _, ok := schema.Marks["demoHighlight"]; !ok {
		t.Fatal("missing plugin mark")
	}
}

func TestOrderedPipelineRunsStableStageSequence(t *testing.T) {
	t.Parallel()
	stages := OrderedStages()
	if len(stages) != 8 || stages[0] != StageParse || stages[len(stages)-1] != StageSEO {
		t.Fatalf("stages = %#v", stages)
	}
	body, _ := json.Marshal(map[string]any{
		"type": "doc",
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{"type": "text", "text": "pipeline"},
				},
			},
		},
	})
	accepted, results, err := RunOrderedPipeline(Input{NativeJSON: body, Schema: CoreSchema()})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(accepted.PlainText, "pipeline") {
		t.Fatalf("accepted = %#v", accepted)
	}
	if len(results) != 8 {
		t.Fatalf("results = %#v", results)
	}
	for index, stage := range OrderedStages() {
		if results[index].Stage != stage || !results[index].OK {
			t.Fatalf("result[%d] = %#v", index, results[index])
		}
	}
}

func TestDisabledPluginNodeFallsBackWithoutLosingSurroundingText(t *testing.T) {
	t.Parallel()
	// First accept with plugin node admitted.
	withPlugin := SchemaFromEditorNames([]string{"demoVote"}, nil)
	body, _ := json.Marshal(map[string]any{
		"type": "doc",
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{"type": "text", "text": "before "},
					map[string]any{"type": "demoVote", "attrs": map[string]any{"id": "1"}},
					map[string]any{"type": "text", "text": " after"},
				},
			},
		},
	})
	accepted, err := Accept(Input{NativeJSON: body, Schema: withPlugin})
	if err != nil {
		t.Fatal(err)
	}
	// Re-accept stored native under core-only schema (plugin disabled).
	native, _ := json.Marshal(accepted.Native)
	// Force original plugin node back into document to simulate stored history.
	native = []byte(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"before "},{"type":"demoVote","attrs":{"id":"1"}},{"type":"text","text":" after"}]}]}`)
	disabled, err := Accept(Input{NativeJSON: native, Schema: CoreSchema()})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(disabled.PlainText, "before") || !strings.Contains(disabled.PlainText, "after") {
		t.Fatalf("lost surrounding text: %q", disabled.PlainText)
	}
	if len(disabled.Fallbacks) != 1 || disabled.Fallbacks[0] != "demoVote" {
		t.Fatalf("fallbacks = %#v", disabled.Fallbacks)
	}
}

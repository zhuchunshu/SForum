package main

import "testing"

func TestSyncInlineDeclarationDigestsRefreshesEditorL2Module(t *testing.T) {
	manifest := map[string]any{
		"editor": []any{
			map[string]any{
				"kind": "command", "l2Module": "frontend/editor/demo.mjs", "l2Digest": "old",
			},
			map[string]any{
				"kind": "toolbar", "commandId": "demo.command",
			},
		},
	}
	syncInlineDeclarationDigests(manifest, map[string]string{
		"frontend/editor/demo.mjs": "new-digest",
	})
	items := manifest["editor"].([]any)
	command := items[0].(map[string]any)
	if command["l2Digest"] != "new-digest" {
		t.Fatalf("l2Digest = %v", command["l2Digest"])
	}
	toolbar := items[1].(map[string]any)
	if _, ok := toolbar["l2Digest"]; ok {
		t.Fatalf("toolbar unexpectedly received l2Digest: %#v", toolbar)
	}
}

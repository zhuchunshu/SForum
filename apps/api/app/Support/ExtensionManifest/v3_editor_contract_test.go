package extensionmanifest

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestV3EditorDeclarationsRequireFrontendPackageFiles(t *testing.T) {
	t.Parallel()
	moduleDigest := strings.Repeat("ab", 32)
	manifest := completeV3Manifest()
	manifest.PackageFiles = append(manifest.PackageFiles, ManifestPackageFile{
		ID: "demo.v3.editor.vote.module", Kind: "frontend",
		Path: "frontend/editor/vote.mjs", Digest: moduleDigest, Version: "1",
	})
	manifest.Editor = []ManifestEditor{
		{
			ID: "demo.v3.editor.node.vote", ContractVersion: "demo.v3.editor.node.vote@1",
			Kind: "node", Schema: "demo.v3.editor.vote@1", ExtensionName: "demoVote",
			L2Module: "frontend/editor/vote.mjs", L2Digest: moduleDigest,
		},
		{
			ID: "demo.v3.editor.command.insert-vote", ContractVersion: "demo.v3.editor.command.insert-vote@1",
			Kind: "command", CommandKey: "insertDemoVote",
			L2Module: "frontend/editor/vote.mjs", L2Digest: moduleDigest,
		},
		{
			ID: "demo.v3.editor.toolbar.vote", ContractVersion: "demo.v3.editor.toolbar.vote@1",
			Kind: "toolbar", CommandID: "demo.v3.editor.command.insert-vote",
			Label: "Vote", Icon: "i-tabler-checkbox", Group: "insert", Order: 10,
		},
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("Validate editor declarations: %v", err)
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("ValidateV3JSONSchema editor: %v", err)
	}
}

func TestV3EditorRejectsToolbarWithoutCommandAndMissingModule(t *testing.T) {
	t.Parallel()
	manifest := completeV3Manifest()
	manifest.Editor = []ManifestEditor{{
		ID: "demo.v3.editor.toolbar.orphan", ContractVersion: "demo.v3.editor.toolbar.orphan@1",
		Kind: "toolbar", CommandID: "demo.v3.editor.command.missing", Label: "Orphan",
	}}
	if err := Validate(manifest); err == nil {
		t.Fatal("expected toolbar without command to fail")
	}
	moduleDigest := strings.Repeat("cd", 32)
	manifest = completeV3Manifest()
	manifest.Editor = []ManifestEditor{{
		ID: "demo.v3.editor.node.vote", ContractVersion: "demo.v3.editor.node.vote@1",
		Kind: "node", Schema: "demo.v3.editor.vote@1", ExtensionName: "demoVote",
		L2Module: "frontend/editor/vote.mjs", L2Digest: moduleDigest,
	}}
	if err := Validate(manifest); err == nil {
		t.Fatal("expected missing frontend package file to fail")
	}
}

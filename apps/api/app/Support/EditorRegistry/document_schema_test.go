package editorregistry

import (
	"strings"
	"testing"
)

func TestDocumentSchemaMergesPluginNodesAndMarks(t *testing.T) {
	t.Parallel()
	registry := New()
	digest := strings.Repeat("ab", 32)
	moduleDigest := strings.Repeat("cd", 32)
	if _, err := registry.Publish(Publication{
		Artifact: Artifact{
			ExtensionID: "demo.editor", ExtensionVersion: "1.0.0",
			PackageDigest: digest, VersionID: 1,
		},
		Editor: []Declaration{
			{
				ID: "demo.editor.node.vote", ContractVersion: "demo.editor.node.vote@1",
				Kind: KindNode, Schema: "demo.editor.vote@1", ExtensionName: "demoVote",
				L2Module: "frontend/editor/vote.mjs", L2Digest: moduleDigest,
			},
			{
				ID: "demo.editor.mark.accent", ContractVersion: "demo.editor.mark.accent@1",
				Kind: KindMark, Schema: "demo.editor.accent@1", ExtensionName: "demoAccent",
				L2Module: "frontend/editor/vote.mjs", L2Digest: moduleDigest,
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	schema := registry.DocumentSchema()
	if _, ok := schema.Nodes["paragraph"]; !ok {
		t.Fatal("expected core paragraph node")
	}
	if _, ok := schema.Nodes["demoVote"]; !ok {
		t.Fatalf("expected plugin node demoVote, nodes=%v", schema.Nodes)
	}
	if _, ok := schema.Marks["demoAccent"]; !ok {
		t.Fatalf("expected plugin mark demoAccent, marks=%v", schema.Marks)
	}
	if _, ok := schema.Marks["bold"]; !ok {
		t.Fatal("expected core bold mark")
	}
}

func TestDocumentSchemaNilRegistryIsCoreOnly(t *testing.T) {
	t.Parallel()
	var registry *Registry
	schema := registry.DocumentSchema()
	if _, ok := schema.Nodes["doc"]; !ok {
		t.Fatal("expected core schema")
	}
	if len(schema.Nodes) < 5 {
		t.Fatalf("core schema too small: %#v", schema.Nodes)
	}
}

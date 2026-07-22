package editorregistry

import (
	"strings"
	"testing"
)

func TestBuildCatalogGroupsTrustedL2Modules(t *testing.T) {
	t.Parallel()
	registry := New()
	digest := strings.Repeat("ab", 32)
	moduleDigest := strings.Repeat("cd", 32)
	if _, err := registry.Publish(Publication{
		Artifact: Artifact{
			ExtensionID: "demo.editor", ExtensionVersion: "1.0.0",
			PackageDigest: digest, VersionID: 3,
		},
		Editor: []Declaration{
			{
				ID: "demo.editor.node.vote", ContractVersion: "demo.editor.node.vote@1",
				Kind: KindNode, Schema: "demo.editor.vote@1", ExtensionName: "demoVote",
				L2Module: "frontend/editor/vote.mjs", L2Digest: moduleDigest,
			},
			{
				ID: "demo.editor.command.insert-vote", ContractVersion: "demo.editor.command.insert-vote@1",
				Kind: KindCommand, CommandKey: "insertDemoVote",
				L2Module: "frontend/editor/vote.mjs", L2Digest: moduleDigest,
			},
			{
				ID: "demo.editor.toolbar.vote", ContractVersion: "demo.editor.toolbar.vote@1",
				Kind: KindToolbar, CommandID: "demo.editor.command.insert-vote",
				Label: "Vote", Order: 1,
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	catalog := registry.BuildCatalog()
	if catalog.SchemaVersion != CatalogSchemaVersion || catalog.Revision != 1 || len(catalog.Modules) != 1 {
		t.Fatalf("catalog = %#v", catalog)
	}
	module := catalog.Modules[0]
	if module.L2Digest != moduleDigest ||
		!strings.HasPrefix(module.AssetPath, "/_sforum/assets/extensions/demo.editor/") ||
		!strings.Contains(module.AssetPath, digest) ||
		!strings.HasSuffix(module.AssetPath, "frontend/editor/vote.mjs") ||
		len(module.Nodes) != 1 || len(module.Commands) != 1 || len(module.Toolbars) != 1 {
		t.Fatalf("module = %#v", module)
	}
	if len(catalog.Toolbars) != 1 || catalog.Toolbars[0].Label != "Vote" {
		t.Fatalf("toolbars = %#v", catalog.Toolbars)
	}
}

func TestBuildCatalogEmptyOnNilRegistry(t *testing.T) {
	t.Parallel()
	var registry *Registry
	catalog := registry.BuildCatalog()
	if catalog.SchemaVersion != CatalogSchemaVersion || len(catalog.Modules) != 0 {
		t.Fatalf("nil catalog = %#v", catalog)
	}
}

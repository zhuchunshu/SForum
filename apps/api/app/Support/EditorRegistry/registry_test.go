package editorregistry

import (
	"strings"
	"testing"
)

func TestEditorRegistryPublishNodeMarkCommandToolbar(t *testing.T) {
	t.Parallel()
	registry := New()
	digest := strings.Repeat("ab", 32)
	moduleDigest := strings.Repeat("cd", 32)
	publication := Publication{
		Artifact: Artifact{
			ExtensionID: "demo.editor", ExtensionVersion: "1.0.0",
			PackageDigest: digest, VersionID: 7,
		},
		Editor: []Declaration{
			{
				ID: "demo.editor.node.vote", ContractVersion: "demo.editor.node.vote@1",
				Kind: KindNode, Schema: "demo.editor.vote@1", ExtensionName: "demoVote",
				L2Module: "frontend/editor/vote.mjs", L2Digest: moduleDigest,
			},
			{
				ID: "demo.editor.mark.highlight", ContractVersion: "demo.editor.mark.highlight@1",
				Kind: KindMark, Schema: "demo.editor.highlight@1", ExtensionName: "demoHighlight",
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
				Label: "投票", Icon: "i-tabler-checkbox", Group: "insert", Order: 10,
			},
		},
	}
	revision, err := registry.Publish(publication)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if revision != 1 {
		t.Fatalf("revision = %d", revision)
	}
	snapshot := registry.Snapshot()
	if snapshot.SchemaVersion != SchemaVersion || snapshot.SafeMode || len(snapshot.Editor) != 4 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	nodes := registry.List(KindNode)
	if len(nodes) != 1 || nodes[0].ExtensionName != "demoVote" || nodes[0].L2Digest != moduleDigest {
		t.Fatalf("nodes = %#v", nodes)
	}
	toolbars := registry.List(KindToolbar)
	if len(toolbars) != 1 || toolbars[0].CommandID != "demo.editor.command.insert-vote" {
		t.Fatalf("toolbars = %#v", toolbars)
	}
	modules := registry.TrustedL2Modules()
	if len(modules) != 1 || modules[0].L2Module != "frontend/editor/vote.mjs" {
		t.Fatalf("modules = %#v", modules)
	}
	if _, err := registry.Resolve("demo.editor.node.vote"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
}

func TestEditorRegistryRejectsToolbarWithoutCommand(t *testing.T) {
	t.Parallel()
	registry := New()
	_, err := registry.Publish(Publication{
		Artifact: Artifact{
			ExtensionID: "demo.editor", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("11", 32), VersionID: 1,
		},
		Editor: []Declaration{{
			ID: "demo.editor.toolbar.orphan", ContractVersion: "demo.editor.toolbar.orphan@1",
			Kind: KindToolbar, CommandID: "demo.editor.command.missing", Label: "Orphan",
		}},
	})
	if err == nil {
		t.Fatal("expected invalid toolbar without command")
	}
}

func TestEditorRegistryRejectsPathTraversalL2Module(t *testing.T) {
	t.Parallel()
	registry := New()
	_, err := registry.Publish(Publication{
		Artifact: Artifact{
			ExtensionID: "demo.editor", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("22", 32), VersionID: 1,
		},
		Editor: []Declaration{{
			ID: "demo.editor.node.bad", ContractVersion: "demo.editor.node.bad@1",
			Kind: KindNode, Schema: "demo.editor.bad@1", ExtensionName: "badNode",
			L2Module: "../escape.mjs", L2Digest: strings.Repeat("33", 32),
		}},
	})
	if err == nil {
		t.Fatal("expected path traversal rejection")
	}
}

func TestEditorRegistrySafeModeRejectsThirdParty(t *testing.T) {
	t.Parallel()
	registry := New()
	coreDigest := strings.Repeat("aa", 32)
	coreArtifact, err := NewCoreArtifact("core.editor", "1.0.0", coreDigest)
	if err != nil {
		t.Fatalf("core artifact: %v", err)
	}
	if _, err := registry.ReplaceAll([]Publication{{
		Artifact: coreArtifact,
		Editor: []Declaration{{
			ID: "core.editor.command.bold", ContractVersion: "core.editor.command.bold@1",
			Kind: KindCommand, CommandKey: "toggleBold",
		}},
	}}, true); err != nil {
		t.Fatalf("core safe mode publish: %v", err)
	}
	_, err = registry.Publish(Publication{
		Artifact: Artifact{
			ExtensionID: "demo.editor", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("bb", 32), VersionID: 2,
		},
		Editor: []Declaration{{
			ID: "demo.editor.command.x", ContractVersion: "demo.editor.command.x@1",
			Kind: KindCommand, CommandKey: "x",
		}},
	})
	if err != ErrSafeMode {
		t.Fatalf("safe mode err = %v", err)
	}
	if len(registry.List("")) != 1 {
		t.Fatalf("safe mode list = %#v", registry.List(""))
	}
}

func TestEditorRegistryDisableDoesNotRewriteSource(t *testing.T) {
	t.Parallel()
	registry := New()
	artifact := Artifact{
		ExtensionID: "demo.editor", ExtensionVersion: "1.0.0",
		PackageDigest: strings.Repeat("cc", 32), VersionID: 3,
	}
	moduleDigest := strings.Repeat("dd", 32)
	if _, err := registry.Publish(Publication{
		Artifact: artifact,
		Editor: []Declaration{{
			ID: "demo.editor.node.card", ContractVersion: "demo.editor.node.card@1",
			Kind: KindNode, Schema: "demo.editor.card@1", ExtensionName: "demoCard",
			L2Module: "frontend/editor/card.mjs", L2Digest: moduleDigest,
		}},
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	// Disable removes the declaration graph only; Registry never holds document
	// bodies, so user media/content cannot be rewritten here.
	revision, removed, err := registry.Remove(artifact)
	if err != nil || !removed || revision != 2 {
		t.Fatalf("remove = rev=%d removed=%v err=%v", revision, removed, err)
	}
	if len(registry.List(KindNode)) != 0 {
		t.Fatal("expected empty graph after disable")
	}
	if _, err := registry.Resolve("demo.editor.node.card"); err != ErrNotFound {
		t.Fatalf("resolve after disable = %v", err)
	}
}

func TestEditorRegistryRejectsSameArtifactDeclarationDrift(t *testing.T) {
	t.Parallel()
	registry := New()
	artifact := Artifact{
		ExtensionID: "demo.editor", ExtensionVersion: "1.0.0",
		PackageDigest: strings.Repeat("ee", 32), VersionID: 4,
	}
	moduleDigest := strings.Repeat("ff", 32)
	first := Publication{
		Artifact: artifact,
		Editor: []Declaration{{
			ID: "demo.editor.node.a", ContractVersion: "demo.editor.node.a@1",
			Kind: KindNode, Schema: "demo.editor.a@1", ExtensionName: "a",
			L2Module: "frontend/editor/a.mjs", L2Digest: moduleDigest,
		}},
	}
	if _, err := registry.Publish(first); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	drift := first
	drift.Editor = []Declaration{{
		ID: "demo.editor.node.a", ContractVersion: "demo.editor.node.a@1",
		Kind: KindNode, Schema: "demo.editor.a@2", ExtensionName: "a",
		L2Module: "frontend/editor/a.mjs", L2Digest: moduleDigest,
	}}
	if _, err := registry.Publish(drift); err != ErrArtifactConflict {
		t.Fatalf("drift err = %v", err)
	}
}

func TestEditorRegistryReplaceAllIfRevisionCAS(t *testing.T) {
	t.Parallel()
	registry := New()
	if _, err := registry.ReplaceAll(nil, false); err != nil {
		t.Fatalf("empty replace: %v", err)
	}
	if _, err := registry.ReplaceAll(nil, false); err != ErrRevisionConflict {
		t.Fatalf("second replaceall = %v", err)
	}
	revision := registry.Revision()
	digest := strings.Repeat("12", 32)
	moduleDigest := strings.Repeat("34", 32)
	if _, err := registry.ReplaceAllIfRevision(revision, []Publication{{
		Artifact: Artifact{
			ExtensionID: "demo.editor", ExtensionVersion: "2.0.0",
			PackageDigest: digest, VersionID: 9,
		},
		Editor: []Declaration{{
			ID: "demo.editor.mark.x", ContractVersion: "demo.editor.mark.x@1",
			Kind: KindMark, Schema: "demo.editor.x@1", ExtensionName: "xMark",
			L2Module: "frontend/editor/x.mjs", L2Digest: moduleDigest,
		}},
	}}, false); err != nil {
		t.Fatalf("cas replace: %v", err)
	}
	if len(registry.List(KindMark)) != 1 {
		t.Fatalf("marks = %#v", registry.List(KindMark))
	}
}

func TestEditorRegistryRejectsCoreFlagWithoutSeal(t *testing.T) {
	t.Parallel()
	registry := New()
	_, err := registry.Publish(Publication{
		Artifact: Artifact{
			ExtensionID: "core.editor", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("56", 32), Core: true,
		},
		Editor: []Declaration{{
			ID: "core.editor.command.x", ContractVersion: "core.editor.command.x@1",
			Kind: KindCommand, CommandKey: "x",
		}},
	})
	if err == nil {
		t.Fatal("expected unsealed core rejection")
	}
}

package contentregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestRegistryEnforcesPublicationAndContentLimits(t *testing.T) {
	publications := make([]Publication, 0, maxPublications+1)
	for index := 0; index < maxPublications+1; index++ {
		publications = append(publications, publication(fmt.Sprintf("limit.plugin.%03d", index), false, 'a'))
	}
	if _, err := New().ReplaceAll(publications[:maxPublications], false); err != nil {
		t.Fatalf("maximum publications rejected=%v", err)
	}
	if _, err := New().ReplaceAll(publications, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("publication overflow=%v", err)
	}

	contentBound := publication("limit.content", false, 'c')
	contentBound.Content = make([]Declaration, maxContentPerPublication+1)
	for index := range contentBound.Content {
		id := fmt.Sprintf("limit.content.item.%03d", index)
		contentBound.Content[index] = content(id, KindBlock, "h", id+".schema@1")
	}
	if _, err := New().ReplaceAll([]Publication{contentBound}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("content overflow=%v", err)
	}
}

func TestRegistryEnforcesTotalContentLimit(t *testing.T) {
	publicationCount := maxContentTotal / maxContentPerPublication
	publications := make([]Publication, 0, publicationCount+1)
	for publicationIndex := 0; publicationIndex < publicationCount; publicationIndex++ {
		extensionID := fmt.Sprintf("total.p%03d", publicationIndex)
		item := publication(extensionID, false, 'a')
		item.Content = make([]Declaration, 0, maxContentPerPublication)
		for contentIndex := 0; contentIndex < maxContentPerPublication; contentIndex++ {
			id := fmt.Sprintf("%s.item.%03d", extensionID, contentIndex)
			item.Content = append(item.Content, content(id, KindBlock, "h", id+".schema@1"))
		}
		publications = append(publications, item)
	}
	if _, err := New().ReplaceAll(publications, false); err != nil {
		t.Fatalf("exact total content limit error = %v", err)
	}
	extra := publication("total.extra", false, 'b')
	extra.Content = []Declaration{
		content("total.extra.item.one", KindBlock, "h", "total.extra.item.one.schema@1"),
	}
	if _, err := New().ReplaceAll(append(publications, extra), false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("total content overflow error = %v", err)
	}
}

func TestRegistryRejectsInvalidArtifactKindAndReferences(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Publication)
	}{
		{name: "core mismatch", edit: func(p *Publication) { p.Artifact.Core = true }},
		{name: "bad digest", edit: func(p *Publication) { p.Artifact.PackageDigest = "not-hex" }},
		{name: "bad version", edit: func(p *Publication) { p.Artifact.ExtensionVersion = "v1" }},
		{name: "long version", edit: func(p *Publication) {
			p.Artifact.ExtensionVersion = "1.0.0+" + strings.Repeat("a", maxExtensionVersionLength)
		}},
		{name: "missing version id", edit: func(p *Publication) { p.Artifact.VersionID = 0 }},
		{name: "missing runtime", edit: func(p *Publication) { p.Artifact.RuntimeInstanceID = "" }},
		{name: "long runtime", edit: func(p *Publication) {
			p.Artifact.RuntimeInstanceID = strings.Repeat("r", maxRuntimeInstanceIDLength+1)
		}},
		{name: "runtime control", edit: func(p *Publication) { p.Artifact.RuntimeInstanceID = "runtime\nforged" }},
		{name: "bad kind", edit: func(p *Publication) { p.Content[0].Kind = "content_type" }},
		{name: "missing handler and renderer", edit: func(p *Publication) {
			p.Content[0].Handler = ""
			p.Content[0].Renderer = ""
		}},
		{name: "unsafe handler url", edit: func(p *Publication) { p.Content[0].Handler = "https://evil.invalid" }},
		{name: "unsafe handler traversal", edit: func(p *Publication) { p.Content[0].Handler = "../escape" }},
		{name: "handler control", edit: func(p *Publication) { p.Content[0].Handler = "h\x00x" }},
		{name: "bad schema", edit: func(p *Publication) { p.Content[0].Schema = "NoCaps" }},
		{name: "schema nul", edit: func(p *Publication) { p.Content[0].Schema = "schemas/a\x00.json" }},
		{name: "long schema", edit: func(p *Publication) {
			p.Content[0].Schema = strings.Repeat("a", maxSchemaRefLength+1) + "@1"
		}},
		{name: "id beyond manifest bound", edit: func(p *Publication) {
			p.Content[0].ID = "plugin.content." + strings.Repeat("a", maxIDLength-len("plugin.content.")+1)
			p.Content[0].ContractVersion = p.Content[0].ID + "@1"
		}},
		{name: "unowned id", edit: func(p *Publication) {
			p.Content[0].ID = "other.content.block"
			p.Content[0].ContractVersion = "other.content.block@1"
		}},
		{name: "bad renderer", edit: func(p *Publication) { p.Content[0].Renderer = "Bad Renderer" }},
		{name: "bad migration", edit: func(p *Publication) { p.Content[0].Migration = ".." }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := publication("plugin.content", false, 'a')
			item.Content = []Declaration{content("plugin.content.block.a", KindBlock, "h", "plugin.content.block.a.schema@1")}
			test.edit(&item)
			if _, err := New().ReplaceAll([]Publication{item}, false); !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected invalid, got %v", err)
			}
		})
	}
}

func TestRegistryCoreAuthorityCannotBeForgedByFlagPrefixOrJSON(t *testing.T) {
	// The core.* namespace is Host-owned even when a package does not set Core.
	coreNamespacePlugin := publication("core.compatible-plugin", false, 'c')
	coreNamespacePlugin.Content = []Declaration{
		content("core.compatible-plugin.block.a", KindBlock, "h", "core.compatible-plugin.block.a.schema@1"),
	}
	if _, err := New().Publish(coreNamespacePlugin); !errors.Is(err, ErrInvalid) {
		t.Fatalf("third-party core namespace error = %v", err)
	}

	forged := publication("plugin.content", false, 'a')
	forged.Artifact.ExtensionID = "core.forged"
	forged.Artifact.Core = true
	forged.Artifact.VersionID = 0
	forged.Artifact.RuntimeInstanceID = ""
	forged.Content = []Declaration{content("core.forged.block.a", KindBlock, "h", "core.forged.block.a.schema@1")}
	if _, err := New().Publish(forged); !errors.Is(err, ErrInvalid) {
		t.Fatalf("flag/prefix forged Core publication = %v", err)
	}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{forged}, true); err != nil {
		t.Fatalf("forged Core input blocked Safe Mode recovery = %v", err)
	}
	if snapshot := registry.Snapshot(); !snapshot.SafeMode || len(snapshot.Publications) != 0 {
		t.Fatalf("Safe Mode retained forged Core publication = %#v", snapshot)
	}

	trusted := publication("core.content", true, 'b')
	body, err := json.Marshal(trusted.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Artifact
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	trusted.Artifact = decoded
	if _, err := New().Publish(trusted); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unsealed decoded Core publication = %v", err)
	}

	// Plain Core=true + core.* without seal must not publish outside Safe Mode filter either.
	literal := Publication{Artifact: Artifact{
		ExtensionID: "core.literal", ExtensionVersion: "1.0.0",
		PackageDigest: strings.Repeat("e", 64), Core: true,
	}, Content: []Declaration{content("core.literal.block.a", KindBlock, "h", "core.literal.block.a.schema@1")}}
	if _, err := New().Publish(literal); !errors.Is(err, ErrInvalid) {
		t.Fatalf("literal Core publication = %v", err)
	}
}

func TestRegistryRejectsNamespaceForgeryAndCrossOwnerIDs(t *testing.T) {
	owner := publication("owner.content", false, 'a')
	owner.Content = []Declaration{content("owner.content.block.a", KindBlock, "h", "owner.content.block.a.schema@1")}
	if _, err := New().Publish(owner); err != nil {
		t.Fatal(err)
	}

	// Another extension cannot claim the owner's content id prefix by changing only the id.
	thief := publication("thief.content", false, 'b')
	thief.Content = []Declaration{content("owner.content.block.stolen", KindBlock, "h", "owner.content.block.stolen.schema@1")}
	if _, err := New().Publish(thief); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-owner id forgery = %v", err)
	}

	// Bare "core" extension id is reserved and rejected.
	bare := publication("core", false, 'c')
	if _, err := New().Publish(bare); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bare core extension id = %v", err)
	}
}

func TestRegistryUsesStrictSemVerAndDigestBounds(t *testing.T) {
	valid := publication("semver.content", false, 'a')
	valid.Artifact.ExtensionVersion = "1.2.3-rc.1+build.7"
	valid.Content = []Declaration{content("semver.content.block.a", KindBlock, "h", "semver.content.block.a.schema@1")}
	if _, err := New().Publish(valid); err != nil {
		t.Fatalf("strict SemVer publication error = %v", err)
	}
	valid.Artifact.ExtensionVersion = "01.2.3"
	valid.Artifact.VersionID = 2
	if _, err := New().Publish(valid); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid SemVer publication error = %v", err)
	}

	// Renderer-only content (no handler) remains valid when schema is present.
	rendererOnly := publication("render.content", false, 'd')
	rendererOnly.Artifact.RuntimeInstanceID = ""
	rendererOnly.Content = []Declaration{{
		ID: "render.content.block.a", ContractVersion: "render.content.block.a@1",
		Kind: KindBlock, Schema: "render.content.block.a.schema@1", Renderer: "render.content.template.a",
	}}
	if _, err := New().Publish(rendererOnly); err != nil {
		t.Fatalf("renderer-only publication error = %v", err)
	}
	rendererOnly.Content[0].Renderer = "render.content.template@1"
	if _, err := New().Publish(rendererOnly); !errors.Is(err, ErrInvalid) {
		t.Fatalf("contract-shaped renderer id error = %v", err)
	}
}

func TestRegistryAllowsAllFrozenKinds(t *testing.T) {
	kinds := []string{KindBlock, KindShortcode, KindEmbed, KindNode, KindMark, KindRenderFilter, KindSanitizer}
	plugin := publication("kinds.content", false, 'a')
	for index, kind := range kinds {
		id := fmt.Sprintf("kinds.content.item.%d", index)
		plugin.Content = append(plugin.Content, content(id, kind, "h", id+".schema@1"))
	}
	registry := New()
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	if len(registry.List("")) != len(kinds) {
		t.Fatalf("list all kinds=%d", len(registry.List("")))
	}
	for _, kind := range kinds {
		if got := registry.List(kind); len(got) != 1 || got[0].Kind != kind {
			t.Fatalf("list %s = %#v", kind, got)
		}
	}
}

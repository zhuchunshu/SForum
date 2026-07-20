package extensionscontroller

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	editorregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/EditorRegistry"
)

func TestPublicEditorCatalogEmptyWithoutRegistry(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	controller := NewController(nil, nil, nil)
	api := app.Group("/api/v1")
	controller.RegisterRoutes(api)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/extensions/runtime/editor-catalog", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var envelope testEnvelope[editorregistry.Catalog]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.SchemaVersion != editorregistry.CatalogSchemaVersion {
		t.Fatalf("schema = %q", envelope.Data.SchemaVersion)
	}
	if len(envelope.Data.Modules) != 0 {
		t.Fatalf("expected empty modules, got %#v", envelope.Data.Modules)
	}
}

func TestPublicEditorCatalogProjectsPublishedModules(t *testing.T) {
	t.Parallel()
	registry := editorregistry.New()
	packageDigest := strings.Repeat("ab", 32)
	moduleDigest := strings.Repeat("cd", 32)
	if _, err := registry.Publish(editorregistry.Publication{
		Artifact: editorregistry.Artifact{
			ExtensionID: "demo.editor", ExtensionVersion: "1.0.0",
			PackageDigest: packageDigest, VersionID: 1,
		},
		Editor: []editorregistry.Declaration{{
			ID: "demo.editor.node.vote", ContractVersion: "demo.editor.node.vote@1",
			Kind: editorregistry.KindNode, Schema: "demo.editor.vote@1", ExtensionName: "demoVote",
			L2Module: "frontend/editor/vote.mjs", L2Digest: moduleDigest,
		}},
	}); err != nil {
		t.Fatal(err)
	}

	app := fiber.New()
	controller := NewController(nil, nil, nil).WithEditorRegistry(registry)
	api := app.Group("/api/v1")
	controller.RegisterRoutes(api)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/extensions/runtime/editor-catalog", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if resp.Header.Get("X-SForum-Editor-Catalog-Digest") == "" {
		t.Fatal("expected catalog digest header")
	}
	var envelope testEnvelope[editorregistry.Catalog]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data.Modules) != 1 {
		t.Fatalf("modules = %#v", envelope.Data.Modules)
	}
	module := envelope.Data.Modules[0]
	if module.ExtensionID != "demo.editor" ||
		!strings.Contains(module.AssetPath, packageDigest) ||
		module.L2Digest != moduleDigest {
		t.Fatalf("module = %#v", module)
	}
}

package extensionmanifest

import (
	"encoding/json"
	"testing"
)

func TestManifestV3JSONSchemaValidatesCanonicalContract(t *testing.T) {
	body, err := json.Marshal(completeV3Manifest())
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("canonical V3 contract should validate: %v", err)
	}
	copyBody := ManifestV3JSONSchema()
	copyBody[0] = 'x'
	if ManifestV3JSONSchema()[0] != '{' {
		t.Fatal("schema accessor must return an immutable copy")
	}
}

func TestManifestV3JSONSchemaRejectsUnknownFields(t *testing.T) {
	manifest := completeV3Manifest()
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		t.Fatal(err)
	}
	object["unknownRoot"] = true
	unknownRoot, _ := json.Marshal(object)
	if err := ValidateV3JSONSchema(unknownRoot); err == nil {
		t.Fatal("unknown root field must be rejected")
	}

	delete(object, "unknownRoot")
	surfaces := object["adminSurfaces"].([]any)
	surfaces[0].(map[string]any)["unknownSurface"] = true
	unknownSurface, _ := json.Marshal(object)
	if err := ValidateV3JSONSchema(unknownSurface); err == nil {
		t.Fatal("unknown platform declaration field must be rejected")
	}

	unknownRoute := []byte(`[{"id":"demo.v3.route.x","contractVersion":"demo.v3.route.x@1","action":"add","path":"/x","methods":["GET"],"guard":"core.guard.public","fallback":"closed","mode":"http","unknown":true}]`)
	if err := validateV3JSONSchemaFragment(unknownRoute, "routes"); err == nil {
		t.Fatal("unknown sharded route field must be rejected")
	}
}

func TestLoadManifestV3RunsSchemaBeforeNormalization(t *testing.T) {
	manifest := completeV3Manifest()
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		t.Fatal(err)
	}
	object["typoField"] = json.RawMessage(`true`)
	body, _ = json.Marshal(object)
	files := FileMapFS{}
	for _, file := range manifest.PackageFiles {
		files[file.Path] = v3FixtureBody()
	}
	if _, err := LoadRootBytes(body, files); err == nil {
		t.Fatal("raw unknown field must fail before Go decoding can discard it")
	}
}

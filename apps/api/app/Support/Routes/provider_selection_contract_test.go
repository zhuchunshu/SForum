package routes

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

func TestRouteProviderMutationOpenAPISchemasAcceptDocumentedExamples(t *testing.T) {
	body, err := os.ReadFile("../../../../../contracts/openapi/schemas/extension-route-providers.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var yamlResource any
	if err := yaml.Unmarshal(body, &yamlResource); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(yamlResource)
	if err != nil {
		t.Fatal(err)
	}
	var resource any
	if err := json.Unmarshal(encoded, &resource); err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err := compiler.AddResource("https://sforum.test/route-providers.json", resource); err != nil {
		t.Fatal(err)
	}
	selectSchema, err := compiler.Compile("https://sforum.test/route-providers.json#/RouteProviderSelectInput")
	if err != nil {
		t.Fatal(err)
	}
	selectExample := map[string]any{
		"targetRouteId": "core.route.forum.create_topic", "targetContractVersion": "sforum.route.forum.create_topic@1",
		"method": "POST", "pathSignature": "/s:api/s:v1/s:topics",
		"providerRouteId": "workflow.topic.create", "providerContractVersion": "workflow.topic.create@1",
		"providerArtifact": map[string]any{
			"extensionId": "workflow", "extensionVersion": "1.0.0",
			"packageDigest": string(make([]byte, 64)), "runtimeInstanceId": "runtime-1",
		},
		"expectedRevision": float64(0),
	}
	selectExample["providerArtifact"].(map[string]any)["packageDigest"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := selectSchema.Validate(selectExample); err != nil {
		t.Fatalf("documented select example rejected: %v", err)
	}
	selectExample["unexpected"] = true
	if err := selectSchema.Validate(selectExample); err == nil {
		t.Fatal("select schema accepted an undocumented property")
	}

	resetSchema, err := compiler.Compile("https://sforum.test/route-providers.json#/RouteProviderResetInput")
	if err != nil {
		t.Fatal(err)
	}
	resetExample := map[string]any{
		"targetRouteId": "core.route.forum.create_topic", "targetContractVersion": "sforum.route.forum.create_topic@1",
		"method": "POST", "pathSignature": "/s:api/s:v1/s:topics",
		"expectedRevision": float64(3), "reasonCode": "contract_changed",
	}
	if err := resetSchema.Validate(resetExample); err != nil {
		t.Fatalf("documented reset example rejected: %v", err)
	}
}

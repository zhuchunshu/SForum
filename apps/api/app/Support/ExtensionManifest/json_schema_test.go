package extensionmanifest

import (
	"encoding/json"
	"strconv"
	"strings"
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

func TestManifestV3JSONSchemaAllowsOnlySafeOmittedRouteGuards(t *testing.T) {
	target := completeV3Manifest()
	target.Routes[0].Action = RouteActionReplace
	target.Routes[0].TargetID = "core.route.demo.target"
	target.Routes[0].Guard = ""
	body, err := json.Marshal(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("target route should allow omitted guard: %v", err)
	}

	add := completeV3Manifest()
	add.Routes[0].Guard = ""
	body, err = json.Marshal(add)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err == nil {
		t.Fatal("add route without guard/access must fail schema validation")
	}

	add.Routes[0].Access = RouteAccessLogin
	body, err = json.Marshal(add)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("add route with explicit access should validate: %v", err)
	}

	add.Routes[0].Access = "anonymous"
	body, err = json.Marshal(add)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err == nil {
		t.Fatal("unknown route access must fail schema validation")
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

func TestManifestV3JSONSchemaAcceptsAdminPageGroup(t *testing.T) {
	grouped := []byte(`[{"path":"/clients","label":"Clients","menu":true,"group":"OAuth2","order":30}]`)
	if err := validateV3JSONSchemaFragment(grouped, "adminPages"); err != nil {
		t.Fatalf("admin page with sidebar group must validate: %v", err)
	}

	tooLong := []byte(`[{"path":"/clients","label":"Clients","menu":true,"group":"` + strings.Repeat("x", 41) + `"}]`)
	if err := validateV3JSONSchemaFragment(tooLong, "adminPages"); err == nil {
		t.Fatal("admin page group label above the length budget must be rejected")
	}
}

// 工具超时上限与会话事实声明都在 JSON Schema 里：schema 是清单的第一道门，
// 语义校验只是第二道。两者不同步时，插件作者会先撞上 schema 的 maximum。
func TestManifestV3JSONSchemaBoundsAIToolTimeoutAndContext(t *testing.T) {
	tool := func(timeout int, context string) []byte {
		suffix := ""
		if context != "" {
			suffix = `,"context":` + context
		}
		return []byte(`[{"id":"demo.plugin.lookup","contractVersion":"demo.plugin.lookup@1","name":"demo-lookup","description":"Look up.","handler":"demo.plugin.lookup","timeoutMs":` +
			strconv.Itoa(timeout) + suffix + `}]`)
	}

	if err := validateV3JSONSchemaFragment(tool(AIToolMaximumTimeoutMS, ""), "aiTools"); err != nil {
		t.Fatalf("tool timeout at the ceiling must validate: %v", err)
	}
	if err := validateV3JSONSchemaFragment(tool(AIToolMaximumTimeoutMS+1, ""), "aiTools"); err == nil {
		t.Fatal("tool timeout above the ceiling must be rejected")
	}
	if err := validateV3JSONSchemaFragment(tool(4000, `["locale","viewer"]`), "aiTools"); err != nil {
		t.Fatalf("declared context must validate: %v", err)
	}
	if err := validateV3JSONSchemaFragment(tool(4000, `["ip"]`), "aiTools"); err == nil {
		t.Fatal("unknown context fact must be rejected")
	}
}

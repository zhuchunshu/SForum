package extensionopenapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

type fixtureOptions struct {
	extensionID    string
	path           string
	manifestPath   string
	method         string
	action         string
	targetID       string
	operationID    string
	namespace      string
	requestSchema  string
	responseSchema string
	guard          string
	permission     string
	rateLimit      string
	idempotency    string
	document       string
	schema         string
}

func defaultFixtureOptions(extensionID string) fixtureOptions {
	namespace := extensionID + ".api"
	return fixtureOptions{
		extensionID: extensionID, path: "/api/catalog/{id}", manifestPath: "/api/catalog/:id",
		method: "GET", action: extensionmanifest.RouteActionAdd,
		operationID: namespace + ".getCatalog", namespace: namespace,
		responseSchema: extensionID + ".catalog.response@1", guard: extensionmanifest.GuardCorePublic,
		rateLimit: "public.read@1", idempotency: "disabled",
		schema: `{"Catalog":{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}}`,
	}
}

func buildFixture(t *testing.T, options fixtureOptions) Artifact {
	t.Helper()
	if options.document == "" {
		options.document = fixtureDocument(options)
	}
	root := t.TempDir()
	writeFixtureFile(t, root, "openapi/routes.yaml", []byte(options.document))
	writeFixtureFile(t, root, "openapi/schemas/common.json", []byte(options.schema))

	routeID := options.extensionID + ".catalog"
	route := extensionmanifest.ManifestRoute{
		ID: routeID, ContractVersion: routeID + "@1", Action: options.action, TargetID: options.targetID,
		Path: options.manifestPath, Methods: []string{options.method}, Guard: options.guard,
		Permission: options.permission, Fallback: "closed", Mode: extensionmanifest.RouteModeHTTP,
		Handler: "route.catalog", RequestSchema: options.requestSchema, ResponseSchema: options.responseSchema,
	}
	if options.action == extensionmanifest.RouteActionAlias || options.action == extensionmanifest.RouteActionRewrite {
		route.Handler = ""
		route.RequestSchema = ""
		route.ResponseSchema = ""
	}
	manifest := extensionmanifest.Manifest{
		ManifestVersion: extensionmanifest.ManifestVersionV3,
		ID:              options.extensionID, Name: "OpenAPI Fixture", Description: "Static aggregate fixture.",
		URL: "https://example.com/fixture", Author: extensionmanifest.ManifestAuthor{Name: "SForum"},
		Version: "1.0.0", Type: extensionmanifest.TypePlugin, SForumVersion: "^1.0.0",
		Routes:      []extensionmanifest.ManifestRoute{route},
		Permissions: nonEmptyValues(options.permission),
		OpenAPI: []extensionmanifest.ManifestOpenAPIFragment{{
			ID: options.extensionID + ".openapi", ContractVersion: options.extensionID + ".openapi@1",
			Path: "openapi/routes.yaml", Digest: digestBytes([]byte(options.document)), Namespace: options.namespace,
		}},
		PackageFiles: []extensionmanifest.ManifestPackageFile{
			{ID: options.extensionID + ".file.openapi", Kind: "openapi", Path: "openapi/routes.yaml", Digest: digestBytes([]byte(options.document))},
			{ID: options.extensionID + ".file.schema", Kind: "schema", Path: "openapi/schemas/common.json", Digest: digestBytes([]byte(options.schema))},
		},
	}
	manifestBody, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, root, extensionmanifest.ManifestFileName, manifestBody)
	loaded, err := extensionmanifest.LoadPackage(root)
	if err != nil {
		t.Fatalf("load fixture manifest: %v\n%s", err, manifestBody)
	}
	packageDigest, err := extensionpackage.DigestTree(root)
	if err != nil {
		t.Fatal(err)
	}
	return Artifact{
		Root: root, ExtensionID: loaded.ID, Version: loaded.Version, PackageDigest: packageDigest, Manifest: loaded,
		Policies: []RoutePolicy{{
			RouteID: routeID, Method: options.method, RateLimit: options.rateLimit,
			Idempotency: options.idempotency, Security: fixtureSecurity(options.guard),
		}},
	}
}

func fixtureSecurity(guard string) string {
	if guard == extensionmanifest.GuardCoreLogin || guard == extensionmanifest.GuardCorePermission {
		return SecurityAuthenticated
	}
	return SecurityPublic
}

func nonEmptyValues(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func fixtureDocument(options fixtureOptions) string {
	parameters := ""
	names := make([]string, 0)
	for name := range pathTemplateParameters(options.path) {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) > 0 {
		parameters = "      parameters:\n"
		for _, name := range names {
			parameters += fmt.Sprintf("        - name: %s\n          in: path\n          required: true\n          schema:\n            type: string\n", name)
		}
	}
	requestBody := ""
	if options.requestSchema != "" {
		requestBody = "      requestBody:\n        required: true\n        content:\n          application/json:\n            schema:\n              $ref: 'schemas/common.json#/Catalog'\n"
	}
	responseContent := ""
	if options.responseSchema != "" {
		responseContent = "          content:\n            application/json:\n              schema:\n                $ref: 'schemas/common.json#/Catalog'\n"
	}
	permission := ""
	if options.permission != "" {
		permission = fmt.Sprintf("      x-sforum-permission: %s\n", options.permission)
	}
	requestMetadata := ""
	if options.requestSchema != "" {
		requestMetadata = fmt.Sprintf("      x-sforum-request-schema: %s\n", options.requestSchema)
	}
	responseMetadata := ""
	if options.responseSchema != "" {
		responseMetadata = fmt.Sprintf("      x-sforum-response-schema: %s\n", options.responseSchema)
	}
	return fmt.Sprintf(`openapi: 3.1.0
info:
  title: Fixture
  version: 1.0.0
paths:
  %s:
    %s:
      operationId: %s
      x-sforum-route-id: %s.catalog
      x-sforum-contract-version: %s.catalog@1
      x-sforum-guard: %s
%s%s%s      x-sforum-rate-limit: %s
      x-sforum-idempotency: %s
%s%s      responses:
        "200":
          description: ok
%s`, options.path, strings.ToLower(options.method), options.operationID,
		options.extensionID, options.extensionID, options.guard,
		permission, requestMetadata, responseMetadata, options.rateLimit, options.idempotency,
		parameters, requestBody, responseContent)
}

func writeFixtureFile(t *testing.T, root, name string, body []byte) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, body, 0o600); err != nil {
		t.Fatal(err)
	}
}

func digestBytes(body []byte) string {
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

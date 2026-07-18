package queryregistry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestJSONResultSchemaCatalogBindsExactArtifactAndValidatesRows(t *testing.T) {
	artifact := publication("core.schema", true, 'a').Artifact
	body := []byte(`{
  "$schema":"https://json-schema.org/draft/2020-12/schema",
  "type":"object",
  "additionalProperties":false,
  "required":["id","title"],
  "properties":{"id":{"type":"string"},"title":{"type":"string","minLength":1}}
}`)
	digest := sha256.Sum256(body)
	binding := JSONResultSchemaBinding{
		QueryID: "core.schema.items", ContractVersion: "core.schema.items@1",
		PlanVersion: "core.schema.items.plan@1", ResultSchema: "core.schema.items.result@1",
		Artifact: artifact, SchemaDigest: hex.EncodeToString(digest[:]), Schema: body,
	}
	catalog, err := NewJSONResultSchemaCatalog([]JSONResultSchemaBinding{binding})
	if err != nil {
		t.Fatal(err)
	}
	claim := ResultSchemaClaim{
		QueryID: binding.QueryID, ContractVersion: binding.ContractVersion,
		PlanVersion: binding.PlanVersion, ResultSchema: binding.ResultSchema, Artifact: artifact,
	}
	if err := catalog.ValidateQueryResult(context.Background(), claim, QueryRow{"id": "1", "title": "valid"}); err != nil {
		t.Fatalf("valid row=%v", err)
	}
	for name, row := range map[string]QueryRow{
		"missing": {"id": "1"},
		"type":    {"id": "1", "title": 2},
		"extra":   {"id": "1", "title": "valid", "secret": "no"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := catalog.ValidateQueryResult(context.Background(), claim, row); !errors.Is(err, ErrResultInvalid) {
				t.Fatalf("invalid row=%v", err)
			}
		})
	}
	drifted := claim
	drifted.Artifact.PackageDigest = strings.Repeat("b", 64)
	if err := catalog.ValidateQueryResult(context.Background(), drifted, QueryRow{"id": "1", "title": "valid"}); !errors.Is(err, ErrResultInvalid) {
		t.Fatalf("artifact drift=%v", err)
	}
}

func TestJSONResultSchemaCatalogRejectsDigestExternalRefsAndDuplicates(t *testing.T) {
	artifact := publication("core.schema", true, 'a').Artifact
	validBody := []byte(`{"type":"object"}`)
	digest := sha256.Sum256(validBody)
	binding := JSONResultSchemaBinding{
		QueryID: "core.schema.items", ContractVersion: "core.schema.items@1",
		PlanVersion: "core.schema.items.plan@1", ResultSchema: "core.schema.items.result@1",
		Artifact: artifact, SchemaDigest: hex.EncodeToString(digest[:]), Schema: validBody,
	}
	wrongDigest := binding
	wrongDigest.SchemaDigest = strings.Repeat("f", 64)
	if _, err := NewJSONResultSchemaCatalog([]JSONResultSchemaBinding{wrongDigest}); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("wrong digest=%v", err)
	}
	external := binding
	external.Schema = []byte(`{"$ref":"https://example.test/schema.json"}`)
	externalDigest := sha256.Sum256(external.Schema)
	external.SchemaDigest = hex.EncodeToString(externalDigest[:])
	if _, err := NewJSONResultSchemaCatalog([]JSONResultSchemaBinding{external}); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("external ref=%v", err)
	}
	externalDynamic := binding
	externalDynamic.Schema = []byte(`{"$dynamicRef":"https://example.test/schema.json#node"}`)
	externalDynamicDigest := sha256.Sum256(externalDynamic.Schema)
	externalDynamic.SchemaDigest = hex.EncodeToString(externalDynamicDigest[:])
	if _, err := NewJSONResultSchemaCatalog([]JSONResultSchemaBinding{externalDynamic}); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("external dynamic ref=%v", err)
	}
	if _, err := NewJSONResultSchemaCatalog([]JSONResultSchemaBinding{binding, binding}); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("duplicate binding=%v", err)
	}
}

func TestBindResultSchemasPublishesPrivateMaterialThroughRegistry(t *testing.T) {
	publication := publication("plugin.bound-schema", false, 'a')
	declaration := query("plugin.bound-schema.items", "plugin.bound-schema.item", PaginationNone, "public")
	declaration.Relations = nil
	publication.Queries = []QueryDeclaration{declaration}
	schemaText := `{"type":"object","required":["id","title"],"properties":{"id":{"type":"string"},"title":{"type":"string"},"private_schema_marker":{"type":"string"}},"additionalProperties":false}`
	body := []byte(schemaText)
	binding := resultSchemaBinding(publication, declaration, body)

	bound, err := BindResultSchemas(publication, []JSONResultSchemaBinding{binding})
	if err != nil {
		t.Fatal(err)
	}
	wantDigest := binding.SchemaDigest
	if bound.Queries[0].ResultSchemaDigest != wantDigest || bound.Queries[0].boundResultSchema == nil {
		t.Fatalf("bound query=%#v", bound.Queries[0])
	}
	legacyRegistry := New()
	if _, err := legacyRegistry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	// BindResultSchemas and Publish both sever caller ownership. Mutating either
	// input must not alter the compiled Registry state or an in-memory rebuild.
	body[0] = 'x'

	registry := New()
	revision, err := registry.Publish(bound)
	if err != nil || revision != 1 {
		t.Fatalf("publish bound Schema: revision=%d err=%v", revision, err)
	}
	bound.Queries[0].boundResultSchema.binding.Schema[0] = 'x'
	bound.Queries[0].boundResultSchema.validator = nil
	bound.Queries[0].ResultSchemaDigest = strings.Repeat("f", 64)
	if registry.Snapshot().Digest == legacyRegistry.Snapshot().Digest {
		t.Fatal("bound Schema digest did not advance the Registry graph identity")
	}
	claim := resultSchemaClaim(binding)
	valid := QueryRow{"id": "1", "title": "visible"}
	if err := registry.ValidateQueryResult(context.Background(), claim, valid); err != nil {
		t.Fatalf("valid row: %v", err)
	}
	if err := registry.ValidateQueryResult(context.Background(), claim, QueryRow{"id": "1", "title": 42}); !errors.Is(err, ErrResultInvalid) {
		t.Fatalf("invalid row=%v", err)
	}
	resolved, err := registry.Resolve(declaration.ID)
	if err != nil || resolved.ResultSchemaDigest != wantDigest || resolved.boundResultSchema != nil {
		t.Fatalf("resolved query=%#v err=%v", resolved, err)
	}

	snapshot := registry.Snapshot()
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("private_schema_marker")) || !bytes.Contains(encoded, []byte(wantDigest)) {
		t.Fatalf("snapshot exposed raw Schema or omitted digest: %s", encoded)
	}
	restored := New()
	if _, err := restored.ReplaceAll(snapshot.Publications, false); err != nil {
		t.Fatalf("restore private publication material: %v", err)
	}
	if err := restored.ValidateQueryResult(context.Background(), claim, valid); err != nil {
		t.Fatalf("restored validator: %v", err)
	}

	// Recompiling the same exact bytes produces another validator pointer but is
	// still an idempotent exact-artifact replay.
	reboundBinding := resultSchemaBinding(publication, declaration, []byte(schemaText))
	rebound, err := BindResultSchemas(publication, []JSONResultSchemaBinding{reboundBinding})
	if err != nil {
		t.Fatal(err)
	}
	if replayRevision, err := registry.Publish(rebound); err != nil || replayRevision != revision {
		t.Fatalf("exact Schema replay: revision=%d err=%v", replayRevision, err)
	}

	forged := publication
	forged.Queries[0].ResultSchemaDigest = wantDigest
	if _, err := New().Publish(forged); !errors.Is(err, ErrInvalid) {
		t.Fatalf("public digest forged private Schema material: %v", err)
	}
	if _, removed, err := registry.Remove(publication.Artifact); err != nil || !removed {
		t.Fatalf("remove bound publication: removed=%t err=%v", removed, err)
	}
	if err := registry.ValidateQueryResult(context.Background(), claim, valid); !errors.Is(err, ErrResultInvalid) {
		t.Fatalf("removed validator remained callable: %v", err)
	}
}

func TestRegistryResultSchemaReplacementIsExactAndAtomic(t *testing.T) {
	source := publication("plugin.atomic-schema", false, 'a')
	sourceQuery := query("plugin.atomic-schema.items", "plugin.atomic-schema.item", PaginationNone, "public")
	sourceQuery.Relations = nil
	source.Queries = []QueryDeclaration{sourceQuery}
	sourceBody := []byte(`{"type":"object","required":["id"],"properties":{"id":{"type":"string"}},"additionalProperties":false}`)
	sourceBinding := resultSchemaBinding(source, sourceQuery, sourceBody)
	source, err := BindResultSchemas(source, []JSONResultSchemaBinding{sourceBinding})
	if err != nil {
		t.Fatal(err)
	}

	target := publication("plugin.atomic-schema", false, 'b')
	target.Artifact.ExtensionVersion = "2.0.0"
	target.Artifact.VersionID = 2
	target.Artifact.RuntimeInstanceID = "runtime-plugin.atomic-schema-v2"
	targetQuery := sourceQuery
	targetQuery.ContractVersion = "plugin.atomic-schema.items@2"
	targetQuery.PlanVersion = "plugin.atomic-schema.items.plan@2"
	targetQuery.ResultSchema = "plugin.atomic-schema.items.result@2"
	target.Queries = []QueryDeclaration{targetQuery}
	targetBody := []byte(`{"type":"object","required":["id"],"properties":{"id":{"type":"integer"}},"additionalProperties":false}`)
	targetBinding := resultSchemaBinding(target, targetQuery, targetBody)
	target, err = BindResultSchemas(target, []JSONResultSchemaBinding{targetBinding})
	if err != nil {
		t.Fatal(err)
	}

	registry := New()
	if _, err := registry.Publish(source); err != nil {
		t.Fatal(err)
	}
	sourceClaim := resultSchemaClaim(sourceBinding)
	if err := registry.ValidateQueryResult(context.Background(), sourceClaim, QueryRow{"id": "one"}); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.PublishIfArtifact(source.Artifact, target); err != nil {
		t.Fatal(err)
	}
	if err := registry.ValidateQueryResult(context.Background(), sourceClaim, QueryRow{"id": "one"}); !errors.Is(err, ErrResultInvalid) {
		t.Fatalf("source validator survived replacement: %v", err)
	}
	targetClaim := resultSchemaClaim(targetBinding)
	if err := registry.ValidateQueryResult(context.Background(), targetClaim, QueryRow{"id": json.Number("2")}); err != nil {
		t.Fatalf("target validator: %v", err)
	}

	forgedTarget := target
	forgedTarget.Queries[0].boundResultSchema = nil
	before := registry.Snapshot()
	if _, err := registry.Publish(forgedTarget); !errors.Is(err, ErrInvalid) {
		t.Fatalf("forged target replay=%v", err)
	}
	after := registry.Snapshot()
	if before.Revision != after.Revision || before.Digest != after.Digest {
		t.Fatal("failed Schema replay changed the Registry snapshot")
	}
}

func TestRegistrySafeModeFiltersForgedPluginSchemaBeforeValidation(t *testing.T) {
	core := publication("core.bound-schema", true, 'a')
	coreQuery := query("core.bound-schema.items", "core.bound-schema.item", PaginationNone, "public")
	coreQuery.Relations = nil
	core.Queries = []QueryDeclaration{coreQuery}
	coreBody := []byte(`{"type":"object","required":["id"],"properties":{"id":{"type":"string"}},"additionalProperties":false}`)
	coreBinding := resultSchemaBinding(core, coreQuery, coreBody)
	core, err := BindResultSchemas(core, []JSONResultSchemaBinding{coreBinding})
	if err != nil {
		t.Fatal(err)
	}
	forged := publication("plugin.forged-schema", false, 'b')
	forgedQuery := query("plugin.forged-schema.items", "plugin.forged-schema.item", PaginationNone, "public")
	forgedQuery.ResultSchemaDigest = strings.Repeat("f", 64)
	forged.Queries = []QueryDeclaration{forgedQuery}

	if _, err := New().ReplaceAll([]Publication{core, forged}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ordinary publication accepted forged Schema: %v", err)
	}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{core, forged}, true); err != nil {
		t.Fatalf("Safe Mode parsed filtered plugin Schema: %v", err)
	}
	claim := resultSchemaClaim(coreBinding)
	if err := registry.ValidateQueryResult(context.Background(), claim, QueryRow{"id": "core"}); err != nil {
		t.Fatalf("Safe Mode lost Core validator: %v", err)
	}
}

func TestBindResultSchemasRejectsUnownedMismatchedAndDuplicateMaterial(t *testing.T) {
	publication := publication("plugin.schema-owner", false, 'a')
	declaration := query("plugin.schema-owner.items", "plugin.schema-owner.item", PaginationNone, "public")
	publication.Queries = []QueryDeclaration{declaration}
	body := []byte(`{"type":"object"}`)
	binding := resultSchemaBinding(publication, declaration, body)

	tests := []struct {
		name     string
		bindings []JSONResultSchemaBinding
	}{
		{name: "unknown query", bindings: func() []JSONResultSchemaBinding {
			value := binding
			value.QueryID = "plugin.schema-owner.unknown"
			value.ContractVersion = value.QueryID + "@1"
			return []JSONResultSchemaBinding{value}
		}()},
		{name: "contract mismatch", bindings: func() []JSONResultSchemaBinding {
			value := binding
			value.ContractVersion = "plugin.schema-owner.items@2"
			return []JSONResultSchemaBinding{value}
		}()},
		{name: "artifact mismatch", bindings: func() []JSONResultSchemaBinding {
			value := binding
			value.Artifact.PackageDigest = strings.Repeat("b", 64)
			return []JSONResultSchemaBinding{value}
		}()},
		{name: "duplicate", bindings: []JSONResultSchemaBinding{binding, binding}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := BindResultSchemas(publication, test.bindings); !errors.Is(err, ErrExecutionInvalid) {
				t.Fatalf("BindResultSchemas=%v", err)
			}
		})
	}
}

func resultSchemaBinding(publication Publication, declaration QueryDeclaration, body []byte) JSONResultSchemaBinding {
	digest := sha256.Sum256(body)
	return JSONResultSchemaBinding{
		QueryID: declaration.ID, ContractVersion: declaration.ContractVersion,
		PlanVersion: declaration.PlanVersion, ResultSchema: declaration.ResultSchema,
		Artifact: publication.Artifact, SchemaDigest: hex.EncodeToString(digest[:]), Schema: body,
	}
}

func resultSchemaClaim(binding JSONResultSchemaBinding) ResultSchemaClaim {
	return ResultSchemaClaim{
		QueryID: binding.QueryID, ContractVersion: binding.ContractVersion,
		PlanVersion: binding.PlanVersion, ResultSchema: binding.ResultSchema, Artifact: binding.Artifact,
	}
}

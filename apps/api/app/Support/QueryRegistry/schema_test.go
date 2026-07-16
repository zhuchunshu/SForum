package queryregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

package extensionmanifest

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestManifestV3DatabaseOperationsValidateExactCatalog(t *testing.T) {
	manifest := completeV3Manifest()
	if err := Validate(manifest); err != nil {
		t.Fatalf("exact database operation catalog should validate: %v", err)
	}
	if got := len(manifest.Database.Operations); got != 2 {
		t.Fatalf("operation count = %d, want 2", got)
	}
	manifest.Database.Operations[1].QueryInvalidationTags = []string{"demo.v3.items"}
	if err := Validate(manifest); err != nil {
		t.Fatalf("execute invalidation tags should validate: %v", err)
	}
	manifest.Database.Operations[1].Parameters = []ManifestDatabaseParameter{}
	manifest.Database.Operations[1].ResultSchema = ""
	manifest.Database.Operations[1].Columns = []ManifestDatabaseColumn{}
	if err := Validate(manifest); err != nil {
		t.Fatalf("execute without parameters or a returning document should validate: %v", err)
	}
}

func TestManifestV3DatabaseOperationsRejectInvalidDeclarations(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Manifest)
	}{
		{name: "grants omit own schema", change: func(manifest *Manifest) {
			manifest.Database.Authority = ""
			manifest.Database.Grants = []string{DatabaseGrantCoreViews}
		}},
		{name: "foreign id", change: func(manifest *Manifest) { manifest.Database.Operations[0].ID = "other.plugin.query" }},
		{name: "duplicate id", change: func(manifest *Manifest) { manifest.Database.Operations[1].ID = manifest.Database.Operations[0].ID }},
		{name: "zero statement version", change: func(manifest *Manifest) { manifest.Database.Operations[0].StatementVersion = "0" }},
		{name: "non numeric statement version", change: func(manifest *Manifest) { manifest.Database.Operations[0].StatementVersion = "v1" }},
		{name: "unknown kind", change: func(manifest *Manifest) { manifest.Database.Operations[0].Kind = "raw" }},
		{name: "unsafe path", change: func(manifest *Manifest) { manifest.Database.Operations[0].Path = "../query.sql" }},
		{name: "wrong digest", change: func(manifest *Manifest) { manifest.Database.Operations[0].Digest = strings.Repeat("f", 64) }},
		{name: "wrong package file kind", change: func(manifest *Manifest) {
			manifest.PackageFiles[len(manifest.PackageFiles)-2].Kind = "asset"
		}},
		{name: "missing parameter list", change: func(manifest *Manifest) { manifest.Database.Operations[0].Parameters = nil }},
		{name: "too many parameters", change: func(manifest *Manifest) {
			parameter := manifest.Database.Operations[0].Parameters[0]
			manifest.Database.Operations[0].Parameters = make([]ManifestDatabaseParameter, manifestDatabaseMaximumParameters+1)
			for index := range manifest.Database.Operations[0].Parameters {
				manifest.Database.Operations[0].Parameters[index] = parameter
			}
		}},
		{name: "undeclared parameter schema", change: func(manifest *Manifest) { manifest.Database.Operations[0].Parameters[0].Schema = "schemas/input.json" }},
		{name: "invalid parameter field", change: func(manifest *Manifest) { manifest.Database.Operations[0].Parameters[0].Field = "bad:field" }},
		{name: "invalid parameter kind", change: func(manifest *Manifest) { manifest.Database.Operations[0].Parameters[0].Kind = "bytes" }},
		{name: "negative parameter bytes", change: func(manifest *Manifest) { manifest.Database.Operations[0].Parameters[0].MaxBytes = -1 }},
		{name: "parameter bytes over limit", change: func(manifest *Manifest) {
			manifest.Database.Operations[0].Parameters[0].MaxBytes = manifestDatabaseMaximumParameterSize + 1
		}},
		{name: "too many columns", change: func(manifest *Manifest) {
			manifest.Database.Operations[0].Columns = make([]ManifestDatabaseColumn, manifestDatabaseMaximumColumns+1)
			for index := range manifest.Database.Operations[0].Columns {
				manifest.Database.Operations[0].Columns[index] = ManifestDatabaseColumn{Name: "column"}
			}
		}},
		{name: "duplicate columns", change: func(manifest *Manifest) {
			manifest.Database.Operations[0].Columns = []ManifestDatabaseColumn{{Name: "ID"}, {Name: "id"}}
		}},
		{name: "query without result schema", change: func(manifest *Manifest) { manifest.Database.Operations[0].ResultSchema = "" }},
		{name: "query without columns", change: func(manifest *Manifest) { manifest.Database.Operations[0].Columns = nil }},
		{name: "query without row limit", change: func(manifest *Manifest) { manifest.Database.Operations[0].MaxRows = 0 }},
		{name: "query row limit over maximum", change: func(manifest *Manifest) { manifest.Database.Operations[0].MaxRows = manifestDatabaseMaximumRows + 1 }},
		{name: "query with execute limit", change: func(manifest *Manifest) { manifest.Database.Operations[0].MaxAffectedRows = 1 }},
		{name: "query with invalidation tags", change: func(manifest *Manifest) {
			manifest.Database.Operations[0].QueryInvalidationTags = []string{"demo.v3.items"}
		}},
		{name: "execute with query limit", change: func(manifest *Manifest) { manifest.Database.Operations[1].MaxRows = 1 }},
		{name: "execute with foreign invalidation tag", change: func(manifest *Manifest) {
			manifest.Database.Operations[1].QueryInvalidationTags = []string{"other.plugin.items"}
		}},
		{name: "execute with duplicate invalidation tags", change: func(manifest *Manifest) {
			manifest.Database.Operations[1].QueryInvalidationTags = []string{"demo.v3.items", " DEMO.V3.ITEMS "}
		}},
		{name: "execute with too many invalidation tags", change: func(manifest *Manifest) {
			manifest.Database.Operations[1].QueryInvalidationTags = make([]string, ManifestQueryMaximumCacheTags+1)
			for index := range manifest.Database.Operations[1].QueryInvalidationTags {
				manifest.Database.Operations[1].QueryInvalidationTags[index] = fmt.Sprintf("demo.v3.items.%02d", index)
			}
		}},
		{name: "execute without affected limit", change: func(manifest *Manifest) { manifest.Database.Operations[1].MaxAffectedRows = 0 }},
		{name: "execute affected limit over maximum", change: func(manifest *Manifest) {
			manifest.Database.Operations[1].MaxAffectedRows = manifestDatabaseMaximumAffectedRows + 1
		}},
		{name: "execute result without columns", change: func(manifest *Manifest) { manifest.Database.Operations[1].Columns = nil }},
		{name: "execute missing column list", change: func(manifest *Manifest) {
			manifest.Database.Operations[1].ResultSchema = ""
			manifest.Database.Operations[1].Columns = nil
		}},
		{name: "execute columns without result", change: func(manifest *Manifest) { manifest.Database.Operations[1].ResultSchema = "" }},
		{name: "negative timeout", change: func(manifest *Manifest) { manifest.Database.Operations[0].TimeoutMS = -1 }},
		{name: "timeout over maximum", change: func(manifest *Manifest) {
			manifest.Database.Operations[0].TimeoutMS = manifestDatabaseMaximumTimeoutMS + 1
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := completeV3Manifest()
			test.change(&manifest)
			if err := Validate(manifest); err == nil {
				t.Fatal("invalid database operation declaration must be rejected")
			}
		})
	}
}

func TestManifestV3DatabaseOperationsNormalizeEveryString(t *testing.T) {
	manifest := completeV3Manifest()
	operation := &manifest.Database.Operations[0]
	operation.ID = " DEMO.V3.Database.Items.Query "
	operation.StatementVersion = " 2 "
	operation.Kind = " QUERY "
	operation.Path = " database/items-query.sql "
	operation.Digest = " SHA256:" + strings.ToUpper(operation.Digest) + " "
	operation.ResultSchema = " demo.v3.database.items.result@1 "
	operation.Parameters[0].Schema = " demo.v3.database.item-id@1 "
	operation.Parameters[0].Field = " ITEM_ID "
	operation.Parameters[0].Kind = " INT64 "
	operation.Columns[0].Name = " ITEM_ID "
	manifest.Database.Operations[1].QueryInvalidationTags = []string{" DEMO.V3.ITEMS ", "demo.v3.members"}

	normalized := Normalize(manifest).Database.Operations[0]
	if normalized.ID != "demo.v3.database.items.query" || normalized.StatementVersion != "2" || normalized.Kind != "query" ||
		normalized.Path != "database/items-query.sql" || normalized.Digest != v3FixtureDigest() ||
		normalized.ResultSchema != "demo.v3.database.items.result@1" ||
		normalized.Parameters[0].Schema != "demo.v3.database.item-id@1" || normalized.Parameters[0].Field != "item_id" ||
		normalized.Parameters[0].Kind != "int64" || normalized.Columns[0].Name != "item_id" ||
		!slices.Equal(Normalize(manifest).Database.Operations[1].QueryInvalidationTags, []string{"demo.v3.items", "demo.v3.members"}) {
		t.Fatalf("database operation strings were not normalized: %#v", normalized)
	}
}

func TestManifestV3DatabaseOperationJSONSchemaRejectsInlineSQLAndDrift(t *testing.T) {
	manifest := completeV3Manifest()
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		t.Fatal(err)
	}
	database := root["database"].(map[string]any)
	operations := database["operations"].([]any)
	query := operations[0].(map[string]any)
	execute := operations[1].(map[string]any)
	execute["queryInvalidationTags"] = []any{"demo.v3.items"}
	executeInvalidator, _ := json.Marshal(root)
	if err := ValidateV3JSONSchema(executeInvalidator); err != nil {
		t.Fatalf("execute invalidation tags should satisfy raw schema: %v", err)
	}
	delete(execute, "queryInvalidationTags")

	query["sql"] = "SELECT * FROM secrets"
	inlineSQL, _ := json.Marshal(root)
	if err := ValidateV3JSONSchema(inlineSQL); err == nil {
		t.Fatal("inline SQL must be rejected by the raw JSON contract")
	}
	delete(query, "sql")

	parameter := query["parameters"].([]any)[0].(map[string]any)
	delete(parameter, "nullable")
	missingRequired, _ := json.Marshal(root)
	if err := ValidateV3JSONSchema(missingRequired); err == nil {
		t.Fatal("database parameter schema drift must be rejected")
	}
	parameter["nullable"] = false

	query["maxAffectedRows"] = float64(1)
	queryOnlyField, _ := json.Marshal(root)
	if err := ValidateV3JSONSchema(queryOnlyField); err == nil {
		t.Fatal("query and execute limits must be mutually exclusive")
	}
	delete(query, "maxAffectedRows")
	query["queryInvalidationTags"] = []any{"demo.v3.items"}
	queryInvalidator, _ := json.Marshal(root)
	if err := ValidateV3JSONSchema(queryInvalidator); err == nil {
		t.Fatal("query operations must not declare invalidation tags")
	}
}

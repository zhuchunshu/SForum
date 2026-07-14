package extensionmanifest

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestLegacyDatabaseAuthorityExpandsCumulatively(t *testing.T) {
	tests := []struct {
		authority string
		want      []string
	}{
		{DatabaseGrantOwnSchema, []string{DatabaseGrantOwnSchema}},
		{DatabaseGrantCoreViews, []string{DatabaseGrantOwnSchema, DatabaseGrantCoreViews}},
		{DatabaseGrantHostCommands, []string{DatabaseGrantOwnSchema, DatabaseGrantCoreViews, DatabaseGrantHostCommands}},
		{DatabaseGrantRawCore, []string{DatabaseGrantOwnSchema, DatabaseGrantCoreViews, DatabaseGrantHostCommands, DatabaseGrantRawCore}},
		{DatabaseGrantKernel, []string{DatabaseGrantOwnSchema, DatabaseGrantCoreViews, DatabaseGrantHostCommands, DatabaseGrantRawCore, DatabaseGrantKernel}},
	}
	for _, test := range tests {
		t.Run(test.authority, func(t *testing.T) {
			database := &ManifestDatabase{Authority: " " + test.authority + " "}
			normalizeDatabaseGrants(database)
			if database.Authority != "" || !reflect.DeepEqual(database.Grants, test.want) {
				t.Fatalf("normalized database = %#v, want grants %#v", database, test.want)
			}
		})
	}
}

func TestDatabaseGrantsNormalizeAsCanonicalExactSet(t *testing.T) {
	database := &ManifestDatabase{Grants: []string{" RAW_CORE ", "own_schema", "host_commands", "core_views"}}
	normalizeDatabaseGrants(database)
	want := []string{DatabaseGrantOwnSchema, DatabaseGrantCoreViews, DatabaseGrantHostCommands, DatabaseGrantRawCore}
	if !reflect.DeepEqual(database.Grants, want) || !HasDatabaseGrant(database, DatabaseGrantRawCore) || HasDatabaseGrant(database, DatabaseGrantKernel) {
		t.Fatalf("canonical grants = %#v", database.Grants)
	}
}

func TestManifestV3DatabaseGrantJSONContractIsExclusive(t *testing.T) {
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
	database["grants"] = []any{DatabaseGrantOwnSchema}
	if invalid, err := json.Marshal(root); err != nil || ValidateV3JSONSchema(invalid) == nil {
		t.Fatalf("authority plus grants must be rejected: %v", err)
	}
	delete(database, "authority")
	if valid, err := json.Marshal(root); err != nil || ValidateV3JSONSchema(valid) != nil {
		t.Fatalf("exact grants must validate: %v", err)
	}
	database["grants"] = []any{DatabaseGrantOwnSchema, DatabaseGrantOwnSchema}
	if invalid, err := json.Marshal(root); err != nil || ValidateV3JSONSchema(invalid) == nil {
		t.Fatalf("duplicate grants must be rejected: %v", err)
	}
}

func TestLegacyRawDatabaseFixtureNormalizesExactGrants(t *testing.T) {
	body, err := os.ReadFile("testdata/v3/raw-db-plugin.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest = Normalize(manifest)
	if err := Validate(manifest); err != nil {
		t.Fatal(err)
	}
	want := []string{DatabaseGrantOwnSchema, DatabaseGrantCoreViews, DatabaseGrantHostCommands, DatabaseGrantRawCore}
	if manifest.Database == nil || manifest.Database.Authority != "" || !reflect.DeepEqual(manifest.Database.Grants, want) {
		t.Fatalf("legacy raw database = %#v, want grants %#v", manifest.Database, want)
	}
}

func TestManifestV3DatabaseGrantSemantics(t *testing.T) {
	t.Run("exact independent grants", func(t *testing.T) {
		manifest := completeV3Manifest()
		manifest.Migrations = nil
		manifest.Database.Authority = ""
		manifest.Database.Grants = []string{DatabaseGrantCoreViews, DatabaseGrantHostCommands}
		manifest.Database.Schema = ""
		manifest.Database.Role = ""
		manifest.Database.Operations = nil
		if err := Validate(manifest); err != nil {
			t.Fatalf("independent grants = %v", err)
		}
	})

	t.Run("raw grant requires compatibility", func(t *testing.T) {
		manifest := completeV3Manifest()
		manifest.Database.Authority = ""
		manifest.Database.Grants = []string{DatabaseGrantRawCore}
		manifest.Database.Schema = ""
		manifest.Database.Role = ""
		manifest.Database.Operations = nil
		manifest.Database.CoreCompatibility = ""
		if err := Validate(manifest); err == nil {
			t.Fatal("raw_core without compatibility must be rejected")
		}
	})

	t.Run("operations require own schema grant", func(t *testing.T) {
		manifest := completeV3Manifest()
		manifest.Database.Authority = ""
		manifest.Database.Grants = []string{DatabaseGrantHostCommands}
		manifest.Database.Schema = ""
		manifest.Database.Role = ""
		if err := Validate(manifest); err == nil {
			t.Fatal("database operations without own_schema must be rejected")
		}
	})

	t.Run("raw declarations cannot mix compatibility inputs", func(t *testing.T) {
		manifest := completeV3Manifest()
		manifest.Database.Grants = []string{DatabaseGrantOwnSchema}
		if err := Validate(manifest); err == nil {
			t.Fatal("authority and grants must be rejected")
		}
	})
}

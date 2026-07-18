package extensionmanifest

import (
	"encoding/json"
	"testing"
)

func TestManifestV3QueryRuntimeKeepsLegacyDeclarationsInspectOnly(t *testing.T) {
	manifest := completeV3Manifest()
	normalized := Normalize(manifest)
	query := normalized.Queries[0]
	if query.Handler != "" || len(query.IdentityFields) != 0 || len(query.DefaultSort) != 0 ||
		len(normalized.QueryResultFilters) != 0 {
		t.Fatalf("legacy query gained executable metadata: %#v", query)
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("legacy declaration-only query must stay valid: %v", err)
	}
}

func TestManifestV3QueryRuntimeNormalizesExecutableProviderAndFilter(t *testing.T) {
	manifest := completeExecutableQueryManifest()
	manifest.Queries[0].CacheTags = []string{" DEMO.V3.ITEMS "}
	manifest.QueryResultFilters[0].FailurePolicy = ""
	manifest.QueryResultFilters[0].TimeoutMS = 0
	normalized := Normalize(manifest)
	filter := normalized.QueryResultFilters[0]
	if filter.FailurePolicy != QueryResultFilterFailureFailClosed ||
		filter.TimeoutMS != ManifestQueryResultFilterDefaultTimeoutMS {
		t.Fatalf("result filter defaults = %#v", filter)
	}
	if len(normalized.Queries[0].CacheTags) != 1 || normalized.Queries[0].CacheTags[0] != "demo.v3.items" {
		t.Fatalf("query cache tags=%#v", normalized.Queries[0].CacheTags)
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("executable query contract: %v", err)
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("embedded query runtime schema: %v", err)
	}
}

func TestManifestV3QueryRuntimeSchemaRejectsMetadataWithoutHandler(t *testing.T) {
	manifest := completeExecutableQueryManifest()
	manifest.Queries[0].Handler = ""
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err == nil {
		t.Fatal("handlerless query accepted executable identity and sort metadata")
	}
}

func TestManifestV3QueryRuntimeAcceptsOrderedMultiFieldIdentitySuffix(t *testing.T) {
	manifest := completeExecutableQueryManifest()
	query := &manifest.Queries[0]
	query.Fields = append(query.Fields, "tenant_id")
	query.Sort = []string{"created_at", "tenant_id", "id"}
	query.IdentityFields = []string{"tenant_id", "id"}
	query.DefaultSort = []ManifestQuerySort{
		{Field: "created_at", Descending: true},
		{Field: "tenant_id"},
		{Field: "id", Descending: true},
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("ordered multi-field identity suffix: %v", err)
	}
	query.DefaultSort[1], query.DefaultSort[2] = query.DefaultSort[2], query.DefaultSort[1]
	if err := Validate(manifest); err == nil {
		t.Fatal("reordered multi-field identity suffix was accepted")
	}
}

func TestManifestV3QueryRuntimeRejectsUnsafeShapes(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Manifest)
	}{
		{name: "handler without identity", change: func(value *Manifest) { value.Queries[0].IdentityFields = nil }},
		{name: "handler without default sort", change: func(value *Manifest) { value.Queries[0].DefaultSort = nil }},
		{name: "identity outside fields", change: func(value *Manifest) { value.Queries[0].IdentityFields = []string{"missing"} }},
		{name: "identity outside sort allowlist", change: func(value *Manifest) { value.Queries[0].Sort = []string{"created_at"} }},
		{name: "unstable identity suffix", change: func(value *Manifest) {
			value.Queries[0].DefaultSort = []ManifestQuerySort{{Field: "id"}, {Field: "created_at", Descending: true}}
		}},
		{name: "duplicate default sort", change: func(value *Manifest) {
			value.Queries[0].DefaultSort = append(value.Queries[0].DefaultSort, ManifestQuerySort{Field: "id"})
		}},
		{name: "ownerless cache tag", change: func(value *Manifest) { value.Queries[0].CacheTags = []string{"items"} }},
		{name: "foreign cache tag", change: func(value *Manifest) { value.Queries[0].CacheTags = []string{"other.plugin.items"} }},
		{name: "duplicate cache tag", change: func(value *Manifest) {
			value.Queries[0].CacheTags = []string{"demo.v3.items", " DEMO.V3.ITEMS "}
		}},
		{name: "executable query without protocol v2", change: func(value *Manifest) { value.Backend.ProtocolVersion = 1 }},
		{name: "executable query without schema file", change: func(value *Manifest) {
			value.PackageFiles = value.PackageFiles[:len(value.PackageFiles)-1]
		}},
		{name: "self filter target contract drift", change: func(value *Manifest) {
			value.QueryResultFilters[0].QueryContractVersion = "demo.v3.query.items@2"
		}},
		{name: "self filter cannot declare dependency", change: func(value *Manifest) {
			value.QueryResultFilters[0].Dependency = &ManifestQueryResultFilterDependency{ExtensionID: "demo.v3", VersionConstraint: "^1.0.0"}
		}},
		{name: "filter timeout overflow", change: func(value *Manifest) {
			value.QueryResultFilters[0].TimeoutMS = ManifestQueryResultFilterMaximumTimeoutMS + 1
		}},
		{name: "filter priority overflow", change: func(value *Manifest) {
			value.QueryResultFilters[0].Priority = ManifestQueryResultFilterMaximumPriority + 1
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := completeExecutableQueryManifest()
			test.change(&manifest)
			if err := Validate(manifest); err == nil {
				t.Fatal("unsafe executable query contract was accepted")
			}
		})
	}
}

func TestManifestV3QueryResultFilterRequiresExactCrossPluginDependency(t *testing.T) {
	manifest := completeExecutableQueryManifest()
	filter := &manifest.QueryResultFilters[0]
	filter.QueryID = "owner.plugin.query.items"
	filter.QueryContractVersion = "owner.plugin.query.items@1"
	filter.QueryPlanVersion = "owner.plugin.query.items.plan@1"
	filter.Dependency = &ManifestQueryResultFilterDependency{
		ExtensionID: "owner.plugin", VersionConstraint: "^2.0.0",
	}
	manifest.Dependencies = append(manifest.Dependencies, ManifestDependency{
		ID: "owner.plugin", Version: "^2.0.0", Kind: "required",
	})
	if err := Validate(manifest); err != nil {
		t.Fatalf("exact required owner dependency: %v", err)
	}
	manifest.Dependencies[len(manifest.Dependencies)-1].Kind = "optional"
	if err := Validate(manifest); err == nil {
		t.Fatal("fail-closed filter accepted an optional owner dependency")
	}
	manifest.QueryResultFilters[0].FailurePolicy = QueryResultFilterFailureFailOpen
	if err := Validate(manifest); err != nil {
		t.Fatalf("fail-open filter should accept an exact optional owner dependency: %v", err)
	}
}

func TestManifestV3QueryResultFilterCannotDeclareItsOwnIdentity(t *testing.T) {
	manifest := completeExecutableQueryManifest()
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatal(err)
	}
	filters := document["queryResultFilters"].([]any)
	filters[0].(map[string]any)["identityFields"] = []any{"nonce"}
	body, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err == nil {
		t.Fatal("result filter must not choose the row identity it is required to preserve")
	}
}

func completeExecutableQueryManifest() Manifest {
	manifest := completeV3Manifest()
	query := &manifest.Queries[0]
	query.Fields = []string{"id", "title", "created_at"}
	query.Sort = []string{"created_at", "id"}
	query.Handler = "demo.v3.query.items.execute"
	query.IdentityFields = []string{"id"}
	query.DefaultSort = []ManifestQuerySort{
		{Field: "created_at", Descending: true},
		{Field: "id", Descending: true},
	}
	manifest.PackageFiles = append(manifest.PackageFiles, ManifestPackageFile{
		ID: "demo.v3.query.items.result", Kind: "schema", Path: "schemas/query-items-result.json",
		Digest: v3FixtureDigest(), Version: "1",
	})
	manifest.QueryResultFilters = []ManifestQueryResultFilter{{
		ID: "demo.v3.query.items.decorate", ContractVersion: "demo.v3.query.items.decorate@1",
		QueryID: query.ID, QueryContractVersion: query.ContractVersion, QueryPlanVersion: query.PlanVersion,
		Handler: "demo.v3.query.items.decorate", Priority: 100,
		FailurePolicy: QueryResultFilterFailureFailClosed, TimeoutMS: ManifestQueryResultFilterDefaultTimeoutMS,
	}}
	return manifest
}

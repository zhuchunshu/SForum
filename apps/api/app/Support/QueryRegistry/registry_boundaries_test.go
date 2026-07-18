package queryregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestRegistryEnforcesPublicationAndQueryLimits(t *testing.T) {
	publications := make([]Publication, 0, maxPublications+1)
	for index := 0; index < maxPublications+1; index++ {
		publications = append(publications, publication(fmt.Sprintf("limit.plugin.%03d", index), false, 'a'))
	}
	if _, err := New().ReplaceAll(publications[:maxPublications], false); err != nil {
		t.Fatalf("maximum publications rejected=%v", err)
	}
	if _, err := New().ReplaceAll(publications, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("publication overflow=%v", err)
	}

	queryBound := publication("limit.queries", false, 'c')
	queryBound.Queries = make([]QueryDeclaration, maxQueriesPerPublication+1)
	for index := range queryBound.Queries {
		id := fmt.Sprintf("limit.queries.item.%03d", index)
		queryBound.Queries[index] = QueryDeclaration{
			ID: id, ContractVersion: id + "@1", Entity: id + ".entity",
			PlanVersion: id + ".plan@1", Fields: []string{"id"}, Pagination: PaginationNone,
			ResultSchema: id + ".result@1", PermissionPolicy: "public",
		}
	}
	if _, err := New().ReplaceAll([]Publication{queryBound}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("query overflow=%v", err)
	}
}

func TestRegistryEnforcesFieldFilterSortAndCacheTagLimits(t *testing.T) {
	fields := make([]string, maxFieldsPerQuery+1)
	for index := range fields {
		fields[index] = fmt.Sprintf("field.%03d", index)
	}
	overFields := publication("limit.fields", false, 'a')
	overFields.Queries = []QueryDeclaration{{
		ID: "limit.fields.items", ContractVersion: "limit.fields.items@1", Entity: "limit.fields.item",
		PlanVersion: "limit.fields.items.plan@1", Fields: fields, Pagination: PaginationNone,
		ResultSchema: "limit.fields.items.result@1", PermissionPolicy: "public",
	}}
	if _, err := New().ReplaceAll([]Publication{overFields}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("field overflow=%v", err)
	}

	tags := make([]string, maxCacheTagsPerQuery+1)
	for index := range tags {
		tags[index] = fmt.Sprintf("limit.tags.tag.%03d", index)
	}
	overTags := publication("limit.tags", false, 'b')
	overTags.Queries = []QueryDeclaration{query("limit.tags.items", "limit.tags.item", PaginationNone, "public")}
	overTags.Queries[0].CacheTags = tags
	if _, err := New().ReplaceAll([]Publication{overTags}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cache tag overflow=%v", err)
	}
	overTags.Queries[0].CacheTags = tags[:maxCacheTagsPerQuery]
	if _, err := New().ReplaceAll([]Publication{overTags}, false); err != nil {
		t.Fatalf("maximum legal owner-scoped cache tags rejected=%v", err)
	}

	dupFields := publication("limit.dup", false, 'c')
	dupFields.Queries = []QueryDeclaration{query("limit.dup.items", "limit.dup.item", PaginationNone, "public")}
	dupFields.Queries[0].Fields = []string{"id", "id"}
	if _, err := New().ReplaceAll([]Publication{dupFields}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("duplicate field names=%v", err)
	}
}

func TestRegistryRequiresOwnerScopedCacheTags(t *testing.T) {
	tests := []struct {
		name string
		tags []string
	}{
		{name: "ownerless", tags: []string{"items"}},
		{name: "foreign owner", tags: []string{"other.plugin.items"}},
		{name: "invalid stable id", tags: []string{"cache.owner/items"}},
		{name: "duplicate after normalization", tags: []string{"cache.owner.items", " CACHE.OWNER.ITEMS "}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := publication("cache.owner", false, 'c')
			candidate.Queries = []QueryDeclaration{query("cache.owner.items", "cache.owner.item", PaginationNone, PermissionPolicyPublic)}
			candidate.Queries[0].CacheTags = test.tags
			if _, err := New().Publish(candidate); !errors.Is(err, ErrInvalid) {
				t.Fatalf("cache tags %#v err=%v", test.tags, err)
			}
		})
	}

	candidate := publication("cache.owner", false, 'd')
	candidate.Queries = []QueryDeclaration{query("cache.owner.items", "cache.owner.item", PaginationNone, PermissionPolicyPublic)}
	candidate.Queries[0].CacheTags = []string{" CACHE.OWNER.ITEMS "}
	registry := New()
	if _, err := registry.Publish(candidate); err != nil {
		t.Fatalf("canonical owner tag: %v", err)
	}
	published, ok := registry.SnapshotPublication("cache.owner")
	if !ok || len(published.Queries) != 1 || !slices.Equal(published.Queries[0].CacheTags, []string{"cache.owner.items"}) {
		t.Fatalf("normalized cache tags=%#v", published.Queries)
	}
}

func TestRegistryRejectsInvalidArtifactAndPermissionPolicy(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Publication)
	}{
		{name: "core mismatch", edit: func(p *Publication) { p.Artifact.Core = true }},
		{name: "bad digest", edit: func(p *Publication) { p.Artifact.PackageDigest = "not-hex" }},
		{name: "bad version", edit: func(p *Publication) { p.Artifact.ExtensionVersion = "v1" }},
		{name: "long version", edit: func(p *Publication) {
			p.Artifact.ExtensionVersion = "1.0.0+" + strings.Repeat("a", maxExtensionVersionLength)
		}},
		{name: "long runtime", edit: func(p *Publication) { p.Artifact.RuntimeInstanceID = strings.Repeat("r", maxRuntimeInstanceIDLength+1) }},
		{name: "runtime control", edit: func(p *Publication) { p.Artifact.RuntimeInstanceID = "runtime\nforged" }},
		{name: "bad pagination", edit: func(p *Publication) { p.Queries[0].Pagination = "page" }},
		{name: "empty fields", edit: func(p *Publication) { p.Queries[0].Fields = nil }},
		{name: "bad schema", edit: func(p *Publication) { p.Queries[0].ResultSchema = "NoCaps" }},
		{name: "long plan version", edit: func(p *Publication) { p.Queries[0].PlanVersion = strings.Repeat("a", maxSchemaRefLength+1) + "@1" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := publication("plugin.query", false, 'a')
			item.Queries = []QueryDeclaration{query("plugin.query.items", "plugin.item", PaginationNone, "public")}
			test.edit(&item)
			if _, err := New().ReplaceAll([]Publication{item}, false); !errors.Is(err, ErrInvalid) && !errors.Is(err, ErrConflict) {
				t.Fatalf("expected invalid/conflict, got %v", err)
			}
		})
	}
}

func TestRegistryCoreAuthorityCannotBeForgedByFlagPrefixOrJSON(t *testing.T) {
	compatiblePlugin := publication("core.compatible-plugin", false, 'c')
	compatiblePlugin.Queries = []QueryDeclaration{
		query("core.compatible-plugin.items", "core.compatible-plugin.item", PaginationNone, "public"),
	}
	if _, err := New().Publish(compatiblePlugin); err != nil {
		t.Fatalf("core.* plugin namespace was incorrectly treated as Host Core = %v", err)
	}

	forged := publication("plugin.query", false, 'a')
	forged.Artifact.ExtensionID = "core.forged"
	forged.Artifact.Core = true
	forged.Artifact.VersionID = 0
	forged.Artifact.RuntimeInstanceID = ""
	forged.Queries = []QueryDeclaration{query("core.forged.items", "core.forged.item", PaginationNone, "public")}

	if _, err := New().Publish(forged); !errors.Is(err, ErrInvalid) {
		t.Fatalf("flag/prefix forged Core publication = %v", err)
	}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{forged}, true); err != nil {
		t.Fatalf("forged Core input blocked Safe Mode recovery = %v", err)
	}
	if snapshot := registry.Snapshot(); !snapshot.SafeMode || len(snapshot.Publications) != 0 {
		t.Fatalf("Safe Mode retained forged Core publication = %#v", snapshot)
	}

	trusted := publication("core.query", true, 'b')
	body, err := json.Marshal(trusted.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Artifact
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	trusted.Artifact = decoded
	if _, err := New().Publish(trusted); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unsealed decoded Core publication = %v", err)
	}
}

func TestRegistryArtifactAndPlanNumericBounds(t *testing.T) {
	plugin := publication("bounds.query", false, 'a')
	plugin.Artifact.RuntimeInstanceID = strings.Repeat("r", maxRuntimeInstanceIDLength)
	plugin.Artifact.ExtensionVersion = "1.0.0+" + strings.Repeat("a", maxExtensionVersionLength-len("1.0.0+"))
	declaration := query("bounds.query.items", "bounds.item", PaginationNone, "public")
	declaration.PlanVersion = strings.Repeat("a", maxSchemaRefLength-len("@1")) + "@1"
	plugin.Queries = []QueryDeclaration{declaration}
	registry := newPlanningRegistry().WithPluginAdmission(func(artifact Artifact) bool {
		return artifact == plugin.Artifact
	})
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatalf("exact artifact/declaration bounds rejected = %v", err)
	}
	if _, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: declaration.ID, MaxCost: maxCostValue, Permission: PermissionInput{},
	}); err != nil {
		t.Fatalf("exact MaxCost bound rejected = %v", err)
	}
	if _, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: declaration.ID, MaxCost: maxCostValue + 1, Permission: PermissionInput{},
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("MaxCost overflow = %v", err)
	}

	overflowPolicy := New(WithCostPolicy(CostPolicyFunc(func(QueryCostInput) (QueryCost, error) {
		return QueryCost{Units: 1, Maximum: maxCostValue + 1}, nil
	}))).WithPluginAdmission(func(artifact Artifact) bool {
		return artifact == plugin.Artifact
	})
	if _, err := overflowPolicy.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	if _, err := overflowPolicy.Plan(context.Background(), PlanRequest{
		QueryID: declaration.ID, Permission: PermissionInput{},
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cost policy overflow = %v", err)
	}
}

func TestRegistryAlignsOpaqueNamesAndPackageSchemaWithManifest(t *testing.T) {
	item := publication("plugin.query", false, 'a')
	declaration := query("plugin.query.items", "plugin.item", PaginationNone, "public")
	declaration.Fields = []string{"displayName", "profile:summary"}
	declaration.ResultSchema = "schemas/query-result.json"
	item.Queries = []QueryDeclaration{declaration}
	registry := New()
	if _, err := registry.Publish(item); err != nil {
		t.Fatalf("manifest-compatible declaration rejected = %v", err)
	}
	resolved, err := registry.Resolve(declaration.ID)
	if err != nil || resolved.Fields[0] != "displayName" || resolved.ResultSchema != "schemas/query-result.json" {
		t.Fatalf("opaque declaration changed = %#v, %v", resolved, err)
	}
}

func TestRegistryRemoveRequiresExactArtifactTuple(t *testing.T) {
	active := publication("exact.query", false, 'a')
	active.Queries = []QueryDeclaration{query("exact.query.items", "exact.item", PaginationNone, "public")}
	registry := New()
	if _, err := registry.Publish(active); err != nil {
		t.Fatal(err)
	}
	before := registry.CacheState()
	tests := []struct {
		name string
		edit func(*Artifact)
		want error
	}{
		{name: "version", edit: func(v *Artifact) { v.ExtensionVersion = "1.0.1" }, want: ErrArtifactConflict},
		{name: "package", edit: func(v *Artifact) { v.PackageDigest = strings.Repeat("b", 64) }, want: ErrArtifactConflict},
		{name: "version id", edit: func(v *Artifact) { v.VersionID++ }, want: ErrArtifactConflict},
		{name: "runtime", edit: func(v *Artifact) { v.RuntimeInstanceID = "runtime-replacement" }, want: ErrArtifactConflict},
		{name: "core", edit: func(v *Artifact) { v.Core = true }, want: ErrInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stale := active.Artifact
			test.edit(&stale)
			revision, removed, err := registry.Remove(stale)
			if !errors.Is(err, test.want) || removed || revision != before.Revision {
				t.Fatalf("remove: revision=%d removed=%t err=%v", revision, removed, err)
			}
		})
	}
}

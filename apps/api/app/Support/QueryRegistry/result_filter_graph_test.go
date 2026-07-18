package queryregistry

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
)

func TestRegistryDerivesCrossPluginFilterIdentityFromOwnerGraph(t *testing.T) {
	owner := crossFilterQueryOwner("1.0.0", 'a', []string{"tenant_id", "id"})
	filter := crossFilterPublication(owner.Queries[0], []string{"title"})
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{filter, owner}, false); err != nil {
		t.Fatal(err)
	}

	first := registry.Snapshot()
	if got := snapshotFilterIdentity(t, first, filter.Artifact.ExtensionID); !reflect.DeepEqual(got, []string{"tenant_id", "id"}) {
		t.Fatalf("Host-derived filter identity=%#v", got)
	}

	upgraded := crossFilterQueryOwner("2.0.0", 'b', []string{"id"})
	if _, err := registry.ReplaceAll([]Publication{upgraded, filter}, false); err != nil {
		t.Fatalf("owner upgrade with stable filter artifact: %v", err)
	}
	second := registry.Snapshot()
	if got := snapshotFilterIdentity(t, second, filter.Artifact.ExtensionID); !reflect.DeepEqual(got, []string{"id"}) {
		t.Fatalf("upgraded Host-derived filter identity=%#v", got)
	}
	if got := snapshotFilterIdentity(t, first, filter.Artifact.ExtensionID); !reflect.DeepEqual(got, []string{"tenant_id", "id"}) {
		t.Fatalf("old snapshot identity mutated=%#v", got)
	}
	if first.Digest == second.Digest {
		t.Fatal("owner identity upgrade did not change graph digest")
	}

	if _, err := registry.ReplaceAll([]Publication{filter}, false); err != nil {
		t.Fatalf("optional owner removal: %v", err)
	}
	missing := registry.Snapshot()
	if got := snapshotFilterIdentity(t, missing, filter.Artifact.ExtensionID); len(got) != 0 {
		t.Fatalf("missing owner retained forged identity=%#v", got)
	}
	if _, err := registry.ReplaceAll([]Publication{filter, upgraded}, false); err != nil {
		t.Fatalf("optional owner restore: %v", err)
	}
	if got := snapshotFilterIdentity(t, registry.Snapshot(), filter.Artifact.ExtensionID); !reflect.DeepEqual(got, []string{"id"}) {
		t.Fatalf("restored owner identity=%#v", got)
	}
}

func TestRegistryMakesOptionalFailOpenFilterDormantOnOwnerVersionDrift(t *testing.T) {
	compatible := crossFilterQueryOwner("2.0.0", 'b', []string{"id"})
	filter := crossFilterPublication(compatible.Queries[0], nil)
	filter.ResultFilters[0].FailurePolicy = ResultFilterFailOpen
	filter.ResultFilters[0].Dependency.VersionConstraint = "^2.0.0"
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{filter, compatible}, false); err != nil {
		t.Fatal(err)
	}
	if got := snapshotFilterIdentity(t, registry.Snapshot(), filter.Artifact.ExtensionID); !reflect.DeepEqual(got, []string{"id"}) {
		t.Fatalf("compatible owner identity=%#v", got)
	}

	incompatible := crossFilterQueryOwner("1.0.0", 'a', []string{"tenant_id", "id"})
	if _, err := registry.ReplaceAll([]Publication{filter, incompatible}, false); err != nil {
		t.Fatalf("owner version drift with stable optional filter: %v", err)
	}
	if got := snapshotFilterIdentity(t, registry.Snapshot(), filter.Artifact.ExtensionID); len(got) != 0 {
		t.Fatalf("incompatible owner retained filter identity=%#v", got)
	}
	if _, err := registry.ReplaceAll([]Publication{filter, compatible}, false); err != nil {
		t.Fatalf("compatible owner restore: %v", err)
	}
	if got := snapshotFilterIdentity(t, registry.Snapshot(), filter.Artifact.ExtensionID); !reflect.DeepEqual(got, []string{"id"}) {
		t.Fatalf("restored compatible owner identity=%#v", got)
	}
}

func TestDynamicResultFilterSourceRejectsMoreThanExecutionBound(t *testing.T) {
	runtime := &ExecutionRuntime{filterSource: ResultFilterSourceFunc(func(QueryContribution) ([]ResultFilterRegistration, error) {
		return make([]ResultFilterRegistration, maximumResultFilters+1), nil
	})}
	if _, err := runtime.resultFilterCandidates(QueryContribution{}); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("dynamic filter bound error=%v", err)
	}
}

func TestResultFilterWithoutHostIdentityNeverExecutes(t *testing.T) {
	var providerCalls atomic.Int32
	var filterCalls atomic.Int32
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		providerCalls.Add(1)
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "owner", "title": "base"}}}, nil
	})
	artifact := publication("plugin.identityless-filter", false, 'e').Artifact
	filter := executionTestFilter(artifact, "plugin.identityless-filter.replace", 10, ResultFilterFailOpen, func([]QueryRow) []QueryRow {
		filterCalls.Add(1)
		return []QueryRow{{"id": "replacement", "title": "filtered"}}
	})
	filter.IdentityFields = nil
	runtime, _ := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider,
		[]ResultFilterRegistration{filter}, func(config *ExecutionConfig) {
			config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
		})
	result, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"})
	if err != nil || len(result.Rows) != 1 || result.Rows[0]["id"] != "owner" ||
		providerCalls.Load() != 1 || filterCalls.Load() != 0 {
		t.Fatalf("fail-open identityless filter: result=%#v err=%v provider=%d filter=%d",
			result, err, providerCalls.Load(), filterCalls.Load())
	}

	filter.FailurePolicy = ResultFilterFailClosed
	if _, _, err := executionTestRuntimeError(t, PaginationNone, PermissionPolicyPublic, provider,
		[]ResultFilterRegistration{filter}, func(config *ExecutionConfig) {
			config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
		}); !errors.Is(err, ErrContractInsufficient) || providerCalls.Load() != 1 || filterCalls.Load() != 0 {
		t.Fatalf("fail-closed identityless filter: err=%v provider=%d filter=%d",
			err, providerCalls.Load(), filterCalls.Load())
	}
}

func TestOptionalFailOpenFilterSkipsIncompatibleOwnerVersion(t *testing.T) {
	var providerCalls atomic.Int32
	var filterCalls atomic.Int32
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		providerCalls.Add(1)
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "owner", "title": "base"}}}, nil
	})
	artifact := publication("plugin.versioned-filter", false, 'e').Artifact
	filter := executionTestFilter(artifact, "plugin.versioned-filter.decorate", 10, ResultFilterFailOpen, func(rows []QueryRow) []QueryRow {
		filterCalls.Add(1)
		return rows
	})
	filter.Dependency.VersionConstraint = "^2.0.0"
	runtime, registry := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider,
		[]ResultFilterRegistration{filter}, func(config *ExecutionConfig) {
			config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
		})
	query, err := registry.Resolve("core.execute.items")
	if err != nil {
		t.Fatal(err)
	}
	selected, evidence, err := runtime.matchingFiltersWithEvidence(query)
	if err != nil || len(selected) != 0 || len(evidence) != 1 ||
		evidence[0].Outcome != ResultFilterTraceDependencyMismatch {
		t.Fatalf("version-drift match: selected=%#v evidence=%#v err=%v", selected, evidence, err)
	}
	result, err := runtime.Execute(t.Context(), PlanRequest{QueryID: query.ID})
	if err != nil || len(result.Rows) != 1 || result.Rows[0]["id"] != "owner" ||
		providerCalls.Load() != 1 || filterCalls.Load() != 0 {
		t.Fatalf("version-drift execution: result=%#v err=%v provider=%d filter=%d",
			result, err, providerCalls.Load(), filterCalls.Load())
	}
}

func crossFilterQueryOwner(version string, digest byte, identity []string) Publication {
	owner := publication("plugin.query-owner", false, digest)
	owner.Artifact.ExtensionVersion = version
	owner.Artifact.VersionID++
	owner.Artifact.RuntimeInstanceID = "runtime-query-owner-" + version
	declaration := query("plugin.query-owner.items", "plugin.query-owner.item", PaginationOffset, PermissionPolicyPublic)
	declaration.Relations = nil
	declaration.Fields = []string{"tenant_id", "id", "title"}
	declaration.Sort = []string{"tenant_id", "id"}
	if len(identity) == 1 {
		declaration.Sort = []string{"id"}
	}
	declaration.Handler = "plugin.query-owner.items"
	declaration.IdentityFields = append([]string(nil), identity...)
	declaration.DefaultSort = make([]SortValue, 0, len(identity))
	for _, field := range identity {
		declaration.DefaultSort = append(declaration.DefaultSort, SortValue{Field: field})
	}
	owner.Queries = []QueryDeclaration{declaration}
	return owner
}

func crossFilterPublication(target QueryDeclaration, forgedIdentity []string) Publication {
	filter := publication("plugin.query-filter", false, 'f')
	filter.ResultFilters = []ResultFilterDeclaration{{
		ID: "plugin.query-filter.items.mask", ContractVersion: "plugin.query-filter.items.mask@1",
		QueryID: target.ID, QueryContractVersion: target.ContractVersion,
		QueryPlanVersion: target.PlanVersion, Handler: "plugin.query-filter.items.mask",
		FailurePolicy: ResultFilterFailClosed, TimeoutMS: 500,
		Dependency: &ResultFilterDependency{
			ExtensionID: "plugin.query-owner", VersionConstraint: ">=1.0.0",
		},
		IdentityFields: append([]string(nil), forgedIdentity...),
	}}
	return filter
}

func snapshotFilterIdentity(t *testing.T, snapshot Snapshot, extensionID string) []string {
	t.Helper()
	for _, publication := range snapshot.Publications {
		if publication.Artifact.ExtensionID == extensionID && len(publication.ResultFilters) == 1 {
			return publication.ResultFilters[0].IdentityFields
		}
	}
	t.Fatalf("filter publication %s not found", extensionID)
	return nil
}

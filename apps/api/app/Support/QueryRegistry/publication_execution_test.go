package queryregistry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestBindExecutableRuntimePublishesPrivateProviderAndFilterMaterial(t *testing.T) {
	publication := executablePublicationFixture()
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "ok"}}}, nil
	})
	filter := ResultFilterFunc(func(_ context.Context, request ResultFilterRequest) (ResultFilterResult, error) {
		return ResultFilterResult{Rows: request.Rows}, nil
	})
	bound, err := BindExecutableRuntime(publication,
		[]ExecutableProviderMaterial{{
			QueryID: publication.Queries[0].ID, ContractVersion: publication.Queries[0].ContractVersion,
			PlanVersion: publication.Queries[0].PlanVersion, ResultSchema: publication.Queries[0].ResultSchema,
			Handler: publication.Queries[0].Handler, Provider: provider,
		}},
		[]ExecutableResultFilterMaterial{{
			ID: publication.ResultFilters[0].ID, ContractVersion: publication.ResultFilters[0].ContractVersion,
			QueryID: publication.ResultFilters[0].QueryID, QueryContractVersion: publication.ResultFilters[0].QueryContractVersion,
			QueryPlanVersion: publication.ResultFilters[0].QueryPlanVersion, Handler: publication.ResultFilters[0].Handler,
			Priority: publication.ResultFilters[0].Priority, FailurePolicy: publication.ResultFilters[0].FailurePolicy,
			TimeoutMS: publication.ResultFilters[0].TimeoutMS, Filter: filter,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bound.Queries[0].ProviderDigest == "" || bound.Queries[0].boundProvider == nil ||
		bound.ResultFilters[0].FilterDigest == "" || bound.ResultFilters[0].boundFilter == nil {
		t.Fatalf("bound executable material missing: %#v", bound)
	}
	if !bytes.Equal([]byte(bound.ResultFilters[0].IdentityFields[0]), []byte("id")) {
		t.Fatalf("Host-copied identity fields = %#v", bound.ResultFilters[0].IdentityFields)
	}

	registry := New()
	if _, err := registry.Publish(bound); err != nil {
		t.Fatal(err)
	}
	// 调用方改写 callable 不得污染已发布 revision。
	bound.Queries[0].boundProvider.provider = nil
	bound.ResultFilters[0].boundFilter.filter = nil

	snapshot := registry.Snapshot()
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("boundProvider")) || bytes.Contains(encoded, []byte("boundFilter")) {
		t.Fatalf("snapshot exposed private executable material: %s", encoded)
	}
	if !bytes.Contains(encoded, []byte(snapshot.Publications[0].Queries[0].ProviderDigest)) ||
		!bytes.Contains(encoded, []byte(snapshot.Publications[0].ResultFilters[0].FilterDigest)) {
		t.Fatalf("snapshot omitted public digests: %s", encoded)
	}

	// 内存 Snapshot 恢复保留 private material。
	restored := New()
	if _, err := restored.ReplaceAll(snapshot.Publications, false); err != nil {
		t.Fatalf("restore private executable material: %v", err)
	}
	restoredPublication, ok := restored.SnapshotPublication(publication.Artifact.ExtensionID)
	if !ok || restoredPublication.Queries[0].boundProvider == nil ||
		restoredPublication.Queries[0].boundProvider.provider == nil ||
		restoredPublication.ResultFilters[0].boundFilter == nil ||
		restoredPublication.ResultFilters[0].boundFilter.filter == nil {
		t.Fatalf("restored private material = %#v", restoredPublication)
	}

	// JSON roundtrip 丢失执行权并 fail closed。
	var decoded Snapshot
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Publications[0].Queries[0].boundProvider != nil ||
		decoded.Publications[0].ResultFilters[0].boundFilter != nil {
		t.Fatal("JSON roundtrip rehydrated private callables")
	}
	if _, err := New().ReplaceAll(decoded.Publications, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("JSON-only executable publication accepted: %v", err)
	}
}

func TestBindExecutableRuntimeRejectsMissingDuplicateUnownedAndMismatchedMaterial(t *testing.T) {
	publication := executablePublicationFixture()
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{}, nil
	})
	filter := ResultFilterFunc(func(context.Context, ResultFilterRequest) (ResultFilterResult, error) {
		return ResultFilterResult{}, nil
	})
	validProvider := ExecutableProviderMaterial{
		QueryID: publication.Queries[0].ID, ContractVersion: publication.Queries[0].ContractVersion,
		PlanVersion: publication.Queries[0].PlanVersion, ResultSchema: publication.Queries[0].ResultSchema,
		Handler: publication.Queries[0].Handler, Provider: provider,
	}
	validFilter := ExecutableResultFilterMaterial{
		ID: publication.ResultFilters[0].ID, ContractVersion: publication.ResultFilters[0].ContractVersion,
		QueryID: publication.ResultFilters[0].QueryID, QueryContractVersion: publication.ResultFilters[0].QueryContractVersion,
		QueryPlanVersion: publication.ResultFilters[0].QueryPlanVersion, Handler: publication.ResultFilters[0].Handler,
		Priority: publication.ResultFilters[0].Priority, FailurePolicy: publication.ResultFilters[0].FailurePolicy,
		TimeoutMS: publication.ResultFilters[0].TimeoutMS, Filter: filter,
	}

	tests := []struct {
		name      string
		mutate    func(*Publication, *[]ExecutableProviderMaterial, *[]ExecutableResultFilterMaterial)
		providers []ExecutableProviderMaterial
		filters   []ExecutableResultFilterMaterial
	}{
		{
			name:      "missing provider",
			providers: nil,
			filters:   []ExecutableResultFilterMaterial{validFilter},
		},
		{
			name:      "missing filter",
			providers: []ExecutableProviderMaterial{validProvider},
			filters:   nil,
		},
		{
			name:      "duplicate provider",
			providers: []ExecutableProviderMaterial{validProvider, validProvider},
			filters:   []ExecutableResultFilterMaterial{validFilter},
		},
		{
			name: "unowned provider",
			providers: []ExecutableProviderMaterial{func() ExecutableProviderMaterial {
				value := validProvider
				value.QueryID = "plugin.exec.unknown"
				value.ContractVersion = value.QueryID + "@1"
				value.Handler = "plugin.exec.unknown.execute"
				return value
			}()},
			filters: []ExecutableResultFilterMaterial{validFilter},
		},
		{
			name: "handler mismatch",
			providers: []ExecutableProviderMaterial{func() ExecutableProviderMaterial {
				value := validProvider
				value.Handler = "plugin.exec.other.handler"
				return value
			}()},
			filters: []ExecutableResultFilterMaterial{validFilter},
		},
		{
			name: "plan mismatch",
			providers: []ExecutableProviderMaterial{func() ExecutableProviderMaterial {
				value := validProvider
				value.PlanVersion = "plugin.exec.items.plan@2"
				return value
			}()},
			filters: []ExecutableResultFilterMaterial{validFilter},
		},
		{
			name:      "filter contract mismatch",
			providers: []ExecutableProviderMaterial{validProvider},
			filters: []ExecutableResultFilterMaterial{func() ExecutableResultFilterMaterial {
				value := validFilter
				value.ContractVersion = "plugin.exec.items.redact@2"
				return value
			}()},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := BindExecutableRuntime(publication, test.providers, test.filters); !errors.Is(err, ErrExecutionInvalid) {
				t.Fatalf("BindExecutableRuntime=%v", err)
			}
		})
	}
}

func TestRegistrySafeModeFiltersExecutablePluginMaterialBeforeValidation(t *testing.T) {
	core := publication("core.exec", true, 'a')
	core.Queries = []QueryDeclaration{query("core.exec.items", "core.exec.item", PaginationNone, "public")}
	forged := executablePublicationFixture()
	forged.Queries[0].ProviderDigest = strings.Repeat("f", 64)
	forged.ResultFilters[0].FilterDigest = strings.Repeat("e", 64)

	if _, err := New().ReplaceAll([]Publication{core, forged}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ordinary publication accepted forged executable digests: %v", err)
	}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{core, forged}, true); err != nil {
		t.Fatalf("Safe Mode parsed third-party executable material: %v", err)
	}
	if snapshot := registry.Snapshot(); !snapshot.SafeMode || len(snapshot.Publications) != 1 ||
		snapshot.Publications[0].Artifact.ExtensionID != "core.exec" {
		t.Fatalf("Safe Mode snapshot = %#v", snapshot)
	}
}

func TestPlanExecutableDefaultSortIdentityFieldsAndRelationFence(t *testing.T) {
	publication := executablePublicationFixture()
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{}, nil
	})
	filter := ResultFilterFunc(func(context.Context, ResultFilterRequest) (ResultFilterResult, error) {
		return ResultFilterResult{}, nil
	})
	bound, err := BindExecutableRuntime(publication,
		[]ExecutableProviderMaterial{{
			QueryID: publication.Queries[0].ID, ContractVersion: publication.Queries[0].ContractVersion,
			PlanVersion: publication.Queries[0].PlanVersion, ResultSchema: publication.Queries[0].ResultSchema,
			Handler: publication.Queries[0].Handler, Provider: provider,
		}},
		[]ExecutableResultFilterMaterial{{
			ID: publication.ResultFilters[0].ID, ContractVersion: publication.ResultFilters[0].ContractVersion,
			QueryID: publication.ResultFilters[0].QueryID, QueryContractVersion: publication.ResultFilters[0].QueryContractVersion,
			QueryPlanVersion: publication.ResultFilters[0].QueryPlanVersion, Handler: publication.ResultFilters[0].Handler,
			Priority: publication.ResultFilters[0].Priority, FailurePolicy: publication.ResultFilters[0].FailurePolicy,
			TimeoutMS: publication.ResultFilters[0].TimeoutMS, Filter: filter,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	registry := newPlanningRegistry().WithPluginAdmission(func(Artifact) bool { return true })
	if _, err := registry.Publish(bound); err != nil {
		t.Fatal(err)
	}

	// 空 sort 使用 DefaultSort。
	plan, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: bound.Queries[0].ID, Fields: []string{"title"},
		Pagination: PaginationRequest{Limit: 10},
	})
	if err != nil {
		t.Fatalf("plan executable: %v", err)
	}
	if len(plan.Sorts) != 2 || plan.Sorts[0].Field != "created_at" || !plan.Sorts[0].Descending ||
		plan.Sorts[1].Field != "id" || !containsString(plan.Fields, "id") {
		t.Fatalf("default sort / identity fields = fields %#v sorts %#v", plan.Fields, plan.Sorts)
	}

	// 调用方 sort 后追加缺失 identity tie-breaker。
	plan, err = registry.Plan(context.Background(), PlanRequest{
		QueryID: bound.Queries[0].ID, Fields: []string{"title", "id"},
		Sorts:      []SortValue{{Field: "created_at", Descending: true}},
		Pagination: PaginationRequest{Limit: 10},
	})
	if err != nil {
		t.Fatalf("plan with caller sort: %v", err)
	}
	if len(plan.Sorts) != 2 || plan.Sorts[0].Field != "created_at" || plan.Sorts[1].Field != "id" {
		t.Fatalf("identity tie-breaker missing: %#v", plan.Sorts)
	}

	// 第三方 executable relations fail closed。
	if _, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: bound.Queries[0].ID, Fields: []string{"id", "title"}, Relations: []string{"owner"},
		Pagination: PaginationRequest{Limit: 10},
	}); !errors.Is(err, ErrContractInsufficient) {
		t.Fatalf("third-party executable relations = %v", err)
	}
}

func TestPlanHandlerlessLegacyRemainsInspectableWithoutProvider(t *testing.T) {
	publication := publication("plugin.legacy", false, 'a')
	declaration := query("plugin.legacy.items", "plugin.legacy.item", PaginationOffset, "public")
	declaration.Relations = nil
	publication.Queries = []QueryDeclaration{declaration}
	registry := newPlanningRegistry().WithPluginAdmission(func(Artifact) bool { return true })
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	plan, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: declaration.ID, Fields: []string{"id", "title"},
		Pagination: PaginationRequest{Limit: 5},
	})
	if err != nil || plan.Query.Handler != "" || plan.Query.ProviderDigest != "" {
		t.Fatalf("handlerless plan = %#v err=%v", plan, err)
	}
	resolved, err := registry.Resolve(declaration.ID)
	if err != nil || resolved.Handler != "" || resolved.boundProvider != nil {
		t.Fatalf("handlerless resolve = %#v err=%v", resolved, err)
	}
}

func executablePublicationFixture() Publication {
	publication := publication("plugin.exec", false, 'a')
	declaration := query("plugin.exec.items", "plugin.exec.item", PaginationOffset, "public")
	declaration.Relations = []string{"owner"}
	declaration.Sort = []string{"created_at", "id"}
	declaration.Handler = "plugin.exec.items.execute"
	declaration.IdentityFields = []string{"id"}
	declaration.DefaultSort = []SortValue{
		{Field: "created_at", Descending: true},
		{Field: "id"},
	}
	publication.Queries = []QueryDeclaration{declaration}
	publication.ResultFilters = []ResultFilterDeclaration{{
		ID: "plugin.exec.items.redact", ContractVersion: "plugin.exec.items.redact@1",
		QueryID: declaration.ID, QueryContractVersion: declaration.ContractVersion,
		QueryPlanVersion: declaration.PlanVersion, Handler: "plugin.exec.items.redact",
		Priority: 10, FailurePolicy: ResultFilterFailClosed, TimeoutMS: 1000,
	}}
	return publication
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

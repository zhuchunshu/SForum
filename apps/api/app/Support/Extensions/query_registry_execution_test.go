package extensionsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	queryregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/QueryRegistry"
)

func TestCompositeQueryProviderResolverPrefersCoreThenProtocolV2(t *testing.T) {
	coreArtifact, err := queryregistry.NewCoreArtifact("core.query", "1.0.0", strings.Repeat("c", 64))
	if err != nil {
		t.Fatal(err)
	}
	coreProvider := queryregistry.ExecutableProviderFunc(func(
		context.Context, queryregistry.ProviderExecutionRequest,
	) (queryregistry.ProviderExecutionResult, error) {
		return queryregistry.ProviderExecutionResult{Rows: []queryregistry.QueryRow{{"id": "core"}}}, nil
	})
	coreResolver, err := queryregistry.NewStaticProviderResolver([]queryregistry.ExecutableProviderBinding{{
		QueryID: "core.query.items", ContractVersion: "core.query.items@1",
		PlanVersion: "core.query.items.plan@1", ResultSchema: "core.query.items.result@1",
		Artifact: coreArtifact, Provider: coreProvider,
	}})
	if err != nil {
		t.Fatal(err)
	}

	starter := &queryExecutionStarter{}
	manager := NewManager(ManagerConfig{Starter: starter})
	extension := managerRuntimeExtension("plugin.query.demo", "1.0.0", strings.Repeat("a", 64))
	extension.ActiveVersionID = 21
	declaration := extensions.ManifestQuery{
		ID: "plugin.query.demo.items", ContractVersion: "plugin.query.demo.items@1",
		Entity: "item", PlanVersion: "plugin.query.demo.items.plan@1",
		Fields: []string{"id", "title"}, Sort: []string{"id"}, Pagination: "offset",
		ResultSchema: "plugin.query.demo.items.result@1", PermissionPolicy: "public",
		Handler: "plugin.query.demo.items", IdentityFields: []string{"id"},
		DefaultSort: []extensions.ManifestQuerySort{{Field: "id", Descending: true}},
	}
	filterDecl := extensions.ManifestQueryResultFilter{
		ID: "plugin.query.demo.items.mask", ContractVersion: "plugin.query.demo.items.mask@1",
		QueryID: declaration.ID, QueryContractVersion: declaration.ContractVersion,
		QueryPlanVersion: declaration.PlanVersion, Handler: "plugin.query.demo.items.mask",
		FailurePolicy: "fail_closed", TimeoutMS: 500,
	}
	extension.Manifest.Queries = []extensions.ManifestQuery{declaration}
	extension.Manifest.QueryResultFilters = []extensions.ManifestQueryResultFilter{filterDecl}
	if err := manager.Start(context.Background(), extension); err != nil {
		t.Fatal(err)
	}
	active, err := manager.ActiveRuntimeInstance(extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	publication := queryregistry.Publication{
		Artifact: queryregistry.Artifact{
			ExtensionID: extension.ID, ExtensionVersion: extension.Version,
			PackageDigest: extension.PackageDigest, VersionID: extension.ActiveVersionID,
			RuntimeInstanceID: active.Identity.InstanceID,
		},
		Queries: []queryregistry.QueryDeclaration{{
			ID: declaration.ID, ContractVersion: declaration.ContractVersion, Entity: declaration.Entity,
			PlanVersion: declaration.PlanVersion, Fields: declaration.Fields, Sort: declaration.Sort,
			Pagination: declaration.Pagination, ResultSchema: declaration.ResultSchema,
			PermissionPolicy: declaration.PermissionPolicy, Handler: declaration.Handler,
			IdentityFields: declaration.IdentityFields,
			DefaultSort:    []queryregistry.SortValue{{Field: "id", Descending: true}},
		}},
		ResultFilters: []queryregistry.ResultFilterDeclaration{{
			ID: filterDecl.ID, ContractVersion: filterDecl.ContractVersion,
			QueryID: filterDecl.QueryID, QueryContractVersion: filterDecl.QueryContractVersion,
			QueryPlanVersion: filterDecl.QueryPlanVersion, Handler: filterDecl.Handler,
			FailurePolicy: filterDecl.FailurePolicy, TimeoutMS: filterDecl.TimeoutMS,
		}},
	}
	registry := queryregistry.New(queryregistry.WithCostPolicy(queryregistry.CostPolicyFunc(
		func(input queryregistry.QueryCostInput) (queryregistry.QueryCost, error) {
			return queryregistry.QueryCost{Units: 10 + len(input.Fields), Maximum: 2_000}, nil
		},
	)))
	// 插件 artifact 需 Host 运行时 admission 才可 plan/execute。
	var pluginAdmitted atomic.Bool
	pluginAdmitted.Store(true)
	registry.WithPluginAdmission(func(artifact queryregistry.Artifact) bool {
		return pluginAdmitted.Load() && artifact == publication.Artifact
	})
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	providers, err := NewCompositeQueryProviderResolver(coreResolver, manager, registry)
	if err != nil {
		t.Fatal(err)
	}
	filterSource, err := NewProtocolV2QueryResultFilterSource(manager, registry)
	if err != nil {
		t.Fatal(err)
	}
	schemas := queryregistry.ResultSchemaValidatorFunc(func(
		context.Context, queryregistry.ResultSchemaClaim, queryregistry.QueryRow,
	) error {
		return nil
	})
	runtime, err := queryregistry.NewExecutionRuntime(queryregistry.ExecutionConfig{
		Registry: registry, Providers: providers, Schemas: schemas, ResultFilterSource: filterSource,
		// 测试用 admission：仅证明 resolver/filter 接线，不模拟 Manager 租约。
		Admission: queryregistry.ExecutionAdmissionFunc(func(context.Context, queryregistry.Artifact) (func(), error) {
			return func() {}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Core 查询仍走静态 Host 绑定，不触碰 Protocol V2 starter。
	corePublication := queryregistry.Publication{
		Artifact: coreArtifact,
		Queries: []queryregistry.QueryDeclaration{{
			ID: "core.query.items", ContractVersion: "core.query.items@1", Entity: "item",
			PlanVersion: "core.query.items.plan@1", Fields: []string{"id"}, Pagination: "none",
			ResultSchema: "core.query.items.result@1", PermissionPolicy: "public",
		}},
	}
	if _, err := registry.Publish(corePublication); err != nil {
		t.Fatal(err)
	}
	coreResult, err := runtime.Execute(context.Background(), queryregistry.PlanRequest{
		QueryID: "core.query.items", Fields: []string{"id"},
	})
	if err != nil || len(coreResult.Rows) != 1 || coreResult.Rows[0]["id"] != "core" || starter.calls.Load() != 0 {
		t.Fatalf("core result=%#v calls=%d err=%v", coreResult, starter.calls.Load(), err)
	}

	pluginResult, err := runtime.Execute(context.Background(), queryregistry.PlanRequest{
		QueryID: declaration.ID, Fields: []string{"id", "title"},
		Pagination: queryregistry.PaginationRequest{Limit: 10},
	})
	// InvokeQuery + FilterQueryResult；filter 仅改写已声明 title 字段。
	if err != nil || len(pluginResult.Rows) != 1 || pluginResult.Rows[0]["title"] != "plugin-filtered" ||
		starter.calls.Load() != 2 {
		t.Fatalf("plugin result=%#v calls=%d err=%v", pluginResult, starter.calls.Load(), err)
	}

	if err := manager.Stop(context.Background(), extension); err != nil {
		t.Fatal(err)
	}
	pluginAdmitted.Store(false)
	if _, err := runtime.Execute(context.Background(), queryregistry.PlanRequest{
		QueryID: declaration.ID, Fields: []string{"id", "title"},
		Pagination: queryregistry.PaginationRequest{Limit: 10},
	}); !errors.Is(err, queryregistry.ErrArtifactUnavailable) &&
		!errors.Is(err, queryregistry.ErrProviderUnavailable) {
		t.Fatalf("disabled plugin error=%v", err)
	}
}

type queryExecutionStarter struct {
	calls atomic.Int32
}

func (*queryExecutionStarter) Start(context.Context, extensions.Extension) (RouteTarget, error) {
	return RouteTarget{InstanceID: "query-reference-runtime"}, nil
}

func (*queryExecutionStarter) Stop(context.Context, extensions.Extension) error { return nil }

func (s *queryExecutionStarter) InvokeQuery(
	_ context.Context,
	_ extensions.Extension,
	request VersionedQueryRequest,
) ([]queryregistry.QueryRow, error) {
	s.calls.Add(1)
	if request.Handler == "" || request.FetchLimit < 1 {
		return nil, errors.New("invalid query invocation")
	}
	return []queryregistry.QueryRow{{
		"id": json.Number("1"), "title": "plugin",
	}}, nil
}

func (s *queryExecutionStarter) FilterQueryResult(
	_ context.Context,
	_ extensions.Extension,
	request VersionedQueryResultFilterRequest,
) ([]queryregistry.QueryRow, error) {
	s.calls.Add(1)
	if request.Handler == "" || len(request.Rows) == 0 {
		return nil, errors.New("invalid filter invocation")
	}
	result := make([]queryregistry.QueryRow, 0, len(request.Rows))
	for _, row := range request.Rows {
		cloned := make(queryregistry.QueryRow, len(row))
		for key, value := range row {
			cloned[key] = value
		}
		// 仅改写已选字段，不得新增 undeclared field。
		cloned["title"] = "plugin-filtered"
		result = append(result, cloned)
	}
	return result, nil
}

func TestProtocolV2QueryResultFilterSourceEmptyWithoutDeclarations(t *testing.T) {
	manager := NewManager(ManagerConfig{})
	registry := queryregistry.New()
	source, err := NewProtocolV2QueryResultFilterSource(manager, registry)
	if err != nil {
		t.Fatal(err)
	}
	filters, err := source.ResultFiltersFor(queryregistry.QueryContribution{
		QueryDeclaration: queryregistry.QueryDeclaration{ID: "missing.query"},
	})
	if err != nil || len(filters) != 0 {
		t.Fatalf("filters=%#v err=%v", filters, err)
	}
	if _, err := NewProtocolV2QueryResultFilterSource(nil, registry); !errors.Is(err, queryregistry.ErrExecutionInvalid) {
		t.Fatalf("nil manager err=%v", err)
	}
}

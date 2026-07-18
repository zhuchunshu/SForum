package extensionsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	queryregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/QueryRegistry"
	pluginwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestProtocolV2QueryRuntimeInvokeAndFilterExactBinding(t *testing.T) {
	declaration := extensions.ManifestQuery{
		ID: "plugin.query.demo.items", ContractVersion: "plugin.query.demo.items@1",
		Entity: "item", PlanVersion: "plugin.query.demo.items.plan@1",
		Fields: []string{"id", "title"}, Pagination: "offset",
		ResultSchema: "plugin.query.demo.items.result@1", PermissionPolicy: "public",
		Handler: "plugin.query.demo.items", IdentityFields: []string{"id"},
		DefaultSort: []extensions.ManifestQuerySort{{Field: "id", Descending: true}},
	}
	filter := extensions.ManifestQueryResultFilter{
		ID: "plugin.query.demo.items.mask", ContractVersion: "plugin.query.demo.items.mask@1",
		QueryID: declaration.ID, QueryContractVersion: declaration.ContractVersion,
		QueryPlanVersion: declaration.PlanVersion, Handler: "plugin.query.demo.items.mask",
		FailurePolicy: "fail_closed", TimeoutMS: 500,
	}
	var invoked *pluginwire.QueryInvocationRequest
	var filtered *pluginwire.QueryResultFilterRequest
	client := newProtocolV2QueryTestClient(t, declaration, filter, func(
		_ context.Context, request *pluginwire.QueryInvocationRequest,
	) (*pluginwire.QueryInvocationResponse, error) {
		invoked = request
		return &pluginwire.QueryInvocationResponse{
			Binding: request.GetBinding(), ShapeDigest: request.GetPlan().GetShapeDigest(),
			Outcome: &pluginwire.QueryInvocationResponse_Success{
				Success: &pluginwire.QueryRuntimeRows{Rows: []*pluginwire.QueryRuntimeRow{
					{CanonicalJson: []byte(`{"id":"1","title":"exact"}`)},
				}},
			},
		}, nil
	}, func(
		_ context.Context, request *pluginwire.QueryResultFilterRequest,
	) (*pluginwire.QueryResultFilterResponse, error) {
		filtered = request
		return &pluginwire.QueryResultFilterResponse{
			Binding: request.GetBinding(), ShapeDigest: request.GetPlan().GetShapeDigest(),
			Outcome: &pluginwire.QueryResultFilterResponse_Success{Success: request.GetInput()},
		}, nil
	})
	plan := protocolV2QueryTestPlan(declaration)
	rows, err := client.InvokeQuery(context.Background(), VersionedQueryRequest{
		QueryID: declaration.ID, ContractVersion: declaration.ContractVersion,
		PlanVersion: declaration.PlanVersion, ResultSchema: declaration.ResultSchema,
		Handler: declaration.Handler, Plan: plan, FetchLimit: 11, Timeout: 500 * time.Millisecond,
	})
	if err != nil || len(rows) != 1 || rows[0]["id"] != "1" || rows[0]["title"] != "exact" {
		t.Fatalf("InvokeQuery rows=%#v err=%v", rows, err)
	}
	if invoked.GetContext().GetGrantedAuthority() != nil ||
		invoked.GetBinding().GetHandler() != declaration.Handler ||
		invoked.GetPlan().GetShapeDigest() != plan.ShapeDigest {
		t.Fatalf("InvokeQuery request drifted: %#v", invoked)
	}
	// 整数词素必须无损往返。
	plan.ShapeDigest = strings.Repeat("b", 64)
	filteredRows, err := client.FilterQueryResult(context.Background(), VersionedQueryResultFilterRequest{
		FilterID: filter.ID, FilterContractVersion: filter.ContractVersion,
		QueryID: declaration.ID, QueryContractVersion: declaration.ContractVersion,
		QueryPlanVersion: declaration.PlanVersion, ResultSchema: declaration.ResultSchema,
		Handler: filter.Handler, Plan: plan,
		Rows:    []queryregistry.QueryRow{{"id": json.Number("9007199254740993"), "title": "keep"}},
		Timeout: 500 * time.Millisecond,
	})
	if err != nil || len(filteredRows) != 1 || filteredRows[0]["id"] != json.Number("9007199254740993") {
		t.Fatalf("FilterQueryResult rows=%#v err=%v", filteredRows, err)
	}
	if filtered.GetBinding().GetFilterId() != filter.ID ||
		filtered.GetContext().GetGrantedAuthority() != nil {
		t.Fatalf("FilterQueryResult request drifted: %#v", filtered)
	}
	if _, err := client.InvokeQuery(context.Background(), VersionedQueryRequest{
		QueryID: declaration.ID, ContractVersion: declaration.ContractVersion,
		PlanVersion: declaration.PlanVersion, ResultSchema: declaration.ResultSchema,
		Handler: "plugin.query.demo.drift", Plan: plan, FetchLimit: 1,
	}); err == nil {
		t.Fatal("drifted handler reached transport")
	}
}

func TestProtocolV2QueryRuntimeMapsDeadlineAndCancellation(t *testing.T) {
	declaration := extensions.ManifestQuery{
		ID: "plugin.query.demo.items", ContractVersion: "plugin.query.demo.items@1",
		Entity: "item", PlanVersion: "plugin.query.demo.items.plan@1",
		Fields: []string{"id"}, Pagination: "none", ResultSchema: "plugin.query.demo.items.result@1",
		PermissionPolicy: "public", Handler: "plugin.query.demo.items",
		IdentityFields: []string{"id"},
		DefaultSort:    []extensions.ManifestQuerySort{{Field: "id"}},
	}
	for _, test := range []struct {
		name string
		code codes.Code
		want error
	}{
		{name: "deadline", code: codes.DeadlineExceeded, want: context.DeadlineExceeded},
		{name: "canceled", code: codes.Canceled, want: context.Canceled},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := newProtocolV2QueryTestClient(t, declaration, extensions.ManifestQueryResultFilter{},
				func(context.Context, *pluginwire.QueryInvocationRequest) (*pluginwire.QueryInvocationResponse, error) {
					return nil, status.Error(test.code, "provider stopped")
				}, nil)
			_, err := client.InvokeQuery(context.Background(), VersionedQueryRequest{
				QueryID: declaration.ID, ContractVersion: declaration.ContractVersion,
				PlanVersion: declaration.PlanVersion, ResultSchema: declaration.ResultSchema,
				Handler: declaration.Handler, Plan: protocolV2QueryTestPlan(declaration), FetchLimit: 1,
			})
			if !errors.Is(err, test.want) {
				t.Fatalf("error=%v want %v", err, test.want)
			}
		})
	}
}

func protocolV2QueryTestPlan(declaration extensions.ManifestQuery) queryregistry.QueryPlan {
	return queryregistry.QueryPlan{
		ShapeDigest: strings.Repeat("a", 64),
		Fields:      append([]string(nil), declaration.Fields...),
		Locale:      "zh-CN",
		Pagination:  queryregistry.PaginationPlan{Mode: declaration.Pagination, Limit: 10},
		Query: queryregistry.QueryContribution{
			QueryDeclaration: queryregistry.QueryDeclaration{
				ID: declaration.ID, ContractVersion: declaration.ContractVersion,
				PlanVersion: declaration.PlanVersion, ResultSchema: declaration.ResultSchema,
				Handler: declaration.Handler, Fields: append([]string(nil), declaration.Fields...),
			},
		},
	}
}

type protocolV2QueryTestServer struct {
	pluginwire.UnimplementedPluginRuntimeServiceServer
	invoke func(context.Context, *pluginwire.QueryInvocationRequest) (*pluginwire.QueryInvocationResponse, error)
	filter func(context.Context, *pluginwire.QueryResultFilterRequest) (*pluginwire.QueryResultFilterResponse, error)
}

func (s *protocolV2QueryTestServer) InvokeQuery(
	ctx context.Context, request *pluginwire.QueryInvocationRequest,
) (*pluginwire.QueryInvocationResponse, error) {
	if s.invoke == nil {
		return nil, status.Error(codes.Unimplemented, "query")
	}
	return s.invoke(ctx, request)
}

func (s *protocolV2QueryTestServer) FilterQueryResult(
	ctx context.Context, request *pluginwire.QueryResultFilterRequest,
) (*pluginwire.QueryResultFilterResponse, error) {
	if s.filter == nil {
		return nil, status.Error(codes.Unimplemented, "filter")
	}
	return s.filter(ctx, request)
}

func newProtocolV2QueryTestClient(
	t *testing.T,
	query extensions.ManifestQuery,
	filter extensions.ManifestQueryResultFilter,
	invoke func(context.Context, *pluginwire.QueryInvocationRequest) (*pluginwire.QueryInvocationResponse, error),
	filterFn func(context.Context, *pluginwire.QueryResultFilterRequest) (*pluginwire.QueryResultFilterResponse, error),
) *protocolV2Client {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	pluginwire.RegisterPluginRuntimeServiceServer(server, &protocolV2QueryTestServer{invoke: invoke, filter: filterFn})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close() })
	connection, err := grpc.NewClient("passthrough:///query-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	identity := &protocolwire.ExtensionIdentity{
		ExtensionId: "plugin.query.demo", ExtensionVersion: "1.0.0", ArtifactDigest: "digest-v1",
		TrustGrantId: "grant-1", RuntimeEpoch: 1, InstanceId: "query-runtime",
	}
	filters := []extensions.ManifestQueryResultFilter{}
	if filter.ID != "" {
		filters = append(filters, filter)
	}
	return newProtocolV2Client(pluginwire.NewPluginRuntimeServiceClient(connection), protocolV2ClientConfig{
		identity: identity, instance: identity.InstanceId,
		queries: []extensions.ManifestQuery{query}, queryResultFilters: filters,
	})
}

package extensionsruntime

import (
	"context"
	"errors"
	"net"
	"net/http"
	"reflect"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	pluginwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProtocolV2RouteCarriesExactTypedRequestAndHostActor(t *testing.T) {
	var received *pluginwire.RouteRequest
	client := newProtocolV2RouteTestClient(t, "runtime-1", func(_ context.Context, request *pluginwire.RouteRequest) (*pluginwire.RouteResponse, error) {
		received = request
		return protocolV2RouteTestResponse(request, map[string]any{"ok": true}), nil
	})
	response, err := client.InvokeRouteContext(context.Background(), ProtocolV2RouteRequest{
		RouteID: "demo.route", ContractVersion: "demo.route@1", Method: http.MethodPost, Path: "/demo/41",
		Authority: protocolV2FilteredHostRequestAuthority(),
		Headers:   http.Header{"X-Test": {"one", "two"}}, PathParameters: map[string]string{"id": "41"},
		QueryParameters: map[string]string{"page": "2"}, RequestSchema: "demo.request@1", ResponseSchema: "demo.response@1",
		Body: map[string]any{"title": "hello"}, BodyPresent: true,
		Actor:         NewProtocolV2RouteActor(42, true, map[string]bool{"topics.write": true, "ignored": false, "*": true}),
		CorrelationID: "trace-route", Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if received.GetRouteId() != "demo.route" || received.GetContractVersion() != "demo.route@1" ||
		received.GetMethod() != http.MethodPost || received.GetPath() != "/demo/41" ||
		received.GetRequestAuthorityMode() != pluginwire.RouteRequestAuthorityMode_ROUTE_REQUEST_AUTHORITY_MODE_FILTERED ||
		received.GetGuardKind() != pluginwire.RouteGuardKind_ROUTE_GUARD_KIND_HOST ||
		received.GetPathParameters()["id"] != "41" || received.GetQueryParameters()["page"] != "2" ||
		received.GetBody().GetSchemaId() != "demo.request" || received.GetBody().GetSchemaVersion() != "1" ||
		received.GetBody().GetValue().AsMap()["title"] != "hello" {
		t.Fatalf("request = %#v", received)
	}
	if actor := received.GetContext().GetActor(); actor.GetUserId() != 42 ||
		!reflect.DeepEqual(actor.GetPermissionKeys(), []string{"*", "topics.write"}) {
		t.Fatalf("actor = %#v", actor)
	}
	if response.StatusCode != http.StatusCreated || !response.BodyPresent || response.Body["ok"] != true ||
		!reflect.DeepEqual(response.Headers.Values("X-Result"), []string{"one", "two"}) {
		t.Fatalf("response = %#v", response)
	}
	if client.requestContext(context.Background(), "ordinary").GetActor() != nil ||
		NewProtocolV2RouteActor(42, false, map[string]bool{"*": true}) != nil {
		t.Fatal("ordinary or anonymous context acquired actor authority")
	}
}

func TestProtocolV2RouteCarriesCatalogIssuedActorDelegations(t *testing.T) {
	var received *pluginwire.RouteRequest
	client := newProtocolV2RouteTestClient(t, "runtime-1", func(_ context.Context, request *pluginwire.RouteRequest) (*pluginwire.RouteResponse, error) {
		received = request
		return protocolV2RouteTestResponse(request, map[string]any{"ok": true}), nil
	})
	issuer := &recordingProtocolV2ActorDelegationIssuer{grants: []hostapi.ProtocolV2ActorDelegationGrant{
		{CommandID: "sforum.user.status", CommandVersion: "1", IdempotencyKey: "route-request-42", Token: "signed-token"},
	}}
	client.delegations = issuer
	client.hostCommands = true
	request := protocolV2RouteTestRequest()
	request.Actor = NewProtocolV2RouteActor(42, true, map[string]bool{"user.manage": true})
	request.IdempotencyKey = "route-request-42"
	if _, err := client.InvokeRouteContext(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if issuer.calls != 1 || issuer.request.ActorUserID != 42 || issuer.request.IdempotencyKey != request.IdempotencyKey ||
		issuer.request.Runtime.GetInstanceId() != client.identity.GetInstanceId() ||
		!reflect.DeepEqual(issuer.request.PermissionKeys, []string{"user.manage"}) {
		t.Fatalf("issuer request = %#v", issuer.request)
	}
	delegations := received.GetContext().GetHostCommandDelegations()
	if received.GetContext().GetIdempotencyKey() != request.IdempotencyKey || len(delegations) != 1 ||
		delegations[0].GetCommandId() != "sforum.user.status" || delegations[0].GetToken() != "signed-token" {
		t.Fatalf("route context = %#v", received.GetContext())
	}

	issuer.calls = 0
	request.Actor = nil
	if _, err := client.InvokeRouteContext(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if issuer.calls != 0 {
		t.Fatal("anonymous invocation acquired an actor delegation")
	}
	request.Actor = NewProtocolV2RouteActor(42, true, map[string]bool{"user.manage": true})
	client.hostCommands = false
	if _, err := client.InvokeRouteContext(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if issuer.calls != 0 || len(received.GetContext().GetHostCommandDelegations()) != 0 {
		t.Fatal("artifact without host_commands acquired an actor delegation")
	}
	client.hostCommands = true
	background := client.requestContext(context.Background(), "background-job")
	if issuer.calls != 0 || background.GetActor() != nil || background.GetIdempotencyKey() != "" ||
		len(background.GetHostCommandDelegations()) != 0 {
		t.Fatalf("background context acquired actor authority: %#v", background)
	}
}

func TestProtocolV2RouteRejectsUnavailableOrMalformedActorDelegations(t *testing.T) {
	request := protocolV2RouteTestRequest()
	request.Actor = NewProtocolV2RouteActor(42, true, map[string]bool{"user.manage": true})
	request.IdempotencyKey = "route-request-42"
	for _, test := range []struct {
		name   string
		issuer hostapi.ProtocolV2ActorDelegationBundleIssuer
		want   error
	}{
		{name: "issuer unavailable", want: ErrProtocolV2ActorDelegationUnavailable},
		{name: "issuer failure", issuer: &recordingProtocolV2ActorDelegationIssuer{
			err: errors.New("signing key unavailable"),
		}, want: ErrProtocolV2ActorDelegationUnavailable},
		{name: "wrong key", issuer: &recordingProtocolV2ActorDelegationIssuer{grants: []hostapi.ProtocolV2ActorDelegationGrant{{
			CommandID: "sforum.user.status", CommandVersion: "1", IdempotencyKey: "other", Token: "signed-token",
		}}}, want: ErrProtocolV2ActorDelegationInvalid},
		{name: "duplicate command", issuer: &recordingProtocolV2ActorDelegationIssuer{grants: []hostapi.ProtocolV2ActorDelegationGrant{
			{CommandID: "sforum.user.status", CommandVersion: "1", IdempotencyKey: "route-request-42", Token: "signed-one"},
			{CommandID: "sforum.user.status", CommandVersion: "1", IdempotencyKey: "route-request-42", Token: "signed-two"},
		}}, want: ErrProtocolV2ActorDelegationInvalid},
	} {
		t.Run(test.name, func(t *testing.T) {
			called := false
			client := newProtocolV2RouteTestClient(t, "runtime-1", func(_ context.Context, request *pluginwire.RouteRequest) (*pluginwire.RouteResponse, error) {
				called = true
				return protocolV2RouteTestResponse(request, map[string]any{"ok": true}), nil
			})
			client.delegations = test.issuer
			client.hostCommands = true
			if _, err := client.InvokeRouteContext(context.Background(), request); !errors.Is(err, test.want) {
				t.Fatalf("error = %v", err)
			}
			if called {
				t.Fatal("malformed delegation reached the plugin")
			}
		})
	}
}

func TestProtocolV2RouteFailsClosedOnResponseDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*pluginwire.RouteResponse)
		want   error
	}{
		{"nil context", func(response *pluginwire.RouteResponse) { response.Context = nil }, ErrProtocolV2RouteInvalid},
		{"wrong request", func(response *pluginwire.RouteResponse) { response.Context.RequestId = "wrong" }, ErrProtocolV2RouteInvalid},
		{"wrong identity", func(response *pluginwire.RouteResponse) { response.Context.Extension.InstanceId = "replacement" }, ErrProtocolV2RouteInvalid},
		{"invalid time", func(response *pluginwire.RouteResponse) { response.Context.ServerTime = nil }, ErrProtocolV2RouteInvalid},
		{"wrong schema", func(response *pluginwire.RouteResponse) { response.Body.SchemaVersion = "2" }, ErrProtocolV2RouteInvalid},
		{"stream with body", func(response *pluginwire.RouteResponse) { response.StreamFollows = true }, ErrProtocolV2RouteInvalid},
		{"invalid header name", func(response *pluginwire.RouteResponse) { response.Headers[0].Name = "Bad\r\nName" }, ErrProtocolV2RouteInvalid},
		{"invalid header value", func(response *pluginwire.RouteResponse) { response.Headers[0].Values[0] = "bad\r\nvalue" }, ErrProtocolV2RouteInvalid},
		{"typed error", func(response *pluginwire.RouteResponse) {
			response.Error = &protocolwire.ErrorDetail{Code: protocolwire.ErrorCode_ERROR_CODE_PERMISSION_DENIED, Reason: "route.denied"}
		}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newProtocolV2RouteTestClient(t, "runtime-1", func(_ context.Context, request *pluginwire.RouteRequest) (*pluginwire.RouteResponse, error) {
				response := protocolV2RouteTestResponse(request, map[string]any{"ok": true})
				test.mutate(response)
				return response, nil
			})
			_, err := client.InvokeRouteContext(context.Background(), protocolV2RouteTestRequest())
			if test.name == "typed error" {
				var typed *ProtocolV2Error
				if !errors.As(err, &typed) || typed.Reason != "route.denied" {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestProtocolV2RouteAcceptsAuthenticatedStreamPreflight(t *testing.T) {
	client := newProtocolV2RouteTestClient(t, "runtime-1", func(_ context.Context, request *pluginwire.RouteRequest) (*pluginwire.RouteResponse, error) {
		response := protocolV2RouteTestResponse(request, nil)
		response.Body = nil
		response.StreamFollows = true
		return response, nil
	})
	response, err := client.InvokeRouteContext(context.Background(), protocolV2RouteTestRequest())
	if err != nil {
		t.Fatal(err)
	}
	if !response.StreamFollows || response.BodyPresent || response.StatusCode != http.StatusCreated || response.Headers.Get("X-Result") != "one" {
		t.Fatalf("stream preflight=%#v", response)
	}
}

func TestProtocolV2RoutePropagatesCallerCancellation(t *testing.T) {
	client := newProtocolV2RouteTestClient(t, "runtime-1", func(ctx context.Context, _ *pluginwire.RouteRequest) (*pluginwire.RouteResponse, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.InvokeRouteContext(ctx, protocolV2RouteTestRequest()); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolV2RouteCreatesUniqueTraceWithoutCallerCorrelation(t *testing.T) {
	traces := make(chan string, 2)
	client := newProtocolV2RouteTestClient(t, "runtime-1", func(_ context.Context, request *pluginwire.RouteRequest) (*pluginwire.RouteResponse, error) {
		traces <- request.GetContext().GetTrace().GetTraceId()
		return protocolV2RouteTestResponse(request, map[string]any{"ok": true}), nil
	})
	request := protocolV2RouteTestRequest()
	for range 2 {
		if _, err := client.InvokeRouteContext(context.Background(), request); err != nil {
			t.Fatal(err)
		}
	}
	first, second := <-traces, <-traces
	if first == "" || second == "" || first == second {
		t.Fatalf("trace ids = %q, %q", first, second)
	}
}

func TestProtocolV2FrozenRouteMatchesHeadAndGlobalMiddleware(t *testing.T) {
	client := &protocolV2Client{routes: []extensions.ManifestRoute{
		{ID: "demo.get", ContractVersion: "demo.get@1", Methods: []string{http.MethodGet}, Mode: extensionmanifest.RouteModeHTTP, Guard: extensionmanifest.GuardCorePublic},
		{ID: "demo.sse", ContractVersion: "demo.sse@1", Methods: []string{http.MethodGet}, Mode: extensionmanifest.RouteModeSSE, Guard: extensionmanifest.GuardCorePublic},
		{ID: "demo.global", ContractVersion: "demo.global@1", Action: extensionmanifest.RouteActionGlobalMiddleware, Guard: extensionmanifest.GuardCorePublic},
	}}
	for _, request := range []ProtocolV2RouteRequest{
		{RouteID: "demo.get", ContractVersion: "demo.get@1", Method: http.MethodHead, Authority: protocolV2FilteredHostRequestAuthority()},
		{RouteID: "demo.global", ContractVersion: "demo.global@1", Method: http.MethodDelete, Authority: protocolV2FilteredHostRequestAuthority()},
	} {
		if err := client.validateFrozenRoute(request); err != nil {
			t.Fatalf("request %#v: %v", request, err)
		}
	}
	if err := client.validateFrozenRoute(ProtocolV2RouteRequest{
		RouteID: "demo.get", ContractVersion: "demo.get@1", Method: http.MethodPost,
		Authority: protocolV2FilteredHostRequestAuthority(),
	}); !errors.Is(err, ErrProtocolV2RouteInvalid) {
		t.Fatalf("POST error = %v", err)
	}
	if err := client.validateFrozenRoute(ProtocolV2RouteRequest{
		RouteID: "demo.sse", ContractVersion: "demo.sse@1", Method: http.MethodHead,
		Authority: protocolV2FilteredHostRequestAuthority(),
	}); !errors.Is(err, ErrProtocolV2RouteInvalid) {
		t.Fatalf("SSE HEAD error = %v", err)
	}
}

type protocolV2RouteTestServer struct {
	pluginwire.UnimplementedPluginRuntimeServiceServer
	invoke func(context.Context, *pluginwire.RouteRequest) (*pluginwire.RouteResponse, error)
}

type recordingProtocolV2ActorDelegationIssuer struct {
	request hostapi.ProtocolV2ActorDelegationBundleRequest
	grants  []hostapi.ProtocolV2ActorDelegationGrant
	err     error
	calls   int
}

func (i *recordingProtocolV2ActorDelegationIssuer) IssueProtocolV2ActorDelegations(
	_ context.Context,
	request hostapi.ProtocolV2ActorDelegationBundleRequest,
) ([]hostapi.ProtocolV2ActorDelegationGrant, error) {
	i.calls++
	i.request = request
	return append([]hostapi.ProtocolV2ActorDelegationGrant(nil), i.grants...), i.err
}

func (s *protocolV2RouteTestServer) InvokeRoute(ctx context.Context, request *pluginwire.RouteRequest) (*pluginwire.RouteResponse, error) {
	return s.invoke(ctx, request)
}

func newProtocolV2RouteTestClient(
	t *testing.T,
	instanceID string,
	invoke func(context.Context, *pluginwire.RouteRequest) (*pluginwire.RouteResponse, error),
) *protocolV2Client {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	pluginwire.RegisterPluginRuntimeServiceServer(server, &protocolV2RouteTestServer{invoke: invoke})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close() })
	connection, err := grpc.NewClient("passthrough:///route-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	identity := &protocolwire.ExtensionIdentity{
		ExtensionId: "demo.plugin", ExtensionVersion: "1.0.0", ArtifactDigest: "digest-v1",
		TrustGrantId: "grant-1", RuntimeEpoch: 1, InstanceId: instanceID,
	}
	return newProtocolV2Client(pluginwire.NewPluginRuntimeServiceClient(connection), protocolV2ClientConfig{
		identity: identity, instance: instanceID,
		routes: []extensions.ManifestRoute{{
			ID: "demo.route", ContractVersion: "demo.route@1", Methods: []string{http.MethodGet, http.MethodPost},
			Guard: extensionmanifest.GuardCorePublic, RequestSchema: "demo.request@1", ResponseSchema: "demo.response@1",
		}},
	})
}

func protocolV2RouteTestRequest() ProtocolV2RouteRequest {
	return ProtocolV2RouteRequest{
		RouteID: "demo.route", ContractVersion: "demo.route@1", Method: http.MethodGet, Path: "/demo",
		Authority:     protocolV2FilteredHostRequestAuthority(),
		RequestSchema: "demo.request@1", ResponseSchema: "demo.response@1", Timeout: time.Second,
	}
}

func protocolV2RouteTestResponse(request *pluginwire.RouteRequest, body map[string]any) *pluginwire.RouteResponse {
	value, _ := structpb.NewStruct(body)
	return &pluginwire.RouteResponse{
		Context: &protocolwire.ResponseContext{
			RequestId: request.GetContext().GetRequestId(), Trace: proto.Clone(request.GetContext().GetTrace()).(*protocolwire.TraceContext),
			Extension: proto.Clone(request.GetContext().GetExtension()).(*protocolwire.ExtensionIdentity), ServerTime: timestamppb.Now(),
		},
		StatusCode: http.StatusCreated,
		Headers:    []*protocolwire.Header{{Name: "X-Result", Values: []string{"one", "two"}}},
		Body:       &protocolwire.TypedDocument{SchemaId: "demo.response", SchemaVersion: "1", Value: value},
	}
}

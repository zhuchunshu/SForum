package extensionsruntime

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	pluginwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	"google.golang.org/grpc"
)

func TestProtocolV2RawAuthorityRequiresExactFrozenRoute(t *testing.T) {
	var received *pluginwire.RouteRequest
	calls := 0
	client := newProtocolV2RouteTestClient(t, "runtime-raw-route", func(
		_ context.Context,
		request *pluginwire.RouteRequest,
	) (*pluginwire.RouteResponse, error) {
		calls++
		received = request
		return protocolV2RouteTestResponse(request, map[string]any{"ok": true}), nil
	})
	client.routes[0].Guard = extensionmanifest.GuardCoreRaw
	request := protocolV2RouteTestRequest()
	request.Authority = protocolV2RawRequestAuthority()
	request.Headers = http.Header{
		"Cookie":        {"session=one", "preferences=two"},
		"Authorization": {"Bearer one", "Bearer two"},
		"X-Api-Key":     {"api-key-one", "api-key-two"},
		"X-Auth-Token":  {"auth-token-one", "auth-token-two"},
		"X-Test":        {"one", "two"},
	}
	if _, err := client.InvokeRouteContext(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	headers, err := protocolV2RouteHTTPHeaders(received.GetHeaders())
	if err != nil || !reflect.DeepEqual(headers.Values("X-Test"), request.Headers.Values("X-Test")) {
		t.Fatalf("raw headers=%#v error=%v", headers, err)
	}
	assertProtocolV2CredentialHeaders(t, headers, request.Headers, true)

	request.Authority = protocolV2FilteredHostRequestAuthority()
	if _, err := client.InvokeRouteContext(context.Background(), request); !errors.Is(err, ErrProtocolV2RouteInvalid) || calls != 1 {
		t.Fatalf("filtered downgrade error=%v calls=%d", err, calls)
	}
	client.routes[0].Guard = extensionmanifest.GuardCorePublic
	request.Authority = protocolV2RawRequestAuthority()
	if _, err := client.InvokeRouteContext(context.Background(), request); !errors.Is(err, ErrProtocolV2RouteInvalid) || calls != 1 {
		t.Fatalf("raw escalation error=%v calls=%d", err, calls)
	}
	request.Authority = protocolV2FilteredHostRequestAuthority()
	received = nil
	if _, err := client.InvokeRouteContext(context.Background(), request); err != nil || calls != 2 {
		t.Fatalf("filtered route error=%v calls=%d", err, calls)
	}
	headers, err = protocolV2RouteHTTPHeaders(received.GetHeaders())
	if err != nil || !reflect.DeepEqual(headers.Values("X-Test"), request.Headers.Values("X-Test")) {
		t.Fatalf("filtered headers=%#v error=%v", headers, err)
	}
	assertProtocolV2CredentialHeaders(t, headers, request.Headers, false)
}

func TestProtocolV2RawAuthorityRequiresExactFrozenGuard(t *testing.T) {
	var received *pluginwire.RouteRequest
	calls := 0
	client := protocolV2GuardTestClient(t, func(
		_ context.Context,
		request *pluginwire.RouteRequest,
	) (*pluginwire.RouteResponse, error) {
		calls++
		received = request
		return protocolV2GuardTestResponse(request, http.StatusNoContent), nil
	})
	client.guards[0].Kind = "raw_request"
	request := protocolV2GuardTestRequest()
	request.Authority = protocolV2RawRequestAuthority()
	request.Headers = http.Header{
		"Cookie": {"session=one", "session=two"}, "Authorization": {"Bearer raw"},
		"X-Api-Key": {"guard-api-key"}, "X-Auth-Token": {"guard-auth-token"},
	}
	if err := client.InvokeGuardContext(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	headers, err := protocolV2RouteHTTPHeaders(received.GetHeaders())
	if err != nil || headers.Get("X-SForum-Guard-Kind") != "raw_request" {
		t.Fatalf("raw guard headers=%#v error=%v", headers, err)
	}
	assertProtocolV2CredentialHeaders(t, headers, request.Headers, true)

	request.Authority = protocolV2FilteredCustomRequestAuthority()
	if err := client.InvokeGuardContext(context.Background(), request); !errors.Is(err, ErrProtocolV2GuardInvalid) || calls != 1 {
		t.Fatalf("guard downgrade error=%v calls=%d", err, calls)
	}
	client.guards[0].Kind = "custom"
	request.Authority = protocolV2RawRequestAuthority()
	if err := client.InvokeGuardContext(context.Background(), request); !errors.Is(err, ErrProtocolV2GuardInvalid) || calls != 1 {
		t.Fatalf("guard escalation error=%v calls=%d", err, calls)
	}
	request.Authority = protocolV2FilteredCustomRequestAuthority()
	request.Headers = http.Header{
		"Cookie": {"filtered-session"}, "Authorization": {"Bearer filtered"},
		"X-Api-Key": {"filtered-api-key"}, "X-Auth-Token": {"filtered-auth-token"}, "X-Test": {"guard-visible"},
	}
	received = nil
	if err := client.InvokeGuardContext(context.Background(), request); err != nil || calls != 2 {
		t.Fatalf("filtered guard error=%v calls=%d", err, calls)
	}
	headers, err = protocolV2RouteHTTPHeaders(received.GetHeaders())
	if err != nil || headers.Get("X-Test") != "guard-visible" || headers.Get("X-SForum-Guard-Kind") != "custom" {
		t.Fatalf("filtered guard headers=%#v error=%v", headers, err)
	}
	assertProtocolV2CredentialHeaders(t, headers, request.Headers, false)
}

func TestProtocolV2RequestAuthorityStripsHostMetadataBeforeRPC(t *testing.T) {
	var received *pluginwire.RouteRequest
	client := newProtocolV2RouteTestClient(t, "runtime-header-fence", func(
		_ context.Context,
		request *pluginwire.RouteRequest,
	) (*pluginwire.RouteResponse, error) {
		received = request
		return protocolV2RouteTestResponse(request, map[string]any{"ok": true}), nil
	})
	client.routes[0].Guard = extensionmanifest.GuardCoreRaw
	request := protocolV2RouteTestRequest()
	request.Authority = protocolV2RawRequestAuthority()
	request.Headers = http.Header{
		"Host":                {"internal.example"},
		"Content-Length":      {"42"},
		"Proxy-Authorization": {"Basic secret"},
		"Proxy-Connection":    {"keep-alive"},
		"X-CSRF-Token":        {"secret"},
		"X-SForum-Actor-ID":   {"42"},
		"Transfer-Encoding":   {"chunked"},
		"Connection":          {"X-Private-Hop"},
		"connection":          {"X-Lower-Hop"},
		"CONNECTION":          {"X-Upper-Hop"},
		"X-Private-Hop":       {"secret"},
		"X-Lower-Hop":         {"lower-secret"},
		"X-Upper-Hop":         {"upper-secret"},
		"X-Keep":              {"one", "two"},
	}
	if _, err := client.InvokeRouteContext(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	headers, err := protocolV2RouteHTTPHeaders(received.GetHeaders())
	if err != nil || !reflect.DeepEqual(headers.Values("X-Keep"), []string{"one", "two"}) {
		t.Fatalf("headers=%#v error=%v", headers, err)
	}
	for _, name := range []string{
		"Host", "Content-Length", "Proxy-Authorization", "Proxy-Connection", "X-Csrf-Token",
		"X-SForum-Actor-ID", "Transfer-Encoding", "Connection", "X-Private-Hop", "X-Lower-Hop", "X-Upper-Hop",
	} {
		if values := headers.Values(name); len(values) != 0 {
			t.Fatalf("blocked header %s survived: %#v", name, values)
		}
	}
}

func TestProtocolV2RawAuthorityIsPreservedForRouteStreams(t *testing.T) {
	opens := make(chan *pluginwire.RouteStreamOpen, 1)
	client := newProtocolV2RouteStreamTestClient(t, func(
		stream grpc.BidiStreamingServer[pluginwire.RouteStreamFrame, pluginwire.RouteStreamFrame],
	) error {
		frame, err := stream.Recv()
		if err != nil {
			return err
		}
		opens <- frame.GetOpen()
		return nil
	})
	client.routes[0].Guard = extensionmanifest.GuardCoreRaw
	request := ProtocolV2RouteStreamRequest{
		RouteID: "demo.stream", ContractVersion: "demo.stream@1", Method: http.MethodPost,
		Path: "/stream", Mode: extensionmanifest.RouteModeStream,
		Authority: protocolV2RawRequestAuthority(),
		Headers: http.Header{
			"Cookie": {"session=stream"}, "Authorization": {"Bearer stream"},
			"X-Api-Key": {"stream-api-key"}, "X-Auth-Token": {"stream-auth-token"}, "X-Test": {"stream-visible"},
			"connection": {"X-Stream-Lower-Hop"}, "CONNECTION": {"X-Stream-Upper-Hop"},
			"X-Stream-Lower-Hop": {"lower-secret"}, "X-Stream-Upper-Hop": {"upper-secret"},
		},
	}
	stream, err := client.OpenRouteStreamContext(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	open := <-opens
	headers, err := protocolV2RouteHTTPHeaders(open.GetHeaders())
	if err != nil || headers.Get("X-Test") != "stream-visible" {
		t.Fatalf("stream headers=%#v error=%v", headers, err)
	}
	assertProtocolV2CredentialHeaders(t, headers, request.Headers, true)
	assertProtocolV2BlockedHeaders(t, headers, "Connection", "X-Stream-Lower-Hop", "X-Stream-Upper-Hop")
	stream.Cancel()

	request.Authority = protocolV2FilteredHostRequestAuthority()
	if _, err := client.OpenRouteStreamContext(context.Background(), request); !errors.Is(err, ErrProtocolV2RouteStreamInvalid) {
		t.Fatalf("stream downgrade error=%v", err)
	}
	client.routes[0].Guard = extensionmanifest.GuardCorePublic
	stream, err = client.OpenRouteStreamContext(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	open = <-opens
	headers, err = protocolV2RouteHTTPHeaders(open.GetHeaders())
	if err != nil || headers.Get("X-Test") != "stream-visible" {
		t.Fatalf("filtered stream headers=%#v error=%v", headers, err)
	}
	assertProtocolV2CredentialHeaders(t, headers, request.Headers, false)
	assertProtocolV2BlockedHeaders(t, headers, "Connection", "X-Stream-Lower-Hop", "X-Stream-Upper-Hop")
	stream.Cancel()

	client.routes[0].Mode = extensionmanifest.RouteModeWebSocket
	request.Mode = extensionmanifest.RouteModeWebSocket
	request.Headers.Set("Host", "internal.example")
	request.Headers.Set("Connection", "Upgrade")
	request.Headers.Set("Upgrade", "websocket")
	request.Headers.Set("Proxy-Authorization", "Basic proxy-secret")
	request.Headers.Set("X-CSRF-Token", "csrf-secret")
	request.Headers.Set("X-SForum-Forged", "forged")
	stream, err = client.OpenRouteStreamContext(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Cancel()
	open = <-opens
	headers, err = protocolV2RouteHTTPHeaders(open.GetHeaders())
	if err != nil || headers.Get("X-Test") != "stream-visible" {
		t.Fatalf("filtered websocket headers=%#v error=%v", headers, err)
	}
	assertProtocolV2CredentialHeaders(t, headers, request.Headers, false)
	assertProtocolV2BlockedHeaders(t, headers,
		"Host", "Connection", "Upgrade", "Proxy-Authorization", "X-CSRF-Token", "X-SForum-Forged",
		"X-Stream-Lower-Hop", "X-Stream-Upper-Hop",
	)
}

func assertProtocolV2BlockedHeaders(t *testing.T, headers http.Header, names ...string) {
	t.Helper()
	for _, name := range names {
		if values := headers.Values(name); len(values) != 0 {
			t.Fatalf("blocked header %s survived: %#v", name, values)
		}
	}
}

func assertProtocolV2CredentialHeaders(t *testing.T, headers, source http.Header, raw bool) {
	t.Helper()
	for _, name := range []string{"Cookie", "Authorization", "X-API-Key", "X-Auth-Token"} {
		got := headers.Values(name)
		if raw && !reflect.DeepEqual(got, source.Values(name)) {
			t.Fatalf("raw credential %s=%#v, want %#v", name, got, source.Values(name))
		}
		if !raw && len(got) != 0 {
			t.Fatalf("filtered credential %s survived: %#v", name, got)
		}
	}
}

func protocolV2FilteredHostRequestAuthority() ProtocolV2RequestAuthority {
	return ProtocolV2RequestAuthority{
		Mode: ProtocolV2RequestAuthorityFiltered, GuardKind: ProtocolV2RequestGuardHost,
	}
}

func protocolV2FilteredCustomRequestAuthority() ProtocolV2RequestAuthority {
	return ProtocolV2RequestAuthority{
		Mode: ProtocolV2RequestAuthorityFiltered, GuardKind: ProtocolV2RequestGuardCustom,
	}
}

func protocolV2RawRequestAuthority() ProtocolV2RequestAuthority {
	return ProtocolV2RequestAuthority{
		Mode: ProtocolV2RequestAuthorityRaw, GuardKind: ProtocolV2RequestGuardRawRequest,
	}
}

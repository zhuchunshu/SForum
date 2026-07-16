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
		"X-Test":        {"one", "two"},
	}
	if _, err := client.InvokeRouteContext(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	headers, err := protocolV2RouteHTTPHeaders(received.GetHeaders())
	if err != nil || !reflect.DeepEqual(headers.Values("Cookie"), request.Headers.Values("Cookie")) ||
		!reflect.DeepEqual(headers.Values("Authorization"), request.Headers.Values("Authorization")) ||
		!reflect.DeepEqual(headers.Values("X-Test"), request.Headers.Values("X-Test")) {
		t.Fatalf("raw headers=%#v error=%v", headers, err)
	}

	request.Authority = protocolV2FilteredHostRequestAuthority()
	if _, err := client.InvokeRouteContext(context.Background(), request); !errors.Is(err, ErrProtocolV2RouteInvalid) || calls != 1 {
		t.Fatalf("filtered downgrade error=%v calls=%d", err, calls)
	}
	client.routes[0].Guard = extensionmanifest.GuardCorePublic
	request.Authority = protocolV2RawRequestAuthority()
	if _, err := client.InvokeRouteContext(context.Background(), request); !errors.Is(err, ErrProtocolV2RouteInvalid) || calls != 1 {
		t.Fatalf("raw escalation error=%v calls=%d", err, calls)
	}
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
	}
	if err := client.InvokeGuardContext(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	headers, err := protocolV2RouteHTTPHeaders(received.GetHeaders())
	if err != nil || !reflect.DeepEqual(headers.Values("Cookie"), request.Headers.Values("Cookie")) ||
		headers.Get("Authorization") != "Bearer raw" || headers.Get("X-SForum-Guard-Kind") != "raw_request" {
		t.Fatalf("raw guard headers=%#v error=%v", headers, err)
	}

	request.Authority = protocolV2FilteredCustomRequestAuthority()
	if err := client.InvokeGuardContext(context.Background(), request); !errors.Is(err, ErrProtocolV2GuardInvalid) || calls != 1 {
		t.Fatalf("guard downgrade error=%v calls=%d", err, calls)
	}
	client.guards[0].Kind = "custom"
	request.Authority = protocolV2RawRequestAuthority()
	if err := client.InvokeGuardContext(context.Background(), request); !errors.Is(err, ErrProtocolV2GuardInvalid) || calls != 1 {
		t.Fatalf("guard escalation error=%v calls=%d", err, calls)
	}
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
		"X-Private-Hop":       {"secret"},
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
		"X-SForum-Actor-ID", "Transfer-Encoding", "Connection", "X-Private-Hop",
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
		},
	}
	stream, err := client.OpenRouteStreamContext(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Cancel()
	open := <-opens
	headers, err := protocolV2RouteHTTPHeaders(open.GetHeaders())
	if err != nil || headers.Get("Cookie") != "session=stream" || headers.Get("Authorization") != "Bearer stream" {
		t.Fatalf("stream headers=%#v error=%v", headers, err)
	}

	request.Authority = protocolV2FilteredHostRequestAuthority()
	if _, err := client.OpenRouteStreamContext(context.Background(), request); !errors.Is(err, ErrProtocolV2RouteStreamInvalid) {
		t.Fatalf("stream downgrade error=%v", err)
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

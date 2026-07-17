package http

import (
	"context"
	"errors"
	"io"
	stdhttp "net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

func TestRouteV2StreamPreAdmissionCancellationStaysPristine(t *testing.T) {
	dispatcher, runtime, traces := routeV2StreamTestDispatcher(t)
	prepared, err := dispatcher.PrepareStream(
		context.Background(), routes.DispatchRequest{Method: stdhttp.MethodGet, Path: "/stream-v2"},
	)
	if err != nil || prepared.Dispatch == nil {
		t.Fatalf("prepared=%#v error=%v", prepared, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = prepared.Dispatch.Open(ctx)
	if !errors.Is(err, context.Canceled) || errors.Is(err, routes.ErrDispatchTransport) ||
		runtime.calls != 0 || runtime.streamOpenCalls != 0 || len(traces.RouteTraces(0)) != 0 {
		t.Fatalf("error=%v routeCalls=%d streamCalls=%d traces=%#v",
			err, runtime.calls, runtime.streamOpenCalls, traces.RouteTraces(0))
	}
}

func TestRouteV2StreamPreflightFailureRecordsObservedExecution(t *testing.T) {
	dispatcher, runtime, traces := routeV2StreamTestDispatcher(t)
	runtime.err = errors.New("runtime crashed after preflight dispatch")
	prepared, err := dispatcher.PrepareStream(
		context.Background(), routes.DispatchRequest{Method: stdhttp.MethodGet, Path: "/stream-v2"},
	)
	if err != nil || prepared.Dispatch == nil {
		t.Fatalf("prepared=%#v error=%v", prepared, err)
	}

	_, err = prepared.Dispatch.Open(context.Background())
	records := traces.RouteTraces(0)
	if !errors.Is(err, routes.ErrDispatchTransport) || runtime.calls != 1 || runtime.streamOpenCalls != 0 ||
		len(records) != 1 || records[0].Outcome != routes.RouteTraceTransportFailed ||
		records[0].CommitState != routes.RouteCommitSideEffectStarted {
		t.Fatalf("error=%v routeCalls=%d streamCalls=%d traces=%#v", err, runtime.calls, runtime.streamOpenCalls, records)
	}
}

func TestRouteV2StreamSharesCorrelationAndPreservesExactQuery(t *testing.T) {
	dispatcher, runtime, _ := routeV2StreamTestDispatcher(t)
	const rawQuery = "tag=one&tag=&tag=a%2Bb&page=2"
	open := func() string {
		prepared, err := dispatcher.PrepareStream(context.Background(), routes.DispatchRequest{
			Method: stdhttp.MethodGet, Path: "/stream-v2", Query: rawQuery,
		})
		if err != nil || prepared.Dispatch == nil {
			t.Fatalf("prepared=%#v error=%v", prepared, err)
		}
		if _, err := prepared.Dispatch.Open(context.Background()); !errors.Is(err, routes.ErrDispatchTransport) {
			t.Fatalf("open error=%v", err)
		}
		if runtime.request.CorrelationID == "" || runtime.request.CorrelationID != runtime.streamRequest.CorrelationID {
			t.Fatalf("correlation preflight=%q stream=%q", runtime.request.CorrelationID, runtime.streamRequest.CorrelationID)
		}
		if !strings.HasPrefix(runtime.request.CorrelationID, "route_") ||
			runtime.request.QueryParameters["tag"] != "one" ||
			!reflect.DeepEqual(runtime.request.QueryParameterValues["tag"], []string{"one", "", "a+b"}) ||
			runtime.streamRequest.Path != "/stream-v2?"+rawQuery {
			t.Fatalf("preflight=%#v stream=%#v", runtime.request, runtime.streamRequest)
		}
		return runtime.request.CorrelationID
	}
	first := open()
	if second := open(); second == first {
		t.Fatalf("separate stream admissions reused correlation %q", first)
	}
}

func TestRouteV2StreamRejectsNonUpgradeInformationalPreflight(t *testing.T) {
	for _, status := range []int{stdhttp.StatusContinue, stdhttp.StatusEarlyHints, stdhttp.StatusOK} {
		t.Run(stdhttp.StatusText(status), func(t *testing.T) {
			dispatcher, runtime, _ := routeV2StreamTestDispatcher(t)
			runtime.response = extensionsruntime.ProtocolV2RouteResponse{StatusCode: status, StreamFollows: true}
			prepared, err := dispatcher.PrepareStream(context.Background(), routes.DispatchRequest{
				Method: stdhttp.MethodGet, Path: "/stream-v2",
			})
			if err != nil || prepared.Dispatch == nil {
				t.Fatalf("prepared=%#v error=%v", prepared, err)
			}
			if _, err := prepared.Dispatch.Open(context.Background()); !errors.Is(err, routes.ErrDispatchTransport) {
				t.Fatalf("status=%d error=%v", status, err)
			}
			if runtime.calls != 1 || runtime.streamOpenCalls != 0 {
				t.Fatalf("status=%d routeCalls=%d streamCalls=%d", status, runtime.calls, runtime.streamOpenCalls)
			}
		})
	}
}

func routeV2StreamTestDispatcher(
	t *testing.T,
) (*routes.Dispatcher, *routeDispatcherV2StreamRuntime, *routes.RouteTraceRing) {
	t.Helper()
	artifact := routeDispatcherArtifact("stream-v2", 'a')
	declaration := routeDispatcherManifestRoute(
		"stream-v2.route", extensionmanifest.RouteActionAdd, "/stream-v2", stdhttp.MethodGet,
	)
	declaration.Mode = extensionmanifest.RouteModeWebSocket
	registry := routes.NewRegistry()
	if _, err := registry.Publish(routes.Publication{Plugins: []routes.PluginRouteSet{{
		Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{declaration},
	}}}); err != nil {
		t.Fatal(err)
	}
	runtime := &routeDispatcherV2StreamRuntime{routeDispatcherV2Runtime: newRouteDispatcherV2RuntimeForArtifact(t, artifact)}
	runtime.response = extensionsruntime.ProtocolV2RouteResponse{
		StatusCode: fiber.StatusSwitchingProtocols, StreamFollows: true,
	}
	traces := routes.NewRouteTraceRing(8)
	return routes.NewDispatcher(routes.DispatcherConfig{
		Plans: routeRegistryPlanResolver{registry: registry}, Steps: NewBufferedRouteStepInvoker(runtime),
		Guard: HostRouteGuardAuthorizer{}, Trace: traces,
	}), runtime, traces
}

type routeDispatcherV2StreamRuntime struct {
	*routeDispatcherV2Runtime
	streamOpenCalls int
	streamRequest   extensionsruntime.ProtocolV2RouteStreamRequest
}

func (r *routeDispatcherV2StreamRuntime) OpenRouteStreamInstance(
	_ context.Context,
	_ extensionsruntime.RuntimeInstanceIdentity,
	request extensionsruntime.ProtocolV2RouteStreamRequest,
) (*extensionsruntime.ProtocolV2RouteStream, error) {
	r.streamOpenCalls++
	r.streamRequest = request
	return nil, errors.New("stream open is not expected")
}

func TestRouteV2StreamSessionCopiesChunksAndAcceptsBoundTerminal(t *testing.T) {
	wire := &fakeRouteV2WireStream{
		chunks: []extensionsruntime.ProtocolV2RouteStreamChunk{{Sequence: 1, Data: []byte("first"), Final: true}},
		response: extensionsruntime.ProtocolV2RouteStreamResponse{
			StatusCode: stdhttp.StatusCreated,
			Headers:    stdhttp.Header{"X-Result": {"done"}, "Set-Cookie": {"private=1"}},
		},
	}
	session := &routeV2StreamSession{stream: wire, expectedStatus: stdhttp.StatusCreated}
	if err := session.Send([]byte("request"), true); err != nil || err == nil && string(wire.sent) != "request" {
		t.Fatalf("send=%q err=%v", wire.sent, err)
	}
	if err := session.CloseRequest(); err != nil || !wire.requestClosed {
		t.Fatalf("request closed=%t err=%v", wire.requestClosed, err)
	}
	chunk, err := session.Recv()
	if err != nil || chunk.Sequence != 1 || string(chunk.Data) != "first" || !chunk.Final {
		t.Fatalf("chunk=%#v err=%v", chunk, err)
	}
	chunk.Data[0] = 'x'
	if _, err := session.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("terminal err=%v", err)
	}
	response, ok := session.Response()
	if !ok || response.Status != stdhttp.StatusCreated || response.Headers.Get("X-Result") != "done" || response.Headers.Get("Set-Cookie") != "" {
		t.Fatalf("response=%#v ok=%t", response, ok)
	}
	if wire.cancelled {
		t.Fatal("successful terminal cancelled the runtime stream")
	}
}

func TestRouteV2StreamSessionCancelsOnTransportFailureOrStatusDrift(t *testing.T) {
	for name, wire := range map[string]*fakeRouteV2WireStream{
		"transport": {recvErr: errors.New("closed")},
		"status": {
			response: extensionsruntime.ProtocolV2RouteStreamResponse{StatusCode: stdhttp.StatusNoContent},
		},
	} {
		t.Run(name, func(t *testing.T) {
			session := &routeV2StreamSession{stream: wire, expectedStatus: stdhttp.StatusOK}
			_, err := session.Recv()
			if err == nil || !wire.cancelled {
				t.Fatalf("error=%v cancelled=%t", err, wire.cancelled)
			}
			if _, ok := session.Response(); ok {
				t.Fatal("failed stream exposed a terminal response")
			}
		})
	}
}

func TestRouteV2StreamSessionAbsorbsCloseErrorOnlyAfterExpectedTerminal(t *testing.T) {
	closeErr := errors.New("request stream already closed")
	for _, test := range []struct {
		name     string
		response extensionsruntime.ProtocolV2RouteStreamResponse
		wantErr  bool
	}{
		{name: "expected terminal", response: extensionsruntime.ProtocolV2RouteStreamResponse{StatusCode: stdhttp.StatusSwitchingProtocols}},
		{name: "status drift", response: extensionsruntime.ProtocolV2RouteStreamResponse{StatusCode: stdhttp.StatusOK}, wantErr: true},
		{name: "missing terminal", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			wire := &fakeRouteV2WireStream{response: test.response, closeErr: closeErr}
			session := &routeV2StreamSession{stream: wire, expectedStatus: stdhttp.StatusSwitchingProtocols}
			err := session.CloseRequest()
			if !wire.requestClosed || test.wantErr != errors.Is(err, closeErr) {
				t.Fatalf("requestClosed=%t error=%v", wire.requestClosed, err)
			}
		})
	}
}

type fakeRouteV2WireStream struct {
	chunks        []extensionsruntime.ProtocolV2RouteStreamChunk
	response      extensionsruntime.ProtocolV2RouteStreamResponse
	recvErr       error
	sent          []byte
	requestClosed bool
	cancelled     bool
	closeErr      error
}

func (s *fakeRouteV2WireStream) Send(data []byte, _ bool) error {
	s.sent = append([]byte(nil), data...)
	return nil
}

func (s *fakeRouteV2WireStream) CloseRequest() error {
	s.requestClosed = true
	return s.closeErr
}

func (s *fakeRouteV2WireStream) Recv() (extensionsruntime.ProtocolV2RouteStreamChunk, error) {
	if s.recvErr != nil {
		return extensionsruntime.ProtocolV2RouteStreamChunk{}, s.recvErr
	}
	if len(s.chunks) == 0 {
		return extensionsruntime.ProtocolV2RouteStreamChunk{}, io.EOF
	}
	chunk := s.chunks[0]
	s.chunks = s.chunks[1:]
	return chunk, nil
}

func (s *fakeRouteV2WireStream) Response() (extensionsruntime.ProtocolV2RouteStreamResponse, bool) {
	return s.response, s.response.StatusCode != 0
}

func (s *fakeRouteV2WireStream) Cancel() { s.cancelled = true }

var _ routeV2WireStream = (*fakeRouteV2WireStream)(nil)

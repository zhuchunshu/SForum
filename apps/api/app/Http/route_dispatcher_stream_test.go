package http

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

func TestRouteDispatcherStreamsSSEThroughFiberAndCommitsTrace(t *testing.T) {
	registry := routes.NewRegistry()
	artifact := routeDispatcherArtifact("stream.http", 'a')
	declaration := routeDispatcherManifestRoute("stream.http.events", extensionmanifest.RouteActionAdd, "/api/v1/events", "GET")
	declaration.Mode = extensionmanifest.RouteModeSSE
	if _, err := registry.Publish(routes.Publication{Plugins: []routes.PluginRouteSet{{
		Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{declaration},
	}}}); err != nil {
		t.Fatal(err)
	}
	session := &streamHTTPTestSession{
		chunks: []routes.RouteStreamChunk{
			{Sequence: 1, Data: []byte("data: one\n\n")},
			{Sequence: 2, Data: []byte("data: two\n\n"), Final: true},
		},
		response: routes.DispatchResponse{Status: stdhttp.StatusOK, Headers: stdhttp.Header{"Content-Type": {"text/event-stream"}}},
	}
	traces := routes.NewRouteTraceRing(8)
	dispatcher := routes.NewDispatcher(routes.DispatcherConfig{
		Plans: routeRegistryPlanResolver{registry: registry}, Steps: &streamHTTPTestInvoker{
			start: routes.RouteStreamStart{
				Response: routes.DispatchResponse{Status: stdhttp.StatusOK, Headers: stdhttp.Header{"Content-Type": {"text/event-stream"}}},
				Session:  session,
			},
		},
		Guard: HostRouteGuardAuthorizer{}, Trace: traces,
	})
	app := fiber.New(fiber.Config{StreamRequestBody: true, DisablePreParseMultipartForm: true})
	app.Use(routeDispatcherMiddleware(dispatcher, nil))
	response, err := app.Test(httptest.NewRequest(stdhttp.MethodGet, "/api/v1/events", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != stdhttp.StatusOK || response.Header.Get("Content-Type") != "text/event-stream" ||
		string(body) != "data: one\n\ndata: two\n\n" || !session.requestClosed || session.cancelled {
		t.Fatalf("status=%d headers=%v body=%q session=%#v", response.StatusCode, response.Header, body, session)
	}
	records := traces.RouteTraces(8)
	if len(records) != 2 || records[0].Outcome != routes.RouteTraceSucceeded || records[1].Outcome != routes.RouteTraceCommitted {
		t.Fatalf("traces=%#v", records)
	}
}

func TestPumpRouteStreamRequestUsesBoundedChunks(t *testing.T) {
	body := bytes.Repeat([]byte("x"), extensionsruntime.MaxProtocolV2RouteChunkSize+17)
	session := &streamHTTPTestSession{}
	if err := pumpRouteStreamRequest(bytes.NewReader(body), session); err != nil {
		t.Fatal(err)
	}
	if !session.requestClosed || session.maxRequestChunk != extensionsruntime.MaxProtocolV2RouteChunkSize || session.requestBytes != len(body) || len(session.requestChunks) != 2 {
		t.Fatalf("closed=%t max=%d bytes=%d chunks=%v", session.requestClosed, session.maxRequestChunk, session.requestBytes, session.requestChunks)
	}
}

func TestStreamRouteResponseCancelsOnRuntimeFailure(t *testing.T) {
	registry := routes.NewRegistry()
	artifact := routeDispatcherArtifact("stream.failure", 'b')
	declaration := routeDispatcherManifestRoute("stream.failure.body", extensionmanifest.RouteActionAdd, "/failure", "GET")
	declaration.Mode = extensionmanifest.RouteModeStream
	if _, err := registry.Publish(routes.Publication{Plugins: []routes.PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{declaration}}}}); err != nil {
		t.Fatal(err)
	}
	session := &streamHTTPTestSession{recvErr: errors.New("runtime crashed")}
	traces := routes.NewRouteTraceRing(8)
	dispatcher := routes.NewDispatcher(routes.DispatcherConfig{
		Plans: routeRegistryPlanResolver{registry: registry}, Steps: &streamHTTPTestInvoker{start: routes.RouteStreamStart{
			Response: routes.DispatchResponse{Status: stdhttp.StatusOK}, Session: session,
		}}, Guard: HostRouteGuardAuthorizer{}, Trace: traces,
	})
	prepared, err := dispatcher.PrepareStream(context.Background(), routes.DispatchRequest{Method: "GET", Path: "/failure"})
	if err != nil {
		t.Fatal(err)
	}
	start, err := prepared.Dispatch.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	prepared.Dispatch.ResponseStarted()
	streamRouteResponse(bufio.NewWriter(bytes.NewBuffer(nil)), start.Session, prepared.Dispatch)
	if !session.cancelled {
		t.Fatal("failed stream was not cancelled")
	}
	records := traces.RouteTraces(8)
	if len(records) != 2 || records[1].Outcome != routes.RouteTraceTransportFailed {
		t.Fatalf("traces=%#v", records)
	}
}

func TestRouteDispatcherBridgesWebSocketMessagesAndDisconnect(t *testing.T) {
	registry := routes.NewRegistry()
	artifact := routeDispatcherArtifact("stream.websocket", 'c')
	declaration := routeDispatcherManifestRoute("stream.websocket.echo", extensionmanifest.RouteActionAdd, "/socket", "GET")
	declaration.Mode = extensionmanifest.RouteModeWebSocket
	if _, err := registry.Publish(routes.Publication{Plugins: []routes.PluginRouteSet{{
		Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{declaration},
	}}}); err != nil {
		t.Fatal(err)
	}
	session := newWebSocketEchoSession()
	traces := routes.NewRouteTraceRing(8)
	invoker := &streamHTTPTestInvoker{start: routes.RouteStreamStart{
		Response: routes.DispatchResponse{
			Status:  fiber.StatusSwitchingProtocols,
			Headers: stdhttp.Header{"Sec-WebSocket-Protocol": {"sforum.echo.v1"}},
		},
		Session: session,
	}}
	contextKey := routeWebSocketRequestContextKey{}
	guard := &countingRouteGuard{contextKey: contextKey, contextValue: "request-bound"}
	dispatcher := routes.NewDispatcher(routes.DispatcherConfig{
		Plans: routeRegistryPlanResolver{registry: registry}, Steps: invoker,
		Guard: guard, Trace: traces,
	})
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.SetContext(context.WithValue(c.Context(), contextKey, "request-bound"))
		return c.Next()
	})
	app.Use(routeDispatcherMiddleware(dispatcher, nil))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serverDone := make(chan error, 1)
	go func() { serverDone <- app.Listener(listener) }()
	t.Cleanup(func() {
		_ = app.Shutdown()
		_ = listener.Close()
		<-serverDone
	})
	dialer := websocket.Dialer{Subprotocols: []string{"sforum.echo.v1"}}
	connection, response, err := dialer.Dial("ws://"+listener.Addr().String()+"/socket", nil)
	if err != nil {
		t.Fatalf("dial status=%v err=%v", response, err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	if response.StatusCode != fiber.StatusSwitchingProtocols || connection.Subprotocol() != "sforum.echo.v1" {
		t.Fatalf("status=%d subprotocol=%q", response.StatusCode, connection.Subprotocol())
	}
	if err := connection.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	messageType, payload, err := connection.ReadMessage()
	if err != nil || messageType != websocket.BinaryMessage || string(payload) != "hello" {
		t.Fatalf("messageType=%d payload=%q err=%v", messageType, payload, err)
	}
	_ = connection.Close()
	select {
	case <-session.done:
	case <-time.After(2 * time.Second):
		t.Fatal("websocket disconnect did not cancel the runtime session")
	}
	if invoker.openCalls.Load() != 1 || string(session.received) != "hello" {
		t.Fatalf("openCalls=%d received=%q", invoker.openCalls.Load(), session.received)
	}
	if guard.calls.Load() != 1 {
		t.Fatalf("guard calls=%d", guard.calls.Load())
	}
	records := traces.RouteTraces(8)
	if len(records) != 2 || records[0].Outcome != routes.RouteTraceSucceeded || records[1].Outcome != routes.RouteTraceCommitted {
		t.Fatalf("traces=%#v", records)
	}
}

func TestRouteDispatcherRejectsMalformedWebSocketBeforeRuntime(t *testing.T) {
	registry := routes.NewRegistry()
	artifact := routeDispatcherArtifact("stream.websocket.invalid", 'd')
	declaration := routeDispatcherManifestRoute("stream.websocket.invalid.socket", extensionmanifest.RouteActionAdd, "/socket-invalid", "GET")
	declaration.Mode = extensionmanifest.RouteModeWebSocket
	if _, err := registry.Publish(routes.Publication{Plugins: []routes.PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{declaration}}}}); err != nil {
		t.Fatal(err)
	}
	invoker := &streamHTTPTestInvoker{}
	guard := &countingRouteGuard{}
	dispatcher := routes.NewDispatcher(routes.DispatcherConfig{
		Plans: routeRegistryPlanResolver{registry: registry}, Steps: invoker, Guard: guard,
	})
	app := fiber.New()
	app.Use(routeDispatcherMiddleware(dispatcher, nil))
	validUpgrade := func() *stdhttp.Request {
		request := httptest.NewRequest(stdhttp.MethodGet, "/socket-invalid", nil)
		request.Header.Set("Connection", "Upgrade")
		request.Header.Set("Upgrade", "websocket")
		request.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
		request.Header.Set("Sec-WebSocket-Version", "13")
		return request
	}
	tests := []struct {
		name    string
		request func() *stdhttp.Request
	}{
		{name: "plain request", request: func() *stdhttp.Request {
			return httptest.NewRequest(stdhttp.MethodGet, "/socket-invalid", nil)
		}},
		{name: "cross origin", request: func() *stdhttp.Request {
			request := validUpgrade()
			request.Header.Set("Origin", "https://evil.example")
			return request
		}},
		{name: "version substring", request: func() *stdhttp.Request {
			request := validUpgrade()
			request.Header.Set("Sec-WebSocket-Version", "113")
			return request
		}},
		{name: "version list", request: func() *stdhttp.Request {
			request := validUpgrade()
			request.Header.Set("Sec-WebSocket-Version", "13, 12")
			return request
		}},
		{name: "duplicate version", request: func() *stdhttp.Request {
			request := validUpgrade()
			request.Header.Add("Sec-WebSocket-Version", "13")
			return request
		}},
		{name: "malformed key", request: func() *stdhttp.Request {
			request := validUpgrade()
			request.Header.Set("Sec-WebSocket-Key", "not-base64")
			return request
		}},
		{name: "key list", request: func() *stdhttp.Request {
			request := validUpgrade()
			request.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==, dGhlIHNhbXBsZSBub25jZQ==")
			return request
		}},
		{name: "wrong nonce length", request: func() *stdhttp.Request {
			request := validUpgrade()
			request.Header.Set("Sec-WebSocket-Key", "YWJj")
			return request
		}},
		{name: "duplicate key", request: func() *stdhttp.Request {
			request := validUpgrade()
			request.Header.Add("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
			return request
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := app.Test(test.request())
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if response.StatusCode != fiber.StatusUpgradeRequired || invoker.openCalls.Load() != 0 || guard.calls.Load() != 0 {
				t.Fatalf("status=%d openCalls=%d guardCalls=%d", response.StatusCode, invoker.openCalls.Load(), guard.calls.Load())
			}
		})
	}
}

type countingRouteGuard struct {
	calls        atomic.Int32
	inner        routes.CoreGuardAuthorizer
	contextKey   any
	contextValue any
}

type routeWebSocketRequestContextKey struct{}

func (g *countingRouteGuard) validateContext(ctx context.Context) error {
	if g.contextKey != nil && ctx.Value(g.contextKey) != g.contextValue {
		return errors.New("route guard lost the request context")
	}
	return nil
}

func (g *countingRouteGuard) Authorize(
	ctx context.Context,
	plan routes.RouteExecutionPlan,
	step routes.RouteExecutionStep,
	request routes.DispatchRequest,
) error {
	g.calls.Add(1)
	if err := g.validateContext(ctx); err != nil {
		return err
	}
	return g.inner.Authorize(ctx, plan, step, request)
}

func (g *countingRouteGuard) AuthorizeRoute(
	ctx context.Context,
	plan routes.RouteExecutionPlan,
	stepIndex int,
	step routes.RouteExecutionStep,
	request routes.DispatchRequest,
) (routes.RouteGuardAuthorization, error) {
	g.calls.Add(1)
	if err := g.validateContext(ctx); err != nil {
		return routes.RouteGuardAuthorization{}, err
	}
	return g.inner.AuthorizeRoute(ctx, plan, stepIndex, step, request)
}

type streamHTTPTestInvoker struct {
	start     routes.RouteStreamStart
	openCalls atomic.Int64
}

func (*streamHTTPTestInvoker) SupportsMode(string) bool { return false }

func (*streamHTTPTestInvoker) Invoke(context.Context, routes.RouteInvocation) (routes.RouteInvocationResult, error) {
	return routes.RouteInvocationResult{}, errors.New("buffered invocation is not expected")
}

func (i *streamHTTPTestInvoker) OpenStream(context.Context, routes.RouteInvocation) (routes.RouteStreamStart, error) {
	i.openCalls.Add(1)
	return i.start, nil
}

type streamHTTPTestSession struct {
	chunks          []routes.RouteStreamChunk
	response        routes.DispatchResponse
	recvErr         error
	requestChunks   []int
	requestBytes    int
	maxRequestChunk int
	requestClosed   bool
	cancelled       bool
	terminal        bool
}

func (s *streamHTTPTestSession) Send(data []byte, _ bool) error {
	s.requestChunks = append(s.requestChunks, len(data))
	s.requestBytes += len(data)
	if len(data) > s.maxRequestChunk {
		s.maxRequestChunk = len(data)
	}
	return nil
}

func (s *streamHTTPTestSession) CloseRequest() error {
	s.requestClosed = true
	return nil
}

func (s *streamHTTPTestSession) Recv() (routes.RouteStreamChunk, error) {
	if s.recvErr != nil {
		return routes.RouteStreamChunk{}, s.recvErr
	}
	if len(s.chunks) == 0 {
		s.terminal = true
		return routes.RouteStreamChunk{}, io.EOF
	}
	chunk := s.chunks[0]
	s.chunks = s.chunks[1:]
	return chunk, nil
}

func (s *streamHTTPTestSession) Response() (routes.DispatchResponse, bool) {
	return s.response, s.terminal
}

func (s *streamHTTPTestSession) Cancel() { s.cancelled = true }

var _ routes.StepInvoker = (*streamHTTPTestInvoker)(nil)
var _ routes.StreamingStepInvoker = (*streamHTTPTestInvoker)(nil)
var _ routes.RouteStreamSession = (*streamHTTPTestSession)(nil)

type webSocketEchoSession struct {
	messages  chan []byte
	done      chan struct{}
	doneOnce  sync.Once
	received  []byte
	terminal  atomic.Bool
	responded atomic.Bool
}

func newWebSocketEchoSession() *webSocketEchoSession {
	return &webSocketEchoSession{messages: make(chan []byte, 1), done: make(chan struct{})}
}

func (s *webSocketEchoSession) Send(data []byte, _ bool) error {
	s.received = append([]byte(nil), data...)
	s.messages <- append([]byte(nil), data...)
	return nil
}

func (*webSocketEchoSession) CloseRequest() error { return nil }

func (s *webSocketEchoSession) Recv() (routes.RouteStreamChunk, error) {
	if s.responded.CompareAndSwap(false, true) {
		message := <-s.messages
		return routes.RouteStreamChunk{Sequence: 1, Data: message}, nil
	}
	s.terminal.Store(true)
	return routes.RouteStreamChunk{}, io.EOF
}

func (s *webSocketEchoSession) Response() (routes.DispatchResponse, bool) {
	return routes.DispatchResponse{Status: fiber.StatusSwitchingProtocols}, s.terminal.Load()
}

func (s *webSocketEchoSession) Cancel() {
	s.doneOnce.Do(func() { close(s.done) })
}

var _ routes.RouteStreamSession = (*webSocketEchoSession)(nil)

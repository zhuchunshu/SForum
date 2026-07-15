package http

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

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

type streamHTTPTestInvoker struct{ start routes.RouteStreamStart }

func (*streamHTTPTestInvoker) SupportsMode(string) bool { return false }

func (*streamHTTPTestInvoker) Invoke(context.Context, routes.RouteInvocation) (routes.RouteInvocationResult, error) {
	return routes.RouteInvocationResult{}, errors.New("buffered invocation is not expected")
}

func (i *streamHTTPTestInvoker) OpenStream(context.Context, routes.RouteInvocation) (routes.RouteStreamStart, error) {
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

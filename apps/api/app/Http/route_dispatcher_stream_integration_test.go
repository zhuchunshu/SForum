package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	stdhttp "net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
	pluginv2sdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
	pluginwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
)

const routeStreamE2EHelperEnv = "route-stream-http-e2e"

func TestRouteStreamAcrossFiberManagerAndRealProtocolV2Process(t *testing.T) {
	extension := routeStreamE2EExtension(t)
	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Trust: routeStreamE2ETrust{identity: extensions.RuntimeTrustIdentity{TrustGrantID: "stream-grant", ImpactDigest: "stream-impact"}},
	})
	manager := extensionsruntime.NewManager(extensionsruntime.ManagerConfig{Starter: starter})
	if err := manager.Start(context.Background(), extension); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Stop(context.Background(), extension) })
	runtime, err := manager.ActiveRuntimeInstance(extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	registry := routes.NewRegistry()
	if _, err := registry.Publish(routes.Publication{Plugins: []routes.PluginRouteSet{{
		Artifact: routes.PluginArtifact{
			ExtensionID: extension.ID, ExtensionVersion: extension.Version, PackageDigest: extension.PackageDigest,
			RuntimeInstanceID: runtime.Identity.InstanceID,
		},
		Routes: extension.Manifest.Routes,
	}}}); err != nil {
		t.Fatal(err)
	}
	traces := routes.NewRouteTraceRing(32)
	dispatcher := routes.NewDispatcher(routes.DispatcherConfig{
		Plans: routeRegistryPlanResolver{registry: registry}, Steps: NewBufferedRouteStepInvoker(manager),
		Guard: HostRouteGuardAuthorizer{}, Trace: traces,
	})
	app := fiber.New(fiber.Config{StreamRequestBody: true, DisablePreParseMultipartForm: true})
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
	baseHTTP := "http://" + listener.Addr().String()

	t.Run("multipart bounded upload", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("upload", "payload.bin")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(bytes.Repeat([]byte("m"), extensionsruntime.MaxProtocolV2RouteChunkSize+257)); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		request, err := stdhttp.NewRequest(stdhttp.MethodPost, baseHTTP+"/upload", bytes.NewReader(body.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", writer.FormDataContentType())
		response, err := (&stdhttp.Client{Timeout: 10 * time.Second}).Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		payload, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		var received, maximum int
		if _, err := fmt.Sscanf(string(payload), "bytes=%d max=%d", &received, &maximum); err != nil {
			t.Fatalf("decode stream evidence %q: %v", payload, err)
		}
		if response.StatusCode != stdhttp.StatusCreated || received != body.Len() || maximum <= 0 ||
			maximum > extensionsruntime.MaxProtocolV2RouteChunkSize {
			t.Fatalf("status=%d bytes=%d/%d max=%d", response.StatusCode, received, body.Len(), maximum)
		}
	})

	t.Run("SSE", func(t *testing.T) {
		response, err := (&stdhttp.Client{Timeout: 10 * time.Second}).Get(baseHTTP + "/events")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		payload, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != stdhttp.StatusOK || response.Header.Get("Content-Type") != "text/event-stream" ||
			string(payload) != "data: one\n\ndata: two\n\n" {
			t.Fatalf("status=%d headers=%v body=%q", response.StatusCode, response.Header, payload)
		}
	})

	t.Run("WebSocket", func(t *testing.T) {
		dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second, Subprotocols: []string{"sforum.stream.v1"}}
		headers := make(stdhttp.Header)
		headers.Set("Cookie", "session=websocket-secret")
		headers.Set("Authorization", "Bearer websocket-secret")
		headers.Set("X-API-Key", "websocket-api-key")
		headers.Set("X-Auth-Token", "websocket-auth-token")
		connection, response, err := dialer.Dial("ws://"+listener.Addr().String()+"/socket", headers)
		if err != nil {
			t.Fatalf("dial response=%v err=%v", response, err)
		}
		defer connection.Close()
		if err := connection.WriteMessage(websocket.TextMessage, []byte("real-process")); err != nil {
			t.Fatal(err)
		}
		messageType, payload, err := connection.ReadMessage()
		if err != nil || messageType != websocket.BinaryMessage || string(payload) != "real-process" {
			t.Fatalf("messageType=%d payload=%q err=%v", messageType, payload, err)
		}
	})

	t.Run("disconnect cancels exact runtime admission", func(t *testing.T) {
		response, err := (&stdhttp.Client{}).Get(baseHTTP + "/disconnect")
		if err != nil {
			t.Fatal(err)
		}
		buffer := make([]byte, 64)
		if _, err := io.ReadFull(response.Body, buffer); err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		deadline := time.Now().Add(5 * time.Second)
		for {
			snapshot, inspectErr := manager.InspectRuntimeInstance(runtime.Identity)
			if inspectErr != nil {
				t.Fatal(inspectErr)
			}
			if snapshot.Admission.ActiveTotal == 0 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("stream admission did not drain: %#v", snapshot.Admission)
			}
			time.Sleep(10 * time.Millisecond)
		}
	})

	if records := traces.RouteTraces(32); len(records) < 8 {
		t.Fatalf("real stream traces=%#v", records)
	}
}

func TestRouteStreamHTTPHelperProcess(t *testing.T) {
	if os.Getenv("SFORUM_PLUGIN_HELPER") != routeStreamE2EHelperEnv {
		return
	}
	server := pluginv2sdk.NewServer().
		WithFeatures(&protocolwire.ProtocolFeature{Name: "stream.routes", Version: "1"}).
		WithRuntimeStreams(pluginv2sdk.RuntimeStreams{Route: routeStreamE2EHandler})
	pluginv2sdk.Serve(&routeStreamE2EServer{Server: server})
	os.Exit(0)
}

type routeStreamE2EServer struct{ *pluginv2sdk.Server }

func (s *routeStreamE2EServer) InvokeRoute(_ context.Context, request *pluginwire.RouteRequest) (*pluginwire.RouteResponse, error) {
	status := uint32(stdhttp.StatusOK)
	headers := []*protocolwire.Header{}
	switch request.GetRouteId() {
	case "runtime.stream.upload":
		status = stdhttp.StatusCreated
	case "runtime.stream.events", "runtime.stream.disconnect":
		headers = append(headers, &protocolwire.Header{Name: "Content-Type", Values: []string{"text/event-stream"}})
	case "runtime.stream.websocket":
		if header := routeStreamE2EForwardedCredential(request.GetHeaders()); header != "" {
			return nil, fmt.Errorf("filtered WebSocket preflight forwarded credential %s", header)
		}
		status = stdhttp.StatusSwitchingProtocols
		headers = append(headers, &protocolwire.Header{Name: "Sec-WebSocket-Protocol", Values: []string{"sforum.stream.v1"}})
	default:
		return &pluginwire.RouteResponse{
			Context: routeStreamE2EResponseContext(request.GetContext()),
			Error:   &protocolwire.ErrorDetail{Code: protocolwire.ErrorCode_ERROR_CODE_NOT_FOUND, Reason: "route.not_found"},
		}, nil
	}
	return &pluginwire.RouteResponse{
		Context: routeStreamE2EResponseContext(request.GetContext()), StatusCode: status,
		Headers: headers, StreamFollows: true,
	}, nil
}

func routeStreamE2EHandler(stream *pluginv2sdk.RouteStream) error {
	switch stream.Open().GetRouteId() {
	case "runtime.stream.upload":
		total, maximum := 0, 0
		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			total += len(chunk.GetData())
			if len(chunk.GetData()) > maximum {
				maximum = len(chunk.GetData())
			}
		}
		payload := []byte("bytes=" + strconv.Itoa(total) + " max=" + strconv.Itoa(maximum))
		if err := stream.Send(&protocolwire.DataChunk{Sequence: 1, Data: payload, Final: true}); err != nil {
			return err
		}
		return stream.Close(&pluginwire.RouteStreamClose{StatusCode: stdhttp.StatusCreated})
	case "runtime.stream.events":
		if _, err := stream.Recv(); err != io.EOF {
			return err
		}
		for index, payload := range [][]byte{[]byte("data: one\n\n"), []byte("data: two\n\n")} {
			if err := stream.Send(&protocolwire.DataChunk{Sequence: uint64(index + 1), Data: payload, Final: index == 1}); err != nil {
				return err
			}
		}
		return stream.Close(&pluginwire.RouteStreamClose{StatusCode: stdhttp.StatusOK})
	case "runtime.stream.websocket":
		if header := routeStreamE2EForwardedCredential(stream.Open().GetHeaders()); header != "" {
			return fmt.Errorf("filtered WebSocket open forwarded credential %s", header)
		}
		chunk, err := stream.Recv()
		if err != nil {
			return err
		}
		if err := stream.Send(&protocolwire.DataChunk{Sequence: 1, Data: chunk.GetData()}); err != nil {
			return err
		}
		return stream.Close(&pluginwire.RouteStreamClose{StatusCode: stdhttp.StatusSwitchingProtocols})
	case "runtime.stream.disconnect":
		if _, err := stream.Recv(); err != io.EOF {
			return err
		}
		payload := bytes.Repeat([]byte("d"), 32<<10)
		for sequence := uint64(1); ; sequence++ {
			if err := stream.Send(&protocolwire.DataChunk{Sequence: sequence, Data: payload}); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unknown stream route")
	}
}

func routeStreamE2EForwardedCredential(headers []*protocolwire.Header) string {
	for _, header := range headers {
		switch strings.ToLower(strings.TrimSpace(header.GetName())) {
		case "cookie", "authorization", "x-api-key", "x-auth-token":
			return header.GetName()
		}
	}
	return ""
}

func routeStreamE2EResponseContext(request *protocolwire.RequestContext) *protocolwire.ResponseContext {
	return &protocolwire.ResponseContext{
		RequestId: request.GetRequestId(), Trace: proto.Clone(request.GetTrace()).(*protocolwire.TraceContext),
		Extension: proto.Clone(request.GetExtension()).(*protocolwire.ExtensionIdentity), ServerTime: timestamppb.Now(),
	}
}

func routeStreamE2EExtension(t *testing.T) extensions.Extension {
	t.Helper()
	packageRoot := filepath.Join(t.TempDir(), "runtime.stream", "1.0.0")
	if err := os.MkdirAll(filepath.Join(packageRoot, "backend"), 0o755); err != nil {
		t.Fatal(err)
	}
	launcher := "#!/bin/sh\nSFORUM_PLUGIN_HELPER=" + routeStreamE2EHelperEnv + " exec " +
		routeStreamShellQuote(os.Args[0]) + " -test.run=TestRouteStreamHTTPHelperProcess -- \"$@\"\n"
	if err := os.WriteFile(filepath.Join(packageRoot, "backend", "plugin"), []byte(launcher), 0o755); err != nil {
		t.Fatal(err)
	}
	route := func(id, path, method, mode string) extensions.ManifestRoute {
		value := extensions.ManifestRoute{
			ID: id, ContractVersion: id + "@1", Action: extensionmanifest.RouteActionAdd,
			Path: path, Methods: []string{method}, Guard: extensionmanifest.GuardCorePublic,
			Fallback: "closed", Mode: mode, Handler: "route.stream", ResponseSchema: id + ".response@1",
		}
		if method != stdhttp.MethodGet {
			value.RequestSchema = id + ".request@1"
		}
		return value
	}
	return extensions.Extension{
		ID: "runtime.stream", Name: "Runtime Stream", Version: "1.0.0", Type: extensions.TypePlugin,
		Status: extensions.StatusEnabled, Source: extensions.SourceUploaded,
		PackageDigest: strings.Repeat("a", 64), PackagePath: packageRoot,
		Manifest: extensions.Manifest{
			ManifestVersion: 3, ID: "runtime.stream", Version: "1.0.0", Type: extensions.TypePlugin,
			Backend: extensions.ManifestBackend{
				Entry: "backend/plugin", RPC: "hashicorp-go-plugin", ProtocolVersion: 2, HostAPIVersion: "sforum.host@2",
			},
			Routes: []extensions.ManifestRoute{
				route("runtime.stream.upload", "/upload", stdhttp.MethodPost, extensionmanifest.RouteModeMultipart),
				route("runtime.stream.events", "/events", stdhttp.MethodGet, extensionmanifest.RouteModeSSE),
				route("runtime.stream.websocket", "/socket", stdhttp.MethodGet, extensionmanifest.RouteModeWebSocket),
				route("runtime.stream.disconnect", "/disconnect", stdhttp.MethodGet, extensionmanifest.RouteModeSSE),
			},
		},
	}
}

func routeStreamShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

type routeStreamE2ETrust struct {
	identity extensions.RuntimeTrustIdentity
}

func (s routeStreamE2ETrust) RuntimeIdentity(context.Context, extensions.Extension) (extensions.RuntimeTrustIdentity, error) {
	return s.identity, nil
}

var _ pluginwire.PluginRuntimeServiceServer = (*routeStreamE2EServer)(nil)

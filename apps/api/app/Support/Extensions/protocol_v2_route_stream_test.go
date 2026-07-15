package extensionsruntime

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"net"
	"net/http"
	"reflect"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	pluginwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestProtocolV2RouteStreamCarriesExactContextAndBoundedChunks(t *testing.T) {
	var receivedOpen *pluginwire.RouteStreamOpen
	client := newProtocolV2RouteStreamTestClient(t, func(stream grpc.BidiStreamingServer[pluginwire.RouteStreamFrame, pluginwire.RouteStreamFrame]) error {
		first, err := stream.Recv()
		if err != nil {
			return err
		}
		receivedOpen = first.GetOpen()
		request, err := stream.Recv()
		if err != nil || request.GetChunk().GetSequence() != 1 || string(request.GetChunk().GetData()) != "request" || !request.GetChunk().GetFinal() {
			return ErrProtocolV2RouteStreamInvalid
		}
		if _, err := stream.Recv(); err != nil {
			return err
		}
		digest := sha256.Sum256([]byte("response"))
		if err := stream.Send(&pluginwire.RouteStreamFrame{Frame: &pluginwire.RouteStreamFrame_Chunk{Chunk: &protocolwire.DataChunk{
			Sequence: 1, Data: []byte("response"), Checksum: digest[:], Final: true,
		}}}); err != nil {
			return err
		}
		return stream.Send(&pluginwire.RouteStreamFrame{Frame: &pluginwire.RouteStreamFrame_Close{Close: &pluginwire.RouteStreamClose{
			StatusCode: http.StatusCreated, Headers: []*protocolwire.Header{{Name: "X-Stream", Values: []string{"done"}}},
		}}})
	})
	stream, err := client.OpenRouteStreamContext(context.Background(), ProtocolV2RouteStreamRequest{
		RouteID: "demo.stream", ContractVersion: "demo.stream@1", Method: http.MethodPost,
		Path: "/stream?part=1", Mode: extensionmanifest.RouteModeStream,
		Headers: http.Header{"X-Test": {"one", "two"}},
		Actor:   NewProtocolV2RouteActor(42, true, map[string]bool{"stream.write": true}),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Cancel()
	if err := stream.Send([]byte("request"), true); err != nil {
		t.Fatal(err)
	}
	if err := stream.CloseRequest(); err != nil {
		t.Fatal(err)
	}
	chunk, err := stream.Recv()
	if err != nil || string(chunk.Data) != "response" || !chunk.Final {
		t.Fatalf("chunk=%#v err=%v", chunk, err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("terminal err=%v", err)
	}
	response, ok := stream.Response()
	if !ok || response.StatusCode != http.StatusCreated || response.Headers.Get("X-Stream") != "done" {
		t.Fatalf("response=%#v ok=%t", response, ok)
	}
	if receivedOpen.GetRouteId() != "demo.stream" || receivedOpen.GetContractVersion() != "demo.stream@1" ||
		receivedOpen.GetPath() != "/stream?part=1" || receivedOpen.GetContext().GetActor().GetUserId() != 42 ||
		!reflect.DeepEqual(receivedOpen.GetContext().GetActor().GetPermissionKeys(), []string{"stream.write"}) ||
		len(receivedOpen.GetHeaders()) != 1 {
		t.Fatalf("open=%#v", receivedOpen)
	}
	remaining := time.Until(receivedOpen.GetContext().GetDeadline().AsTime())
	if remaining < 23*time.Hour || remaining > DefaultProtocolV2RouteStreamTimeout {
		t.Fatalf("default stream deadline remaining=%s", remaining)
	}
}

func TestProtocolV2RouteStreamRejectsDriftAndMalformedPeerFrames(t *testing.T) {
	client := newProtocolV2RouteStreamTestClient(t, func(stream grpc.BidiStreamingServer[pluginwire.RouteStreamFrame, pluginwire.RouteStreamFrame]) error {
		if _, err := stream.Recv(); err != nil {
			return err
		}
		return stream.Send(&pluginwire.RouteStreamFrame{Frame: &pluginwire.RouteStreamFrame_Chunk{Chunk: &protocolwire.DataChunk{
			Sequence: 2, Data: []byte("out-of-order"), Final: true,
		}}})
	})
	base := ProtocolV2RouteStreamRequest{
		RouteID: "demo.stream", ContractVersion: "demo.stream@1", Method: http.MethodPost,
		Path: "/stream", Mode: extensionmanifest.RouteModeStream, Timeout: time.Second,
	}
	for name, mutate := range map[string]func(*ProtocolV2RouteStreamRequest){
		"route":    func(value *ProtocolV2RouteStreamRequest) { value.RouteID = "other" },
		"contract": func(value *ProtocolV2RouteStreamRequest) { value.ContractVersion = "demo.stream@2" },
		"mode":     func(value *ProtocolV2RouteStreamRequest) { value.Mode = extensionmanifest.RouteModeSSE },
		"method":   func(value *ProtocolV2RouteStreamRequest) { value.Method = http.MethodGet },
	} {
		t.Run(name, func(t *testing.T) {
			request := base
			mutate(&request)
			if _, err := client.OpenRouteStreamContext(context.Background(), request); !errors.Is(err, ErrProtocolV2RouteStreamInvalid) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	stream, err := client.OpenRouteStreamContext(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Cancel()
	if _, err := stream.Recv(); !errors.Is(err, ErrProtocolV2RouteStreamInvalid) {
		t.Fatalf("malformed chunk error=%v", err)
	}
}

func TestProtocolV2RouteStreamPropagatesCancellationAndBoundsRequests(t *testing.T) {
	accepted := make(chan struct{})
	client := newProtocolV2RouteStreamTestClient(t, func(stream grpc.BidiStreamingServer[pluginwire.RouteStreamFrame, pluginwire.RouteStreamFrame]) error {
		if _, err := stream.Recv(); err != nil {
			return err
		}
		close(accepted)
		<-stream.Context().Done()
		return stream.Context().Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	stream, err := client.OpenRouteStreamContext(ctx, ProtocolV2RouteStreamRequest{
		RouteID: "demo.stream", ContractVersion: "demo.stream@1", Method: http.MethodPost,
		Path: "/stream", Mode: extensionmanifest.RouteModeStream, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Send(make([]byte, MaxProtocolV2RouteChunkSize+1), false); !errors.Is(err, ErrProtocolV2RouteStreamInvalid) {
		t.Fatalf("oversized request error=%v", err)
	}
	<-accepted
	cancel()
	if _, err := stream.Recv(); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error=%v", err)
	}
}

type protocolV2RouteStreamTestServer struct {
	pluginwire.UnimplementedPluginRuntimeServiceServer
	stream func(grpc.BidiStreamingServer[pluginwire.RouteStreamFrame, pluginwire.RouteStreamFrame]) error
}

func (s *protocolV2RouteStreamTestServer) StreamRoute(stream grpc.BidiStreamingServer[pluginwire.RouteStreamFrame, pluginwire.RouteStreamFrame]) error {
	return s.stream(stream)
}

func newProtocolV2RouteStreamTestClient(
	t *testing.T,
	stream func(grpc.BidiStreamingServer[pluginwire.RouteStreamFrame, pluginwire.RouteStreamFrame]) error,
) *protocolV2Client {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	pluginwire.RegisterPluginRuntimeServiceServer(server, &protocolV2RouteStreamTestServer{stream: stream})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close() })
	connection, err := grpc.NewClient("passthrough:///route-stream-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	return newProtocolV2Client(pluginwire.NewPluginRuntimeServiceClient(connection), protocolV2ClientConfig{
		identity: &protocolwire.ExtensionIdentity{
			ExtensionId: "demo.plugin", ExtensionVersion: "1.0.0", ArtifactDigest: "digest-v1",
			TrustGrantId: "grant-1", RuntimeEpoch: 1, InstanceId: "runtime-1",
		},
		instance: "runtime-1",
		routes: []extensions.ManifestRoute{{
			ID: "demo.stream", ContractVersion: "demo.stream@1", Methods: []string{http.MethodPost}, Mode: extensionmanifest.RouteModeStream,
		}},
	})
}

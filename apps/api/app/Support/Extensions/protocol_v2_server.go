package extensionsruntime

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-plugin"
	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	pluginProtocolV2Name              = "sforum-plugin-v2"
	DefaultProtocolV2MaxMessageBytes  = 4 << 20
	DefaultProtocolV2ConcurrentCalls  = 16
	DefaultProtocolV2RequestTimeout   = 5 * time.Second
	DefaultProtocolV2HandshakeTimeout = 5 * time.Second
)

// ProtocolV2ServerConfig controls process-local transport safety limits.
type ProtocolV2ServerConfig struct {
	MaxMessageBytes int
	MaxConcurrent   int
	DefaultTimeout  time.Duration
}

// ServeProtocolV2Plugin serves only HashiCorp go-plugin protocol 2 over gRPC.
// Protocol v1 remains available through ServeProtocolPlugin.
func ServeProtocolV2Plugin(server pluginv2.PluginRuntimeServiceServer, config ProtocolV2ServerConfig) {
	if server == nil {
		panic("protocol v2 plugin server is required")
	}
	config = normalizeProtocolV2ServerConfig(config)
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		VersionedPlugins: map[int]plugin.PluginSet{
			2: {pluginProtocolV2Name: &protocolV2Plugin{server: server}},
		},
		GRPCServer: protocolV2GRPCServerFactory(config),
	})
}

type protocolV2Plugin struct {
	plugin.NetRPCUnsupportedPlugin
	server       pluginv2.PluginRuntimeServiceServer
	clientConfig *protocolV2ClientConfig
}

func (p *protocolV2Plugin) GRPCServer(_ *plugin.GRPCBroker, server *grpc.Server) error {
	if p.server == nil {
		return fmt.Errorf("protocol v2 plugin server is required")
	}
	pluginv2.RegisterPluginRuntimeServiceServer(server, p.server)
	return nil
}

func (p *protocolV2Plugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	client := pluginv2.NewPluginRuntimeServiceClient(conn)
	if p.clientConfig == nil {
		return client, nil
	}
	return newProtocolV2Client(client, *p.clientConfig), nil
}

func normalizeProtocolV2ServerConfig(config ProtocolV2ServerConfig) ProtocolV2ServerConfig {
	if config.MaxMessageBytes <= 0 || config.MaxMessageBytes > DefaultProtocolV2MaxMessageBytes {
		config.MaxMessageBytes = DefaultProtocolV2MaxMessageBytes
	}
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = DefaultProtocolV2ConcurrentCalls
	}
	if config.DefaultTimeout <= 0 {
		config.DefaultTimeout = DefaultProtocolV2RequestTimeout
	}
	return config
}

func protocolV2GRPCServerFactory(config ProtocolV2ServerConfig) func([]grpc.ServerOption) *grpc.Server {
	return func(options []grpc.ServerOption) *grpc.Server {
		semaphore := make(chan struct{}, config.MaxConcurrent)
		options = append(options,
			grpc.MaxRecvMsgSize(config.MaxMessageBytes),
			grpc.MaxSendMsgSize(config.MaxMessageBytes),
			grpc.ChainUnaryInterceptor(protocolV2UnaryInterceptor(config.DefaultTimeout, semaphore)),
			grpc.ChainStreamInterceptor(protocolV2StreamInterceptor(config.DefaultTimeout, semaphore)),
		)
		return grpc.NewServer(options...)
	}
}

func protocolV2UnaryInterceptor(timeout time.Duration, semaphore chan struct{}) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		callCtx, cancel := withDefaultDeadline(ctx, timeout)
		defer cancel()
		if err := acquireProtocolV2(callCtx, semaphore); err != nil {
			return nil, err
		}
		defer func() { <-semaphore }()
		return handler(callCtx, request)
	}
}

func protocolV2StreamInterceptor(timeout time.Duration, semaphore chan struct{}) grpc.StreamServerInterceptor {
	return func(server any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, cancel := withDefaultDeadline(stream.Context(), timeout)
		defer cancel()
		if err := acquireProtocolV2(ctx, semaphore); err != nil {
			return err
		}
		defer func() { <-semaphore }()
		return handler(server, &protocolV2ServerStream{ServerStream: stream, ctx: ctx})
	}
}

type protocolV2ServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *protocolV2ServerStream) Context() context.Context { return s.ctx }

func withDefaultDeadline(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok || timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

func acquireProtocolV2(ctx context.Context, semaphore chan struct{}) error {
	select {
	case semaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return status.Error(codes.ResourceExhausted, "protocol v2 concurrency limit reached before deadline")
	}
}

package pluginv2

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-plugin"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	hostwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/host/v2"
	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var ErrHostUnavailable = errors.New("protocol v2 host broker is unavailable")

// Host exposes the generated Host API v2 clients on the runtime-scoped broker.
// Plugins should build every request context with RequestContext so the exact
// artifact identity and granted authority cannot drift from the handshake.
type Host struct {
	Queries     hostwire.HostQueryServiceClient
	Commands    hostwire.HostCommandServiceClient
	Database    hostwire.DatabaseServiceClient
	Cache       hostwire.CacheServiceClient
	Jobs        hostwire.JobServiceClient
	Schedules   hostwire.ScheduleServiceClient
	Services    hostwire.ServiceDiscoveryServiceClient
	Secrets     hostwire.SecretServiceClient
	Files       hostwire.FileServiceClient
	HTTP        hostwire.HttpServiceClient
	Admin       hostwire.AdminSurfaceServiceClient
	Identity    hostwire.IdentityServiceClient
	Permissions hostwire.PermissionServiceClient
	Media       hostwire.MediaServiceClient
	Navigation  hostwire.NavigationServiceClient
	Audit       hostwire.AuditServiceClient
	Tracing     hostwire.TracingServiceClient

	conn      *grpc.ClientConn
	identity  *protocolwire.ExtensionIdentity
	authority []*protocolwire.AuthorityGrant
	instance  string
	sequence  atomic.Uint64
}

func newHost(
	broker *plugin.GRPCBroker,
	brokerID uint32,
	token []byte,
	identity *protocolwire.ExtensionIdentity,
	authority []*protocolwire.AuthorityGrant,
) (*Host, error) {
	if broker == nil || brokerID == 0 {
		return nil, ErrHostUnavailable
	}
	conn, err := broker.DialWithOptions(brokerID,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(extensionsruntime.DefaultProtocolV2MaxMessageBytes),
			grpc.MaxCallSendMsgSize(extensionsruntime.DefaultProtocolV2MaxMessageBytes),
		),
		grpc.WithChainUnaryInterceptor(hostUnaryClientInterceptor(token)),
		grpc.WithChainStreamInterceptor(hostStreamClientInterceptor(token)),
	)
	if err != nil {
		return nil, err
	}
	host := &Host{
		conn: conn, identity: cloneIdentity(identity), authority: cloneAuthority(authority),
		instance: identity.GetInstanceId(),
	}
	host.Queries = hostwire.NewHostQueryServiceClient(conn)
	host.Commands = hostwire.NewHostCommandServiceClient(conn)
	host.Database = hostwire.NewDatabaseServiceClient(conn)
	host.Cache = hostwire.NewCacheServiceClient(conn)
	host.Jobs = hostwire.NewJobServiceClient(conn)
	host.Schedules = hostwire.NewScheduleServiceClient(conn)
	host.Services = hostwire.NewServiceDiscoveryServiceClient(conn)
	host.Secrets = hostwire.NewSecretServiceClient(conn)
	host.Files = hostwire.NewFileServiceClient(conn)
	host.HTTP = hostwire.NewHttpServiceClient(conn)
	host.Admin = hostwire.NewAdminSurfaceServiceClient(conn)
	host.Identity = hostwire.NewIdentityServiceClient(conn)
	host.Permissions = hostwire.NewPermissionServiceClient(conn)
	host.Media = hostwire.NewMediaServiceClient(conn)
	host.Navigation = hostwire.NewNavigationServiceClient(conn)
	host.Audit = hostwire.NewAuditServiceClient(conn)
	host.Tracing = hostwire.NewTracingServiceClient(conn)
	return host, nil
}

// RequestContext carries actor/locale/trace data from an incoming plugin call
// while replacing all runtime-owned identity and authority fields.
func (h *Host) RequestContext(parent *protocolwire.RequestContext) *protocolwire.RequestContext {
	result := &protocolwire.RequestContext{}
	if parent != nil {
		result = proto.Clone(parent).(*protocolwire.RequestContext)
	}
	result.RequestId = h.instance + "-host-" + strconv.FormatUint(h.sequence.Add(1), 10)
	result.Extension = cloneIdentity(h.identity)
	result.GrantedAuthority = cloneAuthority(h.authority)
	if result.Locale == "" {
		result.Locale = "und"
	}
	maximum := time.Now().UTC().Add(extensionsruntime.DefaultProtocolV2RequestTimeout)
	if result.Deadline == nil || !result.Deadline.IsValid() || result.Deadline.AsTime().After(maximum) {
		result.Deadline = timestamppb.New(maximum)
	}
	return result
}

func hostUnaryClientInterceptor(token []byte) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, request, reply any, conn *grpc.ClientConn, invoker grpc.UnaryInvoker, options ...grpc.CallOption) error {
		callCtx, cancel := hostCallContext(ctx, token)
		defer cancel()
		return invoker(callCtx, method, request, reply, conn, options...)
	}
}

func hostStreamClientInterceptor(token []byte) grpc.StreamClientInterceptor {
	return func(ctx context.Context, descriptor *grpc.StreamDesc, conn *grpc.ClientConn, method string, streamer grpc.Streamer, options ...grpc.CallOption) (grpc.ClientStream, error) {
		callCtx, cancel := hostCallContext(ctx, token)
		stream, err := streamer(callCtx, descriptor, conn, method, options...)
		if err != nil {
			cancel()
			return nil, err
		}
		return &hostClientStream{ClientStream: stream, cancel: cancel}, nil
	}
}

type hostClientStream struct {
	grpc.ClientStream
	cancel context.CancelFunc
}

func (s *hostClientStream) CloseSend() error {
	err := s.ClientStream.CloseSend()
	s.cancel()
	return err
}

func hostCallContext(ctx context.Context, token []byte) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		ctx, cancel := context.WithCancel(ctx)
		return metadata.AppendToOutgoingContext(ctx, extensionsruntime.ProtocolV2RuntimeTokenMetadataKey, string(token)), cancel
	}
	ctx, cancel := context.WithTimeout(ctx, extensionsruntime.DefaultProtocolV2RequestTimeout)
	return metadata.AppendToOutgoingContext(ctx, extensionsruntime.ProtocolV2RuntimeTokenMetadataKey, string(token)), cancel
}

func cloneIdentity(value *protocolwire.ExtensionIdentity) *protocolwire.ExtensionIdentity {
	if value == nil {
		return nil
	}
	return proto.Clone(value).(*protocolwire.ExtensionIdentity)
}

func cloneAuthority(values []*protocolwire.AuthorityGrant) []*protocolwire.AuthorityGrant {
	result := make([]*protocolwire.AuthorityGrant, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, proto.Clone(value).(*protocolwire.AuthorityGrant))
		}
	}
	return result
}

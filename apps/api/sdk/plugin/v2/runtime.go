package pluginv2

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"sync"
	"time"

	"github.com/hashicorp/go-plugin"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	pluginwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const protocolName = "sforum.plugin"

// ServeOptions controls transport limits for a plugin subprocess.
type ServeOptions = extensionsruntime.ProtocolV2ServerConfig

// Server provides strict handshake, health, and readiness defaults. Plugin
// implementations embed it and override only the generated RPCs they own.
type Server struct {
	pluginwire.UnimplementedPluginRuntimeServiceServer

	mu               sync.RWMutex
	started          bool
	identity         *protocolwire.ExtensionIdentity
	tokenHash        [sha256.Size]byte
	features         []*protocolwire.ProtocolFeature
	selectedFeatures []*protocolwire.ProtocolFeature
	services         []*protocolwire.ServiceDescriptor
	serviceRegistry  *ServiceRegistry
	hookRegistry     *HookRegistry
	providerRegistry *ProviderRegistry
	identityRegistry *IdentityProviderRegistry
	seoRegistry      *SEORegistry
	commandRegistry  *CommandRegistry
	jobRegistry      *JobRegistry
	queryHandlers    QueryRuntimeHandlers
	streams          RuntimeStreams
	broker           *plugin.GRPCBroker
	host             *Host
	brokerID         uint32
	now              func() time.Time
}

// BindProtocolV2Broker is called by the transport before the runtime service is
// registered. Plugin implementations normally receive it through embedding.
func (s *Server) BindProtocolV2Broker(broker *plugin.GRPCBroker) {
	s.mu.Lock()
	s.broker = broker
	s.mu.Unlock()
}

// Host returns the runtime-scoped generated Host API clients after handshake.
func (s *Server) Host() (*Host, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.host == nil {
		return nil, ErrHostUnavailable
	}
	return s.host, nil
}

// NewServer constructs a protocol-v2 default server.
func NewServer() *Server {
	return &Server{now: func() time.Time { return time.Now().UTC() }}
}

// WithFeatures declares optional wire features selected during handshake.
func (s *Server) WithFeatures(features ...*protocolwire.ProtocolFeature) *Server {
	s.mu.Lock()
	if !s.started {
		s.features = cloneFeatures(features)
	}
	s.mu.Unlock()
	return s
}

// WithServices declares versioned plugin-to-plugin services for host discovery.
func (s *Server) WithServices(services ...*protocolwire.ServiceDescriptor) *Server {
	s.mu.Lock()
	if !s.started {
		s.services = cloneServices(services)
	}
	s.mu.Unlock()
	return s
}

// WithServiceRegistry publishes validated service declarations during
// handshake and enables the default unary and bidirectional dispatchers.
// A plugin server may still override either generated RPC explicitly.
func (s *Server) WithServiceRegistry(registry *ServiceRegistry) *Server {
	s.mu.Lock()
	if !s.started {
		s.serviceRegistry = registry
	}
	s.mu.Unlock()
	return s
}

// WithHookRegistry enables default InvokeHook dispatch for declared hooks.
func (s *Server) WithHookRegistry(registry *HookRegistry) *Server {
	s.mu.Lock()
	if !s.started {
		s.hookRegistry = registry
	}
	s.mu.Unlock()
	return s
}

// WithProviderRegistry enables default ProviderCall dispatch.
func (s *Server) WithProviderRegistry(registry *ProviderRegistry) *Server {
	s.mu.Lock()
	if !s.started {
		s.providerRegistry = registry
	}
	s.mu.Unlock()
	return s
}

// WithIdentityProviderRegistry enables the reserved sforum.identity dispatcher.
// Calls remain unavailable until identity.runtime@1 is negotiated exactly.
func (s *Server) WithIdentityProviderRegistry(registry *IdentityProviderRegistry) *Server {
	s.mu.Lock()
	if !s.started {
		s.identityRegistry = registry
	}
	s.mu.Unlock()
	return s
}

// WithSEORegistry enables typed SEO dispatch over the existing ProviderCall
// RPC. It does not grant raw head, HTML, request, actor, or session access.
func (s *Server) WithSEORegistry(registry *SEORegistry) *Server {
	s.mu.Lock()
	if !s.started {
		s.seoRegistry = registry
	}
	s.mu.Unlock()
	return s
}

// WithCommandRegistry enables default InvokeCommand dispatch for CLI commands.
func (s *Server) WithCommandRegistry(registry *CommandRegistry) *Server {
	s.mu.Lock()
	if !s.started {
		s.commandRegistry = registry
	}
	s.mu.Unlock()
	return s
}

// WithJobRegistry installs ExecuteJob dispatch for declared job kinds.
// Explicit RuntimeStreams.Job overrides this registry when both are set.
func (s *Server) WithJobRegistry(registry *JobRegistry) *Server {
	s.mu.Lock()
	if !s.started {
		s.jobRegistry = registry
	}
	s.mu.Unlock()
	return s
}

// WithQueryRuntimeHandlers enables the dedicated query.runtime@1 unary
// dispatchers. Either handler may be omitted; that RPC remains Unimplemented
// for compatibility with plugins that do not declare the corresponding family.
func (s *Server) WithQueryRuntimeHandlers(handlers QueryRuntimeHandlers) *Server {
	s.mu.Lock()
	if !s.started {
		s.queryHandlers = handlers
	}
	s.mu.Unlock()
	return s
}

// WithRuntimeStreams installs route, file, lifecycle, and job stream handlers.
// The complete handler snapshot freezes at the first successful handshake.
func (s *Server) WithRuntimeStreams(streams RuntimeStreams) *Server {
	s.mu.Lock()
	if !s.started {
		s.streams = streams
	}
	s.mu.Unlock()
	return s
}

// Serve runs one protocol-v2-only HashiCorp go-plugin subprocess.
func Serve(server pluginwire.PluginRuntimeServiceServer, options ...ServeOptions) {
	config := ServeOptions{}
	if len(options) > 0 {
		config = options[0]
	}
	extensionsruntime.ServeProtocolV2Plugin(server, config)
}

// Handshake binds this process to one exact runtime token and artifact.
func (s *Server) Handshake(_ context.Context, request *protocolwire.HandshakeRequest) (*protocolwire.HandshakeResponse, error) {
	now := s.nowTime()
	response := &protocolwire.HandshakeResponse{Context: responseContext(requestContext(request), now)}
	if err := validateHandshakeRequest(request); err != nil {
		response.Error = err
		return response, nil
	}

	tokenHash := sha256.Sum256(request.GetRuntimeToken())
	identity := request.GetContext().GetExtension()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		if subtle.ConstantTimeCompare(s.tokenHash[:], tokenHash[:]) != 1 || !proto.Equal(s.identity, identity) ||
			s.brokerID != request.GetHostBrokerId() {
			response.Error = protocolError(protocolwire.ErrorCode_ERROR_CODE_STALE_RUNTIME,
				"protocol.stale_runtime", "Runtime token, exact artifact identity, or Host broker changed.")
			return response, nil
		}
	} else {
		if request.GetHostBrokerId() != 0 {
			host, err := newHost(s.broker, request.GetHostBrokerId(), request.GetRuntimeToken(), identity, request.GetContext().GetGrantedAuthority())
			if err != nil {
				response.Error = protocolError(protocolwire.ErrorCode_ERROR_CODE_UNAVAILABLE,
					"protocol.host_broker_unavailable", "Host API broker connection failed.")
				return response, nil
			}
			s.host = host
		}
		s.started = true
		s.tokenHash = tokenHash
		s.identity = proto.Clone(identity).(*protocolwire.ExtensionIdentity)
		s.brokerID = request.GetHostBrokerId()
		s.selectedFeatures = selectFeatures(request.GetHostFeatures(), s.features)
	}
	response.SelectedProtocol = &protocolwire.ProtocolRange{Protocol: protocolName, Major: ProtocolMajor, MinMinor: 0, MaxMinor: 0}
	response.SelectedFeatures = cloneFeatures(s.selectedFeatures)
	response.Services = serviceHandshakeDescriptors(s.services, s.serviceRegistry)
	response.TokenExpiresAt = timestamppb.New(now.Add(10 * time.Minute))
	return response, nil
}

func (s *Server) Health(_ context.Context, request *protocolwire.HealthRequest) (*protocolwire.HealthResponse, error) {
	now := s.nowTime()
	response := &protocolwire.HealthResponse{Context: responseContext(request.GetContext(), now)}
	if err := s.validateRuntimeContext(request.GetContext()); err != nil {
		response.Status = "stale"
		response.Error = err
		return response, nil
	}
	response.Healthy = true
	response.Status = "healthy"
	return response, nil
}

func (s *Server) Readiness(_ context.Context, request *protocolwire.ReadinessRequest) (*protocolwire.ReadinessResponse, error) {
	now := s.nowTime()
	response := &protocolwire.ReadinessResponse{Context: responseContext(request.GetContext(), now)}
	if err := s.validateRuntimeContext(request.GetContext()); err != nil {
		response.Error = err
		return response, nil
	}
	response.Ready = true
	return response, nil
}

func (s *Server) InvokeService(ctx context.Context, request *pluginwire.ServiceRequest) (*pluginwire.ServiceResponse, error) {
	s.mu.RLock()
	registry := s.serviceRegistry
	s.mu.RUnlock()
	if registry == nil {
		return s.UnimplementedPluginRuntimeServiceServer.InvokeService(ctx, request)
	}
	if detail := s.validateRuntimeContext(request.GetContext()); detail != nil {
		return &pluginwire.ServiceResponse{
			Context: responseContext(request.GetContext(), s.nowTime()), Error: detail,
		}, nil
	}
	return registry.InvokeService(ctx, request)
}

func (s *Server) StreamService(stream grpc.BidiStreamingServer[pluginwire.ServiceStreamFrame, pluginwire.ServiceStreamFrame]) error {
	s.mu.RLock()
	registry := s.serviceRegistry
	s.mu.RUnlock()
	if registry == nil {
		return s.UnimplementedPluginRuntimeServiceServer.StreamService(stream)
	}
	return registry.streamService(stream, s.validateRuntimeContext)
}

func (s *Server) InvokeHook(ctx context.Context, request *pluginwire.HookRequest) (*pluginwire.HookResponse, error) {
	s.mu.RLock()
	registry := s.hookRegistry
	s.mu.RUnlock()
	if registry == nil {
		return s.UnimplementedPluginRuntimeServiceServer.InvokeHook(ctx, request)
	}
	if detail := s.validateRuntimeContext(request.GetContext()); detail != nil {
		return &pluginwire.HookResponse{
			Context: responseContext(request.GetContext(), s.nowTime()), Accepted: false, Error: detail,
		}, nil
	}
	return registry.InvokeHook(ctx, request)
}

func (s *Server) ProviderCall(ctx context.Context, request *pluginwire.ProviderCallRequest) (*pluginwire.ProviderCallResponse, error) {
	s.mu.RLock()
	providerRegistry := s.providerRegistry
	identityRegistry := s.identityRegistry
	seoRegistry := s.seoRegistry
	identityNegotiated := hasExactIdentityRuntimeFeature(s.selectedFeatures)
	s.mu.RUnlock()
	// Reserved families are resolved before the public provider namespace. A
	// missing or unnegotiated identity registry must never fall through generic.
	if request.GetSlotId() == IdentityRuntimeProviderSlot {
		if identityRegistry == nil || !identityNegotiated {
			return s.UnimplementedPluginRuntimeServiceServer.ProviderCall(ctx, request)
		}
		if detail := s.validateRuntimeContext(request.GetContext()); detail != nil {
			return &pluginwire.ProviderCallResponse{
				Context: responseContext(request.GetContext(), s.nowTime()), Error: detail,
			}, nil
		}
		return identityRegistry.ProviderCall(ctx, request)
	}
	if detail := s.validateRuntimeContext(request.GetContext()); detail != nil {
		return &pluginwire.ProviderCallResponse{
			Context: responseContext(request.GetContext(), s.nowTime()), Error: detail,
		}, nil
	}
	if request.GetSlotId() == extensionsruntime.ProtocolV2SEOProviderSlot {
		if seoRegistry == nil {
			return s.UnimplementedPluginRuntimeServiceServer.ProviderCall(ctx, request)
		}
		return seoRegistry.ProviderCall(ctx, request)
	}
	if providerRegistry == nil {
		return s.UnimplementedPluginRuntimeServiceServer.ProviderCall(ctx, request)
	}
	return providerRegistry.ProviderCall(ctx, request)
}

func (s *Server) InvokeCommand(ctx context.Context, request *pluginwire.CommandInvocationRequest) (*pluginwire.CommandInvocationResponse, error) {
	s.mu.RLock()
	registry := s.commandRegistry
	s.mu.RUnlock()
	if registry == nil {
		return s.UnimplementedPluginRuntimeServiceServer.InvokeCommand(ctx, request)
	}
	if detail := s.validateRuntimeContext(request.GetContext()); detail != nil {
		return &pluginwire.CommandInvocationResponse{
			Context: responseContext(request.GetContext(), s.nowTime()), Error: detail,
		}, nil
	}
	return registry.InvokeCommand(ctx, request)
}

func (s *Server) validateRuntimeContext(ctx *protocolwire.RequestContext) *protocolwire.ErrorDetail {
	if ctx == nil || ctx.GetExtension() == nil {
		return protocolError(protocolwire.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			"protocol.identity_required", "Exact extension identity is required.")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.started || !proto.Equal(s.identity, ctx.GetExtension()) {
		return protocolError(protocolwire.ErrorCode_ERROR_CODE_STALE_RUNTIME,
			"protocol.stale_runtime", "Runtime identity is stale.")
	}
	return nil
}

func (s *Server) nowTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now().UTC()
}

func validateHandshakeRequest(request *protocolwire.HandshakeRequest) *protocolwire.ErrorDetail {
	if request == nil || request.GetContext() == nil || request.GetContext().GetExtension() == nil {
		return protocolError(protocolwire.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			"protocol.identity_required", "Exact extension identity is required.")
	}
	identity := request.GetContext().GetExtension()
	if identity.GetExtensionId() == "" || identity.GetExtensionVersion() == "" || identity.GetArtifactDigest() == "" ||
		identity.GetTrustGrantId() == "" || identity.GetRuntimeEpoch() == 0 || identity.GetInstanceId() == "" {
		return protocolError(protocolwire.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			"protocol.identity_incomplete", "Extension id, version, digest, trust grant, epoch, and instance are required.")
	}
	if request.GetHostApiVersion() != HostAPIVersion {
		return protocolError(protocolwire.ErrorCode_ERROR_CODE_PROTOCOL_MISMATCH,
			"protocol.host_api_mismatch", "Host API version is not supported.")
	}
	if len(request.GetRuntimeToken()) < 32 {
		return protocolError(protocolwire.ErrorCode_ERROR_CODE_UNAUTHENTICATED,
			"protocol.runtime_token_invalid", "Runtime token is missing or too short.")
	}
	compatible := false
	for _, candidate := range request.GetHostProtocols() {
		if candidate.GetProtocol() == protocolName && candidate.GetMajor() == ProtocolMajor &&
			candidate.GetMinMinor() <= 0 && candidate.GetMaxMinor() >= 0 {
			compatible = true
			break
		}
	}
	if !compatible {
		return protocolError(protocolwire.ErrorCode_ERROR_CODE_PROTOCOL_MISMATCH,
			"protocol.version_mismatch", "No compatible sforum.plugin protocol version was offered.")
	}
	if request.GetLimits() == nil || request.GetLimits().GetMaxReceiveBytes() == 0 || request.GetLimits().GetMaxSendBytes() == 0 {
		return protocolError(protocolwire.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			"protocol.limits_required", "Runtime message and concurrency limits are required.")
	}
	return nil
}

func requestContext(request *protocolwire.HandshakeRequest) *protocolwire.RequestContext {
	if request == nil {
		return nil
	}
	return request.GetContext()
}

func responseContext(request *protocolwire.RequestContext, now time.Time) *protocolwire.ResponseContext {
	response := &protocolwire.ResponseContext{ServerTime: timestamppb.New(now)}
	if request != nil {
		response.RequestId = request.GetRequestId()
		response.Trace = request.GetTrace()
		response.Extension = request.GetExtension()
	}
	return response
}

func protocolError(code protocolwire.ErrorCode, reason, message string) *protocolwire.ErrorDetail {
	return &protocolwire.ErrorDetail{Code: code, Reason: reason, Message: message}
}

func selectFeatures(host, supported []*protocolwire.ProtocolFeature) []*protocolwire.ProtocolFeature {
	hostVersions := make(map[string]string, len(host))
	for _, feature := range host {
		hostVersions[feature.GetName()] = feature.GetVersion()
	}
	selected := make([]*protocolwire.ProtocolFeature, 0, len(supported))
	for _, feature := range supported {
		if hostVersions[feature.GetName()] == feature.GetVersion() {
			selected = append(selected, proto.Clone(feature).(*protocolwire.ProtocolFeature))
		}
	}
	return selected
}

func cloneFeatures(values []*protocolwire.ProtocolFeature) []*protocolwire.ProtocolFeature {
	out := make([]*protocolwire.ProtocolFeature, 0, len(values))
	for _, value := range values {
		if value != nil {
			out = append(out, proto.Clone(value).(*protocolwire.ProtocolFeature))
		}
	}
	return out
}

func cloneServices(values []*protocolwire.ServiceDescriptor) []*protocolwire.ServiceDescriptor {
	out := make([]*protocolwire.ServiceDescriptor, 0, len(values))
	for _, value := range values {
		if value != nil {
			out = append(out, proto.Clone(value).(*protocolwire.ServiceDescriptor))
		}
	}
	return out
}

var _ pluginwire.PluginRuntimeServiceServer = (*Server)(nil)

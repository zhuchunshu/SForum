package extensionsruntime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	protocolV2Name       = "sforum.plugin"
	hostAPIV2Version     = "sforum.host/v2"
	hostAPIV2Contract    = "sforum.host@2"
	hostAPIV2Legacy      = "sforum.host-api@2"
	protocolV2LocaleNone = "und"
)

// RuntimeTrustSource resolves the live exact-artifact grant before process
// start. The wire identity is disclosure only; this host lookup is authoritative.
type RuntimeTrustSource interface {
	RuntimeIdentity(context.Context, extensions.Extension) (extensions.RuntimeTrustIdentity, error)
}

type protocolV2ClientConfig struct {
	identity     *protocolv2.ExtensionIdentity
	authority    []*protocolv2.AuthorityGrant
	token        []byte
	instance     string
	hostAPI      ProtocolV2HostRegistrar
	hostBrokerID uint32
}

type protocolV2Client struct {
	ProtocolNoop
	client       pluginv2.PluginRuntimeServiceClient
	identity     *protocolv2.ExtensionIdentity
	authority    []*protocolv2.AuthorityGrant
	token        []byte
	instance     string
	hostBrokerID uint32
	sequence     atomic.Uint64
}

// ProtocolV2Error preserves the stable typed error returned by a plugin.
type ProtocolV2Error struct {
	Code      protocolv2.ErrorCode
	Reason    string
	Message   string
	Retryable bool
}

func (e *ProtocolV2Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Reason == "" {
		return e.Message
	}
	return e.Reason + ": " + e.Message
}

func newProtocolV2Client(client pluginv2.PluginRuntimeServiceClient, config protocolV2ClientConfig) *protocolV2Client {
	return &protocolV2Client{
		client: client, identity: cloneV2Identity(config.identity), authority: cloneV2Authority(config.authority),
		token: append([]byte(nil), config.token...), instance: config.instance, hostBrokerID: config.hostBrokerID,
	}
}

func (s *ProtocolStarter) newPluginClientConfig(
	ctx context.Context,
	extension extensions.Extension,
	cmd *exec.Cmd,
) (*plugin.ClientConfig, string, error) {
	version := extension.Manifest.Backend.ProtocolVersion
	if version == 0 {
		version = 1
	}
	base := &plugin.ClientConfig{
		HandshakeConfig: handshakeConfig,
		Logger:          hclog.NewNullLogger(),
		Cmd:             cmd,
	}
	switch version {
	case 1:
		base.VersionedPlugins = map[int]plugin.PluginSet{
			1: {pluginProtocolName: &netRPCPlugin{}},
		}
		base.AllowedProtocols = []plugin.Protocol{plugin.ProtocolNetRPC}
		return base, pluginProtocolName, nil
	case 2:
		if !supportsProtocolV2HostAPI(extension.Manifest.Backend.HostAPIVersion) {
			return nil, "", fmt.Errorf("%w: host API %q", ErrUnsupportedProtocol, extension.Manifest.Backend.HostAPIVersion)
		}
		clientConfig, err := s.protocolV2ClientConfig(ctx, extension)
		if err != nil {
			return nil, "", err
		}
		base.VersionedPlugins = map[int]plugin.PluginSet{
			2: {pluginProtocolV2Name: &protocolV2Plugin{clientConfig: &clientConfig}},
		}
		base.AllowedProtocols = []plugin.Protocol{plugin.ProtocolGRPC}
		base.AutoMTLS = true
		base.GRPCDialOptions = []grpc.DialOption{grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(DefaultProtocolV2MaxMessageBytes),
			grpc.MaxCallSendMsgSize(DefaultProtocolV2MaxMessageBytes),
		)}
		return base, pluginProtocolV2Name, nil
	default:
		return nil, "", ErrUnsupportedProtocol
	}
}

func supportsProtocolV2HostAPI(value string) bool {
	switch strings.TrimSpace(value) {
	case hostAPIV2Contract, hostAPIV2Legacy, hostAPIV2Version:
		return true
	default:
		return false
	}
}

func (s *ProtocolStarter) protocolV2ClientConfig(ctx context.Context, extension extensions.Extension) (protocolV2ClientConfig, error) {
	trustIdentity := extensions.RuntimeTrustIdentity{}
	var err error
	if s.trust != nil {
		trustIdentity, err = s.trust.RuntimeIdentity(ctx, extension)
	} else if extension.Source == extensions.SourceBuiltin {
		trustIdentity = extensions.RuntimeTrustIdentity{TrustGrantID: "builtin", ImpactDigest: extension.PackageDigest}
	} else {
		err = extensions.ErrTrustGrantNotFound
	}
	if err != nil {
		return protocolV2ClientConfig{}, fmt.Errorf("resolve protocol v2 trust identity: %w", err)
	}
	if trustIdentity.TrustGrantID == "" || extension.PackageDigest == "" {
		return protocolV2ClientConfig{}, fmt.Errorf("resolve protocol v2 trust identity: %w", extensions.ErrTrustGrantNotFound)
	}
	token, err := randomProtocolV2Bytes(32)
	if err != nil {
		return protocolV2ClientConfig{}, fmt.Errorf("create protocol v2 runtime token: %w", err)
	}
	instanceBytes, err := randomProtocolV2Bytes(16)
	if err != nil {
		return protocolV2ClientConfig{}, fmt.Errorf("create protocol v2 instance id: %w", err)
	}
	epoch := uint64(time.Now().UTC().UnixNano())
	if epoch == 0 {
		epoch = 1
	}
	return protocolV2ClientConfig{
		identity: &protocolv2.ExtensionIdentity{
			ExtensionId: extension.ID, ExtensionVersion: extension.Version,
			ArtifactDigest: extension.PackageDigest, TrustGrantId: trustIdentity.TrustGrantID,
			RuntimeEpoch: epoch, InstanceId: hex.EncodeToString(instanceBytes),
		},
		authority: protocolV2Authority(extension.CapabilityGrants),
		token:     token,
		instance:  hex.EncodeToString(instanceBytes),
		hostAPI:   protocolV2HostRegistrarFor(s.hostAPI),
	}, nil
}

func (c *protocolV2Client) Handshake(ctx context.Context) error {
	ctx, cancel := protocolV2Deadline(ctx, DefaultProtocolV2HandshakeTimeout)
	defer cancel()
	response, err := c.client.Handshake(ctx, &protocolv2.HandshakeRequest{
		Context: c.requestContext(ctx, "handshake"),
		HostProtocols: []*protocolv2.ProtocolRange{{
			Protocol: protocolV2Name, Major: 2, MinMinor: 0, MaxMinor: 0,
		}},
		HostFeatures: []*protocolv2.ProtocolFeature{
			{Name: "stream.routes", Version: "1"},
			{Name: "stream.files", Version: "1"},
			{Name: "stream.jobs", Version: "1"},
			{Name: "service.discovery", Version: "1"},
		},
		Limits: &protocolv2.RuntimeLimits{
			MaxReceiveBytes:         uint64(DefaultProtocolV2MaxMessageBytes),
			MaxSendBytes:            uint64(DefaultProtocolV2MaxMessageBytes),
			MaxConcurrentUnaryCalls: uint32(DefaultProtocolV2ConcurrentCalls),
			MaxConcurrentStreams:    uint32(DefaultProtocolV2ConcurrentCalls),
			DefaultDeadline:         durationpb.New(DefaultProtocolV2RequestTimeout),
			HealthDeadline:          durationpb.New(DefaultProtocolV2HandshakeTimeout),
			GracefulStopDeadline:    durationpb.New(5 * time.Second),
		},
		HostApiVersion: hostAPIV2Version,
		HostBrokerId:   c.hostBrokerID,
		RuntimeToken:   append([]byte(nil), c.token...),
	})
	if err != nil {
		return err
	}
	if err := protocolV2Error(response.GetError()); err != nil {
		return err
	}
	selected := response.GetSelectedProtocol()
	if selected.GetProtocol() != protocolV2Name || selected.GetMajor() != 2 || selected.GetMinMinor() > 0 {
		return &ProtocolV2Error{Code: protocolv2.ErrorCode_ERROR_CODE_PROTOCOL_MISMATCH, Reason: "protocol.version_mismatch", Message: "Plugin selected an incompatible protocol version."}
	}
	return nil
}

func protocolV2HostRegistrarFor(registrar HostAPIRegistrar) ProtocolV2HostRegistrar {
	if registrar == nil {
		return nil
	}
	result, _ := registrar.(ProtocolV2HostRegistrar)
	return result
}

func (c *protocolV2Client) Readiness(ctx context.Context) error {
	ctx, cancel := protocolV2Deadline(ctx, DefaultProtocolV2HandshakeTimeout)
	defer cancel()
	response, err := c.client.Readiness(ctx, &protocolv2.ReadinessRequest{Context: c.requestContext(ctx, "readiness")})
	if err != nil {
		return err
	}
	if err := protocolV2Error(response.GetError()); err != nil {
		return err
	}
	if !response.GetReady() {
		return &ProtocolV2Error{Code: protocolv2.ErrorCode_ERROR_CODE_UNAVAILABLE, Reason: "protocol.not_ready", Message: "Plugin is not ready."}
	}
	return nil
}

func (c *protocolV2Client) Health() (PluginHealth, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultProtocolV2HandshakeTimeout)
	defer cancel()
	response, err := c.client.Health(ctx, &protocolv2.HealthRequest{Context: c.requestContext(ctx, "health")})
	if err != nil {
		return PluginHealth{}, err
	}
	if err := protocolV2Error(response.GetError()); err != nil {
		return PluginHealth{}, err
	}
	return PluginHealth{OK: response.GetHealthy()}, nil
}

func (*protocolV2Client) RouteTarget() (PluginRouteTarget, error) {
	// V2 routes are served by the Route Registry over gRPC; no loopback URL is trusted.
	return PluginRouteTarget{}, nil
}

func (c *protocolV2Client) InvokeHook(input PluginHookRequest) (PluginHookResponse, error) {
	timeout := DefaultProtocolV2RequestTimeout
	if input.TimeoutMS > 0 {
		timeout = time.Duration(input.TimeoutMS) * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	payload, err := protocolV2Document("sforum.hook."+input.Name, "1", input.Payload)
	if err != nil {
		return PluginHookResponse{}, err
	}
	response, err := c.client.InvokeHook(ctx, &pluginv2.HookRequest{
		Context: c.requestContext(ctx, input.CorrelationID), HookId: input.Name,
		HookName: input.Name, HookKind: input.Kind, ContractVersion: "1",
		DeliveryId: strconv.FormatInt(input.DeliveryID, 10), Payload: payload,
		MutableFields: append([]string(nil), input.PatchFields...),
	})
	if err != nil {
		return PluginHookResponse{}, err
	}
	if err := protocolV2Error(response.GetError()); err != nil {
		return PluginHookResponse{}, err
	}
	result := PluginHookResponse{OK: response.GetAccepted()}
	if values := protocolV2Values(response.GetResult()); len(values) > 0 {
		result.Reason = stringValue(values, "reason")
		result.Message = stringValue(values, "message")
	}
	if patch := response.GetPatch().GetValue(); patch != nil {
		result.Patch = patch.AsMap()
	}
	return result, nil
}

func (c *protocolV2Client) ProviderProbe(input ProviderProbeRequest) (ProviderProbeResponse, error) {
	response, err := c.providerCall(input.Slot, "probe", map[string]any{})
	if err != nil {
		return ProviderProbeResponse{}, err
	}
	values := protocolV2Values(response.GetOutput())
	return ProviderProbeResponse{
		OK: booleanValue(values, "ok"), Reason: stringValue(values, "reason"),
		Message: stringValue(values, "message"), Details: stringMapValue(values, "details"),
		Suggestions: append([]string(nil), response.GetSuggestions()...),
	}, nil
}

func (c *protocolV2Client) SendMail(input MailProviderRequest) (MailProviderResponse, error) {
	response, err := c.providerCall("mail.provider", "send", map[string]any{
		"deliveryId": input.DeliveryID, "correlationId": input.CorrelationID,
		"fromAddress": input.FromAddress, "fromName": input.FromName, "to": input.To,
		"subject": input.Subject, "textBody": input.TextBody, "htmlBody": input.HTMLBody,
	})
	if err != nil {
		return MailProviderResponse{}, err
	}
	values := protocolV2Values(response.GetOutput())
	return MailProviderResponse{
		OK: booleanValue(values, "ok"), Classification: stringValue(values, "classification"),
		Reason: stringValue(values, "reason"), Message: stringValue(values, "message"),
	}, nil
}

func (c *protocolV2Client) providerCall(slot, operation string, input map[string]any) (*pluginv2.ProviderCallResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultProtocolV2RequestTimeout)
	defer cancel()
	document, err := protocolV2Document("sforum.provider."+slot+"."+operation, "1", input)
	if err != nil {
		return nil, err
	}
	response, err := c.client.ProviderCall(ctx, &pluginv2.ProviderCallRequest{
		Context: c.requestContext(ctx, operation), SlotId: slot, Operation: operation,
		ContractVersion: "1", Input: document,
	})
	if err != nil {
		return nil, err
	}
	if err := protocolV2Error(response.GetError()); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *protocolV2Client) requestContext(ctx context.Context, correlation string) *protocolv2.RequestContext {
	requestID := c.instance + "-" + strconv.FormatUint(c.sequence.Add(1), 10)
	if correlation == "" {
		correlation = requestID
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().UTC().Add(DefaultProtocolV2RequestTimeout)
	}
	return &protocolv2.RequestContext{
		RequestId:        requestID,
		Trace:            &protocolv2.TraceContext{TraceId: correlation},
		Locale:           protocolV2LocaleNone,
		Deadline:         timestamppb.New(deadline),
		Extension:        cloneV2Identity(c.identity),
		GrantedAuthority: cloneV2Authority(c.authority),
	}
}

func protocolV2Authority(grants []extensions.CapabilityGrant) []*protocolv2.AuthorityGrant {
	result := make([]*protocolv2.AuthorityGrant, 0, len(grants))
	for _, grant := range grants {
		source := "declared"
		if grant.Implied {
			source = "host-implied"
		}
		result = append(result, &protocolv2.AuthorityGrant{
			Key: grant.Key, ContractVersion: hostAPIV2Version,
			RiskTier: protocolV2Risk(grant.Risk), Source: source,
		})
	}
	return result
}

func protocolV2Risk(risk string) protocolv2.RiskTier {
	switch risk {
	case capabilities.RiskLow:
		return protocolv2.RiskTier_RISK_TIER_LOW
	case capabilities.RiskMedium:
		return protocolv2.RiskTier_RISK_TIER_MEDIUM
	case capabilities.RiskHigh:
		return protocolv2.RiskTier_RISK_TIER_HIGH
	default:
		return protocolv2.RiskTier_RISK_TIER_UNSPECIFIED
	}
}

func protocolV2Document(schemaID, version string, values map[string]any) (*protocolv2.TypedDocument, error) {
	value, err := structpb.NewStruct(values)
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", schemaID, err)
	}
	return &protocolv2.TypedDocument{SchemaId: schemaID, SchemaVersion: version, Value: value}, nil
}

func protocolV2Values(document *protocolv2.TypedDocument) map[string]any {
	if document == nil || document.GetValue() == nil {
		return map[string]any{}
	}
	return document.GetValue().AsMap()
}

func protocolV2Error(detail *protocolv2.ErrorDetail) error {
	if detail == nil || detail.GetCode() == protocolv2.ErrorCode_ERROR_CODE_UNSPECIFIED {
		return nil
	}
	return &ProtocolV2Error{Code: detail.GetCode(), Reason: detail.GetReason(), Message: detail.GetMessage(), Retryable: detail.GetRetryable()}
}

func protocolV2Deadline(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

func randomProtocolV2Bytes(size int) ([]byte, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return nil, err
	}
	return value, nil
}

func cloneV2Identity(value *protocolv2.ExtensionIdentity) *protocolv2.ExtensionIdentity {
	if value == nil {
		return nil
	}
	return proto.Clone(value).(*protocolv2.ExtensionIdentity)
}

func cloneV2Authority(values []*protocolv2.AuthorityGrant) []*protocolv2.AuthorityGrant {
	result := make([]*protocolv2.AuthorityGrant, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, proto.Clone(value).(*protocolv2.AuthorityGrant))
		}
	}
	return result
}

func booleanValue(values map[string]any, key string) bool {
	value, _ := values[key].(bool)
	return value
}

func stringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func stringMapValue(values map[string]any, key string) map[string]string {
	raw, _ := values[key].(map[string]any)
	result := make(map[string]string, len(raw))
	for name, value := range raw {
		if text, ok := value.(string); ok {
			result[name] = text
		}
	}
	return result
}

var _ PluginProtocol = (*protocolV2Client)(nil)
var _ error = (*ProtocolV2Error)(nil)

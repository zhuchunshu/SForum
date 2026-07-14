package extensionsruntime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	events       []extensions.ManifestEvent
	hooks        []extensions.ManifestHook
	providers    []extensions.ManifestProvider
	jobs         []extensions.ManifestJob
	routes       []extensions.ManifestRoute
	lifecycle    *extensions.ManifestLifecycle
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
	events       []extensions.ManifestEvent
	hooks        []extensions.ManifestHook
	providers    []extensions.ManifestProvider
	jobs         []extensions.ManifestJob
	routes       []extensions.ManifestRoute
	lifecycle    *extensions.ManifestLifecycle
	token        []byte
	instance     string
	hostBrokerID uint32
	serviceMu    sync.RWMutex
	services     []*protocolv2.ServiceDescriptor
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
		events: append([]extensions.ManifestEvent(nil), config.events...), hooks: cloneManifestHooks(config.hooks),
		providers: append([]extensions.ManifestProvider(nil), config.providers...),
		jobs:      append([]extensions.ManifestJob(nil), config.jobs...),
		routes:    cloneProtocolV2Routes(config.routes),
		lifecycle: cloneManifestLifecycle(config.lifecycle),
		token:     append([]byte(nil), config.token...), instance: config.instance, hostBrokerID: config.hostBrokerID,
	}
}

func (s *ProtocolStarter) newPluginClientConfig(
	ctx context.Context,
	extension extensions.Extension,
	cmd *exec.Cmd,
	instanceID string,
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
		clientConfig, err := s.protocolV2ClientConfig(ctx, extension, instanceID)
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

func (s *ProtocolStarter) protocolV2ClientConfig(
	ctx context.Context,
	extension extensions.Extension,
	runtimeInstanceID ...string,
) (protocolV2ClientConfig, error) {
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
	instanceID := ""
	if len(runtimeInstanceID) > 1 {
		return protocolV2ClientConfig{}, ErrRuntimeAdmissionInvalid
	}
	if len(runtimeInstanceID) == 1 {
		instanceID = strings.TrimSpace(runtimeInstanceID[0])
	}
	if instanceID == "" {
		instanceID, err = newProtocolV2RuntimeInstanceID()
		if err != nil {
			return protocolV2ClientConfig{}, err
		}
	}
	if _, err := normalizeRuntimeInstanceIdentity(RuntimeInstanceIdentity{
		ExtensionID: extension.ID, InstanceID: instanceID,
	}); err != nil {
		return protocolV2ClientConfig{}, err
	}
	epoch := uint64(time.Now().UTC().UnixNano())
	if epoch == 0 {
		epoch = 1
	}
	return protocolV2ClientConfig{
		identity: &protocolv2.ExtensionIdentity{
			ExtensionId: extension.ID, ExtensionVersion: extension.Version,
			ArtifactDigest: extension.PackageDigest, TrustGrantId: trustIdentity.TrustGrantID,
			RuntimeEpoch: epoch, InstanceId: instanceID,
		},
		authority: protocolV2Authority(extension.CapabilityGrants),
		events:    append([]extensions.ManifestEvent(nil), extension.Manifest.Events...),
		hooks:     cloneManifestHooks(extension.Manifest.Hooks),
		providers: append([]extensions.ManifestProvider(nil), extension.Manifest.Providers...),
		jobs:      append([]extensions.ManifestJob(nil), extension.Manifest.Jobs...),
		routes:    cloneProtocolV2Routes(extension.Manifest.Routes),
		lifecycle: cloneManifestLifecycle(extension.Manifest.Lifecycle),
		token:     token,
		instance:  instanceID,
		hostAPI:   protocolV2HostRegistrarFor(s.hostAPI),
	}, nil
}

func newProtocolV2RuntimeInstanceID() (string, error) {
	instanceBytes, err := randomProtocolV2Bytes(16)
	if err != nil {
		return "", fmt.Errorf("create protocol v2 instance id: %w", err)
	}
	return hex.EncodeToString(instanceBytes), nil
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
	c.serviceMu.Lock()
	c.services = cloneV2Services(response.GetServices())
	c.serviceMu.Unlock()
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
	return c.InvokeHookContext(context.Background(), input)
}

func (c *protocolV2Client) InvokeHookContext(parent context.Context, input PluginHookRequest) (PluginHookResponse, error) {
	declaration, err := c.eventDeclaration(input.DeclarationID, input.Name, input.Kind)
	if err != nil {
		return PluginHookResponse{}, err
	}
	if input.ContractVersion != "" && input.ContractVersion != declaration.ContractVersion {
		return PluginHookResponse{}, fmt.Errorf("protocol v2 hook %q contract does not match %q", declaration.ID, input.ContractVersion)
	}
	timeout := DefaultProtocolV2RequestTimeout
	if input.TimeoutMS > 0 {
		timeout = time.Duration(input.TimeoutMS) * time.Millisecond
	}
	ctx, cancel := protocolV2Deadline(parent, timeout)
	defer cancel()
	payloadSchemaID, payloadSchemaVersion, err := protocolV2SchemaRef(declaration.InputSchema)
	if err != nil {
		return PluginHookResponse{}, err
	}
	payload, err := protocolV2Document(payloadSchemaID, payloadSchemaVersion, input.Payload)
	if err != nil {
		return PluginHookResponse{}, err
	}
	response, err := c.client.InvokeHook(ctx, &pluginv2.HookRequest{
		Context: c.requestContext(ctx, input.CorrelationID), HookId: declaration.ID,
		HookName: declaration.Name, HookKind: declaration.Kind, ContractVersion: declaration.ContractVersion,
		DeliveryId: strconv.FormatInt(input.DeliveryID, 10), Payload: payload,
		MutableFields: append([]string(nil), input.PatchFields...),
	})
	if err != nil {
		if ctx.Err() != nil {
			return PluginHookResponse{}, ctx.Err()
		}
		return PluginHookResponse{}, err
	}
	if err := protocolV2Error(response.GetError()); err != nil {
		return PluginHookResponse{}, err
	}
	if declaration.ResultSchema != "" {
		if err := validateProtocolV2DocumentRef(response.GetResult(), declaration.ResultSchema, "hook result"); err != nil {
			return PluginHookResponse{}, err
		}
	} else if response.GetResult() != nil {
		return PluginHookResponse{}, fmt.Errorf("protocol v2 hook %q returned an undeclared result", declaration.Name)
	}
	if response.GetPatch() != nil {
		patchSchema, err := protocolV2PatchSchemaRef(declaration.ResultSchema)
		if err != nil {
			return PluginHookResponse{}, err
		}
		if err := validateProtocolV2DocumentRef(response.GetPatch(), patchSchema, "hook patch"); err != nil {
			return PluginHookResponse{}, err
		}
	}
	result := PluginHookResponse{OK: response.GetAccepted(), Result: protocolV2Values(response.GetResult())}
	if values := protocolV2Values(response.GetResult()); len(values) > 0 {
		result.Reason = stringValue(values, "reason")
		result.Message = stringValue(values, "message")
	}
	if patch := response.GetPatch().GetValue(); patch != nil {
		result.Patch = patch.AsMap()
	}
	return result, nil
}

func (c *protocolV2Client) ExecutePluginJob(parent context.Context, invocation supportjobs.PluginJobInvocation) error {
	if c == nil || c.client == nil || c.identity == nil {
		return extensions.ErrRuntimeUnavailable
	}
	if parent == nil {
		return fmt.Errorf("%w: caller context is required", supportjobs.ErrPluginJobRuntimeStale)
	}
	contract := invocation.Contract
	if contract.ExtensionID != c.identity.GetExtensionId() || contract.ExtensionVersion != c.identity.GetExtensionVersion() ||
		contract.ArtifactDigest != c.identity.GetArtifactDigest() || invocation.TrustGrantID != c.identity.GetTrustGrantId() {
		return supportjobs.ErrPluginJobRuntimeStale
	}
	frozenContract, err := extensions.PluginJobContractForExtension(extensions.Extension{
		ID: c.identity.GetExtensionId(), Version: c.identity.GetExtensionVersion(), PackageDigest: c.identity.GetArtifactDigest(),
		Manifest: extensions.Manifest{Jobs: append([]extensions.ManifestJob(nil), c.jobs...)},
	}, contract.JobName)
	if err != nil || !frozenContract.Equal(contract) {
		return supportjobs.ErrPluginJobRuntimeStale
	}
	payload, err := protocolV2Document(contract.PayloadSchemaID, contract.PayloadSchemaVersion, invocation.Payload)
	if err != nil {
		return err
	}
	ctx, cancel := protocolV2Deadline(parent, DefaultProtocolV2RequestTimeout)
	defer cancel()
	attempt := uint32(0)
	if invocation.Attempt > 0 {
		attempt = uint32(invocation.Attempt)
	}
	requestContext := c.requestContext(ctx, strconv.FormatInt(invocation.JobID, 10))
	stream, err := c.client.ExecuteJob(ctx, &pluginv2.JobRequest{
		Context: requestContext,
		JobId:   strconv.FormatInt(invocation.JobID, 10), JobKind: contract.JobName,
		PayloadVersion: contract.PayloadSchemaVersion, Attempt: attempt, Payload: payload,
	})
	if err != nil {
		return err
	}
	validator := pluginJobProgressValidator{jobID: strconv.FormatInt(invocation.JobID, 10), requestContext: requestContext}
	var terminalError error
	for {
		update, recvErr := stream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				if validator.terminal {
					return terminalError
				}
				return &ProtocolV2Error{
					Code: protocolv2.ErrorCode_ERROR_CODE_INTERNAL, Reason: "runtime.job_terminal_missing",
					Message: "Plugin job stream ended without a terminal progress state.", Retryable: true,
				}
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return recvErr
		}
		remoteError, err := validator.accept(update)
		if err != nil {
			return err
		}
		if remoteError != nil {
			terminalError = remoteError
		}
	}
}

type pluginJobProgressValidator struct {
	jobID          string
	requestContext *protocolv2.RequestContext
	lastState      protocolv2.ProgressState
	completed      uint32
	total          uint32
	seen           bool
	terminal       bool
}

func (v *pluginJobProgressValidator) accept(update *protocolv2.ProgressUpdate) (error, error) {
	invalid := func(format string, args ...any) (error, error) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidPluginJobStream, fmt.Sprintf(format, args...))
	}
	if update == nil {
		return invalid("job %q returned a nil update", v.jobID)
	}
	if v.terminal {
		return invalid("job %q emitted progress after its terminal update", v.jobID)
	}
	if update.GetStepId() != v.jobID {
		return invalid("expected job %q, got %q", v.jobID, update.GetStepId())
	}
	response := update.GetContext()
	if response == nil || response.GetRequestId() != v.requestContext.GetRequestId() ||
		!proto.Equal(response.GetExtension(), v.requestContext.GetExtension()) ||
		!proto.Equal(response.GetTrace(), v.requestContext.GetTrace()) ||
		response.GetServerTime() == nil || !response.GetServerTime().IsValid() {
		return invalid("progress response context does not match the exact runtime request")
	}
	state := update.GetState()
	if !validLifecycleProgressState(state) {
		return invalid("job %q returned state %s", v.jobID, state)
	}
	if state == protocolv2.ProgressState_PROGRESS_STATE_PLANNED && v.seen && v.lastState != protocolv2.ProgressState_PROGRESS_STATE_PLANNED {
		return invalid("job %q regressed to planned", v.jobID)
	}
	if update.GetCompletedUnits() < v.completed || update.GetTotalUnits() < v.total ||
		(update.GetTotalUnits() > 0 && update.GetCompletedUnits() > update.GetTotalUnits()) {
		return invalid("job %q progress counters are invalid", v.jobID)
	}
	terminal := isLifecycleTerminal(state)
	if state == protocolv2.ProgressState_PROGRESS_STATE_SUCCEEDED && update.GetTotalUnits() > 0 && update.GetCompletedUnits() != update.GetTotalUnits() {
		return invalid("successful job %q did not complete every unit", v.jobID)
	}
	detail := update.GetError()
	hasError := detail != nil && detail.GetCode() != protocolv2.ErrorCode_ERROR_CODE_UNSPECIFIED
	if state == protocolv2.ProgressState_PROGRESS_STATE_FAILED || state == protocolv2.ProgressState_PROGRESS_STATE_CANCELLED {
		if !hasError || strings.TrimSpace(detail.GetReason()) == "" {
			return invalid("terminal job %q has no typed error", v.jobID)
		}
		if (state == protocolv2.ProgressState_PROGRESS_STATE_CANCELLED && detail.GetCode() != protocolv2.ErrorCode_ERROR_CODE_CANCELLED) ||
			(state == protocolv2.ProgressState_PROGRESS_STATE_FAILED && detail.GetCode() == protocolv2.ErrorCode_ERROR_CODE_CANCELLED) {
			return invalid("terminal job %q state and error code disagree", v.jobID)
		}
		if retry := detail.GetRetryAfter(); retry != nil && !retry.IsValid() {
			return invalid("terminal job %q has an invalid retry time", v.jobID)
		}
	} else if hasError {
		return invalid("non-failed job %q returned an error", v.jobID)
	}
	if update.GetResult() != nil {
		return invalid("job %q returned a result without a declared result schema", v.jobID)
	}
	v.seen = true
	v.lastState = state
	v.completed = update.GetCompletedUnits()
	v.total = update.GetTotalUnits()
	v.terminal = terminal
	if state == protocolv2.ProgressState_PROGRESS_STATE_FAILED || state == protocolv2.ProgressState_PROGRESS_STATE_CANCELLED {
		return protocolV2Error(detail), nil
	}
	return nil, nil
}

func (c *protocolV2Client) eventDeclaration(id, name, kind string) (extensions.ManifestEvent, error) {
	name = strings.TrimSpace(name)
	kind = strings.TrimSpace(kind)
	id = strings.TrimSpace(id)
	for _, hook := range c.hooks {
		if (id == "" || hook.ID == id) && hook.Name == name && hook.Kind == kind {
			if hook.ID == "" || hook.ContractVersion == "" || hook.InputSchema == "" ||
				(hook.Kind != "observe" && hook.ResultSchema == "") {
				return extensions.ManifestEvent{}, fmt.Errorf("protocol v2 hook %q has an incomplete contract", name)
			}
			return extensions.ManifestEvent{
				ID: hook.ID, ContractVersion: hook.ContractVersion, Name: hook.Name, Kind: hook.Kind,
				Handler: hook.Handler, InputSchema: hook.InputSchema, ResultSchema: hook.ResultSchema,
				Priority: hook.Priority, TimeoutMS: hook.TimeoutMS,
			}, nil
		}
	}
	for _, event := range c.events {
		if event.Name == name && event.Kind == kind {
			if event.ID == "" || event.ContractVersion == "" || event.InputSchema == "" || event.ResultSchema == "" {
				return extensions.ManifestEvent{}, fmt.Errorf("protocol v2 event %q has an incomplete contract", name)
			}
			return event, nil
		}
	}
	return extensions.ManifestEvent{}, fmt.Errorf("protocol v2 event %q kind %q is not declared by the manifest", name, kind)
}

func cloneManifestHooks(items []extensions.ManifestHook) []extensions.ManifestHook {
	result := make([]extensions.ManifestHook, len(items))
	for index, item := range items {
		result[index] = cloneManifestHook(item)
	}
	return result
}

func protocolV2SchemaRef(reference string) (string, string, error) {
	reference = strings.TrimSpace(reference)
	index := strings.LastIndexByte(reference, '@')
	if index <= 0 || index == len(reference)-1 {
		return "", "", fmt.Errorf("protocol v2 schema reference %q is invalid", reference)
	}
	version := reference[index+1:]
	if version[0] == '0' {
		return "", "", fmt.Errorf("protocol v2 schema reference %q is invalid", reference)
	}
	for _, value := range version {
		if value < '0' || value > '9' {
			return "", "", fmt.Errorf("protocol v2 schema reference %q is invalid", reference)
		}
	}
	return reference[:index], version, nil
}

func protocolV2PatchSchemaRef(resultSchema string) (string, error) {
	schemaID, version, err := protocolV2SchemaRef(resultSchema)
	if err != nil {
		return "", err
	}
	return schemaID + ".patch@" + version, nil
}

func validateProtocolV2DocumentRef(document *protocolv2.TypedDocument, reference, label string) error {
	schemaID, version, err := protocolV2SchemaRef(reference)
	if err != nil {
		return err
	}
	if document == nil || document.GetSchemaId() != schemaID || document.GetSchemaVersion() != version {
		return fmt.Errorf("protocol v2 %s must match schema %q", label, reference)
	}
	return nil
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

func (c *protocolV2Client) InvokeVersionedProvider(
	parent context.Context,
	input VersionedProviderRequest,
) (VersionedProviderResponse, error) {
	if input.Operation != VersionedProviderOperationInvoke {
		return VersionedProviderResponse{}, fmt.Errorf("protocol v2 provider operation %q is not declared", input.Operation)
	}
	var declaration *extensions.ManifestProvider
	for index := range c.providers {
		candidate := &c.providers[index]
		if candidate.ID == input.DeclarationID && candidate.Slot == input.Slot && candidate.ContractVersion == input.ContractVersion {
			declaration = candidate
			break
		}
	}
	if declaration == nil || declaration.RequestSchema != input.RequestSchema || declaration.ResponseSchema != input.ResponseSchema {
		return VersionedProviderResponse{}, fmt.Errorf("protocol v2 provider %q contract is not declared", input.DeclarationID)
	}
	timeout := input.Timeout
	if timeout <= 0 || timeout > DefaultProtocolV2RequestTimeout {
		timeout = DefaultProtocolV2RequestTimeout
	}
	ctx, cancel := protocolV2Deadline(parent, timeout)
	defer cancel()
	requestSchemaID, requestSchemaVersion, err := protocolV2SchemaRef(input.RequestSchema)
	if err != nil {
		return VersionedProviderResponse{}, err
	}
	document, err := protocolV2Document(requestSchemaID, requestSchemaVersion, input.Input)
	if err != nil {
		return VersionedProviderResponse{}, err
	}
	response, err := c.client.ProviderCall(ctx, &pluginv2.ProviderCallRequest{
		Context: c.requestContext(ctx, input.Operation), SlotId: input.Slot, Operation: input.Operation,
		ContractVersion: input.ContractVersion, Input: document, DeclarationId: input.DeclarationID,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.DeadlineExceeded:
			return VersionedProviderResponse{}, context.DeadlineExceeded
		case codes.Canceled:
			return VersionedProviderResponse{}, context.Canceled
		}
		return VersionedProviderResponse{}, err
	}
	if err := protocolV2Error(response.GetError()); err != nil {
		return VersionedProviderResponse{}, err
	}
	if err := validateProtocolV2DocumentRef(response.GetOutput(), input.ResponseSchema, "provider output"); err != nil {
		return VersionedProviderResponse{}, err
	}
	return VersionedProviderResponse{Output: protocolV2Values(response.GetOutput())}, nil
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
var _ pluginHookContextInvoker = (*protocolV2Client)(nil)
var _ error = (*ProtocolV2Error)(nil)

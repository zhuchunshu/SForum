package extensionsruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
)

const maximumIdentityProviderDocumentBytes = 1 << 20

var (
	ErrIdentityProviderInvocationInvalid = errors.New("identity provider invocation is invalid")
	ErrIdentityProviderUnavailable       = errors.New("identity provider runtime is unavailable")
	ErrIdentityProviderStale             = errors.New("identity provider exact contract is stale")
	ErrIdentityProviderAcceptFailed      = errors.New("identity provider Host acceptance failed")
)

// IdentityProviderInvocation contains only Host-selected workflow input. The
// executable declaration, timeout, failure policy, and Schema references are
// always resolved from the immutable Identity Registry publication.
type IdentityProviderInvocation struct {
	ProviderID  string
	Operation   string
	ActorUserID int64
	Input       map[string]any
}

// IdentityProviderInvocationResult is the exact proposal presented to the
// Host acceptance boundary. A provider response is never authority by itself.
type IdentityProviderInvocationResult struct {
	RegistryRevision uint64
	RegistryDigest   string
	Provider         identityregistry.ProviderContribution
	Operation        identityregistry.ProviderOperation
	Output           map[string]any
}

// IdentityProviderCommitFence is a one-shot final admission validation. Accept
// must call it inside its Host transaction immediately before committing the
// effect and audit. It does not provide a cross-system CAS: the Host store must
// keep its own exact-artifact checks and commit rollback-capable.
type IdentityProviderCommitFence func() error

// IdentityProviderAccept applies the Host-owned effect and audit while the
// exact runtime admission lease is still held. The callback must keep its own
// transaction rollback-capable when the fence rejects and must honor ctx.
type IdentityProviderAccept func(
	context.Context,
	IdentityProviderInvocationResult,
	IdentityProviderCommitFence,
) error

// IdentityProviderRuntime binds one Manager to one authoritative Identity
// Registry. Callers cannot substitute a different Registry per invocation.
type IdentityProviderRuntime struct {
	manager  *Manager
	registry *identityregistry.Registry
}

func NewIdentityProviderRuntime(
	manager *Manager,
	registry *identityregistry.Registry,
) (*IdentityProviderRuntime, error) {
	if manager == nil || registry == nil {
		return nil, ErrIdentityProviderInvocationInvalid
	}
	return &IdentityProviderRuntime{manager: manager, registry: registry}, nil
}

type exactIdentityProviderInvoker interface {
	InvokeIdentityProviderInstance(
		context.Context,
		RuntimeInstanceIdentity,
		extensions.Extension,
		VersionedIdentityProviderRequest,
	) (VersionedIdentityProviderResponse, error)
}

// Invoke resolves and invokes one exact active Identity
// provider. It deliberately has no Core or generic-provider fallback.
func (r *IdentityProviderRuntime) Invoke(
	ctx context.Context,
	input IdentityProviderInvocation,
	accept IdentityProviderAccept,
) (IdentityProviderInvocationResult, error) {
	if r == nil || r.manager == nil || r.registry == nil || ctx == nil || accept == nil || input.ActorUserID < 0 {
		return IdentityProviderInvocationResult{}, ErrIdentityProviderInvocationInvalid
	}
	if err := ctx.Err(); err != nil {
		return IdentityProviderInvocationResult{}, err
	}
	input.ProviderID = strings.ToLower(strings.TrimSpace(input.ProviderID))
	input.Operation = strings.ToLower(strings.TrimSpace(input.Operation))
	if input.ProviderID == "" || input.Operation == "" {
		return IdentityProviderInvocationResult{}, ErrIdentityProviderInvocationInvalid
	}

	registrySnapshot := r.registry.Snapshot()
	if registrySnapshot.SafeMode {
		return IdentityProviderInvocationResult{}, errors.Join(ErrIdentityProviderUnavailable, identityregistry.ErrSafeMode)
	}
	provider, err := r.registry.ResolveProvider(input.ProviderID)
	if err != nil {
		return IdentityProviderInvocationResult{}, errors.Join(ErrIdentityProviderUnavailable, err)
	}
	operation, found := exactIdentityProviderOperation(provider, input.Operation)
	if provider.Artifact.Core || strings.TrimSpace(provider.Artifact.RuntimeInstanceID) == "" || !found {
		return IdentityProviderInvocationResult{}, ErrIdentityProviderUnavailable
	}
	if err := validateIdentityRegistryState(
		r.registry, registrySnapshot.Revision, registrySnapshot.Digest, provider,
	); err != nil {
		return IdentityProviderInvocationResult{}, err
	}

	requestInput, err := cloneIdentityProviderDocument(input.Input)
	if err != nil {
		return IdentityProviderInvocationResult{}, fmt.Errorf("%w: clone input: %v", ErrIdentityProviderInvocationInvalid, err)
	}
	claim := identityregistry.ProviderOperationSchemaClaim{
		ProviderID: provider.ID, ContractVersion: provider.ContractVersion,
		Operation: operation.Name, Artifact: provider.Artifact,
	}
	if err := r.registry.ValidateProviderOperationInput(claim, requestInput); err != nil {
		return IdentityProviderInvocationResult{}, fmt.Errorf("%w: input Schema: %w", ErrIdentityProviderInvocationInvalid, err)
	}

	identity := RuntimeInstanceIdentity{
		ExtensionID: provider.Artifact.ExtensionID,
		InstanceID:  provider.Artifact.RuntimeInstanceID,
	}
	lease, err := r.manager.AcquireRuntimeCall(ctx, identity, RuntimeCallProvider)
	if err != nil {
		return IdentityProviderInvocationResult{}, errors.Join(ErrIdentityProviderUnavailable, err)
	}
	defer lease.Release()

	extension, err := r.manager.exactIdentityManagedRuntime(identity, provider.Artifact)
	if err != nil {
		return IdentityProviderInvocationResult{}, err
	}
	if err := validateIdentityRegistryState(
		r.registry, registrySnapshot.Revision, registrySnapshot.Digest, provider,
	); err != nil {
		return IdentityProviderInvocationResult{}, err
	}
	if cause := context.Cause(lease.Context); cause != nil {
		return IdentityProviderInvocationResult{}, cause
	}

	invoker, ok := r.manager.starter.(exactIdentityProviderInvoker)
	if !ok {
		return IdentityProviderInvocationResult{}, ErrIdentityProviderUnavailable
	}
	callCtx, cancelCall := context.WithTimeout(
		lease.Context,
		time.Duration(operation.TimeoutMS)*time.Millisecond,
	)
	response, err := invoker.InvokeIdentityProviderInstance(
		callCtx,
		identity,
		extension,
		VersionedIdentityProviderRequest{
			ProviderID: provider.ID, ContractVersion: provider.ContractVersion,
			Kind: provider.Kind, Handler: provider.Handler, Priority: provider.Priority,
			Operation:   operation.Name,
			InputSchema: operation.InputSchema, InputSchemaWireReference: operation.InputSchemaWireReference,
			OutputSchema: operation.OutputSchema, OutputSchemaWireReference: operation.OutputSchemaWireReference,
			Timeout:       time.Duration(operation.TimeoutMS) * time.Millisecond,
			FailurePolicy: operation.FailurePolicy, ActorUserID: input.ActorUserID,
			Input: requestInput,
		},
	)
	callCause := context.Cause(callCtx)
	cancelCall()
	if callCause != nil {
		return IdentityProviderInvocationResult{}, callCause
	}
	if err != nil {
		return IdentityProviderInvocationResult{}, err
	}
	output, err := cloneIdentityProviderDocument(response.Output)
	if err != nil {
		return IdentityProviderInvocationResult{}, fmt.Errorf("%w: clone output: %v", ErrIdentityProviderInvocationInvalid, err)
	}
	if err := r.registry.ValidateProviderOperationOutput(claim, output); err != nil {
		return IdentityProviderInvocationResult{}, fmt.Errorf("%w: output Schema: %w", ErrIdentityProviderInvocationInvalid, err)
	}

	// Recheck all mutable Host-owned authority immediately before the effect and
	// audit callback. The lease keeps the exact process alive through Accept.
	if err := validateIdentityRegistryState(
		r.registry, registrySnapshot.Revision, registrySnapshot.Digest, provider,
	); err != nil {
		return IdentityProviderInvocationResult{}, err
	}
	if _, err := r.manager.exactIdentityManagedRuntime(identity, provider.Artifact); err != nil {
		return IdentityProviderInvocationResult{}, err
	}
	if cause := context.Cause(lease.Context); cause != nil {
		return IdentityProviderInvocationResult{}, cause
	}
	result := IdentityProviderInvocationResult{
		RegistryRevision: registrySnapshot.Revision,
		RegistryDigest:   registrySnapshot.Digest,
		Provider:         provider,
		Operation:        operation,
		Output:           output,
	}
	accepted := cloneIdentityProviderResult(result)
	var fenceCalls atomic.Int32
	var fenceOpen atomic.Bool
	fenceOpen.Store(true)
	fence := IdentityProviderCommitFence(func() error {
		if !fenceOpen.Load() || fenceCalls.Add(1) != 1 {
			return ErrIdentityProviderInvocationInvalid
		}
		if cause := context.Cause(lease.Context); cause != nil {
			return cause
		}
		if err := validateIdentityRegistryState(
			r.registry, registrySnapshot.Revision, registrySnapshot.Digest, provider,
		); err != nil {
			return err
		}
		_, err := r.manager.exactIdentityManagedRuntime(identity, provider.Artifact)
		return err
	})
	acceptErr := accept(lease.Context, accepted, fence)
	fenceOpen.Store(false)
	if acceptErr != nil {
		return IdentityProviderInvocationResult{}, errors.Join(ErrIdentityProviderAcceptFailed, acceptErr)
	}
	if fenceCalls.Load() != 1 {
		return IdentityProviderInvocationResult{}, errors.Join(
			ErrIdentityProviderAcceptFailed,
			fmt.Errorf("%w: commit fence must be called exactly once", ErrIdentityProviderInvocationInvalid),
		)
	}
	// Accept success is the Host transaction's terminal result. A Registry or
	// admission change after its fence is ordered after this accepted call; do
	// not turn a committed effect into a retryable error with a post-commit read.
	return cloneIdentityProviderResult(result), nil
}

func (m *Manager) exactIdentityManagedRuntime(
	identity RuntimeInstanceIdentity,
	artifact identityregistry.Artifact,
) (extensions.Extension, error) {
	snapshot, err := m.InspectRuntimeInstance(identity)
	if err != nil {
		return extensions.Extension{}, errors.Join(ErrIdentityProviderUnavailable, err)
	}
	extension, available := m.runningExtension(identity.ExtensionID)
	if !available || !snapshot.Active || snapshot.Admission.Draining || snapshot.Admission.Quarantined ||
		snapshot.Admission.Forced || snapshot.Identity.ExtensionID != artifact.ExtensionID ||
		snapshot.Identity.InstanceID != artifact.RuntimeInstanceID ||
		snapshot.ExtensionVersion != artifact.ExtensionVersion ||
		snapshot.ArtifactDigest != artifact.PackageDigest || snapshot.VersionID != artifact.VersionID ||
		extension.ID != artifact.ExtensionID || extension.Version != artifact.ExtensionVersion ||
		extension.PackageDigest != artifact.PackageDigest || extension.ActiveVersionID != artifact.VersionID ||
		extension.Type != extensions.TypePlugin || extension.Manifest.ID != artifact.ExtensionID ||
		extension.Manifest.Version != artifact.ExtensionVersion || extension.Manifest.Type != extensions.TypePlugin ||
		extension.Manifest.Backend.ProtocolVersion != 2 {
		return extensions.Extension{}, ErrIdentityProviderStale
	}
	return extension, nil
}

func exactIdentityProviderOperation(
	provider identityregistry.ProviderContribution,
	name string,
) (identityregistry.ProviderOperation, bool) {
	for _, operation := range provider.Operations {
		if operation.Name == name {
			return operation, true
		}
	}
	return identityregistry.ProviderOperation{}, false
}

func validateIdentityRegistryState(
	registry *identityregistry.Registry,
	revision uint64,
	digest string,
	expected identityregistry.ProviderContribution,
) error {
	if registry == nil {
		return ErrIdentityProviderInvocationInvalid
	}
	snapshot := registry.Snapshot()
	if snapshot.SafeMode {
		return errors.Join(ErrIdentityProviderStale, identityregistry.ErrSafeMode)
	}
	current, err := registry.ResolveProvider(expected.ID)
	if err != nil || snapshot.Revision != revision || snapshot.Digest != digest ||
		registry.Revision() != revision ||
		!reflect.DeepEqual(current, expected) {
		return errors.Join(ErrIdentityProviderStale, err)
	}
	return nil
}

func cloneIdentityProviderDocument(input map[string]any) (map[string]any, error) {
	if input == nil {
		input = map[string]any{}
	}
	if err := preflightComponentDocumentsBounded(maximumIdentityProviderDocumentBytes, input); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	if len(raw) > maximumIdentityProviderDocumentBytes {
		return nil, fmt.Errorf("document exceeds %d bytes", maximumIdentityProviderDocumentBytes)
	}
	result := map[string]any{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func cloneIdentityProviderResult(input IdentityProviderInvocationResult) IdentityProviderInvocationResult {
	result := input
	result.Provider.Operations = append([]identityregistry.ProviderOperation(nil), input.Provider.Operations...)
	result.Output, _ = cloneIdentityProviderDocument(input.Output)
	return result
}

func (s *ProtocolStarter) InvokeIdentityProviderInstance(
	ctx context.Context,
	identity RuntimeInstanceIdentity,
	extension extensions.Extension,
	request VersionedIdentityProviderRequest,
) (VersionedIdentityProviderResponse, error) {
	client, err := s.exactIdentityRuntimeClient(ctx, identity, extension)
	if err != nil {
		return VersionedIdentityProviderResponse{}, err
	}
	return client.InvokeIdentityProvider(ctx, request)
}

func (s *ProtocolStarter) exactIdentityRuntimeClient(
	ctx context.Context,
	identity RuntimeInstanceIdentity,
	extension extensions.Extension,
) (*protocolV2Client, error) {
	if s == nil || ctx == nil {
		return nil, ErrIdentityProviderUnavailable
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return nil, errors.Join(ErrIdentityProviderUnavailable, err)
	}
	var client *protocolV2Client
	if err := func() error {
		unlock, err := s.lockExtensionLifecycleContext(ctx, identity.ExtensionID)
		if err != nil {
			return err
		}
		defer unlock()
		instance := s.protocolInstance(identity)
		if instance == nil {
			return errors.Join(ErrIdentityProviderUnavailable, protocolInstanceNotFound(identity))
		}
		if instance.protocolVersion != 2 {
			return ErrIdentityProviderUnavailable
		}
		manifestDigest, digestErr := protocolRuntimeManifestDigest(extension.Manifest)
		if digestErr != nil {
			return errors.Join(ErrIdentityProviderStale, digestErr)
		}
		var ok bool
		client, ok = instance.protocol.(*protocolV2Client)
		if !ok || client.identity == nil ||
			client.identity.GetExtensionId() != identity.ExtensionID ||
			client.identity.GetInstanceId() != identity.InstanceID ||
			client.identity.GetExtensionVersion() != instance.extensionVersion ||
			client.identity.GetArtifactDigest() != instance.artifactDigest ||
			manifestDigest != instance.manifestDigest ||
			extension.ID != identity.ExtensionID || extension.Version != instance.extensionVersion ||
			extension.PackageDigest != instance.artifactDigest {
			return ErrIdentityProviderStale
		}
		s.mu.Lock()
		s.recordProtocolCallLocked(identity.ExtensionID)
		s.mu.Unlock()
		return nil
	}(); err != nil {
		return nil, err
	}
	return client, nil
}

var _ exactIdentityProviderInvoker = (*ProtocolStarter)(nil)

package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var ErrProtocolV2IdentityProviderInvalid = errors.New("protocol v2 identity provider invocation is invalid")

const protocolV2IdentityServerTimeSkew = 5 * time.Second

// InvokeIdentityProvider calls one exact executable Identity declaration. Host
// effects, Registry JSON Schema validation, and audit remain outside the
// transport so the Manager can keep its exact admission lease through them.
func (c *protocolV2Client) InvokeIdentityProvider(
	parent context.Context,
	input VersionedIdentityProviderRequest,
) (VersionedIdentityProviderResponse, error) {
	if c == nil || c.client == nil || c.identity == nil || parent == nil {
		return VersionedIdentityProviderResponse{}, extensions.ErrRuntimeUnavailable
	}
	if err := parent.Err(); err != nil {
		return VersionedIdentityProviderResponse{}, err
	}
	if input.ActorUserID < 0 {
		return VersionedIdentityProviderResponse{}, fmt.Errorf("%w: actor user id", ErrProtocolV2IdentityProviderInvalid)
	}
	operation, err := c.exactManifestIdentityOperation(input)
	if err != nil {
		return VersionedIdentityProviderResponse{}, err
	}
	declaredTimeout := time.Duration(operation.TimeoutMS) * time.Millisecond
	if input.Timeout != declaredTimeout || declaredTimeout <= 0 || declaredTimeout > DefaultProtocolV2RequestTimeout {
		return VersionedIdentityProviderResponse{}, fmt.Errorf(
			"%w: provider %q timeout drifted", ErrProtocolV2IdentityProviderInvalid, input.ProviderID,
		)
	}
	inputSchemaID, inputSchemaVersion, err := protocolV2SchemaRef(input.InputSchemaWireReference)
	if err != nil {
		return VersionedIdentityProviderResponse{}, fmt.Errorf("%w: %v", ErrProtocolV2IdentityProviderInvalid, err)
	}
	if _, _, err := protocolV2SchemaRef(input.OutputSchemaWireReference); err != nil {
		return VersionedIdentityProviderResponse{}, fmt.Errorf("%w: %v", ErrProtocolV2IdentityProviderInvalid, err)
	}
	document, err := protocolV2Document(inputSchemaID, inputSchemaVersion, input.Input)
	if err != nil {
		return VersionedIdentityProviderResponse{}, fmt.Errorf("%w: %v", ErrProtocolV2IdentityProviderInvalid, err)
	}

	ctx, cancel := protocolV2Deadline(parent, declaredTimeout)
	defer cancel()
	requestContext := c.identityRuntimeRequestContext(ctx, input.ActorUserID)
	startedAt := time.Now().UTC()
	response, err := c.client.ProviderCall(ctx, &pluginv2.ProviderCallRequest{
		Context: requestContext, SlotId: ProtocolV2IdentityProviderSlot,
		DeclarationId: input.ProviderID, ContractVersion: input.ContractVersion,
		Operation: input.Operation, Input: document,
	})
	receivedAt := time.Now().UTC()
	if err != nil {
		return VersionedIdentityProviderResponse{}, mapProtocolV2IdentityCallError(ctx, err)
	}
	if err := validateProtocolV2IdentityResponseContext(
		response.GetContext(), requestContext, startedAt, receivedAt,
	); err != nil {
		return VersionedIdentityProviderResponse{}, err
	}
	if err := protocolV2Error(response.GetError()); err != nil {
		return VersionedIdentityProviderResponse{}, err
	}
	if err := validateProtocolV2DocumentRef(
		response.GetOutput(), input.OutputSchemaWireReference, "identity provider output",
	); err != nil {
		return VersionedIdentityProviderResponse{}, fmt.Errorf("%w: %v", ErrProtocolV2IdentityProviderInvalid, err)
	}
	return VersionedIdentityProviderResponse{Output: protocolV2Values(response.GetOutput())}, nil
}

func (c *protocolV2Client) exactManifestIdentityOperation(
	input VersionedIdentityProviderRequest,
) (extensions.ManifestIdentityProviderOperation, error) {
	if c.manifestIdentity == nil {
		return extensions.ManifestIdentityProviderOperation{}, fmt.Errorf(
			"%w: provider %q is not declared", ErrProtocolV2IdentityProviderInvalid, input.ProviderID,
		)
	}
	for _, provider := range c.manifestIdentity.Providers {
		if provider.ID != input.ProviderID || provider.ContractVersion != input.ContractVersion {
			continue
		}
		if provider.Kind != input.Kind || provider.Handler != input.Handler || provider.Priority != input.Priority {
			return extensions.ManifestIdentityProviderOperation{}, fmt.Errorf(
				"%w: provider %q declaration drifted", ErrProtocolV2IdentityProviderInvalid, input.ProviderID,
			)
		}
		for _, operation := range provider.Operations {
			if operation.Name != input.Operation {
				continue
			}
			if operation.InputSchema != input.InputSchema || operation.OutputSchema != input.OutputSchema ||
				operation.FailurePolicy != input.FailurePolicy {
				return extensions.ManifestIdentityProviderOperation{}, fmt.Errorf(
					"%w: provider %q operation %q drifted",
					ErrProtocolV2IdentityProviderInvalid, input.ProviderID, input.Operation,
				)
			}
			return operation, nil
		}
		return extensions.ManifestIdentityProviderOperation{}, fmt.Errorf(
			"%w: provider %q operation %q is not declared",
			ErrProtocolV2IdentityProviderInvalid, input.ProviderID, input.Operation,
		)
	}
	return extensions.ManifestIdentityProviderOperation{}, fmt.Errorf(
		"%w: provider %q is not declared", ErrProtocolV2IdentityProviderInvalid, input.ProviderID,
	)
}

// Identity calls carry no subprocess authority or raw request/session
// projection. A positive UserID is the only optional actor field.
func (c *protocolV2Client) identityRuntimeRequestContext(
	ctx context.Context,
	actorUserID int64,
) *protocolv2.RequestContext {
	request := c.requestContext(ctx, "identity.runtime")
	request.Trace = nil
	request.Actor = nil
	request.GrantedAuthority = nil
	request.IdempotencyKey = ""
	request.HostCommandDelegations = nil
	request.HostQueryDelegations = nil
	if actorUserID > 0 {
		request.Actor = &protocolv2.Actor{UserId: actorUserID}
	}
	return request
}

func validateProtocolV2IdentityResponseContext(
	response *protocolv2.ResponseContext,
	request *protocolv2.RequestContext,
	startedAt time.Time,
	receivedAt time.Time,
) error {
	if response == nil || request == nil || response.GetRequestId() != request.GetRequestId() ||
		!proto.Equal(response.GetTrace(), request.GetTrace()) ||
		!proto.Equal(response.GetExtension(), request.GetExtension()) {
		return fmt.Errorf("%w: response context does not match the exact runtime request", ErrProtocolV2IdentityProviderInvalid)
	}
	serverTime := response.GetServerTime()
	if serverTime == nil || !serverTime.IsValid() {
		return fmt.Errorf("%w: response server time is invalid", ErrProtocolV2IdentityProviderInvalid)
	}
	value := serverTime.AsTime()
	if value.Before(startedAt.Add(-protocolV2IdentityServerTimeSkew)) ||
		value.After(receivedAt.Add(protocolV2IdentityServerTimeSkew)) {
		return fmt.Errorf("%w: response server time is outside the request window", ErrProtocolV2IdentityProviderInvalid)
	}
	return nil
}

func mapProtocolV2IdentityCallError(ctx context.Context, err error) error {
	switch status.Code(err) {
	case codes.DeadlineExceeded:
		return context.DeadlineExceeded
	case codes.Canceled:
		return context.Canceled
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

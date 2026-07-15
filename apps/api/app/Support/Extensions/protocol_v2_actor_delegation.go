package extensionsruntime

import (
	"context"
	"errors"
	"sort"
	"strings"

	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
)

var (
	ErrProtocolV2ActorDelegationInvalid     = errors.New("protocol v2 actor delegation is invalid")
	ErrProtocolV2ActorDelegationUnavailable = errors.New("protocol v2 actor delegation is unavailable")
)

// invocationRequestContext is used only by Host-owned route/admin transports.
// Jobs and other background calls continue to use requestContext directly and
// therefore cannot acquire an actor delegation.
func (c *protocolV2Client) invocationRequestContext(
	ctx context.Context,
	correlation string,
	actor *ProtocolV2InvocationActor,
	idempotencyKey string,
) (*protocolv2.RequestContext, error) {
	requestContext := c.requestContext(ctx, correlation)
	if idempotencyKey != "" {
		if !validProtocolV2InvocationIdempotencyKey(idempotencyKey) {
			return nil, ErrProtocolV2ActorDelegationInvalid
		}
		requestContext.IdempotencyKey = idempotencyKey
	}
	if actor == nil {
		return requestContext, nil
	}
	if actor.UserID <= 0 {
		return nil, ErrProtocolV2ActorDelegationInvalid
	}
	requestContext.Actor = &protocolv2.Actor{
		UserId: actor.UserID, PermissionKeys: append([]string(nil), actor.PermissionKeys...),
	}
	if idempotencyKey == "" {
		return requestContext, nil
	}
	if !c.hostCommands {
		return requestContext, nil
	}
	if c.delegations == nil {
		return nil, ErrProtocolV2ActorDelegationUnavailable
	}
	grants, err := c.delegations.IssueProtocolV2ActorDelegations(ctx, hostapi.ProtocolV2ActorDelegationBundleRequest{
		ActorUserID: actor.UserID, PermissionKeys: append([]string(nil), actor.PermissionKeys...),
		Runtime: cloneV2Identity(c.identity), IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.Join(ErrProtocolV2ActorDelegationUnavailable, err)
	}
	sort.Slice(grants, func(left, right int) bool {
		if grants[left].CommandID == grants[right].CommandID {
			return grants[left].CommandVersion < grants[right].CommandVersion
		}
		return grants[left].CommandID < grants[right].CommandID
	})
	seen := make(map[string]bool, len(grants))
	requestContext.HostCommandDelegations = make([]*protocolv2.HostCommandDelegation, 0, len(grants))
	for _, grant := range grants {
		commandID := strings.TrimSpace(grant.CommandID)
		commandVersion := strings.TrimSpace(grant.CommandVersion)
		key := commandID + "\x00" + commandVersion
		if commandID == "" || grant.CommandID != commandID || commandVersion == "" || grant.CommandVersion != commandVersion ||
			grant.IdempotencyKey != idempotencyKey || strings.TrimSpace(grant.Token) == "" || seen[key] {
			return nil, ErrProtocolV2ActorDelegationInvalid
		}
		seen[key] = true
		requestContext.HostCommandDelegations = append(requestContext.HostCommandDelegations, &protocolv2.HostCommandDelegation{
			CommandId: grant.CommandID, CommandVersion: grant.CommandVersion,
			IdempotencyKey: grant.IdempotencyKey, Token: grant.Token,
		})
	}
	return requestContext, nil
}

// ValidateProtocolV2InvocationIdempotencyKey lets HTTP adapters reject a bad
// key before audit or plugin execution. Empty means the optional header was not
// supplied and remains valid.
func ValidateProtocolV2InvocationIdempotencyKey(value string) error {
	if value == "" || validProtocolV2InvocationIdempotencyKey(value) {
		return nil
	}
	return ErrProtocolV2ActorDelegationInvalid
}

func validProtocolV2InvocationIdempotencyKey(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < 0x21 || value[index] > 0x7e {
			return false
		}
	}
	return true
}

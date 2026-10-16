package extensionsruntime

import (
	"context"
	"errors"
	"sort"
	"strings"

	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
)

var (
	ErrProtocolV2ActorDelegationInvalid       = errors.New("protocol v2 actor delegation is invalid")
	ErrProtocolV2ActorDelegationUnavailable   = errors.New("protocol v2 actor delegation is unavailable")
	ErrProtocolV2SessionDelegationUnavailable = errors.New("protocol v2 session delegation is unavailable")
)

// invocationRequestContext is used only by Host-owned route/admin transports.
// Jobs and other background calls continue to use requestContext directly and
// therefore cannot acquire an actor delegation.
//
// The returned retire function settles the S9 session delegation for this
// invocation and is never nil; callers must run it (defer) when the invocation
// ends, because that is what makes the delegation single-invocation.
func (c *protocolV2Client) invocationRequestContext(
	ctx context.Context,
	correlation string,
	actor *ProtocolV2InvocationActor,
	idempotencyKey string,
) (*protocolv2.RequestContext, func(), error) {
	requestContext := c.requestContext(ctx, correlation)
	if err := c.attachInvocationActorDelegation(ctx, requestContext, actor, idempotencyKey); err != nil {
		return nil, noopProtocolV2Retire, err
	}
	// 会话委派独立于 actor：匿名会话同样有会话状态（S5），因此这里不能挂在
	// actor != nil 的分支里。
	retire, err := c.attachInvocationSessionDelegation(ctx, requestContext)
	if err != nil {
		return nil, noopProtocolV2Retire, err
	}
	return requestContext, retire, nil
}

// attachInvocationActorDelegation seals the Host-authored actor plus its
// command delegations onto one invocation context.
func (c *protocolV2Client) attachInvocationActorDelegation(
	ctx context.Context,
	requestContext *protocolv2.RequestContext,
	actor *ProtocolV2InvocationActor,
	idempotencyKey string,
) error {
	if idempotencyKey != "" {
		if !validProtocolV2InvocationIdempotencyKey(idempotencyKey) {
			return ErrProtocolV2ActorDelegationInvalid
		}
		requestContext.IdempotencyKey = idempotencyKey
	}
	if actor == nil {
		return nil
	}
	if actor.UserID <= 0 {
		return ErrProtocolV2ActorDelegationInvalid
	}
	requestContext.Actor = &protocolv2.Actor{
		UserId: actor.UserID, PermissionKeys: append([]string(nil), actor.PermissionKeys...),
	}
	if idempotencyKey == "" || !c.hostCommands {
		return nil
	}
	if c.delegations == nil {
		return ErrProtocolV2ActorDelegationUnavailable
	}
	grants, err := c.delegations.IssueProtocolV2ActorDelegations(ctx, hostapi.ProtocolV2ActorDelegationBundleRequest{
		ActorUserID: actor.UserID, PermissionKeys: append([]string(nil), actor.PermissionKeys...),
		Runtime: cloneV2Identity(c.identity), IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.Join(ErrProtocolV2ActorDelegationUnavailable, err)
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
			return ErrProtocolV2ActorDelegationInvalid
		}
		seen[key] = true
		requestContext.HostCommandDelegations = append(requestContext.HostCommandDelegations, &protocolv2.HostCommandDelegation{
			CommandId: grant.CommandID, CommandVersion: grant.CommandVersion,
			IdempotencyKey: grant.IdempotencyKey, Token: grant.Token,
		})
	}
	return nil
}

// noopProtocolV2Retire 让「没有会话委派」与「有委派」的调用方形状一致，
// 避免每个调用点都写一次 nil 判断。
func noopProtocolV2Retire() {}

// attachInvocationSessionDelegation mints the S9 session delegation for one
// Host-initiated plugin invocation and returns its retire function.
//
// Two gates, both deliberate:
//
//   - the scope must be on the Go context, which only the Host's HTTP dispatch
//     layer puts there. A background call (job, schedule, queue) therefore
//     cannot acquire a session delegation even though it uses the same client.
//   - a mint failure fails the invocation instead of silently running without
//     session state, because a plugin that loses state silently is worse than a
//     visible, retryable failure.
//
// The token itself carries no session identifier; the Host keeps the scope in
// its own mint ledger and reads it back at the service boundary.
func (c *protocolV2Client) attachInvocationSessionDelegation(
	ctx context.Context,
	requestContext *protocolv2.RequestContext,
) (func(), error) {
	if c == nil || requestContext == nil || ctx == nil {
		return noopProtocolV2Retire, nil
	}
	scope, ok := authsession.PluginSessionScopeFromContext(ctx)
	if !ok {
		return noopProtocolV2Retire, nil
	}
	if c.sessionDelegations == nil || c.identity == nil {
		return noopProtocolV2Retire, ErrProtocolV2SessionDelegationUnavailable
	}
	token, err := c.sessionDelegations.MintProtocolV2SessionDelegation(
		ctx, cloneV2Identity(c.identity), hostapi.ProtocolV2SessionScope{
			SessionFingerprint: scope.SessionFingerprint,
			IdleExpiresAt:      scope.IdleExpiresAt,
			AbsoluteExpiresAt:  scope.AbsoluteExpiresAt,
			ActorUserID:        scope.ActorUserID,
		},
	)
	if err != nil {
		if ctx.Err() != nil {
			return noopProtocolV2Retire, ctx.Err()
		}
		return noopProtocolV2Retire, errors.Join(ErrProtocolV2SessionDelegationUnavailable, err)
	}
	requestContext.HostSessionDelegation = &protocolv2.HostSessionDelegation{Token: token}
	issuer := c.sessionDelegations
	return func() { _ = issuer.RetireProtocolV2SessionDelegation(token) }, nil
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

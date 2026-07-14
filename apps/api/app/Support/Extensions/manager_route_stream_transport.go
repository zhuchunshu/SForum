package extensionsruntime

import (
	"context"
	"fmt"
)

type exactRouteStreamInstanceInvoker interface {
	OpenRouteStreamInstance(context.Context, RuntimeInstanceIdentity, ProtocolV2RouteStreamRequest) (*ProtocolV2RouteStream, error)
}

// OpenRouteStreamInstance delegates only after the caller has acquired the
// exact runtime's route admission lease. The stream lifetime therefore remains
// visible to drain and upgrade coordination without double-counting leases.
func (m *Manager) OpenRouteStreamInstance(
	ctx context.Context,
	identity RuntimeInstanceIdentity,
	request ProtocolV2RouteStreamRequest,
) (*ProtocolV2RouteStream, error) {
	if m == nil || ctx == nil {
		return nil, ErrProtocolV2RouteStreamInvalid
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return nil, err
	}
	invoker, ok := m.starter.(exactRouteStreamInstanceInvoker)
	if !ok {
		return nil, ErrProtocolV2RouteUnsupported
	}
	stream, err := invoker.OpenRouteStreamInstance(ctx, identity, request)
	if err != nil {
		return nil, fmt.Errorf("open exact route stream %s/%s: %w", identity.ExtensionID, identity.InstanceID, err)
	}
	return stream, nil
}

// OpenRouteStreamInstance never re-selects the active runtime. A retained
// source process and its replacement may both drain streams under distinct
// exact identities during a rolling upgrade.
func (s *ProtocolStarter) OpenRouteStreamInstance(
	ctx context.Context,
	identity RuntimeInstanceIdentity,
	request ProtocolV2RouteStreamRequest,
) (*ProtocolV2RouteStream, error) {
	if s == nil || ctx == nil {
		return nil, ErrProtocolV2RouteStreamInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return nil, err
	}
	var v2 *protocolV2Client
	if err := func() error {
		unlock := s.lockExtensionLifecycle(identity.ExtensionID)
		defer unlock()
		instance := s.protocolInstance(identity)
		if instance == nil {
			return protocolInstanceNotFound(identity)
		}
		if instance.protocolVersion != 2 {
			return ErrProtocolV2RouteUnsupported
		}
		var ok bool
		v2, ok = instance.protocol.(*protocolV2Client)
		if !ok || v2.identity == nil || v2.identity.GetExtensionId() != identity.ExtensionID ||
			v2.identity.GetInstanceId() != identity.InstanceID || v2.identity.GetExtensionVersion() != instance.extensionVersion ||
			v2.identity.GetArtifactDigest() != instance.artifactDigest {
			return fmt.Errorf("%w: exact runtime identity drift", ErrProtocolV2RouteStreamInvalid)
		}
		s.mu.Lock()
		s.recordProtocolCallLocked(identity.ExtensionID)
		s.mu.Unlock()
		return nil
	}(); err != nil {
		return nil, err
	}
	return v2.OpenRouteStreamContext(ctx, request)
}

var _ exactRouteStreamInstanceInvoker = (*ProtocolStarter)(nil)

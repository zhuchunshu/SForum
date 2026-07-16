package extensionsruntime

import (
	"context"
	"fmt"
)

type exactRouteInstanceInvoker interface {
	InvokeRouteInstance(context.Context, RuntimeInstanceIdentity, ProtocolV2RouteRequest) (ProtocolV2RouteResponse, error)
}

// InvokeRouteInstance dispatches through the exact process selected by a
// caller that already owns the Manager route admission lease. It must not open
// another lease, otherwise drain accounting becomes double-counted.
func (m *Manager) InvokeRouteInstance(
	ctx context.Context,
	identity RuntimeInstanceIdentity,
	request ProtocolV2RouteRequest,
) (ProtocolV2RouteResponse, error) {
	if m == nil || ctx == nil {
		return ProtocolV2RouteResponse{}, ErrProtocolV2RouteInvalid
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return ProtocolV2RouteResponse{}, err
	}
	invoker, ok := m.starter.(exactRouteInstanceInvoker)
	if !ok {
		return ProtocolV2RouteResponse{}, ErrProtocolV2RouteUnsupported
	}
	response, err := invoker.InvokeRouteInstance(ctx, identity, request)
	if err != nil {
		return ProtocolV2RouteResponse{}, fmt.Errorf("invoke exact route runtime %s/%s: %w", identity.ExtensionID, identity.InstanceID, err)
	}
	return response, nil
}

// InvokeRouteInstance never selects the active process as a fallback. Staged,
// published, and retained instances remain distinguishable by exact identity.
func (s *ProtocolStarter) InvokeRouteInstance(
	ctx context.Context,
	identity RuntimeInstanceIdentity,
	request ProtocolV2RouteRequest,
) (ProtocolV2RouteResponse, error) {
	if s == nil || ctx == nil {
		return ProtocolV2RouteResponse{}, ErrProtocolV2RouteInvalid
	}
	if err := ctx.Err(); err != nil {
		return ProtocolV2RouteResponse{}, err
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return ProtocolV2RouteResponse{}, err
	}
	// Lifecycle mutation is serialized only while resolving the exact process.
	// The Manager admission lease held by the caller protects the network call;
	// retaining this mutex across gRPC would serialize every route for a plugin.
	var v2 *protocolV2Client
	if err := func() error {
		unlock, err := s.lockExtensionLifecycleContext(ctx, identity.ExtensionID)
		if err != nil {
			return err
		}
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
			return fmt.Errorf("%w: exact runtime identity drift", ErrProtocolV2RouteInvalid)
		}
		s.mu.Lock()
		s.recordProtocolCallLocked(identity.ExtensionID)
		s.mu.Unlock()
		return nil
	}(); err != nil {
		return ProtocolV2RouteResponse{}, err
	}
	return v2.InvokeRouteContext(ctx, request)
}

var _ exactRouteInstanceInvoker = (*ProtocolStarter)(nil)

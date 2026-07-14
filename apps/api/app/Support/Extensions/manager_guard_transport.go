package extensionsruntime

import (
	"context"
	"fmt"
)

type exactGuardInstanceInvoker interface {
	InvokeGuardInstance(context.Context, RuntimeInstanceIdentity, ProtocolV2GuardRequest) error
}

// InvokeGuardInstance uses only the exact process selected by the immutable
// Route Registry step. The caller owns the separate guard admission lease.
func (m *Manager) InvokeGuardInstance(
	ctx context.Context,
	identity RuntimeInstanceIdentity,
	request ProtocolV2GuardRequest,
) error {
	if m == nil || ctx == nil {
		return ErrProtocolV2GuardInvalid
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return err
	}
	invoker, ok := m.starter.(exactGuardInstanceInvoker)
	if !ok {
		return ErrProtocolV2RouteUnsupported
	}
	if err := invoker.InvokeGuardInstance(ctx, identity, request); err != nil {
		return fmt.Errorf("invoke exact guard runtime %s/%s: %w", identity.ExtensionID, identity.InstanceID, err)
	}
	return nil
}

func (s *ProtocolStarter) InvokeGuardInstance(
	ctx context.Context,
	identity RuntimeInstanceIdentity,
	request ProtocolV2GuardRequest,
) error {
	if s == nil || ctx == nil {
		return ErrProtocolV2GuardInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return err
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
			v2.identity.GetInstanceId() != identity.InstanceID ||
			v2.identity.GetExtensionVersion() != instance.extensionVersion ||
			v2.identity.GetArtifactDigest() != instance.artifactDigest {
			return fmt.Errorf("%w: exact runtime identity drift", ErrProtocolV2GuardInvalid)
		}
		s.mu.Lock()
		s.recordProtocolCallLocked(identity.ExtensionID)
		s.mu.Unlock()
		return nil
	}(); err != nil {
		return err
	}
	return v2.InvokeGuardContext(ctx, request)
}

var _ exactGuardInstanceInvoker = (*ProtocolStarter)(nil)

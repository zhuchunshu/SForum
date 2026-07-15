package extensionsruntime

import (
	"context"
	"fmt"
)

func (s *ProtocolStarter) InvokeAdminSurface(
	ctx context.Context,
	identity RuntimeInstanceIdentity,
	contract AdminSurfaceContract,
	input map[string]any,
) (map[string]any, error) {
	if s == nil || ctx == nil {
		return nil, ErrAdminSurfaceRegistryInvalid
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
			return ErrProtocolInstanceUnsupported
		}
		var ok bool
		v2, ok = instance.protocol.(*protocolV2Client)
		if !ok || v2.identity == nil || v2.identity.GetExtensionId() != identity.ExtensionID ||
			v2.identity.GetInstanceId() != identity.InstanceID ||
			contract.ExtensionID != identity.ExtensionID || contract.InstanceID != identity.InstanceID ||
			contract.ExtensionVersion != instance.extensionVersion || contract.ArtifactDigest != instance.artifactDigest {
			return fmt.Errorf("%w: exact runtime identity drift", ErrAdminSurfaceRuntimeStale)
		}
		s.mu.Lock()
		s.recordProtocolCallLocked(identity.ExtensionID)
		s.mu.Unlock()
		return nil
	}(); err != nil {
		return nil, err
	}
	return v2.invokeAdminSurface(ctx, contract, input)
}

var _ adminSurfaceRuntime = (*ProtocolStarter)(nil)

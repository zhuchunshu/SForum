package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

var (
	ErrAdminSurfaceNotInvokable = errors.New("admin surface has no typed runtime handler")
	ErrAdminSurfaceRuntimeStale = errors.New("admin surface runtime contract is stale")
)

type AdminSurfaceInvocation struct {
	SurfaceID       string
	ContractVersion string
	Input           map[string]any
}

type AdminSurfaceInvocationResult struct {
	Contract AdminSurfaceContract
	Output   map[string]any
}

type adminSurfaceRuntime interface {
	InvokeAdminSurface(
		context.Context,
		RuntimeInstanceIdentity,
		AdminSurfaceContract,
		map[string]any,
	) (map[string]any, error)
}

// InvokeAdminSurface resolves one exact Registry contract and holds runtime
// admission while the typed props/result round trip crosses Protocol V2.
func (m *Manager) InvokeAdminSurface(
	ctx context.Context,
	input AdminSurfaceInvocation,
) (AdminSurfaceInvocationResult, error) {
	if m == nil || ctx == nil || m.hooks == nil || m.hooks.adminSurfaces == nil {
		return AdminSurfaceInvocationResult{}, ErrAdminSurfaceRegistryInvalid
	}
	contract, err := m.hooks.adminSurfaces.Resolve(input.SurfaceID)
	if err != nil {
		return AdminSurfaceInvocationResult{}, err
	}
	if version := strings.TrimSpace(input.ContractVersion); version == "" || version != contract.ContractVersion {
		return AdminSurfaceInvocationResult{}, ErrAdminSurfaceRuntimeStale
	}
	if strings.TrimSpace(contract.Handler) == "" || strings.TrimSpace(contract.Schema) == "" {
		return AdminSurfaceInvocationResult{}, ErrAdminSurfaceNotInvokable
	}
	if err := m.hooks.adminSurfaces.ValidateDocument(contract, input.Input); err != nil {
		return AdminSurfaceInvocationResult{}, err
	}
	runtime, ok := m.starter.(adminSurfaceRuntime)
	if !ok {
		return AdminSurfaceInvocationResult{}, ErrProtocolInstanceUnsupported
	}
	identity := RuntimeInstanceIdentity{ExtensionID: contract.ExtensionID, InstanceID: contract.InstanceID}
	lease, err := m.AcquireRuntimeCall(ctx, identity, RuntimeCallAdminSurface)
	if err != nil {
		return AdminSurfaceInvocationResult{}, err
	}
	defer lease.Release()
	props, err := cloneHookDocument(input.Input)
	if err != nil {
		return AdminSurfaceInvocationResult{}, fmt.Errorf("clone admin surface props: %w", err)
	}
	output, err := runtime.InvokeAdminSurface(lease.Context, identity, contract, props)
	if err != nil {
		return AdminSurfaceInvocationResult{}, err
	}
	if err := m.hooks.adminSurfaces.ValidateDocument(contract, output); err != nil {
		return AdminSurfaceInvocationResult{}, err
	}
	result, err := cloneHookDocument(output)
	if err != nil {
		return AdminSurfaceInvocationResult{}, fmt.Errorf("clone admin surface result: %w", err)
	}
	return AdminSurfaceInvocationResult{Contract: contract, Output: result}, nil
}

func validateFrozenAdminSurface(
	contract AdminSurfaceContract,
	declarations []extensions.ManifestAdminSurface,
) error {
	for _, declaration := range declarations {
		if declaration.ID != contract.ID {
			continue
		}
		if declaration.ContractVersion != contract.ContractVersion || declaration.Kind != contract.Kind ||
			declaration.Action != contract.Action || declaration.TargetID != contract.TargetID ||
			declaration.Label != contract.Label || declaration.Handler != contract.Handler ||
			declaration.Schema != contract.Schema || declaration.Permission != contract.Permission ||
			declaration.Priority != contract.Priority {
			return ErrAdminSurfaceRuntimeStale
		}
		return nil
	}
	return fmt.Errorf("%w: surface %s", ErrAdminSurfaceRuntimeStale, strings.TrimSpace(contract.ID))
}

package extensionsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

var (
	ErrLifecycleHostGateInvalid     = errors.New("extension lifecycle Host gate is invalid")
	ErrLifecycleHostBoundaryMissing = errors.New("extension lifecycle Host boundary is unavailable")
)

// LifecycleHostRuntime is the exact-instance process surface used by Host
// lifecycle gates. Manager implements it directly.
type LifecycleHostRuntime interface {
	StageRuntimeInstance(context.Context, extensions.Extension) (RuntimeInstanceSnapshot, error)
	InspectRuntimeInstance(RuntimeInstanceIdentity) (RuntimeInstanceSnapshot, error)
	ActiveRuntimeInstance(string) (RuntimeInstanceSnapshot, error)
	HealthRuntimeInstance(context.Context, RuntimeInstanceIdentity) (ProtocolRuntimeInstanceSnapshot, error)
	BeginDrain(RuntimeInstanceIdentity) (RuntimeAdmissionSnapshot, error)
	WaitDrain(context.Context, RuntimeInstanceIdentity) error
	ForceDrain(RuntimeInstanceIdentity, error) (RuntimeAdmissionSnapshot, error)
}

// LifecycleHostBoundaryResult deliberately excludes runtime bindings. Exact
// process selection belongs to this dispatcher; database/registry/job/removal
// boundaries may only return their durable checkpoint and typed result.
type LifecycleHostBoundaryResult struct {
	Checkpoint     string
	ResultDocument json.RawMessage
}

// LifecycleHostBoundary executes Host-owned boundaries that require composed
// database, registry, job, schedule, or removal state. A missing implementation
// fails closed instead of treating an unimplemented safety gate as a no-op.
type LifecycleHostBoundary interface {
	RunLifecycleHostBoundary(context.Context, extensions.LifecycleCoordinatorGateRequest) (LifecycleHostBoundaryResult, error)
}

// ExactLifecycleCoordinatorHost dispatches every Host gate in the authoritative
// lifecycle path. It owns exact process preparation, health, and drain; composed
// publication/migration/cleanup remains an explicit mandatory boundary.
type ExactLifecycleCoordinatorHost struct {
	runtime  LifecycleHostRuntime
	boundary LifecycleHostBoundary
}

func NewExactLifecycleCoordinatorHost(runtime LifecycleHostRuntime, boundary LifecycleHostBoundary) *ExactLifecycleCoordinatorHost {
	return &ExactLifecycleCoordinatorHost{runtime: runtime, boundary: boundary}
}

func (h *ExactLifecycleCoordinatorHost) RunLifecycleHostGate(
	ctx context.Context,
	request extensions.LifecycleCoordinatorGateRequest,
) (extensions.LifecycleCoordinatorGateResult, error) {
	if h == nil || h.runtime == nil {
		return extensions.LifecycleCoordinatorGateResult{}, fmt.Errorf("%w: exact runtime is required", ErrLifecycleHostGateInvalid)
	}
	if ctx == nil {
		return extensions.LifecycleCoordinatorGateResult{}, fmt.Errorf("%w: context is required", ErrLifecycleHostGateInvalid)
	}
	if err := ctx.Err(); err != nil {
		return extensions.LifecycleCoordinatorGateResult{}, err
	}
	if err := validateLifecycleHostGateRequest(request); err != nil {
		return extensions.LifecycleCoordinatorGateResult{}, err
	}

	switch lookupLifecycleHostGateKind(request.Operation, request.Position) {
	case lifecycleHostGatePrepare:
		return h.prepareExactRuntimes(ctx, request)
	case lifecycleHostGateStarting:
		return h.inspectTarget(request)
	case lifecycleHostGateHealthy:
		return h.healthTarget(ctx, request)
	case lifecycleHostGateDrain:
		return h.drainSource(ctx, request)
	case lifecycleHostGateBoundary:
		return h.runBoundary(ctx, request)
	default:
		return extensions.LifecycleCoordinatorGateResult{}, fmt.Errorf(
			"%w: unsupported %s position %d", ErrLifecycleHostGateInvalid, request.Operation, request.Position,
		)
	}
}

type lifecycleHostGateKind uint8

const (
	lifecycleHostGateUnknown lifecycleHostGateKind = iota
	lifecycleHostGatePrepare
	lifecycleHostGateStarting
	lifecycleHostGateHealthy
	lifecycleHostGateDrain
	lifecycleHostGateBoundary
)

var lifecycleHostGateKinds = map[extensions.LifecycleMachineOperation]map[int]lifecycleHostGateKind{
	extensions.LifecycleMachineInstall: {
		0: lifecycleHostGatePrepare, 2: lifecycleHostGateBoundary, 4: lifecycleHostGateStarting,
		6: lifecycleHostGateHealthy, 7: lifecycleHostGateBoundary, 8: lifecycleHostGateBoundary,
	},
	extensions.LifecycleMachineEnable: {
		0: lifecycleHostGatePrepare, 1: lifecycleHostGateStarting, 3: lifecycleHostGateHealthy,
		4: lifecycleHostGateBoundary, 5: lifecycleHostGateBoundary,
	},
	extensions.LifecycleMachineDisable: {
		0: lifecycleHostGatePrepare, 1: lifecycleHostGateDrain, 3: lifecycleHostGateBoundary,
	},
	extensions.LifecycleMachineUpgrade: {
		0: lifecycleHostGatePrepare, 2: lifecycleHostGateDrain, 4: lifecycleHostGateBoundary,
		5: lifecycleHostGateStarting, 6: lifecycleHostGateHealthy, 7: lifecycleHostGateBoundary,
		8: lifecycleHostGateBoundary, 10: lifecycleHostGateBoundary,
	},
	extensions.LifecycleMachineRollback: {
		0: lifecycleHostGatePrepare, 1: lifecycleHostGateDrain, 2: lifecycleHostGateStarting,
		4: lifecycleHostGateHealthy, 5: lifecycleHostGateBoundary, 6: lifecycleHostGateBoundary,
	},
	extensions.LifecycleMachineUninstall: {
		0: lifecycleHostGatePrepare, 2: lifecycleHostGateDrain,
		3: lifecycleHostGateBoundary, 6: lifecycleHostGateBoundary,
	},
}

func lookupLifecycleHostGateKind(operation extensions.LifecycleMachineOperation, position int) lifecycleHostGateKind {
	return lifecycleHostGateKinds[operation][position]
}

func (h *ExactLifecycleCoordinatorHost) prepareExactRuntimes(
	ctx context.Context,
	request extensions.LifecycleCoordinatorGateRequest,
) (extensions.LifecycleCoordinatorGateResult, error) {
	result := extensions.LifecycleCoordinatorGateResult{
		Checkpoint: request.Checkpoint, RevalidationPolicy: extensions.LifecycleGateRevalidationRequired,
	}
	if lifecycleHostRequiresSource(request.Operation) {
		source := lifecycleHostSourceExtension(request)
		snapshot, err := h.ensureRuntime(ctx, source, request.SourceBinding)
		if err != nil {
			return extensions.LifecycleCoordinatorGateResult{}, fmt.Errorf("prepare source runtime: %w", err)
		}
		result.SourceBinding = lifecycleHostBinding(source, snapshot)
	}
	if lifecycleHostRequiresTarget(request.Operation) {
		target := request.TargetExtension
		snapshot, err := h.ensureRuntime(ctx, target, request.TargetBinding)
		if err != nil {
			return extensions.LifecycleCoordinatorGateResult{}, fmt.Errorf("prepare target runtime: %w", err)
		}
		result.TargetBinding = lifecycleHostBinding(target, snapshot)
	}
	return result, nil
}

func (h *ExactLifecycleCoordinatorHost) ensureRuntime(
	ctx context.Context,
	extension extensions.Extension,
	binding extensions.LifecycleRuntimeBinding,
) (RuntimeInstanceSnapshot, error) {
	if binding.RuntimeInstanceID != "" {
		identity := RuntimeInstanceIdentity{ExtensionID: extension.ID, InstanceID: binding.RuntimeInstanceID}
		snapshot, err := h.runtime.InspectRuntimeInstance(identity)
		if err == nil {
			if err := validateLifecycleHostRuntimeSnapshot("persisted", snapshot, extension, identity); err != nil {
				return RuntimeInstanceSnapshot{}, err
			}
			return snapshot, nil
		}
		if !errors.Is(err, ErrRuntimeInstanceNotFound) {
			return RuntimeInstanceSnapshot{}, err
		}
	}
	if active, err := h.runtime.ActiveRuntimeInstance(extension.ID); err == nil {
		if runtimeInstanceMatchesExtension(active, extension) {
			if err := validateLifecycleHostRuntimeSnapshot("active", active, extension, RuntimeInstanceIdentity{}); err != nil {
				return RuntimeInstanceSnapshot{}, err
			}
			return active, nil
		}
	} else if !errors.Is(err, ErrRuntimeInstanceNotFound) {
		return RuntimeInstanceSnapshot{}, err
	}
	staged, err := h.runtime.StageRuntimeInstance(ctx, extension)
	if err != nil {
		return RuntimeInstanceSnapshot{}, err
	}
	if err := validateLifecycleHostRuntimeSnapshot("staged", staged, extension, RuntimeInstanceIdentity{}); err != nil {
		return RuntimeInstanceSnapshot{}, err
	}
	return staged, nil
}

func (h *ExactLifecycleCoordinatorHost) inspectTarget(
	request extensions.LifecycleCoordinatorGateRequest,
) (extensions.LifecycleCoordinatorGateResult, error) {
	identity, err := lifecycleHostIdentity(request.TargetBinding, request.TargetExtension)
	if err != nil {
		return extensions.LifecycleCoordinatorGateResult{}, err
	}
	snapshot, err := h.runtime.InspectRuntimeInstance(identity)
	if err != nil {
		return extensions.LifecycleCoordinatorGateResult{}, err
	}
	if err := validateLifecycleHostRuntimeSnapshot("starting target", snapshot, request.TargetExtension, identity); err != nil {
		return extensions.LifecycleCoordinatorGateResult{}, err
	}
	return lifecycleHostProcessResult(request), nil
}

func (h *ExactLifecycleCoordinatorHost) healthTarget(
	ctx context.Context,
	request extensions.LifecycleCoordinatorGateRequest,
) (extensions.LifecycleCoordinatorGateResult, error) {
	identity, err := lifecycleHostIdentity(request.TargetBinding, request.TargetExtension)
	if err != nil {
		return extensions.LifecycleCoordinatorGateResult{}, err
	}
	snapshot, err := h.runtime.HealthRuntimeInstance(ctx, identity)
	if err != nil {
		return extensions.LifecycleCoordinatorGateResult{}, err
	}
	if snapshot.Identity != identity || snapshot.ExtensionVersion != request.TargetExtension.Version ||
		snapshot.ArtifactDigest != request.TargetExtension.PackageDigest {
		return extensions.LifecycleCoordinatorGateResult{}, fmt.Errorf(
			"%w: healthy target changed exact runtime", ErrRuntimeInstanceConflict,
		)
	}
	if !snapshot.Healthy || !snapshot.Ready || !snapshot.ReadinessChecked {
		return extensions.LifecycleCoordinatorGateResult{}, ErrProtocolInstanceNotReady
	}
	return lifecycleHostProcessResult(request), nil
}

func (h *ExactLifecycleCoordinatorHost) drainSource(
	ctx context.Context,
	request extensions.LifecycleCoordinatorGateRequest,
) (extensions.LifecycleCoordinatorGateResult, error) {
	source := lifecycleHostSourceExtension(request)
	identity, err := lifecycleHostIdentity(request.SourceBinding, source)
	if err != nil {
		return extensions.LifecycleCoordinatorGateResult{}, err
	}
	draining, err := h.runtime.BeginDrain(identity)
	if err != nil {
		return extensions.LifecycleCoordinatorGateResult{}, err
	}
	if err := validateLifecycleHostAdmissionSnapshot("begin drain", draining, identity); err != nil {
		return extensions.LifecycleCoordinatorGateResult{}, err
	}
	if request.Forced {
		cause := fmt.Errorf("forced uninstall operation %d", request.OperationID)
		forced, err := h.runtime.ForceDrain(identity, cause)
		if err != nil {
			return extensions.LifecycleCoordinatorGateResult{}, err
		}
		if err := validateLifecycleHostAdmissionSnapshot("force drain", forced, identity); err != nil {
			return extensions.LifecycleCoordinatorGateResult{}, err
		}
	}
	if err := h.runtime.WaitDrain(ctx, identity); err != nil {
		return extensions.LifecycleCoordinatorGateResult{}, err
	}
	return lifecycleHostProcessResult(request), nil
}

func (h *ExactLifecycleCoordinatorHost) runBoundary(
	ctx context.Context,
	request extensions.LifecycleCoordinatorGateRequest,
) (extensions.LifecycleCoordinatorGateResult, error) {
	if h.boundary == nil {
		return extensions.LifecycleCoordinatorGateResult{}, fmt.Errorf(
			"%w: %s position %d", ErrLifecycleHostBoundaryMissing, request.Operation, request.Position,
		)
	}
	result, err := h.boundary.RunLifecycleHostBoundary(ctx, request)
	if err != nil {
		return extensions.LifecycleCoordinatorGateResult{}, err
	}
	return extensions.LifecycleCoordinatorGateResult{
		Checkpoint: result.Checkpoint, ResultDocument: cloneHostGateJSON(result.ResultDocument),
		RevalidationPolicy: extensions.LifecycleGateDurable,
	}, nil
}

func lifecycleHostProcessResult(request extensions.LifecycleCoordinatorGateRequest) extensions.LifecycleCoordinatorGateResult {
	result := extensions.LifecycleCoordinatorGateResult{
		Checkpoint: request.Checkpoint, RevalidationPolicy: extensions.LifecycleGateRevalidationRequired,
	}
	if lifecycleHostRequiresSource(request.Operation) {
		result.SourceBinding = request.SourceBinding
	}
	if lifecycleHostRequiresTarget(request.Operation) {
		result.TargetBinding = request.TargetBinding
	}
	return result
}

func lifecycleHostBinding(extension extensions.Extension, snapshot RuntimeInstanceSnapshot) extensions.LifecycleRuntimeBinding {
	return extensions.LifecycleRuntimeBinding{
		ExtensionID: extension.ID, ExtensionVersion: extension.Version, PackageDigest: extension.PackageDigest,
		RuntimeInstanceID: snapshot.Identity.InstanceID, VersionID: extension.ActiveVersionID,
	}
}

func validateLifecycleHostRuntimeSnapshot(
	label string,
	snapshot RuntimeInstanceSnapshot,
	extension extensions.Extension,
	expected RuntimeInstanceIdentity,
) error {
	if !runtimeInstanceMatchesExtension(snapshot, extension) || snapshot.Identity.InstanceID == "" ||
		(expected != (RuntimeInstanceIdentity{}) && snapshot.Identity != expected) {
		return fmt.Errorf("%w: %s runtime changed exact instance or artifact", ErrRuntimeInstanceConflict, label)
	}
	return nil
}

func validateLifecycleHostAdmissionSnapshot(
	label string,
	snapshot RuntimeAdmissionSnapshot,
	expected RuntimeInstanceIdentity,
) error {
	if snapshot.Identity != expected {
		return fmt.Errorf("%w: %s returned another runtime instance", ErrRuntimeInstanceConflict, label)
	}
	return nil
}

func lifecycleHostIdentity(
	binding extensions.LifecycleRuntimeBinding,
	extension extensions.Extension,
) (RuntimeInstanceIdentity, error) {
	if err := validateExactCoordinatorBinding("Host", binding, extension, true); err != nil {
		return RuntimeInstanceIdentity{}, err
	}
	return RuntimeInstanceIdentity{ExtensionID: binding.ExtensionID, InstanceID: binding.RuntimeInstanceID}, nil
}

func lifecycleHostRequiresSource(operation extensions.LifecycleMachineOperation) bool {
	switch operation {
	case extensions.LifecycleMachineDisable, extensions.LifecycleMachineUpgrade,
		extensions.LifecycleMachineRollback, extensions.LifecycleMachineUninstall:
		return true
	default:
		return false
	}
}

func lifecycleHostRequiresTarget(operation extensions.LifecycleMachineOperation) bool {
	switch operation {
	case extensions.LifecycleMachineInstall, extensions.LifecycleMachineEnable,
		extensions.LifecycleMachineUpgrade, extensions.LifecycleMachineRollback:
		return true
	default:
		return false
	}
}

func lifecycleHostSourceExtension(request extensions.LifecycleCoordinatorGateRequest) extensions.Extension {
	if request.SourceExtension != nil {
		return *request.SourceExtension
	}
	return request.TargetExtension
}

func validateLifecycleHostGateRequest(request extensions.LifecycleCoordinatorGateRequest) error {
	if request.OperationID <= 0 || request.Attempt <= 0 || request.ActorUserID <= 0 || request.AuditEventID <= 0 {
		return fmt.Errorf("%w: operation, attempt, actor, and audit identities are required", ErrLifecycleHostGateInvalid)
	}
	path, err := extensions.RecommendedLifecyclePath(request.Operation)
	if err != nil || request.Position < 0 || request.Position >= len(path) {
		return fmt.Errorf("%w: unknown operation position", ErrLifecycleHostGateInvalid)
	}
	step := path[request.Position]
	if step.Action != "" || step.State != request.State {
		return fmt.Errorf("%w: position does not identify the requested Host gate", ErrLifecycleHostGateInvalid)
	}
	wantStepID := fmt.Sprintf("lifecycle.%s.%02d.host.%s", request.Operation, request.Position, request.State)
	if request.StepID != wantStepID {
		return fmt.Errorf("%w: stable step id %q does not match %q", ErrLifecycleHostGateInvalid, request.StepID, wantStepID)
	}
	if lookupLifecycleHostGateKind(request.Operation, request.Position) == lifecycleHostGateUnknown {
		return fmt.Errorf("%w: Host gate has no production dispatcher", ErrLifecycleHostGateInvalid)
	}
	if err := validateExactCoordinatorArtifact("target", request.TargetExtension); err != nil {
		return err
	}
	if err := validateExactCoordinatorSelectedExtension(request.Extension, request.TargetExtension); err != nil {
		return err
	}
	if err := validateLifecycleHostSource(request); err != nil {
		return err
	}
	if err := validateExactCoordinatorBinding("target", request.TargetBinding, request.TargetExtension, request.Position != 0 && lifecycleHostRequiresTarget(request.Operation)); err != nil {
		return err
	}
	source := lifecycleHostSourceExtension(request)
	if lifecycleHostRequiresSource(request.Operation) {
		if err := validateExactCoordinatorArtifact("source", source); err != nil {
			return err
		}
		if err := validateExactCoordinatorBinding("source", request.SourceBinding, source, request.Position != 0); err != nil {
			return err
		}
	}
	if err := validateExactCoordinatorRemovalContext(extensions.LifecycleCoordinatorActionRequest{
		Operation: request.Operation, RemovalMode: request.RemovalMode, Forced: request.Forced,
	}); err != nil {
		return err
	}
	if err := validateExactCoordinatorAuthority(extensions.LifecycleCoordinatorActionRequest{
		AuthorityType: request.AuthorityType, TrustGrantID: request.TrustGrantID,
		AuthoritySnapshot: request.AuthoritySnapshot, ActorUserID: request.ActorUserID,
	}, request.TargetExtension); err != nil {
		return err
	}
	return nil
}

func validateLifecycleHostSource(request extensions.LifecycleCoordinatorGateRequest) error {
	switch request.Operation {
	case extensions.LifecycleMachineInstall:
		if request.SourceExtension != nil {
			return fmt.Errorf("%w: install cannot carry a source artifact", ErrLifecycleHostGateInvalid)
		}
	case extensions.LifecycleMachineEnable, extensions.LifecycleMachineDisable, extensions.LifecycleMachineUninstall:
		if request.SourceExtension != nil {
			if err := validateExactCoordinatorSelectedExtension(*request.SourceExtension, request.TargetExtension); err != nil {
				return fmt.Errorf("%w: current operation source changed: %v", ErrLifecycleHostGateInvalid, err)
			}
		}
	case extensions.LifecycleMachineUpgrade, extensions.LifecycleMachineRollback:
		if request.SourceExtension == nil {
			return fmt.Errorf("%w: upgrade and rollback require a source artifact", ErrLifecycleHostGateInvalid)
		}
		source := *request.SourceExtension
		if source.ID != request.TargetExtension.ID || source.Type != request.TargetExtension.Type ||
			source.Source != request.TargetExtension.Source ||
			(source.Version == request.TargetExtension.Version && source.PackageDigest == request.TargetExtension.PackageDigest) {
			return fmt.Errorf("%w: source and target are not different exact versions", ErrLifecycleHostGateInvalid)
		}
	default:
		return fmt.Errorf("%w: unknown operation %q", ErrLifecycleHostGateInvalid, request.Operation)
	}
	return nil
}

func cloneHostGateJSON(value json.RawMessage) json.RawMessage {
	if value == nil {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}

var _ extensions.LifecycleCoordinatorHost = (*ExactLifecycleCoordinatorHost)(nil)

package extensions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type lifecycleCoordinatorGateResultEnvelope struct {
	Schema string                         `json:"schema"`
	Result LifecycleCoordinatorGateResult `json:"result"`
}

func hydrateLifecycleCoordinatorBindings(
	machine *LifecycleStateMachine,
	target Extension,
	source *Extension,
) error {
	if machine == nil {
		return fmt.Errorf("%w: lifecycle machine is required", ErrLifecycleCoordinatorInvalid)
	}
	targetBinding := lifecycleRuntimeBindingFor(target)
	mergedTarget, err := mergeLifecycleRuntimeBinding(machine.TargetBinding, targetBinding)
	if err != nil {
		return fmt.Errorf("%w: target binding changed: %v", ErrLifecycleCoordinatorInvalid, err)
	}
	machine.TargetBinding = mergedTarget

	if source == nil && lifecycleOperationUsesCurrentArtifact(machine.Operation) {
		source = &target
	}
	if source != nil {
		mergedSource, err := mergeLifecycleRuntimeBinding(machine.SourceBinding, lifecycleRuntimeBindingFor(*source))
		if err != nil {
			return fmt.Errorf("%w: source binding changed: %v", ErrLifecycleCoordinatorInvalid, err)
		}
		machine.SourceBinding = mergedSource
	}
	return nil
}

func validateLifecycleSourceArtifact(
	operation LifecycleMachineOperation,
	target Extension,
	source *Extension,
) error {
	if target.Type != TypePlugin || target.ID == "" || target.Version == "" || target.PackageDigest == "" {
		return fmt.Errorf("%w: target artifact is not an exact plugin binding", ErrLifecycleCoordinatorInvalid)
	}
	switch operation {
	case LifecycleMachineInstall:
		if source != nil {
			return fmt.Errorf("%w: install cannot declare a source artifact", ErrLifecycleCoordinatorInvalid)
		}
		return nil
	case LifecycleMachineEnable, LifecycleMachineDisable, LifecycleMachineUninstall:
		if source == nil {
			return nil
		}
		if !sameLifecycleExactArtifact(*source, target) {
			return fmt.Errorf("%w: current-artifact operation source does not match the exact target artifact", ErrLifecycleCoordinatorInvalid)
		}
		return nil
	case LifecycleMachineUpgrade, LifecycleMachineRollback:
		if source == nil {
			return fmt.Errorf("%w: upgrade and rollback require the exact source artifact", ErrLifecycleCoordinatorInvalid)
		}
		if source.ID != target.ID || source.Type != TypePlugin || source.Type != target.Type ||
			source.Source != target.Source || strings.TrimSpace(lifecycleManifestContract(*source)) == "" ||
			strings.TrimSpace(lifecycleManifestContract(target)) == "" ||
			source.Version == "" || source.PackageDigest == "" {
			return fmt.Errorf("%w: source artifact is not a compatible exact plugin binding", ErrLifecycleCoordinatorInvalid)
		}
		if sameLifecycleStableVersion(*source, target) {
			return fmt.Errorf("%w: upgrade and rollback require a different exact version artifact", ErrLifecycleCoordinatorInvalid)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown lifecycle operation %q", ErrLifecycleCoordinatorInvalid, operation)
	}
}

func sameLifecycleExactArtifact(left, right Extension) bool {
	return left.ID == right.ID && left.Type == TypePlugin && left.Type == right.Type &&
		left.Source == right.Source && lifecycleManifestContract(left) == lifecycleManifestContract(right) &&
		sameLifecycleExactVersion(left, right)
}

func sameLifecycleExactVersion(left, right Extension) bool {
	return sameLifecycleStableVersion(left, right) && left.ActiveVersionID == right.ActiveVersionID
}

func sameLifecycleStableVersion(left, right Extension) bool {
	return left.Version == right.Version && left.PackageDigest == right.PackageDigest
}

func lifecycleManifestContract(extension Extension) string {
	if extension.Manifest.Lifecycle == nil {
		return ""
	}
	return extension.Manifest.Lifecycle.ContractVersion
}

func lifecycleOperationUsesCurrentArtifact(operation LifecycleMachineOperation) bool {
	switch operation {
	case LifecycleMachineEnable, LifecycleMachineDisable, LifecycleMachineUninstall:
		return true
	default:
		return false
	}
}

func lifecycleRuntimeBindingsReady(machine LifecycleStateMachine) bool {
	switch machine.Operation {
	case LifecycleMachineInstall, LifecycleMachineEnable:
		return machine.TargetBinding.RuntimeInstanceID != ""
	case LifecycleMachineDisable, LifecycleMachineUninstall:
		return machine.SourceBinding.RuntimeInstanceID != ""
	case LifecycleMachineUpgrade, LifecycleMachineRollback:
		return machine.SourceBinding.RuntimeInstanceID != "" && machine.TargetBinding.RuntimeInstanceID != ""
	default:
		return false
	}
}

func lifecycleRuntimeBindingFor(extension Extension) LifecycleRuntimeBinding {
	return LifecycleRuntimeBinding{
		ExtensionID: extension.ID, ExtensionVersion: extension.Version,
		PackageDigest: extension.PackageDigest, VersionID: extension.ActiveVersionID,
	}
}

func lifecycleOptionalRuntimeBinding(extension *Extension) LifecycleRuntimeBinding {
	if extension == nil {
		return LifecycleRuntimeBinding{}
	}
	return lifecycleRuntimeBindingFor(*extension)
}

func mergeLifecycleRuntimeBinding(current, next LifecycleRuntimeBinding) (LifecycleRuntimeBinding, error) {
	if lifecycleRuntimeBindingEmpty(next) {
		return current, nil
	}
	if current.ExtensionID != "" && next.ExtensionID != "" && current.ExtensionID != next.ExtensionID {
		return LifecycleRuntimeBinding{}, fmt.Errorf("extension id %q does not match %q", next.ExtensionID, current.ExtensionID)
	}
	if current.ExtensionVersion != "" && next.ExtensionVersion != "" && current.ExtensionVersion != next.ExtensionVersion {
		return LifecycleRuntimeBinding{}, fmt.Errorf("version %q does not match %q", next.ExtensionVersion, current.ExtensionVersion)
	}
	if current.PackageDigest != "" && next.PackageDigest != "" && current.PackageDigest != next.PackageDigest {
		return LifecycleRuntimeBinding{}, fmt.Errorf("digest %q does not match %q", next.PackageDigest, current.PackageDigest)
	}
	if current.VersionID != 0 && next.VersionID != 0 && current.VersionID != next.VersionID {
		return LifecycleRuntimeBinding{}, fmt.Errorf("version id %d does not match %d", next.VersionID, current.VersionID)
	}
	if next.ExtensionID != "" {
		current.ExtensionID = next.ExtensionID
	}
	if next.ExtensionVersion != "" {
		current.ExtensionVersion = next.ExtensionVersion
	}
	if next.PackageDigest != "" {
		current.PackageDigest = next.PackageDigest
	}
	if next.VersionID != 0 {
		current.VersionID = next.VersionID
	}
	if next.RuntimeInstanceID != "" {
		current.RuntimeInstanceID = next.RuntimeInstanceID
	}
	return current, nil
}

func lifecycleRuntimeBindingEmpty(binding LifecycleRuntimeBinding) bool {
	return binding == (LifecycleRuntimeBinding{})
}

func applyLifecycleHostGateResult(
	machine LifecycleStateMachine,
	stepID string,
	position int,
	result LifecycleCoordinatorGateResult,
) (LifecycleStateMachine, error) {
	original := machine
	previousSourceInstance := machine.SourceBinding.RuntimeInstanceID
	previousTargetInstance := machine.TargetBinding.RuntimeInstanceID
	var err error
	machine.SourceBinding, err = mergeLifecycleRuntimeBinding(machine.SourceBinding, result.SourceBinding)
	if err != nil {
		return original, fmt.Errorf("%w: Host changed source binding: %v", ErrLifecycleCoordinatorInvalid, err)
	}
	machine.TargetBinding, err = mergeLifecycleRuntimeBinding(machine.TargetBinding, result.TargetBinding)
	if err != nil {
		return original, fmt.Errorf("%w: Host changed target binding: %v", ErrLifecycleCoordinatorInvalid, err)
	}
	if err := validateLifecycleRuntimeInstanceBinding(machine.SourceBinding); err != nil {
		return original, err
	}
	if err := validateLifecycleRuntimeInstanceBinding(machine.TargetBinding); err != nil {
		return original, err
	}
	runtimeBindingChanged :=
		(result.SourceBinding.RuntimeInstanceID != "" && result.SourceBinding.RuntimeInstanceID != previousSourceInstance) ||
			(result.TargetBinding.RuntimeInstanceID != "" && result.TargetBinding.RuntimeInstanceID != previousTargetInstance)
	if runtimeBindingChanged && result.RevalidationPolicy != LifecycleGateRevalidationRequired {
		return original, fmt.Errorf("%w: a new runtime instance requires explicit revalidation", ErrLifecycleCoordinatorInvalid)
	}
	switch result.RevalidationPolicy {
	case "":
	case LifecycleGateRevalidationRequired:
		if err := validateLifecycleRequiredRuntimeBindings(machine.Operation, result); err != nil {
			return original, err
		}
		machine.Revalidation = LifecycleGateRevalidation{StepID: stepID, Position: position}
	case LifecycleGateDurable:
		// A durable Host gate does not make a process-local runtime durable. The
		// existing marker remains until the operation reaches terminal state.
	default:
		return original, fmt.Errorf("%w: unknown Host revalidation policy %q", ErrLifecycleCoordinatorInvalid, result.RevalidationPolicy)
	}
	if position == 0 {
		machine.HostSideEffectsStarted = true
	}
	return machine, nil
}

func validateLifecycleRequiredRuntimeBindings(
	operation LifecycleMachineOperation,
	result LifecycleCoordinatorGateResult,
) error {
	requireSource := false
	requireTarget := false
	switch operation {
	case LifecycleMachineInstall, LifecycleMachineEnable:
		requireTarget = true
	case LifecycleMachineDisable, LifecycleMachineUninstall:
		requireSource = true
	case LifecycleMachineUpgrade, LifecycleMachineRollback:
		requireSource, requireTarget = true, true
	default:
		return fmt.Errorf("%w: unknown lifecycle operation %q", ErrLifecycleCoordinatorInvalid, operation)
	}
	if requireSource && result.SourceBinding.RuntimeInstanceID == "" {
		return fmt.Errorf("%w: required revalidation did not prove the source runtime instance", ErrLifecycleCoordinatorInvalid)
	}
	if requireTarget && result.TargetBinding.RuntimeInstanceID == "" {
		return fmt.Errorf("%w: required revalidation did not prove the target runtime instance", ErrLifecycleCoordinatorInvalid)
	}
	return nil
}

func validateLifecycleRuntimeInstanceBinding(binding LifecycleRuntimeBinding) error {
	if binding.RuntimeInstanceID == "" {
		return nil
	}
	if binding.RuntimeInstanceID != strings.TrimSpace(binding.RuntimeInstanceID) ||
		binding.ExtensionID == "" || binding.ExtensionVersion == "" || binding.PackageDigest == "" {
		return fmt.Errorf("%w: runtime instance must retain its exact artifact binding", ErrLifecycleCoordinatorInvalid)
	}
	return nil
}

func encodeLifecycleHostGateResult(result LifecycleCoordinatorGateResult) (json.RawMessage, error) {
	if result.Checkpoint != strings.TrimSpace(result.Checkpoint) {
		return nil, fmt.Errorf("%w: Host checkpoint must be stable", ErrLifecycleCoordinatorInvalid)
	}
	if len(bytes.TrimSpace(result.ResultDocument)) > 0 && !json.Valid(result.ResultDocument) {
		return nil, fmt.Errorf("%w: Host result document must be JSON", ErrLifecycleCoordinatorInvalid)
	}
	value, err := json.Marshal(lifecycleCoordinatorGateResultEnvelope{
		Schema: lifecycleCoordinatorGateResultSchema, Result: result,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: encode Host gate result: %v", ErrLifecycleCoordinatorInvalid, err)
	}
	return value, nil
}

func decodeLifecycleHostGateResult(value json.RawMessage) (LifecycleCoordinatorGateResult, bool, error) {
	if len(bytes.TrimSpace(value)) == 0 {
		return LifecycleCoordinatorGateResult{}, false, nil
	}
	var envelope lifecycleCoordinatorGateResultEnvelope
	if err := json.Unmarshal(value, &envelope); err != nil {
		return LifecycleCoordinatorGateResult{}, false, fmt.Errorf("%w: decode Host gate result: %v", ErrLifecycleCoordinatorInvalid, err)
	}
	if envelope.Schema == "" {
		// Host attempts written before typed results are durable and replayable,
		// but they carry no ephemeral binding promise.
		return LifecycleCoordinatorGateResult{}, false, nil
	}
	if envelope.Schema != lifecycleCoordinatorGateResultSchema {
		return LifecycleCoordinatorGateResult{}, false, fmt.Errorf("%w: unsupported Host gate result %q", ErrLifecycleCoordinatorInvalid, envelope.Schema)
	}
	return envelope.Result, true, nil
}

func lifecycleActionRuntimeRole(action LifecycleMachineAction) LifecycleCoordinatorRuntimeRole {
	switch action {
	case LifecycleMachineDisableAction,
		LifecycleMachineUpgradeBefore,
		LifecycleMachineUninstallPlan,
		LifecycleMachineUninstallStep,
		LifecycleMachineUninstallAfter:
		return LifecycleRuntimeSource
	default:
		return LifecycleRuntimeTarget
	}
}

func lifecycleActionExtension(input LifecycleCoordinatorRunInput, role LifecycleCoordinatorRuntimeRole) Extension {
	if role == LifecycleRuntimeSource && input.SourceExtension != nil {
		return *input.SourceExtension
	}
	return input.Extension
}

func lifecycleSourceExtension(input LifecycleCoordinatorRunInput) *Extension {
	if input.SourceExtension != nil {
		value := *input.SourceExtension
		return &value
	}
	return nil
}

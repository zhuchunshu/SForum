package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

var ErrLifecycleAuthorityNotFound = errors.New("extensions: successful lifecycle authority not found")

// LifecycleCoordinatorRunner keeps Models independent from the production
// runtime/Host adapters assembled under Support/Extensions.
type LifecycleCoordinatorRunner interface {
	Run(context.Context, LifecycleCoordinatorRunInput) (LifecycleCoordinatorRunResult, error)
}

// LifecycleStaticPreflight is deliberately a callback: bootstrap may adapt the
// Support implementation without Models importing the process-owning package.
type LifecycleStaticPreflight func(
	context.Context,
	LifecycleMachineOperation,
	*Extension,
	Extension,
) error

// LifecycleAuthorityRepository exposes only the immutable successful snapshot
// needed for deactivation and historical rollback.
type LifecycleAuthorityRepository interface {
	LastSuccessfulLifecycleAuthority(context.Context, ExactExtensionVersionInput) (LifecycleAuthoritySnapshot, error)
	OperationByIdempotencyKey(context.Context, string, string) (LifecycleOperation, error)
}

type lifecycleServiceRequest struct {
	operation         LifecycleMachineOperation
	source            *Extension
	target            Extension
	idempotencyKey    string
	confirmationToken string
	frozenAuthority   bool
}

func usesLifecycleV2(extension Extension) bool {
	return extension.Type == TypePlugin && extension.Manifest.Backend.ProtocolVersion == 2 &&
		extension.Manifest.Lifecycle != nil && strings.TrimSpace(extension.Manifest.Lifecycle.ContractVersion) != ""
}

func (s *Service) enableLifecycleV2(
	ctx context.Context,
	actor identity.Actor,
	extension Extension,
	input EnableInput,
) (Extension, error) {
	if replayed, found, err := s.replayLifecycleV2(ctx, actor, extension, input.IdempotencyKey); found || err != nil {
		return replayed, err
	}
	request := lifecycleServiceRequest{
		operation: LifecycleMachineEnable, target: extension,
		idempotencyKey: input.IdempotencyKey, confirmationToken: input.ConfirmationToken,
	}
	switch extension.Status {
	case StatusInstalled:
		request.operation = LifecycleMachineInstall
		if staged, ok := extension.StagedArtifact(); ok {
			request.target = staged
		}
	case StatusDisabled:
		request.operation = LifecycleMachineEnable
	case StatusEnabled:
		if staged, ok := extension.StagedArtifact(); ok {
			request.operation = LifecycleMachineUpgrade
			request.source = exactLifecycleCopy(extension)
			request.target = staged
		}
	default:
		return Extension{}, fmt.Errorf("%w: unsupported extension status", ErrLifecycleCoordinatorInvalid)
	}
	return s.runLifecycleV2(ctx, actor, request)
}

func (s *Service) disableLifecycleV2(
	ctx context.Context,
	actor identity.Actor,
	extension Extension,
	input LifecycleRequestInput,
) (Extension, error) {
	if replayed, found, err := s.replayLifecycleV2(ctx, actor, extension, input.IdempotencyKey); found || err != nil {
		return replayed, err
	}
	if extension.Status != StatusEnabled {
		return Extension{}, fmt.Errorf("%w: only an enabled plugin can be disabled", ErrLifecycleCoordinatorInvalid)
	}
	return s.runLifecycleV2(ctx, actor, lifecycleServiceRequest{
		operation: LifecycleMachineDisable, source: exactLifecycleCopy(extension), target: extension,
		idempotencyKey: input.IdempotencyKey, frozenAuthority: true,
	})
}

// Upgrade activates the exact currently staged artifact. Upload remains inert;
// this method is the first operation allowed to execute candidate code.
func (s *Service) Upgrade(ctx context.Context, actor identity.Actor, id string, input UpgradeInput) (Extension, error) {
	if !canManagePlugins(actor) {
		return Extension{}, identity.ErrPermissionDenied
	}
	if s.safeMode {
		return Extension{}, ErrSafeModeActive
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return Extension{}, err
	}
	if extension.Type != TypePlugin {
		return Extension{}, ErrThemeActivationRequired
	}
	if !usesLifecycleV2(extension) {
		return Extension{}, fmt.Errorf("%w: lifecycle V2 is required", ErrLifecycleCoordinatorInvalid)
	}
	if replayed, found, err := s.replayLifecycleV2(ctx, actor, extension, input.IdempotencyKey); found || err != nil {
		return replayed, err
	}
	if extension.Status != StatusEnabled {
		return Extension{}, fmt.Errorf("%w: staged upgrade requires an enabled source", ErrLifecycleCoordinatorInvalid)
	}
	target, ok := extension.StagedArtifact()
	if !ok || !usesLifecycleV2(target) {
		return Extension{}, ErrStagedVersionNotFound
	}
	return s.runLifecycleV2(ctx, actor, lifecycleServiceRequest{
		operation: LifecycleMachineUpgrade, source: exactLifecycleCopy(extension), target: target,
		idempotencyKey: input.IdempotencyKey, confirmationToken: input.ConfirmationToken,
	})
}

// Rollback activates an exact historical artifact and reuses only that
// artifact's last successful frozen authority snapshot.
func (s *Service) Rollback(ctx context.Context, actor identity.Actor, id string, input RollbackInput) (Extension, error) {
	if !canManagePlugins(actor) {
		return Extension{}, identity.ErrPermissionDenied
	}
	if s.safeMode {
		return Extension{}, ErrSafeModeActive
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return Extension{}, err
	}
	if extension.Type != TypePlugin {
		return Extension{}, ErrThemeActivationRequired
	}
	if !usesLifecycleV2(extension) {
		return Extension{}, fmt.Errorf("%w: rollback requires an enabled lifecycle V2 source", ErrLifecycleCoordinatorInvalid)
	}
	if replayed, found, err := s.replayLifecycleV2(ctx, actor, extension, input.IdempotencyKey); found || err != nil {
		return replayed, err
	}
	if extension.Status != StatusEnabled {
		return Extension{}, fmt.Errorf("%w: rollback requires an enabled lifecycle V2 source", ErrLifecycleCoordinatorInvalid)
	}
	versions, ok := s.store.(ExactExtensionVersionRepository)
	if !ok {
		return Extension{}, ErrLifecycleCoordinatorUnavailable
	}
	version, err := versions.GetExtensionVersion(ctx, ExactExtensionVersionInput{
		ExtensionID: extension.ID, Version: input.TargetVersion, PackageDigest: input.TargetPackageDigest,
	})
	if err != nil {
		if errors.Is(err, ErrExtensionVersionNotFound) || errors.Is(err, ErrExtensionVersionInvalid) {
			return Extension{}, err
		}
		return Extension{}, errors.Join(ErrLifecycleCoordinatorUnavailable, err)
	}
	target := extensionFromExactVersion(extension, version)
	if !usesLifecycleV2(target) {
		return Extension{}, fmt.Errorf("%w: rollback target has no lifecycle V2 contract", ErrLifecycleCoordinatorInvalid)
	}
	return s.runLifecycleV2(ctx, actor, lifecycleServiceRequest{
		operation: LifecycleMachineRollback, source: exactLifecycleCopy(extension), target: target,
		idempotencyKey: input.IdempotencyKey, frozenAuthority: true,
	})
}

func (s *Service) runLifecycleV2(
	ctx context.Context,
	actor identity.Actor,
	request lifecycleServiceRequest,
) (Extension, error) {
	if s.safeMode {
		return Extension{}, ErrSafeModeActive
	}
	if !validLifecycleServiceIdempotencyKey(request.idempotencyKey) {
		return Extension{}, fmt.Errorf("%w: stable Idempotency-Key is required", ErrLifecycleCoordinatorInvalid)
	}
	if s.lifecycleCoordinator == nil || s.lifecyclePreflight == nil || s.lifecycleAuthority == nil {
		return Extension{}, ErrLifecycleCoordinatorUnavailable
	}
	if err := validateLifecycleSourceArtifact(request.operation, request.target, request.source); err != nil {
		return Extension{}, err
	}

	authority, err := s.lifecycleServiceAuthority(ctx, actor, request)
	if err != nil {
		return Extension{}, err
	}
	auditEventID, err := s.appendLifecycleRequestAudit(ctx, actor, request)
	if err != nil {
		return Extension{}, err
	}
	// This callback runs before coordinator position zero, so no Host adapter can
	// stage a runtime instance until all static facts have passed.
	if err := s.lifecyclePreflight(ctx, request.operation, request.source, request.target); err != nil {
		return Extension{}, errors.Join(ErrPreflightFailed, err)
	}
	runInput, err := BuildLifecycleCoordinatorRunInput(request.target, actor, authority, LifecycleOperationIntent{
		Operation: request.operation, IdempotencyKey: request.idempotencyKey,
		SourceExtension: request.source, AuditEventID: auditEventID,
		FrozenAuthority: request.frozenAuthority,
	})
	if err != nil {
		return Extension{}, err
	}
	result, err := s.lifecycleCoordinator.Run(ctx, runInput)
	if err != nil {
		return Extension{}, lifecycleCoordinatorServiceError(err)
	}
	return s.finishLifecycleV2(ctx, actor, request, result)
}

func (s *Service) finishLifecycleV2(
	ctx context.Context,
	actor identity.Actor,
	request lifecycleServiceRequest,
	result LifecycleCoordinatorRunResult,
) (Extension, error) {
	if result.Operation.TerminalResult != LifecycleTerminalSucceeded &&
		result.Operation.TerminalResult != LifecycleTerminalSkipped {
		return Extension{}, ErrLifecycleCoordinatorActionFailed
	}
	current, err := s.store.Get(ctx, request.target.ID)
	if err != nil {
		return Extension{}, errors.Join(ErrLifecycleCoordinatorUnavailable, err)
	}
	if result.Operation.TerminalResult == LifecycleTerminalSucceeded && !result.Replayed {
		s.emitLifecycleCompatibilityEvent(ctx, actor, request.operation, current)
	}
	return s.decorateRuntime(ctx, current), nil
}

// replayLifecycleV2 reconstructs the immutable request before consulting the
// extension's now-mutated status/staged pointer. This is what makes a network
// retry stable after state publication has already committed.
func (s *Service) replayLifecycleV2(
	ctx context.Context,
	actor identity.Actor,
	current Extension,
	idempotencyKey string,
) (Extension, bool, error) {
	if !validLifecycleServiceIdempotencyKey(idempotencyKey) {
		return Extension{}, true, fmt.Errorf("%w: stable Idempotency-Key is required", ErrLifecycleCoordinatorInvalid)
	}
	if s.lifecycleCoordinator == nil || s.lifecyclePreflight == nil || s.lifecycleAuthority == nil {
		return Extension{}, true, ErrLifecycleCoordinatorUnavailable
	}
	operation, err := s.lifecycleAuthority.OperationByIdempotencyKey(ctx, current.ID, idempotencyKey)
	if errors.Is(err, ErrLifecycleOperationNotFound) {
		return Extension{}, false, nil
	}
	if err != nil {
		return Extension{}, true, errors.Join(ErrLifecycleCoordinatorUnavailable, err)
	}
	if operation.RequestedByUserID > 0 && operation.RequestedByUserID != actor.ID {
		return Extension{}, true, ErrLifecycleFingerprintConflict
	}
	if operation.CompletedAt != nil {
		switch operation.TerminalResult {
		case LifecycleTerminalSucceeded, LifecycleTerminalSkipped:
			// A completed replay has no coordinator work and therefore no process
			// staging boundary to preflight again. Return the published state directly.
			item, loadErr := s.store.Get(ctx, current.ID)
			if loadErr != nil {
				return Extension{}, true, errors.Join(ErrLifecycleCoordinatorUnavailable, loadErr)
			}
			return s.decorateRuntime(ctx, item), true, nil
		case LifecycleTerminalFailed, LifecycleTerminalCancelled:
			// Continue so coordinator returns the explicit-recovery requirement.
		default:
			return Extension{}, true, ErrLifecycleCoordinatorInvalid
		}
	}
	request, input, err := s.rebuildLifecycleReplay(ctx, current, operation)
	if err != nil {
		return Extension{}, true, err
	}
	if err := s.lifecyclePreflight(ctx, request.operation, request.source, request.target); err != nil {
		return Extension{}, true, errors.Join(ErrPreflightFailed, err)
	}
	result, err := s.lifecycleCoordinator.Run(ctx, input)
	if err != nil {
		return Extension{}, true, lifecycleCoordinatorServiceError(err)
	}
	// Existing logical requests never emit compatibility events. Coordinator
	// creation owns that one-shot side effect even when this caller finishes it.
	result.Replayed = true
	item, err := s.finishLifecycleV2(ctx, actor, request, result)
	return item, true, err
}

func (s *Service) rebuildLifecycleReplay(
	ctx context.Context,
	current Extension,
	operation LifecycleOperation,
) (lifecycleServiceRequest, LifecycleCoordinatorRunInput, error) {
	operationKind := LifecycleMachineOperation(operation.Operation)
	if _, err := RecommendedLifecyclePath(operationKind); err != nil {
		return lifecycleServiceRequest{}, LifecycleCoordinatorRunInput{}, ErrLifecycleCoordinatorInvalid
	}
	target, err := s.lifecycleExactArtifact(ctx, current, ExactExtensionVersionInput{
		ExtensionID: operation.ExtensionID, Version: operation.ExtensionVersion, PackageDigest: operation.PackageDigest,
	})
	if err != nil {
		return lifecycleServiceRequest{}, LifecycleCoordinatorRunInput{}, err
	}
	request := lifecycleServiceRequest{
		operation: operationKind, target: target, idempotencyKey: operation.IdempotencyKey,
		frozenAuthority: operationKind == LifecycleMachineDisable || operationKind == LifecycleMachineRollback,
	}
	switch operationKind {
	case LifecycleMachineDisable:
		request.source = exactLifecycleCopy(target)
	case LifecycleMachineUpgrade, LifecycleMachineRollback:
		source, sourceErr := s.lifecycleReplaySource(ctx, current, operation)
		if sourceErr != nil {
			return lifecycleServiceRequest{}, LifecycleCoordinatorRunInput{}, sourceErr
		}
		request.source = &source
	}
	input := LifecycleCoordinatorRunInput{
		Extension: target, SourceExtension: request.source,
		Acquire: AcquireLifecycleOperationInput{
			ExtensionID: operation.ExtensionID, ExtensionVersion: operation.ExtensionVersion,
			PackageDigest: operation.PackageDigest, ArtifactDigests: cloneLifecycleJSON(operation.ArtifactDigests),
			Operation: operation.Operation, PlanVersion: operation.PlanVersion,
			IdempotencyKey: operation.IdempotencyKey, RequestFingerprint: operation.RequestFingerprint,
			AuthorityType: operation.AuthorityType, TrustGrantID: operation.TrustGrantID,
			AuthoritySnapshot: cloneLifecycleJSON(operation.AuthoritySnapshot),
			RequestedByUserID: operation.RequestedByUserID, AuditEventID: operation.AuditEventID,
			RemovalMode: operation.RemovalMode, Forced: operation.Forced, ExistingOnly: true,
		},
	}
	return request, input, nil
}

func (s *Service) lifecycleReplaySource(ctx context.Context, current Extension, operation LifecycleOperation) (Extension, error) {
	if len(operation.Progress) > 0 && string(operation.Progress) != "{}" {
		machine, err := decodeLifecycleCoordinatorMachine(operation.Progress)
		if err != nil {
			return Extension{}, err
		}
		binding := machine.SourceBinding
		if binding.ExtensionID != "" && binding.ExtensionVersion != "" && binding.PackageDigest != "" {
			return s.lifecycleExactArtifact(ctx, current, ExactExtensionVersionInput{
				ExtensionID: binding.ExtensionID, Version: binding.ExtensionVersion, PackageDigest: binding.PackageDigest,
			})
		}
	}
	if current.ID == operation.ExtensionID &&
		(current.Version != operation.ExtensionVersion || current.PackageDigest != operation.PackageDigest) {
		copy := current
		copy.StagedVersion = nil
		return copy, nil
	}
	return Extension{}, fmt.Errorf("%w: exact lifecycle source is unavailable", ErrLifecycleCoordinatorInvalid)
}

func (s *Service) lifecycleExactArtifact(
	ctx context.Context,
	current Extension,
	input ExactExtensionVersionInput,
) (Extension, error) {
	if current.ID == input.ExtensionID && current.Version == input.Version && current.PackageDigest == input.PackageDigest {
		copy := current
		copy.StagedVersion = nil
		return copy, nil
	}
	versions, ok := s.store.(ExactExtensionVersionRepository)
	if !ok {
		return Extension{}, ErrLifecycleCoordinatorUnavailable
	}
	version, err := versions.GetExtensionVersion(ctx, input)
	if err != nil {
		if errors.Is(err, ErrExtensionVersionNotFound) || errors.Is(err, ErrExtensionVersionInvalid) {
			return Extension{}, err
		}
		return Extension{}, errors.Join(ErrLifecycleCoordinatorUnavailable, err)
	}
	return extensionFromExactVersion(current, version), nil
}

func (s *Service) lifecycleServiceAuthority(
	ctx context.Context,
	actor identity.Actor,
	request lifecycleServiceRequest,
) (LifecycleAuthoritySnapshot, error) {
	if request.frozenAuthority {
		if s.lifecycleAuthority == nil {
			return LifecycleAuthoritySnapshot{}, ErrLifecycleCoordinatorUnavailable
		}
		authority, err := s.lifecycleAuthority.LastSuccessfulLifecycleAuthority(ctx, ExactExtensionVersionInput{
			ExtensionID: request.target.ID, Version: request.target.Version,
			PackageDigest: request.target.PackageDigest,
		})
		if err != nil && !errors.Is(err, ErrLifecycleAuthorityNotFound) {
			return LifecycleAuthoritySnapshot{}, errors.Join(ErrLifecycleCoordinatorUnavailable, err)
		}
		return authority, err
	}
	if s.executableTrust == nil {
		return LifecycleAuthoritySnapshot{}, ErrTrustChallengeRequired
	}
	authority, err := s.executableTrust.ConfirmLifecycleAuthority(ctx, actor, request.target, request.confirmationToken)
	if err != nil {
		return LifecycleAuthoritySnapshot{}, errors.Join(ErrLifecycleCoordinatorUnavailable, err)
	}
	return authority, nil
}

func (s *Service) appendLifecycleRequestAudit(
	ctx context.Context,
	actor identity.Actor,
	request lifecycleServiceRequest,
) (int64, error) {
	writer, ok := s.auditor.(audit.IDWriter)
	if !ok || writer == nil {
		return 0, ErrLifecycleCoordinatorUnavailable
	}
	action := audit.ActionExtensionEnable
	switch request.operation {
	case LifecycleMachineDisable:
		action = audit.ActionExtensionDisable
	case LifecycleMachineUpgrade:
		action = audit.ActionExtensionUpgraded
	case LifecycleMachineRollback:
		action = audit.ActionExtensionRollback
	}
	id, err := writer.AppendReturningID(ctx, audit.Event{
		ActorUserID: actor.ID,
		Action:      action,
		Metadata: map[string]any{
			"extensionId":   request.target.ID,
			"operation":     request.operation,
			"version":       request.target.Version,
			"packageDigest": request.target.PackageDigest,
		},
	})
	if err != nil {
		return 0, errors.Join(ErrLifecycleCoordinatorUnavailable, fmt.Errorf("write lifecycle request audit: %w", err))
	}
	if id <= 0 {
		return 0, ErrLifecycleCoordinatorUnavailable
	}
	return id, nil
}

func lifecycleCoordinatorServiceError(err error) error {
	if errors.Is(err, ErrLifecycleCoordinatorInvalid) ||
		errors.Is(err, ErrLifecycleCoordinatorUnavailable) ||
		errors.Is(err, ErrLifecycleCoordinatorRetryRequired) ||
		errors.Is(err, ErrLifecycleCoordinatorActionFailed) ||
		errors.Is(err, ErrLifecycleFingerprintConflict) ||
		errors.Is(err, ErrLifecycleOperationInProgress) {
		return err
	}
	return errors.Join(ErrLifecycleCoordinatorUnavailable, err)
}

func (s *Service) emitLifecycleCompatibilityEvent(
	ctx context.Context,
	actor identity.Actor,
	operation LifecycleMachineOperation,
	extension Extension,
) {
	action, message, hook := EventEnabled, "Extension enabled.", appevents.ExtensionEnabled
	switch operation {
	case LifecycleMachineDisable:
		action, message, hook = EventDisabled, "Extension disabled.", appevents.ExtensionDisabled
	case LifecycleMachineUpgrade:
		action, message = EventUpgraded, "Staged extension upgrade activated."
	case LifecycleMachineRollback:
		action, message = EventRolledBack, "Historical extension version restored."
	}
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: extension.ID, ActorUserID: actor.ID, Action: action, Message: message,
	})
	if s.runtime != nil && hook != "" {
		s.runtime.EmitHook(ctx, hook, map[string]any{"extensionId": extension.ID})
	}
}

func exactLifecycleCopy(extension Extension) *Extension {
	copy := extension
	copy.StagedVersion = nil
	return &copy
}

func extensionFromExactVersion(base Extension, version ExtensionVersion) Extension {
	base.Name = version.Manifest.Name
	base.Version = version.Version
	base.Manifest = version.Manifest
	base.PackageDigest = version.PackageDigest
	base.AdminFrontendDigest = version.AdminFrontendDigest
	base.PackagePath = version.PackagePath
	base.ActiveVersionID = version.ID
	base.InstalledAt = version.InstalledAt
	base.StagedVersion = nil
	return base
}

func validLifecycleServiceIdempotencyKey(value string) bool {
	if len(value) == 0 || len(value) > 128 || value != strings.TrimSpace(value) {
		return false
	}
	for _, char := range []byte(value) {
		if char < 0x21 || char > 0x7e {
			return false
		}
	}
	return true
}

func decodeLifecycleAuthoritySnapshot(value json.RawMessage) (LifecycleAuthoritySnapshot, error) {
	var authority LifecycleAuthoritySnapshot
	if err := json.Unmarshal(value, &authority); err != nil {
		return LifecycleAuthoritySnapshot{}, fmt.Errorf("decode lifecycle authority snapshot: %w", err)
	}
	if authority.SchemaVersion != LifecycleAuthoritySnapshotSchemaV1 || authority.ActorUserID <= 0 {
		return LifecycleAuthoritySnapshot{}, ErrLifecycleAuthorityNotFound
	}
	return authority, nil
}

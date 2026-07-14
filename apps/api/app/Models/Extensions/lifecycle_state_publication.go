package extensions

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrLifecycleStatePublicationInvalid     = errors.New("extensions: invalid lifecycle state publication")
	ErrLifecycleStatePublicationNotPrepared = errors.New("extensions: lifecycle state publication not prepared")
	ErrLifecycleStatePublicationConflict    = errors.New("extensions: lifecycle state publication conflict")
	ErrLifecycleStatePublicationCommitted   = errors.New("extensions: lifecycle state publication already committed")
)

type LifecycleStatePublicationMode string

const (
	LifecycleStatePublicationActivate   LifecycleStatePublicationMode = "activate"
	LifecycleStatePublicationDeactivate LifecycleStatePublicationMode = "deactivate"
)

type LifecycleStatePublicationPhase string

const (
	LifecycleStatePublicationSource LifecycleStatePublicationPhase = "source"
	LifecycleStatePublicationTarget LifecycleStatePublicationPhase = "target"
)

// LifecycleStatePublicationArtifact 是数据库状态切换所需的不可变制品身份。
// Runtime instance 属于进程发布层，不得成为恢复 extension row 的事实来源。
type LifecycleStatePublicationArtifact struct {
	ExtensionID   string
	Version       string
	PackageDigest string
	VersionID     int64
}

type PrepareLifecycleStatePublicationInput struct {
	OperationID int64
	Operation   LifecycleMachineOperation
	Position    int
	StepID      string
	Attempt     int
	Mode        LifecycleStatePublicationMode
	Source      *LifecycleStatePublicationArtifact
	Target      LifecycleStatePublicationArtifact
}

// LifecycleStatePublicationRef 只携带 durable row 的查找和 attempt fence。
// source/target status 与 active/staged 指针必须在 PostgreSQL 中重建。
type LifecycleStatePublicationRef struct {
	OperationID int64
	StepID      string
	Mode        LifecycleStatePublicationMode
	Attempt     int
}

type LifecycleStatePublicationRepository interface {
	PrepareLifecycleStatePublication(context.Context, PrepareLifecycleStatePublicationInput) (LifecycleStatePublicationRef, error)
	InspectLifecycleStatePublication(context.Context, LifecycleStatePublicationRef) (LifecycleStatePublicationPhase, error)
	PublishLifecycleState(context.Context, LifecycleStatePublicationRef) error
	RestoreLifecycleState(context.Context, LifecycleStatePublicationRef) error
}

func validatePrepareLifecycleStatePublicationInput(input PrepareLifecycleStatePublicationInput) error {
	position, mode, err := lifecycleStatePublicationPoint(input.Operation)
	if err != nil || input.OperationID <= 0 || input.Position != position || input.Mode != mode ||
		input.Attempt <= 0 || !validLifecycleStateStepID(input.StepID) {
		return ErrLifecycleStatePublicationInvalid
	}
	path, err := RecommendedLifecyclePath(input.Operation)
	if err != nil || position >= len(path) || path[position].Action != "" ||
		input.StepID != fmt.Sprintf("lifecycle.%s.%02d.host.%s", input.Operation, position, path[position].State) {
		return ErrLifecycleStatePublicationInvalid
	}
	if !validLifecycleStateArtifact(input.Target) {
		return ErrLifecycleStatePublicationInvalid
	}
	if input.Source != nil {
		if !validLifecycleStateArtifact(*input.Source) || input.Source.ExtensionID != input.Target.ExtensionID {
			return ErrLifecycleStatePublicationInvalid
		}
	}
	switch input.Operation {
	case LifecycleMachineInstall:
		if input.Source != nil {
			return ErrLifecycleStatePublicationInvalid
		}
	case LifecycleMachineEnable:
		if input.Source != nil && *input.Source != input.Target {
			return ErrLifecycleStatePublicationInvalid
		}
	case LifecycleMachineDisable, LifecycleMachineUninstall:
		if input.Source == nil || *input.Source != input.Target {
			return ErrLifecycleStatePublicationInvalid
		}
	case LifecycleMachineUpgrade, LifecycleMachineRollback:
		if input.Source == nil || *input.Source == input.Target {
			return ErrLifecycleStatePublicationInvalid
		}
	default:
		return ErrLifecycleStatePublicationInvalid
	}
	return nil
}

func validateLifecycleStatePublicationRef(ref LifecycleStatePublicationRef) error {
	if ref.OperationID <= 0 || ref.Attempt <= 0 || !validLifecycleStateStepID(ref.StepID) ||
		(ref.Mode != LifecycleStatePublicationActivate && ref.Mode != LifecycleStatePublicationDeactivate) {
		return ErrLifecycleStatePublicationInvalid
	}
	return nil
}

func lifecycleStatePublicationPoint(operation LifecycleMachineOperation) (int, LifecycleStatePublicationMode, error) {
	switch operation {
	case LifecycleMachineInstall:
		return 8, LifecycleStatePublicationActivate, nil
	case LifecycleMachineEnable:
		return 5, LifecycleStatePublicationActivate, nil
	case LifecycleMachineDisable:
		return 3, LifecycleStatePublicationDeactivate, nil
	case LifecycleMachineUpgrade:
		return 8, LifecycleStatePublicationActivate, nil
	case LifecycleMachineRollback:
		return 6, LifecycleStatePublicationActivate, nil
	case LifecycleMachineUninstall:
		return 3, LifecycleStatePublicationDeactivate, nil
	default:
		return 0, "", ErrLifecycleStatePublicationInvalid
	}
}

func validLifecycleStateArtifact(artifact LifecycleStatePublicationArtifact) bool {
	return artifact.ExtensionID != "" && artifact.ExtensionID == normalizeID(artifact.ExtensionID) &&
		isExactExtensionVersionIdentity(artifact.VersionID, artifact.Version, artifact.PackageDigest)
}

func validLifecycleStateStepID(stepID string) bool {
	return stepID != "" && stepID == strings.TrimSpace(stepID) && len(stepID) <= 512
}

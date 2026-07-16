package mediaregistry

import (
	"encoding/json"
	"fmt"
)

// PlanSummary is the only plan representation intended for inspectors, logs,
// or support exports. It deliberately excludes actor identity, permission
// fingerprints, source ids/digests/filenames, URLs, and storage handles.
type PlanSummary struct {
	SchemaVersion  string            `json:"schemaVersion"`
	Revision       uint64            `json:"revision"`
	RegistryDigest string            `json:"registryDigest"`
	PlanDigest     string            `json:"planDigest"`
	SafeMode       bool              `json:"safeMode,omitempty"`
	Kind           string            `json:"kind"`
	Purpose        string            `json:"purpose"`
	SourceKind     string            `json:"sourceKind"`
	SourceMIME     string            `json:"sourceMime"`
	SourceSize     int64             `json:"sourceSizeBytes"`
	Policy         ProviderRef       `json:"policy"`
	Steps          []PlanStepSummary `json:"steps"`
}

type PlanStepSummary struct {
	ID        string   `json:"id"`
	Stage     string   `json:"stage"`
	Execution string   `json:"execution"`
	Artifact  Artifact `json:"artifact"`
	Variants  int      `json:"variants,omitempty"`
}

func SummarizePlan(plan Plan) (PlanSummary, error) {
	if plan.SchemaVersion != SchemaVersion || plan.Revision == 0 || plan.Digest == "" ||
		plan.Digest != computePlanDigest(plan) || plan.Source != plan.OriginalFallback {
		return PlanSummary{}, ErrPlanStale
	}
	result := PlanSummary{
		SchemaVersion: plan.SchemaVersion, Revision: plan.Revision,
		RegistryDigest: plan.RegistryDigest, PlanDigest: plan.Digest,
		SafeMode: plan.SafeMode, Kind: plan.Kind, Purpose: plan.Purpose,
		SourceKind: plan.Source.Kind, SourceMIME: plan.Source.MIME,
		SourceSize: plan.Source.SizeBytes, Policy: policyRef(plan.Policy),
		Steps: make([]PlanStepSummary, 0, len(plan.Steps)),
	}
	for _, step := range plan.Steps {
		result.Steps = append(result.Steps, PlanStepSummary{
			ID: step.ID, Stage: step.Processor.Stage,
			Execution: step.Processor.Execution, Artifact: step.Processor.Artifact,
			Variants: len(step.Variants),
		})
	}
	return result, nil
}

// Plans contain Host-private authorization and original filename material.
// Production queue wiring must persist a Host-protected reference/envelope,
// not raw plan JSON.
func (Plan) MarshalJSON() ([]byte, error) { return nil, ErrPrivatePlan }

func (plan Plan) String() string {
	summary, err := SummarizePlan(plan)
	if err != nil {
		return "MediaPlan(invalid)"
	}
	return fmt.Sprintf("MediaPlan(revision=%d,digest=%s,kind=%s,purpose=%s,steps=%d)",
		summary.Revision, summary.PlanDigest, summary.Kind, summary.Purpose, len(summary.Steps))
}

func (plan Plan) GoString() string { return plan.String() }

func (BackgroundOperation) MarshalJSON() ([]byte, error) { return nil, ErrPrivatePlan }

func (operation BackgroundOperation) String() string {
	return fmt.Sprintf("MediaOperation(key=%s,step=%s,attempt=%d,plan=%s)",
		operation.Key, operation.StepID, operation.Attempt, operation.Plan.String())
}

func (operation BackgroundOperation) GoString() string { return operation.String() }

var _ json.Marshaler = Plan{}
var _ json.Marshaler = BackgroundOperation{}

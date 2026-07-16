package contentregistry

import (
	"strings"
)

type InspectorStep struct {
	TargetContractVersion string   `json:"targetContractVersion"`
	ContentID             string   `json:"contentId"`
	ContractVersion       string   `json:"contractVersion"`
	Kind                  string   `json:"kind"`
	Action                string   `json:"action"`
	Priority              int      `json:"priority"`
	Fallback              string   `json:"fallback"`
	Artifact              Artifact `json:"artifact"`
	Selected              bool     `json:"selected"`
}

type InspectorConflict struct {
	TargetID   string          `json:"targetId"`
	Candidates []InspectorStep `json:"candidates"`
}

type InspectorSnapshot struct {
	SchemaVersion string               `json:"schemaVersion"`
	Revision      uint64               `json:"revision"`
	Digest        string               `json:"digest"`
	SafeMode      bool                 `json:"safeMode,omitempty"`
	TargetID      string               `json:"targetId"`
	Chain         []InspectorStep      `json:"chain"`
	Conflicts     []InspectorConflict  `json:"conflicts"`
	Stale         []InspectorStep      `json:"stale"`
	Traces        []ContentTraceRecord `json:"traces"`
}

// Inspect returns only bounded contract/attribution metadata. It never exposes
// editor source, rendered segments, permission fingerprints, or raw failures.
func (e *Executor) Inspect(targetID string) (InspectorSnapshot, error) {
	if e == nil {
		return InspectorSnapshot{}, ErrExecutionInvalid
	}
	targetID = strings.ToLower(strings.TrimSpace(targetID))
	state := e.registry.load()
	target, found := state.content[targetID]
	if !found {
		return InspectorSnapshot{}, ErrNotFound
	}
	plan, err := e.buildPlanFromState(state, ExecutionRequest{TargetID: targetID, ContractVersion: target.ContractVersion})
	if err != nil {
		return InspectorSnapshot{}, err
	}
	result := InspectorSnapshot{
		SchemaVersion: ExecutionSchemaVersion, Revision: state.revision, Digest: state.digest,
		SafeMode: state.safeMode, TargetID: targetID,
		Chain: []InspectorStep{}, Conflicts: []InspectorConflict{}, Stale: []InspectorStep{},
		Traces: []ContentTraceRecord{},
	}
	for _, step := range plan.selectedSteps() {
		result.Chain = append(result.Chain, inspectorPlannedStep(step, true))
	}
	if len(plan.conflicts) > 0 {
		candidates := make([]InspectorStep, 0, len(plan.conflicts)+1)
		if plan.terminal != nil {
			candidates = append(candidates, inspectorPlannedStep(*plan.terminal, true))
		}
		for _, step := range plan.conflicts {
			candidates = append(candidates, inspectorPlannedStep(step, false))
		}
		result.Conflicts = append(result.Conflicts, InspectorConflict{TargetID: targetID, Candidates: candidates})
	}
	for _, binding := range plan.stale {
		result.Stale = append(result.Stale, InspectorStep{
			TargetContractVersion: binding.TargetContractVersion,
			ContentID:             binding.DeclarationID, ContractVersion: binding.ContractVersion,
			Action: binding.Action, Priority: binding.Priority, Fallback: binding.Fallback,
			Artifact: binding.Artifact, Selected: false,
		})
	}
	if reader, ok := e.trace.(ContentTargetTraceReader); ok {
		result.Traces = append(result.Traces, reader.ContentTracesForTarget(targetID, 100)...)
	} else if reader, ok := e.trace.(ContentTraceReader); ok {
		for _, record := range reader.ContentTraces(100) {
			if record.TargetID == targetID {
				result.Traces = append(result.Traces, record)
			}
		}
	}
	return result, nil
}

func inspectorPlannedStep(step plannedBinding, selected bool) InspectorStep {
	return InspectorStep{
		TargetContractVersion: step.binding.TargetContractVersion,
		ContentID:             step.contribution.ID, ContractVersion: step.contribution.ContractVersion,
		Kind: step.contribution.Kind, Action: step.binding.Action, Priority: step.binding.Priority,
		Fallback: step.binding.Fallback, Artifact: step.contribution.Artifact, Selected: selected,
	}
}

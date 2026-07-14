package extensionsruntime

import (
	"context"
	"fmt"
	"time"
)

type ProviderSlotProbeResult struct {
	ContractID  string            `json:"contractId"`
	CandidateID string            `json:"candidateId"`
	Artifact    HookArtifact      `json:"artifact"`
	OK          bool              `json:"ok"`
	Reason      string            `json:"reason"`
	Message     string            `json:"message"`
	Details     map[string]string `json:"details,omitempty"`
	Suggestions []string          `json:"suggestions,omitempty"`
	DurationMS  int64             `json:"durationMs"`
}

// ProbeProviderSlotCandidate checks the active exact runtime through the
// provider's side-effect-free probe RPC. It never sends a business invocation
// document and therefore cannot accidentally create provider-owned work.
func (m *Manager) ProbeProviderSlotCandidate(ctx context.Context, contractID, candidateID string) (ProviderSlotProbeResult, error) {
	started := time.Now()
	if m == nil || ctx == nil {
		return ProviderSlotProbeResult{}, ErrProviderSlotSelectionInvalid
	}
	contract, candidate, err := exactProviderSlotCandidate(m.HookBus().ProviderSlots().Snapshot(), contractID, candidateID)
	if err != nil {
		return ProviderSlotProbeResult{}, err
	}
	result := ProviderSlotProbeResult{ContractID: contract.ID, CandidateID: candidate.ID, Artifact: candidate.Artifact}
	extension, available := m.runningExtension(candidate.Artifact.ExtensionID)
	if !available || extension.Version != candidate.Artifact.ExtensionVersion || extension.PackageDigest != candidate.Artifact.PackageDigest {
		return result, ErrProviderSlotSelectionStale
	}
	instance, admission, err := m.AcquireActiveRuntimeCall(ctx, extension.ID, RuntimeCallProvider)
	if err != nil {
		return result, err
	}
	defer admission.Release()
	if instance.Identity.InstanceID != candidate.Artifact.RuntimeInstanceID {
		return result, ErrProviderSlotSelectionStale
	}
	prober, ok := m.starter.(interface {
		ProviderProbe(context.Context, string, ProviderProbeRequest) (ProviderProbeResponse, error)
	})
	if !ok {
		return result, fmt.Errorf("%w: provider probe RPC", ErrProviderSlotNoProvider)
	}
	probeCtx, cancel := context.WithTimeout(admission.Context, time.Duration(contract.TimeoutMS)*time.Millisecond)
	defer cancel()
	probe, err := prober.ProviderProbe(probeCtx, extension.ID, ProviderProbeRequest{Slot: contract.Slot})
	result.OK, result.Reason, result.Message = probe.OK, probe.Reason, probe.Message
	result.Details = cloneProviderProbeDetails(probe.Details)
	result.Suggestions = append([]string(nil), probe.Suggestions...)
	result.DurationMS = time.Since(started).Milliseconds()
	if err != nil {
		return result, err
	}
	return result, nil
}

func cloneProviderProbeDetails(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

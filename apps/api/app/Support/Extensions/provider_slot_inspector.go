package extensionsruntime

import (
	"sort"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

const (
	providerSlotAvailable          = "available"
	providerSlotUnavailable        = "unavailable"
	providerSlotNoCandidates       = "no_candidates"
	providerSlotRuntimeUnavailable = "runtime_unavailable"
	providerSlotPriorityTie        = "priority_tie"
)

func (m *Manager) ProviderSlotInspection() extensions.ProviderSlotInspection {
	snapshot := m.HookBus().ProviderSlots().Snapshot()
	candidatesByTarget := make(map[string][]ProviderSlotCandidate, len(snapshot.Contracts))
	for _, candidate := range snapshot.Candidates {
		candidatesByTarget[candidate.TargetID] = append(candidatesByTarget[candidate.TargetID], candidate)
	}
	result := extensions.ProviderSlotInspection{
		Revision: snapshot.Revision,
		Slots:    make([]extensions.ProviderSlotInspectionItem, 0, len(snapshot.Contracts)),
	}
	for _, contract := range snapshot.Contracts {
		contractArtifact := providerSlotInspectionArtifact(contract.Artifact)
		item := extensions.ProviderSlotInspectionItem{
			Contract: extensions.ProviderSlotContractInspection{
				ID: contract.ID, Slot: contract.Slot, ContractVersion: contract.ContractVersion,
				RequestSchema: contract.RequestSchema, ResponseSchema: contract.ResponseSchema,
				RequestSchemaDigest: contract.RequestSchemaDigest, ResponseSchemaDigest: contract.ResponseSchemaDigest,
				Fallback: contract.Fallback, TimeoutMS: contract.TimeoutMS, Artifact: contractArtifact,
				ContractRuntimeAvailable: m.providerSlotArtifactAvailable(contract.Artifact),
			},
			Candidates: []extensions.ProviderSlotCandidateInspection{},
			Conflicts:  []extensions.ProviderSlotConflictInspection{},
		}
		priorityCandidates := make(map[int][]string)
		for index, candidate := range candidatesByTarget[contract.ID] {
			availability := providerSlotUnavailable
			if m.providerSlotArtifactAvailable(candidate.Artifact) {
				availability = providerSlotAvailable
				item.Availability = providerSlotAvailable
			}
			item.Candidates = append(item.Candidates, extensions.ProviderSlotCandidateInspection{
				ID: candidate.ID, TargetID: candidate.TargetID, Label: candidate.Label, Handler: candidate.Handler,
				Priority: candidate.Priority, Rank: index + 1,
				Artifact: providerSlotInspectionArtifact(candidate.Artifact), Availability: availability,
			})
			priorityCandidates[candidate.Priority] = append(priorityCandidates[candidate.Priority], candidate.ID)
		}
		for priority, ids := range priorityCandidates {
			if len(ids) > 1 {
				item.Conflicts = append(item.Conflicts, extensions.ProviderSlotConflictInspection{
					Kind: providerSlotPriorityTie, Priority: priority, CandidateIDs: append([]string(nil), ids...),
				})
			}
		}
		sort.Slice(item.Conflicts, func(i, j int) bool { return item.Conflicts[i].Priority > item.Conflicts[j].Priority })
		if item.Availability == "" {
			item.Availability = providerSlotUnavailable
			if len(item.Candidates) == 0 {
				item.UnavailabilityReason = providerSlotNoCandidates
			} else {
				item.UnavailabilityReason = providerSlotRuntimeUnavailable
			}
		}
		result.Slots = append(result.Slots, item)
	}
	return result
}

func (m *Manager) providerSlotArtifactAvailable(artifact HookArtifact) bool {
	if m == nil {
		return false
	}
	return m.RuntimeInstanceAvailable(RuntimeInstanceIdentity{
		ExtensionID: artifact.ExtensionID, InstanceID: artifact.RuntimeInstanceID,
	})
}

func providerSlotInspectionArtifact(artifact HookArtifact) extensions.ProviderSlotArtifactInspection {
	return extensions.ProviderSlotArtifactInspection{
		ExtensionID: artifact.ExtensionID, ExtensionVersion: artifact.ExtensionVersion,
		PackageDigest: artifact.PackageDigest, RuntimeInstanceID: artifact.RuntimeInstanceID,
	}
}

var _ extensions.ProviderSlotInspectionSource = (*Manager)(nil)

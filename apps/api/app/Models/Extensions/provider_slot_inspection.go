package extensions

import (
	"context"
	"errors"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

var ErrProviderSlotInspectionUnavailable = errors.New("extensions: provider slot inspection is unavailable")

type ProviderSlotArtifactInspection struct {
	ExtensionID       string `json:"extensionId"`
	ExtensionVersion  string `json:"extensionVersion"`
	PackageDigest     string `json:"packageDigest"`
	RuntimeInstanceID string `json:"runtimeInstanceId"`
}

type ProviderSlotContractInspection struct {
	ID                       string                         `json:"id"`
	Slot                     string                         `json:"slot"`
	ContractVersion          string                         `json:"contractVersion"`
	RequestSchema            string                         `json:"requestSchema"`
	ResponseSchema           string                         `json:"responseSchema"`
	RequestSchemaDigest      string                         `json:"requestSchemaDigest,omitempty"`
	ResponseSchemaDigest     string                         `json:"responseSchemaDigest,omitempty"`
	Fallback                 string                         `json:"fallback"`
	TimeoutMS                int                            `json:"timeoutMs"`
	Artifact                 ProviderSlotArtifactInspection `json:"artifact"`
	ContractRuntimeAvailable bool                           `json:"contractRuntimeAvailable"`
}

type ProviderSlotCandidateInspection struct {
	ID           string                         `json:"id"`
	TargetID     string                         `json:"targetId"`
	Label        string                         `json:"label"`
	Handler      string                         `json:"handler"`
	Priority     int                            `json:"priority"`
	Rank         int                            `json:"rank"`
	Artifact     ProviderSlotArtifactInspection `json:"artifact"`
	Availability string                         `json:"availability"`
}

type ProviderSlotConflictInspection struct {
	Kind         string   `json:"kind"`
	Priority     int      `json:"priority"`
	CandidateIDs []string `json:"candidateIds"`
}

type ProviderSlotInspectionItem struct {
	Contract             ProviderSlotContractInspection    `json:"contract"`
	Candidates           []ProviderSlotCandidateInspection `json:"candidates"`
	Conflicts            []ProviderSlotConflictInspection  `json:"conflicts"`
	Availability         string                            `json:"availability"`
	UnavailabilityReason string                            `json:"unavailabilityReason,omitempty"`
}

type ProviderSlotInspection struct {
	Revision uint64                       `json:"revision"`
	Slots    []ProviderSlotInspectionItem `json:"slots"`
}

type ProviderSlotInspectionSource interface {
	ProviderSlotInspection() ProviderSlotInspection
}

func (s *Service) InspectProviderSlots(ctx context.Context, actor identity.Actor) (ProviderSlotInspection, error) {
	if ctx == nil {
		return ProviderSlotInspection{}, ErrProviderSlotInspectionUnavailable
	}
	if err := ctx.Err(); err != nil {
		return ProviderSlotInspection{}, err
	}
	if !canViewExtensions(actor) {
		return ProviderSlotInspection{}, identity.ErrPermissionDenied
	}
	source, ok := s.runtime.(ProviderSlotInspectionSource)
	if !ok || source == nil {
		return ProviderSlotInspection{}, ErrProviderSlotInspectionUnavailable
	}
	return source.ProviderSlotInspection(), nil
}

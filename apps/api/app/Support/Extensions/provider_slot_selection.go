package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrProviderSlotSelectionInvalid          = errors.New("extension provider slot selection is invalid")
	ErrProviderSlotSelectionNotFound         = errors.New("extension provider slot selection is not found")
	ErrProviderSlotSelectionRevisionConflict = errors.New("extension provider slot selection revision conflict")
	ErrProviderSlotSelectionStale            = errors.New("extension provider slot selection is stale")
)

type ProviderSlotSelection struct {
	ContractID        string       `json:"contractId"`
	ContractVersion   string       `json:"contractVersion"`
	Slot              string       `json:"slot"`
	ContractArtifact  HookArtifact `json:"contractArtifact"`
	ContractVersionID int64        `json:"-"`
	CandidateID       string       `json:"candidateId"`
	ProviderArtifact  HookArtifact `json:"providerArtifact"`
	ProviderVersionID int64        `json:"-"`
	SelectedByUserID  int64        `json:"selectedByUserId"`
	SelectionAuditID  int64        `json:"selectionAuditEventId"`
	Revision          int64        `json:"revision"`
	SelectedAt        time.Time    `json:"selectedAt"`
	UpdatedAt         time.Time    `json:"updatedAt"`
}

type SelectProviderSlotRequest struct {
	Contract         ProviderSlotContract
	Candidate        ProviderSlotCandidate
	ExpectedRevision int64
	ActorUserID      int64
	AuditEventID     int64
}

type ResetProviderSlotRequest struct {
	ContractID       string
	ExpectedRevision int64
	ActorUserID      int64
	AuditEventID     int64
	ReasonCode       string
}

type InvalidateProviderSlotRequest struct {
	ExtensionID  string
	ActorUserID  int64
	AuditEventID int64
	ReasonCode   string
}

type ProviderSlotSelectionEvent struct {
	ID                int64                  `json:"id"`
	ContractID        string                 `json:"contractId"`
	ContractVersion   string                 `json:"contractVersion"`
	Slot              string                 `json:"slot"`
	Action            string                 `json:"action"`
	PreviousSelection *ProviderSlotSelection `json:"previousSelection,omitempty"`
	SelectedSelection *ProviderSlotSelection `json:"selectedSelection,omitempty"`
	ActorUserID       int64                  `json:"actorUserId,omitempty"`
	AuditEventID      int64                  `json:"auditEventId"`
	ReasonCode        string                 `json:"reasonCode,omitempty"`
	SelectionRevision int64                  `json:"selectionRevision"`
	CreatedAt         time.Time              `json:"createdAt"`
}

type ProviderSlotSelectionStore interface {
	Desired(context.Context, string) (ProviderSlotSelection, error)
	Selected(context.Context, string) (ProviderSlotSelection, error)
	Select(context.Context, SelectProviderSlotRequest) (ProviderSlotSelection, error)
	Reset(context.Context, ResetProviderSlotRequest) error
	InvalidateExtension(context.Context, InvalidateProviderSlotRequest) (int64, error)
	ListEvents(context.Context, string, int) ([]ProviderSlotSelectionEvent, error)
}

// ProviderSlotSelectionAPI validates every write against one immutable
// registry snapshot. The store then binds the chosen pair to active immutable
// extension-version rows with a revision compare-and-swap.
type ProviderSlotSelectionAPI struct {
	registry *VersionedProviderSlotRegistry
	store    ProviderSlotSelectionStore
}

func NewProviderSlotSelectionAPI(registry *VersionedProviderSlotRegistry, store ProviderSlotSelectionStore) *ProviderSlotSelectionAPI {
	return &ProviderSlotSelectionAPI{registry: registry, store: store}
}

func (a *ProviderSlotSelectionAPI) Select(ctx context.Context, contractID, candidateID string, expectedRevision, actorUserID, auditEventID int64) (ProviderSlotSelection, error) {
	if a == nil || a.registry == nil || a.store == nil || ctx == nil || expectedRevision < 0 || actorUserID <= 0 || auditEventID <= 0 {
		return ProviderSlotSelection{}, ErrProviderSlotSelectionInvalid
	}
	contract, candidate, err := exactProviderSlotCandidate(a.registry.Snapshot(), contractID, candidateID)
	if err != nil {
		return ProviderSlotSelection{}, err
	}
	return a.store.Select(ctx, SelectProviderSlotRequest{
		Contract: contract, Candidate: candidate, ExpectedRevision: expectedRevision,
		ActorUserID: actorUserID, AuditEventID: auditEventID,
	})
}

func (a *ProviderSlotSelectionAPI) Reset(ctx context.Context, request ResetProviderSlotRequest) error {
	request.ContractID = strings.TrimSpace(request.ContractID)
	request.ReasonCode = strings.TrimSpace(request.ReasonCode)
	if a == nil || a.store == nil || ctx == nil || request.ContractID == "" || request.ExpectedRevision <= 0 ||
		request.ActorUserID <= 0 || request.AuditEventID <= 0 || !validProviderSlotReason(request.ReasonCode, false) {
		return ErrProviderSlotSelectionInvalid
	}
	return a.store.Reset(ctx, request)
}

func (a *ProviderSlotSelectionAPI) Current(ctx context.Context, contractID string) (ProviderSlotSelection, error) {
	contractID = strings.TrimSpace(contractID)
	if a == nil || a.store == nil || ctx == nil || contractID == "" {
		return ProviderSlotSelection{}, ErrProviderSlotSelectionInvalid
	}
	return a.store.Desired(ctx, contractID)
}

func (a *ProviderSlotSelectionAPI) Events(ctx context.Context, contractID string, limit int) ([]ProviderSlotSelectionEvent, error) {
	contractID = strings.TrimSpace(contractID)
	if a == nil || a.store == nil || ctx == nil || contractID == "" {
		return nil, ErrProviderSlotSelectionInvalid
	}
	return a.store.ListEvents(ctx, contractID, limit)
}

func (a *ProviderSlotSelectionAPI) InvalidateExtension(ctx context.Context, request InvalidateProviderSlotRequest) (int64, error) {
	request.ExtensionID = strings.TrimSpace(request.ExtensionID)
	request.ReasonCode = strings.TrimSpace(request.ReasonCode)
	if a == nil || a.store == nil || ctx == nil || request.ExtensionID == "" || request.ActorUserID <= 0 ||
		request.AuditEventID <= 0 || !validProviderSlotReason(request.ReasonCode, true) {
		return 0, ErrProviderSlotSelectionInvalid
	}
	return a.store.InvalidateExtension(ctx, request)
}

// InvalidateProviderSlotSelections is the narrow lifecycle adapter used by
// Models/Extensions without importing registry or PostgreSQL implementation
// details.
func (a *ProviderSlotSelectionAPI) InvalidateProviderSlotSelections(
	ctx context.Context,
	extensionID string,
	actorUserID int64,
	auditEventID int64,
	reasonCode string,
) error {
	_, err := a.InvalidateExtension(ctx, InvalidateProviderSlotRequest{
		ExtensionID: extensionID, ActorUserID: actorUserID,
		AuditEventID: auditEventID, ReasonCode: reasonCode,
	})
	return err
}

func (a *ProviderSlotSelectionAPI) Resolve(ctx context.Context, caller ProviderSlotCaller, contractID, contractVersion string) (ProviderSlotResolution, string, error) {
	if a == nil || a.registry == nil || ctx == nil {
		return ProviderSlotResolution{}, "", ErrProviderSlotSelectionInvalid
	}
	resolution, err := a.registry.Discover(caller, contractID, contractVersion)
	if err != nil {
		return ProviderSlotResolution{}, "", err
	}
	if a.store == nil {
		return resolution, "default", nil
	}
	selection, err := a.store.Selected(ctx, resolution.Contract.ID)
	switch {
	case err == nil:
		selectedIndex := exactProviderSlotSelectionIndex(resolution, selection)
		if selectedIndex < 0 {
			err = ErrProviderSlotSelectionStale
		} else {
			selected := resolution.Candidates[selectedIndex]
			if resolution.Contract.Fallback == "closed" {
				resolution.Candidates = []ProviderSlotCandidate{selected}
			} else {
				ordered := []ProviderSlotCandidate{selected}
				ordered = append(ordered, resolution.Candidates[:selectedIndex]...)
				ordered = append(ordered, resolution.Candidates[selectedIndex+1:]...)
				resolution.Candidates = ordered
			}
			return resolution, "selected", nil
		}
	case errors.Is(err, ErrProviderSlotSelectionNotFound):
		return resolution, "default", nil
	}
	if errors.Is(err, ErrProviderSlotSelectionStale) && resolution.Contract.Fallback == "next" {
		return resolution, "stale_fallback", nil
	}
	return ProviderSlotResolution{}, "stale_closed", err
}

func exactProviderSlotCandidate(snapshot ProviderSlotRegistrySnapshot, contractID, candidateID string) (ProviderSlotContract, ProviderSlotCandidate, error) {
	contractID = strings.TrimSpace(contractID)
	candidateID = strings.TrimSpace(candidateID)
	var contract *ProviderSlotContract
	for index := range snapshot.Contracts {
		if snapshot.Contracts[index].ID == contractID || snapshot.Contracts[index].Slot == contractID {
			value := snapshot.Contracts[index]
			contract = &value
			break
		}
	}
	if contract == nil {
		return ProviderSlotContract{}, ProviderSlotCandidate{}, ErrProviderSlotSelectionStale
	}
	for _, candidate := range snapshot.Candidates {
		if candidate.TargetID == contract.ID && candidate.ID == candidateID {
			return cloneProviderSlotContract(*contract), candidate, nil
		}
	}
	return ProviderSlotContract{}, ProviderSlotCandidate{}, ErrProviderSlotSelectionStale
}

func exactProviderSlotSelectionIndex(resolution ProviderSlotResolution, selection ProviderSlotSelection) int {
	if selection.ContractID != resolution.Contract.ID || selection.ContractVersion != resolution.Contract.ContractVersion ||
		selection.Slot != resolution.Contract.Slot || !sameProviderArtifact(selection.ContractArtifact, resolution.Contract.Artifact) {
		return -1
	}
	for index, candidate := range resolution.Candidates {
		if candidate.ID == selection.CandidateID && sameProviderArtifact(selection.ProviderArtifact, candidate.Artifact) {
			return index
		}
	}
	return -1
}

func sameProviderArtifact(left, right HookArtifact) bool {
	return left.ExtensionID == right.ExtensionID && left.ExtensionVersion == right.ExtensionVersion && left.PackageDigest == right.PackageDigest
}

func validProviderSlotReason(reason string, required bool) bool {
	if reason == "" {
		return !required
	}
	if len(reason) > 128 {
		return false
	}
	for index, value := range reason {
		if (value >= 'a' && value <= 'z') || (value >= '0' && value <= '9') || value == '.' || value == '_' || value == '-' {
			continue
		}
		if index == 0 {
			return false
		}
		return false
	}
	return true
}

func providerSlotSelectionError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s provider slot selection: %w", operation, err)
}

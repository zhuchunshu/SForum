package extensionsruntime

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestManagerProbesExactProviderSlotCandidate(t *testing.T) {
	starter := newProviderInvocationStarter()
	starter.probe = func(_ context.Context, extensionID string, request ProviderProbeRequest) (ProviderProbeResponse, error) {
		return ProviderProbeResponse{
			OK: true, Reason: "provider.ready", Message: extensionID,
			Details: map[string]string{"slot": request.Slot}, Suggestions: []string{"none"},
		}, nil
	}
	manager := NewManager(ManagerConfig{Starter: starter})
	_, consumer := startProviderOwnerAndConsumer(t, manager, "next")
	result, err := manager.ProbeProviderSlotCandidate(t.Context(), providerSlotID, consumer.Manifest.Providers[0].ID)
	if err != nil || !result.OK || result.CandidateID != consumer.Manifest.Providers[0].ID ||
		result.Artifact.RuntimeInstanceID == "" || result.Details["slot"] != providerSlotID+".slot" {
		t.Fatalf("probe = %#v, %v", result, err)
	}
	result.Details["slot"] = "forged"
	result.Suggestions[0] = "forged"
	second, err := manager.ProbeProviderSlotCandidate(t.Context(), providerSlotID, consumer.Manifest.Providers[0].ID)
	if err != nil || second.Details["slot"] == "forged" || second.Suggestions[0] == "forged" {
		t.Fatalf("probe output was not isolated = %#v, %v", second, err)
	}
}

func TestManagerProviderSlotProbeRejectsStaleCandidate(t *testing.T) {
	manager := NewManager(ManagerConfig{Starter: newProviderInvocationStarter()})
	owner := versionedProviderExtension("providers.owner", strings.Repeat("a", 64), providerSlotDefinition(10))
	if err := manager.Start(t.Context(), owner); err != nil {
		t.Fatal(err)
	}
	snapshot := manager.HookBus().ProviderSlots().Snapshot()
	candidate := snapshot.Candidates[0]
	if err := manager.Stop(t.Context(), owner); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ProbeProviderSlotCandidate(t.Context(), providerSlotID, candidate.ID); !errors.Is(err, ErrProviderSlotSelectionStale) {
		t.Fatalf("stale candidate probe = %v", err)
	}
}

func (s *providerInvocationStarter) ProviderProbe(ctx context.Context, extensionID string, request ProviderProbeRequest) (ProviderProbeResponse, error) {
	if s.probe == nil {
		return ProviderProbeResponse{}, errors.New("provider probe is not configured")
	}
	return s.probe(ctx, extensionID, request)
}

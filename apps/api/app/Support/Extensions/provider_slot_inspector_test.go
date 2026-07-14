package extensionsruntime

import (
	"context"
	"reflect"
	"strings"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestProviderSlotInspectorPublishesExactAvailabilityFallbackAndConflicts(t *testing.T) {
	manager := NewManager(ManagerConfig{Starter: newManagerStagedStarter()})
	owner := versionedProviderExtension("providers.owner", strings.Repeat("a", 64), providerSlotDefinition(30))
	peer := versionedProviderExtension("providers.peer", strings.Repeat("b", 64), providerSlotConsumer("providers.peer.delivery", 30))
	peer.Manifest.Dependencies = []extensions.ManifestDependency{{ID: owner.ID, Version: "^1.0.0", Kind: "required"}}
	if err := manager.Start(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background(), peer); err != nil {
		t.Fatal(err)
	}
	peerRuntime, err := manager.ActiveRuntimeInstance(peer.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.BeginDrain(peerRuntime.Identity); err != nil {
		t.Fatal(err)
	}

	inspection := manager.ProviderSlotInspection()
	if inspection.Revision == 0 || len(inspection.Slots) != 1 {
		t.Fatalf("inspection = %#v", inspection)
	}
	slot := inspection.Slots[0]
	if slot.Contract.ID != providerSlotID || slot.Contract.Fallback != "next" || !slot.Contract.ContractRuntimeAvailable ||
		slot.Contract.Artifact.PackageDigest != owner.PackageDigest || slot.Contract.Artifact.RuntimeInstanceID == "" {
		t.Fatalf("contract inspection = %#v", slot.Contract)
	}
	if slot.Availability != providerSlotAvailable || slot.UnavailabilityReason != "" || len(slot.Candidates) != 2 {
		t.Fatalf("slot availability = %#v", slot)
	}
	if got := []string{slot.Candidates[0].ID, slot.Candidates[1].ID}; !reflect.DeepEqual(got, []string{owner.ID + ".delivery", peer.ID + ".delivery"}) {
		t.Fatalf("candidate order = %#v", got)
	}
	if slot.Candidates[0].Rank != 1 || slot.Candidates[0].Availability != providerSlotAvailable ||
		slot.Candidates[1].Rank != 2 || slot.Candidates[1].Availability != providerSlotUnavailable {
		t.Fatalf("candidate availability = %#v", slot.Candidates)
	}
	if len(slot.Conflicts) != 1 || slot.Conflicts[0].Kind != providerSlotPriorityTie ||
		!reflect.DeepEqual(slot.Conflicts[0].CandidateIDs, []string{owner.ID + ".delivery", peer.ID + ".delivery"}) {
		t.Fatalf("priority conflicts = %#v", slot.Conflicts)
	}
}

func TestProviderSlotInspectorReportsNoCandidate(t *testing.T) {
	manager := NewManager(ManagerConfig{Starter: newManagerStagedStarter()})
	owner := versionedProviderExtension("providers.owner", strings.Repeat("a", 64), providerSlotDefinition(30))
	owner.Manifest.Providers[0].Handler = ""
	if err := manager.Start(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	inspection := manager.ProviderSlotInspection()
	if len(inspection.Slots) != 1 || inspection.Slots[0].Availability != providerSlotUnavailable ||
		inspection.Slots[0].UnavailabilityReason != providerSlotNoCandidates || inspection.Slots[0].Candidates == nil {
		t.Fatalf("no-candidate inspection = %#v", inspection)
	}
}

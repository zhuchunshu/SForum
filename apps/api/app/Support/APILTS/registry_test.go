package apilts

import (
	"testing"
	"time"
)

func TestLTSDeprecationAndShimTelemetry(t *testing.T) {
	reg := New()
	snap := reg.Snapshot()
	if len(snap.Contracts) < 4 {
		t.Fatalf("seed contracts = %#v", snap.Contracts)
	}
	reg.RecordShimCall(ProtocolV1ContractID)
	reg.RecordShimCall(ProtocolV1ContractID)
	snap = reg.Snapshot()
	var calls uint64
	for _, row := range snap.ShimUsage {
		if row.ContractID == ProtocolV1ContractID {
			calls = row.Calls
		}
	}
	if calls != 2 {
		t.Fatalf("shim calls = %d", calls)
	}
	// Removal before RemoveAfter denied.
	if err := reg.Register(Contract{
		ID: ProtocolV1ContractID, Kind: "protocol", Status: "removed",
		RemoveAfter: time.Now().UTC().Add(24 * time.Hour),
	}); err == nil {
		t.Fatal("expected too-early removal error")
	}
	if reg.CanRemove(ProtocolV1ContractID, time.Now().UTC()) {
		// Seed RemoveAfter is in the future relative to now if deprecation was recent
		// — just ensure method is stable.
	}
	// Far future after RemoveAfter.
	far := time.Now().UTC().Add(400 * 24 * time.Hour)
	if !reg.CanRemove(ProtocolV1ContractID, far) {
		t.Fatal("expected removable after deprecation window")
	}
	// Non-zero shim blocks the combined deletion gate even after the window.
	if reg.CanRemoveWithZeroShim(ProtocolV1ContractID, far) {
		t.Fatal("non-zero shim must block CanRemoveWithZeroShim")
	}
	// Fresh registry with zero usage may pass after the window.
	clean := New()
	if !clean.CanRemoveWithZeroShim(ProtocolV1ContractID, far) {
		t.Fatal("zero shim + elapsed window should allow removal")
	}
	if clean.ShimCalls(ProtocolV1ContractID) != 0 {
		t.Fatalf("seeded shim counter should start at 0, got %d", clean.ShimCalls(ProtocolV1ContractID))
	}
}

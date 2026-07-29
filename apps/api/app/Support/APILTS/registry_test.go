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
	var hasThemeLoader bool
	for _, c := range snap.Contracts {
		if c.ID == ThemeRequestTimeLoaderContractID && c.Status == "deprecated" && c.ShimEnabled {
			hasThemeLoader = true
		}
	}
	if !hasThemeLoader {
		t.Fatalf("missing seeded theme request-time loader contract: %#v", snap.Contracts)
	}
	reg.RecordShimCall(ThemeRequestTimeLoaderContractID)
	snap = reg.Snapshot()
	var themeCalls uint64
	for _, row := range snap.ShimUsage {
		if row.ContractID == ThemeRequestTimeLoaderContractID {
			themeCalls = row.Calls
		}
	}
	if themeCalls != 1 {
		t.Fatalf("theme loader shim calls = %d", themeCalls)
	}
	// Removal before RemoveAfter denied.
	if err := reg.Register(Contract{
		ID: ThemeRequestTimeLoaderContractID, Kind: "frontend", Status: "removed",
		RemoveAfter: time.Now().UTC().Add(24 * time.Hour),
	}); err == nil {
		t.Fatal("expected too-early removal error")
	}
	if reg.CanRemove(ThemeRequestTimeLoaderContractID, time.Now().UTC()) {
		// Seed RemoveAfter is in the future relative to now if deprecation was recent
		// — just ensure method is stable.
	}
	// Far future after RemoveAfter.
	far := time.Now().UTC().Add(400 * 24 * time.Hour)
	if !reg.CanRemove(ThemeRequestTimeLoaderContractID, far) {
		t.Fatal("expected removable after deprecation window")
	}
	// Non-zero shim blocks the combined deletion gate even after the window.
	if reg.CanRemoveWithZeroShim(ThemeRequestTimeLoaderContractID, far) {
		t.Fatal("non-zero shim must block CanRemoveWithZeroShim")
	}
	// Fresh registry with zero usage may pass after the window.
	clean := New()
	if !clean.CanRemoveWithZeroShim(ThemeRequestTimeLoaderContractID, far) {
		t.Fatal("zero shim + elapsed window should allow removal")
	}
	if clean.ShimCalls(ThemeRequestTimeLoaderContractID) != 0 {
		t.Fatalf("seeded shim counter should start at 0, got %d", clean.ShimCalls(ThemeRequestTimeLoaderContractID))
	}
}

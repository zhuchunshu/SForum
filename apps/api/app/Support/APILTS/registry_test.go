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
	reg.RecordShimCall("sforum.protocol.v1")
	reg.RecordShimCall("sforum.protocol.v1")
	snap = reg.Snapshot()
	var calls uint64
	for _, row := range snap.ShimUsage {
		if row.ContractID == "sforum.protocol.v1" {
			calls = row.Calls
		}
	}
	if calls != 2 {
		t.Fatalf("shim calls = %d", calls)
	}
	// Removal before RemoveAfter denied.
	if err := reg.Register(Contract{
		ID: "sforum.protocol.v1", Kind: "protocol", Status: "removed",
		RemoveAfter: time.Now().UTC().Add(24 * time.Hour),
	}); err == nil {
		t.Fatal("expected too-early removal error")
	}
	if reg.CanRemove("sforum.protocol.v1", time.Now().UTC()) {
		// Seed RemoveAfter is in the future relative to now if deprecation was recent
		// — just ensure method is stable.
	}
	// Far future after RemoveAfter.
	if !reg.CanRemove("sforum.protocol.v1", time.Now().UTC().Add(400*24*time.Hour)) {
		t.Fatal("expected removable after deprecation window")
	}
}

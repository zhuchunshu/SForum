package apilts

import "testing"

func TestProcessRegistryRecordsProtocolV1Shim(t *testing.T) {
	reg := New()
	ResetProcessForTest(reg)
	t.Cleanup(func() { ResetProcessForTest(nil) })

	Process().RecordShimCall(ProtocolV1ContractID)
	Process().RecordShimCall(ProtocolV1ContractID)

	var calls uint64
	for _, row := range Process().Snapshot().ShimUsage {
		if row.ContractID == ProtocolV1ContractID {
			calls = row.Calls
		}
	}
	if calls != 2 {
		t.Fatalf("process shim calls = %d", calls)
	}
}

func TestProcessCreatesSeededRegistry(t *testing.T) {
	ResetProcessForTest(nil)
	t.Cleanup(func() { ResetProcessForTest(nil) })

	snap := Process().Snapshot()
	if snap.SchemaVersion != SchemaVersion {
		t.Fatalf("schema = %q", snap.SchemaVersion)
	}
	found := false
	for _, c := range snap.Contracts {
		if c.ID == ProtocolV1ContractID && c.Status == "deprecated" && c.ShimEnabled {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing seeded protocol v1: %#v", snap.Contracts)
	}
}

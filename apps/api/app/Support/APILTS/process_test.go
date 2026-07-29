package apilts

import "testing"

func TestProcessRegistryRecordsThemeRequestTimeLoaderShim(t *testing.T) {
	reg := New()
	ResetProcessForTest(reg)
	t.Cleanup(func() { ResetProcessForTest(nil) })

	Process().RecordShimCall(ThemeRequestTimeLoaderContractID)
	Process().RecordShimCall(ThemeRequestTimeLoaderContractID)

	var calls uint64
	for _, row := range Process().Snapshot().ShimUsage {
		if row.ContractID == ThemeRequestTimeLoaderContractID {
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
		if c.ID == ThemeRequestTimeLoaderContractID && c.Status == "deprecated" && c.ShimEnabled {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing seeded theme request-time loader: %#v", snap.Contracts)
	}
}

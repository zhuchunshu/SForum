package storage

import "testing"

func TestDriverCatalogAndNormalize(t *testing.T) {
	if ProviderSlot != "attachment.storage.provider" {
		t.Fatalf("slot=%s", ProviderSlot)
	}
	if !IsKnownDriver("") || !IsKnownDriver(ProviderLocal) {
		t.Fatal("local should be known")
	}
	if IsKnownDriver("unknown-cloud") {
		t.Fatal("unknown driver")
	}
	if NormalizeProvider("") != ProviderLocal {
		t.Fatal("blank → local")
	}
	if got := len(DriverCatalog()); got != 5 {
		t.Fatalf("drivers=%d", got)
	}
}

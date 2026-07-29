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
	if got := len(DriverCatalog()); got != 1 {
		t.Fatalf("drivers=%d", got)
	}
	// plugin: 不是 core 驱动；Normalize 保留前缀供 ParseSelection 使用。
	if IsKnownDriver("plugin:sforum.s3") {
		t.Fatal("plugin selection must not count as core driver")
	}
	if NormalizeProvider("plugin:sforum.s3") != "plugin:sforum.s3" {
		t.Fatal("normalize must preserve plugin selection")
	}
}

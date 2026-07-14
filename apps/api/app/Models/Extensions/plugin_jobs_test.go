package extensions

import (
	"context"
	"errors"
	"testing"
)

func TestServicePluginJobContractUsesInstalledArtifact(t *testing.T) {
	item := Extension{
		ID: "demo.plugin", Version: "1.2.3", Type: TypePlugin, Status: StatusEnabled,
		PackageDigest: "digest-v1",
		Manifest: Manifest{Jobs: []ManifestJob{{
			ID: "demo.plugin.job.sync", ContractVersion: "demo.plugin.job.sync@2",
			Name: "demo.sync", PayloadSchema: "demo.plugin.sync.payload@3",
			RetryPolicy: "exponential", MaxAttempts: 8, ConcurrencyLimit: 2,
		}}},
	}
	service := NewService(newFakeExtensionStore(map[string]Extension{item.ID: item}), t.TempDir())
	contract, err := service.PluginJobContract(context.Background(), item.ID, "demo.sync")
	if err != nil {
		t.Fatal(err)
	}
	if contract.ExtensionVersion != item.Version || contract.ArtifactDigest != item.PackageDigest ||
		contract.JobContract != "demo.plugin.job.sync@2" || contract.PayloadSchemaID != "demo.plugin.sync.payload" ||
		contract.PayloadSchemaVersion != "3" || contract.RetryPolicy != "exponential" ||
		contract.MaxAttempts != 8 || contract.ConcurrencyLimit != 2 {
		t.Fatalf("contract = %#v", contract)
	}

	item.Status = StatusDisabled
	service = NewService(newFakeExtensionStore(map[string]Extension{item.ID: item}), t.TempDir())
	if _, err := service.PluginJobContract(context.Background(), item.ID, "demo.sync"); !errors.Is(err, ErrExtensionDisabled) {
		t.Fatalf("disabled contract error = %v", err)
	}
}

package hostapi

import (
	"errors"
	"testing"
)

func TestServiceRegistryExtensionSnapshotIsExactAndCallerOwned(t *testing.T) {
	registry := NewServiceRegistry()
	provider := &registryTestProvider{id: "snapshot"}
	if err := registry.ReplaceExtension("demo.plugin", []ServiceRegistration{
		serviceRegistration("demo.plugin", "runtime-2", "demo.alpha", "1.0.0", 0, provider),
		serviceRegistration("demo.plugin", "runtime-2", "demo.beta", "1.0.0", 0, provider),
	}); err != nil {
		t.Fatal(err)
	}

	snapshot, ok, err := registry.ExtensionSnapshot("demo.plugin")
	if err != nil || !ok || snapshot.Revision != 1 || snapshot.InstanceID != "runtime-2" || len(snapshot.Registrations) != 2 {
		t.Fatalf("extension snapshot = %#v, %t, %v", snapshot, ok, err)
	}
	snapshot.Registrations[0].Descriptor.ServiceId = "forged"
	again, ok, err := registry.ExtensionSnapshot("demo.plugin")
	if err != nil || !ok || again.Registrations[0].Descriptor.GetServiceId() == "forged" {
		t.Fatalf("caller mutated registry snapshot = %#v, %t, %v", again, ok, err)
	}
}

func TestServiceRegistryExtensionSnapshotRejectsMixedRuntimeOwnership(t *testing.T) {
	registry := NewServiceRegistry()
	provider := &registryTestProvider{id: "mixed"}
	if err := registry.ReplaceExtension("demo.plugin", []ServiceRegistration{
		serviceRegistration("demo.plugin", "runtime-1", "demo.alpha", "1.0.0", 0, provider),
		serviceRegistration("demo.plugin", "runtime-2", "demo.beta", "1.0.0", 0, provider),
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := registry.ExtensionSnapshot("demo.plugin"); !errors.Is(err, ErrInvalidServiceRegistration) {
		t.Fatalf("mixed runtime snapshot error = %v", err)
	}
}

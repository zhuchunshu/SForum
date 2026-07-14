package extensionmanifest

import "testing"

func TestManifestV3NormalizesProviderSlotDefaults(t *testing.T) {
	manifest := completeV3Manifest()
	manifest.Providers[0].Fallback = ""
	manifest.Providers[0].TimeoutMS = 0
	normalized := Normalize(manifest)
	provider := normalized.Providers[0]
	if provider.Fallback != "next" || provider.TimeoutMS != ProviderSlotMaximumTimeoutMS {
		t.Fatalf("provider defaults = %#v", provider)
	}
	if err := Validate(normalized); err != nil {
		t.Fatal(err)
	}
}

func TestManifestV3ProviderSlotSupportsPassiveDefinitionAndDependencyTarget(t *testing.T) {
	manifest := completeV3Manifest()
	manifest.Providers = []ManifestProvider{{
		ID: "demo.v3.provider.slot", ContractVersion: "demo.v3.provider.slot@1",
		Slot: "demo.v3.delivery", Label: "Delivery", RequestSchema: "demo.v3.delivery.request@1",
		ResponseSchema: "demo.v3.delivery.response@1", Fallback: "next", TimeoutMS: 1000,
	}}
	if err := Validate(manifest); err != nil {
		t.Fatalf("passive slot definition: %v", err)
	}
	manifest.Providers[0] = ManifestProvider{
		ID: "demo.v3.provider.consumer", ContractVersion: "owner.provider.slot@1",
		Slot: "owner.delivery", TargetID: "owner.provider.slot", Label: "Consumer",
		Handler: "provider.delivery", RequestSchema: "owner.delivery.request@1",
		ResponseSchema: "owner.delivery.response@1", Fallback: "next", TimeoutMS: 1000,
	}
	manifest.Dependencies = append(manifest.Dependencies, ManifestDependency{ID: "owner", Version: "^1.0.0", Kind: "required"})
	if err := Validate(manifest); err != nil {
		t.Fatalf("targeted provider: %v", err)
	}
	manifest.Providers[0].Handler = ""
	if err := Validate(manifest); err == nil {
		t.Fatal("targeted provider without handler must fail")
	}
}

func TestManifestV3ProviderSlotRejectsUntypedAndUnboundedContracts(t *testing.T) {
	manifest := completeV3Manifest()
	manifest.Providers[0].RequestSchema = ""
	if err := Validate(manifest); err == nil {
		t.Fatal("custom provider without request schema must fail")
	}
	manifest = completeV3Manifest()
	manifest.Providers[0].TimeoutMS = ProviderSlotMaximumTimeoutMS + 1
	if err := Validate(manifest); err == nil {
		t.Fatal("provider timeout above Host deadline must fail")
	}
	manifest = completeV3Manifest()
	manifest.Providers[0].Fallback = "unsafe"
	if err := Validate(manifest); err == nil {
		t.Fatal("unknown provider fallback must fail")
	}
}

func TestManifestV3KeepsLegacyHostProviderTimeoutSemantics(t *testing.T) {
	manifest := completeV3Manifest()
	manifest.Providers = []ManifestProvider{{
		ID: "demo.v3.provider.mail", ContractVersion: "demo.v3.provider.mail@1",
		Slot: "mail.provider", Label: "Mail", Handler: "provider.mail", TimeoutMS: 15000,
	}}
	normalized := Normalize(manifest)
	if normalized.Providers[0].TimeoutMS != 15000 || normalized.Providers[0].Fallback != "" {
		t.Fatalf("legacy provider changed = %#v", normalized.Providers[0])
	}
	if err := Validate(normalized); err != nil {
		t.Fatalf("legacy provider compatibility: %v", err)
	}
}

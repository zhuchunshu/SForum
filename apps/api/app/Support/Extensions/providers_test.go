package extensionsruntime

import "testing"

func TestProviderRegistryKeepsSelectsAndRestoresDefault(t *testing.T) {
	registry := NewProviderRegistry()
	if selected := registry.Selected("search.provider"); selected.ExtensionID != "" || selected.Label != "Built-in search" {
		t.Fatalf("expected built-in default, got %#v", selected)
	}
	registry.Select("search.provider", ProviderSelection{ExtensionID: "demo.plugin", Label: "Demo Search"})
	if selected := registry.Selected("search.provider"); selected.ExtensionID != "demo.plugin" {
		t.Fatalf("expected plugin provider, got %#v", selected)
	}
	registry.RestoreDefault("search.provider")
	if selected := registry.Selected("search.provider"); selected.ExtensionID != "" || selected.Label != "Built-in search" {
		t.Fatalf("expected restored built-in default, got %#v", selected)
	}
}

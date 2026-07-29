package storage

import "testing"

func TestFormatAndParsePluginSelection(t *testing.T) {
	raw := FormatPluginSelection("sforum.s3")
	if raw != "plugin:sforum.s3" {
		t.Fatalf("format: %q", raw)
	}
	sel := ParseSelection(raw)
	if sel.Kind != SelectionKindPlugin || sel.ExtensionID != "sforum.s3" {
		t.Fatalf("parse plugin: %#v", sel)
	}
	if !sel.IsValidPluginSelection() || sel.IsCoreDriverSelection() {
		t.Fatalf("flags: %#v", sel)
	}
}

func TestParseSelectionCoreDrivers(t *testing.T) {
	for _, tc := range []struct {
		in, driver string
	}{
		{"", ProviderLocal},
		{"local", ProviderLocal},
		{"  local ", ProviderLocal},
	} {
		sel := ParseSelection(tc.in)
		if sel.Kind != SelectionKindCore || sel.Driver != tc.driver {
			t.Fatalf("in=%q got %#v want driver %q", tc.in, sel, tc.driver)
		}
		if !sel.IsCoreDriverSelection() {
			t.Fatalf("expected core: %#v", sel)
		}
	}
}

func TestIsPluginSelection(t *testing.T) {
	if !IsPluginSelection("plugin:acme.store") {
		t.Fatal("expected plugin")
	}
	if IsPluginSelection("local") || IsPluginSelection("plugin") {
		t.Fatal("core must not look like plugin without colon form")
	}
}

func TestParseSelectionEmptyPluginID(t *testing.T) {
	sel := ParseSelection("plugin:")
	if sel.Kind != SelectionKindPlugin || sel.IsValidPluginSelection() {
		t.Fatalf("empty ext id should be invalid plugin selection: %#v", sel)
	}
}

package localization

import "testing"

func TestParseSupportedLocalesDefaults(t *testing.T) {
	locales := ParseSupportedLocales("")
	if len(locales) != 2 {
		t.Fatalf("expected default locales, got %v", locales)
	}
	if locales[0] != "zh-CN" || locales[1] != "en-US" {
		t.Fatalf("unexpected default locale order: %v", locales)
	}
}

func TestNormalizeUsesAliasesAndFallback(t *testing.T) {
	supported := []string{"zh-CN", "en-US"}

	if got := Normalize("en", supported); got != "en-US" {
		t.Fatalf("expected en-US, got %s", got)
	}

	if got := Normalize("fr-FR", supported); got != "zh-CN" {
		t.Fatalf("expected zh-CN fallback, got %s", got)
	}
}

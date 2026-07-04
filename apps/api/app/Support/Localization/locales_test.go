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

func TestMessageReturnsLocalizedAPIMessages(t *testing.T) {
	if got := Message("zh-CN", "auth.required"); got != "请先登录。" {
		t.Fatalf("expected Chinese auth message, got %q", got)
	}

	if got := Message("en-US", "auth.required"); got != "Please sign in first." {
		t.Fatalf("expected English auth message, got %q", got)
	}
}

func TestMessageFallsBackToDefaultLocaleAndKey(t *testing.T) {
	if got := Message("fr-FR", "permission.denied"); got != "没有权限执行此操作。" {
		t.Fatalf("expected default-locale fallback, got %q", got)
	}

	if got := Message("en-US", "unknown.reason"); got != "unknown.reason" {
		t.Fatalf("expected unknown key fallback, got %q", got)
	}
}

func TestNegotiateAcceptLanguage(t *testing.T) {
	supported := []string{"zh-CN", "en-US"}

	if got := NegotiateAcceptLanguage("en-US,en;q=0.9,zh-CN;q=0.8", supported, "zh-CN"); got != "en-US" {
		t.Fatalf("expected en-US, got %q", got)
	}

	if got := NegotiateAcceptLanguage("fr-FR,en;q=0.9", supported, "zh-CN"); got != "en-US" {
		t.Fatalf("expected en-US after unsupported locale, got %q", got)
	}

	if got := NegotiateAcceptLanguage("fr-FR,zh;q=0.9", supported, "zh-CN"); got != "zh-CN" {
		t.Fatalf("expected zh-CN fallback, got %q", got)
	}
}

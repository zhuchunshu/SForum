package identityregistry

import "testing"

func TestResolveProviderLabelUsesPluginLocales(t *testing.T) {
	provider := Provider{
		ID:    "sforum.auth-github.auth",
		Label: "GitHub",
		LabelLocales: map[string]string{
			"zh-CN": "GitHub",
			"en-US": "GitHub",
		},
		Icon: "i-tabler-brand-github",
	}
	if got := ResolveProviderLabel(provider, "zh-CN"); got != "GitHub" {
		t.Fatalf("zh-CN label: got %q", got)
	}
	if got := ResolveProviderLabel(provider, "en-US"); got != "GitHub" {
		t.Fatalf("en-US label: got %q", got)
	}
	// Host 不得在此回退到任何硬编码品牌；无 locale 时用 Label。
	if got := ResolveProviderLabel(Provider{Label: "Other"}, "fr-FR"); got != "Other" {
		t.Fatalf("fallback label: got %q", got)
	}
	if got := ResolveProviderLabel(Provider{}, "zh-CN"); got != "" {
		t.Fatalf("empty provider must not invent a brand label, got %q", got)
	}
}

func TestCloneProviderCopiesPresentationFields(t *testing.T) {
	src := Provider{
		ID:           "demo.auth",
		Label:        "Demo",
		LabelLocales: map[string]string{"zh-CN": "演示"},
		Icon:         "i-lucide-key-round",
	}
	cloned := cloneProvider(src)
	cloned.LabelLocales["zh-CN"] = "mutated"
	if src.LabelLocales["zh-CN"] != "演示" {
		t.Fatal("cloneProvider must deep-copy LabelLocales")
	}
	if cloned.Icon != "i-lucide-key-round" || cloned.Label != "Demo" {
		t.Fatalf("presentation fields not cloned: %#v", cloned)
	}
}

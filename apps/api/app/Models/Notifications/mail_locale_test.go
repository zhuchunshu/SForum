package notifications

import (
	"context"
	"errors"
	"testing"
)

type staticMailLocaleResolver struct {
	locale string
	err    error
}

func (r staticMailLocaleResolver) DefaultMailLocale(context.Context) (string, error) {
	return r.locale, r.err
}

func TestResolveMailLocaleUsesAccountThenSiteDefault(t *testing.T) {
	resolver := staticMailLocaleResolver{locale: "en-US"}
	if got := resolveMailLocale(context.Background(), "zh-CN", resolver); got != "zh-CN" {
		t.Fatalf("account locale = %q, want zh-CN", got)
	}
	if got := resolveMailLocale(context.Background(), "", resolver); got != "en-US" {
		t.Fatalf("site fallback = %q, want en-US", got)
	}
	if got := resolveMailLocale(context.Background(), "", staticMailLocaleResolver{err: errors.New("unavailable")}); got != "zh-CN" {
		t.Fatalf("hard fallback = %q, want zh-CN", got)
	}
}

func TestMailBrandUsesSiteAssetsAndAppearanceTheme(t *testing.T) {
	brand := MailBrandFromSettings("Blue Forum", "https://forum.test", "/brand/logo.png", "/brand/icon.png", "ocean_blue")
	if brand.LogoURL != "https://forum.test/brand/logo.png" || brand.IconURL != "https://forum.test/brand/icon.png" {
		t.Fatalf("brand asset URLs = %#v", brand)
	}
	if brand.AccentColor != "#2563eb" || brand.AccentSoft != "#eff6ff" {
		t.Fatalf("blue appearance = %#v", brand)
	}
	custom := MailBrandFromSettings("自定义论坛", "https://forum.test", "", "", "custom:#4f46e5")
	if custom.Mark != "自" || custom.AccentColor != "#4f46e5" || custom.AccentSoft == "" {
		t.Fatalf("custom appearance = %#v", custom)
	}
}

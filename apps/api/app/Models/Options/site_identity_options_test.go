package options

import (
	"context"
	"testing"
	"time"
)

func TestSiteIdentityOptionsDefaults(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)

	publicItems, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if got := adminValueFromPublic(publicItems, NameSiteTagline); got != "" {
		t.Fatalf("expected empty public tagline default, got %q", got)
	}
	if got := adminValueFromPublic(publicItems, NameSiteAboutURL); got != "" {
		t.Fatalf("expected empty public about URL default, got %q", got)
	}
	if got := adminValueFromPublic(publicItems, NameSiteAboutOpenInNewTab); got != "disabled" {
		t.Fatalf("expected about new-tab default disabled, got %q", got)
	}
	// admin_email 非 public，公开列表不应出现。
	for _, item := range publicItems {
		if item.Name == NameSiteAdminEmail {
			t.Fatalf("public list must not expose site.admin_email")
		}
	}

	adminItems, err := service.ListAdmin(context.Background(), settingsActor())
	if err != nil {
		t.Fatalf("ListAdmin returned error: %v", err)
	}
	if got := adminValue(adminItems, NameSiteAdminEmail); got != "" {
		t.Fatalf("expected empty admin email default, got %q", got)
	}
}

func TestSiteURLFallsBackToEnvironmentDefault(t *testing.T) {
	store := &fakeStore{items: map[string]string{}}
	service := NewServiceWithDefaultsAndCacheTTL(store, Defaults{
		SiteURL: "https://env.example.com",
	}, time.Minute)

	publicURL, err := service.WebOption(context.Background(), NameSiteURL)
	if err != nil {
		t.Fatalf("WebOption returned error: %v", err)
	}
	if publicURL != "https://env.example.com" {
		t.Fatalf("public site URL = %q, want environment fallback", publicURL)
	}

	adminItems, err := service.ListAdmin(context.Background(), settingsActor())
	if err != nil {
		t.Fatalf("ListAdmin returned error: %v", err)
	}
	option := adminSecret(adminItems, NameSiteURL)
	if option.Value != "https://env.example.com" || option.OverrideValue == nil || *option.OverrideValue != "" || !option.Inherited {
		t.Fatalf("admin site URL should expose an empty inherited override: %#v", option)
	}
	if option.FallbackValue != "https://env.example.com" {
		t.Fatalf("admin site URL fallback metadata mismatch: %#v", option)
	}
}

func TestEnsureDefaultsClearsLegacyMaterializedSiteURL(t *testing.T) {
	store := &fakeStore{items: map[string]string{NameSiteURL: "https://env.example.com"}}
	service := NewServiceWithDefaultsAndCacheTTL(store, Defaults{
		SiteURL: "https://env.example.com",
	}, time.Minute)

	if err := service.EnsureDefaults(context.Background()); err != nil {
		t.Fatalf("EnsureDefaults returned error: %v", err)
	}
	if store.items[NameSiteURL] != "" {
		t.Fatalf("legacy materialized site URL should become an empty override, got %q", store.items[NameSiteURL])
	}
}

func TestSiteURLOverrideCanBeCleared(t *testing.T) {
	store := &fakeStore{items: map[string]string{NameSiteURL: "https://custom.example.com"}}
	service := NewServiceWithDefaultsAndCacheTTL(store, Defaults{
		SiteURL: "https://env.example.com",
	}, time.Minute)

	if got, err := service.WebOption(context.Background(), NameSiteURL); err != nil || got != "https://custom.example.com" {
		t.Fatalf("custom site URL = %q, err = %v", got, err)
	}

	updated, err := service.UpdateMany(context.Background(), settingsActor(), []UpdateInput{
		{Name: NameSiteURL, Value: ""},
	})
	if err != nil {
		t.Fatalf("clearing site URL override returned error: %v", err)
	}
	option := adminSecret(updated, NameSiteURL)
	if option.Value != "https://env.example.com" || option.OverrideValue == nil || *option.OverrideValue != "" || !option.Inherited {
		t.Fatalf("cleared site URL should inherit environment fallback: %#v", option)
	}
	if store.items[NameSiteURL] != "" {
		t.Fatalf("cleared site URL override should be stored as empty, got %q", store.items[NameSiteURL])
	}
}

func TestSiteDomainDefaultsFromSiteURLAndNormalizesInput(t *testing.T) {
	store := &fakeStore{items: map[string]string{}}
	service := NewServiceWithDefaultsAndCacheTTL(store, Defaults{
		SiteURL: "https://Forum.Example.com:8443/base",
	}, time.Minute)

	domain, err := service.WebOption(context.Background(), NameSiteDomain)
	if err != nil {
		t.Fatalf("WebOption returned error: %v", err)
	}
	if domain != "forum.example.com:8443" {
		t.Fatalf("default site domain = %q, want forum.example.com:8443", domain)
	}

	items, err := service.UpdateMany(context.Background(), settingsActor(), []UpdateInput{
		{Name: NameSiteDomain, Value: "  HTTPS://Community.Example.com///  "},
	})
	if err != nil {
		t.Fatalf("UpdateMany returned error: %v", err)
	}
	if got := adminValue(items, NameSiteDomain); got != "community.example.com" {
		t.Fatalf("normalized admin site domain = %q", got)
	}
	if got := store.items[NameSiteDomain]; got != "community.example.com" {
		t.Fatalf("stored site domain = %q", got)
	}
}

func TestSiteDomainRejectsNonDomainValues(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)
	for _, value := range []string{"", "ftp://forum.example.com", "forum.example.com/community", "forum.example.com?preview=1"} {
		if _, err := service.Update(context.Background(), settingsActor(), UpdateInput{Name: NameSiteDomain, Value: value}); err == nil {
			t.Fatalf("expected site domain %q to be rejected", value)
		}
	}
}

func TestSiteIdentityOptionsAcceptValidValues(t *testing.T) {
	store := &fakeStore{items: map[string]string{}}
	service := NewServiceWithCacheTTL(store, time.Minute)
	actor := settingsActor()

	_, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameSiteTagline, Value: "  一个友好的社区  "},
		{Name: NameSiteAboutURL, Value: "  /guidelines  "},
		{Name: NameSiteAboutOpenInNewTab, Value: "true"},
		{Name: NameSiteAdminEmail, Value: "ops@example.com"},
	})
	if err != nil {
		t.Fatalf("UpdateMany returned error: %v", err)
	}
	if store.items[NameSiteTagline] != "一个友好的社区" {
		t.Fatalf("tagline not trimmed/saved: %#v", store.items)
	}
	if store.items[NameSiteAboutURL] != "/guidelines" {
		t.Fatalf("about URL not trimmed/saved: %#v", store.items)
	}
	if store.items[NameSiteAboutOpenInNewTab] != "enabled" {
		t.Fatalf("about new-tab flag not normalized/saved: %#v", store.items)
	}
	if store.items[NameSiteAdminEmail] != "ops@example.com" {
		t.Fatalf("admin email not saved: %#v", store.items)
	}

	// AdminEmail helper 供 mail-test 等内部路径读取。
	got, err := service.AdminEmail(context.Background())
	if err != nil {
		t.Fatalf("AdminEmail returned error: %v", err)
	}
	if got != "ops@example.com" {
		t.Fatalf("AdminEmail = %q, want ops@example.com", got)
	}
}

func TestSiteIdentityOptionsRejectInvalidValues(t *testing.T) {
	store := &fakeStore{items: map[string]string{}}
	service := NewServiceWithCacheTTL(store, time.Minute)
	actor := settingsActor()

	longTagline := stringsRepeat("标", siteTaglineMaxRunes+1)
	cases := []UpdateInput{
		{Name: NameSiteTagline, Value: longTagline},
		{Name: NameSiteAboutURL, Value: "javascript:alert(1)"},
		{Name: NameSiteAboutURL, Value: "/api/v1/users"},
		{Name: NameSiteAboutOpenInNewTab, Value: "maybe"},
		{Name: NameSiteAdminEmail, Value: "not-an-email"},
		{Name: NameSiteAdminEmail, Value: "Name <ops@example.com>"},
	}
	for _, input := range cases {
		if _, err := service.UpdateMany(context.Background(), actor, []UpdateInput{input}); err == nil {
			t.Fatalf("expected rejection for %s=%q", input.Name, input.Value)
		}
	}

	// 清空邮箱应允许。
	if _, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameSiteAdminEmail, Value: ""},
	}); err != nil {
		t.Fatalf("empty admin email should be allowed: %v", err)
	}
}

func stringsRepeat(s string, n int) string {
	out := make([]rune, 0, n)
	r := []rune(s)
	for i := 0; i < n; i++ {
		out = append(out, r...)
	}
	return string(out)
}

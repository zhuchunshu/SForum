package options

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSiteAttachmentReferenceContexts(t *testing.T) {
	for name, want := range map[string]string{
		NameSiteLogoAttachmentID:           "logo",
		NameSiteFaviconAttachmentID:        "favicon",
		NameSiteAppleTouchIconAttachmentID: "apple-touch-icon",
	} {
		got, ok := siteAttachmentReferenceContext(name)
		if !ok || got != want {
			t.Fatalf("%s context=%q ok=%v", name, got, ok)
		}
	}
	if _, ok := siteAttachmentReferenceContext(NameSiteLogoURL); ok {
		t.Fatal("plain URL option must not create an attachment reference")
	}
}

func TestSiteBrandOptionsDefaults(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)

	publicItems, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	// 品牌资源默认空 → 主题自带图标。
	for _, name := range []string{
		NameSiteLogoURL,
		NameSiteLogoAttachmentID,
		NameSiteFaviconURL,
		NameSiteFaviconAttachmentID,
		NameSiteAppleTouchIconURL,
		NameSiteAppleTouchIconAttachmentID,
	} {
		if got := adminValueFromPublic(publicItems, name); got != "" {
			t.Fatalf("expected empty public default for %s, got %q", name, got)
		}
	}

	// 法律 stub 默认非空且 public。
	if got := adminValueFromPublic(publicItems, NameLegalTermsBodyZHCN); !strings.Contains(got, "服务条款") {
		t.Fatalf("expected zh-CN terms stub, got %q", got)
	}
	if got := adminValueFromPublic(publicItems, NameLegalTermsBodyENUS); !strings.Contains(got, "Terms of Service") {
		t.Fatalf("expected en-US terms stub, got %q", got)
	}
	if got := adminValueFromPublic(publicItems, NameLegalPrivacyBodyZHCN); !strings.Contains(got, "隐私政策") {
		t.Fatalf("expected zh-CN privacy stub, got %q", got)
	}
	if got := adminValueFromPublic(publicItems, NameLegalGuidelinesBodyENUS); !strings.Contains(got, "Community Guidelines") {
		t.Fatalf("expected en-US guidelines stub, got %q", got)
	}
}

func TestSiteBrandOptionsAcceptValidValues(t *testing.T) {
	store := &fakeStore{items: map[string]string{}}
	service := NewServiceWithCacheTTL(store, time.Minute)
	actor := settingsActor()

	_, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameSiteLogoURL, Value: "  https://cdn.example.com/logo.png  "},
		{Name: NameSiteLogoAttachmentID, Value: " 42 "},
		{Name: NameSiteFaviconURL, Value: "/favicon.ico"},
		{Name: NameSiteAppleTouchIconURL, Value: "https://cdn.example.com/apple-touch.png"},
		{Name: NameLegalTermsBodyZHCN, Value: "  自定义条款  "},
		{Name: NameLegalPrivacyBodyENUS, Value: ""}, // 允许清空
	})
	if err != nil {
		t.Fatalf("UpdateMany returned error: %v", err)
	}

	if store.items[NameSiteLogoURL] != "https://cdn.example.com/logo.png" {
		t.Fatalf("logo url not trimmed: %#v", store.items[NameSiteLogoURL])
	}
	if store.items[NameSiteLogoAttachmentID] != "42" {
		t.Fatalf("logo attachment id not normalized: %#v", store.items[NameSiteLogoAttachmentID])
	}
	if store.items[NameSiteFaviconURL] != "/favicon.ico" {
		t.Fatalf("relative favicon not accepted: %#v", store.items[NameSiteFaviconURL])
	}
	if store.items[NameLegalTermsBodyZHCN] != "自定义条款" {
		t.Fatalf("legal body not trimmed: %#v", store.items[NameLegalTermsBodyZHCN])
	}
	if store.items[NameLegalPrivacyBodyENUS] != "" {
		t.Fatalf("empty legal body should clear: %#v", store.items[NameLegalPrivacyBodyENUS])
	}
}

func TestSiteBrandOptionsRejectInvalidValues(t *testing.T) {
	store := &fakeStore{items: map[string]string{}}
	service := NewServiceWithCacheTTL(store, time.Minute)
	actor := settingsActor()

	cases := []UpdateInput{
		{Name: NameSiteLogoURL, Value: "javascript:alert(1)"},
		{Name: NameSiteLogoURL, Value: "//evil.example/logo.png"},
		{Name: NameSiteLogoAttachmentID, Value: "0"},
		{Name: NameSiteLogoAttachmentID, Value: "-1"},
		{Name: NameSiteLogoAttachmentID, Value: "not-a-number"},
		{Name: NameSiteFaviconURL, Value: "ftp://files.example/favicon.ico"},
		{Name: NameLegalTermsBodyZHCN, Value: strings.Repeat("法", legalBodyMaxRunes+1)},
	}
	for _, input := range cases {
		if _, err := service.UpdateMany(context.Background(), actor, []UpdateInput{input}); err == nil {
			t.Fatalf("expected rejection for %s=%q", input.Name, truncateForTest(input.Value, 40))
		}
	}

	// 清空品牌字段应允许。
	if _, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameSiteLogoURL, Value: ""},
		{Name: NameSiteLogoAttachmentID, Value: ""},
	}); err != nil {
		t.Fatalf("empty brand assets should be allowed: %v", err)
	}
}

func truncateForTest(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "…"
}

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

func TestSiteIdentityOptionsAcceptValidValues(t *testing.T) {
	store := &fakeStore{items: map[string]string{}}
	service := NewServiceWithCacheTTL(store, time.Minute)
	actor := settingsActor()

	_, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameSiteTagline, Value: "  一个友好的社区  "},
		{Name: NameSiteAdminEmail, Value: "ops@example.com"},
	})
	if err != nil {
		t.Fatalf("UpdateMany returned error: %v", err)
	}
	if store.items[NameSiteTagline] != "一个友好的社区" {
		t.Fatalf("tagline not trimmed/saved: %#v", store.items)
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

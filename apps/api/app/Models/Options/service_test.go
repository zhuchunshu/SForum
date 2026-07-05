package options

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestServiceReturnsDefaultSiteName(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)

	value, err := service.WebOption(context.Background(), NameSiteName)
	if err != nil {
		t.Fatalf("WebOption returned error: %v", err)
	}
	if value != "SForum" {
		t.Fatalf("expected default site name, got %q", value)
	}
}

func TestServiceListsOnlyPublicOptions(t *testing.T) {
	store := &fakeStore{items: map[string]string{
		NameAltchaSecret: "secret",
	}}
	service := NewServiceWithCacheTTL(store, time.Minute)

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	for _, item := range items {
		if item.Name == NameAltchaSecret {
			t.Fatalf("public list should not expose altcha secret: %#v", items)
		}
	}
	if len(items) != 9 {
		t.Fatalf("expected 9 public options, got %#v", items)
	}
}

func TestServiceAdminListMasksSecrets(t *testing.T) {
	store := &fakeStore{items: map[string]string{
		NameAltchaSecret: "secret",
	}}
	service := NewServiceWithCacheTTL(store, time.Minute)
	actor := settingsActor()

	items, err := service.ListAdmin(context.Background(), actor)
	if err != nil {
		t.Fatalf("ListAdmin returned error: %v", err)
	}

	var secret optionsAdminItem
	for _, item := range items {
		if item.Name == NameAltchaSecret {
			secret = optionsAdminItem{Value: item.Value, Secret: item.Secret, SecretSet: item.SecretSet}
		}
	}
	if !secret.Secret || !secret.SecretSet || secret.Value != "" {
		t.Fatalf("expected masked configured secret, got %#v", secret)
	}
}

func TestServiceUsesCacheUntilInvalidated(t *testing.T) {
	store := &fakeStore{items: map[string]string{NameSiteName: "First"}}
	service := NewServiceWithCacheTTL(store, time.Minute)
	ctx := context.Background()

	first, err := service.WebOption(ctx, NameSiteName)
	if err != nil {
		t.Fatalf("first WebOption returned error: %v", err)
	}
	store.items[NameSiteName] = "Second"
	second, err := service.WebOption(ctx, NameSiteName)
	if err != nil {
		t.Fatalf("second WebOption returned error: %v", err)
	}
	service.Invalidate()
	third, err := service.WebOption(ctx, NameSiteName)
	if err != nil {
		t.Fatalf("third WebOption returned error: %v", err)
	}

	if first != "First" || second != "First" || third != "Second" {
		t.Fatalf("expected cache then invalidation, got %q %q %q", first, second, third)
	}
	if store.listCalls != 2 {
		t.Fatalf("expected 2 list calls, got %d", store.listCalls)
	}
}

func TestServiceUpdateRequiresSettingsPermission(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)
	actor := identity.Actor{ID: 2, Status: identity.UserStatusActive, Permissions: map[string]bool{}}

	_, err := service.Update(context.Background(), actor, UpdateInput{Name: NameSiteName, Value: "Example"})
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestServiceUpdateInvalidatesCache(t *testing.T) {
	store := &fakeStore{items: map[string]string{NameSiteName: "First"}}
	service := NewServiceWithCacheTTL(store, time.Minute)
	ctx := context.Background()
	actor := settingsActor()

	if _, err := service.WebOption(ctx, NameSiteName); err != nil {
		t.Fatalf("prime cache: %v", err)
	}
	updated, err := service.Update(ctx, actor, UpdateInput{Name: NameSiteName, Value: "  Example Forum  "})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	value, err := service.WebOption(ctx, NameSiteName)
	if err != nil {
		t.Fatalf("WebOption after update returned error: %v", err)
	}

	if updated.Value != "Example Forum" || value != "Example Forum" {
		t.Fatalf("expected updated value, got option=%q value=%q", updated.Value, value)
	}
}

func TestServiceUpdateManyValidatesMergedLocaleSettings(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)
	actor := settingsActor()

	_, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameSiteSupportedLocales, Value: "en-US"},
	})
	if !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("expected default locale outside supported locales to be invalid, got %v", err)
	}

	updated, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameSiteSupportedLocales, Value: "en-US"},
		{Name: NameSiteDefaultLocale, Value: "en"},
	})
	if err != nil {
		t.Fatalf("UpdateMany returned error: %v", err)
	}
	if value := adminValue(updated, NameSiteDefaultLocale); value != "en-US" {
		t.Fatalf("expected normalized default locale, got %q", value)
	}
}

func TestServiceUpdateManyKeepsBlankSecretAndRequiresSecretForAltcha(t *testing.T) {
	store := &fakeStore{items: map[string]string{
		NameAltchaSecret: "existing-secret",
	}}
	service := NewServiceWithCacheTTL(store, time.Minute)
	actor := settingsActor()

	updated, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameAltchaSecret, Value: "   "},
		{Name: NameHumanVerificationProvider, Value: "altcha"},
	})
	if err != nil {
		t.Fatalf("UpdateMany should keep existing secret: %v", err)
	}
	if store.items[NameAltchaSecret] != "existing-secret" {
		t.Fatalf("expected blank secret update to keep existing secret, got %q", store.items[NameAltchaSecret])
	}
	if item := adminSecret(updated, NameAltchaSecret); !item.SecretSet || item.Value != "" {
		t.Fatalf("expected masked existing secret after update, got %#v", item)
	}

	emptySecretService := NewServiceWithDefaultsAndCacheTTL(&fakeStore{}, Defaults{AltchaSecret: ""}, time.Minute)
	_, err = emptySecretService.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameHumanVerificationProvider, Value: "altcha"},
	})
	if !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("expected altcha without secret to be invalid, got %v", err)
	}
}

func TestServicePersonalizationDefaultsAndValidation(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)
	actor := settingsActor()

	theme, err := service.WebOption(context.Background(), NameAppearanceTheme)
	if err != nil {
		t.Fatalf("default theme returned error: %v", err)
	}
	if theme != "pine_teal" {
		t.Fatalf("expected default pine teal theme, got %q", theme)
	}

	if _, err := service.Update(context.Background(), actor, UpdateInput{Name: NameAppearanceTheme, Value: "neon"}); !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("expected invalid custom theme, got %v", err)
	}
	customTheme, err := service.Update(context.Background(), actor, UpdateInput{Name: NameAppearanceTheme, Value: "custom:#4F46E5"})
	if err != nil {
		t.Fatalf("expected valid custom theme color, got %v", err)
	}
	if customTheme.Value != "custom:#4f46e5" {
		t.Fatalf("expected normalized custom theme color, got %q", customTheme.Value)
	}
	if _, err := service.Update(context.Background(), actor, UpdateInput{Name: NameAppearanceTheme, Value: "custom:not-a-color"}); !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("expected invalid custom theme color, got %v", err)
	}
	if _, err := service.Update(context.Background(), actor, UpdateInput{Name: NameFooterCopyrightZHCN, Value: stringsOfRunes("长", 201)}); !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("expected oversized footer copyright to be invalid, got %v", err)
	}
}

func TestServiceNormalizesFooterLinks(t *testing.T) {
	store := &fakeStore{}
	service := NewServiceWithCacheTTL(store, time.Minute)
	actor := settingsActor()
	value := `[
		{"key":"privacy","labels":{"zh-CN":"隐私","en-US":"Privacy"},"url":"/privacy"},
		{"key":"guidelines","labels":{"zh-CN":"指南","en-US":"Guidelines"},"url":""},
		{"key":"terms","labels":{"zh-CN":"条款","en-US":"Terms"},"url":"https://example.com/terms"}
	]`

	updated, err := service.Update(context.Background(), actor, UpdateInput{Name: NameFooterLinks, Value: value})
	if err != nil {
		t.Fatalf("Update footer links returned error: %v", err)
	}

	got := updated.Value
	want := `[{"key":"terms","labels":{"zh-CN":"条款","en-US":"Terms"},"url":"https://example.com/terms"},{"key":"privacy","labels":{"zh-CN":"隐私","en-US":"Privacy"},"url":"/privacy"},{"key":"guidelines","labels":{"zh-CN":"指南","en-US":"Guidelines"},"url":""}]`
	if got != want {
		t.Fatalf("expected normalized footer links\nwant: %s\n got: %s", want, got)
	}
}

func TestServiceRejectsInvalidFooterLinks(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)
	actor := settingsActor()

	cases := []string{
		`not-json`,
		`[{"key":"terms","labels":{"zh-CN":"条款","en-US":"Terms"},"url":"mailto:test@example.com"},{"key":"privacy","labels":{"zh-CN":"隐私","en-US":"Privacy"},"url":"#"},{"key":"guidelines","labels":{"zh-CN":"指南","en-US":"Guidelines"},"url":"#"}]`,
		`[{"key":"terms","labels":{"zh-CN":"","en-US":"Terms"},"url":"#"},{"key":"privacy","labels":{"zh-CN":"隐私","en-US":"Privacy"},"url":"#"},{"key":"guidelines","labels":{"zh-CN":"指南","en-US":"Guidelines"},"url":"#"}]`,
		`[{"key":"terms","labels":{"zh-CN":"条款","en-US":"Terms"},"url":"#"},{"key":"terms","labels":{"zh-CN":"隐私","en-US":"Privacy"},"url":"#"},{"key":"guidelines","labels":{"zh-CN":"指南","en-US":"Guidelines"},"url":"#"}]`,
	}

	for _, value := range cases {
		if _, err := service.Update(context.Background(), actor, UpdateInput{Name: NameFooterLinks, Value: value}); !errors.Is(err, ErrInvalidOption) {
			t.Fatalf("expected invalid footer links for %s, got %v", value, err)
		}
	}
}

func TestServiceRejectsUnknownOrEmptyOption(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)
	actor := settingsActor()

	if _, err := service.Get(context.Background(), "missing"); !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("expected invalid option for unknown get, got %v", err)
	}
	if _, err := service.Update(context.Background(), actor, UpdateInput{Name: NameSiteName, Value: ""}); !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("expected invalid option for empty update, got %v", err)
	}
}

func TestServiceEnsureDefaultsDoesNotOverwriteExistingOptions(t *testing.T) {
	store := &fakeStore{items: map[string]string{NameSiteName: "Existing"}}
	service := NewServiceWithDefaultsAndCacheTTL(store, Defaults{SiteName: "Default"}, time.Minute)

	if err := service.EnsureDefaults(context.Background()); err != nil {
		t.Fatalf("EnsureDefaults returned error: %v", err)
	}
	if got := store.items[NameSiteName]; got != "Existing" {
		t.Fatalf("expected existing site name to be kept, got %q", got)
	}
	if got := store.items[NameSiteURL]; got != "http://127.0.0.1:3000" {
		t.Fatalf("expected missing site URL to be inserted, got %q", got)
	}
}

type optionsAdminItem struct {
	Value     string
	Secret    bool
	SecretSet bool
}

func settingsActor() identity.Actor {
	return identity.Actor{
		ID:          1,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionSettingsManage: true},
	}
}

func adminValue(items []AdminOption, name string) string {
	for _, item := range items {
		if item.Name == name {
			return item.Value
		}
	}
	return ""
}

func adminSecret(items []AdminOption, name string) AdminOption {
	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	return AdminOption{}
}

func stringsOfRunes(value string, count int) string {
	var builder strings.Builder
	for i := 0; i < count; i++ {
		builder.WriteString(value)
	}
	return builder.String()
}

type fakeStore struct {
	items     map[string]string
	listCalls int
}

func (s *fakeStore) List(context.Context) ([]Option, error) {
	s.listCalls++
	options := make([]Option, 0, len(s.items))
	for name, value := range s.items {
		options = append(options, Option{Name: name, Value: value})
	}
	return options, nil
}

func (s *fakeStore) InsertMissing(_ context.Context, input UpdateInput) error {
	if s.items == nil {
		s.items = map[string]string{}
	}
	if _, ok := s.items[input.Name]; !ok {
		s.items[input.Name] = input.Value
	}
	return nil
}

func (s *fakeStore) Upsert(_ context.Context, input UpdateInput) (Option, error) {
	if s.items == nil {
		s.items = map[string]string{}
	}
	s.items[input.Name] = input.Value
	return Option{Name: input.Name, Value: input.Value}, nil
}

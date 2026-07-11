package options

import (
	"context"
	"testing"
	"time"
)

func TestCommunityPolicyDefaultsPresent(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)

	items, err := service.ListAdmin(context.Background(), settingsActor())
	if err != nil {
		t.Fatalf("ListAdmin: %v", err)
	}
	want := map[string]string{
		NameIdentityRegistrationMode:     "open",
		NameIdentityUsernameMinLength:    "3",
		NameIdentityUsernameMaxLength:    "20",
		NameIdentityUsernameCharset:      "unicode_letters_numbers",
		NameIdentityLoginMaxFailures:     "10",
		NameIdentityLoginLockoutMinutes:  "15",
		NameTrustNewUserDays:             "7",
		NameTrustNewUserForbidOutboundLinks: enabledOptionValue(true),
		NameSiteMaintenanceEnabled:       enabledOptionValue(false),
		NameForumGuestRead:               "public",
		NameForumListDefaultSort:         "latest",
		NameForumMentionsEnabled:         enabledOptionValue(true),
		NameForumMentionsMaxPerPost:      "10",
	}
	for name, expected := range want {
		if got := adminValue(items, name); got != expected {
			t.Fatalf("%s default = %q, want %q", name, got, expected)
		}
	}
}

func TestCommunityPolicyRegistrationModeClosesOpenRegistration(t *testing.T) {
	store := &fakeStore{items: map[string]string{}}
	service := NewServiceWithCacheTTL(store, time.Minute)
	actor := settingsActor()

	if _, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameIdentityRegistrationMode, Value: "invite"},
	}); err != nil {
		t.Fatalf("UpdateMany invite: %v", err)
	}
	enabled, err := service.RegistrationEnabled(context.Background())
	if err != nil {
		t.Fatalf("RegistrationEnabled: %v", err)
	}
	if enabled {
		t.Fatal("invite mode should disable open registration")
	}

	if _, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameIdentityRegistrationMode, Value: "open"},
		{Name: NameIdentityRegistrationEnabled, Value: enabledOptionValue(true)},
	}); err != nil {
		t.Fatalf("UpdateMany open: %v", err)
	}
	enabled, err = service.RegistrationEnabled(context.Background())
	if err != nil || !enabled {
		t.Fatalf("open mode should allow registration, enabled=%v err=%v", enabled, err)
	}
}

func TestCommunityPolicyAcceptsAndRejectsValues(t *testing.T) {
	store := &fakeStore{items: map[string]string{}}
	service := NewServiceWithCacheTTL(store, time.Minute)
	actor := settingsActor()

	if _, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameIdentityUsernameMinLength, Value: "4"},
		{Name: NameIdentityUsernameMaxLength, Value: "16"},
		{Name: NameIdentityUsernameCharset, Value: "ascii"},
		{Name: NameIdentityUsernameReserved, Value: " admin , Root "},
		{Name: NameForumGuestRead, Value: "login_required"},
		{Name: NameForumListDefaultSort, Value: "hot"},
		{Name: NameForumTopicsDuplicateTitlePolicy, Value: "block"},
		{Name: NameSiteMaintenanceEnabled, Value: "enabled"},
		{Name: NameSiteMaintenanceMessage, Value: "  升级中  "},
		{Name: NameTrustNewUserDays, Value: "14"},
	}); err != nil {
		t.Fatalf("valid UpdateMany: %v", err)
	}
	if got := store.items[NameIdentityUsernameReserved]; got != "admin,root" {
		t.Fatalf("reserved names = %q, want admin,root", got)
	}
	if store.items[NameForumGuestRead] != "login_required" {
		t.Fatalf("guest read not saved: %q", store.items[NameForumGuestRead])
	}
	if store.items[NameSiteMaintenanceMessage] != "升级中" {
		t.Fatalf("maintenance message not trimmed: %q", store.items[NameSiteMaintenanceMessage])
	}

	rejects := []UpdateInput{
		{Name: NameIdentityRegistrationMode, Value: "maybe"},
		{Name: NameIdentityUsernameMinLength, Value: "1"},
		{Name: NameIdentityUsernameMaxLength, Value: "2"}, // max < min after min was 4? use absolute invalid
		{Name: NameForumGuestRead, Value: "members_only"},
		{Name: NameForumListDefaultSort, Value: "random"},
		{Name: NameTrustNewUserDays, Value: "999"},
		{Name: NameForumMentionsMaxPerPost, Value: "51"},
	}
	// 先重置用户名范围为合法，再单独测 max < min
	if _, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameIdentityUsernameMinLength, Value: "3"},
		{Name: NameIdentityUsernameMaxLength, Value: "20"},
	}); err != nil {
		t.Fatalf("reset username bounds: %v", err)
	}
	for _, input := range rejects {
		if _, err := service.UpdateMany(context.Background(), actor, []UpdateInput{input}); err == nil {
			t.Fatalf("expected rejection for %s=%q", input.Name, input.Value)
		}
	}
	// min > max
	if _, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameIdentityUsernameMinLength, Value: "20"},
		{Name: NameIdentityUsernameMaxLength, Value: "5"},
	}); err == nil {
		t.Fatal("expected rejection when username max < min")
	}
}

func TestCommunityPolicyHelpers(t *testing.T) {
	store := &fakeStore{items: map[string]string{
		NameSiteMaintenanceEnabled:              "enabled",
		NameSiteMaintenanceMessage:              "维护中",
		NameIdentityRegistrationMode:            "closed",
		NameTrustNewUserDays:                    "7",
		NameTrustNewUserTopicCooldownSeconds:     "300",
		NameTrustNewUserCommentCooldownSeconds:    "60",
		NameTrustNewUserDailyTopicLimit:         "3",
		NameTrustNewUserDailyCommentLimit:       "30",
		NameTrustNewUserForbidOutboundLinks:     "enabled",
		NameTrustNewUserForbidAttachments:       "disabled",
		NameIdentityUsernameMinLength:           "3",
		NameIdentityUsernameMaxLength:           "20",
		NameIdentityUsernameCharset:             "unicode_letters_numbers",
		NameIdentityUsernameReserved:            "admin,system",
		NameIdentityLoginMaxFailures:            "10",
		NameIdentityLoginLockoutMinutes:         "15",
	}}
	service := NewServiceWithCacheTTL(store, time.Minute)

	policy, err := service.MaintenancePolicy(context.Background())
	if err != nil || !policy.Enabled || policy.Message != "维护中" {
		t.Fatalf("MaintenancePolicy = %#v err=%v", policy, err)
	}
	mode, err := service.RegistrationMode(context.Background())
	if err != nil || mode != "closed" {
		t.Fatalf("RegistrationMode = %q err=%v", mode, err)
	}
	username, err := service.UsernamePolicy(context.Background())
	if err != nil {
		t.Fatalf("UsernamePolicy: %v", err)
	}
	if username.MinLength != 3 || username.MaxLength != 20 || username.Charset != "unicode_letters_numbers" {
		t.Fatalf("UsernamePolicy unexpected: %#v", username)
	}
	lockout, err := service.LoginLockoutPolicy(context.Background())
	if err != nil || lockout.MaxFailures != 10 || lockout.LockoutMinutes != 15 {
		t.Fatalf("LoginLockoutPolicy = %#v err=%v", lockout, err)
	}
}

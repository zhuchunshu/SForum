package moderation

import (
	"testing"
	"time"
)

func TestRecommendedSettingsKeepExistingSitesPublishing(t *testing.T) {
	settings := RecommendedSettings()
	if settings.Mode != ModeOff {
		t.Fatalf("expected compatibility default %q, got %q", ModeOff, settings.Mode)
	}
	if !settings.ReviewNewUsers || settings.NewUserMaxAgeDays != 7 || !settings.ReviewExternalLinks {
		t.Fatalf("unexpected recommended rules: %#v", settings)
	}
}

func TestSettingsValidateRejectsInvalidModeAndAge(t *testing.T) {
	for _, settings := range []Settings{
		{Mode: "sometimes", NewUserMaxAgeDays: 7},
		{Mode: ModeRules, NewUserMaxAgeDays: -1},
		{Mode: ModeRules, NewUserMaxAgeDays: 3651},
	} {
		if err := settings.Validate(); err == nil {
			t.Fatalf("expected invalid settings to fail: %#v", settings)
		}
	}
}

func TestPolicyRulesExplainWhyContentIsPending(t *testing.T) {
	now := time.Date(2026, time.July, 11, 12, 0, 0, 0, time.UTC)
	policy := Settings{
		Mode:                ModeRules,
		ReviewNewUsers:      true,
		NewUserMaxAgeDays:   7,
		ReviewExternalLinks: true,
	}
	got := policy.Evaluate(PublicationInput{
		Now:           now,
		UserCreatedAt: now.Add(-24 * time.Hour),
		RawContent:    "站内 https://forum.test/docs，站外 https://outside.test/path。",
		SiteURL:       "https://forum.test",
	})

	if !got.Pending {
		t.Fatal("expected matching rules to require review")
	}
	want := []string{TriggerNewUser, TriggerExternalLink}
	if len(got.Triggers) != len(want) {
		t.Fatalf("expected triggers %#v, got %#v", want, got.Triggers)
	}
	for index := range want {
		if got.Triggers[index] != want[index] {
			t.Fatalf("expected triggers %#v, got %#v", want, got.Triggers)
		}
	}
}

func TestPolicyRulesIgnoreEstablishedUserAndSiteLinks(t *testing.T) {
	now := time.Date(2026, time.July, 11, 12, 0, 0, 0, time.UTC)
	policy := Settings{Mode: ModeRules, ReviewNewUsers: true, NewUserMaxAgeDays: 7, ReviewExternalLinks: true}
	got := policy.Evaluate(PublicationInput{
		Now:           now,
		UserCreatedAt: now.Add(-30 * 24 * time.Hour),
		RawContent:    "请阅读 https://forum.test/docs 和 http://forum.test/help",
		SiteURL:       "https://forum.test",
	})
	if got.Pending || len(got.Triggers) != 0 {
		t.Fatalf("expected content to publish directly, got %#v", got)
	}
}

func TestPolicyModesOverrideRules(t *testing.T) {
	input := PublicationInput{RawContent: "https://outside.test"}
	if got := (Settings{Mode: ModeOff}).Evaluate(input); got.Pending {
		t.Fatal("off mode must publish directly")
	}
	if got := (Settings{Mode: ModeAll}).Evaluate(input); !got.Pending || len(got.Triggers) != 1 || got.Triggers[0] != TriggerAllContent {
		t.Fatalf("all mode must require review: %#v", got)
	}
}

package ai_test

import (
	"errors"
	"strings"
	"testing"

	ai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
)

func TestRecommendedSettingsAreValidAndConservative(t *testing.T) {
	settings := ai.RecommendedSettings()
	if err := settings.Validate(); err != nil {
		t.Fatalf("recommended settings must validate: %v", err)
	}
	if settings.Enabled {
		t.Fatal("gateway must ship disabled")
	}
	if settings.AutoAction.Enabled {
		t.Fatal("automatic action must ship disabled")
	}
	if settings.Gates.ExtensionDailyQuota <= 0 || settings.Gates.MonthlyBudgetMicros <= 0 {
		t.Fatal("default gates must be active")
	}
}

func TestSettingsKeepUnfinishedDisabledProfileSavable(t *testing.T) {
	settings := ai.RecommendedSettings()
	// 自定义提供商的初始草稿：尚未填写 baseUrl 与 model。
	settings.Profiles = append(settings.Profiles, ai.Profile{
		ID:        "custom",
		Label:     "自定义",
		Protocol:  ai.ProtocolOpenAIChat,
		APIKeyRef: ai.SecretReferencePrefix + "core/ai.custom.api_key",
		CostClass: ai.CostClassStandard,
	})
	if err := settings.Validate(); err != nil {
		t.Fatalf("an unfinished draft must stay savable: %v", err)
	}
	// 启用同一份草稿时必须被拒绝：不完整但启用的配置会直接导致调用失败。
	settings.Profiles[len(settings.Profiles)-1].Enabled = true
	if err := settings.Validate(); !errors.Is(err, ai.ErrSettingsInvalid) {
		t.Fatalf("enabling an incomplete profile must be rejected, got %v", err)
	}
}

func TestSettingsRejectInvalidProfileIdentityEvenWhenDisabled(t *testing.T) {
	settings := ai.RecommendedSettings()
	settings.Profiles = append(settings.Profiles, ai.Profile{ID: "CUSTOM!"})
	if err := settings.Validate(); !errors.Is(err, ai.ErrSettingsInvalid) {
		t.Fatalf("id stays required even for drafts, got %v", err)
	}
}

func TestSettingsAllowEnabledGatewayWithoutEnabledProfile(t *testing.T) {
	settings := ai.RecommendedSettings()
	settings.Enabled = true
	// 先开总开关再配供应商是正常顺序：这里必须允许保存。
	if err := settings.Validate(); err != nil {
		t.Fatalf("enabling the gateway alone must stay savable: %v", err)
	}
	warnings := strings.Join(settings.Warnings(), ",")
	if !strings.Contains(warnings, "gateway_enabled_without_enabled_profile") {
		t.Fatalf("missing provider must be warned about, got %v", settings.Warnings())
	}
	settings.Profiles[0].Enabled = true
	if warnings := strings.Join(settings.Warnings(), ","); strings.Contains(warnings, "gateway_enabled_without_enabled_profile") {
		t.Fatalf("warning must clear once a provider is enabled: %v", settings.Warnings())
	}
}

func TestSettingsRejectContractViolations(t *testing.T) {
	cases := map[string]func(*ai.Settings){
		"duplicate profile": func(s *ai.Settings) { s.Profiles = append(s.Profiles, s.Profiles[0]) },
		"unknown binding":   func(s *ai.Settings) { s.CostClassProfiles[ai.CostClassEconomy] = "missing" },
		"unknown class key": func(s *ai.Settings) { s.CostClassProfiles["free"] = "deepseek" },
		"bad posture":       func(s *ai.Settings) { s.PurposeFailurePosture["moderation"] = "whatever" },
		"bad action":        func(s *ai.Settings) { s.AutoAction.AllowedActions = []string{"delete"} },
		"confidence > 1":    func(s *ai.Settings) { s.AutoAction.ConfidenceThreshold = 1.2 },
		"rate over cap":     func(s *ai.Settings) { s.Gates.RateLimitPerMinute = ai.MaxRateLimitPerMinute + 1 },
		"budget over cap":   func(s *ai.Settings) { s.Gates.MonthlyBudgetMicros = ai.MaxMonthlyBudgetMicros + 1 },
		"user quota over":   func(s *ai.Settings) { s.Gates.UserDailyQuota = ai.MaxUserDailyQuota + 1 },
	}
	for name, mutate := range cases {
		settings := ai.RecommendedSettings()
		mutate(&settings)
		if err := settings.Validate(); !errors.Is(err, ai.ErrSettingsInvalid) {
			t.Fatalf("%s: expected ErrSettingsInvalid, got %v", name, err)
		}
	}
}

func TestSettingsEnableAutoActionRequiresAllowedActions(t *testing.T) {
	settings := ai.RecommendedSettings()
	settings.AutoAction.Enabled = true
	settings.AutoAction.AllowedActions = nil
	if !errors.Is(settings.Validate(), ai.ErrSettingsInvalid) {
		t.Fatal("enabled auto action without allowed actions must be rejected")
	}
}

func TestFailurePostureDefaultsToOpen(t *testing.T) {
	settings := ai.RecommendedSettings()
	if got := settings.FailurePosture("moderation.review"); got != ai.FailurePostureOpen {
		t.Fatalf("posture = %q", got)
	}
	settings.PurposeFailurePosture["moderation.review"] = ai.FailurePostureClosed
	if got := settings.FailurePosture("moderation.review"); got != ai.FailurePostureClosed {
		t.Fatalf("posture = %q", got)
	}
}

func TestResolveProfileHonoursCostClassBinding(t *testing.T) {
	settings := ai.RecommendedSettings()
	if _, err := settings.ResolveProfile(ai.CostClassEconomy, ""); !errors.Is(err, ai.ErrProfileUnavailable) {
		t.Fatalf("disabled profile must be unavailable, got %v", err)
	}
	settings.Profiles[0].Enabled = true
	profile, err := settings.ResolveProfile(ai.CostClassEconomy, "")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if profile.ID != "deepseek" {
		t.Fatalf("resolved %q", profile.ID)
	}
	if _, err := settings.ResolveProfile(ai.CostClassEconomy, "nope"); !errors.Is(err, ai.ErrProfileUnavailable) {
		t.Fatalf("unknown explicit profile must be unavailable, got %v", err)
	}
}

func TestWarningsStayQuietUntilGatewayIsEnabled(t *testing.T) {
	if warnings := ai.RecommendedSettings().Warnings(); len(warnings) != 0 {
		t.Fatalf("disabled gateway should not warn: %v", warnings)
	}
	settings := ai.RecommendedSettings()
	settings.Enabled = true
	settings.Profiles[0].Enabled = true
	warnings := settings.Warnings()
	joined := strings.Join(warnings, ",")
	if !strings.Contains(joined, "budget_gate_inactive") {
		t.Fatalf("expected unpriced warning, got %v", warnings)
	}
}

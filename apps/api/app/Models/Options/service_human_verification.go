package options

import (
	"context"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
)

type humanVerificationScenario struct {
	name           string
	purpose        humanverify.Purpose
	defaultEnabled bool
}

var humanVerificationScenarios = []humanVerificationScenario{
	{name: NameHumanVerificationRegister, purpose: humanverify.PurposeRegister, defaultEnabled: true},
	{name: NameHumanVerificationPasswordReset, purpose: humanverify.PurposePasswordReset, defaultEnabled: true},
	{name: NameHumanVerificationEmailVerification, purpose: humanverify.PurposeEmailVerification, defaultEnabled: true},
	{name: NameHumanVerificationLoginRisk, purpose: humanverify.PurposeLoginRisk},
	{name: NameHumanVerificationPostRisk, purpose: humanverify.PurposePostRisk},
}

func init() {
	for _, scenario := range humanVerificationScenarios {
		optionDefinitions = append(optionDefinitions, optionDefinition{
			name:             scenario.name,
			public:           true,
			managePermission: identity.PermissionSettingsSiteManage,
		})
	}
}

func (s *Service) HumanVerificationConfig(ctx context.Context) (humanverify.RuntimeConfig, error) {
	values, err := s.loadMap(ctx)
	if err != nil {
		return humanverify.RuntimeConfig{}, err
	}

	ttl, _ := parsePositiveDuration(values[NameAltchaChallengeTTL])
	cost, _ := parsePositiveInt(values[NameAltchaCost])
	purposeEnabled := map[humanverify.Purpose]bool{}
	for _, scenario := range humanVerificationScenarios {
		purposeEnabled[scenario.purpose] = isEnabledOption(values[scenario.name])
	}
	return humanverify.RuntimeConfig{
		Provider:        values[NameHumanVerificationProvider],
		AltchaSecret:    values[NameAltchaSecret],
		AltchaTTL:       ttl,
		AltchaCost:      cost,
		PurposeEnabled:  purposeEnabled,
		RateLimit:       60,
		RateLimitWindow: time.Minute,
	}, nil
}

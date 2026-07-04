package options

import "errors"

const (
	NameSiteName                  = "site.name"
	NameSiteURL                   = "site.url"
	NameSiteDefaultLocale         = "site.default_locale"
	NameSiteSupportedLocales      = "site.supported_locales"
	NameHumanVerificationProvider = "human_verification.provider"
	NameAltchaSecret              = "human_verification.altcha.secret"
	NameAltchaChallengeTTL        = "human_verification.altcha.challenge_ttl"
	NameAltchaCost                = "human_verification.altcha.cost"

	CodeInvalid = "options.invalid"
)

var ErrInvalidOption = errors.New("options: invalid option")

type Option struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type AdminOption struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	Public    bool   `json:"public"`
	Secret    bool   `json:"secret"`
	SecretSet bool   `json:"secretSet"`
}

type UpdateInput struct {
	Name  string
	Value string
}

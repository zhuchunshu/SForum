package options

import (
	"context"
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Localization"
	storage "github.com/zhuchunshu/sforum/apps/api/app/Support/Storage"
)

const defaultCacheTTL = 30 * time.Second
const footerCopyrightMaxRunes = 200
const footerLinkLabelMaxRunes = 40
const seoTitleTemplateMaxRunes = 120
const seoDescriptionMaxRunes = 320
const seoKeywordsMaxRunes = 200
const seoVerificationMaxRunes = 120
const seoTwitterSiteMaxRunes = 80
const seoRobotsPathListMaxRunes = 1000
const altchaWidgetMinDurationMin = 0
const altchaWidgetMinDurationMax = 10000
const altchaWidgetWorkersMin = 1
const altchaWidgetWorkersMax = 16
const forumTagMaxPerTopicMin = 0
const forumTagMaxPerTopicMax = 10
const customAppearanceThemePrefix = "custom:"

var builtInLocales = []string{localization.DefaultLocale, "en-US"}
var appearanceThemes = []string{"pine_teal", "ocean_blue", "violet", "rose", "amber"}
var footerLinkKeys = []string{"terms", "privacy", "guidelines"}
var seoTwitterCards = []string{"summary", "summary_large_image"}
var altchaWidgetTypes = []string{"native", "checkbox", "switch"}
var altchaWidgetAutoModes = []string{"off", "onfocus", "onload", "onsubmit"}
var altchaWidgetDisplays = []string{"standard", "bar", "floating", "overlay", "invisible"}
var forumTagCreationModes = []string{"controlled", "review", "open"}
var forumSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type humanVerificationScenario struct {
	name           string
	purpose        humanverify.Purpose
	defaultEnabled bool
}

var humanVerificationScenarios = []humanVerificationScenario{
	{name: NameHumanVerificationRegister, purpose: humanverify.PurposeRegister, defaultEnabled: true},
	{name: NameHumanVerificationPasswordReset, purpose: humanverify.PurposePasswordReset},
	{name: NameHumanVerificationLoginRisk, purpose: humanverify.PurposeLoginRisk},
	{name: NameHumanVerificationPostRisk, purpose: humanverify.PurposePostRisk},
}

type footerLinkLabels struct {
	ZHCN string `json:"zh-CN"`
	ENUS string `json:"en-US"`
}

type footerLinkOption struct {
	Key    string           `json:"key"`
	Labels footerLinkLabels `json:"labels"`
	URL    string           `json:"url"`
}

type Defaults struct {
	SiteName                  string
	SiteURL                   string
	DefaultLocale             string
	SupportedLocales          []string
	HumanVerificationProvider string
	AltchaSecret              string
	AltchaChallengeTTL        time.Duration
	AltchaCost                int
}

type RuntimeSettings struct {
	SiteName                  string
	SiteURL                   string
	DefaultLocale             string
	SupportedLocales          []string
	HumanVerificationProvider string
}

type optionDefinition struct {
	name             string
	public           bool
	secret           bool
	managePermission string
}

var optionDefinitions = []optionDefinition{
	{name: NameSiteName, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameSiteURL, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameSiteDefaultLocale, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameSiteSupportedLocales, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameHumanVerificationProvider, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameHumanVerificationRegister, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameHumanVerificationPasswordReset, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameHumanVerificationLoginRisk, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameHumanVerificationPostRisk, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameAltchaSecret, secret: true, managePermission: identity.PermissionSettingsManage},
	{name: NameAltchaChallengeTTL, managePermission: identity.PermissionSettingsManage},
	{name: NameAltchaCost, managePermission: identity.PermissionSettingsManage},
	{name: NameAltchaWidgetType, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameAltchaWidgetAuto, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameAltchaWidgetDisplay, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameAltchaWidgetHideLogo, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameAltchaWidgetHideFooter, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameAltchaWidgetWorkers, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameAltchaWidgetMinDuration, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameAppearanceTheme, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameFooterCopyrightZHCN, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameFooterCopyrightENUS, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameFooterLinks, public: true, managePermission: identity.PermissionSettingsManage},
	{name: NameForumDefaultCategorySlug, public: true, managePermission: identity.PermissionCategoryManage},
	{name: NameForumTagCreationMode, public: true, managePermission: identity.PermissionTagManage},
	{name: NameForumTagPublicPages, public: true, managePermission: identity.PermissionTagManage},
	{name: NameForumTagMaxPerTopic, public: true, managePermission: identity.PermissionTagManage},
	{name: NameSEOMetaTitleTemplate, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOMetaDescription, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOMetaKeywords, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOOGImageURL, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOTwitterCard, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOTwitterSite, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOAllowIndexing, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOGoogleVerification, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOBingVerification, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOBaiduVerification, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOYandexVerification, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEORobotsExtraAllow, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEORobotsExtraDisallow, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEORobotsBlockAIBots, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEORobotsBlockNonSEOBots, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOSitemapEnabled, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOSitemapIncludeStaticPages, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOSitemapIncludeForumContent, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOSchemaOrgEnabled, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOSchemaOrgSearchAction, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOSchemaOrgDiscussion, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameSEOSchemaOrgOrganizationLogo, public: true, managePermission: identity.PermissionSEOManage},
	{name: NameAttachmentProvider, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentUploadEnabled, public: true, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentPathTemplate, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentPublicBaseURL, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentMaxFileSizeMB, public: true, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentAllowedExtensions, public: true, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentAllowedMIMETypes, public: true, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentDefaultVisibility, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentCleanupOrphanDays, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentLocalRoot, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentLocalPublicPrefix, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentAliyunEndpoint, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentAliyunBucket, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentAliyunRegion, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentAliyunAccessKeyID, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentAliyunAccessKeySecret, secret: true, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentTencentRegion, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentTencentBucket, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentTencentSecretID, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentTencentSecretKey, secret: true, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentTencentCDNDomain, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentFTPHost, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentFTPPort, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentFTPUsername, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentFTPPassword, secret: true, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentFTPRootPath, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentFTPPassive, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentFTPExplicitTLS, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentFTPPublicBaseURL, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentSFTPHost, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentSFTPPort, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentSFTPUsername, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentSFTPPassword, secret: true, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentSFTPPrivateKey, secret: true, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentSFTPPassphrase, secret: true, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentSFTPRootPath, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentSFTPHostKeyFingerprint, managePermission: identity.PermissionAttachmentSettings},
	{name: NameAttachmentSFTPPublicBaseURL, managePermission: identity.PermissionAttachmentSettings},
}

type Service struct {
	store    Store
	cacheTTL time.Duration
	defaults map[string]string

	mu        sync.RWMutex
	cached    map[string]string
	expiresAt time.Time
}

func NewService(store Store) *Service {
	return NewServiceWithDefaultsAndCacheTTL(store, Defaults{}, defaultCacheTTL)
}

func NewServiceWithDefaults(store Store, defaults Defaults) *Service {
	return NewServiceWithDefaultsAndCacheTTL(store, defaults, defaultCacheTTL)
}

func NewServiceWithCacheTTL(store Store, cacheTTL time.Duration) *Service {
	return NewServiceWithDefaultsAndCacheTTL(store, Defaults{}, cacheTTL)
}

func NewServiceWithDefaultsAndCacheTTL(store Store, defaults Defaults, cacheTTL time.Duration) *Service {
	if cacheTTL <= 0 {
		cacheTTL = defaultCacheTTL
	}
	return &Service{
		store:    store,
		cacheTTL: cacheTTL,
		defaults: normalizedDefaults(defaults),
	}
}

func (s *Service) EnsureDefaults(ctx context.Context) error {
	defaults := s.defaultValues()
	for _, name := range allOptionNames() {
		if err := s.store.InsertMissing(ctx, UpdateInput{Name: name, Value: defaults[name]}); err != nil {
			return err
		}
	}
	s.Invalidate()
	return nil
}

func (s *Service) List(ctx context.Context) ([]Option, error) {
	values, err := s.loadMap(ctx)
	if err != nil {
		return nil, err
	}

	options := make([]Option, 0, len(optionDefinitions))
	for _, definition := range optionDefinitions {
		if !definition.public {
			continue
		}
		options = append(options, Option{Name: definition.name, Value: values[definition.name]})
	}
	return options, nil
}

func (s *Service) ListAdmin(ctx context.Context, actor identity.Actor) ([]AdminOption, error) {
	if !actorCanManageAnyOption(actor) {
		return nil, identity.ErrPermissionDenied
	}

	values, err := s.loadMap(ctx)
	if err != nil {
		return nil, err
	}
	return s.adminOptions(values, actor), nil
}

func (s *Service) Get(ctx context.Context, name string) (Option, error) {
	name = normalizeName(name)
	if !isPublicOption(name) {
		return Option{}, ErrInvalidOption
	}

	values, err := s.loadMap(ctx)
	if err != nil {
		return Option{}, err
	}
	return Option{Name: name, Value: values[name]}, nil
}

func (s *Service) WebOption(ctx context.Context, name string) (string, error) {
	option, err := s.Get(ctx, name)
	if err != nil {
		return "", err
	}
	return option.Value, nil
}

func (s *Service) SiteName(ctx context.Context) (string, error) {
	return s.WebOption(ctx, NameSiteName)
}

func (s *Service) RuntimeSettings(ctx context.Context) (RuntimeSettings, error) {
	values, err := s.loadMap(ctx)
	if err != nil {
		return RuntimeSettings{}, err
	}
	return RuntimeSettings{
		SiteName:                  values[NameSiteName],
		SiteURL:                   values[NameSiteURL],
		DefaultLocale:             values[NameSiteDefaultLocale],
		SupportedLocales:          parseStoredLocales(values[NameSiteSupportedLocales]),
		HumanVerificationProvider: values[NameHumanVerificationProvider],
	}, nil
}

func (s *Service) InternalValues(ctx context.Context) (map[string]string, error) {
	return s.loadMap(ctx)
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

func (s *Service) Update(ctx context.Context, actor identity.Actor, input UpdateInput) (Option, error) {
	updated, err := s.UpdateMany(ctx, actor, []UpdateInput{input})
	if err != nil {
		return Option{}, err
	}

	name := normalizeName(input.Name)
	for _, item := range updated {
		if item.Name == name {
			return Option{Name: item.Name, Value: item.Value}, nil
		}
	}
	return Option{}, ErrInvalidOption
}

func (s *Service) UpdateMany(ctx context.Context, actor identity.Actor, inputs []UpdateInput) ([]AdminOption, error) {
	if !actorCanManageAnyOption(actor) {
		return nil, identity.ErrPermissionDenied
	}

	current, err := s.loadMap(ctx)
	if err != nil {
		return nil, err
	}
	merged := copyValues(current)
	pending := map[string]string{}

	for _, input := range inputs {
		name := normalizeName(input.Name)
		definition, ok := optionDefinitionFor(name)
		if !ok {
			return nil, ErrInvalidOption
		}
		if !actor.Can(definition.managePermission) {
			return nil, identity.ErrPermissionDenied
		}
		if isSecretOption(name) && strings.TrimSpace(input.Value) == "" {
			continue
		}

		value, ok := normalizeOptionValue(name, input.Value)
		if !ok {
			return nil, ErrInvalidOption
		}
		merged[name] = value
		pending[name] = value
	}

	if !isValidValueSet(merged) {
		return nil, ErrInvalidOption
	}

	for _, name := range allOptionNames() {
		value, ok := pending[name]
		if !ok {
			continue
		}
		if _, err := s.store.Upsert(ctx, UpdateInput{Name: name, Value: value}); err != nil {
			return nil, err
		}
	}

	s.Invalidate()
	values, err := s.loadMap(ctx)
	if err != nil {
		return nil, err
	}
	return s.adminOptions(values, actor), nil
}

func (s *Service) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cached = nil
	s.expiresAt = time.Time{}
}

func (s *Service) loadMap(ctx context.Context) (map[string]string, error) {
	now := time.Now()

	s.mu.RLock()
	if s.cached != nil && now.Before(s.expiresAt) {
		cached := copyValues(s.cached)
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cached != nil && now.Before(s.expiresAt) {
		return copyValues(s.cached), nil
	}

	rows, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}

	values := s.defaultValues()
	for _, row := range rows {
		name := normalizeName(row.Name)
		if !isKnownOption(name) {
			continue
		}
		value, ok := normalizeOptionValue(name, row.Value)
		if ok {
			values[name] = value
		}
	}
	values = s.coerceValueSet(values)

	s.cached = values
	s.expiresAt = now.Add(s.cacheTTL)
	return copyValues(values), nil
}

func (s *Service) adminOptions(values map[string]string, actor identity.Actor) []AdminOption {
	items := make([]AdminOption, 0, len(optionDefinitions))
	for _, definition := range optionDefinitions {
		if !actor.Can(definition.managePermission) {
			continue
		}
		value := values[definition.name]
		if definition.secret {
			items = append(items, AdminOption{
				Name:      definition.name,
				Public:    definition.public,
				Secret:    true,
				SecretSet: strings.TrimSpace(value) != "",
			})
			continue
		}
		items = append(items, AdminOption{
			Name:   definition.name,
			Value:  value,
			Public: definition.public,
		})
	}
	return items
}

func (s *Service) defaultValues() map[string]string {
	return copyValues(s.defaults)
}

func (s *Service) coerceValueSet(values map[string]string) map[string]string {
	defaults := s.defaultValues()
	coerced := copyValues(values)

	if strings.TrimSpace(coerced[NameSiteName]) == "" {
		coerced[NameSiteName] = defaults[NameSiteName]
	}
	if !isValidURL(coerced[NameSiteURL]) {
		coerced[NameSiteURL] = defaults[NameSiteURL]
	}

	supported := parseStoredLocales(coerced[NameSiteSupportedLocales])
	if len(supported) == 0 {
		supported = parseStoredLocales(defaults[NameSiteSupportedLocales])
	}
	coerced[NameSiteSupportedLocales] = strings.Join(supported, ",")

	if locale, ok := normalizeLocaleChoice(coerced[NameSiteDefaultLocale], supported); ok {
		coerced[NameSiteDefaultLocale] = locale
	} else {
		coerced[NameSiteDefaultLocale] = supported[0]
	}

	if provider, ok := normalizeHumanVerificationProvider(coerced[NameHumanVerificationProvider]); ok {
		coerced[NameHumanVerificationProvider] = provider
	} else {
		coerced[NameHumanVerificationProvider] = defaults[NameHumanVerificationProvider]
	}
	if coerced[NameHumanVerificationProvider] == humanverify.ProviderAltcha && strings.TrimSpace(coerced[NameAltchaSecret]) == "" {
		coerced[NameHumanVerificationProvider] = humanverify.ProviderDisabled
	}
	for _, scenario := range humanVerificationScenarios {
		value, ok := normalizeEnabledOption(coerced[scenario.name])
		if !ok {
			coerced[scenario.name] = defaults[scenario.name]
			continue
		}
		coerced[scenario.name] = value
	}

	if _, ok := parsePositiveDuration(coerced[NameAltchaChallengeTTL]); !ok {
		coerced[NameAltchaChallengeTTL] = defaults[NameAltchaChallengeTTL]
	}
	if _, ok := parsePositiveInt(coerced[NameAltchaCost]); !ok {
		coerced[NameAltchaCost] = defaults[NameAltchaCost]
	}
	if _, ok := normalizeAltchaWidgetType(coerced[NameAltchaWidgetType]); !ok {
		coerced[NameAltchaWidgetType] = defaults[NameAltchaWidgetType]
	}
	if _, ok := normalizeAltchaWidgetAuto(coerced[NameAltchaWidgetAuto]); !ok {
		coerced[NameAltchaWidgetAuto] = defaults[NameAltchaWidgetAuto]
	}
	if _, ok := normalizeAltchaWidgetDisplay(coerced[NameAltchaWidgetDisplay]); !ok {
		coerced[NameAltchaWidgetDisplay] = defaults[NameAltchaWidgetDisplay]
	}
	if value, ok := normalizeEnabledOption(coerced[NameAltchaWidgetHideLogo]); ok {
		coerced[NameAltchaWidgetHideLogo] = value
	} else {
		coerced[NameAltchaWidgetHideLogo] = defaults[NameAltchaWidgetHideLogo]
	}
	if value, ok := normalizeEnabledOption(coerced[NameAltchaWidgetHideFooter]); ok {
		coerced[NameAltchaWidgetHideFooter] = value
	} else {
		coerced[NameAltchaWidgetHideFooter] = defaults[NameAltchaWidgetHideFooter]
	}
	if _, ok := parseBoundedInt(coerced[NameAltchaWidgetWorkers], altchaWidgetWorkersMin, altchaWidgetWorkersMax); !ok {
		coerced[NameAltchaWidgetWorkers] = defaults[NameAltchaWidgetWorkers]
	}
	if _, ok := parseBoundedInt(coerced[NameAltchaWidgetMinDuration], altchaWidgetMinDurationMin, altchaWidgetMinDurationMax); !ok {
		coerced[NameAltchaWidgetMinDuration] = defaults[NameAltchaWidgetMinDuration]
	}
	if _, ok := normalizeAppearanceTheme(coerced[NameAppearanceTheme]); !ok {
		coerced[NameAppearanceTheme] = defaults[NameAppearanceTheme]
	}
	if _, ok := normalizeFooterCopyright(coerced[NameFooterCopyrightZHCN]); !ok {
		coerced[NameFooterCopyrightZHCN] = defaults[NameFooterCopyrightZHCN]
	}
	if _, ok := normalizeFooterCopyright(coerced[NameFooterCopyrightENUS]); !ok {
		coerced[NameFooterCopyrightENUS] = defaults[NameFooterCopyrightENUS]
	}
	if _, ok := normalizeFooterLinks(coerced[NameFooterLinks]); !ok {
		coerced[NameFooterLinks] = defaults[NameFooterLinks]
	}
	if _, ok := normalizeForumSlug(coerced[NameForumDefaultCategorySlug]); !ok {
		coerced[NameForumDefaultCategorySlug] = defaults[NameForumDefaultCategorySlug]
	}
	if _, ok := normalizeForumTagCreationMode(coerced[NameForumTagCreationMode]); !ok {
		coerced[NameForumTagCreationMode] = defaults[NameForumTagCreationMode]
	}
	if value, ok := normalizeEnabledOption(coerced[NameForumTagPublicPages]); ok {
		coerced[NameForumTagPublicPages] = value
	} else {
		coerced[NameForumTagPublicPages] = defaults[NameForumTagPublicPages]
	}
	if _, ok := parseBoundedInt(coerced[NameForumTagMaxPerTopic], forumTagMaxPerTopicMin, forumTagMaxPerTopicMax); !ok {
		coerced[NameForumTagMaxPerTopic] = defaults[NameForumTagMaxPerTopic]
	}
	for _, name := range seoEnabledOptionNames() {
		value, ok := normalizeEnabledOption(coerced[name])
		if !ok {
			coerced[name] = defaults[name]
			continue
		}
		coerced[name] = value
	}
	if _, ok := normalizeSEOTitleTemplate(coerced[NameSEOMetaTitleTemplate]); !ok {
		coerced[NameSEOMetaTitleTemplate] = defaults[NameSEOMetaTitleTemplate]
	}
	if _, ok := normalizeBoundedText(coerced[NameSEOMetaDescription], seoDescriptionMaxRunes); !ok {
		coerced[NameSEOMetaDescription] = defaults[NameSEOMetaDescription]
	}
	if _, ok := normalizeBoundedText(coerced[NameSEOMetaKeywords], seoKeywordsMaxRunes); !ok {
		coerced[NameSEOMetaKeywords] = defaults[NameSEOMetaKeywords]
	}
	if _, ok := normalizeOptionalURL(coerced[NameSEOOGImageURL]); !ok {
		coerced[NameSEOOGImageURL] = defaults[NameSEOOGImageURL]
	}
	if _, ok := normalizeSEOTwitterCard(coerced[NameSEOTwitterCard]); !ok {
		coerced[NameSEOTwitterCard] = defaults[NameSEOTwitterCard]
	}
	if _, ok := normalizeSEOTwitterSite(coerced[NameSEOTwitterSite]); !ok {
		coerced[NameSEOTwitterSite] = defaults[NameSEOTwitterSite]
	}
	for _, name := range seoVerificationOptionNames() {
		if _, ok := normalizeSEOVerificationToken(coerced[name]); !ok {
			coerced[name] = defaults[name]
		}
	}
	if _, ok := normalizeSEORobotsPathList(coerced[NameSEORobotsExtraAllow]); !ok {
		coerced[NameSEORobotsExtraAllow] = defaults[NameSEORobotsExtraAllow]
	}
	if _, ok := normalizeSEORobotsPathList(coerced[NameSEORobotsExtraDisallow]); !ok {
		coerced[NameSEORobotsExtraDisallow] = defaults[NameSEORobotsExtraDisallow]
	}
	if _, ok := normalizeOptionalURL(coerced[NameSEOSchemaOrgOrganizationLogo]); !ok {
		coerced[NameSEOSchemaOrgOrganizationLogo] = defaults[NameSEOSchemaOrgOrganizationLogo]
	}
	coerceAttachmentOptions(coerced, defaults)

	return coerced
}

func normalizedDefaults(defaults Defaults) map[string]string {
	values := map[string]string{
		NameSiteName:                         "SForum",
		NameSiteURL:                          "http://127.0.0.1:3000",
		NameSiteDefaultLocale:                localization.DefaultLocale,
		NameSiteSupportedLocales:             "zh-CN,en-US",
		NameHumanVerificationProvider:        humanverify.ProviderDisabled,
		NameHumanVerificationRegister:        enabledOptionValue(true),
		NameHumanVerificationPasswordReset:   enabledOptionValue(false),
		NameHumanVerificationLoginRisk:       enabledOptionValue(false),
		NameHumanVerificationPostRisk:        enabledOptionValue(false),
		NameAltchaSecret:                     "",
		NameAltchaChallengeTTL:               (10 * time.Minute).String(),
		NameAltchaCost:                       "1000",
		NameAltchaWidgetType:                 "checkbox",
		NameAltchaWidgetAuto:                 "off",
		NameAltchaWidgetDisplay:              "standard",
		NameAltchaWidgetHideLogo:             enabledOptionValue(true),
		NameAltchaWidgetHideFooter:           enabledOptionValue(true),
		NameAltchaWidgetWorkers:              "2",
		NameAltchaWidgetMinDuration:          "500",
		NameAppearanceTheme:                  "pine_teal",
		NameFooterCopyrightZHCN:              "© {year} {siteName}。保留所有权利。",
		NameFooterCopyrightENUS:              "© {year} {siteName}. All rights reserved.",
		NameFooterLinks:                      defaultFooterLinksValue(),
		NameForumDefaultCategorySlug:         "general",
		NameForumTagCreationMode:             "controlled",
		NameForumTagPublicPages:              enabledOptionValue(true),
		NameForumTagMaxPerTopic:              "5",
		NameSEOMetaTitleTemplate:             "",
		NameSEOMetaDescription:               "",
		NameSEOMetaKeywords:                  "",
		NameSEOOGImageURL:                    "",
		NameSEOTwitterCard:                   "summary_large_image",
		NameSEOTwitterSite:                   "",
		NameSEOAllowIndexing:                 enabledOptionValue(true),
		NameSEOGoogleVerification:            "",
		NameSEOBingVerification:              "",
		NameSEOBaiduVerification:             "",
		NameSEOYandexVerification:            "",
		NameSEORobotsExtraAllow:              "",
		NameSEORobotsExtraDisallow:           "",
		NameSEORobotsBlockAIBots:             enabledOptionValue(false),
		NameSEORobotsBlockNonSEOBots:         enabledOptionValue(false),
		NameSEOSitemapEnabled:                enabledOptionValue(true),
		NameSEOSitemapIncludeStaticPages:     enabledOptionValue(true),
		NameSEOSitemapIncludeForumContent:    enabledOptionValue(false),
		NameSEOSchemaOrgEnabled:              enabledOptionValue(true),
		NameSEOSchemaOrgSearchAction:         enabledOptionValue(true),
		NameSEOSchemaOrgDiscussion:           enabledOptionValue(true),
		NameSEOSchemaOrgOrganizationLogo:     "",
		NameAttachmentProvider:               storage.ProviderLocal,
		NameAttachmentUploadEnabled:          enabledOptionValue(true),
		NameAttachmentPathTemplate:           "{yyyy}/{mm}/{dd}/{public_id}{ext}",
		NameAttachmentPublicBaseURL:          "",
		NameAttachmentMaxFileSizeMB:          "20",
		NameAttachmentAllowedExtensions:      ".jpg,.jpeg,.png,.gif,.webp,.pdf,.txt,.zip",
		NameAttachmentAllowedMIMETypes:       "image/jpeg,image/png,image/gif,image/webp,application/pdf,text/plain,application/zip",
		NameAttachmentDefaultVisibility:      "public",
		NameAttachmentCleanupOrphanDays:      "30",
		NameAttachmentLocalRoot:              "storage/app/attachments",
		NameAttachmentLocalPublicPrefix:      "",
		NameAttachmentAliyunEndpoint:         "",
		NameAttachmentAliyunBucket:           "",
		NameAttachmentAliyunRegion:           "",
		NameAttachmentAliyunAccessKeyID:      "",
		NameAttachmentAliyunAccessKeySecret:  "",
		NameAttachmentTencentRegion:          "",
		NameAttachmentTencentBucket:          "",
		NameAttachmentTencentSecretID:        "",
		NameAttachmentTencentSecretKey:       "",
		NameAttachmentTencentCDNDomain:       "",
		NameAttachmentFTPHost:                "",
		NameAttachmentFTPPort:                "21",
		NameAttachmentFTPUsername:            "",
		NameAttachmentFTPPassword:            "",
		NameAttachmentFTPRootPath:            "/",
		NameAttachmentFTPPassive:             enabledOptionValue(true),
		NameAttachmentFTPExplicitTLS:         enabledOptionValue(false),
		NameAttachmentFTPPublicBaseURL:       "",
		NameAttachmentSFTPHost:               "",
		NameAttachmentSFTPPort:               "22",
		NameAttachmentSFTPUsername:           "",
		NameAttachmentSFTPPassword:           "",
		NameAttachmentSFTPPrivateKey:         "",
		NameAttachmentSFTPPassphrase:         "",
		NameAttachmentSFTPRootPath:           "/",
		NameAttachmentSFTPHostKeyFingerprint: "",
		NameAttachmentSFTPPublicBaseURL:      "",
	}

	if value := strings.TrimSpace(defaults.SiteName); value != "" {
		values[NameSiteName] = value
	}
	if value := strings.TrimSpace(defaults.SiteURL); isValidURL(value) {
		values[NameSiteURL] = value
	}
	if len(defaults.SupportedLocales) > 0 {
		if locales := normalizeLocaleList(defaults.SupportedLocales); len(locales) > 0 {
			values[NameSiteSupportedLocales] = strings.Join(locales, ",")
		}
	}
	supported := parseStoredLocales(values[NameSiteSupportedLocales])
	if value, ok := normalizeLocaleChoice(defaults.DefaultLocale, supported); ok {
		values[NameSiteDefaultLocale] = value
	} else if len(supported) > 0 {
		values[NameSiteDefaultLocale] = supported[0]
	}
	if value, ok := normalizeHumanVerificationProvider(defaults.HumanVerificationProvider); ok {
		values[NameHumanVerificationProvider] = value
	}
	if value := strings.TrimSpace(defaults.AltchaSecret); value != "" {
		values[NameAltchaSecret] = value
	}
	if defaults.AltchaChallengeTTL > 0 {
		values[NameAltchaChallengeTTL] = defaults.AltchaChallengeTTL.String()
	}
	if defaults.AltchaCost > 0 {
		values[NameAltchaCost] = strconv.Itoa(defaults.AltchaCost)
	}
	if values[NameHumanVerificationProvider] == humanverify.ProviderAltcha && values[NameAltchaSecret] == "" {
		values[NameHumanVerificationProvider] = humanverify.ProviderDisabled
	}

	return values
}

func copyValues(values map[string]string) map[string]string {
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func allOptionNames() []string {
	names := make([]string, 0, len(optionDefinitions))
	for _, definition := range optionDefinitions {
		names = append(names, definition.name)
	}
	return names
}

func normalizeName(name string) string {
	return strings.TrimSpace(name)
}

func normalizeOptionValue(name string, value string) (string, bool) {
	value = strings.TrimSpace(value)
	switch name {
	case NameSiteName:
		return value, value != "" && len([]rune(value)) <= 80
	case NameSiteURL:
		return value, isValidURL(value)
	case NameSiteDefaultLocale:
		return normalizeLocaleChoice(value, builtInLocales)
	case NameSiteSupportedLocales:
		locales := parseStoredLocales(value)
		if len(locales) == 0 {
			return "", false
		}
		return strings.Join(locales, ","), true
	case NameHumanVerificationProvider:
		return normalizeHumanVerificationProvider(value)
	case NameHumanVerificationRegister, NameHumanVerificationPasswordReset, NameHumanVerificationLoginRisk, NameHumanVerificationPostRisk:
		return normalizeEnabledOption(value)
	case NameAltchaSecret:
		return value, true
	case NameAltchaChallengeTTL:
		duration, ok := parsePositiveDuration(value)
		if !ok {
			return "", false
		}
		return duration.String(), true
	case NameAltchaCost:
		parsed, ok := parsePositiveInt(value)
		if !ok {
			return "", false
		}
		return strconv.Itoa(parsed), true
	case NameAltchaWidgetType:
		return normalizeAltchaWidgetType(value)
	case NameAltchaWidgetAuto:
		return normalizeAltchaWidgetAuto(value)
	case NameAltchaWidgetDisplay:
		return normalizeAltchaWidgetDisplay(value)
	case NameAltchaWidgetHideLogo, NameAltchaWidgetHideFooter:
		return normalizeEnabledOption(value)
	case NameAltchaWidgetWorkers:
		parsed, ok := parseBoundedInt(value, altchaWidgetWorkersMin, altchaWidgetWorkersMax)
		if !ok {
			return "", false
		}
		return strconv.Itoa(parsed), true
	case NameAltchaWidgetMinDuration:
		parsed, ok := parseBoundedInt(value, altchaWidgetMinDurationMin, altchaWidgetMinDurationMax)
		if !ok {
			return "", false
		}
		return strconv.Itoa(parsed), true
	case NameAppearanceTheme:
		return normalizeAppearanceTheme(value)
	case NameFooterCopyrightZHCN, NameFooterCopyrightENUS:
		return normalizeFooterCopyright(value)
	case NameFooterLinks:
		return normalizeFooterLinks(value)
	case NameForumDefaultCategorySlug:
		return normalizeForumSlug(value)
	case NameForumTagCreationMode:
		return normalizeForumTagCreationMode(value)
	case NameForumTagPublicPages:
		return normalizeEnabledOption(value)
	case NameForumTagMaxPerTopic:
		return normalizeBoundedInt(value, forumTagMaxPerTopicMin, forumTagMaxPerTopicMax)
	case NameSEOMetaTitleTemplate:
		return normalizeSEOTitleTemplate(value)
	case NameSEOMetaDescription:
		return normalizeBoundedText(value, seoDescriptionMaxRunes)
	case NameSEOMetaKeywords:
		return normalizeBoundedText(value, seoKeywordsMaxRunes)
	case NameSEOOGImageURL, NameSEOSchemaOrgOrganizationLogo:
		return normalizeOptionalURL(value)
	case NameSEOTwitterCard:
		return normalizeSEOTwitterCard(value)
	case NameSEOTwitterSite:
		return normalizeSEOTwitterSite(value)
	case NameSEOAllowIndexing, NameSEORobotsBlockAIBots, NameSEORobotsBlockNonSEOBots, NameSEOSitemapEnabled, NameSEOSitemapIncludeStaticPages, NameSEOSitemapIncludeForumContent, NameSEOSchemaOrgEnabled, NameSEOSchemaOrgSearchAction, NameSEOSchemaOrgDiscussion:
		return normalizeEnabledOption(value)
	case NameSEOGoogleVerification, NameSEOBingVerification, NameSEOBaiduVerification, NameSEOYandexVerification:
		return normalizeSEOVerificationToken(value)
	case NameSEORobotsExtraAllow, NameSEORobotsExtraDisallow:
		return normalizeSEORobotsPathList(value)
	case NameAttachmentProvider:
		return normalizeAttachmentProvider(value)
	case NameAttachmentUploadEnabled, NameAttachmentFTPPassive, NameAttachmentFTPExplicitTLS:
		return normalizeEnabledOption(value)
	case NameAttachmentPathTemplate:
		return normalizeAttachmentPathTemplate(value)
	case NameAttachmentLocalRoot:
		return normalizeAttachmentLocalRoot(value)
	case NameAttachmentPublicBaseURL, NameAttachmentLocalPublicPrefix, NameAttachmentTencentCDNDomain, NameAttachmentFTPPublicBaseURL, NameAttachmentSFTPPublicBaseURL:
		return normalizeOptionalURL(value)
	case NameAttachmentMaxFileSizeMB:
		return normalizeBoundedInt(value, 1, 1024)
	case NameAttachmentAllowedExtensions:
		return normalizeAttachmentExtensions(value)
	case NameAttachmentAllowedMIMETypes:
		return normalizeAttachmentMIMETypes(value)
	case NameAttachmentDefaultVisibility:
		return normalizeStringChoice(value, attachmentVisibilities)
	case NameAttachmentCleanupOrphanDays:
		return normalizeBoundedInt(value, 1, 3650)
	case NameAttachmentFTPPort, NameAttachmentSFTPPort:
		return normalizeBoundedInt(value, 1, 65535)
	case NameAttachmentAliyunEndpoint, NameAttachmentAliyunBucket, NameAttachmentAliyunRegion, NameAttachmentAliyunAccessKeyID,
		NameAttachmentTencentRegion, NameAttachmentTencentBucket, NameAttachmentTencentSecretID,
		NameAttachmentFTPHost, NameAttachmentFTPUsername,
		NameAttachmentSFTPHost, NameAttachmentSFTPUsername, NameAttachmentSFTPHostKeyFingerprint:
		return normalizeBoundedText(value, attachmentProviderTextMaxRunes)
	case NameAttachmentFTPRootPath, NameAttachmentSFTPRootPath:
		return normalizeAttachmentRootPath(value)
	case NameAttachmentAliyunAccessKeySecret, NameAttachmentTencentSecretKey, NameAttachmentFTPPassword, NameAttachmentSFTPPassword, NameAttachmentSFTPPrivateKey, NameAttachmentSFTPPassphrase:
		return normalizeBoundedText(value, attachmentSecretMaxRunes)
	default:
		return "", false
	}
}

func isValidValueSet(values map[string]string) bool {
	if strings.TrimSpace(values[NameSiteName]) == "" || len([]rune(values[NameSiteName])) > 80 {
		return false
	}
	if !isValidURL(values[NameSiteURL]) {
		return false
	}

	supported := parseStoredLocales(values[NameSiteSupportedLocales])
	if len(supported) == 0 {
		return false
	}
	if _, ok := normalizeLocaleChoice(values[NameSiteDefaultLocale], supported); !ok {
		return false
	}

	provider, ok := normalizeHumanVerificationProvider(values[NameHumanVerificationProvider])
	if !ok {
		return false
	}
	if provider == humanverify.ProviderAltcha && strings.TrimSpace(values[NameAltchaSecret]) == "" {
		return false
	}
	for _, scenario := range humanVerificationScenarios {
		if _, ok := normalizeEnabledOption(values[scenario.name]); !ok {
			return false
		}
	}
	if _, ok := parsePositiveDuration(values[NameAltchaChallengeTTL]); !ok {
		return false
	}
	if _, ok := parsePositiveInt(values[NameAltchaCost]); !ok {
		return false
	}
	if _, ok := normalizeAltchaWidgetType(values[NameAltchaWidgetType]); !ok {
		return false
	}
	if _, ok := normalizeAltchaWidgetAuto(values[NameAltchaWidgetAuto]); !ok {
		return false
	}
	if _, ok := normalizeAltchaWidgetDisplay(values[NameAltchaWidgetDisplay]); !ok {
		return false
	}
	if _, ok := normalizeEnabledOption(values[NameAltchaWidgetHideLogo]); !ok {
		return false
	}
	if _, ok := normalizeEnabledOption(values[NameAltchaWidgetHideFooter]); !ok {
		return false
	}
	if _, ok := parseBoundedInt(values[NameAltchaWidgetWorkers], altchaWidgetWorkersMin, altchaWidgetWorkersMax); !ok {
		return false
	}
	if _, ok := parseBoundedInt(values[NameAltchaWidgetMinDuration], altchaWidgetMinDurationMin, altchaWidgetMinDurationMax); !ok {
		return false
	}
	if _, ok := normalizeAppearanceTheme(values[NameAppearanceTheme]); !ok {
		return false
	}
	if _, ok := normalizeFooterCopyright(values[NameFooterCopyrightZHCN]); !ok {
		return false
	}
	if _, ok := normalizeFooterCopyright(values[NameFooterCopyrightENUS]); !ok {
		return false
	}
	if _, ok := normalizeFooterLinks(values[NameFooterLinks]); !ok {
		return false
	}
	if _, ok := normalizeForumSlug(values[NameForumDefaultCategorySlug]); !ok {
		return false
	}
	if _, ok := normalizeForumTagCreationMode(values[NameForumTagCreationMode]); !ok {
		return false
	}
	if _, ok := normalizeEnabledOption(values[NameForumTagPublicPages]); !ok {
		return false
	}
	if _, ok := parseBoundedInt(values[NameForumTagMaxPerTopic], forumTagMaxPerTopicMin, forumTagMaxPerTopicMax); !ok {
		return false
	}
	if _, ok := normalizeSEOTitleTemplate(values[NameSEOMetaTitleTemplate]); !ok {
		return false
	}
	if _, ok := normalizeBoundedText(values[NameSEOMetaDescription], seoDescriptionMaxRunes); !ok {
		return false
	}
	if _, ok := normalizeBoundedText(values[NameSEOMetaKeywords], seoKeywordsMaxRunes); !ok {
		return false
	}
	if _, ok := normalizeOptionalURL(values[NameSEOOGImageURL]); !ok {
		return false
	}
	if _, ok := normalizeSEOTwitterCard(values[NameSEOTwitterCard]); !ok {
		return false
	}
	if _, ok := normalizeSEOTwitterSite(values[NameSEOTwitterSite]); !ok {
		return false
	}
	for _, name := range seoEnabledOptionNames() {
		if _, ok := normalizeEnabledOption(values[name]); !ok {
			return false
		}
	}
	for _, name := range seoVerificationOptionNames() {
		if _, ok := normalizeSEOVerificationToken(values[name]); !ok {
			return false
		}
	}
	if _, ok := normalizeSEORobotsPathList(values[NameSEORobotsExtraAllow]); !ok {
		return false
	}
	if _, ok := normalizeSEORobotsPathList(values[NameSEORobotsExtraDisallow]); !ok {
		return false
	}
	if _, ok := normalizeOptionalURL(values[NameSEOSchemaOrgOrganizationLogo]); !ok {
		return false
	}
	if !isValidAttachmentOptions(values) {
		return false
	}
	return true
}

func isKnownOption(name string) bool {
	_, ok := optionDefinitionFor(name)
	return ok
}

func isPublicOption(name string) bool {
	definition, ok := optionDefinitionFor(name)
	return ok && definition.public
}

func isSecretOption(name string) bool {
	definition, ok := optionDefinitionFor(name)
	return ok && definition.secret
}

func optionDefinitionFor(name string) (optionDefinition, bool) {
	for _, definition := range optionDefinitions {
		if definition.name == name {
			return definition, true
		}
	}
	return optionDefinition{}, false
}

func actorCanManageAnyOption(actor identity.Actor) bool {
	for _, definition := range optionDefinitions {
		if actor.Can(definition.managePermission) {
			return true
		}
	}
	return false
}

func isValidURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed == nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func normalizeLocaleList(values []string) []string {
	locales := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		locale, ok := normalizeLocaleChoice(value, builtInLocales)
		if !ok || seen[locale] {
			continue
		}
		seen[locale] = true
		locales = append(locales, locale)
	}
	return locales
}

func parseStoredLocales(value string) []string {
	parts := strings.Split(value, ",")
	locales := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		locale, ok := normalizeLocaleChoice(part, builtInLocales)
		if !ok || seen[locale] {
			continue
		}
		seen[locale] = true
		locales = append(locales, locale)
	}
	return locales
}

func normalizeLocaleChoice(value string, allowed []string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	candidate := localization.Normalize(value, nil)
	for _, locale := range allowed {
		if strings.EqualFold(candidate, locale) {
			return locale, true
		}
	}
	return "", false
}

func normalizeHumanVerificationProvider(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", humanverify.ProviderDisabled:
		return humanverify.ProviderDisabled, true
	case humanverify.ProviderAltcha:
		return humanverify.ProviderAltcha, true
	default:
		return "", false
	}
}

func normalizeEnabledOption(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "enabled", "true", "1", "yes", "on":
		return enabledOptionValue(true), true
	case "disabled", "false", "0", "no", "off":
		return enabledOptionValue(false), true
	default:
		return "", false
	}
}

func enabledOptionValue(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func isEnabledOption(value string) bool {
	normalized, ok := normalizeEnabledOption(value)
	return ok && normalized == enabledOptionValue(true)
}

func parsePositiveDuration(value string) (time.Duration, bool) {
	duration, err := time.ParseDuration(strings.TrimSpace(value))
	return duration, err == nil && duration > 0
}

func parsePositiveInt(value string) (int, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	return parsed, err == nil && parsed > 0
}

func parseBoundedInt(value string, min int, max int) (int, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	return parsed, err == nil && parsed >= min && parsed <= max
}

func normalizeAltchaWidgetType(value string) (string, bool) {
	return normalizeChoice(value, altchaWidgetTypes)
}

func normalizeAltchaWidgetAuto(value string) (string, bool) {
	return normalizeChoice(value, altchaWidgetAutoModes)
}

func normalizeAltchaWidgetDisplay(value string) (string, bool) {
	return normalizeChoice(value, altchaWidgetDisplays)
}

func normalizeChoice(value string, allowed []string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, item := range allowed {
		if value == item {
			return item, true
		}
	}
	return "", false
}

func normalizeStringChoice(value string, allowed []string) (string, bool) {
	value = strings.TrimSpace(value)
	for _, item := range allowed {
		if strings.EqualFold(value, item) {
			return item, true
		}
	}
	return "", false
}

func normalizeForumSlug(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	return value, forumSlugPattern.MatchString(value)
}

func normalizeForumTagCreationMode(value string) (string, bool) {
	return normalizeChoice(value, forumTagCreationModes)
}

func normalizeBoundedInt(value string, min int, max int) (string, bool) {
	parsed, ok := parseBoundedInt(value, min, max)
	if !ok {
		return "", false
	}
	return strconv.Itoa(parsed), true
}

func normalizeAppearanceTheme(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, theme := range appearanceThemes {
		if value == theme {
			return theme, true
		}
	}

	if strings.HasPrefix(value, customAppearanceThemePrefix) {
		color, ok := normalizeAppearanceThemeColor(strings.TrimPrefix(value, customAppearanceThemePrefix))
		if ok {
			return customAppearanceThemePrefix + color, true
		}
	}
	return "", false
}

func normalizeAppearanceThemeColor(value string) (string, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, "#")
	if len(value) != 6 {
		return "", false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return "", false
		}
	}
	return "#" + value, true
}

func normalizeFooterCopyright(value string) (string, bool) {
	value = strings.TrimSpace(value)
	return value, len([]rune(value)) <= footerCopyrightMaxRunes
}

func normalizeSEOTitleTemplate(value string) (string, bool) {
	return normalizeBoundedText(value, seoTitleTemplateMaxRunes)
}

func normalizeBoundedText(value string, maxRunes int) (string, bool) {
	value = strings.TrimSpace(value)
	return value, len([]rune(value)) <= maxRunes
}

func normalizeOptionalURL(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	if !isValidURL(value) {
		return "", false
	}
	return value, true
}

func normalizeSEOTwitterCard(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, card := range seoTwitterCards {
		if value == card {
			return card, true
		}
	}
	return "", false
}

func normalizeSEOTwitterSite(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	if len([]rune(value)) > seoTwitterSiteMaxRunes {
		return "", false
	}
	if strings.HasPrefix(value, "@") {
		value = strings.TrimPrefix(value, "@")
	}
	if value == "" {
		return "", true
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '_' {
			return "", false
		}
	}
	return "@" + value, true
}

func normalizeSEOVerificationToken(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > seoVerificationMaxRunes {
		return "", false
	}
	if value == "" {
		return "", true
	}
	for _, char := range value {
		if char <= ' ' || char == '<' || char == '>' || char == '"' || char == '\'' {
			return "", false
		}
	}
	return value, true
}

func normalizeSEORobotsPathList(value string) (string, bool) {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(strings.TrimSpace(value), "\n")
	if strings.TrimSpace(value) == "" {
		return "", true
	}

	normalized := make([]string, 0, len(lines))
	seen := map[string]bool{}
	for _, line := range lines {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		if !isValidRobotsPath(path) || seen[path] {
			return "", false
		}
		seen[path] = true
		normalized = append(normalized, path)
	}

	result := strings.Join(normalized, "\n")
	return result, len([]rune(result)) <= seoRobotsPathListMaxRunes
}

func isValidRobotsPath(value string) bool {
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return false
	}
	for _, char := range value {
		if char <= ' ' || char == '<' || char == '>' || char == '"' || char == '\'' {
			return false
		}
	}
	return true
}

func seoEnabledOptionNames() []string {
	return []string{
		NameSEOAllowIndexing,
		NameSEORobotsBlockAIBots,
		NameSEORobotsBlockNonSEOBots,
		NameSEOSitemapEnabled,
		NameSEOSitemapIncludeStaticPages,
		NameSEOSitemapIncludeForumContent,
		NameSEOSchemaOrgEnabled,
		NameSEOSchemaOrgSearchAction,
		NameSEOSchemaOrgDiscussion,
	}
}

func seoVerificationOptionNames() []string {
	return []string{
		NameSEOGoogleVerification,
		NameSEOBingVerification,
		NameSEOBaiduVerification,
		NameSEOYandexVerification,
	}
}

func defaultFooterLinksValue() string {
	value, _ := marshalFooterLinks(defaultFooterLinks())
	return value
}

func defaultFooterLinks() []footerLinkOption {
	return []footerLinkOption{
		{
			Key:    "terms",
			Labels: footerLinkLabels{ZHCN: "服务条款", ENUS: "Terms of Service"},
			URL:    "#",
		},
		{
			Key:    "privacy",
			Labels: footerLinkLabels{ZHCN: "隐私政策", ENUS: "Privacy Policy"},
			URL:    "#",
		},
		{
			Key:    "guidelines",
			Labels: footerLinkLabels{ZHCN: "社区指南", ENUS: "Guidelines"},
			URL:    "#",
		},
	}
}

func normalizeFooterLinks(value string) (string, bool) {
	var links []footerLinkOption
	if err := json.Unmarshal([]byte(strings.TrimSpace(value)), &links); err != nil {
		return "", false
	}
	if len(links) != len(footerLinkKeys) {
		return "", false
	}

	byKey := map[string]footerLinkOption{}
	for _, link := range links {
		key := strings.TrimSpace(link.Key)
		if !isFooterLinkKey(key) || byKey[key].Key != "" {
			return "", false
		}

		normalized := footerLinkOption{
			Key: key,
			Labels: footerLinkLabels{
				ZHCN: strings.TrimSpace(link.Labels.ZHCN),
				ENUS: strings.TrimSpace(link.Labels.ENUS),
			},
			URL: strings.TrimSpace(link.URL),
		}
		if !isValidFooterLinkLabel(normalized.Labels.ZHCN) || !isValidFooterLinkLabel(normalized.Labels.ENUS) {
			return "", false
		}
		if !isValidFooterURL(normalized.URL) {
			return "", false
		}
		byKey[key] = normalized
	}

	ordered := make([]footerLinkOption, 0, len(footerLinkKeys))
	for _, key := range footerLinkKeys {
		link, ok := byKey[key]
		if !ok {
			return "", false
		}
		ordered = append(ordered, link)
	}
	return marshalFooterLinks(ordered)
}

func marshalFooterLinks(links []footerLinkOption) (string, bool) {
	value, err := json.Marshal(links)
	if err != nil {
		return "", false
	}
	return string(value), true
}

func isFooterLinkKey(value string) bool {
	for _, key := range footerLinkKeys {
		if value == key {
			return true
		}
	}
	return false
}

func isValidFooterLinkLabel(value string) bool {
	return value != "" && len([]rune(value)) <= footerLinkLabelMaxRunes
}

func isValidFooterURL(value string) bool {
	if value == "" || value == "#" {
		return true
	}
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return true
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed == nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

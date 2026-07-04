package options

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Localization"
)

const defaultCacheTTL = 30 * time.Second

var builtInLocales = []string{localization.DefaultLocale, "en-US"}

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
	name   string
	public bool
	secret bool
}

var optionDefinitions = []optionDefinition{
	{name: NameSiteName, public: true},
	{name: NameSiteURL, public: true},
	{name: NameSiteDefaultLocale, public: true},
	{name: NameSiteSupportedLocales, public: true},
	{name: NameHumanVerificationProvider, public: true},
	{name: NameAltchaSecret, secret: true},
	{name: NameAltchaChallengeTTL},
	{name: NameAltchaCost},
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
	if !actor.Can(identity.PermissionSettingsManage) {
		return nil, identity.ErrPermissionDenied
	}

	values, err := s.loadMap(ctx)
	if err != nil {
		return nil, err
	}
	return s.adminOptions(values), nil
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

func (s *Service) HumanVerificationConfig(ctx context.Context) (humanverify.RuntimeConfig, error) {
	values, err := s.loadMap(ctx)
	if err != nil {
		return humanverify.RuntimeConfig{}, err
	}

	ttl, _ := parsePositiveDuration(values[NameAltchaChallengeTTL])
	cost, _ := parsePositiveInt(values[NameAltchaCost])
	return humanverify.RuntimeConfig{
		Provider:        values[NameHumanVerificationProvider],
		AltchaSecret:    values[NameAltchaSecret],
		AltchaTTL:       ttl,
		AltchaCost:      cost,
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
	if !actor.Can(identity.PermissionSettingsManage) {
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
		if !isKnownOption(name) {
			return nil, ErrInvalidOption
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
	return s.adminOptions(values), nil
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

func (s *Service) adminOptions(values map[string]string) []AdminOption {
	items := make([]AdminOption, 0, len(optionDefinitions))
	for _, definition := range optionDefinitions {
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

	if _, ok := parsePositiveDuration(coerced[NameAltchaChallengeTTL]); !ok {
		coerced[NameAltchaChallengeTTL] = defaults[NameAltchaChallengeTTL]
	}
	if _, ok := parsePositiveInt(coerced[NameAltchaCost]); !ok {
		coerced[NameAltchaCost] = defaults[NameAltchaCost]
	}

	return coerced
}

func normalizedDefaults(defaults Defaults) map[string]string {
	values := map[string]string{
		NameSiteName:                  "SForum",
		NameSiteURL:                   "http://127.0.0.1:3000",
		NameSiteDefaultLocale:         localization.DefaultLocale,
		NameSiteSupportedLocales:      "zh-CN,en-US",
		NameHumanVerificationProvider: humanverify.ProviderDisabled,
		NameAltchaSecret:              "",
		NameAltchaChallengeTTL:        (10 * time.Minute).String(),
		NameAltchaCost:                "1000",
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
	if _, ok := parsePositiveDuration(values[NameAltchaChallengeTTL]); !ok {
		return false
	}
	if _, ok := parsePositiveInt(values[NameAltchaCost]); !ok {
		return false
	}
	return true
}

func isKnownOption(name string) bool {
	for _, definition := range optionDefinitions {
		if definition.name == name {
			return true
		}
	}
	return false
}

func isPublicOption(name string) bool {
	for _, definition := range optionDefinitions {
		if definition.name == name {
			return definition.public
		}
	}
	return false
}

func isSecretOption(name string) bool {
	for _, definition := range optionDefinitions {
		if definition.name == name {
			return definition.secret
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

func parsePositiveDuration(value string) (time.Duration, bool) {
	duration, err := time.ParseDuration(strings.TrimSpace(value))
	return duration, err == nil && duration > 0
}

func parsePositiveInt(value string) (int, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	return parsed, err == nil && parsed > 0
}

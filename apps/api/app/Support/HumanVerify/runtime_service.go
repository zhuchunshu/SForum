package humanverify

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type ConfigSource interface {
	HumanVerificationConfig(ctx context.Context) (RuntimeConfig, error)
}

type RuntimeService struct {
	source   ConfigSource
	store    Store
	fallback RuntimeConfig
}

func NewRuntimeService(source ConfigSource, store Store, fallback RuntimeConfig) *RuntimeService {
	return &RuntimeService{source: source, store: store, fallback: normalizeRuntimeConfig(fallback)}
}

func NewConfiguredService(cfg RuntimeConfig, store Store) (*Service, error) {
	cfg = normalizeRuntimeConfig(cfg)
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "", ProviderDisabled:
		return NewDisabledService(), nil
	case ProviderAltcha:
		if strings.TrimSpace(cfg.AltchaSecret) == "" {
			return nil, fmt.Errorf("altcha secret is required")
		}
		return NewService(
			ServiceConfig{
				Enabled:         true,
				ChallengeTTL:    cfg.AltchaTTL,
				RateLimit:       cfg.RateLimit,
				RateLimitWindow: cfg.RateLimitWindow,
			},
			NewAltchaProvider(AltchaConfig{
				Secret:       cfg.AltchaSecret,
				Cost:         cfg.AltchaCost,
				ChallengeTTL: cfg.AltchaTTL,
			}),
			store,
		), nil
	default:
		return nil, fmt.Errorf("unsupported human verification provider %q", cfg.Provider)
	}
}

func (s *RuntimeService) Challenge(ctx context.Context, purpose Purpose, subject Subject) (Challenge, error) {
	service, err := s.service(ctx)
	if err != nil {
		return Challenge{}, err
	}
	return service.Challenge(ctx, purpose, subject)
}

func (s *RuntimeService) Verify(ctx context.Context, req VerifyRequest) error {
	service, err := s.service(ctx)
	if err != nil {
		return err
	}
	return service.Verify(ctx, req)
}

func (s *RuntimeService) service(ctx context.Context) (*Service, error) {
	cfg := s.fallback
	if s != nil && s.source != nil {
		if current, err := s.source.HumanVerificationConfig(ctx); err == nil {
			cfg = current
		}
	}
	return NewConfiguredService(cfg, s.store)
}

func normalizeRuntimeConfig(cfg RuntimeConfig) RuntimeConfig {
	cfg.Provider = strings.ToLower(strings.TrimSpace(cfg.Provider))
	if cfg.Provider == "" {
		cfg.Provider = ProviderDisabled
	}
	cfg.AltchaSecret = strings.TrimSpace(cfg.AltchaSecret)
	if cfg.AltchaTTL <= 0 {
		cfg.AltchaTTL = 10 * time.Minute
	}
	if cfg.AltchaCost <= 0 {
		cfg.AltchaCost = 1000
	}
	if cfg.RateLimitWindow <= 0 {
		cfg.RateLimitWindow = time.Minute
	}
	return cfg
}

package humanverify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

type Service struct {
	enabled  bool
	cfg      ServiceConfig
	provider Provider
	store    Store
}

func NewService(cfg ServiceConfig, provider Provider, store Store) *Service {
	if cfg.ChallengeTTL <= 0 {
		cfg.ChallengeTTL = 10 * time.Minute
	}
	if cfg.RateLimitWindow <= 0 {
		cfg.RateLimitWindow = time.Minute
	}
	if store == nil {
		store = NewMemoryStore()
	}
	return &Service{
		enabled:  cfg.Enabled,
		cfg:      cfg,
		provider: provider,
		store:    store,
	}
}

func NewDisabledService() *Service {
	return NewService(ServiceConfig{Enabled: false}, nil, NewMemoryStore())
}

func (s *Service) Enabled() bool {
	return s != nil && s.enabled
}

func (s *Service) Challenge(ctx context.Context, purpose Purpose, subject Subject) (Challenge, error) {
	if !s.Enabled() {
		return Challenge{Provider: ProviderDisabled, Purpose: purpose, Payload: map[string]any{}}, nil
	}
	if err := s.checkRate(ctx, "challenge", purpose, subject.IP); err != nil {
		return Challenge{}, err
	}
	return s.provider.Challenge(ctx, purpose, subject)
}

func (s *Service) Verify(ctx context.Context, req VerifyRequest) error {
	if !s.Enabled() {
		return nil
	}
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		return ErrRequired
	}
	req.Provider = strings.TrimSpace(req.Provider)
	if req.Provider != ProviderAltcha {
		return ErrInvalid
	}
	if err := s.checkRate(ctx, "verify", req.Purpose, req.IP); err != nil {
		return err
	}

	result, err := s.provider.Verify(ctx, req)
	if err != nil {
		return err
	}
	if !result.Verified {
		return errForCode(result.Code)
	}
	if err := s.store.MarkUsed(ctx, replayKey(req), s.cfg.ChallengeTTL); err != nil {
		return err
	}
	return nil
}

func (s *Service) checkRate(ctx context.Context, bucket string, purpose Purpose, subject string) error {
	if s.cfg.RateLimit <= 0 || s.store == nil {
		return nil
	}
	if strings.TrimSpace(subject) == "" {
		subject = "unknown"
	}
	limited, err := s.store.IncrementRate(ctx, bucket+":"+string(purpose)+":"+subject, s.cfg.RateLimitWindow, s.cfg.RateLimit)
	if err != nil {
		return err
	}
	if limited {
		return ErrRateLimited
	}
	return nil
}

func errForCode(code string) error {
	switch code {
	case CodeExpired:
		return ErrExpired
	case CodeReplayed:
		return ErrReplayed
	case CodeRateLimited:
		return ErrRateLimited
	case CodeRequired:
		return ErrRequired
	default:
		return ErrInvalid
	}
}

func ErrorCode(err error) string {
	switch {
	case err == nil:
		return CodeOK
	case errors.Is(err, ErrRequired):
		return CodeRequired
	case errors.Is(err, ErrExpired):
		return CodeExpired
	case errors.Is(err, ErrReplayed):
		return CodeReplayed
	case errors.Is(err, ErrRateLimited):
		return CodeRateLimited
	default:
		return CodeInvalid
	}
}

func replayKey(req VerifyRequest) string {
	sum := sha256.Sum256([]byte(string(req.Purpose) + "\x00" + req.Provider + "\x00" + req.Token))
	return hex.EncodeToString(sum[:])
}

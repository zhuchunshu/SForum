package humanverify

import (
	"context"
	"errors"
	"time"
)

type Purpose string

const (
	PurposeRegister      Purpose = "register"
	PurposePasswordReset Purpose = "password_reset"
	PurposeLoginRisk     Purpose = "login_risk"
	PurposePostRisk      Purpose = "post_risk"
)

const (
	ProviderAltcha   = "altcha"
	ProviderDisabled = "disabled"

	CodeOK          = "ok"
	CodeRequired    = "human_verification.required"
	CodeInvalid     = "human_verification.invalid"
	CodeExpired     = "human_verification.expired"
	CodeReplayed    = "human_verification.replayed"
	CodeRateLimited = "rate_limit.exceeded"
)

var (
	ErrRequired    = errors.New("human verification: required")
	ErrInvalid     = errors.New("human verification: invalid")
	ErrExpired     = errors.New("human verification: expired")
	ErrReplayed    = errors.New("human verification: replayed")
	ErrRateLimited = errors.New("human verification: rate limited")
)

type Challenge struct {
	Provider string
	Purpose  Purpose
	Payload  any
}

type VerifyRequest struct {
	Provider string
	Purpose  Purpose
	Token    string
	IP       string
	UserID   *int64
}

type VerifyResult struct {
	Verified bool
	Code     string
}

type Subject struct {
	IP        string
	SessionID string
	UserID    *int64
}

type Provider interface {
	Challenge(ctx context.Context, purpose Purpose, subject Subject) (Challenge, error)
	Verify(ctx context.Context, req VerifyRequest) (VerifyResult, error)
}

type Verifier interface {
	Challenge(ctx context.Context, purpose Purpose, subject Subject) (Challenge, error)
	Verify(ctx context.Context, req VerifyRequest) error
}

type Store interface {
	MarkUsed(ctx context.Context, key string, ttl time.Duration) error
	IncrementRate(ctx context.Context, key string, window time.Duration, limit int) (bool, error)
}

type ServiceConfig struct {
	Enabled         bool
	ChallengeTTL    time.Duration
	RateLimit       int
	RateLimitWindow time.Duration
}

type RuntimeConfig struct {
	Provider        string
	AltchaSecret    string
	AltchaTTL       time.Duration
	AltchaCost      int
	RateLimit       int
	RateLimitWindow time.Duration
}

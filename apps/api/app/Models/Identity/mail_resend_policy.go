package identity

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	RecommendedMailResendCooldown     = 30 * time.Second
	RecommendedMailResendWindow       = time.Hour
	RecommendedMailResendMaxPerTarget = 3
	RecommendedMailResendMaxPerIP     = 10
)

type MailResendPolicy struct {
	Cooldown     time.Duration
	Window       time.Duration
	MaxPerTarget int
	MaxPerIP     int
}

type MailResendPolicyResolver interface {
	MailResendPolicy(context.Context) (MailResendPolicy, error)
}

type MailResendRateLimiter interface {
	Allow(ctx context.Context, key string, max int, window time.Duration) (bool, error)
}

type mailResendRetryReader interface {
	RetryAfter(ctx context.Context, key string) (time.Duration, error)
}

type MailResendResult struct {
	Sent    bool
	RetryAt time.Time
}

type MailResendRateLimitError struct {
	Cause   error
	RetryAt time.Time
}

func (e *MailResendRateLimitError) Error() string {
	if e == nil || e.Cause == nil {
		return "mail resend rate limited"
	}
	return e.Cause.Error()
}

func (e *MailResendRateLimitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func RecommendedMailResendPolicy() MailResendPolicy {
	return MailResendPolicy{
		Cooldown:     RecommendedMailResendCooldown,
		Window:       RecommendedMailResendWindow,
		MaxPerTarget: RecommendedMailResendMaxPerTarget,
		MaxPerIP:     RecommendedMailResendMaxPerIP,
	}
}

func enforceMailResendPolicy(
	ctx context.Context,
	limiter MailResendRateLimiter,
	namespace, target, ip string,
	policy MailResendPolicy,
	cause error,
) (time.Time, error) {
	now := time.Now().UTC()
	if limiter == nil {
		return retryAtForCooldown(now, policy.Cooldown), nil
	}

	policyKey := fmt.Sprintf("%s:%d:%d:%d:%d", namespace, policy.Cooldown/time.Second, policy.Window/time.Second, policy.MaxPerTarget, policy.MaxPerIP)
	if policy.Cooldown > 0 {
		key := policyKey + ":cooldown:" + target
		if err := allowMailResend(ctx, limiter, key, 1, policy.Cooldown, cause); err != nil {
			return time.Time{}, err
		}
	}

	targetKey := policyKey + ":target:" + target
	if err := allowMailResend(ctx, limiter, targetKey, policy.MaxPerTarget, policy.Window, cause); err != nil {
		return time.Time{}, err
	}
	if normalizedIP := strings.TrimSpace(ip); normalizedIP != "" {
		ipKey := policyKey + ":ip:" + hashIP(normalizedIP)
		if err := allowMailResend(ctx, limiter, ipKey, policy.MaxPerIP, policy.Window, cause); err != nil {
			return time.Time{}, err
		}
	}
	return retryAtForCooldown(now, policy.Cooldown), nil
}

func allowMailResend(ctx context.Context, limiter MailResendRateLimiter, key string, max int, window time.Duration, cause error) error {
	allowed, err := limiter.Allow(ctx, key, max, window)
	if err != nil {
		return err
	}
	if allowed {
		return nil
	}
	retryAfter := window
	if reader, ok := limiter.(mailResendRetryReader); ok {
		if remaining, retryErr := reader.RetryAfter(ctx, key); retryErr != nil {
			return retryErr
		} else if remaining > 0 {
			retryAfter = remaining
		}
	}
	if retryAfter <= 0 {
		retryAfter = time.Second
	}
	return &MailResendRateLimitError{Cause: cause, RetryAt: time.Now().UTC().Add(retryAfter)}
}

func retryAtForCooldown(now time.Time, cooldown time.Duration) time.Time {
	if cooldown <= 0 {
		return time.Time{}
	}
	return now.Add(cooldown)
}

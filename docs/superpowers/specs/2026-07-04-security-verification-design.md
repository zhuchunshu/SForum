# Security Verification Design

## Goal

Define SForum's first human-verification and anti-automation architecture for
open registration, account recovery, and later high-risk forum actions.

## Context

SForum keeps registration open after the first administrator exists. That is
good for a forum, but it creates predictable abuse surfaces: automated account
creation, credential stuffing, password-reset abuse, low-effort spam posts, and
link spam from new users.

The existing architecture already includes:

- Nuxt 4 for browser UI and server-side rendering.
- Go Fiber v3 for authoritative API checks.
- Redis for sessions, cache, rate limits, and temporary verification state.
- PostgreSQL for canonical users, roles, forum data, and audit records.
- Same-origin `/api/v1/*` routing through Nuxt to the Fiber API.

Human verification should reduce automated abuse without turning the forum into
a frustrating maze for normal users.

## Library Survey

Options considered:

- ALTCHA open-source proof-of-work CAPTCHA.
- Cloudflare Turnstile.
- hCaptcha.
- reCAPTCHA.
- Custom honeypots and rate limits only.

Recommendation: use ALTCHA as the default provider. Keep the backend interface
small enough to support Cloudflare Turnstile later if a deployment already uses
Cloudflare and wants a managed bot-detection service.

ALTCHA fits SForum's first architecture because it is self-hostable,
privacy-oriented, supports server-side verification libraries including Go, and
does not require the browser or API to call an external CAPTCHA service during
normal validation. This works well for deployments where third-party network
access may be unreliable.

Turnstile remains the best later alternative when SForum is deployed behind
Cloudflare and the site operator accepts a third-party managed service. hCaptcha
and reCAPTCHA are not recommended as defaults because they add more external
service dependency and privacy/user-experience tradeoffs than SForum needs for
the first milestone.

## Core Decisions

- Use ALTCHA as the default human-verification provider.
- Treat CAPTCHA as one anti-automation layer, not as the full security model.
- Combine ALTCHA with Redis-backed rate limits, short challenge expiration,
  single-use challenge tracking, CSRF protection, audit logs, and email
  verification gates where product policy needs them.
- Do not challenge every login by default. Challenge registration, password
  reset initiation, and login only after suspicious failure patterns.
- Keep the provider replaceable through a small backend interface.
- Keep verification authoritative in the Fiber API. Nuxt renders the widget and
  carries the payload, but it does not decide whether verification passed.

## Backend Boundaries

Create a platform-level human-verification boundary during implementation:

```text
apps/api/internal/platform/humanverify
  - Provider interface
  - ALTCHA challenge generation and payload verification
  - challenge purpose names
  - verification result codes
```

Domain modules call this boundary from their services or handlers:

- `identity` requires verification for open registration.
- `identity` requires verification for password reset initiation.
- `identity` may require verification for login after repeated failures.
- `forum` may require verification for posting links, rapid replies, or other
  new-user risk events after posting exists.

Redis stores temporary anti-abuse state:

- rate-limit counters by IP, account, session, and action.
- issued challenge keys when server-side single-use tracking is needed.
- accepted challenge keys until expiration, so a valid payload cannot be
  replayed.
- login failure counters used to decide whether a login attempt needs a
  challenge.

PostgreSQL stores only durable identity and moderation facts. Do not store all
challenge attempts in PostgreSQL. Audit only meaningful security events, such as
repeated blocked registration attempts or sensitive account actions.

## Provider Interface

Use a narrow interface so the identity and forum modules do not know whether the
provider is ALTCHA, Turnstile, or disabled in local tests.

```go
type Purpose string

const (
	PurposeRegister      Purpose = "register"
	PurposePasswordReset Purpose = "password_reset"
	PurposeLoginRisk     Purpose = "login_risk"
	PurposePostRisk      Purpose = "post_risk"
)

type Challenge struct {
	Provider string
	Purpose  Purpose
	Payload  map[string]any
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
```

The implementation can adjust exact type names, but the boundary should stay
boring: generate a challenge, verify a submitted token, return a stable result.

## API Shape

Initial API surface:

- `GET /human-verification/challenge?purpose=register`
- `GET /human-verification/challenge?purpose=password_reset`

Later risk-triggered flows can reuse the same endpoint with more purposes.

Form submissions include a provider token:

- `POST /auth/register` includes `humanVerification.provider` and
  `humanVerification.token`.
- `POST /auth/password-reset/request` includes the same structure once password
  reset exists.
- Login includes verification only when the server has already told the client
  that the current attempt requires it.

Stable error codes:

- `human_verification.required`
- `human_verification.invalid`
- `human_verification.expired`
- `human_verification.replayed`
- `rate_limit.exceeded`

Nuxt maps these codes to localized messages.

## Frontend Behavior

Nuxt renders the ALTCHA widget only on forms that require human verification.

Default MVP behavior:

- Registration page always shows the widget.
- Password reset request page shows the widget when implemented.
- Login page does not show the widget until the API reports
  `human_verification.required` for the current risk state.

The widget challenge URL points at the same-origin API route so no browser-side
third-party service is required for the default ALTCHA flow.

## Security Rules

- Generate a fresh challenge for each user interaction.
- Set challenge expiration; use a short default suitable for form submission.
- Enforce single-use verification through Redis, even when cryptographic
  validation succeeds.
- Rate limit challenge generation as well as protected form submission.
- Use separate rate-limit buckets for IP, account/email/username, and action.
- Log enough metadata to debug abuse patterns without storing sensitive tokens
  or full challenge payloads.
- Keep the HMAC/signing secret in environment configuration, not in source
  control.
- Disable human verification only in explicit local/test configuration.

## Testing Strategy

Backend tests should cover:

- Registering without a required token fails with
  `human_verification.required`.
- Registering with an invalid token fails with `human_verification.invalid`.
- Reusing a previously accepted token fails with `human_verification.replayed`.
- Expired challenges fail with `human_verification.expired`.
- Registration still enforces rate limits when challenge generation is spammed.
- Login requires a challenge only after configured suspicious failure patterns.

Frontend tests should cover:

- Registration loads the widget and submits the token.
- Localized error messages render for required, invalid, expired, replayed, and
  rate-limited states.
- Login can transition from normal password form to challenge-required state
  without losing the entered identifier.

## Out Of Scope For The First Milestone

- ALTCHA Sentinel deployment.
- Machine-learning spam classification.
- Full trust-level scoring.
- Third-party provider implementation.
- Two-factor authentication.
- Device fingerprinting.

Those can be added later without changing the first provider boundary.

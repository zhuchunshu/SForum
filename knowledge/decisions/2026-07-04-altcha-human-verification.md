# Decision: ALTCHA Default Human Verification

## Status

Accepted

## Context

SForum keeps public registration open after the first administrator is created.
Open registration is appropriate for a forum, but it creates abuse surfaces:
automated account creation, password-reset abuse, credential-stuffing pressure,
and early link spam.

The project should prefer mature libraries and avoid custom security
infrastructure. Deployments may run in environments where third-party CAPTCHA
services are slow, blocked, or undesirable for privacy reasons.

## Decision

Use ALTCHA as the default human-verification provider.

ALTCHA is used as one layer in a broader anti-automation design:

- Redis-backed rate limits by IP, account, session, and action.
- Fresh single-use challenges with short expiration.
- Server-side payload verification in the Fiber API.
- CSRF protection for cookie-authenticated writes.
- Email verification and new-user posting gates where product policy needs
  them.
- Audit logs for meaningful abuse and sensitive-account events.

Keep a small backend provider interface so deployments can add Cloudflare
Turnstile later without changing identity or forum module logic. hCaptcha and
reCAPTCHA remain possible integrations but are not the default recommendation.

## Consequences

- SForum can ship a privacy-friendly default that does not require browser or
  API calls to third-party CAPTCHA services during normal verification.
- Redis becomes the short-lived store for challenge replay protection and rate
  limits.
- Human verification must not be treated as complete bot protection. Forum
  posting and account flows still need rate limits, moderation policy, email
  verification gates, and audit visibility.
- The first implementation needs environment configuration for the ALTCHA
  secret, challenge cost, expiration, and provider mode.
- A future Turnstile provider can be added if a deployment wants managed
  bot-detection and accepts the external dependency.

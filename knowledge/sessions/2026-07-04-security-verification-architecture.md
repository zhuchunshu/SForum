# 2026-07-04 Security Verification Architecture

## Changed

- Accepted ALTCHA as SForum's default human-verification provider.
- Added a security verification design spec.
- Recorded a decision record for ALTCHA-backed human verification.
- Updated architecture, product, backend, identity, research, and knowledge
  index notes with the anti-automation direction.

## Decisions

- Use ALTCHA by default for registration, password reset, and later risk-based
  actions.
- Pair ALTCHA with Redis-backed rate limits and single-use challenge tracking.
- Keep the provider replaceable so Cloudflare Turnstile can be added later for
  deployments that want managed bot detection.
- Do not challenge every login by default; challenge login only after suspicious
  failure patterns.

## Next

- Include `internal/platform/humanverify` in the identity implementation plan.
- Add provider configuration for ALTCHA secret, challenge cost, expiration, and
  disabled/test mode.
- Add registration and password-reset API contracts once those flows are
  implemented.

## Open Questions

- Exact default challenge expiration and ALTCHA cost values.
- Whether email verification should be required before first post, before links,
  or only before account recovery.

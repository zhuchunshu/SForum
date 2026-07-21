# 2026-07-04 ALTCHA Human Verification Implementation Handoff

## Changed

- Added `apps/api/app/Support/HumanVerify` with a small provider boundary,
  ALTCHA v2 adapter, in-memory store, Redis store, replay protection, and
  rate-limit helpers.
- Wired human verification into registration before account creation.
- Added `GET /api/v1/human-verification/challenge?purpose=register`.
- Added config/env support for `HUMAN_VERIFICATION_PROVIDER`,
  `ALTCHA_SECRET`, `ALTCHA_CHALLENGE_TTL`, `ALTCHA_COST`, and Redis password.
- Added Nuxt ALTCHA client plugin and registration-page widget integration.
- Updated OpenAPI with the challenge endpoint, `HumanVerification` schema, and
  registration `429` response.

## Decisions

- Use ALTCHA widget v3 with the `challenge` attribute pointing at the API
  challenge endpoint.
- Keep backend verification authoritative; the frontend submits the token but
  does not block submission purely on local widget state.
- Use Redis for production replay/rate-limit state and an in-memory store for
  tests and disabled/local flows.

## Next

- Tune production ALTCHA cost/TTL/rate-limit values on representative devices.
- Add CSRF protection for cookie-authenticated unsafe requests.
- Reuse the `HumanVerify` service for password-reset initiation and later
  risk-based login/post checks.

## Open Questions

- What target solve time is acceptable for low-end mobile devices?
- Should challenge/rate-limit buckets include session IDs once CSRF/session
  hardening lands?

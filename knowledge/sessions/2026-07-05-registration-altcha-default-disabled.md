# 2026-07-05 Registration ALTCHA Default Disabled Handoff

## Changed

- Changed API config and env examples so `HUMAN_VERIFICATION_PROVIDER` defaults
  to `disabled` instead of `altcha`.
- Added Nuxt public runtime provider wiring and made the registration page hide
  ALTCHA and omit `humanVerification` when the provider is disabled.
- Updated the identity UI validation script and OpenAPI registration contract
  so human verification is optional unless enabled by deployment config.
- Updated product, architecture, and knowledge-base notes to record the
  default-off behavior.

## Decisions

- ALTCHA remains the first supported self-hosted provider, but it is opt-in via
  `HUMAN_VERIFICATION_PROVIDER=altcha`.
- The frontend uses only the public provider name; secrets and verification
  rules stay backend-owned.

## Next

- If an admin UI for security settings is added later, do not move ALTCHA
  secrets into `web_options`; keep secrets in environment configuration.

## Open Questions

- What ALTCHA cost and challenge TTL should production use when the provider is
  explicitly enabled?

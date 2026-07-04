# 2026-07-04 API Response Envelope And Localized Message Handoff

## Changed

- Accepted a unified backend API response envelope:
  `code`, `message`, and `data`.
- Added the design spec at
  `docs/superpowers/specs/2026-07-04-api-response-envelope-localized-message-design.md`.
- Added the decision record at
  `knowledge/decisions/2026-07-04-api-response-envelope-localized-message.md`.
- Updated backend, localization, architecture, and knowledge index notes to
  replace the older frontend-maps-error-code wording.

## Decisions

- `code` is an integer equal to the HTTP status code.
- `message` is backend-owned and localized from request locale.
- `data` holds successful payloads or structured error details.
- Stable machine-readable error reasons live in `data.reason`.
- API routes should return `200` with `data: null` instead of `204 No Content`.
- Frontend should display API `message` first for API-originated prompts and
  use frontend text only for frontend-owned states or fallback behavior.

## Next

- Implement HTTP response helpers and localized error mapping under
  `apps/api/app/Http`.
- Update existing health, identity, role, and human-verification endpoints to
  return the envelope.
- Update OpenAPI responses to describe the envelope.
- Update Nuxt API consumers to unwrap `data`, send locale context, and prefer
  backend `message` for API feedback.

## Open Questions

- Whether the first implementation should include cookie/profile locale
  negotiation immediately or start with `Accept-Language` plus default locale.
- Exact English wording for each initial backend API message.

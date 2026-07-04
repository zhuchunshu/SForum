# Decision: API Response Envelope And Localized Messages

## Status

Accepted

## Context

SForum needs a stable backend API response shape. The current API returns
successful payloads directly, returns errors with a string problem code, and
uses `204 No Content` for some mutations.

The project now requires every backend API response to include `code`,
`message`, and `data`. API messages also need backend-side multilingual support
so the frontend can display operation results without duplicating most backend
response text.

## Decision

Use one envelope for backend API JSON responses:

```json
{
  "code": 200,
  "message": "OK",
  "data": null
}
```

Rules:

- `code` is an integer and equals the HTTP status code.
- `message` is backend-owned, user-facing, and localized from the request
  locale.
- `data` contains successful payloads or structured error details.
- Stable machine-readable reason codes live in `data.reason`.
- API routes should not return `204 No Content`; no-payload success returns
  `200` with `data: null`.
- HTTP status codes remain meaningful and are not replaced by the envelope.

Backend API messages support `zh-CN` and `en-US`, defaulting to `zh-CN`.
Frontend requests should carry the current locale, with the first
implementation using the standard `Accept-Language` header. Frontend UI should
display API `message` first when the response includes a prompt or operation
result. Frontend hard-coded or catalog messages remain appropriate for local
form validation, network failure, missing-response fallbacks, static UI, and
other frontend-owned states.

## Consequences

- Controllers should use shared HTTP helpers instead of hand-writing JSON
  envelopes.
- The centralized Fiber error handler must return the same envelope as success
  responses.
- OpenAPI must describe envelope-wrapped responses.
- Frontend API consumers need to unwrap `data` and update error handling to
  prefer backend `message`.
- Existing `data.reason` values remain useful for branching behavior such as
  human-verification reset flows.
- Backend localization becomes part of the API contract, not only emails and
  notifications.

## Follow-up

- Implement response helpers and localized error mapping in `apps/api/app/Http`.
- Update the existing identity, role, human-verification, and health endpoints.
- Update Nuxt API consumers to unwrap the envelope and send locale context.
- Update OpenAPI schemas and tests with the envelope.

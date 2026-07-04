# API Response Envelope And Localized Message Design

## Goal

Define one JSON response envelope for every SForum backend API response and
make backend API messages localizable.

The required envelope is:

```json
{
  "code": 200,
  "message": "OK",
  "data": {}
}
```

## Context

The current Fiber API returns successful data directly and returns errors as a
small problem object with string `code` and `message`. Some mutation endpoints
also return `204 No Content`.

SForum now requires every backend API response to include:

- `code`: integer.
- `message`: string.
- `data`: response payload chosen by the API design.

The project also requires backend APIs to support multilingual user-facing
messages. The frontend should use the backend `message` when an API response
includes a prompt or operation result, instead of duplicating most API response
copy in Vue code.

## Library Survey

No new runtime dependency is needed for the envelope itself. Go Fiber already
supports centralized error handling and JSON responses. The existing
`app/Support/Localization` package should grow into the backend locale
normalization and message lookup boundary.

OpenAPI should describe the envelope explicitly instead of relying on
convention. Frontend TypeScript can use small local generic types for API
responses until generated API clients are introduced.

## Core Decisions

- All backend API JSON responses use the same top-level envelope:
  `code`, `message`, and `data`.
- `code` is an integer and equals the HTTP status code.
- `message` is the backend-owned, request-locale-aware user-facing message.
- `data` contains the successful payload or structured error details.
- Do not return `204 No Content` from API routes that are part of this contract.
  Mutations with no domain payload return `200` and `data: null`.
- Keep HTTP status codes meaningful. The envelope does not replace HTTP status.
- Keep stable machine-readable reason codes in `data.reason`, not in top-level
  `code`.

## Envelope Shapes

Successful object response:

```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "id": 1,
    "username": "admin"
  }
}
```

Successful list response:

```json
{
  "code": 200,
  "message": "OK",
  "data": []
}
```

Successful mutation without a domain payload:

```json
{
  "code": 200,
  "message": "OK",
  "data": null
}
```

Validation or domain error:

```json
{
  "code": 422,
  "message": "请先完成人机验证。",
  "data": {
    "reason": "human_verification.required"
  }
}
```

Field validation can extend `data` later:

```json
{
  "code": 422,
  "message": "请求参数不正确。",
  "data": {
    "reason": "validation.invalid",
    "fields": {
      "email": ["邮箱格式不正确"]
    }
  }
}
```

## Backend Localization

Backend APIs must support `zh-CN` and `en-US` messages, with `zh-CN` as the
default locale.

Locale negotiation should follow the project localization direction:

1. Localized route or explicit current-locale signal from Nuxt.
2. Locale cookie when available.
3. Signed-in user profile preference when available.
4. Browser `Accept-Language`.
5. `zh-CN`.

The first implementation can have Nuxt send the current locale through the
standard `Accept-Language` header and have the API fall back to configured
`zh-CN`. Cookie and profile negotiation can be added when those boundaries
exist in the HTTP kernel.

Backend message lookup should use stable reason keys such as:

- `auth.required`
- `auth.invalid_credentials`
- `permission.denied`
- `validation.invalid`
- `human_verification.required`
- `human_verification.invalid`
- `human_verification.expired`
- `human_verification.replayed`
- `rate_limit.exceeded`
- `internal_error`

The backend should not leak internal error details in localized messages.
Unexpected errors return a generic localized message and are logged server-side.

## Frontend Contract

Frontend API consumers should unwrap the envelope before handing data to pages
and components.

When an API response includes a user-facing operation result or error prompt:

- Prefer the backend `message`.
- Use frontend hard-coded or catalog messages for client-side validation,
  network failures, missing responses, static UI copy, and fallback behavior.
- Use `data.reason` for flow control, such as resetting human verification when
  the reason is `human_verification.expired`.
- Avoid duplicating backend API messages in Vue pages unless the API response
  has no usable message.

This does not ban frontend text. It assigns ownership: backend-originated API
results use backend-localized `message`; frontend-owned UI states remain in
Nuxt i18n catalogs.

## Backend Boundaries

Add a small HTTP response helper in `apps/api/app/Http` so controllers do not
hand-roll envelope objects.

Expected helpers:

- `OK(c, data)` for `200`.
- `Created(c, data)` for `201`.
- `NoData(c)` for `200` with `data: null`.
- `Error` or centralized error handler support that maps errors to localized
  messages and structured `data.reason`.

Controllers should continue to stay thin: bind requests, call services, map
domain errors to stable reason keys, and return through HTTP helpers.

The Fiber error handler should emit the envelope for all errors, including
framework errors and unexpected panics recovered by middleware.

## OpenAPI Contract

OpenAPI should define reusable envelope schemas:

- `ApiResponse_CurrentUser`
- `ApiResponse_Role`
- `ApiResponse_RoleList`
- `ApiResponse_AltchaChallenge`
- `ApiResponse_Null`
- `ApiErrorResponse`

The exact schema names can change during implementation, but each endpoint must
show the envelope at the response boundary so generated clients and humans see
the same contract.

## Testing Strategy

Backend tests should verify:

- Health returns `code/message/data`.
- Successful identity endpoints wrap current user data.
- Successful list endpoints wrap arrays under `data`.
- No-payload mutations return `200` with `data: null`.
- Known domain errors return HTTP status as `code`, localized `message`, and
  stable `data.reason`.
- `Accept-Language: en-US` returns English API messages.
- Unsupported locales fall back to `zh-CN`.

Frontend tests should verify:

- Auth session and role pages unwrap `data`.
- API error handling displays backend `message` first.
- Human-verification flow still uses `data.reason` for component reset logic.
- Network failures still use frontend fallback messages.

## Migration Notes

The first migration should update the existing identity and health endpoints
only:

- `/health`
- `/human-verification/challenge`
- `/auth/register`
- `/auth/login`
- `/auth/logout`
- `/auth/session`
- `/roles`
- `/roles/{roleKey}`
- `/roles/{roleKey}/permissions`

Future endpoints must use the envelope from the start.

## Out Of Scope

- Generated OpenAPI TypeScript clients.
- Full field-level validation catalogs for every form field.
- Locale negotiation from signed-in profile before the HTTP session/profile
  boundary is ready.
- Changing non-API frontend static UI copy ownership.

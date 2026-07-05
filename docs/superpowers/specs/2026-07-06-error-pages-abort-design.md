# Error Pages And Abort Helper Design

## Status

Approved for planning.

## Context

SForum currently has a consistent API error envelope in `apps/api/app/Http`
and a Nuxt frontend using the Pine Teal `SF*` component system. The backend
already exposes `NewError(status, reason)` and the Fiber error handler maps
missing API routes into localized JSON envelopes, but controllers do not yet
have a Laravel-like `abort(...)` convenience API. The frontend also relies on
Nuxt's default error rendering instead of a project-owned 404/503 experience.

The user reviewed three browser mockups and selected direction A:
community-style empty state. The first release will use one shared public error
page for frontend and admin routes. A dedicated admin-dashboard error state can
be added later if the dashboard needs different density or navigation.

## Library Survey

- Frontend routing and error rendering should use Nuxt's native `error.vue`,
  `createError`, and `clearError` behavior instead of a custom router or error
  boundary library.
- Icons should continue using the existing Nuxt Icon integration with Lucide or
  Tabler collections. No new icon dependency is needed.
- Backend HTTP errors should keep Fiber's native status handling and the
  existing SForum `APIError` envelope helper. No third-party abort package is
  needed.

## Goals

- Provide a polished project-owned error page for common public rendering
  failures: `404`, `403`, `500`, and `503`.
- Keep the selected visual direction: a forum-native empty-state page with the
  SForum navbar, footer, teal accent, thin borders, familiar actions, and no
  emoji or decorative icon hacks.
- Add a Go-explicit Laravel-style helper for backend controllers:
  `Abort`, `AbortIf`, and `AbortUnless`.
- Preserve the existing API response envelope:
  `{ "code": status, "message": localizedMessage, "data": { "reason": reason } }`.
- Keep first release scope small: no admin-only error layout, no configurable
  error-page settings, and no new permission keys.

## Non-Goals

- Do not build a CMS for custom error page copy in this release.
- Do not introduce a panic/recover-based abort flow. Controller code should
  still return errors explicitly.
- Do not change OpenAPI response envelope schemas unless an endpoint response
  shape changes independently.
- Do not add application routes only for demo screenshots after the real
  `error.vue` exists.

## Frontend Design

Add a root-level Nuxt `error.vue` that renders a full-page SForum error state.
It should reuse the default site chrome (`SFNavbar` and `SFFooter`) so users
remain inside the forum experience even when a route fails. The main panel uses
the selected community empty-state layout:

- small status badge with the HTTP status code,
- concise localized title and description,
- icon from Nuxt Icon, chosen by status category,
- primary action to return to the forum homepage,
- secondary action to go back when browser history is available,
- refresh/retry action for `500` and `503`.

Status copy should be localized in `apps/web/i18n/locales/zh-CN.json` and
`apps/web/i18n/locales/en-US.json`. Known statuses get specific messages:

- `404`: resource or page not found,
- `403`: no permission to view the page,
- `503`: service temporarily unavailable,
- `500`: unexpected server error.

Unknown statuses fall back to a generic error message while still displaying
the numeric status code.

The page should set SEO metadata to `noindex` and use the runtime site name in
the title. It should avoid exposing sensitive backend details from
`statusMessage`, especially for `500`-class errors.

## Backend Design

Add a small helper file under `apps/api/app/Http` with:

```go
func Abort(status int, reasons ...string) *APIError
func AbortIf(condition bool, status int, reasons ...string) *APIError
func AbortUnless(condition bool, status int, reasons ...string) *APIError
```

`Abort` returns an `*APIError` so controller handlers can write:

```go
return http.Abort(fiber.StatusNotFound)
return http.Abort(fiber.StatusForbidden, "permission.denied")
```

`AbortIf` and `AbortUnless` return `nil` when the condition does not abort, so
controllers can write:

```go
if err := http.AbortUnless(policy.CanManage(user), fiber.StatusForbidden, "permission.denied"); err != nil {
    return err
}
```

Default reason mapping should reuse existing localization keys:

- `400` -> `validation.invalid`,
- `401` -> `auth.required`,
- `403` -> `permission.denied`,
- `404` -> `not_found`,
- `405` -> `method_not_allowed`,
- `429` -> `rate_limit.exceeded`,
- all other statuses -> `internal_error`.

If a controller passes a reason explicitly, the helper uses that reason as the
machine-readable `data.reason` and localization key, matching existing
`NewError` behavior.

## Data Flow

Frontend route errors flow through Nuxt's native error lifecycle into
`error.vue`. The page derives the status code from the `NuxtError`, picks
localized UI copy, and offers navigation actions through Nuxt routing and
`clearError`.

Backend controller aborts flow through `Abort` helpers into the existing Fiber
error handler. The error handler already recognizes `*APIError` and returns the
localized SForum JSON envelope.

## Permissions

No new permission keys are required. Error rendering itself is public. Protected
routes and admin operations must continue checking authorization before calling
`Abort(fiber.StatusForbidden, "permission.denied")`; frontend route guards
remain only usability helpers.

## Testing

Backend tests should be written first for:

- `Abort(404)` produces an `APIError` with status `404` and reason `not_found`,
- explicit reasons are preserved,
- `AbortIf(false, ...)` and `AbortUnless(true, ...)` return `nil`,
- `AbortIf(true, ...)` and `AbortUnless(false, ...)` return the expected
  `APIError`,
- a test route returning `Abort(503)` still renders the existing localized API
  envelope.

Frontend verification should include:

- Nuxt typecheck,
- rendered 404 page in a browser viewport,
- rendered 503 page through an induced Nuxt error state or temporary test route,
- mobile viewport check for no overflow and readable actions.

Run `./scripts/test.sh` before final handoff if the local dependency services
are available. If they are not available, run the narrower backend and frontend
checks and report the limitation.

## Documentation And Knowledge Base

After implementation, update the frontend and backend module notes with the new
error-page and abort-helper behavior. Add a short session handoff with changed
files, verification commands, and any remaining follow-up such as a future
admin-specific error layout.

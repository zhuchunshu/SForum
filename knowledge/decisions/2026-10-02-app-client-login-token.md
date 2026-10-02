# App And Machine Clients Mint A PAT During Login

Date: 2026-10-02

## Context

SForum's personal access token (PAT, `sft_...`) is the only non-cookie
credential, and it is the right credential for a native app: `Bearer sft_...`
skips CSRF (`apps/api/app/Http/server.go`), the actor is narrowed to
`scopes ∩ current permissions` (`app/Models/APITokens/scoped_store.go`), and
`GET /auth/session` already resolves a bearer actor
(`app/Http/Controllers/Identity/session_handlers.go`).

The problem is acquisition. Token management routes deliberately require a
cookie credential — both the core guard evaluator
(`app/Http/core_guard_identity_self.go` → `revoke_apitoken` /
`rotate_apitoken` use `requireCookieCredentialAuthority`) and the controller
(`api_token.cookie_required`) reject bearer callers. So an app had to:

1. `POST /auth/login` (password) and keep the session cookie,
2. read the readable `csrf_` cookie and echo it as `X-Csrf-Token`,
3. `POST /auth/tokens` with that cookie plus CSRF header,
4. discard the cookie and use the returned `sft_...`.

That two-phase dance exists purely because PAT creation is modeled as an
account-security *management* action, not as a login outcome. It forces every
native client to implement cookie-jar persistence, CSRF double-submit handling,
and an ordering constraint that has no security value for the client itself: the
credential being proven in step 3 is the same password proven in step 1.

Alternatives rejected:

- A dedicated `POST /auth/tokens/exchange` endpoint that accepts a password
  would create a second credential-verification authority, duplicating lockout,
  risk evaluation, human verification, session policy, and login auditing.
- Letting bearer tokens manage tokens would weaken the deliberate theft-surface
  mitigation (a stolen token could mint or revoke siblings).
- Raising `MaxScopes` so a token can inherit every account permission would
  silently broaden authority; the existing permission catalog plus plugin
  permission keys already sits at the 64-scope ceiling.

## Decision

- `POST /auth/login` accepts an optional `issueApiToken` object with exactly the
  `POST /auth/tokens` shape (`name`, `scopes`, optional `expiresAt`).
- One credential proof, one authority: token minting reuses
  `APITokens.Create`, so scope narrowing, the `super_admin` strip, hashing,
  audit, and expiry rules are identical to the existing endpoint.
- `scopes` stays explicit and must be a subset of the account's current
  permissions. There is no implicit "inherit all permissions" default, because
  that set can exceed `MaxScopes` and would be a silent privilege grant.
- Request shape validation happens before session issue, so a malformed token
  request returns `422` (`api_token.name_required`,
  `api_token.scopes_required`, `api_token.invalid`) without issuing a session.
- The response shape is additive, not rewritten: `data` stays `CurrentUser` when
  `issueApiToken` is absent; with it, `data` becomes
  `LoginIssuedToken { user, apiToken }`. Browsers never send the field, so the
  existing web contract is untouched.
- The plaintext secret is returned only in that login response, exactly once,
  matching `POST /auth/tokens`.
- Token management (list/create/rotate/revoke) remains cookie-only. App sign-out
  clears the local secret; server-side revocation still requires the browser
  session or the account-security page.
- The login risk path is unchanged: if the account requires human verification,
  the client solves the challenge and retries the same request with
  `issueApiToken` intact.

## Consequences

- A native app can authenticate with one request and then run entirely on
  `Authorization: Bearer sft_...`, with no cookie jar, no CSRF handling, and no
  ordering constraint.
- The API gains a second response shape on one route, expressed in OpenAPI as
  `ApiResponseLogin` with `data` as `oneOf [CurrentUser, LoginIssuedToken]`.
- Removing a token from a lost device still depends on the web session. A
  token-authenticated self-revoke would require adding bearer token identity to
  `routes.DispatchRequest` and relaxing a reviewed guard posture; it is
  deliberately out of scope and remains an open gap for App sign-out.
- App-facing client bootstrap (minimum supported version, maintenance notice,
  capability flags) is still unowned. Public site configuration already has a
  single owner in `GET /web-options`, so no aggregate endpoint was added.

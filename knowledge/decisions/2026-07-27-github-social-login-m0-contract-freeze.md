# GitHub Social Login M0 - Contract Freeze And Library Survey

Date: 2026-07-27
Status: Accepted (M0 contract audit output; **corrected by M1R/T1E 2026-07-27**)

## Context

The task book
(`plans/2026-07-27-github-social-login-builtin-plugin.md`) requires M0 to
audit the existing baseline, verify current official GitHub OAuth App behavior,
complete the OAuth library survey, and freeze the additive schemas/config before
any production behavior changes. This record captures the M0 output so later
milestones can refer back without re-deriving it.

**M1R/T1E correction:** Host consistently owns OAuth `state`, PKCE verifier,
absolute callback URL, and the callback transaction. Earlier wording that
implied plugin-owned state/callback was wrong and is corrected below. Frozen
start/complete schemas list the actual additive Host fields used by
implementation.

## GitHub OAuth App Behavior - Verified Sources

Verified against official GitHub Docs on 2026-07-27.

Sources:

- Authorizing OAuth Apps:
  https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps
  (accessed 2026-07-27)
- REST API - Users:
  https://docs.github.com/en/rest/users/users
  (accessed 2026-07-27)
- Creating a GitHub App / OAuth App basics:
  https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/creating-an-oauth-app

Verified facts V1 will rely on:

- Authorization endpoint: `GET https://github.com/login/oauth/authorize`
- Token endpoint: `POST https://github.com/login/oauth/access_token`
- Authorization Code grant only. The implicit grant type is NOT supported.
- PKCE support: GitHub supports PKCE with `code_challenge_method=S256` only.
  `plain` is NOT supported. PKCE is strongly recommended.
- `state`: unguessable random string, strongly recommended for CSRF defense.
- `scope`: space-delimited. Omitted = no scopes for new users.
- Token response (with `Accept: application/json`): `{ access_token, scope,
  token_type }`. Errors return JSON `{ error, error_description, error_uri }`.
- Callback/redirect URL: host (excluding subdomains) and port MUST match the
  registered OAuth App callback URL; path MUST be a subdirectory of it.
- `GET https://api.github.com/user` returns `id` (stable integer, int64),
  `login` (mutable username), `name` (nullable), `avatar_url`, `email` (nullable
  - it is only the public email; null if user has no public email), `node_id`.
- `id` is the durable stable subject. `login` is presentation only.
- Email: `/user` returns the public email only and it may be null. To obtain the
  primary verified email reliably, the plugin MUST request the `user:email`
  scope and call `GET https://api.github.com/user/emails`. SForum V1 treats any
  email as a registration hint only and never matches/links by email.
- Minimum scopes for V1 identity proof: no scope is strictly required to read
  the public profile (id/login/name/avatar_url), but SForum requests
  `read:user` so the plugin can read the authenticated user profile through the
  token. `user:email` is requested so the primary verified email can be used as
  a registration hint. The plugin discards the access token immediately after
  identity proof; V1 does no retained GitHub API access.

Endpoint URLs are fixed official GitHub.com URLs in V1. Operator-configurable
OAuth issuer/base URL is deferred (no Enterprise Server support in V1).

## Library Survey - x/oauth2 vs goth vs Direct HTTP

| Option | Fit for SForum | Decision |
| --- | --- | --- |
| `golang.org/x/oauth2` | Mature, narrowly-scoped OAuth 2.0 protocol helper. Builds auth URL, PKCE, token exchange, and HTTP client wrapping. It does NOT own provider registration, callbacks, state, routing, cookies, sessions, or storage. Composes cleanly with Host-owned state/callback. | Selected |
| `github.com/markbates/goth` | Higher-level multi-provider library. Owns provider registry, session store, callback handler, and route integration. Conflicts with Host-owned callback state + versioned Identity Registry. | Rejected |
| Direct HTTP (`net/http`) | Hand-rolling OAuth 2.0 + PKCE + JSON token parsing duplicates mature code. | Rejected for token exchange; still used for GitHub user/email JSON fetch |

Decision: use `golang.org/x/oauth2` for OAuth 2.0 protocol mechanics and
standard `net/http` for profile/email fetches.

## Host Ownership (authoritative)

The SForum **Host alone** owns:

| Material | Owner | Notes |
| --- | --- | --- |
| OAuth `state` | Host | High-entropy; generated at start; bound into callback transaction |
| PKCE `code_verifier` | Host | Generated at start; stored only in callback transaction; injected into complete |
| PKCE `code_challenge` | Host | Derived from verifier; passed to plugin start for authorize URL only |
| Absolute callback URL | Host | From trusted `APP_URL` (+ reserved Core path); never request `Host` |
| Callback transaction | Host | Redis/memory store under hash of opaque state; 10-minute TTL; atomic one-use |
| Subject HMAC digest | Host | `HMAC-SHA256(IDENTITY_SUBJECT_HMAC_SECRET, providerId \|\| 0x00 \|\| subject)` |
| Users, links, sessions, risk | Host | Plugin never mutates these |

The plugin receives bounded transient inputs (state, code_challenge, callbackUrl
on start; completionToken/code, codeVerifier, callbackUrl on complete), builds
or exchanges with GitHub, and returns a bounded assertion. **Neither browser
input nor plugin storage may replace the Host-owned callback transaction.**

## Existing Baseline - Reuse, Confirmed By Audit

- `app/Support/IdentityRegistry` multi-provider catalog and `identity.runtime@1`
  Protocol V2 transport (`ProviderCall`).
- `app/Http/Controllers/Identity/auth_providers.go` start transport + reserved
  Core callback route in `external_auth.go`.
- `app/Models/Identity/auth_provider_flow.go` exact-invocation fencing;
  complete is assertion-only.
- `identity_external_links` table + `ExternalIdentityLinkStore` (Link/Unlink/
  Erase/Get/FindActive/ListUser; `LinkTx` / `TransitionTx` for transactional
  composition).
- `app/Support/AuthSession` session Begin/Destroy/renewal/revocation.
- `risk.evaluate` / `session.evaluate` evaluators wired to login/register.
- `SecretStore` with namespace isolation.
- `SyncBuiltins` stages built-in packages from `extensions/builtin/`.

## Subject Digest Contract Correction

- Plugin returns bounded raw GitHub subject (`providerSubject`) only inside the
  typed internal plugin response.
- Core validates it, then computes the keyed HMAC digest and stores only the
  digest.
- Existing 64-char lowercase hex validators validate the Core-computed digest.
- Production secret `IDENTITY_SUBJECT_HMAC_SECRET` on `config.Config`; production
  rejects missing/weak/default; development uses stable configurable default.
- Rotation needs a future versioned dual-read migration.

## Frozen Schemas (actual additive fields)

### Plugin start input (`start.input` / Host → plugin)

| Field | Source | Required |
| --- | --- | --- |
| `correlationId` | browser/Host | yes |
| `deviceFingerprint` | browser | optional |
| `clientClass` | browser | optional |
| `redirectHint` | browser; Host validates safe local path before store/plugin | optional |
| `accountHint` | browser (recovery) | optional |
| `actorUserId` | Host session (link only) | link |
| **`state`** | **Host-generated** | yes for OAuth |
| **`codeChallenge`** | **Host-generated (S256)** | yes for OAuth |
| **`codeChallengeMethod`** | Host fixed `S256` | yes for OAuth |
| **`callbackUrl`** | **Host absolute URL from APP_URL** | yes for OAuth |

### Plugin start output

- `status`: `continue|redirect|challenge`
- `continueToken`, `redirectUrl`, `challengeKind` as applicable
- GitHub emits `redirect` with the GitHub authorize URL

### Plugin complete input (`complete.input` / Host → plugin)

| Field | Source | Required |
| --- | --- | --- |
| `correlationId` | Host callback transaction | yes |
| `completionToken` | browser OAuth `code` (not trusted bindings) | yes |
| `deviceFingerprint` / `clientClass` | transaction | optional |
| `idempotencyKey` | Host (`callback:` + state) | yes |
| `actorUserId` / `targetUserId` | transaction (link) | link |
| **`codeVerifier`** | **Host transaction only** | yes for OAuth |
| **`callbackUrl`** | **same Host absolute URL as start** | yes for OAuth |

Browser must not supply trusted transaction bindings, verifier, or callback URL.

### Plugin complete output (auth)

- `providerSubject` (string, GitHub numeric id as decimal)
- `displayName`, `emailHint`
- Plugin MUST NOT emit a digest for auth completes; Core computes keyed digest

### Host public/admin response schemas

- Public `GET /auth/providers`: `activatedOperations` per provider; only
  effectively available providers; no raw subject/digest/token/secret
- Admin `GET /admin/identity/providers`: artifact, discovered/trusted/enabled/
  probed/activated, callback path, redacted probe reason
- Self-service `GET /auth/external-identities`: linkId, providerId, status,
  linkedAt, ownerExtensionId only
- No public response carries raw subject, digest, code, token, state, verifier,
  or Client Secret

## Frozen Host Additions

- Reserved Core callback route in Core Route Catalog (closed to Route Registry
  replacement): `GET /api/v1/auth/providers/{providerId}/callback`
- Why closed: callback state, PKCE verifier, absolute callback URL, and browser
  session issue/renew/destroy are identity integrity controls. Allowing Route
  Registry replace would let a plugin intercept OAuth codes, forge sessions, or
  skip exact-artifact/activation fencing.
- Public `POST …/{operation}/complete` for login/registration/link returns
  `410 auth.provider_callback_required` (recovery complete remains)
- Redis/memory callback transaction store: 10-minute TTL, hashed keys, atomic
  one-use consume, used-tombstone replay detection
- Opaque one-time external-registration ticket: 10-minute TTL,
  operation/provider/artifact bound, consumed in user+role+link TX
- `identity.provider.manage` for admin activation/probe/reset
- Provider activation/order with CAS and audit; defaults all off

## Frozen Stable Error Reasons

New: `auth.provider_not_enabled`, `auth.provider_callback_expired`,
`auth.provider_callback_invalid`, `auth.provider_callback_replayed`,
`auth.provider_callback_required`, `auth.external_identity_unlinked`,
`auth.external_registration_ticket_invalid`,
`auth.external_registration_ticket_expired`,
`auth.last_login_method_required`. Plus existing
`auth.provider_not_found`, `auth.provider_unavailable`,
`auth.provider_input_invalid`, `auth.external_subject_conflict`,
`auth.external_link_conflict`, `auth.registration_disabled`, and existing
risk/session policy denial reasons.

## Exit Criteria For M0

This record + the audited baseline map above are the M0 output. M0 introduced
no production behavior change. Implementation proceeded milestone by milestone
from M1; M1R/T1E corrected ownership wording and additive schema fields to match
the running Host foundation.

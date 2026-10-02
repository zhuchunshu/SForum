# API usage

[← Development guide](./README.md)

For integrators: how to call the SForum JSON API, including authentication,
CSRF, tokens, and the unified response envelope. The OpenAPI contract at
[`contracts/openapi.yaml`](../../../contracts/openapi.yaml) is authoritative;
this page is an entry point only and must not invent endpoints.

## Basics

- Base URL: `/api/v1` (browsers go through Nuxt's same-origin proxy; servers
  and scripts may hit the API loopback port directly).
- Response envelope: `code` (integer, equal to the HTTP status), `message`
  (localized by the backend), `data`; a stable machine-readable reason lives at
  `data.reason` and field-level errors at `data.fields`.

```json
{
  "code": 422,
  "message": "注册失败：请按标出的提示修改后再提交。",
  "data": {
    "reason": "auth.register_invalid",
    "fields": { "username": ["请填写用户名。"] }
  }
}
```

## Authentication

### 1. Browser session (cookie)

`POST /auth/register` and `POST /auth/login` issue a session cookie on success;
browsers send it automatically. `GET /auth/session` returns the current user.

### 2. Personal Access Token (PAT)

For scripts and external services:

1. Create a token on the `/settings/tokens` page or via
   `POST /auth/tokens` (cookie session only). Tokens look like
   `sft_<publicId>.<secret>`; the secret is returned once at creation.
2. Call APIs with `Authorization: Bearer sft_<publicId>.<secret>`.
3. Scopes are restricted to permission keys the user already holds; the
   `super_admin` bypass is stripped, so a PAT can never act as an unlimited
   session.
4. Token management endpoints (list/revoke/rotate) reject Bearer auth with
   `api_token.cookie_required`. Unsafe requests with a cookie session still
   need CSRF (below); unsafe requests with a valid PAT skip it.

### 3. Native app / machine clients: log in and receive a token

Apps do not need the two-step "log in for a cookie, then exchange the cookie for
a token" dance. The login request can mint a PAT in the same round trip:

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "login": "alice",
  "password": "…",
  "issueApiToken": { "name": "SForum iOS", "scopes": ["topic.create", "post.create"] }
}
```

- `issueApiToken` is optional. When omitted, `data` stays `CurrentUser` (the
  browser path is unchanged).
- When present, `data` becomes `{ user, apiToken }`; `apiToken.token` is the
  plaintext `sft_…`, **returned only in this response**. Store it in the
  Keychain / Keystore immediately.
- Rules match `POST /auth/tokens` exactly: `name` is required, `scopes` must be
  declared explicitly and be a subset of the account's current permissions, and
  `expiresAt` is an optional RFC 3339 timestamp. Scope escalation returns
  `422 api_token.invalid`.
- Malformed token requests fail before any session is issued (`422` with
  `api_token.name_required` or `api_token.scopes_required`), so there is no
  half-successful state.
- Afterwards a single `Authorization: Bearer sft_…` header is enough, including
  for `GET /auth/session` and the notification stream. No cookie, no CSRF header.
- Login risk control (`humanVerification`) and account lockout apply to this path
  too: solve the challenge and retry the same request.
- On sign-out, drop the local token; server-side revocation still goes through
  `DELETE /auth/tokens/{tokenID}` (browser session required, or the user acts on
  `/settings/tokens`).

- Client version policy (`client.*`, Site settings → Client apps):
  `client.minimum_version` drives a forced-update screen,
  `client.recommended_version` drives a dismissible prompt, and
  `client.update_notice` carries the copy. All empty means no restriction; the client
  compares versions itself. Maintenance state is public here too:
  `site.maintenance.enabled` / `site.maintenance.message`.

### 4. Native push device registration

After obtaining an FCM / APNs token, the app registers the device with its session
(or an `sft_` token):

```http
POST /api/v1/push/devices
Content-Type: application/json

{
  "deviceId": "0f0a1b2c-3d4e-5f60-7a8b-9c0d1e2f3a4b",
  "platform": "ios",
  "token": "<FCM registration token or APNs device token>",
  "appVersion": "1.4.0",
  "locale": "zh-CN",
  "deviceName": "iPhone"
}
```

- Core owns ownership and lifecycle only: the token is stored as a SHA-256 digest
  (for dedupe) plus Core-key ciphertext, and **no response ever echoes it**.
- Idempotent: one row per `(platform, token)`; re-registration refreshes
  `lastSeenAt` and rebinds ownership to the current account (account switch,
  token rotation, reinstall).
- Token rotation: older tokens on the same `deviceId` are revoked automatically so
  deliveries never target a stale token.
- List: `GET /api/v1/push/devices` (`includeRevoked=true` includes revoked devices).
- Sign-out: `DELETE /api/v1/push/devices/{deviceId}`; a `404
  notification.push_device_not_found` means the device does not exist or belongs to
  another account — both cases are indistinguishable on purpose.
- Actual delivery (FCM/APNs sends) belongs to a notification channel provider
  plugin; Core ships no vendor SDK.

## CSRF

All unsafe methods (POST/PUT/PATCH/DELETE) under `/api/v1` are protected by
double-submit CSRF:

1. Fetch a safe request (GET/HEAD/OPTIONS) first to obtain the readable
   `csrf_` cookie.
2. Send unsafe requests with an `X-Csrf-Token: <csrf_ value>` header.
3. Missing/mismatched tokens return `403` with `data.reason = "csrf.invalid"`;
   untrusted `Origin`/`Referer` return `403` with
   `data.reason = "csrf.origin_invalid"`.
4. Trusted origins are configured via `CSRF_TRUSTED_ORIGINS` (defaults to the
   `APP_URL` origin).

CSRF exemptions:

- requests with a valid `Authorization: Bearer sft_…` (non-browser clients)
  skip CSRF;
- the inbound webhook path `POST /webhooks/inbound/{source}` skips CSRF (it is
  currently a gateway skeleton: it acknowledges non-empty bodies only;
  plugin verify/parse hooks are not wired yet).

## Idempotency

`POST /topics` and `POST /topics/{topicID}/comments` accept an **optional**
`Idempotency-Key` header for safe retries; omitting it is not an error. Only
plugin routes that declare required replay in the OpenAPI reject missing or
invalid keys with `400`; the other semantics (`409` conflict, `503` storage
failure, …) are documented at the top of the OpenAPI entrypoint.

## Common endpoints (excerpt; the OpenAPI is authoritative)

| Use | Endpoint |
| --- | --- |
| Register / login / logout | `POST /auth/register`, `POST /auth/login`, `POST /auth/logout` |
| Current user / locale / appearance | `GET /auth/session`, `PUT /auth/locale`, `PUT /auth/appearance` |
| Sessions | `GET /auth/sessions`, `DELETE /auth/sessions/{sessionId}`, `POST /auth/sessions/revoke-others` |
| PATs | `POST /auth/tokens`, `GET /auth/tokens`, `DELETE /auth/tokens/{tokenID}`, `POST /auth/tokens/{tokenID}/rotate` |
| Password | `POST /auth/password-reset/request`, `POST /auth/password-reset/confirm`, `POST /auth/password` |
| Email verification | `POST /auth/email-verification/request`, `POST /auth/email-verification/confirm` |
| External login | `GET /auth/providers`, `POST /auth/providers/{providerId}/{operation}/start`, `POST /auth/providers/{providerId}/{operation}/complete`, `GET /auth/providers/{providerId}/callback`, `GET /auth/external-identities` |
| Inbound webhook | `POST /webhooks/inbound/{source}` (gateway skeleton, CSRF-skipped) |
| Health / ready | `GET /api/v1/health`, `GET /api/v1/ready` |

Admin endpoints under `/admin/…` require backend permissions such as
`admin.access`; backend policy is authoritative. Permission keys and
user/role management: see [admin guide](../usage/admin.md).

## References

- Full OpenAPI: `contracts/openapi.yaml` (index entrypoint; paths split by
  module under `contracts/openapi/paths/`, schemas under
  `contracts/openapi/schemas/`).
- Auth and security model: [Account & security](../usage/account-security.md).

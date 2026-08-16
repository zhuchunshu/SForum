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

# 2026-07-27 Session Handoff — GitHub Social Login T7 / M5

## Status

**T7 / M5 complete. M5 exit complete.**  
**GitHub social login V1 program is closed** (M0–M5 / T1A–T7).

Do **not** start follow-on provider work (Google/OIDC/etc.) in the same
conversation that closed M5. Deferred providers remain out of V1 scope.

Prior: `sessions/2026-07-27-github-social-login-m4b-handoff.md`.

## Changed

### Lifecycle + security matrix (executable)

- `apps/api/app/Models/Identity/external_auth_m5_lifecycle_security_test.go`
  - restart (stable HMAC + callback TTL)
  - disable / uninstall / Safe Mode / ForceDrain (empty RuntimeInstanceID)
  - staged upgrade → unavailable; new-digest activation; rollback re-activation
  - trust revoke + callback during artifact change
  - replay / expiry / cross-provider / operation / actor
  - activation CAS + concurrent CAS
  - registration ticket one-use
  - subject digest isolation
  - unlink race
  - non-enumerating unlinked login
  - rate limiter unit coverage
  - public catalog entry redaction shape

### Rate limits + HTTP redaction

- `external_auth_rate_limit.go`: Host IP rate limits for start/callback
  (Redis production, memory fallback; Redis errors fail-open).
- Wired in `bootstrap/api_assembly.go` + Identity controller/provider.
- `external_auth_m5_http_test.go`: start/callback rate paths + callback
  redirect never echoes code/state/secrets; public providers list redaction.

### Extension Surface Matrix + docs

- Identity matrix: routes closed callback/session authority clarified;
  admin/public components + lifecycle opened with external-auth contracts.
- Routes catalog policy text for reserved callback expanded.
- Bilingual operator guides:
  - `docs/zh-CN/usage/github-login.md`
  - `docs/en-US/usage/github-login.md`
- Author notes in `docs/extensions/authoring-guide.md` and plugin README.
- Knowledge: identity/extensions/frontend modules, plan, index, this handoff.

## Decisions

- Start uses dedicated IP limiter **in addition to** global write limiter;
  callback GET uses dedicated limiter only.
- Rate-limit Redis failures fail-open (login availability over hard lockout).
- Intentionally closed surfaces remain Host-only: OAuth callback route, PKCE
  verifier, callback state, AuthSession issue/renew/destroy, subject HMAC.
- No product code changes to M1R–M4B flows beyond additive rate-limit hooks.

## Verification

```text
cd apps/api && go test ./app/Models/Identity/ -run 'TestM5_' -count=1
# ok

cd apps/api && go test ./app/Http/Controllers/Identity/ -run 'TestM5_|TestT1E_' -count=1
# ok

cd apps/api && go build -o /dev/null ./bootstrap/
# ok

ruby scripts/validate-openapi-refs.rb
# OpenAPI references OK: checked 2262 refs across 54 files (via ./scripts/test.sh)

./scripts/test.sh
# PASS (2026-07-27) — full repo gate exit 0
# - go test ./... ok (Identity M5 + external auth packages included)
# - compat farm / protobuf / Host API v2 docs / OpenAPI / staged trust / WS proxy ok
# - Nuxt typecheck ok
# - admin framework + identity UI + homepage/SEO/moderation/theme/SF components ok
# - V3 P0 catalogs: 278 routes, 191 UI surfaces, 99 traceability rows
#
# Gate-only unblocks (not product flow rewrites):
# - tests/validate-admin-framework.ts: register /settings/login-methods page path +
#   system-folder order; assert shouldShowAdminPageInNav (current layout)
# - catalog-identities + generate.mjs: map topic edit page + 17 Host UI islands
#   (incl. SFAuthProviderButtons / SFLinkedAccountsSection) so --check matches tree
# - validate-v3-p0-catalogs.mjs: inventory counts 278 routes / 191 UI

Browser QA (user-owned web :3000; not automated in gate):
- Password-only login/register path unchanged when no public activation
- Admin → Login methods: list/activation CAS/probe/callback copy/settings/reset
- Login/register: provider buttons only when Host-activated + trusted + configured
- Account security: linked accounts section; unlink/link feedback without secrets
- Desktop + mobile shell: no code/state/token/secret/verifier in browser history
  after callback reason redirects
- Live GitHub OAuth end-to-end requires operator OAuth App credentials (manual)
```

## Next

Program closed. Optional later tracks (not M5):

- Additional providers reusing Host contracts
- `hasPassword` self-service field for unlink UX soft-hints
- Dual-read HMAC key rotation

## Open Questions

- None blocking V1 close.

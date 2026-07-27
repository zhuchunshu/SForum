# External Auth R1-R7 Requirements And Evidence Matrix

Date: 2026-07-27

Status: **整改完成，等待独立复审**. This record is a reproducible implementation
and evidence packet. It does not self-accept or close the third-party-login
program.

## Scope

Core remains provider-generic. `sforum.auth-github` is the only executable
vendor adapter; Host owns callback state, PKCE, HMAC subject digest, users,
links, risk, sessions, recent-auth, audit, and public catalog filtering.

| Requirement | Current evidence | Result |
| --- | --- | --- |
| R1 login-effect fence | `TestR1_ExternalLoginEffectFenceBlocksEveryAuthorizationRevocationDuringRisk` uses barriers for Safe Mode, activation, publication/trust, artifact, contract, and operation revocation. Every denial asserts no redirect, cookie, session, recent-auth, or success audit. | PASS |
| R2 registration transaction | `TestRegistrationEnabledTxReadsAuthoritativePolicyAndSerializesUpdates` and `TestR2_*` use the registration `pgx.Tx` for authoritative policy, user, role, link, and audit. PostgreSQL tests observe lock acquisition without timing sleeps and prove close-race zero writes. | PASS |
| R3 redaction | Migration 059 removes prohibited historical audit keys. New writes and public/admin responses reject secret, raw subject, and subject/package digest fields. | PASS |
| R4 true lifecycle disable | `TestProductionLifecycleStackDisablesExactAuthIdentityPublication` and R7 HTTP call the real extension disable endpoint/`DisableWithInput`, then prove retired publication, empty catalog, and `503 auth.provider_unavailable`; real enable/probe/activation restores it. | PASS |
| R5 durable migration truthfulness | PostgreSQL scenarios for migrations 058-061 prove exact-evidence recovery, exclude evidence-free/wrong/tombstoned artifacts, preserve eligible full-set members, remain idempotent, and leave fresh installs untouched. | PASS |
| R6 production Vue rendering | Bun tests mount `login-methods.vue`, `SFLoginFormPage.vue`, `SFRegisterFormPage.vue`, and `SFSecuritySettingsPage.vue`. The R7 Browser packet checks rendered catalog button/password fallback at desktop and mobile breakpoints. | PASS |
| R7 integrated packet | `tests/external-auth-runtime-evidence.mjs` hard-asserted every required HTTP flow on isolated services; Browser JSON/screenshots were produced separately. Final catalog is restored and non-empty. | PASS |

## Verification Commands

Focused suites and build gates were run before this R7 packet, including:

```sh
cd apps/api
GOCACHE=/private/tmp/sforum-go-cache \
SFORUM_TEST_DATABASE_URL='postgres://sforum:sforum@127.0.0.1:15432/sforum_external_auth_r7_20260727d?sslmode=disable' \
go test -race ./app/Http/Controllers/Identity -run 'TestR1_' -count=1 -timeout 120s

GOCACHE=/private/tmp/sforum-go-cache \
SFORUM_TEST_DATABASE_URL='postgres://sforum:sforum@127.0.0.1:15432/sforum_external_auth_r7_20260727d?sslmode=disable' \
go test ./app/Models/Identity ./app/Models/Options ./app/Models/Extensions ./database/migrations \
  -run 'TestR2_|TestR3_|TestR5_|TestR7_Migration0(58|59|60|61)|TestProductionLifecycleStackDisablesExactAuthIdentityPublication' \
  -count=1 -timeout 240s

cd ../../extensions/builtin/plugins/sforum-auth-github/backend && go test ./... -count=1
cd ../../../.. && ruby scripts/validate-openapi-refs.rb
cd apps/web && bun test tests/adminLoginMethods.test.ts tests/authRouteRendering.test.ts tests/authProvidersPublicUi.test.ts tests/accountSecurityM4b.test.ts
bun run typecheck && bun run build
cd ../.. && GOCACHE=/private/tmp/sforum-go-cache ./scripts/build-builtin-plugins.sh
```

The pre-R7 full repository gate was recorded as successful against isolated
PostgreSQL in `/private/tmp/sforum-r7-full-gate-verified.log` with marker
`/private/tmp/sforum-r7-full-gate.passed`. The user subsequently requested no
repeat of the full gate or Browser run and manually verified the UI after the
R7 packet.

## Runtime Evidence

The R7 HTTP packet ran against:

- PostgreSQL: `postgres://sforum:sforum@127.0.0.1:15432/sforum_external_auth_r7_20260727d?sslmode=disable`
- Normal API: `http://127.0.0.1:8082`
- Safe Mode API: `http://127.0.0.1:8083`
- Fake GitHub OAuth: `http://127.0.0.1:18082`
- Package/version: `sforum.auth-github` `1.0.0`
- Exact package digest: `d390715efd1892bc6d7607c40cb16fa360d7f07d87ec7044e15dcf25e2a89974`

`/private/tmp/sforum-external-auth-r7/http-evidence.json` SHA-256:
`666fe6fd3dea1a0fbc5bfe26766c2847a7e8b223e5c74db84cbab958865145ac`.
Its hard assertions cover readiness, password fallback, redacted configure,
enable/probe/activation, public catalog, OAuth login and replay rejection,
explicit registration, link/unlink/password setup/last-method protection,
429 rate limit, Safe Mode, artifact drift, real disable, and real restore.
The final provider is artifact-bound and publicly activated; `/auth/providers`
contains exactly `sforum.auth-github.auth` with login, registration, and link
operations.

## Browser Evidence

The Browser packet used the temporary same-origin fixture
`http://localhost:3001/login`, proxying the R7 API on `8082` and the production
Nuxt build on `3002`; it never used or changed the user-owned port `3000`.

| Artifact | SHA-256 | Verified facts |
| --- | --- | --- |
| `/private/tmp/sforum-external-auth-r7/browser-evidence.json` | `9bb048b67268308342902cbd41311f500a1bfcf252695e2f073d0d5bee81406e` | Desktop/mobile URL and title, password input, catalog-driven button, no error overlay, no console errors, real button click |
| `/private/tmp/sforum-external-auth-r7/browser-desktop-login.png` | `99a2b068d9d2c60135087f40b794d764c26006cc4714f513902ea97df360707b` | `1440x1000` rendered login |
| `/private/tmp/sforum-external-auth-r7/browser-mobile-login.png` | `7fafc807970344a741c8b8bf81600f35ed6ed8136c795b0ed3996004fdcb0b91` | `390x844` rendered login |

The automation records that the catalog-driven provider button received a real
click. OAuth callback semantics are hard-asserted by the HTTP packet. After
that evidence was produced, the operator manually tested the provider UI and
reported no problem; this is recorded as manual confirmation, not as a
replacement for the hard HTTP assertions.

## Review Notes

- Both JSON files are redacted and their checks assert that password, client
  secret, OAuth code/token, raw provider subject, subject digest, and session
  cookies are absent.
- The exact-digest drift test mutates only the isolated fixture's durable
  activation digest, proves catalog/start fail closed, then restores through
  the actual lifecycle and activation APIs.
- An independent reviewer should rerun the listed targeted suites and inspect
  the packet/script before accepting the program. This matrix requests review;
  it does not declare the program closed.

# Built-in GitHub Social Login V1 - Task Book

Status: **active - R1-R7 remediation complete; independent re-review requested**.
M0 + T1A–T1E + T2–T7 are implemented; T8A–T8C are done. The 2026-07-27 Codex
review found security, product, and verification gaps that prevent M5/program
closure. The T8D independent review rejected login-effect fencing,
registration transaction, audit redaction, extension disable, migration, and
frontend/runtime evidence claims. Continue only through
`2026-07-27-external-auth-core-plugin-review-remediation.md`.

Date: 2026-07-27

Goal: ship one secure GitHub login provider as a protected built-in plugin while
Core remains the sole owner of SForum accounts, links, risk decisions, and
browser sessions.

Implement this task book milestone by milestone. Every milestone must leave the
repository buildable and report exact test results. Do not combine the program
into one large patch.

## Required Reading Before Coding

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/identity.md`
4. `knowledge/modules/extensions.md`
5. `knowledge/decisions/2026-07-27-github-social-login-builtin-v1.md`
6. `knowledge/decisions/2026-07-19-identity-provider-automation-authority.md`
7. `knowledge/decisions/2026-07-13-login-risk-step-up-without-account-lock.md`
8. `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
9. `extensions/README.md`
10. `docs/extensions/authoring-guide.md`
11. This task book

## Product Outcome

An operator can configure the bundled GitHub plugin from the normal admin
identity settings, confirm its exact executable artifact, enable it, inspect a
truthful configuration probe, and explicitly expose GitHub login, registration,
and account linking.

A visitor can:

- sign in when their GitHub identity is already linked;
- explicitly create a new SForum account through a GitHub-backed registration
  flow when registration policy permits it;
- sign in to an existing SForum account and then link GitHub;
- inspect and safely unlink GitHub from account security;
- add a local password before removing their final usable login method.

GitHub failure affects only GitHub. Password login and Host-owned recovery
remain available.

## V1 Scope

### Included

- GitHub.com OAuth App Authorization Code flow with PKCE where current official
  GitHub documentation supports it.
- Identity operations:
  `login.start`, `login.complete`, `registration.start`,
  `registration.complete`, `link.start`, and `link.complete`.
- Host-owned activation, callback state, explicit external registration,
  login, linking, unlinking, password setup, audit, and UI.
- Protected built-in package under
  `extensions/builtin/plugins/sforum-auth-github`.
- Local fake-GitHub tests; ordinary tests require no network or credentials.

### Deferred

- GitHub Enterprise Server and operator-configurable OAuth endpoint URLs.
- Generic OIDC, Google, Discord, Telegram, Gitee, WeChat, and QQ.
- SAML, LDAP, SCIM, passkeys, phone/SMS, and native-app deep links.
- Retained GitHub API access. V1 tokens are identity-proof material only.

## Existing Baseline - Reuse, Do Not Rebuild

| Area | Current evidence | Required treatment |
| --- | --- | --- |
| Identity Registry | `app/Support/IdentityRegistry` | Reuse the multi-provider exact-artifact catalog |
| Executable operations | auth `start/complete` operations | Preserve versioned names and fail-closed policy |
| HTTP transport | `Controllers/Identity/auth_providers.go` | Replace assertion-only public effects; do not add a parallel stack |
| Provider flow | `Models/Identity/auth_provider_flow.go` | Keep exact invocation fencing; add Host product orchestration |
| External links | `identity_external_links` + store | Reuse uniqueness, lifecycle, audit, and erase semantics |
| Plugin runtime | Protocol V2 `identity.runtime@1` | Reuse typed schemas, deadlines, and response validation |
| Sessions | `Support/AuthSession` | Remain the only browser session authority |
| Risk/session policy | existing evaluators | Run before every external session issue |
| Settings/secrets | Schema renderer + SecretStore | Keep Client Secret out of Core options and browsers |
| Built-in sync | `SyncBuiltins` + build scripts | Stage the exact GitHub artifact without auto-activation |

## Frozen Architecture Rules

### Core Owns Identity Effects

The GitHub plugin verifies GitHub and returns a bounded external assertion. Core
alone may:

- create or mutate SForum users;
- assign the default role or apply first-user rules;
- resolve and persist external links;
- inspect account status and effective permissions;
- evaluate risk and selected session policy;
- issue, renew, and revoke browser sessions;
- validate return paths, rate limit, audit, and map public errors.

The plugin never receives a Core password, password hash, raw cookie, raw
session id, CSRF token, PAT plaintext, role assignment, or permission grant.

### Built-in Is Not Activated

- Source presence and `SyncBuiltins` discovery only stage the package.
- Executable trust remains exact-artifact and `super_admin` controlled.
- Plugin enable does not expose a login button.
- Host owns durable, revisioned, audited activation for `login`,
  `registration`, and `link`; all default off.
- Effective availability requires exact active bytes, compatible operations,
  required configuration, Host activation, and non-Safe-Mode runtime.
- Artifact changes invalidate public activation until the new digest is
  deliberately confirmed.

### Account Matching And Registration

- GitHub's stable numeric user `id` is the external subject. Login and display
  names are mutable presentation data.
- Email is a registration hint only. Never log in, link, or merge by email,
  including a GitHub-verified primary email.
- An unlinked `login.complete` returns a generic unlinked outcome; it does not
  create an account.
- Registration is an explicit separate user choice and operation.
- The first `super_admin` must still use Core bootstrap registration.
- GitHub registration obeys `identity.registration.enabled` and
  `identity.registration.mode`.
- User creation, default-role assignment, external link, and audit boundary
  commit in one database transaction.

### External-only Accounts

- A user may exist without a row in `user_credentials`; never create a fake
  password.
- Password login treats a missing credential as generic invalid credentials.
- Password reset and admin/self-service password setup create a credential when
  absent.
- A user cannot unlink their final active login method.
- Sensitive link, unlink, and first password setup require recent
  authentication or explicit GitHub step-up bound to the current actor.

### Host Callback And Continuation State

- OAuth callbacks use a reserved Core route outside Route Registry and theme
  replacement authority.
- Core creates high-entropy state, correlation id, and PKCE material.
- Shared Redis stores bounded callback transactions for 10 minutes and consumes
  them atomically once.
- State binds provider, operation, linking actor/session evidence, exact
  artifact, safe local return path, PKCE material, and creation time.
- Missing, expired, replayed, cross-provider, cross-operation, cross-actor, and
  artifact-mismatched callbacks fail closed.
- A completed registration assertion is retained only behind a Host-generated
  opaque one-time Redis ticket. The browser never receives a raw subject or
  stable subject digest.
- Registration tickets expire after 10 minutes, are operation/artifact bound,
  and are consumed in the user+role+link transaction.

### Subject Digest Correction

The fixture currently returns an unkeyed SHA-256-shaped value while persistence
describes a keyed digest. V1 must make the contract truthful:

- The plugin returns the bounded raw GitHub subject only inside the exact typed
  plugin response.
- Core validates it, then computes
  `HMAC-SHA256(identity-subject-key, provider-id || 0x00 || subject)`.
- Core stores only the digest. Raw subject must not enter logs, audits,
  browser/API responses, callback URLs, or durable Core storage.
- Add a dedicated stable production secret such as
  `IDENTITY_SUBJECT_HMAC_SECRET`; reject weak/default values in production.
- Document that the key is part of identity backup/restore. Rotation needs a
  future versioned dual-read migration before changing it on a populated site.
- M0 must check for non-fixture durable external-link rows before changing the
  assertion contract.

## GitHub Plugin Contract

### Package Identity

- Directory: `extensions/builtin/plugins/sforum-auth-github`
- Extension id: `sforum.auth-github`
- Provider id: `sforum.auth-github.auth`
- Provider kind: `auth`
- Runtime feature: `identity.runtime@1`
- UI icon: approved Tabler/Nuxt Icon `brand-github`

### Configuration

- `clientId`: required non-secret string.
- `clientSecret`: required SecretStore-backed secret, never returned to the
  browser.
- Callback URL: computed by Host and displayed read-only with copy action.
- V1 uses fixed official GitHub.com endpoints; no arbitrary issuer/base URL.
- Reset turns operations off and resets order while preserving secrets, with
  that preservation stated explicitly.

### Protocol Behavior

M0 must verify current official GitHub documentation and record source URLs and
check date in the plugin README:

- authorization and token endpoints;
- OAuth App callback restrictions;
- Authorization Code and PKCE support;
- minimum scopes;
- stable user subject field;
- authenticated user and email endpoints;
- token response formats, errors, and rate limits.

Expected baseline is `golang.org/x/oauth2`. Compare it briefly with `goth` and
direct HTTP. Prefer a protocol helper that does not take over SForum provider
registration, callbacks, state, routing, or sessions.

The plugin:

- builds the authorization URL using Host state, callback, and PKCE challenge;
- exchanges the code with bounded timeouts;
- fetches the authenticated user and verified primary email when required;
- returns stable numeric GitHub `id`, display hints, and email hint;
- discards access tokens immediately after identity proof;
- redacts codes, tokens, Client Secret, raw subject, and upstream error bodies.

The bounded probe can prove settings presence, endpoint reachability, and
response shape. It cannot prove a Client Secret without an authorization code;
the admin UI must not claim otherwise.

## API Target

Final names may follow existing controller conventions, but semantics cannot be
weakened.

### Public And Self-service

| Method | Suggested path | Access | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/auth/providers` | public | Effectively enabled localized provider catalog |
| `POST` | `/api/v1/auth/providers/{providerId}/{operation}/start` | contextual | Create Host transaction and authorization URL |
| `GET` | `/api/v1/auth/providers/{providerId}/callback` | public callback | Consume Host state and complete exact attempt |
| `POST` | `/api/v1/auth/external-registration` | public + ticket | Create user, role, and link atomically |
| `GET` | `/api/v1/auth/external-identities` | login | List the current user's redacted links |
| `DELETE` | `/api/v1/auth/external-identities/{linkId}` | login + step-up | Unlink with last-method protection |
| `POST` | `/api/v1/auth/password` | login + step-up | Add or change a local password |

No public response accepts or returns arbitrary user ids, artifact ids, raw
subjects, subject digests, callback URLs, roles, or provider tokens.

### Admin

| Method | Suggested path | Permission | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/admin/identity/providers` | `identity.provider.manage` | Catalog, activation, health, callback metadata |
| `PATCH` | `/api/v1/admin/identity/providers/{providerId}` | same | CAS operation activation and ordering |
| `POST` | `/api/v1/admin/identity/providers/{providerId}/probe` | same | Truthful configuration/reachability probe |
| `POST` | `/api/v1/admin/identity/providers/reset` | same | Restore operations-off defaults, preserve secrets |

Every unsafe route needs API-authoritative policy checks, stale-revision
handling, audit, and allowed plus denied tests. Executable trust remains
`super_admin` only even when provider activation is delegated.

## UX Contract

### Admin Login Methods

Add “登录方式 / Login methods” beside registration and account security:

- compact GitHub row with icon, version, exact-artifact state, and health;
- separate login, registration, and account-link toggles;
- callback URL copy and settings actions;
- truthful probe result and persistent inline configuration errors;
- last probe time and redacted reason;
- restore recommended defaults;
- theme-aware success Toasts that dismiss after 10 seconds; errors persist.

Prevent removal of the current `super_admin`'s final usable login method.

### Public And Account Security

- Render GitHub from the Host catalog, not hard-coded executable state.
- Preserve the existing validated local `redirect`.
- Keep the password form usable on cancellation or provider failure.
- Unlinked login offers safe actions: start explicit GitHub registration, or
  sign in through an existing method and link GitHub.
- Do not reveal whether a hinted email already exists.
- Registration collects required local fields and uses only the opaque ticket.
- Account security shows provider, linked time, active/inert state, link,
  unlink, recent-auth requirements, and password setup.
- Never show raw GitHub id, token, scopes, subject digest, or Client Secret.

## Milestones

### M0 - Contract Audit And Freeze

- [x] Inspect current routes, provider flow, typed schemas, external-link store,
  credential model, session issue, risk/session evaluators, built-in lifecycle,
  SecretStore, and settings renderer.
- [x] Verify official current GitHub OAuth App behavior and record sources/date.
- [x] Complete the `x/oauth2` versus `goth` versus direct-HTTP survey.
- [x] Freeze typed start/complete schemas with transient raw subject.
- [x] Freeze HMAC config, production validation, backup rule, and data
  compatibility.
- [x] Freeze callback, registration-ticket, public/admin response schemas, and
  stable error reasons.
- [x] Update this plan or add an ADR if code evidence disproves an assumption.

**Exit:** reviewed implementation map, dependency decision, official protocol
evidence, and frozen contracts with no production behavior change.

M0 was implemented, but the 2026-07-27 M1 review found contradictory ownership
wording and incomplete additive fields in its ADR. M1R must correct the ADR
before M0 is treated as the authoritative implementation contract.

### M1 - Host Security And Final Identity Effects

- [ ] Add exact-artifact provider activation/order persistence with CAS and
  audit; all defaults off.
- [ ] Filter the public catalog by activation, live registry, configuration,
  artifact match, runtime health, and Safe Mode.
- [ ] Add Redis callback transactions, secure state, PKCE, 10-minute TTL, and
  atomic one-use consume.
- [ ] Add reserved Core callback routes.
- [ ] Add Host HMAC subject derivation; remove digest from public responses.
- [ ] Add opaque one-time external registration tickets.
- [ ] Support users without password credentials and safe later setup.
- [ ] Add transactional external registration with complete rollback.
- [ ] Add external login through account reload, risk/session evaluation,
  normal AuthSession issue, session directory, and audit.
- [ ] Refactor linking so recent-auth, actor binding, artifact fencing, and Host
  policy run before persistence.
- [ ] Add redacted link list/unlink and last-login-method enforcement.
- [ ] Preserve first-user bootstrap and every registration mode.

**Exit:** the fixture provider completes real registration, login, link, unlink,
and password-setup effects without vendor code or polished UI.

### M1R - Host Foundation Remediation And Re-review

M1 is not accepted. Do not start M2 until every item below is implemented,
covered by focused tests, and independently reviewed.

#### Callback, Link, And Exact-artifact Safety

- [x] Move every link persistence effect behind current-session actor binding,
  recent-auth/step-up, operation binding, activation, and live exact-artifact
  checks. An unauthorized callback must make no database change.
- [x] Remove the public legacy
  `POST /auth/providers/{providerId}/{operation}/complete` path or reduce it to
  a strictly internal/test-only adapter that cannot bypass Host callback state,
  activation, artifact fencing, actor policy, or response redaction.
- [x] On callback, resolve the current live Registry provider and compare
  provider id, operation, owner extension/version, and package digest against
  the stored transaction. Re-run effective activation and availability after
  consuming state and before any effect.
- [x] Pass the Host-owned PKCE verifier and the same trusted absolute callback
  URL into `complete`; never reconstruct either from browser input.
- [x] Generate callback URLs from trusted public application configuration
  such as validated `APP_URL`, not request `Host`; reject startup/configuration
  when an absolute HTTPS production callback cannot be formed.
- [x] Validate `redirectHint` as a safe local path before storing or sending it
  to a plugin. Registration continuation must use the fixed Host registration
  route with an opaque ticket and separately encoded safe redirect.

**T1A done (2026-07-27):** AuthProviderFlow complete is assertion-only; public
auth complete returns `410 auth.provider_callback_required`; Host callback
re-resolves live artifact + activation before effects; link requires session
actor + recent-auth before complete/persist; absolute callback from `APP_URL`;
PKCE verifier passed into complete; redirectHint validated; registration
continuation fixed at `/register?ticket=…&redirect=…`.

#### Stable Secrets, State Stores, And Tickets

- [x] Add the subject HMAC secret to `config.Config`, validate it through the
  real `APP_ENV=production` path, and inject a stable digest service from
  bootstrap. Production must fail startup for missing/weak/default secrets;
  development must use a stable configured default rather than process-random
  material.
- [x] Make in-memory callback/ticket stores concurrency-safe and enforce the
  same TTL semantics as Redis.
- [x] Store Redis state/ticket records under hashes of opaque browser tokens,
  preserve atomic one-use consumption, and define truthful stable behavior for
  invalid, expired, and replayed inputs without logging raw material.
- [x] Populate and enforce registration ticket creation/expiry timestamps and
  operation/provider/artifact binding.

**T1B done (2026-07-27):** `IDENTITY_SUBJECT_HMAC_SECRET` on `config.Config`;
production `APP_ENV=production` rejects missing/weak/dev-default; bootstrap
`ConfigureIdentitySubjectHMAC`; stable dev default (no process-random); memory
callback/ticket stores mutex + TTL clamp + hash keys; Redis hash keys + atomic
consume + used-tombstone for replay; registration ticket
CreatedAt/ExpiresAt + operation/provider/artifact binding enforced on
Save/Consume.

#### Activation, Catalog, Probe, And Audit

- [x] Implement real atomic optimistic concurrency for activation mutations
  (`WHERE revision = expected` or an equivalent locked transaction plus
  affected-row check).
- [x] Derive provider ownership, live artifact, supported operations, trust,
  enablement, configuration, runtime health, and Safe Mode from Host state.
  Reject browser-supplied ownership/artifact claims and unsupported operations.
- [x] Bind activation to the exact live extension version/package digest.
  Artifact change, disable, trust revoke, uninstall, or Safe Mode must remove
  effective availability until deliberate reactivation where applicable.
- [x] Record actor-bound audit for activation/order/reset/probe mutations and
  include allowed, denied, stale-revision, and no-mutation tests.
- [x] Return only effectively available providers from the public catalog.
  Host-state lookup failures must fail closed, not silently expose partial
  catalog data.
- [x] Until a real provider probe RPC exists, report probe as unavailable or
  pending without persisting `ok=true`. Never present `probe_pending` as a
  successful health check.

**T1C done (2026-07-27):** Memory + Postgres activation stores use
`FOR UPDATE` / `WHERE revision = expected` + RowsAffected CAS;
`ErrProviderActivationNoMutation` for equivalent state; Host-derived
`PrepareActivationInput` rejects browser ownership and unsupported ops;
`EvaluateOperationAvailability` / public catalog bind live artifact +
RuntimeInstanceID + Safe Mode; activation/order/reset/probe actor-bound audit;
probe persists `ok=false` for `probe_pending`/`probe_unavailable`.

#### Recent Authentication, Unlink, And Password Credentials

- [x] Bind recent authentication to the current session (for example a
  non-reversible SID fingerprint), not only `user_id`. Mark it after successful
  password or external authentication for that session and test cross-session
  isolation.
- [x] Load the target external link before unlink, verify actor ownership,
  active status, and expected revision, and use an idempotency key scoped to
  user/link/revision/request rather than client IP.
- [x] Keep last-login-method enforcement and unlink mutation in a transaction
  or otherwise prove race-safe behavior.
- [x] Keep `user_credentials.password_hash` non-null for rows that exist.
  Represent external-only users by absence of a credential row; revise the
  migration accordingly.
- [x] Implement the M1 password setup endpoint and ensure password reset,
  self-service setup, and authorized admin set-password create a password
  credential when absent. Test external-only setup followed by password login.

**T1D done (2026-07-27):** session-bound recent-auth
(`user_recent_auth` PK `(user_id, session_fingerprint)`); mark after password
and external login; unlink loads link + ownership/active/revision + TX
last-method; idempotency `unlink:{user}:{link}:r{rev}:{requestId}` (no IP);
`password_hash` NOT NULL + external-only = no credential row; upsert on
setup/reset/admin set-password; `POST /auth/password`; authoritative
`ValidateExternalRegister` + field validation before ticket consume; default
role RowsAffected==1; canonical `GetCurrentUser` before session issue.

#### Registration, Session Loading, And Host Policy

- [x] Reuse the authoritative username/email/password-independent registration
  validators, reserved-name rules, events/hooks, human verification, and
  registration-mode policy. Do not maintain a weaker external-registration
  validator.
- [x] Validate editable fields before consuming the opaque ticket, or use a
  reservation/transaction design that permits correction without replaying an
  assertion. Re-check authoritative policy inside the creation transaction.
- [x] Require default-role assignment to affect exactly one expected role, and
  roll back user/link/audit creation otherwise.
- [x] Reload authenticated users through the canonical `CurrentUser` path so
  token version, roles, permissions, avatar, status, and other session claims
  are complete before issuing a session.
- [x] Preserve the zero-user bootstrap rule, non-enumerating email behavior,
  and atomic user/default-role/link/audit boundary.

#### Contracts, Routes, Tests, And Documentation

- [x] Add every new callback, registration, password, link-list/unlink, and
  admin-provider endpoint to modular OpenAPI with security/error schemas.
- [x] Declare reserved Core routes in the Core Route Catalog and document why
  callback/session authority is closed to Route Registry replacement.
- [x] Add controller-level allowed and denied HTTP tests, callback replay and
  exact-artifact tests, real database transition tests, and meaningful
  two-provider tests that execute operations rather than inspect only metadata.
- [x] Cover provider isolation, Safe Mode, disable/trust-revoke/artifact
  upgrade, state/ticket expiry, actor/session mismatch, unauthorized link with
  zero persistence, registration rollback, unlink race, and redaction.
- [x] Correct
  `knowledge/decisions/2026-07-27-github-social-login-m0-contract-freeze.md`
  so Host consistently owns state, PKCE verifier, callback URL, and callback
  transaction; update frozen start/complete schemas with the actual additive
  fields.
- [x] Update identity/extensions knowledge, this task book, the current hot
  handoff, and `knowledge/index.md` with exact test evidence.

**T1E done (2026-07-27):** modular OpenAPI for callback, external-registration,
password, external-identities, admin providers; Core Route Catalog entries for
reserved callback + related Host routes with non-replaceable policy text;
controller HTTP allowed/denied + replay/artifact + redaction tests; model
lifecycle + two-provider operation tests; postgres transition/rollback tests
(skip without `SFORUM_TEST_DATABASE_URL`); M0 ADR Host ownership + additive
schema correction.

**M1R exit:** all M1 exit behavior is demonstrably complete; no unauthorized
callback can persist a link; PKCE works through the exact live artifact;
production identity digests are stable across restarts/instances; activation is
atomic and audited; external registration/password setup reuse authoritative
Host policy; public/admin routes are documented and permission-tested. Only
then may status advance to M2.

### M2 - Built-in GitHub Headless Vertical Slice

#### M2A / T2 - Protocol package (done 2026-07-27)

- [x] Add the built-in package, Manifest V3 declarations, and exact schemas.
- [x] Add Client ID/Client Secret settings and SecretStore use
  (`type: secret` + Host `SFORUM_SETTING_*` injection).
- [x] Implement Authorization Code, PKCE S256, user/email fetch, stable subject,
  timeout, cancellation, and redaction via `golang.org/x/oauth2` + `net/http`.
- [x] Add a truthful bounded probe (credentials present + API root shape; does
  not claim Client Secret proof without an authorization code).
- [x] Test against local fake GitHub endpoints: success
  (login/registration/link assertion fields), token error, PKCE mismatch,
  invalid user/email, subject missing, timeout, malformed response, rate limit,
  email soft-unavailable, and secret redaction.
- [x] Wire `scripts/build-builtin-plugins.sh` for `sforum.auth-github` (staging
  build + digest refresh). Full SyncBuiltins / headless Host E2E remains M2B.

#### M2B / T3 - Packaging and headless E2E (done 2026-07-27)

- [x] Prove `SyncBuiltins` stages exact bytes without Host public activation
  (immutable snapshot digest identity; no activation events; empty public
  catalog without Host activation rows).
- [x] Extend release/container packaging (`apps/api/Dockerfile` + Docker gate
  test for `sforum-auth-github`; `build-builtin-plugins.sh` already wired in
  T2).
- [x] Run headless end-to-end tests through Protocol V2 into Host session
  effects (login CompleteLogin + recent-auth mark; registration opaque ticket
  one-use + continuation path; link CompleteLink with session-bound recent-auth
  and unauthorized zero-write).

**Exit:** GitHub works end to end through API tests and the exact built-in
runtime; no UI completion is claimed. **Met (2026-07-27).**

### M3 - Admin Login Methods Surface

- [x] Add `identity.provider.manage` migration, seed/catalog/i18n, role UI,
  OpenAPI permission notes, and allowed/denied tests.
- [x] Add modular OpenAPI admin paths and schemas.
- [x] Build the Host aggregate over Registry, activation, settings, lifecycle,
  runtime health, and callback URL.
- [x] Add the Login Methods tab and reuse `SFExtensionSettingsRenderer`.
- [x] Implement toggles, callback copy, probe, inline errors, Toasts, optimistic
  revisions, and restore defaults.
- [x] Distinguish discovered, trusted, enabled, configured, probed, and
  publicly activated states.
- [x] Test 403, stale revision, artifact upgrade, Safe Mode, SSR, typecheck,
  desktop, and mobile behavior.

**Exit:** a non-expert operator can configure and deliberately expose GitHub
without visiting raw extension internals. **Met (2026-07-27).**

**T4 / M3 done (2026-07-27):** Host admin aggregate exposes
discovered/trusted/enabled/configured/probed/publiclyActivated + absolute
callbackUrl + settingsPath; `identity.provider.manage` may manage auth-plugin
settings (mail-style delegation); Login Methods page at
`/admin/settings/login-methods` embeds `SFExtensionSettingsRenderer`; operator
role template + i18n + OpenAPI updated; focused Go/web tests + OpenAPI refs +
identity-ui validation pass.

### M4 - Public Auth And Account Security UI

#### M4A - Public login, callback, explicit registration (T5)

- [x] Add an SSR-safe auth-provider composable using the Host catalog.
- [x] Add provider controls to login and registration shells (only when Host
  activates the corresponding operation; no hard-coded executable state).
- [x] Inject vendor presentation (`label` / `icon`) from the plugin Identity
  declaration through the public catalog — Core shell keeps only generic
  templates and Host stable reasons.
- [x] Add safe callback feedback (`ext_auth`) and opaque registration-ticket
  continuation at fixed `/register?ticket=…`.
- [x] Preserve validated return navigation and session-state rules.
- [x] Add Host-shell zh-CN / en-US copy for stable reasons and ticket mode
  (no GitHub brand strings in Core i18n).
- [x] Test password-only, catalog-driven entry, ticket mode, cancellation/
  failure reasons, non-enumerating unlinked copy, responsive provider buttons,
  typecheck.

**M4A exit (2026-07-27):** visitors can start Host-activated external login/
registration, see safe callback feedback, and complete explicit registration
via opaque ticket. Brand label/icon come from the plugin catalog, not Core.

#### M4B - Account security (T6) — done (2026-07-27)

- [x] Add linked accounts, link/unlink, inert state, recent-auth, and password
  setup to account security.
- [x] Test link, unlink blocked, password setup, and related responsive layouts.

**M4B exit (2026-07-27):** account security shows redacted external identities,
Host-gated link entry, unlink with last-method + recent-auth feedback, inert
provider state, and external-only local password setup. Catalog label/icon only;
no Core GitHub brand strings. **Full M4 exit complete.**

### M5 - Lifecycle, Security, Documentation, And Release Gate

- [x] Test restart, disable, uninstall, staged upgrade, new-digest activation,
  rollback, trust revoke, Safe Mode, ForceDrain, and callback during change.
- [x] Test replay, expiry, cross-provider/operation/actor, duplicate
  registration, subject races, activation CAS, and unlink races.
- [x] Verify logs, audits, APIs, browser history, and diagnostics contain no
  code, token, secret, verifier, raw state, raw subject, digest, or upstream
  error body.
- [x] Verify start/callback rate limits and non-enumerating public errors.
- [x] Update the Identity Extension Surface Matrix and document intentionally
  closed callback/session surfaces.
- [x] Add bilingual operator setup, troubleshooting, and author docs.
- [x] Update identity/extensions knowledge, plan/index, and hot handoff.
- [ ] Run focused tests, OpenAPI validation, full repo gate, and desktop/mobile
  Browser QA against the user-owned web server.

**Exit:** GitHub V1 is secure, lifecycle-tested, documented, and product-usable.
**Met (2026-07-27 / T7):** `external_auth_m5_*` lifecycle/security/rate-limit/
redaction tests; Host start/callback IP rate limits; Identity Extension Surface
Matrix + bilingual operator/author docs; knowledge/plan/index/handoff updated.
The M5 handoff reported full-gate and Browser QA evidence, but independent
review did not reproduce a green gate and found no reproducible Browser QA
record. T8D owns closure of this item.

**Independent review (2026-07-27): not accepted.** The T7 implementation report
overstated closure. T8A–T8C remediated registration fencing, truthful probe +
generic admin discovery, production endpoint boundary, atomic Redis rate-limit
TTL, auth-start fail-closed wiring, and migration 057 password_hash safety.
T8D still owns real frontend interaction coverage, rate-limit HTTP assertion
strength, exact runtime Browser QA, and the independent re-review matrix. See
T8D and `sessions/2026-07-27-github-social-login-final-review-handoff.md`.

## Stable Error Reasons

Reuse existing reasons when semantics match. Add only the minimum missing set:

- `auth.provider_not_found`
- `auth.provider_unavailable`
- `auth.provider_not_enabled`
- `auth.provider_callback_expired`
- `auth.provider_callback_invalid`
- `auth.provider_callback_replayed`
- `auth.provider_cancelled`
- `auth.external_identity_unlinked`
- `auth.external_subject_conflict`
- `auth.external_link_conflict`
- `auth.external_registration_ticket_invalid`
- `auth.external_registration_ticket_expired`
- `auth.last_login_method_required`
- existing `auth.registration_disabled`
- existing risk/session policy denial reasons

## Required Verification Matrix

| Scenario | Required result |
| --- | --- |
| Password-only site | Existing login/register behavior unchanged |
| Built-in discovered | Visible to admin; not trusted/enabled/activated |
| Plugin configured/enabled | No public button until Host activation |
| Known active link | Core issues normal Redis session after policies |
| Unknown link | Generic unlinked result; no account creation |
| Explicit registration | Local fields + ticket; user/role/link atomic |
| Email matches existing user | No automatic link, login, or disclosure |
| Zero-user site | GitHub registration denied; Core bootstrap works |
| Disabled/banned user | No session; generic public error |
| Replayed callback/ticket | Rejected with no duplicate effect |
| Artifact changes mid-flow | Attempt fails closed |
| Plugin disabled/uninstalled | Button disappears; link retained inert |
| Safe Mode | GitHub unavailable; Core password/recovery works |
| Final method unlink | Blocked until another method exists |
| External-only password setup | Credential created; password login works |
| Admin lacks permission | 403 and no provider mutation |
| Restore defaults | Operations off; secrets preserved and stated |

## Required Commands

```bash
cd apps/api && go test ./...
ruby scripts/validate-openapi-refs.rb
cd apps/web && bun run typecheck
cd apps/web && bun run build
./scripts/test.sh
```

Provider tests use checked-in local fake servers and never call live GitHub.
Apply the repository proxy environment before dependency downloads.

## Delivery Rules

1. Keep milestones reviewable and preserve unrelated dirty files.
2. Inspect existing types, routes, schemas, and lifecycle code before adding an
   interface.
3. Add allowed and denied tests with every unsafe route.
4. Report files, migrations, contracts, permissions, security impact, and exact
   command results per milestone.
5. Stop for a product decision if implementation would violate a frozen rule.
6. Do not claim built-in runtime completion from source tests alone: rebuild,
   restart, stage, confirm/enable, activate, and verify the exact provider
   artifact through runtime APIs.

## Conversation-sized Delivery Protocol

After the M1 review, remaining work is divided into independent conversations.
Do not combine adjacent tasks, and do not continue automatically into the next
task.

| Task | Scope | May start when |
| --- | --- | --- |
| T1A | M1R callback authorization ordering, legacy bypass removal, live artifact recheck, PKCE and trusted callback/redirect | **done** (2026-07-27) |
| T1B | M1R stable HMAC config, concurrent/expiring hashed state and ticket stores | **done** (2026-07-27) |
| T1C | M1R atomic audited activation, effective catalog, Safe Mode and truthful probe | **done** (2026-07-27) |
| T1D | M1R session-bound recent-auth, unlink/password credentials, authoritative registration and canonical session user | **done** (2026-07-27) |
| T1E | M1R OpenAPI/Core Route Catalog, HTTP/database/two-provider/lifecycle tests, M1 re-review report | **done** (2026-07-27) |
| T2 | M2A GitHub protocol package, manifest/schemas, fake GitHub server | **done** (2026-07-27) |
| T3 | M2B built-in packaging, exact-artifact runtime staging and headless E2E | **done** (2026-07-27) |
| T4 | M3 admin API contract and Login Methods UI | **done** (2026-07-27) |
| T5 | M4A public login, callback, and explicit registration UI | **done** (2026-07-27) |
| T6 | M4B linked accounts, session-bound recent-auth, unlink, password setup UI | **done** (2026-07-27) |
| T7 | M5 lifecycle/security matrix, docs, release gate, final report | **done** (2026-07-27; closure rejected by independent review) |
| T8A | Registration commit authorization, transaction policy, event/audit correctness | **done** (2026-07-27) |
| T8B | Real provider probe and generic discovered/admin presentation pipeline | **done** (2026-07-27) |
| T8C | Production/runtime hardening and migration correction | **done** (2026-07-27) |
| T8D | Real persistence/controller tests, exact runtime Browser QA, final re-review report | **active** (evidence prepared; independent review pending) |

### T8A - Registration Commit Authorization And Transaction Correctness

- [x] Before any registration effect, require the `registration` operation to
  remain Host-activated and compare the assertion/ticket owner extension,
  extension version where present, package digest, and provider contract with
  the live Registry contribution. Artifact drift, disable, trust revoke,
  uninstall, Safe Mode, and operation-off must fail closed.
- [x] Re-check the authoritative registration policy inside the same database
  transaction that creates the user/default role/external link. Do not rely on
  the pre-transaction fast check.
- [x] Emit the normal `user.registered` observe event exactly once after a
  successful commit, with the same bounded public semantics as password
  registration. Persist the required external-registration audit through the
  transaction-owned path; a later session audit is not a substitute for the
  registration mutation audit.
- [x] Add focused PostgreSQL tests that call `CompleteRegistration` itself and
  prove ticket-after-disable, artifact upgrade, policy-close race, rollback,
  event once-only, and audit behavior. Tests must prove zero user/link/role
  effects on every denied path.

**Exit:** a valid callback assertion cannot be converted into an account after
its exact authorization or registration policy ceases to be valid.
**T8A done (2026-07-27):** `CompleteRegistration` re-checks Host activation +
live Registry exact contribution before any user/role/link write; re-reads
registration policy inside the creation TX; writes
`auth.external_register.success` on the same TX as link audit; emits
`user.registered` once after commit. Focused PG tests under
`external_auth_t8a_registration_postgres_test.go`.

### T8B - Truthful Probe And Generic Admin Provider Pipeline

- [x] Define and wire a bounded versioned provider-probe runtime operation;
  invoke the GitHub `Probe` implementation with deadlines and redacted stable
  reasons. `probe_pending` is not an implementation of the product probe.
- [x] Build the admin provider directory from Host extension/package catalog
  plus live Registry state so a protected built-in can be shown as discovered
  before executable enable, and remains inspectable when disabled or drifted.
  Keep trust/enable authority in the existing extension lifecycle.
- [x] Return localized label/icon presentation metadata from the Host catalog
  and remove every `github` ID/owner substring branch from Core admin UI.
- [x] Add allowed/denied API tests and rendered interaction tests for
  discovered, trusted, enabled, configured, probed, activated, disabled,
  drifted, Safe Mode, reset, and a second fake provider.

**Exit:** a non-expert operator can inspect and configure any conforming auth
provider without Core vendor branches, and the probe truthfully exercises the
selected exact runtime.
**T8B done (2026-07-27):** versioned `auth:provider.probe` operation (manifest
schema + Identity Registry + identity.runtime@1); GitHub plugin wires
`GitHubOAuth.Probe` with deadline and redacted reason/message; Host
`AuthProviderFlow.Probe` + admin probe persists real ok/reason (never product
`probe_pending`); admin directory merges package catalog discovery with live
Registry executable authority + activation; admin UI consumes Host `label`/
`icon` only. Focused tests under Identity model, Identity controllers, and
`apps/web/tests/adminLoginMethods.test.ts` (happy-dom rendered rows).

### T8C - Production And Runtime Hardening

- [x] Reject or ignore fake-GitHub endpoint overrides in production. Test
  endpoints must be injected only under an explicit test/development boundary;
  production OAuth code, Client Secret, and token must only reach fixed
  GitHub.com endpoints in V1.
- [x] Make Redis `INCR` plus TTL establishment atomic for start/callback rate
  limits, or otherwise guarantee an `EXPIRE` failure cannot leave a permanent
  lockout key. Preserve the documented fail-open dependency policy.
- [x] Make auth start fail closed unless the complete external-auth stack,
  activation service, and callback transaction store are present.
- [x] Correct migration `057`: it must not redundantly change
  `user_credentials.password_hash`, and its Down path must preserve the
  `NOT NULL` invariant established by migration `055`.
- [x] Add focused regression tests for production config, Redis expiry failure,
  partial bootstrap wiring, and migration up/down invariants.

**Exit:** production cannot redirect secrets to test endpoints, infrastructure
errors cannot permanently lock out an address, and partial wiring cannot start
an unusable OAuth flow.
**T8C done (2026-07-27):** Host `buildPluginProcessEnv` strips
`SFORUM_AUTH_GITHUB_*_URL` when `APP_ENV=production`; plugin
`LoadGitHubConfigFromEnv` ignores overrides in production (defense in depth).
Redis external-auth rate limit uses atomic Lua `INCR`+`PEXPIRE` (heals no-TTL
keys; fail-open + Del on script error). Auth start requires complete
external-auth wiring (`externalAuthService`, `callbackStateStore`,
`activationStore`, `providerCatalog`) before OAuth. Migration 057 no longer
touches `password_hash` / `user_credentials`. Focused tests under Identity
rate-limit, Identity controllers, Extensions protocol env, migrations, and
GitHub plugin backend.

### T8D - Release Evidence And Independent Re-review

- [x] Replace source-text `toContain` checks for the critical admin/public/
  account flows with rendered component or browser interaction coverage.
- [x] Strengthen rate-limit HTTP tests so an expected `429` is an assertion,
  not a log-only branch.
- [x] Run focused Go/PostgreSQL/controller/plugin tests, OpenAPI validation,
  Nuxt tests/typecheck/build, and `./scripts/test.sh`; report every failure
  honestly and distinguish unrelated environmental failures.
- [x] Rebuild the GitHub built-in, restart API, stage/confirm/enable the exact
  digest through normal lifecycle APIs, configure via the normal admin path,
  activate deliberately, and record exact runtime API evidence.
- [x] Perform desktop/mobile Browser QA for password fallback, admin lifecycle,
  login, explicit registration, link/unlink/password setup, callback cleanup,
  disabled/drifted/Safe Mode states, and secret/history redaction. Record URLs,
  viewport sizes, artifact digest, commands, and screenshots or equivalent
  reproducible evidence.
- [x] Update all knowledge/docs claims and produce a requirements matrix for a
  fresh Codex independent review. Do not self-declare the program closed.

**Exit:** all Definition of Done items have executable or reproducible runtime
evidence and are ready for independent acceptance.

**T8D evidence prepared (2026-07-27):** source-text frontend checks were
replaced with interaction regressions and retained Playwright QA; rate-limit
HTTP now hard-asserts 429; focused backend/plugin/PostgreSQL, OpenAPI, Nuxt,
and final repository gate ran. The exact built-in package digest was
`5d73651dd4013bc04abeb6f99f9ef0686303ee683c40d9a483f927f6b5c09942`.
Migration 058 safely resets only an evidence-free pre-T8D `enabled` GitHub
built-in to installed: it requires no current active durable root, no exact
successful lifecycle activation, and no `extension.enable` audit evidence.
Partial/damaged durable publication alone never proves an operator state is
stale. Evidence matrix:
`reports/2026-07-27-github-social-login-t8d-requirements-matrix.md`.
This does not close the program: independent review must examine the retained
Browser QA and explicitly decide the remaining artifact-drift screenshot gap.

Every task conversation must:

1. Read `AGENTS.md`, `knowledge/index.md`, identity/extensions module notes,
   this task book, the relevant decisions, and the current GitHub-login hot
   handoff before editing.
2. Work only on its named task and preserve unrelated dirty work.
3. Run focused verification plus every applicable repository contract check.
   A green pre-existing test suite is not evidence for untested exit criteria.
4. Update the checkboxes/status in this task book, `knowledge/plans/README.md`,
   `knowledge/modules/identity.md`, `knowledge/modules/extensions.md`, the
   single current GitHub-login hot handoff, and `knowledge/index.md`.
5. Stop and output a small report containing: completed scope, important
   decisions, files/contracts/migrations changed, security impact, exact
   commands and results, remaining risks, and the next task id.
6. End with a ready-to-copy prompt for a fresh Grok conversation for the next
   task. The prompt must forbid starting later tasks.
7. Never commit, push, or discard unrelated changes unless the user explicitly
   requests it.

T7 produced the first final implementation report; independent review rejected
its closure. T8D must produce the replacement requirements matrix and exact
runtime evidence for another Codex review. Completion must not be self-declared
solely from green tests.

## Definition Of Done

- One protected built-in GitHub plugin ships and is boot-discovered.
- Core contains no GitHub vendor behavior or tokens.
- GitHub login, explicit registration, link, unlink, and external-only password
  setup work end to end.
- Subject identity uses a truthful Host-keyed digest and neither form reaches
  public responses.
- Public activation is explicit, exact-artifact bound, audited, and default-off.
- Core password bootstrap, login, recovery, and Safe Mode remain usable.
- Admin and account surfaces are beginner-friendly and permission-aware.
- Replay, account-takeover, lifecycle, race, and redaction tests pass.
- OpenAPI, bilingual docs/UI, knowledge base, and Extension Surface Matrix are
  current.
- Full repo gate and Browser QA pass with exact runtime evidence.

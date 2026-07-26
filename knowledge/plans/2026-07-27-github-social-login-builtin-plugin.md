# Built-in GitHub Social Login V1 - Task Book

Status: **ready** - approved product scope and implementation checklist; no
production GitHub login is shipped yet

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

- [ ] Inspect current routes, provider flow, typed schemas, external-link store,
  credential model, session issue, risk/session evaluators, built-in lifecycle,
  SecretStore, and settings renderer.
- [ ] Verify official current GitHub OAuth App behavior and record sources/date.
- [ ] Complete the `x/oauth2` versus `goth` versus direct-HTTP survey.
- [ ] Freeze typed start/complete schemas with transient raw subject.
- [ ] Freeze HMAC config, production validation, backup rule, and data
  compatibility.
- [ ] Freeze callback, registration-ticket, public/admin response schemas, and
  stable error reasons.
- [ ] Update this plan or add an ADR if code evidence disproves an assumption.

**Exit:** reviewed implementation map, dependency decision, official protocol
evidence, and frozen contracts with no production behavior change.

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

### M2 - Built-in GitHub Headless Vertical Slice

- [ ] Add the built-in package, Manifest V3 declarations, and exact schemas.
- [ ] Add Client ID/Client Secret settings and SecretStore use.
- [ ] Implement Authorization Code, PKCE, user/email fetch, stable subject,
  timeout, cancellation, and redaction.
- [ ] Add a truthful bounded probe.
- [ ] Extend `scripts/build-builtin-plugins.sh` and release/container packaging.
- [ ] Prove `SyncBuiltins` stages exact bytes without trusting, enabling, or
  activating them.
- [ ] Test against local fake GitHub endpoints: success, denial, token error,
  invalid user/email, subject drift, timeout, malformed response, rate limit,
  and secret redaction.
- [ ] Run headless end-to-end tests through Protocol V2 into a Host session.

**Exit:** GitHub works end to end through API tests and the exact built-in
runtime; no UI completion is claimed.

### M3 - Admin Login Methods Surface

- [ ] Add `identity.provider.manage` migration, seed/catalog/i18n, role UI,
  OpenAPI permission notes, and allowed/denied tests.
- [ ] Add modular OpenAPI admin paths and schemas.
- [ ] Build the Host aggregate over Registry, activation, settings, lifecycle,
  runtime health, and callback URL.
- [ ] Add the Login Methods tab and reuse `SFExtensionSettingsRenderer`.
- [ ] Implement toggles, callback copy, probe, inline errors, Toasts, optimistic
  revisions, and restore defaults.
- [ ] Distinguish discovered, trusted, enabled, configured, probed, and
  publicly activated states.
- [ ] Test 403, stale revision, artifact upgrade, Safe Mode, SSR, typecheck,
  desktop, and mobile behavior.

**Exit:** a non-expert operator can configure and deliberately expose GitHub
without visiting raw extension internals.

### M4 - Public Auth And Account Security UI

- [ ] Add an SSR-safe auth-provider composable using the Host catalog.
- [ ] Add GitHub controls to login and registration shells.
- [ ] Add safe callback feedback and opaque registration-ticket continuation.
- [ ] Preserve validated return navigation and session-state rules.
- [ ] Add linked accounts, link/unlink, inert state, recent-auth, and password
  setup to account security.
- [ ] Add zh-CN and en-US copy for all states and stable reasons.
- [ ] Test password-only, linked/unlinked login, explicit registration,
  cancellation, provider failure, duplicate email hint, link, unlink blocked,
  password setup, and responsive layouts.

**Exit:** GitHub is product-usable from browser entry through account security.

### M5 - Lifecycle, Security, Documentation, And Release Gate

- [ ] Test restart, disable, uninstall, staged upgrade, new-digest activation,
  rollback, trust revoke, Safe Mode, ForceDrain, and callback during change.
- [ ] Test replay, expiry, cross-provider/operation/actor, duplicate
  registration, subject races, activation CAS, and unlink races.
- [ ] Verify logs, audits, APIs, browser history, and diagnostics contain no
  code, token, secret, verifier, raw state, raw subject, digest, or upstream
  error body.
- [ ] Verify start/callback rate limits and non-enumerating public errors.
- [ ] Update the Identity Extension Surface Matrix and document intentionally
  closed callback/session surfaces.
- [ ] Add bilingual operator setup, troubleshooting, and author docs.
- [ ] Update identity/extensions knowledge, plan/index, and hot handoff.
- [ ] Run focused tests, OpenAPI validation, full repo gate, and desktop/mobile
  Browser QA against the user-owned web server.

**Exit:** GitHub V1 is secure, lifecycle-tested, documented, and product-usable.

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

# Social Login Provider Plugins — Task Book

Status: **superseded** — replaced by the focused built-in GitHub V1 task book
`knowledge/plans/2026-07-27-github-social-login-builtin-plugin.md`

Date: 2026-07-22
Goal: ship secure GitHub, Google, Discord, and Telegram login through trusted
plugins while Core owns accounts, links, risk decisions, and browser sessions.

This task book is intended to be implemented milestone by milestone. Do not
collapse it into one large patch. Each milestone must leave the repository
buildable and must report the exact tests that passed.

## Required Reading Before Coding

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/identity.md`
4. `knowledge/modules/extensions.md`
5. `knowledge/decisions/2026-07-19-identity-provider-automation-authority.md`
6. `knowledge/decisions/2026-07-13-login-risk-step-up-without-account-lock.md`
7. `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
8. `docs/extensions/authoring-guide.md`
9. This task book

## Product Outcome

An operator can install and configure provider plugins, explicitly expose one
or more login methods, and see them on the Host login/register surfaces. A user
can log in, register, link, inspect, and safely unlink external accounts without
any plugin receiving Core passwords, raw browser sessions, or permission
authority.

The first provider set is:

- GitHub OAuth 2.0
- Google OpenID Connect
- Discord OAuth 2.0
- Telegram through its supported login protocol, isolated behind a dedicated
  adapter rather than forced into assumptions made for GitHub/Google

## Current Baseline — Reuse, Do Not Rebuild

| Area | Current evidence | Required treatment |
| --- | --- | --- |
| Identity Registry | `app/Support/IdentityRegistry` | Extend existing multi-provider catalog |
| Executable operations | `registration/login/link.start|complete` | Preserve names and failure policy |
| Provider HTTP routes | `Controllers/Identity/auth_providers.go` | Complete product effects; do not add a parallel auth stack |
| External links | `identity_external_links` + `external_links*.go` | Reuse uniqueness, audit, lifecycle, and erase semantics |
| Protocol V2 transport | `identity.runtime@1` | Reuse exact-artifact invocation and Schema checks |
| Session/risk policies | `SessionPolicyEvaluator`, `RiskEvaluator` | Run before every external login session issue |
| Browser sessions | `Support/AuthSession` | Remain the only first-party browser session authority |
| Plugin settings | Manifest Schema + Host renderer | Reuse for provider credentials and options |
| Secrets | production `SecretStore` | No raw Client Secret/Bot Token in Core options or logs |
| Safe Mode | Host-owned Core auth fallback | Third-party login is unavailable; Core password path survives |
| Public UI | `SFLoginFormPage.vue`, register/auth shell | Add Host-owned provider controls |
| Admin settings | `/admin/settings` tabbed settings | Add product-facing Login Methods tab |

The existing public `complete` path currently returns an external assertion but
does not perform the full Host-owned login/registration session effect. That is
a known gap, not permission to move session creation into plugins.

## Frozen Architecture Decisions

These are implementation law for this task book. Change them only through an
explicit product/architecture decision, not ad hoc during coding.

### A. Multi-provider registry, not a singleton slot

- GitHub, Google, Discord, and Telegram may be enabled simultaneously.
- Auth providers remain Identity Registry contributions. Do not model social
  login as one selected generic Provider Slot.
- A user's explicit provider choice is exact for that attempt. Failure never
  falls through to another provider or silently succeeds through Core password
  login.

### B. Core owns every identity effect

Plugins may verify an external provider and return a bounded subject assertion.
Core alone owns:

- user creation and local username/email invariants;
- default role and first-user rules;
- external-link uniqueness and lifecycle;
- account status and permission checks;
- risk and selected session-policy evaluation;
- browser session issue/renew/revoke;
- audit, rate limits, redirect validation, and final error mapping.

Plugins must never create users, assign roles, set permissions, mint SForum
cookies, receive raw SForum cookies, or decide that an actor is authorized.

### C. External account matching

- Stable provider id + provider subject is the identity key. Usernames and
  mutable handles are display data only.
- Never link or log in by email match alone, including a provider-verified
  email.
- An unlinked `login.complete` returns a stable unlinked result. It does not
  silently create an account.
- Registration is a separate explicit operation and must create the user plus
  external link in one transaction.
- Linking requires a live logged-in actor and a fresh provider completion.
- Provider email may prefill a registration field, but it does not bypass
  SForum email verification in v1.

### D. Bootstrap and usable login methods

- The first `super_admin` must be created through the existing Core bootstrap
  registration. External registration is unavailable while the site has zero
  users.
- Core password login remains enabled by default.
- Admin UI must not allow the last usable login method for the current
  `super_admin` to be removed accidentally.
- An external-only account is supported. Do not generate a fake known password.
  The credential store and password-reset/admin-set-password paths must support
  adding a password credential later.
- A user cannot unlink their final usable login method until another external
  link or a valid password credential exists.

### E. Explicit Host activation

- Installing, configuring, trusting, or enabling a plugin does not
  automatically expose its login button.
- Core owns a durable, revisioned, audited activation record per provider and
  operation (`login`, `registration`, `link`; recovery remains separate).
- Recommended defaults are all external methods off.
- Effective availability requires an exact active executable provider,
  compatible operations, valid Host activation, and non-Safe-Mode runtime.
- Disable/uninstall/trust revocation removes effective availability immediately
  while retaining external links as inert data.
- Executable upgrades must not silently transfer an activation to unconfirmed
  package bytes. Bind activation evidence to the exact artifact and require a
  deliberate post-upgrade confirmation when the artifact changes.

### F. Host-owned callback and state

- OAuth/OIDC callback paths are reserved Core routes and are not replaceable by
  Route Registry or themes.
- Core mints high-entropy state and correlation ids, stores only bounded
  short-lived transaction data in shared Redis, and atomically consumes state.
- Default callback transaction TTL: 10 minutes.
- Exact provider id, operation, actor (for link), validated local return path,
  artifact identity, state digest, PKCE material, and creation time are bound in
  the transaction.
- Reject missing, expired, reused, provider-mismatched, operation-mismatched,
  actor-mismatched, and artifact-mismatched callbacks.
- Redirect-after-login accepts only validated local absolute paths. Never trust
  plugin-returned or request-supplied external redirect targets.
- Authorization code, access token, refresh token, ID token, Bot Token, raw
  state, PKCE verifier, and provider error payloads must not appear in logs,
  audit metadata, URLs rendered after completion, or public API responses.

### G. Protocol libraries and vendor isolation

Perform a short dependency check before adding packages. Preferred baseline:

- `golang.org/x/oauth2` for OAuth 2.0 mechanics;
- `github.com/coreos/go-oidc/v3/oidc` for OIDC discovery and ID-token checks;
- provider-specific code only for documented vendor differences.

Shared protocol helpers belong in the plugin SDK or a focused reusable module,
not in Core vendor business logic. Do not adopt a library that takes ownership
of SForum sessions, routing, or provider registration. Apply the repository
proxy environment before every network dependency command.

## UX Contract

### Admin: Login Methods

Add a `loginProviders`/“登录方式” tab to `/admin/settings`, adjacent to account
security and registration. This is the primary operator surface; the generic
extension page remains an inspection and lifecycle surface.

Display every discovered or installed auth provider as one compact row/card
with:

- localized provider label and approved Tabler brand icon;
- extension name/version and exact-artifact status;
- state: not installed, needs configuration, ready, enabled, unavailable, or
  unhealthy;
- separate toggles for login, registration, and account linking, shown only
  when the provider declares the matching operations;
- callback URL with copy action;
- configure action using the existing extension settings Schema renderer;
- connection/probe action with success Toast and inline persistent error;
- ordering controls for public buttons;
- last probe time and redacted reason when unavailable;
- restore recommended defaults action: turn external methods off, reset order,
  and preserve secrets with an explicit message.

No raw secret value may be returned to the browser. Save, enable, disable,
probe, copy, and restore actions follow the repository Toast rules.

Add a distinct `identity.provider.manage` permission because login-provider
control is an identity/security capability. Update migration/seed data,
permission catalog translations, OpenAPI permission notes, allowed/denied
tests, and role UI. Exact executable trust remains `super_admin` controlled.

### Public login and registration

- `GET /auth/providers` returns only effectively enabled public operations,
  with localized label, allowed icon identifier, order, and operation list.
- Login and registration pages render provider buttons from that Host response;
  themes may style the Host component but cannot supply executable callback
  behavior.
- Use Tabler/Nuxt Icon (`brand-github`, `brand-google`, `brand-discord`,
  `brand-telegram`); no emoji or handwritten SVG.
- Separate the external methods from the password form with accessible text.
- A provider outage affects only that provider. Keep the password form usable.
- Preserve the existing validated local auth return path.
- Success updates `useAuthSession`, shows a theme-aware success Toast for 10
  seconds, and uses replace-style return navigation.
- Blocking callback and account-link errors remain visible until resolved or
  dismissed.

### User account security

Extend the existing account security surface with “已关联账号”:

- provider label/icon, linked time, and status;
- link another account;
- unlink with explicit confirmation and recent-auth/step-up enforcement;
- explain when unlink is blocked because it is the last login method;
- do not display raw external subject, tokens, scopes, or provider secrets;
- provider disable shows an unavailable/inert state rather than deleting data.

## API And Data Contract Target

Names may be adjusted to existing controller conventions, but behavior must not
be weakened.

### Public/self-service routes

| Method | Suggested path | Access | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/auth/providers` | public | Effective localized provider catalog |
| `POST` | `/api/v1/auth/providers/{providerId}/{operation}/start` | operation-specific | Existing start flow, now Host-state bound |
| `GET/POST` | `/api/v1/auth/providers/{providerId}/callback` | public callback | Reserved OAuth/OIDC callback; consumes Host state |
| `POST` | `/api/v1/auth/providers/{providerId}/{operation}/complete` | operation-specific | Keep for non-redirect/challenge completion if needed |
| `GET` | `/api/v1/auth/external-identities` | login | List current user's redacted links |
| `DELETE` | `/api/v1/auth/external-identities/{linkId}` | login + step-up | Unlink while preserving last-method invariant |
| `POST` | `/api/v1/auth/password` | login + step-up | Add/change a local password credential safely |

Do not expose an endpoint that lets a browser submit an arbitrary user id,
artifact id, subject digest, callback URL, or role assignment.

### Admin routes

| Method | Suggested path | Permission | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/admin/identity/providers` | `identity.provider.manage` | Catalog + activation + health + callback metadata |
| `PATCH` | `/api/v1/admin/identity/providers/{providerId}` | same | CAS update operation toggles/order |
| `POST` | `/api/v1/admin/identity/providers/{providerId}/probe` | same | Bounded exact-provider health/config test |
| `POST` | `/api/v1/admin/identity/providers/reset` | same | Recommended defaults; secrets preserved |

Use optimistic revision checks. Every mutation must audit actor, provider id,
exact artifact, old/new non-secret state, reason, and result.

### Data work

Add the smallest cohesive Host-owned persistence needed for:

1. provider activation/order/exact-artifact evidence and revision;
2. short-lived callback transactions in shared Redis with atomic consume;
3. optional password credentials for external-only users;
4. transactional external registration (user + role + link + audit boundary);
5. recent-auth/step-up evidence for sensitive link/unlink operations if the
   current session model cannot express it safely.

Migrations must preserve existing users and credentials. Do not weaken the
`identity_external_links` uniqueness or audit protections. Migration down paths
must not erase durable security evidence silently.

## Provider Manifest/SDK Contract

Extend the existing Identity provider declaration only where the Host UI needs
stable, non-executable metadata:

- localized label;
- approved icon id;
- optional short localized description;
- supported operations already remain authoritative;
- probe/settings linkage through existing extension contracts.

Do not accept arbitrary HTML, JavaScript, remote icon URLs, callback routes, or
CSS from the provider declaration for the Core auth buttons.

SDK helpers should provide:

- OAuth/OIDC start URL construction with Host state/callback/PKCE input;
- callback error normalization;
- token exchange with bounded timeouts;
- OIDC issuer/audience/nonce/time validation;
- stable subject-digest helper compatible with the Host contract;
- redaction helpers that make token/error logging difficult;
- provider fixture helpers for tests using local `httptest` servers.

Core stores no provider access or refresh token. A login-only plugin should not
persist them after identity proof unless the provider protocol strictly
requires a documented lifecycle. Any retained provider credential belongs to
the plugin's own schema/SecretStore and privacy lifecycle.

---

## Milestone M0 — Contract Audit And Freeze

- [ ] Re-read current provider routes, flow, schemas, link store, session issue,
  risk/session policies, Manifest validation, and Protocol V2 SDK.
- [ ] Write a short implementation map in the first PR description identifying
  exact reused types and required additive changes.
- [ ] Confirm official current provider requirements before coding: endpoints,
  scopes, callback restrictions, stable subject claim, PKCE/OIDC support, and
  token-validation rules. Record URLs and check date in provider README files.
- [ ] Compare OAuth/OIDC libraries for maintenance, license, documentation,
  ecosystem fit, and session ownership; record the chosen helper in the plan or
  a decision note if it changes architecture.
- [ ] Freeze public/admin response schemas and stable error reasons before UI
  implementation.
- [ ] Update this task book if code evidence disproves an assumption; do not
  silently invent a parallel contract.

**Exit:** reviewed design map, dependency decision, route/schema draft, and no
production behavior change.

## Milestone M1 — Core State, Activation, And Final Effects

### M1.1 Activation catalog

- [ ] Add durable exact-artifact provider activation/order records with CAS and
  audit.
- [ ] Install/enable alone leaves all external operations off.
- [ ] Resolve effective providers against the live Identity Registry and Safe
  Mode on every relevant read/mutation.
- [ ] Invalidate effective activation on lifecycle/trust/artifact mismatch.
- [ ] Filter `GET /auth/providers` to effective providers and return safe
  presentation metadata.

### M1.2 Callback transaction

- [ ] Add shared-Redis callback transaction store with high-entropy state,
  digest-at-rest where practical, 10-minute TTL, and atomic one-use consume.
- [ ] Bind provider, operation, actor, exact artifact, PKCE, and safe local
  return target.
- [ ] Add Host callback route(s) for query and `form_post` only if a selected
  provider requires both.
- [ ] Map provider denial/cancel separately from invalid/replayed callback and
  transient provider failure without leaking upstream payloads.
- [ ] Keep callback route outside plugin/theme replacement authority.

### M1.3 Login/register/link effect

- [ ] External login resolves only an active link, reloads current user status,
  runs risk/session policies, issues the normal browser session, records the
  session directory, and writes login audit.
- [ ] Disabled/banned/missing users fail closed with generic public messaging.
- [ ] External registration enforces registration mode and zero-user bootstrap
  rule, validates local fields, creates user + default role + external link in
  one transaction, then evaluates and issues a session.
- [ ] Link completion remains actor-bound and exact-provider fenced.
- [ ] Add external-only user support without fake passwords; make password
  reset/admin set-password create a credential when absent.
- [ ] Add a self-service password setup/change path. Existing password users
  confirm the current password; external-only users complete recent external
  provider step-up. Apply current password policy, rotate token/session
  authority as appropriate, and never accept actor/user id from the browser.
- [ ] Add stable errors for unlinked identity, subject conflict, registration
  disabled, callback expired/replayed, provider unavailable, account disabled,
  and last-login-method protection.

### M1.4 Tests

- [ ] Allowed and denied controller/service tests for every unsafe route.
- [ ] State replay, cross-provider, cross-operation, cross-actor, expired,
  artifact-change, Safe Mode, disable, and concurrent-completion tests.
- [ ] Transaction rollback proves no orphan user or link when any registration
  step fails.
- [ ] Session issue tests prove risk/session policy denial remains authoritative.
- [ ] Existing password registration/login remains unchanged.

**Exit:** a fixture auth provider can complete real Host login, registration,
and link flows; no vendor plugin or polished UI required yet.

## Milestone M2 — Admin Management Product Surface

- [ ] Add `identity.provider.manage` seed/migration/catalog/i18n and protected
  initial role mapping consistent with existing permission policy.
- [ ] Add modular OpenAPI admin paths and schemas.
- [ ] Build admin aggregate service over Identity Registry, activation store,
  extension settings metadata, runtime health, and callback URL resolution.
- [ ] Add Login Methods tab to `/admin/settings`; keep the file cohesive by
  extracting focused components/composables rather than growing the page into a
  monolith.
- [ ] Reuse `SFExtensionSettingsRenderer` for provider configuration; do not
  duplicate a vendor-specific form in Core.
- [ ] Implement toggles, ordering, callback copy, probe, inline blocking errors,
  Toasts, and restore defaults with secrets preserved.
- [ ] Require a successful config/probe state before first activation when the
  plugin declares required credentials.
- [ ] Add 403 tests, super-admin trust boundary tests, stale-revision 409 tests,
  SSR/type tests, and mobile/desktop UI checks.

**Exit:** a non-expert operator can configure and deliberately expose a fixture
provider from the unified page without visiting raw extension internals.

## Milestone M3 — Public Auth And Linked Accounts UI

- [ ] Add a focused `useAuthProviders` composable with SSR-safe provider list,
  start, callback result, and error handling.
- [ ] Add Host-owned provider button group to login and registration pages.
- [ ] Add callback page/shell only if needed for completion feedback; it must
  not expose provider code/state/token in client-visible state.
- [ ] Preserve `redirect` through start/callback using only the existing local
  return-path validator.
- [ ] On success update `useAuthSession`, Toast, and replace navigation.
- [ ] Add linked-account list/link/unlink UI to account security.
- [ ] Show whether a local password exists and provide the self-service
  setup/change flow needed before unlinking a final external method.
- [ ] Enforce and explain last-login-method and recent-auth requirements.
- [ ] Add zh-CN and en-US strings for all states and stable errors.
- [ ] Add frontend unit tests plus browser tests for password-only, multi-
  provider, cancellation, provider failure, link, unlink blocked, and responsive
  layouts.

**Exit:** fixture provider flow is complete from browser UI through Core session
and account management.

## Milestone M4 — Shared SDK And GitHub Reference Plugin

- [ ] Add the smallest reusable OAuth helper to the plugin SDK after M0 library
  review. Keep vendor endpoints out of Core.
- [ ] Create an optional or protected plugin in the correct `extensions/` tier;
  document why that tier is chosen.
- [ ] Manifest declares exact auth operations, localized identity, Tabler brand
  icon, settings Schema, required secrets, and probe action.
- [ ] GitHub start uses Host callback/state and PKCE where supported by the
  selected current GitHub application flow.
- [ ] Complete exchanges the code, fetches the authenticated user and email as
  required, and keys identity by stable numeric GitHub user id, never login.
- [ ] Default scopes are minimal and documented. Email remains a hint.
- [ ] Test with local fake GitHub endpoints; normal tests require no internet or
  real credentials.
- [ ] Add plugin README: GitHub app creation, exact callback URL, scopes,
  configuration, test, rotation, disable/uninstall effects, and troubleshooting.

**Exit:** GitHub is the real reference implementation and passes end-to-end
fixture tests through Protocol V2 and Host session issue.

## Milestone M5 — Google And Discord Plugins

### Google

- [ ] Use OIDC discovery and Authorization Code + PKCE.
- [ ] Validate issuer, audience, nonce, signature, expiry, and stable `sub`.
- [ ] Default scopes: `openid profile email`.
- [ ] Treat `email_verified` as display/prefill evidence only in v1.

### Discord

- [ ] Use Authorization Code flow with minimal `identify` and `email` scopes.
- [ ] Key identity by stable Discord user id, not username/global name.
- [ ] Bound Discord HTTP calls and normalize rate-limit/transient errors.

### Shared acceptance

- [ ] Separate settings/secrets and exact provider ids.
- [ ] Local fake-provider tests cover success, denial, invalid token/claim,
  subject mismatch, timeouts, malformed responses, and redaction.
- [ ] Documentation includes current official setup links and callback steps.

**Exit:** GitHub, Google, and Discord can be enabled together and remain
independent under failure or disable.

## Milestone M6 — Telegram Plugin

- [ ] Confirm Telegram's current official login mechanism during M0/M6 and
  document the selected protocol and why it fits the common Host contract.
- [ ] Keep Telegram verification in a dedicated adapter; do not weaken the
  OAuth/OIDC contract to mimic Telegram.
- [ ] Verify every required signature/hash, audience/origin/callback binding,
  freshness (`auth_date` or protocol equivalent), and replay constraint.
- [ ] Store Bot Token/client secret only through plugin SecretStore settings.
- [ ] Key identity by stable Telegram subject/user id, never username.
- [ ] Add local deterministic signature fixtures, stale payload, tamper, replay,
  wrong bot/client, and redaction tests.
- [ ] Provide setup and troubleshooting README consistent with the unified admin
  page.

**Exit:** Telegram coexists with the OAuth/OIDC providers without special
vendor code in Core UI or account/session services.

## Milestone M7 — Hardening, Documentation, And Release Gate

- [ ] Run lifecycle tests: restart, disable, uninstall, upgrade, rollback,
  trust revoke, Safe Mode, ForceDrain, and stale exact-artifact activation.
- [ ] Run concurrency tests for duplicate registration, subject linking, state
  consume, unlink, and provider activation CAS.
- [ ] Verify logs/audit/API/browser history contain no code, token, secret,
  verifier, raw state, raw provider subject, or sensitive upstream payload.
- [ ] Verify CSP/connect/navigation requirements are minimal and documented.
- [ ] Update OpenAPI and run `ruby scripts/validate-openapi-refs.rb`.
- [ ] Update Extension Surface Matrix for identity routes, hooks, queries,
  admin/public components, permissions, cache, jobs, and lifecycle.
- [ ] Update bilingual operator and extension-author docs.
- [ ] Update `knowledge/modules/identity.md`, `knowledge/modules/extensions.md`,
  this plan status, `knowledge/plans/README.md`, `knowledge/index.md`, and one
  hot session handoff.
- [ ] Run focused Go/plugin/frontend tests, then `./scripts/test.sh`.
- [ ] Browser QA at desktop and mobile widths against the existing user-owned
  port 3000 server; never kill that server.

**Exit:** all four providers are product-usable, security evidence is recorded,
full repo gate passes, and this plan becomes `completed`.

---

## Stable Error Reasons

Use existing reasons where semantics match. Add only the minimum missing set,
with backend localization and zh-CN/en-US UI mapping. Recommended semantic set:

- `auth.provider_not_found`
- `auth.provider_unavailable`
- `auth.provider_input_invalid`
- `auth.provider_not_enabled`
- `auth.provider_callback_expired`
- `auth.provider_callback_invalid`
- `auth.provider_callback_replayed`
- `auth.provider_cancelled`
- `auth.external_identity_unlinked`
- `auth.external_subject_conflict`
- `auth.external_link_conflict`
- `auth.last_login_method_required`
- existing `auth.registration_disabled`
- existing session/risk policy denial reasons

Public messages must not reveal whether an email belongs to an account.

## Verification Matrix

| Scenario | Required result |
| --- | --- |
| Password-only site | Existing login/register behavior unchanged |
| Plugin installed/configured | No public button until Host activation |
| Multiple providers | Deterministic configured order; independent failures |
| Known active link | Core issues normal browser session after policies |
| Unknown link on login | Explicit unlinked result; no account creation |
| External registration | Explicit local fields; user + role + link atomic |
| Email matches existing user | No automatic link or login |
| First user | External registration denied; Core bootstrap remains available |
| Disabled/banned user | No session; generic public error |
| Replay callback | Rejected; no second session/link/user |
| Provider/artifact changes mid-flow | Attempt fails closed |
| Plugin disabled/uninstalled | Button disappears; link retained inert |
| Safe Mode | All third-party auth unavailable; Core password works |
| Last login method unlink | Blocked until another method exists |
| External-only password setup | Credential created safely; future password login works |
| Admin without permission | 403, no provider/config/probe mutation |
| Restore defaults | External operations off; secrets preserved and stated |

## Required Commands

Run commands from the repository root unless the command says otherwise.

```bash
cd apps/api && go test ./...
ruby scripts/validate-openapi-refs.rb
cd apps/web && bun run typecheck
cd apps/web && bun run build
./scripts/test.sh
```

For focused provider plugin tests, use their checked-in package commands and
local fake HTTP servers. Do not make CI or ordinary unit tests call live vendor
services.

Before `go get`, `go mod tidy`, `bun install`, or `bun add`:

```bash
export https_proxy=http://127.0.0.1:7897
export http_proxy=http://127.0.0.1:7897
export all_proxy=socks5://127.0.0.1:7897
```

## Out Of Scope

- SForum acting as an OAuth authorization server
- SAML, LDAP, SCIM, passkeys/WebAuthn, phone/SMS login
- Native mobile deep-link callbacks
- Automatic account merge by email
- Importing provider tokens from legacy SForum
- Provider access to raw Core database, raw request/session authority, roles,
  or permissions
- Runtime frontend builds or provider-supplied executable login UI
- Analytics dashboards beyond minimal health/audit evidence

## Grok Delivery Rules

For every milestone:

1. Keep changes limited to that milestone and preserve unrelated dirty files.
2. State assumptions before coding when repository evidence is incomplete.
3. Do not guess an interface; inspect the existing type/route/Manifest contract.
4. Reuse existing libraries, registries, settings renderer, SecretStore,
   session manager, risk/session policy, audit, and external-link store.
5. Add allowed and denied tests in the same milestone as each unsafe route.
6. Report files changed, migrations added, stable API changes, permissions,
   security implications, and exact command results.
7. Stop and request a product decision if implementation would violate a
   frozen decision above; do not silently widen Core or plugin authority.
8. Prefer one reviewable commit per milestone. Do not claim completion while a
   required test is skipped or failing.

## Definition Of Done

- Four provider plugins are available and can be enabled simultaneously.
- Core contains no GitHub/Google/Discord/Telegram vendor behavior.
- External login, registration, linking, unlinking, and external-only password
  setup work end to end.
- Unified admin and user surfaces are beginner-friendly and permission-aware.
- Callback/state/account-link takeover protections have explicit passing tests.
- Safe Mode and Core password recovery remain usable.
- Secrets and provider tokens stay out of Core storage, logs, audit, and public
  responses.
- OpenAPI, bilingual UI/docs, knowledge base, and extension surfaces are current.
- `./scripts/test.sh` passes and browser QA evidence is recorded.

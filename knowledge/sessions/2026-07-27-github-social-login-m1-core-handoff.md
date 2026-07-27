# 2026-07-27 GitHub Social Login M1 Core Handoff

## Status

**M0 + M1 complete and tested** (paused per user request for review before
M2-M5). No production behavior change is claimed: the GitHub plugin package is
not yet built, so no provider is publicly activatable yet. The Host security
foundation, bootstrap wiring, provider-catalog activation filter, admin
Login Methods API surface, and multi-provider integration tests are all
implemented and green.

## Changed

M0:
- Added `decisions/2026-07-27-github-social-login-m0-contract-freeze.md`: verified
  GitHub OAuth App behavior (authorization/token endpoints, PKCE S256, stable
  `id` subject, `user:email` scope), library survey (selected
  `golang.org/x/oauth2` over `goth` and direct HTTP), frozen additive schemas,
  Core-HMAC subject digest contract, callback/ticket/public-admin response
  contracts, and stable error reasons.

M1 core (new files under `apps/api/app/Models/Identity/`):
- `external_subject_digest.go` (+test): `IDENTITY_SUBJECT_HMAC_SECRET`, Core
  `ComputeSubjectDigest(providerId, subject) = HMAC-SHA256(key, pid||NUL||subject)`.
  Production rejects weak/missing secret; dev falls back to process-random key.
- `external_auth_callback_state.go` (+test): `CallbackTransaction`,
  `RegistrationTicket`, `CallbackStateStore`/`RegistrationTicketStore` interfaces
  with in-memory + Redis implementations, atomic one-use consume (lua GETDEL),
  PKCE/state/token generators, `ExternalAuthOperation` enum, frozen error
  reasons (`auth.provider_callback_expired/invalid/replayed`,
  `auth.external_registration_ticket_invalid/expired`).
- `external_auth_provider_activation.go`: `ProviderActivation` Host-owned
  activation state with CAS (revision) + audit, all defaults off; PostgreSQL
  store `Get/List/Upsert/RecordProbe/Delete/ResetOperationsToDefaults`.
- `external_auth_store.go`: `PostgresExternalAuthStore` for credential-less
  user support (`HasPasswordCredential`, `CountActiveExternalLinks`),
  recent-auth markers (`MarkUserRecentlyAuthenticated`/`IsUserRecentlyAuthenticated`,
  5-min TTL), `RecordExternalAuthAudit`; tx-aware helpers
  (`createUserWithoutCredentialTx`, `assignDefaultRoleTx`, etc.).
- `external_auth_service.go`: `ExternalAuthService` orchestration —
  `CompleteLogin` (FindActive + status check, generic unlinked failure),
  `CompleteRegistration` (atomic user+default role+link in one tx, zero-user
  bootstrap guard, registration-policy guard),
  `CompleteLink` (subject-conflict + actor binding),
  `CanUnlink`/`CanRemovePassword` (last-login-method protection).

M1 controller (new `apps/api/app/Http/Controllers/Identity/external_auth.go`):
- Reserved Core callback route `GET /auth/providers/{providerId}/callback`
  (browser GET after GitHub redirect): consumes callback tx, calls plugin
  complete, dispatches login/registration/link.
- `handleExternalLoginCallback`: account resolution + risk eval + session
  issue (reuses `runSessionIssue`/`beginSessionIssue`/`applySessionDeviceInfo`/
  `auditLogin`/`enforceMaxSessions`) + `AuditActionExternalLogin` audit.
- `handleExternalRegistrationCallback`: mints opaque one-time registration ticket,
  302 to `/register?ticket=…&provider=…`.
- `handleExternalLinkCallback`: binds current session user.
- `POST /auth/external-registration`: ticket consume + atomic registration +
  session issue + `AuditActionExternalRegister`.
- `GET /auth/external-identities`: redacted self-service link list.
- `DELETE /auth/external-identities/{linkId}`: recent-auth + last-method
  protected unlink.
- Safe-return path validator (open-redirect defense); all failures 302 with
  minimal safe reason; raw subject/digest/code/token/state/verifier/secret
  never reach browser.

M1 controller wiring (`controller.go`, `routes.go`):
- Controller struct fields: `externalAuthService`, `callbackStateStore`,
  `registrationTicketStore`, `externalLinkStore`, `externalAuthStore`.
- `WithExternalAuthService(...)` builder.
- Routes registered in `RegisterRoutes`.

M1 contracts:
- `auth_provider_flow.go`: `AuthProviderCompleteResult` gained
  `ProviderSubject` + `ProviderContractVersion`; `parseAuthCompleteOutput` now
  supports Core-HMAC mode (raw `providerSubject`) AND legacy fixture mode
  (`providerSubjectDigest`), preferring raw subject when both present; Complete
  computes the Core digest when plugin returns raw subject. Backward compatible
  with the membership-reference fixture.
- `types.go`: `AuditActionExternalLogin/Register/Link/Unlink` constants.

Permission:
- `seeds.go`: `PermissionIdentityProviderManage = "identity.provider.manage"`
  constant, seed entry, operator role template mapping.
- `seeds_test.go`: added to the required-catalog list (drift guard).
- Migration `202607270056_identity_provider_manage_permission.sql`: seeds the
  permission and grants to `super_admin` + `operator`.
- `apps/web/i18n/locales/{zh-CN,en-US}.json`: `permissionCatalog.identity.provider.manage`
  label + description.
- `tests/validate-identity-ui.js`: added the new catalog keys to the check list.

Migration:
- `202607270055_external_auth_host_state.sql`: drops `user_credentials.password_hash`
  NOT NULL (credential-less users), adds `user_credentials.method`,
  `user_recent_auth` table (recent-auth durable fallback), and
  `identity_provider_activations` table (Host activation catalog). Goose Down
  refuses to drop non-empty tables.

## Decisions

- Browser GET callback (not frontend-mediated POST): frozen after user
  confirmation. Reserved Core route outside Route Registry.
- Core HMAC (not plugin-side digest): frozen after user confirmation. Plugin
  returns raw `providerSubject`; Core computes keyed digest; raw subject and
  digest never leave the Host process.
- `x/oauth2` selected for OAuth 2.0 protocol mechanics; goth rejected (owns
  session/callback/state); direct HTTP rejected for token exchange.
- External registration reuses the password-registration transaction primitives
  (`createUserWithoutCredentialTx` + `assignDefaultRoleTx` + `LinkTx`) in one
  caller-owned tx; no fake password is ever created.

## Changed (M1.5–M1.7 follow-up)

M1.5 — bootstrap wiring (host stack now runtime-active):
- `app/Models/Identity/external_auth_stack.go` (new): `ExternalAuthStack`
  aggregate + `NewExternalAuthStack(pool, redisClient, registry,
  registrationEnabled)`. Redis callback/ticket stores in production;
  in-memory fallback for tests/local-without-Redis.
- `app/Providers/identity.go`: `WithExternalAuthStack(stack)` builder on
  `IdentityProvider` (no bootstrap import → no cycle).
- `bootstrap/api_assembly.go`: `newExternalAuthStack` constructed with the
  shared Redis client + pool + IdentityRegistry + options service, injected
  via `WithExternalAuthStack`. The reserved Core callback route, external
  registration, list, unlink, and admin Login Methods endpoints are now
  runtime-active (not just registered).

M1.6 — provider-catalog activation filter + Host-owned OAuth material:
- `auth_providers.go::listAuthProviders`: now merges Host activation state per
  provider and returns `activatedOperations` (login/registration/link) +
  `ownerExtensionId`. Display comes entirely from the Host catalog — no
  `if provider == github` branch anywhere.
- `auth_providers.go::authProviderStart`: now gates on Host activation
  (`RequireActivated`) BEFORE calling the plugin, generates Host-owned
  `state` + PKCE (`GenerateCallbackState`, `GeneratePKCE`), stores the
  one-use `CallbackTransaction`, and passes `state`/`codeChallenge`/
  `callbackUrl` to the plugin so it can build the GitHub authorize URL.
- `auth_provider_flow.go`: `AuthProviderStartInput` gained
  `State`/`CodeChallenge`/`CallbackURL`; `Start` forwards them to the plugin
  input map; `prepareAuthStartInput` trims + length-validates them.
- `external_auth_callback_state.go`: `ExternalAuthCallbackPath/URL(providerID)`
  returns the reserved Core callback path `/api/v1/auth/providers/{pid}/callback`.
- `controller.go`: new `activationStore` field, auto-wired from the service
  via `WithExternalAuthService` → `svc.ActivationStore()`.
- `external_auth_service.go`: `ActivationStore()` accessor.

M1.6b — admin Login Methods API surface:
- `external_auth_admin.go` (new): `GET /admin/identity/providers`
  (discovered/trusted/enabled/configured/probed/activated aggregate),
  `PATCH /admin/identity/providers/{pid}` (CAS activation/order),
  `POST /admin/identity/providers/{pid}/probe` (records probe result;
  M1 returns `probe_pending` — real provider_probe RPC lands in M2 with
  the GitHub plugin), `POST /admin/identity/providers/reset` (operations
  off, secrets preserved — stated in response). All gated on
  `identity.provider.manage`; `actor.Can(...)` authoritative.
- `external_auth_admin_test.go` (new): `fakeProviderActivationStore` +
  `TestFakeActivationStore_CASAndReset` (default-off, CAS conflict,
  reset-to-defaults) + `TestFakeActivationStore_ProbeRecording`.
- `routes.go`: four new admin routes registered.

M1.7 — multi-provider integration tests:
- `external_auth_multi_provider_test.go` (new) proves the hard multi-provider
  acceptance requirements against the real `IdentityRegistry`, real
  `CallbackStateStore`, real `ComputeSubjectDigest`, and a fake link store:
  - `TestMultiProvider_DiscoveryAndOrdering` — two providers discovered +
    sorted (kind asc, priority desc, id asc).
  - `TestMultiProvider_IndependentActivation` — A activated, B not; states
    independent.
  - `TestMultiProvider_CallbackStateCrossProviderRejected` — A's state
    cannot satisfy B's provider/digest binding.
  - `TestMultiProvider_CrossOperationRejected` — login state cannot satisfy
    registration/link.
  - `TestMultiProvider_SameSubjectAcrossProvidersNoConflict` —
    `HMAC(key, pidA||NUL||s) != HMAC(key, pidB||NUL||s)`; same provider+subject
    stable. Core property of the keyed digest.
  - `TestMultiProvider_OneFailureDoesNotAffectOther` — A's replay/expiry does
    not affect B's usable state.
  - `TestMultiProvider_CallbackTransactionActorBinding` — link binds actor;
    login/registration must be actorless.
  - `TestMultiProvider_CompleteLoginUnlinkedGeneric` — generic unlinked
    failure for any provider (no provider-specific branch).
- `external_auth_service_test.go` (new): activation gating (default-off,
  when-enabled, unknown-provider), operation-mismatch rejection,
  `resolvedDigest` Core-HMAC-vs-legacy preference, empty-assertion fail-closed,
  `ActivationStore()` accessor.

## Verified This Session

- `cd apps/api && go build ./...` — passes.
- `cd apps/api && go test ./app/Models/Identity/` — passes (including new
  HMAC/callback/ticket/parse tests: `TestComputeSubjectDigest_*`,
  `TestVerifySubjectDigest_*`, `TestInMemoryCallbackStateStore_*`,
  `TestInMemoryRegistrationTicketStore_*`, `TestGeneratePKCE_*`,
  `TestGenerateCallbackState_*`, `TestParseExternalAuthOperation`,
  `TestCallbackTransaction_*`, `TestParseAuthCompleteOutput_*`).
- `cd apps/api && go test ./app/Http/Controllers/Identity/` — passes.
- `cd apps/api && go test ./app/Support/Localization/` — passes (fixed a
  pre-existing `auth.registration_disabled` gap; added all new external-auth
  reasons + `auth.provider_activation_cas_conflict` to zh-CN + en-US).
- `cd apps/api && go test ./app/...` — passes (full app test sweep, exit 0,
  no regressions introduced by M1).
- `ruby scripts/validate-openapi-refs.rb` — OK (2200 refs / 54 files).
- `node tests/validate-identity-ui.js` — passes (new permission catalog keys
  validated).

## Next (M2 → M5)

M1 is complete and runtime-wired. Next work starts at M2 (the built-in GitHub
plugin package). The reserved Core callback route, external registration,
list, unlink, and admin Login Methods endpoints are now active at runtime —
they will return empty/unavailable until a provider is Host-activated, which
requires M2 to ship the actual `sforum.auth-github` package.

M2: built-in `sforum.auth-github` plugin package (manifest V3, identity
provider declaration with 6 operations, schemas, settings with
`clientId`/`clientSecret` secret, backend using `x/oauth2` + fake-GitHub test
server, bounded probe); extend `scripts/build-builtin-plugins.sh` + Dockerfile;
prove `SyncBuiltins` stages without trusting/enabling/activating; headless
end-to-end tests through Protocol V2. Also wire the real `provider_probe`
action so the admin probe endpoint (currently `probe_pending`) reports
truthful reachability/config-presence.

M3: admin Login Methods tab (Host aggregate UI + `SFExtensionSettingsRenderer`),
toggles/callback copy/probe/inline errors/Toasts/optimistic revisions/restore
defaults, distinguish discovered/trusted/enabled/configured/probed/activated,
403/stale/upgrade/Safe Mode/SSR/typecheck/desktop/mobile tests, modular OpenAPI
for the four new admin paths + the public callback/registration/list/unlink
paths (currently undocumented).

M4: SSR-safe auth-provider composable using the Host catalog, GitHub controls
on login and registration shells, safe callback feedback and opaque
registration-ticket continuation, validated return navigation + session-state
rules, linked accounts/unlink/inert/recent-auth/password-setup on account
security, zh-CN/en-US copy for all states and reasons, password-only /
linked-unlinked / explicit-registration / cancellation / provider-failure /
duplicate-email-hint / link / unlink-blocked / password-setup / responsive
tests.

M5: lifecycle (restart/disable/uninstall/staged upgrade/new-digest
activation/rollback/trust-revoke/Safe Mode/ForceDrain/callback-during-change),
replay/expiry/cross-binding/concurrent/subject-race/activation-CAS/unlink-race
tests, log/audit/API/history/diagnostics redaction audit, start/callback rate
limits, non-enumerating public errors, Identity Extension Surface Matrix +
document intentionally closed callback/session surfaces, bilingual operator/
author docs, knowledge base update (identity/extensions/plan/index/handoff),
full repo gate, desktop/mobile Browser QA against the user-owned web server.

M2: built-in `sforum.auth-github` plugin package (manifest V3, identity
provider declaration with 6 operations, schemas, settings with
`clientId`/`clientSecret` secret, backend using `x/oauth2` + fake-GitHub test
server, bounded probe); extend `scripts/build-builtin-plugins.sh` + Dockerfile;
prove `SyncBuiltins` stages without trusting/enabling/activating; headless
end-to-end tests through Protocol V2.

M3: admin Login Methods tab (Host aggregate + `SFExtensionSettingsRenderer`),
toggles/callback copy/probe/inline errors/Toasts/restore defaults, 403/stale/
upgrade/Safe Mode/SSR/typecheck/desktop/mobile tests, modular OpenAPI.

M4: SSR-safe auth-provider composable, GitHub controls on login/register,
callback feedback, opaque ticket continuation, validated return navigation,
linked accounts/unlink/inert/recent-auth/password-setup on account security,
zh-CN/en-US copy, responsive tests.

M5: lifecycle (restart/disable/uninstall/staged upgrade/new-digest
activation/rollback/trust-revoke/Safe Mode/ForceDrain/callback-during-change),
replay/expiry/cross-binding/race tests, log/audit/API/history redaction audit,
rate limits, non-enumerating errors, Identity Extension Surface Matrix,
bilingual operator/author docs, knowledge base update, full repo gate, desktop/
mobile Browser QA.

## Open Questions

- Admin probe shape: confirm `provider_probe` action (like SMTP/storage) is the
  intended mechanism vs a new identity-specific probe action. Current M1
  implementation returns `probe_pending` and defers the real RPC to M2 when the
  GitHub plugin ships.
- DB integration tests for `ExternalAuthService.CompleteRegistration/Login/Link`
  end-to-end (last-method protection, recent-auth gating, replay/expiry/cross-
  binding with a real Postgres pool) are deferred to M2's e2e slice so they can
  run against the actual plugin runtime; M1 unit-level coverage is in place.

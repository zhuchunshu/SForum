# 2026-07-27 GitHub Social Login T1E Handoff

## Status

**T1E complete. Full M1R Host foundation exit is met.** Next is **M2 / T2**
(GitHub protocol package) in a **fresh conversation only**. Do not start M2,
plugin, or UI work in the same dialogue that finished T1E.

Prior: `sessions/2026-07-27-github-social-login-t1d-handoff.md`.

## Changed

- Modular OpenAPI paths/schemas for:
  - `GET /auth/providers/{providerId}/callback`
  - `POST /auth/external-registration`
  - `GET /auth/external-identities`
  - `DELETE /auth/external-identities/{linkId}`
  - `POST /auth/password`
  - `GET|PATCH|POST /admin/identity/providers…`
  - legacy complete `410 auth.provider_callback_required`
  - public catalog `activatedOperations`
- Core Route Catalog + `catalog-identities.json` + routes docs: reserved
  callback and related Host routes; policy text states callback/session
  authority is closed to Route Registry replacement.
- `scripts/v3-catalog/generate.mjs` routePolicy + reviewed guards for
  `identity.provider.manage` and reserved callback.
- Extension surface matrix identity routes note: callback/session closed.
- Controller HTTP tests (`external_auth_http_test.go`): allowed/denied admin,
  redacted identities, callback replay/artifact mismatch, ticket invalid,
  legacy 410, reserved callback registration.
- Model lifecycle + two-provider operation tests
  (`external_auth_t1e_lifecycle_test.go`, multi-provider suite).
- Postgres transition/rollback tests (`external_auth_t1e_postgres_test.go`;
  skip without `SFORUM_TEST_DATABASE_URL`).
- M0 ADR corrected: Host owns state, PKCE verifier, callback URL, callback
  transaction; additive start/complete fields listed.
- Production Route Guard partition closeout: `self_credentials` admits
  `external_identities` + cookie-bound `setup_password`; unlink remains
  resource-dependent fail-closed; `forum.read` admits cataloged
  `comment_page` (catalog discovery consistency).

## Decisions

- Full `scripts/v3-catalog/generate.mjs` regen was not run: unrelated UI surface
  identity drift (16 components/pages) would pull non-T1E catalog churn.
  Core route entries were hand-synced into `core_catalog_gen.go` + identities +
  routes.md/json for the new Host routes (plus three pre-existing discovered
  routes already mapped in identities for consistency).
- Callback remains a Core Route Catalog entry with contextual bootstrap guard
  and explicit non-replaceable policy — declared for inventory, closed to
  plugin replacement.
- `external_identity_unlink` stays resource-dependent in the production
  `self_credentials` evaluator (ownership/revision enforced in Host handler;
  inherited Route Registry path fails closed without a link resource policy).

## Verification

Focused packages (2026-07-27 closeout):

```text
cd apps/api && go test ./app/Models/Identity/ ./app/Http/Controllers/Identity/ \
  ./app/Http/ ./app/Support/Routes/ ./app/Support/Localization/ -count=1
# ok Identity 3.195s
# ok Controllers/Identity 2.271s
# ok Http 6.251s
# ok Routes 2.805s
# ok Localization 3.221s
# EXIT:0
```

T1E-named + multi-provider (all PASS; Postgres SKIP without env):

```text
cd apps/api && go test ./app/Models/Identity/ ./app/Http/Controllers/Identity/ \
  -count=1 -run 'T1E_|MultiProvider_'
# MultiProvider_*: DiscoveryAndOrdering, IndependentActivation,
#   CallbackStateCrossProviderRejected, CrossOperationRejected,
#   SameSubjectAcrossProvidersNoConflict, OneFailureDoesNotAffectOther,
#   CallbackTransactionActorBinding, CompleteLoginUnlinkedGeneric
# T1E_*: TwoProvidersExecuteIndependentLinkAndLogin,
#   SafeModeDisableArtifactAndRevoke, StateAndTicketExpiry,
#   ActorSessionMismatchAndUnauthorizedLinkZeroWrite,
#   UnlinkRaceRevisionConflict, RegistrationTicketConsumeIsOneUse,
#   LegacyAuthCompleteReturns410, CallbackReplayRedirectsWithoutSecrets,
#   CallbackExactArtifactMismatchRedirect,
#   ExternalIdentitiesDeniedWithoutSession,
#   ExternalIdentitiesRedactedWhenAuthed,
#   UnlinkAndPasswordDeniedWithoutSession,
#   AdminProvidersDeniedWithoutPermission,
#   AdminProvidersAllowedWithPermission,
#   ExternalRegistrationTicketInvalid, ReservedCallbackRouteRegistered
# T1E_Postgres* SKIP (SFORUM_TEST_DATABASE_URL unset)
# EXIT:0
```

OpenAPI:

```text
ruby scripts/validate-openapi-refs.rb
# OpenAPI references OK: checked 2262 refs across 54 files.
```

Full API:

```text
cd apps/api && go test ./... -count=1
# 104 ok packages, 0 FAIL, EXIT:0
```

## Next

Start a **new** conversation for **T2 / M2A only**: built-in GitHub protocol
package, manifest/schemas, fake GitHub server. Do not continue M2 in this
session. Do not start admin UI (M3) or public UI (M4).

## Open Questions

- None for M2 entry from Host-foundation perspective. Runtime GitHub evidence
  still requires M2 packaging + fake-server E2E.
- Optional residual: wire an `ExternalLinkGuard` resource policy for production
  Route Registry inheritance of unlink (Core Fiber path already enforces
  ownership). Not required for M2 entry.

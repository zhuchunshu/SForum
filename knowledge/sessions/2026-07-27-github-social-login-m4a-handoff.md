# 2026-07-27 GitHub Social Login T5 / M4A Handoff

## Status

**T5 / M4A complete. M4A exit complete.** Next is **T6 / M4B** (account security
link/unlink, session-bound recent-auth, password setup UI) in a **fresh
conversation only**.

Do **not** start M4B or M5 in the same dialogue that finished T5/M4A.

Prior: `sessions/2026-07-27-github-social-login-m3-handoff.md`.

## Changed

### Plugin presentation → Host catalog injection

- Manifest V3 identity providers accept optional `label` (LocalizedText) and
  `icon` (schema + Go types; `Label` is `*LocalizedText` so omitempty works).
- Lifecycle Identity publication maps plugin `label`/`icon` into
  `identityregistry.Provider` (`Label`, `LabelLocales`, `Icon`).
- `ListEffectivePublicCatalog(ctx, registry, locale)` resolves plugin labels
  via `ResolveProviderLabel`; public `GET /auth/providers` returns
  `label` + `icon` for the request Accept-Language.
- Built-in `sforum.auth-github` declares:
  - `label: { "zh-CN": "GitHub", "en-US": "GitHub" }`
  - `icon: "i-tabler-brand-github"`
  - optional `manifest/langs` authProvider copy (plugin-owned)

### Public auth UI (Host shell only)

- `useAuthProviders` — SSR-safe catalog reader; filters by
  `activatedOperations`; **no** vendor id hard-coding.
- `SFAuthProviderButtons` — generic divider/button chrome; name/icon from
  catalog.
- `SFLoginFormPage` / `SFRegisterFormPage` — provider entry + password path
  preserved; registration ticket mode at `/register?ticket=` posts
  `POST /auth/external-registration` without password.
- `useExternalAuthFeedback` + `externalAuthFeedback` utils — consume Host
  `ext_auth` reasons; success Toast (including guest bounce via pending state).
- Core i18n: generic shell + Host stable reasons only (**no** GitHub brand keys).

### Contracts / OpenAPI

- `AuthProviderListItem` adds optional `label`, `icon` with plugin-ownership
  descriptions.

## Decisions

- **Brand copy and icons belong to the plugin**, injected through the Host
  public catalog. Core must not hard-code GitHub (or any vendor) brand strings
  in web i18n or `includes('github')` display branches on public shells.
- **Host still owns** the login/register shell geometry, password form, opaque
  ticket continuation, return-path validation, and stable `ext_auth` reason
  mapping (product/security semantics, not vendor marketing).
- Full plugin Vue component injection for provider buttons is **out of M4A**;
  catalog presentation metadata is the multi-provider V1 contract.
- M3 admin Login Methods still has residual github id heuristics for admin
  display; left untouched (not M4A scope; not a public-shell defect).

## Verification

```text
cd apps/web && bun test tests/authProvidersPublicUi.test.ts \
  tests/authRouteRendering.test.ts tests/guestMiddleware.test.ts
# 25 pass

cd apps/api && go test ./app/Support/ExtensionManifest/ \
  ./app/Support/IdentityRegistry/ ./app/Models/Identity/ \
  ./app/Http/Controllers/Identity/ -count=1
# ok

cd apps/api && go test ./app/Support/Extensions/ \
  -run 'TestGitHubAuthPluginProtocolV2HeadlessHostSession|TestReferenceMembership' \
  -count=1
# ok

ruby scripts/validate-openapi-refs.rb
# OpenAPI references OK

cd apps/web && bun run typecheck
# useAuthProviders clean; pre-existing admin.vue MouseEvent return type errors remain
```

## Next

Start a **new** conversation for **T6 / M4B only**:

1. Account security linked-accounts list (redacted)
2. Link / unlink with session-bound recent-auth
3. Inert state when provider disabled
4. External-only password setup UI
5. zh-CN / en-US for account-security states

Do **not** start M5 in that conversation.

## Open Questions

- Optionally align admin Login Methods display with the same catalog
  `label`/`icon` fields (remove residual github id heuristics) in a later
  cleanup — not required for M4A exit.
- Rebuild/restage the GitHub built-in after manifest presentation fields so
  runtime package digest matches source before product QA.

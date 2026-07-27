# 2026-07-27 GitHub Social Login T4 / M3 Handoff

## Status

**T4 / M3 complete. M3 exit complete.** Next is **T5 / M4A** (public login,
callback feedback, explicit registration UI) in a **fresh conversation only**.

Do **not** start M4 public UI, M4B account security, or M5 in the same dialogue
that finished T4/M3.

Prior: `sessions/2026-07-27-github-social-login-m2b-handoff.md`.

## Changed

- **Host admin aggregate** (`external_auth_admin.go`):
  - states: discovered / trusted / enabled / configured / probed /
    publiclyActivated (plus activated intent + artifactBound)
  - absolute `callbackUrl` from trusted `APP_URL`
  - `settingsPath` for extension Host settings page
  - `IsProviderConfigured` wired from extension settings (bootstrap)
- **Settings permission delegation** (mail-style):
  `identity.provider.manage` may bootstrap/update settings only for plugins
  that declare Identity auth providers
- **Login Methods UI**: `apps/web/app/pages/admin/settings/login-methods.vue`
  - nav entry + `identity.provider.manage` gate
  - embeds `SFExtensionSettingsRenderer`
  - CAS operation toggles, callback copy, probe, restore defaults, Toasts
  - zh-CN / en-US copy; operator `roleTemplates` aligned with seeds
- **OpenAPI** admin item schemas: `configured`, `publiclyActivated`,
  `callbackUrl`, `settingsPath`, state descriptions
- **Catalogs**: stable UI identity + admin surface placements for
  `/admin/settings/login-methods` (full `generate.mjs` blocked by unrelated
  missing topic-edit identity; entries applied manually)

## Decisions

- Probe remains truthful: without a real identity provider probe RPC, Host
  records `probe_pending` / `probe_unavailable` with `ok=false`; UI never
  treats pending as success.
- `publiclyActivated` requires intent + artifactBound + enabled + configured +
  non-Safe-Mode (stricter than raw activation intent).
- Auth-plugin settings for operators reuse extension settings APIs under
  `identity.provider.manage` rather than inventing a second settings store.

## Verification

```text
cd apps/api && go test ./app/Http/Controllers/Identity/ \
  -run 'TestListAdminIdentityProviderItems|TestT1E_Admin|TestMapActivation|TestAdminPatch' -count=1
# ok  .../Identity  EXIT:0

cd apps/api && go test ./app/Models/Extensions/ \
  -run 'TestAuthProvider|TestCanManage|TestAdminPageBootstrapAllowsIdentity|TestServiceAdminPageBootstrapAllows' -count=1
# ok  .../Extensions  EXIT:0

ruby scripts/validate-openapi-refs.rb
# OpenAPI references OK

cd apps/web && bun test tests/adminLoginMethods.test.ts
# 3 pass

node tests/validate-identity-ui.js
# Identity UI validation passed

cd apps/web && bun run typecheck
# login-methods.vue clean; pre-existing admin.vue MouseEvent return type errors remain
```

## Next

Start a **new** conversation for **T5 / M4A only**:

1. SSR-safe auth-provider composable from Host public catalog
2. GitHub controls on login/registration shells
3. Safe callback feedback + opaque registration-ticket continuation
4. Preserve validated return navigation

Do **not** start M4B account security or M5 in that conversation.

## Open Questions

- Wire real provider probe RPC through identity.runtime when product wants
  `ok=true` beyond credentials-presence/reachability pending (not required for
  M3 exit).
- Full `scripts/v3-catalog/generate.mjs` needs a reviewed identity for
  unrelated `topics/[topicId]/edit.vue` before a clean regenerate.

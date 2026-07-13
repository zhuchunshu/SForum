# 2026-07-13 Session Handoff — Buildless Extension Settings UI Complete

## Changed

### P0 — compatibility inventory

- Captured legacy array, document, includes/shards, tabs/actions, and component
  manifest fixtures before changing runtime behavior.
- Reconciled the existing settings storage, secret handling, provider Probe,
  frontend trust, quarantine, Web Release, and public Page Registry paths.

### P1 — versioned Settings Document

- Plugins and themes now share `schemaVersion: 1` Settings Documents.
- Legacy `[]settings` and sorted `includes.settings` shards normalize to
  `mode=schema`, `layout=form` without breaking canonical manifests.
- Validation covers versions, ids, group/tab references, component conflicts,
  safe package paths, actions, locale fallback, and theme restrictions.
- Settings API responses expose a localized renderer model rather than asking
  the browser to interpret raw manifest JSON.

### P2 — host Schema renderer

- Added `SFExtensionSettingsRenderer` and focused tabs/group/field/callout/
  actions/footer/trusted-component children.
- Supports forms, tabs, ordered groups, columns, callouts, optional
  presentation fallback, existing field controls, masked secret preservation,
  save, and recommended reset.
- Removed the default theme's stale admin Vue package. Its settings are a rich
  Schema document in development and production through the same renderer.
- Added an uploaded schema-only theme fixture.

### P3 — Settings Actions and configure-before-enable

- Added the host-owned settings action endpoint and OpenAPI contract.
- Backend enforces permission, declaration/allowlist, input keys and size,
  timeout, structured results, audit, lifecycle, and secret boundaries.
- Installed/disabled plugins and inactive themes can read/save settings without
  starting normal routes, jobs, hooks, providers, events, or schedules.
- Code-requiring pre-enable probes use a restricted short-lived plugin runtime.
- SMTP now implements `ProviderProbe`; SMTP and filesystem storage use Schema
  + Actions and no longer need custom settings SFCs.

### P4 — independent admin frontend identity and release reuse

- Added migration `202607130004_admin_frontend_digest.sql` and deterministic
  `adminFrontendDigest`, based only on legacy admin frontend inputs or prebuilt
  component assets.
- Grants and Web Release composition bind the admin digest rather than the full
  package digest. Backend/settings/public-theme-only bytes no longer trigger an
  admin rebuild or invalidate unchanged admin code trust.
- Active/ready compositions are reused; schema-only/actions/prebuilt paths skip
  Web Release creation.
- Runtime typecheck policy is `off | report | block`; CI and `test.sh` remain
  mandatory.

### P5 — author-prebuilt Component UI

- Published Admin Micro-frontend API v1 in `@sforum/admin-sdk` with a
  framework-neutral `mount(target, bridge)` and cleanup contract.
- Prebuilt `.mjs`/optional `.css` must come from the installed package under
  the safe admin dist root. Source SFCs and arbitrary remote URLs are rejected.
- Assets are served only from authenticated immutable digest URLs with
  allowlisted names, exact-byte digest verification, MIME/size/path checks,
  `nosniff`, and same-origin resource policy.
- Uploaded trust uses an actor-bound, one-use, five-minute confirmation
  challenge. The technical grant binds extension id, version, API version,
  component id, and `adminFrontendDigest`.
- The client dynamically imports JS from the digest URL, fetches authenticated
  CSS from the matching URL, serializes mount/dispose operations, passes the
  narrow bridge, cleans up on navigation, and reuses session quarantine.
- Missing/revoked/stale trust, import/contract/mount failure, or quarantine
  always leaves the Schema fallback usable.
- Added `sforum-prebuilt-settings` fixture and verified the complete trust,
  mount, save, revoke, and fallback loop without creating a Web Release.

### P6 — operator and author loop

- `make:plugin` and `make:theme` default to Schema UI.
- `--provider-slot` scaffolds host-rendered Probe actions;
  `--prebuilt-settings` scaffolds the v1 component plus Schema fallback.
- Updated authoring, trusted component, runtime theme, scenario, OpenAPI, SDK,
  CLI validation, extension badges, deprecation copy, module notes, ADR, and
  task book.
- Legacy settings arrays and trusted Vue Web Release contributions remain
  compatible; new settings-only packages are directed to Schema/Actions or
  prebuilt Component UI.

## Decisions

- “能不构建就不构建” is now enforced by separate presentation paths:
  Schema and Actions are host-rendered; prebuilt components are dynamically
  loaded; full Web Release remains only for host releases and legacy trusted
  Vue compatibility.
- Confirmation code is UX evidence of intent, not the trust boundary. Digest
  grants are the authority and byte changes invalidate old trust.
- Public theme Page Registry activation and admin component trust/loading are
  separate lifecycles. The default theme intentionally remains Schema-only.
- Optional CSS uses authenticated fetch plus MIME validation and `<style>`
  injection because a cookie-protected `<link>` is not reliable across the
  supported browser path.

## Compatibility

- Old array settings, include shards, legacy settings-page/header/footer
  contributions, and trusted Vue Web Releases continue to work.
- Existing secret encryption, masked reads, preserve-on-empty, reset semantics,
  enabled-plugin restart/rollback, audit, and API policy remain authoritative.
- Declining or revoking component trust does not delete settings, secrets,
  backend state, or the package.

## Verification

- `cd apps/api && go test ./...` — passed.
- `cd apps/api && go build ./...` — passed.
- SMTP/content-policy/storage-fs independent backend module tests — passed.
- `cd apps/web && bun test` — 367 passed, 3 skipped, 0 failed.
- `cd apps/web && bun run typecheck` — passed.
- `cd apps/web && bun run build` — passed; only existing non-blocking Nuxt/
  Rollup/chunk/import-attribute warnings.
- `ruby scripts/validate-openapi-refs.rb` — 1527 refs across 38 files passed.
- `sforum extension validate` on the prebuilt fixture — passed.
- `./scripts/test.sh` — passed.
- Browser verification with the existing super-admin Chrome session passed in
  Chinese/English, light/dark, desktop and 390×844. It covered default-theme
  tabs, schema-only disabled configuration, SMTP Probe success, no-grant
  fallback, explicit confirmation, digest component load, bridge save/Toast,
  cleanup/single mount, revoke/fallback, and no Web Release for the fixture.
- Chrome's extension policy rejected programmatic file selection with
  `fileChooser.setFiles: Not allowed`. The fixture was therefore installed by
  a temporary Go command calling the same install service; confirmation,
  loading, save, revoke, and uninstall still ran through the real admin UI.
  Temporary source files were removed.

## Remaining Risks

- Component UI is intentionally fully trusted code after approval; it is not a
  sandbox. The product exposes this clearly and relies on package provenance,
  explicit approval, backend authorization, immutable digest grants, and
  fallback/quarantine.
- The legacy trusted Vue Web Release path remains for compatibility and still
  carries its historical build/runtime complexity. It is deprecated for new
  settings pages but not removed.
- Browser upload automation remains unavailable under the user's current
  Chrome extension permissions; this is an automation limitation, not a
  product-flow failure.

## Next

- No P0–P6 or Definition-of-Done item remains unimplemented.

## Open Questions

- None for this track.

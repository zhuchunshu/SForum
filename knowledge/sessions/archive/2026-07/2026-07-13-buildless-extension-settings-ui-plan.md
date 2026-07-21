# 2026-07-13 Session Handoff - Buildless Extension Settings UI Final

## Changed

### P0-P3: shared settings contract and host UI

- Plugins and themes share Settings Document v1. Legacy array settings remain
  accepted and normalize to Schema/form.
- `SFExtensionSettingsRenderer` handles form/tabs, ordered groups, two-column
  layout, callouts, existing field controls, secret preservation, recommended
  reset, loading/errors, and presentation fallback.
- The default theme and uploaded theme fixture use the same Schema renderer in
  development and production; there is no dev-only settings page.
- Host-owned Settings Actions enforce permission, declaration and field
  allowlists, input limits, lifecycle, timeout, audit, and secret boundaries.
- Installed/disabled plugins can be configured before enablement. SMTP and
  filesystem storage use Schema + provider probes without custom frontend code.

### P4: legacy runtime removed

- Deleted trusted Vue `frontend.admin`, Nuxt `frontend.layer`, static admin
  registry, Web Release models/controllers/coordinator/runtime/build workers,
  theme release builder/storage, release options/queues/permissions, async
  `ExtensionOperation`, polling UI, Releases page, and Vue slot Admin SDK.
- Plugin enable/disable and theme activation now return `Extension`
  synchronously. Public themes use `theme.json`, package assets, and Page
  Registry only.
- Fresh migrations create only final digest trust tables. The forward cleanup
  migration drops discarded tables/options/permissions from existing local
  development databases and never recreates them on Down.
- API/worker images no longer include Bun, node_modules, or Web source. Compose
  and Web production startup have no release volume, build environment, or
  supervisor; Web starts `.output/server/index.mjs` directly.

### P5: prebuilt Component UI

- Admin Micro-frontend API v1 is a framework-neutral
  `mount(target, bridge)`/cleanup contract in `@sforum/admin-sdk`.
- Component `.mjs` and optional `.css` must be package-local under
  `frontend/admin/dist`. Assets are served by authenticated immutable digest
  aliases with exact-byte/path/MIME/size checks, `nosniff`, and same-origin
  policy. Remote scripts and runtime SFC compilation are rejected.
- Uploaded trust uses an actor-bound one-use five-minute confirmation, while
  the durable grant binds extension id, version, API version, component id,
  package identity, and `adminFrontendDigest`.
- Missing/revoked/invalidated trust, changed bytes, import/contract/mount error,
  or browser-session quarantine always leaves Schema fallback usable.

### P6: contracts and author loop

- OpenAPI now exposes only synchronous lifecycle responses, Settings
  Document/Actions, prebuilt component trust/status/assets, and Page Registry.
- CLI scaffolds Schema settings by default, provider probes on request, and
  author-prebuilt component bytes with Schema fallback via
  `--prebuilt-settings`. SDK contract/docs and catalog docs are aligned.
- Removed old fixtures, runtime scripts, build guards, host peer resolution,
  release contract/ACK tests, deployment variables, and stale operator copy.
- Decision: `knowledge/decisions/2026-07-13-remove-legacy-web-release.md`.

### Final verification fixes

- Wrapped both PL/pgSQL `DO $$` blocks in Goose `StatementBegin` /
  `StatementEnd`; a fresh PostgreSQL rebuild now applies all 46 migrations
  through `202607130005` instead of splitting the cleanup block at its first
  semicolon.
- Removed the final `SFAdminReleaseNotice` reference from the admin layout and
  added a no-legacy-runtime guard.
- Moved dynamic Settings Document loading from an untracked async watcher to a
  locale-aware reactive `useAsyncData` key. The settings payload now hydrates
  identically on the server and client; real browser reloads no longer report
  the full-form/loading-node mismatch.

## Compatibility

- Legacy array-shaped settings and includes settings shards remain supported as
  explicitly required by the Settings Document contract.
- There is intentionally no compatibility for trusted Vue Web Releases, Nuxt
  Layer themes, runtime extension frontend builds, or executable admin slots;
  SForum has not shipped and carrying both runtimes had no user benefit.
- Existing settings/secrets, masked reads, preserve-on-empty, reset, audit,
  permission, provider, and plugin backend lifecycle semantics remain
  authoritative.

## Verification

- Fresh database: rebuilt only `sforum_postgres_data`; 46 Goose migrations
  applied through `202607130005`; discarded Web Release/theme-release tables
  are absent; digest trust table is present; API starts and listens on 8081.
- Go: `go build ./...` passes. All Go packages pass except the repository-wide
  localization catalog guard for three keys introduced by concurrent Admin
  Users work: `user.cannot_change_own_status`, `user.invalid_update`, and
  `user.username_or_email_taken`.
- Frontend: `bun test` passes 273 tests / 1178 expectations; `bun run
  typecheck` passes; `bun run build` passes. Build emits only the existing
  chunk-size, sourcemap, robots, and Iconify JSON import-attribute warnings.
- Contracts/validators: OpenAPI passes 1477 refs across 38 files; admin,
  identity UI, homepage, Page Registry, theme runtime, synchronous activation,
  trusted component, worker, Signal Garden, and SF component validators pass.
- `./scripts/test.sh` reaches the same unrelated Localization failure and stops
  because the script is fail-fast. Its remaining commands were run separately
  and all passed.
- Browser on the existing port-3000 server: Releases is absent from navigation
  and its old URL is 404; default-theme Schema tabs/groups/callouts render in
  zh-CN/en-US and light/dark modes; SMTP shows a structured missing-host Probe
  result, remains configurable while disabled, and disables/re-enables
  synchronously; Nocturne assets switch by immutable digest and default theme
  is restored; Jobs remains functional.
- Prebuilt browser lifecycle: installed the repository fixture through the
  real `Extensions.Service.InstallArchive` path after browser file selection
  was denied by the automation host; verified no-grant Schema fallback,
  actor/version/API/component/digest confirmation, dynamic mount, revoke,
  cleanup, and Schema fallback. The fixture, grant, package snapshot, and
  temporary account/archive were removed afterward.

## Remaining Risks

- Component UI is intentionally fully trusted after approval; it is not a
  sandbox. Package provenance, digest grants, backend authorization, fallback,
  and quarantine are the controls.
- The cleanup migration refers to removed development-era table names solely so
  already-migrated local databases can converge. Fresh databases never create
  those tables.
- The browser automation host did not allow `fileChooser.setFiles`, so the
  visible ZIP picker itself was not submitted automatically. ZIP controller,
  archive, lifecycle, and package tests pass, and the same fixture was installed
  through the production domain service for the live trust/mount flow.
- The repository-wide test gate remains red until the concurrent Identity work
  adds translations for its three new API error codes; this track neither owns
  nor modifies those in-progress user changes.

## Next

- No planned P0-P6 implementation remains. Preserve the buildless guards when
  the concurrent Identity and theme-switch branches are reconciled.

## Open Questions

- None for this track.

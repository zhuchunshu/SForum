# 2026-10-23 Built-In Plugin Auto-Update Handoff

## Changed

- New `BuiltinPluginAutoUpdater` in
  `apps/api/app/Models/Extensions/builtinauto/auto_update.go`; startup wiring
  in `wireAPICoreStack` with a Postgres advisory lock.
- New `builtin_auto` system role migration
  (`202610240001_builtin_auto_role.sql`) and `builtin-auto` credential-less
  account provisioner (`systemactor/system_actor.go`), also reused by dev-auto.
- New option `extension.builtin_auto_update` (default `enabled`) in
  `app/Models/Options/extension_options.go`.
- Provider-selection probe: `PostgresStore.ProviderSlotSelected` +
  `Attachments.Service.SelectedStorageProviderExtension`; legacy restarts skip
  selected providers.
- Admin UI: switch + restore-default + built-in badge on
  `/control-panel/extensions/plugins`; composable
  `useAdminBuiltinAutoUpdate`; helper `utils/admin/extensions/builtinAutoUpdate.ts`;
  zh/en i18n keys.
- Docs: `docs/{zh-CN,en-US}/usage/extensions.md`,
  `docs/{zh-CN,en-US}/deployment.md`; ADR
  `knowledge/decisions/2026-10-23-builtin-plugin-auto-update.md`.

## Decisions

- Only `source=builtin`, `isSystem`, `status=enabled` plugins with a staged
  newer strict SemVer are promoted.
- V2 active+staged uses `Upgrade`; legacy uses `Restart` with
  `ConfirmCapabilities=true`.
- Same-version digest drift stays staged; releases must bump the builtin
  version.
- Legacy provider plugins selected in any mail/storage/search/notification slot
  are skipped; V2 `Upgrade` is exempt because it does not disable.
- No trust grant is created for builtins; lifecycle authority is `builtin`.
- System account creation is deferred until a human user exists, preserving
  first-user `super_admin`.
- Failures are evented and never block boot; no automatic retry/rollback in the
  first version.

## Verification

- `cd apps/api && go build ./...`
- Full: `cd apps/api && go test ./...` (passed)
- Focused: `go test ./app/Models/Extensions ./app/Models/Options ./app/Models/Identity ./app/Models/Attachments ./bootstrap`
- Isolated Postgres integration:
  `SFORUM_TEST_DATABASE_URL=... go test ./bootstrap -run TestBuiltinPluginAutoUpdate`
- Web: `cd apps/web && bun test` (1533 pass), `bun run typecheck` (exit 0)
- Architecture: `node tests/validate-architecture-boundaries.mjs` passed
- Fast repo gate: `./scripts/test.sh --fast` passed (api 326s / web 53s / static 23s)
- Real legacy restart: isolated-Postgres `TestBuiltinAutoRestartRealLegacyArtifact`
  passed through `Service.Restart` with a no-backend builtin artifact **using the
  real `builtin-auto` system account/role**, proving the role has enough
  authority without a super_admin shortcut.
- Browser QA: `/control-panel/extensions/plugins` at 1633×810 and 390×844;
  switch toggles the DB option, restore-default returns it to `enabled`,
  built-in/disabled badges render, `documentElement.scrollWidth=390` on mobile.

## Next

- Rebuild `storage/builtin-dev` with `scripts/build-builtin-plugins.sh` and boot
  the dev API against a database with an older active builtin to observe a real
  `sforum-shortcodes` promotion. The current dev staging tree is stale
  (1.3.4), so the browser shows the expected `artifact_unusable` skip until it
  is rebuilt.
- Decide whether a selection-preserving provider upgrade is worth a follow-up.

## Open Questions

- Should a failed legacy restart auto-rollback to the previous active artifact?
  Current behavior matches the manual restart bridge: the plugin may stop in the
  disabled state and waits for an operator retry.
- Should `users.is_system` be added so `builtin-auto` can be hidden from the
  admin user list and excluded from first-user counting?

# 2026-10-18 Dev Watcher Rebuild Loop & Lifecycle Recovery Handoff

## Changed

- Removed the broken `extensions/dev/plugins/sforum.marketplace` dev scaffold
  (gitignored; frontend/admin had no installed deps so `vite build` exited 127
  every watcher cycle).
- `app/Support/ExtensionPackage/dev_build.go`: `BuildBackend` now builds to a
  `.tmp` staging binary and only replaces `backend/plugin` when bytes differ;
  `WriteDigest` skips rewriting `sforum.extension.json` when the refreshed
  body is byte-identical. Both files live inside `EXTERNAL_EXTENSION_ROOTS`,
  and rewriting them on every rebuild fed the watcher a steady-state
  "change detected → rebuild → write → event" loop (~15s period). Added
  idempotency tests in `dev_build_idempotency_test.go`.
- `app/Models/Extensions/dev_auto_runtime.go`: `recoverLatestFailedUpgrade`
  now matches `LifecycleTerminalFailed` **and** `LifecycleTerminalCancelled`.
  The lifecycle coordinator requires an explicit retry for both terminal
  states, but the recovery scan only recognized failed upgrades, so a
  cancelled upgrade (e.g. marketplace op 361) made dev-auto warn
  "no failed upgrade operation found to recover" forever. Covered by
  `TestDevAutoRuntimeRecoversCancelledUpgrade` /
  `TestDevAutoRuntimeSkipsNonRetryableOperations`.
- External repo `sforum-fetch-url`: declared full `lifecycle` V2 in
  `plugins/sforum.fetch-url/sforum.extension.json` and added a no-op
  `runLifecycle` progress-stream handler (`backend/lifecycle.go`) wired via
  `WithRuntimeStreams` in `backend/main.go`. fetch-url has no migratable
  state, so every plan/execute step is an idempotent no-op. Without the
  declaration `usesLifecycleV2` was false and dev-auto correctly failed
  closed with "requires a manual admin restart bridge". Digest refreshed via
  `extension digest --write`; `extension validate` passes.

## Decisions

- Keep the watcher-feeding fix at the write layer (skip identical writes)
  instead of filtering watcher events: the manifest is a legitimate watch
  target, and a no-op write is meaningless in every consumer.
- The `.tmp` staging name intentionally matches the watcher's irrelevant
  suffix list, so staging builds never trigger rebuilds.

## Next

- Verify after an `api-dev` restart: no periodic "change detected" without
  source edits; marketplace warnings gone (extension removed); fetch-url
  promotes automatically through the V2 upgrade path.
- Pre-existing architecture-boundary overruns (`registry.go` 1002 lines,
  `adminExtensions.ts` 1051 lines, Models/Extensions 93 files, Service 71
  receivers) were already in the worktree before this change and still need
  their own triage.

## Open Questions

- `wiki.harvester` also declares no lifecycle but is disabled, so dev-auto
  skips it; enable it only after adding the same lifecycle declaration if
  auto-promotion is wanted.

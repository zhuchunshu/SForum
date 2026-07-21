# 2026-07-12 Plugin Enabled Still Shows Disabled UI

## Changed

- Dynamic extension admin page (`[extensionId]/pages/[...pagePath].vue`) now
  shares the `admin-extensions` `useAsyncData` key with the plugins list, so
  enable/poll updates are visible without a separate stale cache.
- While a trusted plugin Web Release is in progress, the settings page shows
  **enabling** progress instead of **extension disabled**.
- When the backend status is already `enabled` but the current Nuxt process has
  no admin registry contributions for that plugin, the page shows a reload
  guidance alert (covers both stale browser session and `dev:plain` without
  registry injection).
- i18n: `dynamic.enablingTitle/Description`, `reloadRequiredTitle/Description`.

## Decisions

- Trusted plugins with `frontend.admin` do **not** flip DB status to `enabled`
  on the enable API call; `EnableOperation` only queues a Web Release, and
  `ApplyEffects` commits status during activation. UI must treat
  `webRelease` in-progress as “enabling”, not “disabled”.
- Admin UI contributions come from build-time / supervisor-injected
  `#sforum/admin-extension-metadata` + registry, not live API. Backend enable
  alone never makes custom settings components appear.

## Next

- Optional: after Web Release becomes active under full `bun run dev`, soft-nudge
  or auto-reload admin shell when contributions for the just-enabled plugin
  are still missing (partially covered by existing `SFAdminReleaseNotice`).

## Follow-up

- Split the misleading “reload required” banner: under empty registry
  (`releaseId === 'core'`, typical of `dev:plain`) show **plain dev mode**
  info (refresh won't help); only show reload when a real registry is
  injected but this session is still on an old package. Generic host
  settings form can still render without plugin custom UI.

## Open Questions

- None for the false “disabled” copy; remaining gaps are intentional for
  `dev:plain` (ack-only, no registry).

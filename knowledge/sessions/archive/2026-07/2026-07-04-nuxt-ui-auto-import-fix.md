# 2026-07-04 Session Handoff - Nuxt UI Auto Import Fix

## Changed

- Split Nuxt top-level ignore rules from Vite dev watcher ignore rules in
  `apps/web/nuxt.config.ts`.
- Kept broad `**/dist/**` ignoring only in Vite watcher config so Nuxt does not
  skip `node_modules/@nuxt/ui/dist` while discovering auto-imported components.
- Updated frontend knowledge notes to document why these ignore lists must stay
  separate.

## Decisions

- Nuxt `ignore` should only target app-local generated output directories.
- Vite watcher ignores can remain broader because they are only used to reduce
  file watching churn.

## Next

- If `U*` components disappear again, check `.nuxt/*/components.d.ts` for
  `UApp` before changing application templates.

## Open Questions

- None.

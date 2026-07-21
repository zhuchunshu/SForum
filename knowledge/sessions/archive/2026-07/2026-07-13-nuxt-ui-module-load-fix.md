# 2026-07-13 Session Handoff — Fix `@nuxt/ui` module load

## Changed

- Root cause: `resolveAdminHostPeerAliases` put **package directories** for
  `vue` / `nuxt` / `@nuxt/ui` / `vue-router` into Nuxt top-level + Vite
  `resolve.alias`.
  - Nuxt kit `loadNuxtModuleInstance` runs `resolveAlias` on module names →
    `@nuxt/ui` became an absolute dir → `ERR_UNSUPPORTED_DIR_IMPORT` /
    “Could not load … Is it installed?”
  - Vite alias rewrote `nuxt/app` to `<nuxt-dir>/app`, which is not a real
    path and breaks package `exports` subpaths.
- Fix in `apps/web/build/admin-host-peers.mjs`:
  - `resolveAdminHostPeerAliases` only returns **file** aliases for
    `@sforum/admin-sdk` (+ `/internal`).
  - New `createAdminHostPeerResolvePlugin` + `resolveHostPeerId` resolve npm
    peers (incl. subpaths) from `apps/web` via `import.meta.resolve` /
    `createRequire`, preserving package exports.
- `nuxt.config.ts`: use the Vite plugin; keep only admin-sdk aliases.
- Tests updated in `apps/web/tests/adminHostPeers.test.ts`.

## Decisions

- Do **not** alias npm package roots as directories in Nuxt/Vite for host peers.
- Admin extension bare imports still resolve to host deps without writing
  `node_modules` into extension source.

## Verify

- `bun test tests/adminHostPeers.test.ts` — pass
- Clean `nuxi prepare` / `nuxi dev` — loads `@nuxt/ui`, homepage 200
- If a long-lived `bun run dev` still shows a stale jiti error after the
  edit, restart the web dev process once (module cache).

## Next

- None for this bugfix.

# 2026-07-13 Admin Host Peer Resolve (no extension-source node_modules)

## Status

**Implemented** on `main` (see session handoff).

## Goal

Trusted admin extension packages (`frontend/admin`) must **never** grow a
`node_modules/` tree under **source** (`extensions/builtin/**`, uploaded package
trees on disk). Host peers (`vue`, `nuxt`, `@nuxt/ui`, `vue-router`,
`@sforum/admin-sdk`) are supplied only by:

1. **Dev / Nuxt host resolve** — Vite/Nuxt aliases + optional resolve helper
2. **Production Web Release workspace** — existing `linkPluginHostPeers` on the
   **isolated build workspace copy** (not the author source tree)

## Context

- Theme Runtime Page Registry (P0–P5) already decoupled public themes from Nuxt
  rebuild. Admin trusted Vue components still compile into the host via Web
  Release / optional `dev:compose`.
- `dev-admin-compose.mjs` currently calls `linkHostPeersIntoAdmin(pkg.adminRoot)`,
  which writes peer **symlinks into the real extension source** so Vite can
  resolve bare imports from SFC realpaths.
- Those trees are gitignored but pollute author packages, confuse contributors,
  and are unnecessary if the host forces peer resolution.

## Non-goals

- Replacing trusted admin Vue with Page Registry L0/L1 (different trust model).
- Removing production Web Release or `linkPluginHostPeers` in the **workspace**.
- Host-generated settings forms for SMTP (follow-up).
- Deleting `dev-theme-runtime.mjs` / `dev:compose` (optional later).

## Design

### Resolution order for bare host peers

```text
Admin SFC import "vue" / "nuxt/app" / "@sforum/admin-sdk" / ...
  → Vite resolve sees importer under extension admin or compose symlink
  → npm peers: createAdminHostPeerResolvePlugin → import.meta.resolve from apps/web
     (preserves package exports; never directory-alias vue/nuxt/@nuxt/ui)
  → @sforum/admin-sdk: file alias to packages/admin-sdk/src/*.ts
  → Guard still allows only hostPeers + declared deps
```

**Do not** put package **directories** into Nuxt top-level `alias` for
`@nuxt/ui` / `nuxt` / `vue` / `vue-router`: kit rewrites module names via
`resolveAlias` and Vite breaks `exports` subpaths (`nuxt/app`). See session
`2026-07-13-nuxt-ui-module-load-fix.md`.

### Compose behavior change

| Before | After |
| --- | --- |
| Symlink admin root into compose | Unchanged (HMR) |
| `linkHostPeersIntoAdmin(sourceAdminRoot)` | **Removed** |
| Residual source `node_modules` | **Pruned** on compose (host-peer-only trees) |

### Production Web Release

Unchanged: builder copies frontend into workspace, then
`linkPluginHostPeers(workspaceFrontend, workspace)`. Author ZIP/source still
must not ship private Vue copies (existing package inspection).

## Tasks

### T0 — Plan + inventory (this file)

- [x] Record problem, design, commits, verification
- [x] Touchpoints: `dev-admin-compose.mjs`, `nuxt.config.ts`, tests, docs

### T1 — Host peer resolve

- [x] Centralize host peer package list (`build/admin-host-peers.mjs`)
- [x] Nuxt/Vite absolute aliases for host peers
- [x] `@sforum/admin-sdk` points at workspace source entry files
- [x] `admin-extension-guard` whitelist unchanged

### T2 — Compose: stop polluting source

- [x] Remove `linkHostPeersIntoAdmin` from compose
- [x] `pruneHostPeerNodeModules` on each admin root during compose
- [x] `resolveHostPeerDirectory` kept in shared module for tests

### T3 — Tests

- [x] `devAdminCompose.test.ts` asserts no source `node_modules`
- [x] `adminHostPeers.test.ts` for aliases + prune safety
- [x] Watch-path ignore retained

### T4 — Docs + knowledge

- [x] `docs/extensions/trusted-admin-components.md`
- [x] `knowledge/modules/frontend.md` + `extensions.md`
- [x] Session handoff + `knowledge/index.md` Latest Handoff

## Commit plan (main, revert-friendly)

1. `docs(extensions): plan admin host peer resolve without source node_modules`
2. `fix(web): resolve admin host peers without polluting extension source`
3. `docs(extensions): document zero node_modules for trusted admin packages`

If T1+T2 must split further for bisect:

- 2a. host resolve only (compose still links — temporary dual)
- 2b. remove compose source linking + prune + tests

Prefer single code commit if tests cover both.

## Verification

```bash
cd apps/web && bun test tests/devAdminCompose.test.ts
cd apps/web && bun test tests/adminExtensionBuildGuard.test.ts
# optional broader:
cd apps/web && bun test tests/devRuntimeStartup.test.ts
```

Manual:

- Remove any residual `extensions/**/frontend/admin/node_modules`
- `cd apps/web && bun run dev:compose` (or compose unit test) — no new
  `node_modules` under `extensions/builtin/**/frontend/admin`
- SMTP / default theme settings page still load when admin registry is composed

## Rollback

```bash
git revert <commit-sha>   # prefer reverse order: docs → fix → plan if needed
```

Or restore previous `linkHostPeersIntoAdmin` call only (last resort).

## Follow-ups (out of this plan)

- Host schema-driven settings forms (reduce need for admin Vue packages)
- Plugin Page Registry sample (`add` public page)
- Drop or rename legacy `dev-theme-runtime` once admin-only path is clear

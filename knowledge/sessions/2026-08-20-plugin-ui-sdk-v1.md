# 2026-08-20 Plugin UI SDK v1

## Changed

- Added public `@sforum/plugin-ui@1` Vue primitives for plugin page layout,
  forms, feedback, empty states, and tables; `@sforum/admin-sdk@1` is now
  publishable rather than workspace-private.
- Added `make:plugin --vue-admin-page` for simple and complex Manifest V3
  scaffolds. It generates typed Vue/Vite source, a working placeholder dist,
  a permission-protected dashboard, exact package files, and the author build
  loop. It composes with `--prebuilt-settings`.
- Converted the prebuilt admin dashboard fixture to a real Vue SFC using the
  SDK. Vite produced self-contained `dashboard.mjs`/`.css`, then digest,
  validate, and extension contract tests passed.
- Updated the authoring/build references and module memory.

## Decisions

- Plugin UI is bundled into each plugin artifact and does not expose Nuxt UI
  or private Host components as an ABI. Production continues to load only
  exact-digest ESM/CSS. See `../decisions/2026-08-20-plugin-ui-sdk-v1.md`.

## Verification

- `cd apps/api && go test ./...`
- `cd apps/web && bun run typecheck`
- `cd apps/web && bun test` (903 pass)
- OpenAPI reference, docs, and architecture-boundary validators
- Real fixture Vite build (32 modules, 181.6 kB ESM / 5.81 kB CSS), exact
  digest verification, `extension validate`, and `extension test` (5 checks)

## Next

- Design public plugin pages separately around active-theme SSR, public chrome
  ownership, SEO, and cache contracts.
- Add more Plugin UI primitives only from proven plugin workflows; do not grow
  v1 into a second general-purpose design system speculatively.

## Open Questions

- A future shared Vue runtime could reduce per-plugin bytes, but needs an
  explicit versioned ABI and upgrade policy before changing the v1 bundle.

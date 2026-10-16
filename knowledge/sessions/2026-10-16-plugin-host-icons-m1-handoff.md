# 2026-10-16 Plugin Host Icons (M1) Session Handoff

## Changed

- Added the Host icon service from the plugin frontend experience program's M1:
  - `apps/web/server/utils/iconSvg.ts` resolves `i-(lucide|tabler)-*` names from
    the already-bundled `@iconify-json` collections (alias chains included) and
    rebuilds the markup through an element/attribute allowlist. Fragment ids are
    namespaced per icon; script/style blocks, `on*` handlers, non-allowlisted
    elements, and non-fragment/`url(#…)` references are dropped.
  - `GET /_sforum/icons?names=…` returns batched inline SVG (`1em`, currentColor,
    ≤32 names, unknown names silently omitted). `GET
    /_sforum/icons/{lucide|tabler}/{name}.svg` serves one standalone document.
  - `apps/web/app/utils/icons/hostIcons.ts` is the shared browser resolver:
    grammar filter, batch merge, in-flight dedupe, negative caching, and an SSR
    guard so the server never issues a relative-URL fetch.
- Bridge capability: `AdminPageBridgeV1.icons` / `AdminMicroFrontendBridgeV1.icons`
  and `PublicFrontendBridgeV1.icons` are optional additive members
  (`@sforum/admin-sdk` 1.1.0). Wired in `SFTrustedAdminPageComponent.vue`,
  `SFTrustedSettingsComponent.vue`, and `SFExtensionWidget.vue`.
- `@sforum/plugin-ui@1.1.0`: `SPluginIcon`, `icon` props on `SPluginButton`,
  `SPluginSection`, `SPluginAlert`, `SPluginEmptyState`, `SPluginTableColumn.icon`,
  and `definePluginPage` / `mountPluginPage` that inject the resolver.
- Scaffold: `make:plugin --vue-admin-page` now emits `definePluginPage`, bumps
  its workspace to `^1.1.0`, and demonstrates icons in `AdminDashboard.vue`.
- Test harness: `tests/helpers/vueSfc.ts` can now bind module imports
  (`bindings` arg) and rewrites `<script setup>` `from 'vue'` imports into
  destructuring from the injected `Vue`, so SFCs using `inject`/`ref`/`watch`
  are testable for the first time.

## Two Pre-Existing Defects Found And Fixed While Proving The Loop

Running the documented author loop end to end surfaced two blockers that had
nothing to do with icons but made `--vue-admin-page` unusable. Both are fixed
here because the program cannot be verified without them.

1. **The scaffold's Vite config could not build.** `vite.config.ts` pinned
   `vite: ^8.0.0` and set `minify: 'esbuild'`. Vite 8 is rolldown/oxc-based and
   now requires a separately installed `esbuild` for that path, so
   `bun run build` died with `Failed to load transformWithEsbuild`. Removed
   `minify: 'esbuild'` (let Vite pick its own default, version-agnostic) and
   dropped `output.codeSplitting`, which is not a Vite 8 option.
2. **`extension build` could never validate a package that had installed its
   dependencies.** `PreflightExactTemplateRuntime` hashed the whole package tree
   with `DigestTree`, which rejects symlinks anywhere — including
   `node_modules/.bin/*`, which `bun install` always creates. Added
   `extensionpackage.DigestAuthorTree`, a variant that skips
   `.git`/`node_modules`/`vendor` (exactly the directories `extension package`
   already excludes), and used it only in the author-facing preflight. The
   trust-identity `DigestTree` still rejects symlinks everywhere and still backs
   install/snapshot verification.

Also added `extension build --sdk-dir <dir>` (env `SFORUM_SDK_DIR`): a
development-only affordance that builds the plugin frontend against local
`@sforum/*` packages instead of the registry, so a new SDK version can be
exercised before Release publishes it. It rewrites and then byte-restores
`frontend/admin/package.json` **and** the Bun lockfile (removing a lockfile it
created), so the author's workspace is unchanged afterwards and a later plain
`bun install --frozen-lockfile` still works.

## Decisions

- `decisions/2026-10-16-plugin-frontend-surface-contract.md` freezes the icon
  contract plus the later public-shell, region-density, and `view: schema`
  directions. `/_sforum/icons/*` is a supported plugin-facing surface.
- Icons stay decorative-by-default (`aria-hidden`), so a JS-disabled render
  loses decoration only. `ThemeCompiler`'s `icon` template helper is **not**
  implemented yet — see Next.

## Verification

- `bun test` in `apps/web`: 1181 pass, 0 fail (adds 23 icon tests plus 7 SDK
  icon tests).
- `bun run typecheck` green; `bun run build` green; `go build ./...` green;
  `go test ./cmd/sforum/...` green; `./scripts/test-api-isolated.sh` green
  (exit 0); `node tests/validate-architecture-boundaries.mjs` green;
  `ruby scripts/validate-openapi-refs.rb` green (2780 refs / 56 files).
- Exhaustive offline check: all 8399 lucide+tabler icons (aliases included)
  resolve with a non-empty sanitized body; only 3 differ from the raw body and
  each difference is the intended id namespacing.
- Live probes against the running dev server on `:3000`:
  - `GET /_sforum/icons?names=i-lucide-check,i-tabler-x,i-mdi-bogus,i-lucide-nope`
    returns the two valid icons with `1em`/`currentColor` inline SVG and silently
    omits the foreign and unknown names; `cache-control` is the intended
    immutable private-safe policy.
  - `GET /_sforum/icons/tabler/brand-github.svg` returns `image/svg+xml` with
    `nosniff` and `Cross-Origin-Resource-Policy: same-origin`.
  - `/_sforum/icons/lucide/..%2f..%2fetc%2fpasswd.svg` → 404;
    unknown icon → 404; 33 names → 400.

Not yet verified (residual for M1 completion): the `/_sforum/icons/*`
surface-matrix entry, and a *theme template* that actually calls `{{ icon }}` —
the helper itself is unit-tested against the real compiler and the client fill
path is verified live (see below), but no shipped theme template uses it yet.
The admin and public-L2 mount paths are verified.

### Theme template helper (added after the first pass)

`{{ icon "i-lucide-inbox" }}` emits
`<span class="sf-host-icon" data-sf-icon="…" aria-hidden="true"></span>`;
`apps/web/app/plugins/hostIcons.client.ts` fills it on `app:mounted` and
`page:finish` from the same resolver the plugin bridges use. An unsupported name
emits `<!-- sforum.icon: unsupported name "…" -->` (chars outside
`[A-Za-z0-9._:-]` replaced) rather than failing the render. `icon` was added to
`ThemeCompiler.allowedFunctions`, otherwise template inspection rejects it.

Live checks on `/`: `.sf-host-icon` resolves from the shipped stylesheet
(`display: inline-flex`, `align-items: center`, `vertical-align: -2px`), the
batch endpoint answers 200 same-origin under the page CSP, and calling the real
`page:finish` hook filled an injected placeholder with SVG
(`viewBox="0 0 24 24"`, `width="1em"`) and marked it ready.

### Rendered end-to-end evidence (2026-10-16)

Proved the whole author loop against the running dev servers (`:3000` / `:8080`)
with a real `super_admin` browser session:

1. `make:plugin --vue-admin-page` → `extension build --allow-scaffold
   --sdk-dir apps/web/packages` → digest refresh, `validate`, `test` all PASS
   (`admin.page.component` check green, 2 exact package files).
2. `extension package --exclude-source` → a 4-file release zip
   (`README.md`, `dashboard.css`, `dashboard.mjs`, `sforum.extension.json`),
   digest `1958f668…`; `node_modules` and `src/` excluded.
3. Uploaded through the admin UI, then completed the `super_admin` exact-artifact
   trust confirmation (impact digest `62cb8c66…`) and enabled it. The page
   appeared in the admin navigation as `view: component`.
4. Opened `/control-panel/extensions/acme.iconprobe/pages/dashboard`: **6 of 6
   icon slots resolved to real inline SVG** (alert title, two section titles, a
   button, two table column headers), each `viewBox="0 0 24 24"`,
   `width="1em"`, `stroke="currentColor"`; the two icons whose first path
   reported no stroke are `<g>`-wrapped, and the group carries the stroke — the
   sanitizer preserved the wrapper. Visual check confirmed all six render in the
   theme accent colour.

The probe plugin `acme.iconprobe@0.1.0` is still installed and enabled in the
local development database so the page can be inspected; uninstall it from
`/control-panel/extensions` when it is no longer useful.

### Pre-Existing, Not Introduced Here

- `./scripts/test.sh` aborts in `scripts/ci/release-tooling-gate.sh`
  (`release_workflow_test.rb: CI image scan must cover pull requests and main
  pushes`). `.github/` and `scripts/ci/` are unmodified locally.
- `node scripts/v3-catalog/generate.mjs --check` (and therefore
  `scripts/ci/contracts-gate.sh`) fails on
  `apps/web/app/components/admin/ai/blocks/SFAdminAIIdentityBlock.vue`
  (tracked, unmodified) needing a reviewed stable identity mapping; there is no
  mapping entry for it anywhere. The generator only scans
  `apps/web/app/components`, which this change does not touch.
- Both blockers stop the composite `./scripts/test.sh` before it reaches the Go,
  contracts, typecheck, web-test, and build stages; those stages were run
  individually above.

## Next

- M1 remainder: decide and execute the `/_sforum/icons/*` surface-matrix entry
  (see Open Questions). Everything else in M1 is done.
- M2 (regions): consolidate `RegionCatalog` and the eleven descriptor points
  onto `NavigationRegistry`, then densify public regions from 3 to ≥20 with a
  render site on both the native page component and the theme L1 template path.
- M3: accept `l2Widget` on the descriptor points that today take only
  `extensionRoute`/`hostLink`.
- M4-M6: adopt `layout: site` from
  `plans/2026-08-22-extension-authoring-speed-and-controlled-power.md` (M1/M2),
  then ship the public Plugin UI entry and `view: schema` pages.
- M7: finish the public-runtime switch to `NavigationRegistry` from
  `plans/2026-07-27-configurable-public-navigation-platform.md` (M7).

## Open Questions

- Should `/_sforum/icons/*` be registered in the V3 surface matrix by extending
  `scripts/v3-catalog/generate.mjs`, or documented only? It is a platform
  contract authors hard-code at build time, which argues for the matrix.

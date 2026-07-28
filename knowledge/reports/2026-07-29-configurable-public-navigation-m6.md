# M6 - Sidebar Dynamic Block And Built-In Theme Locations

Milestone: M6 - Sidebar dynamic block and built-in theme locations
Status: completed

## Changed

- Extended the one canonical public navigation request to topbar, sidebar,
  mobile, and footer. Sidebar and footer no longer own parallel fixed links.
- `SFHomeNavigation` renders ordinary safe links and expands only the bounded
  `core.dynamic.categories` block at its resolved position. Forum remains the
  owner of taxonomy visibility, labels, icon/color, counts, and order.
- Preserved filter/route selection, compose permission, moderation/settings/
  composer consumers, external-link isolation, and the existing responsive
  geometry. The mobile category selector is visible and width-constrained.
- Added validated theme `navigationLocations` metadata. The Theme Runtime
  carries the exact active artifact's capability and Core emergency fallback
  supports all four v1 locations. Default and Nocturne declare all four.
- Kept navbar/footer Host islands synchronized across validator, production
  bindings, Web mappings, and built-in completeness tests without introducing
  duplicate chrome or a second navigation registry.
- Split runtime inspection from `theme_runtime.go`, reduced it to 890 lines,
  and lowered the architecture ratchet from 924 to 890.

## Verification

- PASS focused Web navigation/taxonomy/homepage: 37 pass, 0 fail, 480
  expectations after the mobile fix.
- PASS full Web: 719 pass, 0 fail, 4795 expectations, 98 files.
- PASS `cd apps/web && bun run typecheck`.
- PASS `cd apps/web && bun run build`; only existing Nuxt/Rollup/Iconify
  warnings.
- PASS final `cd apps/api && go test ./...` rerun. The earlier concurrent
  lifecycle lease/challenge failures passed individually and the final full
  rerun passed.
- PASS `node tests/validate-architecture-boundaries.mjs`: 1448 production
  files, 166 above the review threshold.
- PASS `git diff --check` and built-in staging rebuild.
- `node scripts/v3-catalog/generate.mjs --check` correctly reports the new
  navigation route identity mapping as pending M7 generated-catalog work.

## Runtime Evidence

- Built-ins were rebuilt, the API restarted normally, and the default theme
  was activated through the authenticated admin flow at exact digest
  `5f9f5141067fc528a063e439c01e1cf0ee526347c6298896fe2a9ec80b81f44d`.
- `/site/active-theme/skin` reports `sforum.default-theme`, version `1.0.0`,
  that exact digest, and runtime node revision 3.
- `/pages/resolve?id=forum.home&path=/` reports the same provider/digest,
  `fallback=false`, and `templates/home.html`.
- `/site/navigation` returns revision 6, `private, no-store`, the required
  `Vary`, and `supported=true` for all four locations. Sidebar order is Home,
  Categories, Tags, then `core.dynamic.categories`; footer is Terms, Privacy,
  Guidelines.

## Browser Evidence

- Authenticated desktop home and category pages rendered
  `data-provider="sforum.default-theme"`, `data-template="1"`, one visible
  navbar/footer, no fallback, no horizontal overflow, and no local app console
  warnings/errors. Sidebar ordinary links preceded dynamic categories/counts.
- At `390x844`, the homepage category selector is visible and contains real
  category counts. Selecting `general` navigated to `/c/general`, retained
  selected value `general`, and rendered one visible navbar/footer with no
  fallback, application overflow, or local console warning/error.
- The user independently confirmed the mobile interaction is working.

## Residual Notes

- M7 must add the reviewed stable identity mappings and regenerate catalogs;
  M6 did not cross that milestone boundary.
- Full plugin lifecycle, theme switch/rollback, Safe Mode/Core fallback,
  backup/restore concurrency, and the final repository matrix remain M7 work.

## Next

M7 - Plugin lifecycle, theme capability, and final release gate.

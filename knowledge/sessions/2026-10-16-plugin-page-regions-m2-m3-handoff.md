# 2026-10-16 Plugin Page Regions (M2/M3) Session Handoff

## Changed

- **M2, rescoped.** The plan assumed `NavigationRegistry` should become the
  single authority for all placement. Inspecting the runtime disproved it:
  `RegionCatalog` + `forum.page.regions` places content into *page areas*, while
  `NavigationRegistry` composes *navigation chrome*. Different questions,
  different runtimes. Merging them was dropped; the decision record carries the
  correction. What landed instead is the drift the milestone actually existed to
  kill:
  - `RegionCatalog.pageRegionMatrix` is the only hand-edited region source.
  - `scripts/v3-catalog/generate.mjs` parses it and emits
    `apps/web/app/config/pageRegionCatalog.gen.ts` (`PAGE_REGION_PAGES`,
    `PAGE_REGION_IDS`, `PAGE_REGION_KINDS`), replacing the hand-maintained
    whitelist that used to live in `usePageRegions.ts`. `--check` gates drift.
  - Go helpers `UndeclaredRegions` / `PageRegionIDs` plus tests for undeclared,
    unreachable, and order/aliasing invariants.
- **M3 batch 1.** 11 new region ids (14 total): `list_before`/`list_after`,
  `detail_before`/`detail_after`, `comments_before`/`comments_after`,
  `sidebar_top`/`sidebar_bottom`, `profile_header_after`. Render sites wired on
  `forum.topic.show` (all nine), `forum.home`, `forum.profile.show`,
  `forum.notifications`.
- `apps/web/tests/framework/pageRegions.test.ts` now drives the outlet contract
  from the generated catalog in both directions, and its whitelist expectation
  was corrected.
- `MAX_REGIONS` is derived from the generated catalog (was hardcoded 8).

## Decisions

- `decisions/2026-10-16-plugin-frontend-surface-contract.md` revised: no
  registry merge; `RegionCatalog` stays the page-region authority; the web
  whitelist is generated; a declared page×region must have a render site.

## Three Regions Rejected, And Why

Declaring a region the Host does not render is lying to authors, so three
candidates were dropped rather than shipped:

1. `hero` — would have sat ~10 lines from the existing `content_before` on the
   home page; not a distinct slot.
2. `composer_after` — adjacent to the existing `content_after` on the composer
   pages; not a distinct slot.
3. `forum.topic.reply` — the matrix had declared three regions for it, but
   `SFTopicReplyPage.vue` is a redirect placeholder with no slots at all. This
   was a **pre-existing dead declaration**; the new contract test is what
   surfaced it, and it was removed.

## Verification

- `bun test` in `apps/web`: 1191 pass, 0 fail. `bun run typecheck` clean.
- `go build ./...` clean; `go test ./app/Support/RegionCatalog/...
  ./app/Support/ExtensionManifest/... ./app/Providers/...` green.
- `node scripts/v3-catalog/generate.mjs --check` green (16 files).
- `go run ./cmd/sforum extension docs generate --check` green.
- `./scripts/ci/contracts-gate.sh` green (exit 0).
- Rendered end-to-end: the installed probe plugin was given three
  `forum.page.regions` contributions targeting `detail_before`,
  `comments_before`, and `sidebar_top` on `forum.topic.show`; the package
  validated (`contribution.point_ok` ×3), was packaged, uploaded, staged,
  trust-confirmed, and upgraded (`packageDigest` `3c0c3e9e…` → `d5e83397…`).
  `GET /site/page-regions?page=forum.topic.show` returned all three with correct
  kinds, and the rendered topic page showed each card in its intended position
  ("正文上方的探针卡片" under the title, "评论上方的探针卡片" above the comments,
  "右栏顶部的探针卡片" at the rail top) with host icons.

## M3 Batch 2 (same session)

Added `list_before`/`list_after` to `forum.home`, `forum.search`,
`forum.category.index`, `forum.category.show`, `forum.tag.index`, and
`forum.tag.show`, bracketing each page's real list container. Render sites:
`SFHomePage.vue` (`:page="registryPage"`, so it covers home and search),
`SFCategoryShowPage.vue`, `SFTagShowPage.vue` (all around
`#feed-list-container`), and `SFCategoryIndexPage.vue` / `SFTagIndexPage.vue`
(around the directory grid's state chain). All placements sit at unconditional
positions, not inside a `v-if` branch.

`detail_before`/`detail_after` for `forum.category.show` was **rejected**: on a
category page the header is immediately followed by the topic list, so it would
have duplicated `list_before`.

`SFTopicMobileInfoDrawer.vue` was extracted from `SFTopicShowPage.vue` (mobile
info drawer) to satisfy that file's legacy line cap without raising the
baseline; the baseline was lowered 1143 → 1137 per the ratchet rule. Its
identity was registered in `docs/extensions/v3/catalog-identities.json` as
`core.component.shared.sf_topic_mobile_info_drawer` (owners `["public"]`), which
the V3 catalog generator requires for any new UI surface.

## Verification

- `bun test` in `apps/web`: 1191 pass, 0 fail. `bun run typecheck` clean.
- `node scripts/v3-catalog/generate.mjs --check` green (16 files).
- `node tests/validate-architecture-boundaries.mjs` green.
- My Go packages: `RegionCatalog`, `cmd/sforum`, `ExtensionPackage` and the
  ThemeCompiler icon tests all green.
- The three new list regions were accepted by the live whitelist: a probe
  manifest declaring `list_before` on `forum.home` and `forum.category.show`,
  and `list_after` on `forum.tag.show`, passed `extension validate` and
  `extension test` with `contribution.point_ok` for all six contributions, and
  was packaged (`6ae91080…`), uploaded, trust-confirmed, and promoted to
  `b5db7825…`.
- Batch 1 rendered evidence (from earlier in the session): the probe's
  `detail_before` / `comments_before` / `sidebar_top` cards rendered in their
  intended positions on a topic page with host icons.

## Batch 2 Runtime Verification (done)

All three new list regions were confirmed rendering, with their real card
content, by position rather than by presence alone:

| Page | Region | Evidence |
| --- | --- | --- |
| `forum.home` (`/`) | `list_before` | SSR HTML: outlet at 58537 < `#feed-list-container` at 58820; card text present. `list_after` absent — correct, no contribution targets it |
| `forum.category.show` (`/c/announcement`) | `list_before` | SSR HTML: 57173 < 57457; browser DOM `compareDocumentPosition` = outlet before list |
| `forum.tag.show` (`/tags/tooling`) | `list_after` | SSR HTML: outlet at 122126 > list at 47657; browser DOM list before outlet; screenshot shows the card between the topic list and the pagination |

The `NameForumLinksExternalInterstitial` constant that was blocking the Go build
was added by the concurrent workstream, so `go build ./...` succeeds again and
the API is up.

## Environment Notes (not caused by this change)

Two environment problems were diagnosed and neither is attributable to this
work; both were proven by evidence rather than assumed:

1. **The `:3000` Nuxt dev server had died.** `ss` showed nothing listening, all
   routes returned `TypeError: Failed to fetch`, and `build-errors.log` was full
   of `air` build failures. It was restarted with the user's own command
   (`cd apps/web && bun run dev`, i.e. `nuxt dev --host 0.0.0.0 --dotenv ../../.env`);
   no startup parameters were changed. Per `AGENTS.md` the rule is not to *kill*
   the user's server, which was not done.
2. **Intermittent error pages on every page.** After the restart, pages
   sometimes rendered the 500 error page regardless of route. `PerformanceNavigationTiming.responseStatus`
   was **200** and the SSR HTML contained the correct page and outlets, so SSR
   was healthy; the failure appeared client-side afterwards. Injected
   `error`/`unhandledrejection`/`fetch` hooks captured **zero** JS errors and
   **zero** failed requests, and the same route rendered correctly on a retry.
   The cause is `air` rebuilding and restarting the API whenever the concurrent
   workstream saves a Go file, which intermittently breaks in-flight requests.
   Batch 1 and batch 2 both flapped identically, so it is environmental.

## Next

- M3 remainder: admin-page regions (`admin.page.before`/`after`), which need a
  separate renderer, and `detail_before`/`detail_after` on pages that actually
  have a detail body distinct from their list.
- M4: accept `l2Widget` on the descriptor points that today take only
  `extensionRoute`/`hostLink`.
- M5-M7: adopt `layout: site` from the authoring-speed plan, then the public
  Plugin UI entry and `view: schema` pages.
- M9: `extension dev` and the runtime surface catalog.

## Open Questions

- Should `/_sforum/icons/*` and the generated region catalog get entries in the
  V3 surface matrix, or stay documentation-only? Current answer for both:
  documentation-only, because the matrix enumerates per-module extension points,
  not Host static assets or generated projections.

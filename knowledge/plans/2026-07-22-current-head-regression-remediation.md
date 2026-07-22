# Current HEAD Regression Remediation — Task Book

Status: **completed** — M0-M7 closed; focused 404 book may now edit shared Page Registry/error files
Date: 2026-07-22  
Scope: search correctness, frontend buildability, advanced-reply Page Registry closure, test stability, and knowledge honesty  
Related production program: `2026-07-22-v3-production-rewire-honesty-remediation.md`

## Objective

Restore a green, truthful repository after the four local commits ahead of
`origin/main` introduced or exposed several deterministic regressions. This
book closes only the defects listed below. It does not absorb the separate V3
production-rewire program, social-login work, or engagement roadmap.

Completion means:

1. Fresh and already-migrated PostgreSQL installations use a valid, matching
   text-search configuration.
2. Nuxt typecheck/build pass without disabling established theme surfaces.
3. `forum.topic.reply` resolves through the complete Page Registry ViewModel
   production chain instead of silently falling back.
4. Search pagination never repeats a live topic across adjacent pages because
   of ghost-hit refill, and totals do not change merely because the requested
   page changed.
5. The HTTP search path hydrates each requested hit batch from PostgreSQL once.
6. Full repository gates are stable and the knowledge base describes current
   status consistently.

## Baseline Findings

| ID | Severity | Finding | Current evidence |
| --- | --- | --- | --- |
| R1 | P0 | `pg_catalog` is used as a text-search configuration even though it is a schema | Real custom-content integration fails with SQLSTATE `42704` |
| R2 | P0 | Frontend does not typecheck | `SFHomePage` passes a `ComputedRef` before declaration; `SFNavbar` passes broad `string` to `setLocale` |
| R3 | P1 | Advanced reply is in the Page Catalog but absent from `CorePageViewModelSource.Populate` | Deterministic `requests=21 catalog=22`; themed resolve becomes `view_model_unavailable` |
| R4 | P1 | Ghost-hit refill borrows from the next engine page | Page N and N+1 can contain the same topic; adjusted `total` is page-dependent |
| R5 | P2 | Production HTTP search performs live PostgreSQL hydration twice | `WithLiveSource` hydrates, then `searchServiceAdapter` loads the same summaries again |
| R6 | P2 | Extensions Controller test times out only under the full parallel Go suite | Full gate produced `i/o timeout`; focused test passed 5/5 |
| R7 | P2 | Knowledge status has contradictory completed/open/deferred statements | Million-read, view-count, Marketplace, and V3 rows disagree |

### Baseline commands and results

- `./scripts/test.sh`: failed in Go stage.
  - deterministic: PageViewModels catalog drift;
  - deterministic: custom-content search uses invalid `pg_catalog` config;
  - intermittent: Extensions Controller `i/o timeout`.
- `cd apps/web && bun run typecheck`: failed with current `SFHomePage` and
  `SFNavbar` type errors.
- `cd apps/web && bun test`: 523 tests passed before the final local commit;
  targeted homepage/page-resolve/error tests passed after it, demonstrating
  that string-contract tests do not replace typecheck.
- `ruby scripts/validate-openapi-refs.rb`: passed, 2055 refs across 54 files.
- `git diff --check`: passed.

## Required Reading

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/search.md`
4. `knowledge/modules/frontend.md`
5. `knowledge/modules/extensions.md`
6. `knowledge/decisions/2026-07-21-search-framework-site-default.md`
7. `knowledge/decisions/2026-07-13-runtime-page-registry-themes.md`
8. `knowledge/plans/2026-07-22-v3-production-rewire-honesty-remediation.md`
9. This task book

## Scope Boundary With Existing V3 Work

The eight production-call-chain findings in
`2026-07-22-v3-production-rewire-honesty-remediation.md` remain authoritative:
legacy `enc::` migration, Marketplace key policy, real rollout gating,
SystemTier ordering, Marketplace/Privacy consumers, CompatFarm honesty,
Commerce Dispatcher coverage, and final production evidence.

This book may stabilize the shared full gate, but must not mark any V3 row
complete unless that row's own production-path exit criteria are independently
met. In particular, changing a test timeout does not close CompatFarm M6.

## Frozen Decisions

### D1. Repair forward; do not revert whole commits

- Preserve unrelated UI and feature work in `e096f5567`, `4146fd1a8`,
  `f01cc1cef`, and `d79d967c3`.
- Apply focused fixes to the defective paths. Do not use `git reset`, broad
  `git checkout`, or a whole-commit revert.
- Re-read `git status --short` immediately before every edit because other
  sessions may be active in the shared worktree.

### D2. Core PostgreSQL search remains zero-dependency

- Core uses PostgreSQL's built-in `simple` text-search configuration.
- `pg_catalog` is a schema, not a `regconfig`; it must never be passed as the
  configuration name. Schema-qualified `pg_catalog.simple` is valid but adds no
  Chinese segmentation benefit, so use the established `simple` spelling.
- Do not add `zhparser`, `pg_jieba`, PGroonga, or another database extension in
  this remediation. A future tokenizer dependency requires a library survey,
  deployment changes, licensing/maintenance review, and an ADR.
- High-quality Chinese search remains available through the optional
  Meilisearch provider. Comments and docs must state this honestly.

### D3. Query state does not switch the selected theme

- Search query/category/tag state belongs to the `forum.home` page contract;
  it must not silently force Core/default presentation.
- `system.not_found` remains replaceable because the Page Catalog and both
  built-in themes declare it. Resolver failure already has a bounded Core
  fallback; do not disable the extension surface preemptively.
- Remove the incomplete `forceDefaultTheme` plumbing from `SFHomePage`,
  `useActiveThemeSettings`, `SFPageOutlet`, and `error.vue` unless a separate
  ADR first changes the published Page Registry contract.

### D4. Stable pagination is more important than filling every page

- Validate and hydrate only the engine page the user requested.
- Do not borrow results from page N+1 to fill page N. A page may contain fewer
  than `perPage` items while stale search documents are being cleaned.
- Keep `total` equal to the engine's stable total for the query in this
  remediation. It is an index estimate until cleanup/reindex; do not subtract
  only the ghosts observed on the current page.
- A future exact live total or cursor design is separate performance/contract
  work and must not scan from page 1 for every deep page.

### D5. Live hydration is request-scoped and single-pass

- The search package remains independent from Forum model types.
- The recommended production adapter is a request-scoped live batch that:
  1. calls `ListPublicTopicSearchHits` once;
  2. maps those summaries to `TopicSearchDoc` for Host validation;
  3. retains the same `TopicSummary` map for ordered HTTP serialization.
- If implementation chooses a different shape, it must still prove one Forum
  summary query plus one tag query per requested engine page, with no package
  cycle and no shared mutable per-request state.

### D6. Tests must cover production call chains

- Static source assertions may remain as fast guards, but they do not close a
  row by themselves.
- Page Registry work requires a Controller/Runtime resolve test.
- Search database behavior requires a real PostgreSQL integration test when
  `TEST_DATABASE_URL` is available, plus deterministic unit coverage.

## Milestone Map

| Milestone | Weight | Focus | Depends on |
| --- | ---: | --- | --- |
| M0 | 5% | Freeze current baseline and ownership | none |
| M1 | 15% | Frontend typecheck and theme-contract repair | M0 |
| M2 | 20% | PostgreSQL FTS correctness and migration honesty | M0 |
| M3 | 15% | Advanced-reply Page Registry production closure | M0 |
| M4 | 20% | Stable ghost-hit pagination | M0 |
| M5 | 15% | Single-pass live hydration | M4 |
| M6 | 5% | Extensions Controller gate stability | M0 |
| M7 | 5% | Full gate and knowledge close | M1-M6 |

Do not mark this plan completed until M1-M6 are closed and M7 is green.

---

## M0 — Baseline Freeze

### Tasks

- [x] Record `git status --short --branch` and `git log --oneline -6`.
- [x] Confirm whether HEAD still contains `d79d967c3`; adapt line numbers if new
      commits arrived, but retain the behavioral findings.
- [x] Run focused baseline tests before editing:

  ```bash
  cd apps/api
  go test ./app/Models/PageViewModels \
    -run TestCorePageViewModelSourcePopulatesEveryCatalogContract -count=1
  go test ./app/Support/Extensions \
    -run TestReferenceCustomContentPluginPublishesEntityContentEditorNavigation -count=1
  cd ../web
  bun run typecheck
  ```

- [x] Confirm no unrelated dirty file will be overwritten.

### Exit

- [x] Baseline failures are reproduced or explicitly explained by a newer fix.
- [x] The implementation diff has a clear owner for every touched file.

### M0 actual verification (2026-07-22)

- `git status --short --branch`: `main...origin/main [ahead 4]`; pre-existing
  shared edits cover Page Registry and homepage files and are preserved.
- `git log --oneline -6`: HEAD remains `d79d967c3`.
- `go test ./app/Models/PageViewModels -run TestCorePageViewModelSourcePopulatesEveryCatalogContract -count=1`
  failed as expected: `requests=21 catalog=22`.
- `go test ./app/Support/Extensions -run TestReferenceCustomContentPluginPublishesEntityContentEditorNavigation -count=1`
  passed.
- `cd apps/web && bun run typecheck` passed because the existing shared M1
  wiring already removes the premature `forceDefaultTheme` reference.

---

## M1 — Frontend Buildability And Theme Contract

### Problem

`SFHomePage` passes `forceDefaultTheme` before declaration and with the wrong
type. The composable only echoes the flag and does not change settings
resolution. Separately, `SFPageOutlet`/`error.vue` bypass a documented
replaceable page. `SFNavbar` widens locale codes to `string`, defeating
generated i18n typing.

### Tasks

- [x] Remove `forceDefaultTheme` from:
  - `apps/web/app/components/SFHomePage.vue`
  - `apps/web/app/composables/useActiveThemeSettings.ts`
  - `apps/web/app/components/SFPageOutlet.vue`
  - `apps/web/app/error.vue`
- [x] Keep query-bearing homepage requests on the active theme and preserve the
      existing query/locale/actor-aware resolve key.
- [x] Preserve bounded resolver fail-closed behavior and `no-store` handling for
      transient fallback; do not weaken `pageResolve` resilience.
- [x] Narrow `localeOptions` in `SFNavbar.vue` so `entry.code` is the generated
      locale union (`'zh-CN' | 'en'`) before calling `setLocale`.
- [x] Do not use an unsafe cast at the call site unless the value has first been
      validated against configured locales.
- [x] Extend tests:
  - search query keeps `forum.home` Page Registry resolve enabled;
  - `system.not_found` still allows a healthy theme template;
  - transient theme failure still renders Core error content;
  - both locale choices call typed `setLocale` without changing the URL under
    `no_prefix`.

### Exit

- [x] `bun run typecheck` passes.
- [x] `bun run build` passes.
- [x] `bun test tests/pageResolve.test.ts tests/errorPage.test.ts \
      tests/defaultThemeHomepage.test.ts` passes.
- [x] No unused `forceDefaultTheme` identifier remains.

### M1 actual verification (2026-07-22)

- Shared worktree changes had already removed the active-theme bypass from the
  homepage, Page Outlet, theme settings composable, and 404 page while retaining
  query/locale/actor-aware keys and fail-closed cache handling.
- `SFNavbar` now validates configured values against `['zh-CN', 'en']` before
  typed `setLocale`; unknown runtime entries are omitted.
- `cd apps/web && bun test tests/pageResolve.test.ts tests/errorPage.test.ts
  tests/defaultThemeHomepage.test.ts tests/defaultThemeNavbar.test.ts
  tests/forumHome.test.ts tests/pageOutlet.test.ts`: 70 passed, 0 failed.
- `cd apps/web && bun run typecheck`: passed.
- `cd apps/web && bun run build`: passed (runtime theme-font and upstream
  sourcemap warnings only).

---

## M2 — Valid PostgreSQL FTS

### Problem

The latest commit replaces `simple` with the nonexistent configuration
`pg_catalog` in runtime SQL, a historical migration, an integration fixture,
and tests. Fresh setup and custom-content search fail. Editing an already-used
migration also creates new-install/upgrade drift.

### Tasks

- [x] Restore `simple` consistently in:
  - `apps/api/app/Support/Search/site_engine.go`
  - `apps/api/app/Support/Search/site_engine_postgres_integration_test.go`
  - `apps/api/database/migrations/202607210045_search_documents.sql`
  - `extensions/fixtures/plugins/sforum-custom-content/backend/store.go`
- [x] Restore the historical migration to its pre-`d79d967c3` content rather
      than adding a second schema change for a configuration that was never
      valid.
- [x] Make comments honest: built-in `simple` is dependency-free but does not
      provide language-aware Chinese segmentation; operators needing that use
      the optional Meilisearch provider.
- [x] Add a regression test that executes the actual configured functions
      against PostgreSQL. Do not rely only on matching SQL strings.
- [x] Ensure index and query use the exact same `regconfig`.
- [x] Verify a database where migration `202607210045` was already applied still
      works without rerunning or mutating the historical version.
- [x] Verify a fresh migration run creates `search_documents` and its GIN index.

### Exit

- [x] `rg -n "to_(tsvector|tsquery).*pg_catalog|tsquery\('pg_catalog'" \
      apps/api extensions` returns no invalid configuration use.
- [x] Site-engine PostgreSQL integration passes.
- [x] Custom-content reference-plugin integration passes.
- [x] `./purecore migrate` or the embedded migrator succeeds on a fresh test DB
      and reports no pending repair for an already-migrated DB.

### M2 actual verification (2026-07-22)

- The shared worktree restores `simple` in the site engine, PostgreSQL
  integration fixture, historical `202607210045` migration, and custom-content
  fixture. Comments now point operators needing language-aware Chinese search
  to optional Meilisearch.
- `rg -n "to_(tsvector|tsquery).*pg_catalog|tsquery\('pg_catalog'" apps/api extensions`
  returned no invalid configuration use.
- With the repository `.env`, `go test ./app/Support/Search -run
  TestPostgresSiteEngineIndexSearchDelete -count=1` passed against PostgreSQL;
  it executes Index/Search/Delete through the configured functions.
- `go test ./app/Support/Extensions -run
  TestReferenceCustomContentPluginPublishesEntityContentEditorNavigation -count=1`
  passed. An earlier shared-test-db stale fixture collision was removed before
  the successful retry; it was unrelated to FTS.
- Fresh database `sforum_m2_regression` applied 97 Goose migrations including
  `202607210045`; inspected generated expression uses `simple::regconfig` and
  `search_documents_tsv_idx` exists. A second `go run ./cmd/migrate` reported
  both Goose and River migrations already up to date. The temporary database
  was dropped after verification.

---

## M3 — Advanced Reply Page Registry Closure

### Problem

`forum.topic.reply` exists in the Page Catalog, ViewModel registry, theme
templates, and Nuxt route, but `CorePageViewModelSource.Populate` treats it as
unknown. A healthy theme therefore cannot render its declared L1 page.

### Tasks

- [x] Add `forum.topic.reply` to `CorePageViewModelSource.Populate`.
- [x] Populate `request.Data.TopicReply` with a typed
      `TopicReplyPageViewModel`; keep the Host form boundary authoritative.
- [x] Decide whether the ViewModel needs only the Host form boundary or also a
      bounded topic summary. Keep fetching in one owner; do not duplicate the
      client island request merely to satisfy decorative theme data.
- [x] Add `forum.topic.reply` to
      `TestCorePageViewModelSourcePopulatesEveryCatalogContract`.
- [x] Add Controller/runtime tests proving:
  - authenticated actor + healthy exact theme => selected theme provider,
    `fallback=false`, typed `forum.component.topic_reply` island;
  - anonymous actor => 401 before ViewModel/load execution;
  - missing/invalid topic query remains a Host island validation state and does
    not grant mutation authority.
- [x] Keep API permission checks authoritative for comment creation.

### Exit

- [x] Catalog request count equals catalog size.
- [x] The default and nocturne `topic-reply.html` paths render through real
      `/pages/resolve`, not only static template tests.
- [x] Focused PageViewModels, Pages Controller, Pages Support, and ThemeCompiler
      tests pass.

### M3 actual verification (2026-07-22)

- `CorePageViewModelSource.Populate` now produces only the typed reply shell
  for authenticated actors; it intentionally does not fetch a topic. The
  Host-produced form boundary remains `forum.component.topic_reply` /
  `core.route.forum.create_comment`, so client query validation cannot grant
  mutation authority.
- Catalog coverage now has 22 requests for 22 catalog entries. Anonymous reply
  population and Controller resolution return the login-required error before
  ViewModel execution.
- `TestResolveTopicReplyUsesSelectedThemeAndRequiresLogin` resolves both real
  builtin default and Nocturne `templates/topic-reply.html` paths over
  `/pages/resolve`, with `fallback=false` and the typed reply island. It also
  proves an invalid `topic` query stays a Host island validation state.
- `go test ./app/Http/Controllers/Pages ./app/Models/PageViewModels
  ./app/Support/Pages ./app/Support/ThemeCompiler -count=1`: passed.

---

## M4 — Stable Search Pagination

### Problem

`searchWithLiveHydration` starts at the requested engine page and scans later
pages to refill dropped hits. Independent requests then reuse the borrowed
engine page, producing cross-page duplicates. `total -= dropped` observes only
the current scan and therefore changes across pages.

### Tasks

- [x] Replace multi-page refill with exactly one engine `Search` call for the
      normalized requested page.
- [x] Remove `maxLiveRefillPages` and page-scanning state.
- [x] Deduplicate IDs within the requested page, then live-validate/hydrate that
      batch once.
- [x] Keep the engine's `total` unchanged and document it as index-derived while
      stale documents await delete/reindex.
- [x] Return an empty non-nil item slice when every requested hit is stale.
- [x] Add a page-aware fake engine and tests for:
  - page 1 contains a ghost and returns fewer items;
  - page 2 returns only its own live items;
  - page 1 and page 2 have no overlapping live ID;
  - both pages report the same total;
  - requested engine page is called once;
  - duplicate/invalid IDs inside one engine page are removed safely;
  - live-source failure returns an error without partial results.

### Exit

- [x] A regression test fails against `f01cc1cef` behavior and passes after the
      fix.
- [x] No loop advances `SearchInput.Page` inside live hydration.
- [x] Search service unit and race tests pass.

### M4 actual verification (2026-07-22)

- `searchWithLiveHydration` now makes exactly one normalized engine request,
  removes invalid/duplicate IDs within that page, hydrates it once, and returns
  the engine-derived `total` unchanged. Empty all-ghost pages return `[]`.
- Page-aware regressions prove page 1 with a ghost remains short, page 2 returns
  only its own IDs, adjacent pages do not overlap, both totals remain stable,
  and live-source failure returns no partial result.
- `rg -n "maxLiveRefillPages|enginePage\\+\\+|pageInput\\.Page"
  apps/api/app/Support/Search` returned no matches.
- `go test ./app/Support/Search -count=1` and
  `go test -race ./app/Support/Search -count=1`: passed.

---

## M5 — Single-Pass Forum Hydration

### Problem

Production assembly injects `forumLiveSearchSource`, then the HTTP adapter calls
`ListPublicTopicSearchHits` again to recover avatar/last-reply/tag fields. Each
call performs the summary query and a tag query.

### Recommended implementation

Use a request-scoped `forumLiveSearchBatch` in `bootstrap/search_adapter.go`:

1. The batch implements `search.LiveTopicSource`.
2. Its `ListPublicByIDs` calls `ListPublicTopicSearchHits` once, retains the full
   `map[int64]forum.TopicSummary`, and returns mapped `TopicSearchDoc` values.
3. Add a non-mutating per-call search entry point such as
   `SearchWithLiveSource(ctx, input, live)`; ordinary `Search` may continue to
   use the service default source for Page ViewModels.
4. The HTTP adapter orders the retained summaries by `res.Items` IDs instead of
   querying the store again.

Equivalent designs are allowed only if they satisfy D5 and remain simpler.

### Tasks

- [x] Remove the second `ListPublicTopicSearchHits` call from the successful
      HTTP search request.
- [x] Keep avatar, tags, last reply author, timestamps, status, and counts
      identical to `GET /topics` rows.
- [x] Ensure per-call caches are not stored on a shared adapter/service object.
- [x] Add counting-fake tests proving one live batch call and stable engine
      ordering.
- [x] Add a concurrency/race test for simultaneous searches with different IDs.
- [x] Confirm Page ViewModel search continues to drop ghosts.

### Exit

- [x] One HTTP search page performs one summary query and one tag query after
      the engine call.
- [x] No search package import of Forum Models is introduced.
- [x] `go test -race ./app/Support/Search ./bootstrap` passes.

### M5 actual verification (2026-07-22)

- `SearchWithLiveSource` accepts a per-call live source without changing the
  default source used by Page ViewModels. Bootstrap's `forumLiveSearchBatch`
  makes the sole summary/tag query during validation, retains that map, and
  serializes it in engine order without a second store call.
- `TestSearchServiceAdapterUsesOneForumBatchAndPreservesEngineOrder` proves one
  batch call and preserves list-only fields including tags and last reply
  author. `TestSearchServiceAdapterKeepsLiveBatchesRequestScoped` proves two
  concurrent searches cannot share a batch.
- `go test ./app/Support/Search ./bootstrap -count=1`,
  `go test -race ./app/Support/Search ./bootstrap -count=1`, and
  `go test -v ./bootstrap -run 'TestSearchServiceAdapter' -count=1`: passed.
- `rg` confirms `app/Support/Search` has no Forum Models import.

---

## M6 — Extensions Controller Test Stability

### Problem

The filtered extension-list request exceeded Fiber's default one-second test
deadline only during the full parallel Go suite. The same test passed 5/5 in
isolation. CompatFarm and plugin builds create substantial concurrent load.

### Tasks

- [x] Measure the list handler in isolation and during package-parallel tests;
      confirm it does not start runtimes, build plugins, or perform request-time
      package scanning unexpectedly.
- [x] If handler behavior is bounded and the failure is harness-only, update the
      shared request helpers to pass an explicit `fiber.TestConfig`, for example
      a 5-10 second timeout with `FailOnTimeout: true`.
- [x] Apply the same explicit test policy consistently to login/JSON helpers in
      that package; do not scatter arbitrary sleeps.
- [x] Run the focused test at least 20 times.
- [x] Run the Extensions Controller package while CompatFarm or the full Go
      suite is under normal load.
- [x] Do not claim the separate V3 CompatFarm single-execution task complete;
      that remains M6 in the production-rewire book.

### Exit

- [x] `go test ./app/Http/Controllers/Extensions -count=20` passes.
- [x] Two consecutive `go test ./...` runs pass without `i/o timeout`.
- [x] Any real request-time blocking discovered is fixed instead of hidden by a
      larger timeout.

### M6 actual verification (2026-07-22)

- The list handler only authorizes then calls Service `Detail` or `List`; it
  starts no runtime, build, or request-time package scan.
- Shared Extensions Controller helpers now pass `fiber.TestConfig{Timeout:
  10s, FailOnTimeout:true}` for login, regular, and JSON requests. This keeps
  timeout failure explicit while tolerating normal parallel build pressure.
- `go test -v ./app/Http/Controllers/Extensions -run
  'TestControllerListsAndEnablesExtensionsForManager' -count=20`: 20 passed
  (roughly 0.3-0.8s per run).
- `go test ./app/Http/Controllers/Extensions -count=20` and two consecutive
  `go test ./...`: passed without `i/o timeout`.
- This does not close or alter the separate V3 production-rewire CompatFarm M6.

---

## M7 — Full Gate And Knowledge Close

### Knowledge corrections

- [x] Update `knowledge/modules/search.md` with final FTS and pagination
      semantics.
- [x] Update `knowledge/modules/frontend.md` with advanced-reply resolve proof.
- [x] Reconcile `knowledge/index.md`:
  - million-scale read path is completed M0-M7;
  - view-count increment is complete; likes/reactions/bookmarks remain open;
  - clarify Marketplace product deferral versus mandatory V3 production wiring;
  - retain genuine open questions.
- [x] Mark this plan completed in `knowledge/plans/README.md` only after all
      exits pass.
- [x] Replace the planning handoff with a completion handoff containing exact
      commands and results.

### Required verification

```bash
git status --short
git diff --check

cd apps/api
go test ./app/Support/Search ./app/Models/PageViewModels \
  ./app/Http/Controllers/Pages ./app/Http/Controllers/Forum \
  ./app/Http/Controllers/Extensions ./bootstrap
go test -race ./app/Support/Search ./bootstrap
go test ./...

cd ../web
bun test
bun run typecheck
bun run build

cd ../..
ruby scripts/validate-openapi-refs.rb
./scripts/test.sh
```

### Browser smoke

Use the user's existing port 3000 server if it is running; do not kill or
replace it. With a running API and an authenticated account, verify:

- homepage search keeps the active theme and does not duplicate adjacent pages;
- `/topics/reply?topic=<valid>` renders the active theme L1 and submits once;
- anonymous advanced reply redirects to login;
- themed 404 renders, and a forced theme-resolve failure falls back to Core;
- locale switching changes copy without adding a locale URL prefix.

### M7 actual verification (2026-07-23)

- `git diff --check`: passed.
- Focused backend:
  - `go test ./app/Support/Search ./app/Models/PageViewModels ./app/Http/Controllers/Pages ./app/Http/Controllers/Forum ./app/Http/Controllers/Extensions ./bootstrap`: passed.
  - `go test -race ./app/Support/Search ./bootstrap`: passed.
  - `go test -v ./bootstrap -run 'TestProductionLifecycleStackUninstallsPreservedDataThroughRealRuntimeAndPostgres|TestReferenceSEOFormalZipUploadTrustEnableRestartDisableUpgradeUninstall' -count=1`: passed on a fresh migrated PostgreSQL database.
- Full gate was run once against fresh database `sforum_codex_m7_final_20260723`.
  It passed Go, CompatFarm, protobuf/SDK docs, OpenAPI refs, staged extension,
  production trust, WebSocket proxy, Nuxt typecheck, trusted editor/catalog web
  unit tests, admin framework, identity UI, homepage, SEO, moderation, and theme
  runtime checks. It then exposed stale trusted-admin and V3 catalog validation
  checks; those were fixed and the failed/residual stages were rerun:
  - `node tests/validate-trusted-admin-runtime.js`: passed.
  - `node tests/validate-theme-activation.js`: passed.
  - `node tests/validate-dev-worker-script.js`: passed.
  - `node tests/validate-signal-garden-theme.js`: passed.
  - `node tests/validate-sf-components.js`: passed.
  - `node tests/validate-page-registry-runtime.js`: passed offline contracts; live HTTP smoke skipped by script because `PAGE_REGISTRY_API` was not set.
  - `node tests/validate-v3-p0-catalogs.mjs`: passed with 265 routes and 153 UI surfaces.
  - `cd apps/web && bun run typecheck`: passed after the trusted-admin loader repair.
  - `cd apps/web && bun test tests/extensionSettingsOwnership.test.ts`: passed.
- `cd apps/web && bun test`, `cd apps/web && bun run build`, and
  `ruby scripts/validate-openapi-refs.rb` passed earlier in M7 closure before
  the final fixture repairs.
- Browser smoke: the user's Nuxt dev server was listening on port 3000 and
  `curl -i -sS http://127.0.0.1:3000/` returned HTTP 200 with active theme
  HTML and `data-sforum-theme="ocean_blue"`. Direct API probes on 9000/9002 were
  not running, so authenticated advanced-reply and forced-failure browser flows
  were not completed in this G0 commit.

### Exit

- [ ] Every command above passes or an environment-only skip is recorded with
      exact reason and unaffected substitute evidence.
- [ ] `./scripts/test.sh` is green end to end.
- [ ] No P0/P1 finding in this book remains open.
- [ ] Knowledge status matches the final code and tests.

## Out Of Scope

- V3 production-rewire findings already owned by the related task book.
- Protocol V1/request-time-loader LTS deletion before its removal window.
- Social-login provider implementation.
- Likes/reactions/bookmarks.
- Payments, Marketplace product UI expansion, or category-scoped ACL.
- Adding a new PostgreSQL Chinese tokenizer dependency.

## Final Acceptance Checklist

- [ ] Valid `simple` FTS in runtime, migration, fixture, and tests.
- [ ] Historical migration restored; no new-install/upgrade drift.
- [ ] Nuxt unit, typecheck, and build green.
- [ ] `forum.topic.reply` exact theme resolve proven through HTTP/runtime.
- [ ] Adjacent search pages contain no duplicate live IDs caused by refill.
- [ ] Search totals are stable across pages.
- [ ] One Forum live-hydration batch per HTTP search page.
- [ ] Extensions Controller stable under repeated/full-suite load.
- [ ] Full repo gate green twice where flake evidence is required.
- [ ] Knowledge plan/module/index/handoff updated.

## Suggested First Prompt For The New Conversation

```text
请按 knowledge/plans/2026-07-22-current-head-regression-remediation.md 从 M0
开始执行。先读 AGENTS.md、knowledge/index.md、search/frontend/extensions
模块说明和任务书。保留共享工作树中的其他修改，不要整提交回滚；每完成一个
里程碑就运行该节验收测试并更新任务书状态。不要把既有 V3 production-rewire
任务混入本任务，也不要提前声称 V3 100%。
```

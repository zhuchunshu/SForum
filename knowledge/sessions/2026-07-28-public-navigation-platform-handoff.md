# 2026-07-29 Public Navigation Platform Handoff

## Changed

- M0-M6 are complete. The only next milestone is M7.
- One revisioned SiteChrome document owns topbar, sidebar, mobile, and footer;
  admin CAS/defaults/history/restore/export/import and API-LTS compatibility
  remain intact.
- Public SSR uses one actor/locale-sensitive `private, no-store` resolver.
  Topbar/mobile/sidebar/footer fail closed without replacing Host utilities or
  target API authorization.
- Sidebar ordinary links and `core.dynamic.categories` render in resolver
  order; Forum remains the only taxonomy data owner. Footer legal links are
  canonical while copyright/friend links retain their owners.
- Exact active-theme runtime state projects validated `navigationLocations`.
  Default and Nocturne declare all four v1 locations; Core emergency fallback
  supports all four without rewriting operator configuration.
- Mobile homepage selector visibility and width were repaired without changing
  the canonical mobile drawer authority.
- Fixed a post-M6 topbar ordering regression: canonical placements were sorted
  correctly, but their `order` was omitted when entering Navigation Registry,
  causing the registry to break ties by source key (`categories`, `home`,
  `tags`). Base composition now retains placement order, with a focused
  registry-enabled regression test.
- Fixed authenticated hard-refresh SSR on `/categories`, `/tags`, and `/u/**`:
  these routes now retain SSR but statically disable Nitro whole-page caching,
  alongside the existing `/c/**`, `/tags/**`, and `/t/**` rules. Session-bearing
  HTML/payload responses remain `private, no-store`; the ineffective request-
  time route-rule mutation was removed, and authenticated HTML is never cached
  or varied by Cookie.
- Refined the Personalization navigation editor into separate Public
  Navigation and Recovery/Transfer tab cards. Add/edit operator-link forms now
  open in one shared modal instead of permanently occupying a side column.

## Verification

### Authenticated hard-refresh cache fix

- PASS cache/session regression suite: 18 tests, 178 expectations across
  `publicTaxonomyPages.test.ts` and `protectedRouteRendering.test.ts`.
- Nuxt typecheck reached `vue-tsc` but is currently blocked by the pre-existing
  dirty-worktree navigation edit at
  `SFAdminPersonalizationNavigationTab.vue:215`: it passes `aria-label` where
  `SFAdminFixedTabNav` requires `ariaLabel`. This cache fix does not touch that
  component.
- Browser/runtime hard-refresh verification was deliberately not run; the user
  will restart the Nuxt dev server and verify manually without clearing old
  cache files.

### Prior M6 evidence

- PASS Web full suite: 719 pass, 0 fail, 4795 expectations; focused final
  navigation/home/taxonomy suite: 37 pass, 0 fail, 480 expectations.
- PASS Web typecheck/build, full Go rerun, architecture validation (1448
  production files), built-in rebuild, and `git diff --check`.
- PASS exact default-theme activation at digest
  `5f9f5141067fc528a063e439c01e1cf0ee526347c6298896fe2a9ec80b81f44d`.
- PASS runtime skin/resolve identity and four-location navigation revision 6.
- PASS desktop and `390x844` Chrome: default-theme provider, template 1, one
  navbar/footer, no fallback/overflow/app console errors; mobile selection of
  `general` navigated to `/c/general`. User independently confirmed behavior.
- The topbar-order follow-up was not executed locally; the user owns manual
  verification for this small correction.
- The navigation-editor UI refinement was code-only by explicit request; no
  additional browser, test, typecheck, or build verification was run, and the
  user owns manual verification.

## Decisions

- M7 owns stable route identity mappings, catalog regeneration, docs, complete
  lifecycle/recovery scenarios, and the final release gate. Do not implement a
  separate M8.
- Preserve the user's port-3000 service and current dirty worktree.

## Next

- Execute M7 only, in the task-book order.
- Start with reviewed route identity mappings and the real fixture/reference
  plugin lifecycle matrix, then theme switch/rollback, Safe Mode/Core fallback,
  cache/concurrency/backup restore, docs/OpenAPI/catalogs, full gates, Browser
  matrix, and the required final report.

## Open Questions

- No M6 blocker. M7 remains not started.

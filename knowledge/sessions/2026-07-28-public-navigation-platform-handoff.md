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
  order; Forum remains the only taxonomy data owner. Mobile defaults contain
  the three rendered route links, while footer navigation defaults to empty and
  copyright/friend links retain their owners.
- Fixed admin/public document divergence: migration `202607290074` materializes
  missing topbar/sidebar/mobile defaults without overwriting explicit placement
  settings, and the public resolver no longer injects invisible read-time
  defaults. Missing placements now remain absent in both admin and public views.
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
- The navigation list now renders each item's effective icon (location override
  first, definition fallback), and Core-owned routes use an explicit system
  built-in source label.
- Core-owned links can now edit location-scoped label, icon, and visibility
  overrides while their route identity remains read-only. The editor provides
  a one-click reset to the code-owned presentation defaults before draft save.
- Desktop full-width three-column shells now give the left and right rails
  independent vertical overflow, contained overscroll, and stable scrollbar
  gutters instead of clipping long navigation or contextual content. Host/Core
  compatibility CSS and the default-theme source share the same rule.
- Core link icon overrides are tri-state: inherit the built-in default, select
  a custom icon, or explicitly render no icon for the current placement. The
  public contract carries the suppression flag through topbar/sidebar/mobile/
  footer renderers, and admin save invalidates cached public-navigation data.
- Admins can edit `core.dynamic.categories` and set a placement-level maximum
  of `0..100` categories. Zero is the recommended legacy-compatible unlimited
  default. The API persists and validates the limit, public resolution carries
  it, and `SFHomeNavigation` applies it to desktop and mobile category lists;
  truncated lists expose a localized `/categories` link below the control.

## Verification

### Stored navigation authority correction

- PASS current development database migration: embedded migrator applied
  `202607290074_materialize_public_navigation_defaults.sql` as the only pending
  Core migration. Browser, unit, integration, typecheck, and build verification
  were not run; the user owns desktop/mobile manual verification.

### Side-rail overflow correction

- Source and static contract changes only by explicit request. Browser, focused
  test, typecheck, build, built-in restaging, and exact-theme activation were not
  run; the user owns rendered verification and active-artifact refresh.

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

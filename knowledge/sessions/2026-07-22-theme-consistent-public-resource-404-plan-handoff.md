# 2026-07-23 Theme-Consistent Public Resource 404 Completion Handoff

## Changed

- Completed focused task book M0-M6 for missing/hidden/deleted topic,
  category, tag, profile, and unmatched public routes.
- Semantic Page Registry 404s now skip retry, preserve reason/status, enter the
  Nuxt error boundary, and render `system.not_found` through the selected theme.
- Default and Nocturne 404 templates keep their own navbar/sidebar/body/footer.
  Reviewed Host islands own generic 404 copy/actions; Core is a bounded,
  non-recursive emergency page for real theme/runtime/API failure.
- Error documents return HTTP 404, `Cache-Control: no-store`, and
  `noindex,nofollow`, with success canonical and JSON-LD removed.
- Follow-up hard-refresh fix: `error.vue` no longer promotes readiness from a
  blanket `allSettled`. L1 and L0 remain uncommitted candidates until the
  active renderer source plus extension/version/package digest/node revision
  match; only then are CSS head links and the selected provider committed in
  one batch. Any required request or identity failure enters complete Core and
  clears active theme identity plus links.
- 404 SSR requests now explicitly use `NUXT_API_INTERNAL_BASE_URL` through the
  shared API client option instead of relying on relative `$fetch` behavior in
  Nuxt's error renderer. Hydration reuses the serialized provider/artifact/CSS
  decision without immediately fetching a second skin.
- `SFPageOutletRender` and the system-error Host island chain now use direct
  component imports. Ordinary Page Registry island laziness uses explicit Vue
  async import wrappers instead of runtime Nuxt `Lazy*` component lookup,
  removing the observed undefined vnode source.

## Verification

- Focused web suites: `50 pass, 0 fail, 251 expect()`; earlier wider focused
  run: `75 pass`.
- Web typecheck and production build passed before the final HMR/loading guard.
- Full Go tests and OpenAPI validation passed (`2165` refs, `54` files).
- Full Bun/repository gate reached one unrelated existing failure:
  `apps/web/tests/prebuiltSettingsComponent.test.ts` still expects obsolete
  `/_sforum/private-assets/extensions/...`; result was `542 pass, 1 fail`.
- HTTP/cache/SEO probes passed. Browser coverage included healthy/missing
  routes, default/Nocturne, desktop/mobile, light/dark, both locales, signed-in
  chrome, navigation recovery, hydration/console health, and emergency Core.
  Later HMR/Core-flash/Reka hydration reports were addressed with exact `SF*`
  imports and client-only dropdowns with stable SSR placeholders. The interim
  unresolved-state skeleton was replaced by the atomic candidate/Core state
  flow above. Per user direction, those final guards were committed without
  another browser/build cycle.
- The hard-refresh/exact-artifact and vnode-resolution follow-up documented
  above has **not** run automated tests, typecheck, build, Go tests, repository
  scripts, curl, Playwright, or Browser, per explicit user instruction. Manual
  verification is required.

## Next

1. Resume `../plans/2026-07-22-theme-defined-system-error-pages.md` at M1.
2. Reuse this request-local context, system-theme AST renderer, document
   policy, and Core fallback when adding 403/429/5xx; do not reimplement 404.
3. Repair the unrelated stale prebuilt settings asset-path assertion in its
   owning workstream.

## Open Questions

- None for the focused public-resource 404 scope.

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
  imports, an unresolved-state skeleton, and client-only dropdowns with stable
  SSR placeholders. Per user direction, those final guards were committed
  without another browser/build cycle.

## Next

1. Resume `../plans/2026-07-22-theme-defined-system-error-pages.md` at M1.
2. Reuse this request-local context, system-theme AST renderer, document
   policy, and Core fallback when adding 403/429/5xx; do not reimplement 404.
3. Repair the unrelated stale prebuilt settings asset-path assertion in its
   owning workstream.

## Open Questions

- None for the focused public-resource 404 scope.

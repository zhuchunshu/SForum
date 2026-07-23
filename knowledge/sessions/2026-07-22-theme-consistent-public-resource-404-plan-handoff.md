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
- Nuxt payload extraction is disabled: request-local `no-store` policy can make
  Nuxt 4.4 extracted `_payload.json` requests return HTML and cancel client
  navigation. SSR payload is inline in both development and production.
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
- Default-theme 404 keeps its footer in current and already-installed theme
  artifacts. The development-only Nuxt error overlay entry is suppressed only
  for body-marked ordinary 404 pages, then removed after successful error
  recovery; genuine runtime/5xx diagnostics remain while active.

## Verification

- Full Bun gate passed: `546 pass, 0 fail, 3303 expect()`.
- Nuxt typecheck and the final production build passed after all follow-up
  guards; Nitro reported `Build complete!`.
- Full Go tests and OpenAPI validation passed (`2165` refs, `54` files).
- `./scripts/test.sh` passed, including Page Registry and V3 catalog validation
  (`265` routes, `161` UI surfaces, `99` traceability rows).
- Production HTTP probes covered missing topic/category/tag/profile/unmatched
  routes: every HTML document was HTTP 404 with `no-store`,
  `noindex,nofollow`, selected-theme/template markers, and no success canonical
  or JSON-LD. The homepage control was HTTP 200 with the expected shared-cache
  policy and success SEO data.
- Browser coverage included healthy/missing routes, default/Nocturne,
  desktop/mobile, light/dark, both locales, signed-in chrome, navigation
  recovery, hydration/console health, and emergency Core. Final latest-build
  production checks at `1440x1000` and `390x844`, plus fresh development tabs,
  found no blank screen, visible framework overlay, horizontal overflow, or
  console warning/error. Selected-theme navbar/sidebar/body/footer remained
  coherent; home recovery and homepage-to-topic SPA navigation succeeded.

## Next

1. Resume `../plans/2026-07-22-theme-defined-system-error-pages.md` at M1.
2. Reuse the server pre-preparation plugin, request-local presentation
   composable, exact-artifact validation, document policy, system AST renderer,
   and Core emergency fallback when adding 403/429/5xx; do not reimplement 404.

## Open Questions

- None for the focused public-resource 404 scope.

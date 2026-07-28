# M5 - Topbar, Mobile, And Core Fallback Runtime Wiring

Milestone: M5 - Topbar, mobile, and Core fallback runtime wiring
Status: completed

## Changed

- Replaced `SFNavbar`'s legacy `/site/nav-items` merge and hardcoded fallback
  links with one canonical `/site/navigation` request for topbar and mobile.
- Added request-local navigation state plus focused link and mobile renderers.
  Ordinary navigation fails closed while Host-owned search, compose, session,
  locale, appearance, notification, and recovery controls remain independent.
- Rewired Page ViewModels to `ResolvePublicNavigation`; resolver absence,
  failure, or missing topbar now returns `ErrCorePageDataUnavailable` so Page
  Registry can use its existing Core emergency path.
- Added bounded four-item topbar overflow through the existing Nuxt UI menu,
  retained locale, active-state, tag-policy, extension-route, and safe external
  link behavior, and removed the second frontend extension merge authority.
- Marked actor-sensitive public navigation responses `private, no-store` with
  `Vary: Cookie, Authorization, Accept-Language`.
- Reduced `SFNavbar.vue` from 1075 to 902 lines and lowered the architecture
  ratchet to the reclaimed size.

## Permission, Cache, And Fallback Evidence

- The API remains authoritative for actor and permission visibility. Web cache
  keys include locale and actor state, and login/logout or locale changes do
  not share navigation payloads.
- `TestResolvePublicNavigationFiltersActorVisibilityUnsafeAndMissingExtension`
  proves guest/member filtering, unsafe-link removal, and inert missing-plugin
  behavior.
- `TestResolvePublicNavigationUsesExactRegistryArtifactAndSafeMode` proves an
  exact-artifact contribution appears normally and disappears in Safe Mode.
- Page ViewModel tests prove actor/locale/location forwarding and fail closed on
  resolver error; Page controller emergency-render tests prove Core fallback
  remains the existing authoritative transport path.

## Verification

- PASS `cd apps/web && bun test`: 714 pass, 0 fail, 4758 expectations, 98 files.
- PASS `cd apps/web && bun run typecheck`.
- PASS `cd apps/web && bun run build`; only existing sourcemap, chunk-size,
  mixed-import, and Iconify JSON warnings.
- PASS `cd apps/api && go test ./app/Models/SiteChrome/... ./app/Support/Pages/... ./app/Http/...`.
- PASS `node tests/validate-architecture-boundaries.mjs`: 1447 production files,
  166 above review threshold.
- PASS `git diff --check`.

## Runtime Evidence

- `GET /api/v1/site/navigation?locations=public.topbar.primary,public.mobile.primary`
  returned HTTP 200, schema `sforum.site-navigation@1`, revision 6,
  `Cache-Control: private, no-store`, and the required `Vary` dimensions.
- `/pages/resolve?id=forum.home&path=/` returned selected provider
  `sforum.default-theme`, `fallback=false`, template `templates/home.html`, and
  digest `663dff1fbb52e317e83db3e43e51ace46a87b7e70e32c54b862b9347aee089d3`.
- Authenticated Chrome desktop `1920x936`: `data-provider` is
  `sforum.default-theme`, `data-template="1"`, topbar is `分类/首页/标签`, one
  visible navbar, one visible footer, no fallback notice, no overflow, and no
  console warnings/errors.
- Authenticated Chrome `390x844`: independent drawer is
  `社区导航/首页/分类/标签`, with one visible navbar/footer, no overflow, and no
  console warnings/errors. Hard refresh, SPA close-on-navigation, English
  labels, and dark mode were also exercised; locale and appearance were
  restored afterward.

## Residual Notes

- The current topbar has three visible items after duplicate search suppression,
  so the rendered overflow menu did not open in this runtime configuration.
  Its four-item boundary and overflow projection are covered by focused Web
  tests without mutating operator data solely to fabricate evidence.
- Core fallback was verified by Page Registry/Page ViewModel integration tests;
  the aborted API-offline browser attempt is not counted as passing evidence.

## Next

M6 - Sidebar dynamic block and built-in theme locations.

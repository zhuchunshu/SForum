# 2026-07-22 i18n no_prefix seamless locale switch

## Changed

- Nuxt i18n `strategy: 'no_prefix'` — no `/en` routes; language via cookie +
  `setLocale`.
- Dropped all `/en/**` `routeRules` mirrors.
- Middleware: `locale-prefix-compat` (301 strip `/en`), `locale-cache` (non-
  default cookie bypasses shared SWR).
- Sitemap static home: single `/` entry (no en alternate URL).
- `useLocaleHead` seo alternates off; keep `lang`/`dir`.
- Navbar language menu already used `setLocale` (stays; comment updated).
- Tests updated for no_prefix contracts.

## Decisions

- See `knowledge/decisions/2026-07-22-i18n-no-prefix-seamless.md`.

## Next

- Hard-refresh dev server after config change; toggle language on `/` and a deep
  page — URL must stay put, copy must flip, cookie `sforum_locale` must update.
- Hit `/en/categories` once — expect 301 → `/categories`.

## Open Questions

- None for this switch.

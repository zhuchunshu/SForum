# Decision: Cookie-based UI locale without `/en` routes

## Status

Accepted

## Context

Operators and members expect language switching to feel instant and not rewrite
the address bar (`/categories` → `/en/categories`). The previous Nuxt i18n
strategy `prefix_except_default` forced a new URL for English and interacted
badly with `detectBrowserLanguage` + cookie when returning to the default
locale.

Product priority for SForum’s current stage: **seamless UI language** over
separate SEO URLs per UI language. User-generated forum content remains
author-language; it is not auto-translated.

## Decision

1. Set `@nuxtjs/i18n` `strategy: 'no_prefix'`.
2. Switch languages only with `setLocale(code)` (cookie `sforum_locale` + message
   catalogs). Do not generate `/en/*` routes.
3. Keep `localePath()` helpers for future-proofing; under `no_prefix` they are
   identity for path strings.
4. 301-strip legacy `/en` and `/en/**` to the unprefixed path.
5. Bypass shared SWR/HTML cache when `sforum_locale` is set and not `zh-CN`.
6. Disable multi-locale `hreflang` alternates in `useLocaleHead` (`seo: false`);
   still set `html lang`.

## Consequences

- Switching language no longer changes the URL.
- Crawlers see one URL surface; default UI language for anonymous/no-cookie
  traffic is `zh-CN` (plus browser detection on first visit when enabled).
- True bilingual SEO (separate indexable EN/ZH URLs for the same chrome) is
  deferred. Revisit only if product later needs locale-prefixed public marketing
  pages.
- Supersedes the route portion of
  `knowledge/decisions/2026-07-03-multilingual-default-zh-cn.md` (default locale
  and catalogs remain).

## Follow-up

- Optional: persist signed-in user profile locale to cookie on login.
- Optional: `Accept-Language` negotiation only on first visit (already via
  detectBrowserLanguage).

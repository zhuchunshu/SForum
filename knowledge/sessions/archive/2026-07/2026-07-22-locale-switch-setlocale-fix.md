# 2026-07-22 Locale switch: setLocale + active state

## Changed

- Public topbar language menu no longer uses bare `switchLocalePath` NuxtLink
  targets. It calls `setLocale(code)` so `sforum_locale` cookie and path stay
  in sync.
- Menu `active` is forced from i18n `locale` (not Vue Router path matching).

## Root cause

- Strategy `prefix_except_default`: default `zh-CN` has no prefix; `en` uses
  `/en/*`. Language switch **must** change the URL for SEO (hreflang/canonical).
- Bug: menu only navigated via `to: switchLocalePath(...)`. Switching en→zh
  went to `/` while cookie stayed `en`. With
  `detectBrowserLanguage.redirectOn: 'root'`, a request to `/` with
  `sforum_locale=en` 302s back to `/en` — stuck on English.
- “zh is active” while on English: default-locale target `/` can look
  router-active under non-exact link matching; checkmark/active was not
  strictly bound to `locale === code`.

## Verification

- `curl -sI -H 'Cookie: sforum_locale=en' http://127.0.0.1:3000/` → `302` `/en`
  (pre-fix server behavior that pure path links hit).
- `bun test tests/defaultThemeNavbar.test.ts` — pass (includes setLocale contract).

## Files

- `apps/web/app/components/SFNavbar.vue`
- `apps/web/tests/defaultThemeNavbar.test.ts`

## Next

- Optional: after HMR, manually toggle zh↔en on home and a deep page (`/en/categories`).

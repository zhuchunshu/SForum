# Localization Module

## Purpose

Owns product internationalization rules, supported locales, locale negotiation,
translation key conventions, and server-owned localized templates.

## Current Status

Foundation scaffold exists:

- Frontend locale catalogs under `apps/web/i18n/locales`.
- Backend locale normalization under `apps/api/app/Support/Localization`.
- Built-in API envelope catalog lives in
  `apps/api/app/Support/Localization/messages.go` (`zh-CN` / `en-US`).
  `Message(locale, key)` falls back to the raw key when missing — so every
  HTTP error reason must be listed in the catalog or the admin UI shows the
  machine key (e.g. `site_chrome.invalid`).
- 2026-07-12: filled missing admin/API codes for site chrome, profile,
  database inspector, moderation, jobs/schedules, mail, notifications, CSRF,
  password-reset, forum search/reindex, extension runtime routes, SEO sitemap,
  and related identity guards. Regression:
  `TestAPIErrorCodesHaveLocalizedMessages` walks `Code*` constants and
  `fiber.NewError` string reasons so new codes fail CI if untranslated.
- Nuxt i18n SEO links use `APP_URL` as the base URL.
- Runtime options now store the operator-selected default locale and enabled
  locale list. Environment values are first-run fallbacks for missing option
  rows.
- Runtime language pack management has an accepted design: operators will
  upload ZIP language packs from an admin "Language settings" page, package
  files will live under `LOCALE_PACK_ROOT`/`storage/locale-packs` outside Git,
  and database tables will track package versions, provided locales, status,
  and events.
- The first language pack release will apply uploaded packages only to frontend
  runtime UI messages. Backend message files are reserved in the package format,
  but backend API envelope messages continue to use the built-in catalog.

## Requirements

- Default locale: `zh-CN`.
- First secondary locale: `en-US`.
- All user-facing product features must be localizable.
- Simplified Chinese translations must be complete before a feature is done.
- English translations should ship with the same feature unless explicitly
  deferred.

## Frontend Responsibilities

- Configure Nuxt i18n.
- Keep message catalogs under `apps/web/i18n/locales`.
- Use stable translation keys in Vue pages, components, layouts, form messages,
  navigation, empty states, and admin UI.
- Use localized routes and metadata for public SEO pages.
- Send locale context with API requests, preferably through `Accept-Language`.
- Display backend API `message` first for API-originated prompts and operation
  results.
- Use frontend catalog or fallback messages for client-side validation, network
  failures, missing responses, static UI, and frontend-owned states.

## Backend Responsibilities

- Define supported locale normalization and fallback locale behavior.
- Negotiate locale from route, cookie, user profile, `Accept-Language`, then
  `zh-CN`.
- Return localized API `message` values in the unified response envelope.
- Return stable machine-readable reason keys in `data.reason` for API errors.
- Localize backend-owned emails, notifications, moderation reason labels, and
  seed/admin labels when they are rendered by the backend.
- Store user locale preference separately from user-generated content.
- API request locale negotiation reads runtime `site.default_locale` and
  `site.supported_locales`; if option loading fails, it falls back to startup
  configuration.
- Once runtime packs are implemented, `site.supported_locales` should accept
  built-in locales plus locales provided by enabled language packs.

## SEO Responsibilities

- UI locale strategy is `no_prefix`: Chinese and English share the same public
  URLs; language is selected via `sforum_locale` cookie / `setLocale`.
- Public pages emit `html lang` (and dir) from the active locale. They do **not**
  emit multi-URL `hreflang` alternates (same URL cannot honestly represent two
  languages for crawlers).
- Sitemaps list content paths once (default-locale UI). Legacy `/en` and `/en/*`
  bookmarks 301-strip to the unprefixed path via
  `server/middleware/locale-prefix-compat.ts`.
- Non-default locale cookie requests bypass shared Nitro SWR
  (`server/middleware/locale-cache.ts`) so cached zh HTML is not served to en.

## Open Questions

- Should English translations be mandatory for MVP launch or allowed to lag
  behind during internal development?
- Should user-generated topics/posts allow an explicit content-language field in
  MVP?
- Which locales should follow after `zh-CN` and `en-US`?
- When should uploaded backend message files become active for API envelope
  messages, emails, notifications, and worker-owned templates?

## Next Steps

- Expand locale coverage as user-facing pages are added.
- Add backend locale config and profile preference field during identity schema
  design.
- Implement the admin language settings page, `locale.manage`, runtime locale
  pack tables, package upload/enable APIs, and frontend runtime message loader.

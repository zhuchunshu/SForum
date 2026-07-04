# Localization Module

## Purpose

Owns product internationalization rules, supported locales, locale negotiation,
translation key conventions, and server-owned localized templates.

## Current Status

Foundation scaffold exists:

- Frontend locale catalogs under `apps/web/i18n/locales`.
- Backend locale normalization under `apps/api/app/Support/Localization`.
- Nuxt i18n SEO links use `APP_URL` as the base URL.

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

- Define supported locales and fallback locale.
- Negotiate locale from route, cookie, user profile, `Accept-Language`, then
  `zh-CN`.
- Return localized API `message` values in the unified response envelope.
- Return stable machine-readable reason keys in `data.reason` for API errors.
- Localize backend-owned emails, notifications, moderation reason labels, and
  seed/admin labels when they are rendered by the backend.
- Store user locale preference separately from user-generated content.

## SEO Responsibilities

- Default Simplified Chinese pages use unprefixed canonical URLs.
- English pages use `/en/*`.
- Public pages emit `lang`, `hreflang`, canonical, and Open Graph locale tags.
- Sitemaps include localized public URLs only when the corresponding content is
  public and indexable.

## Open Questions

- Should English translations be mandatory for MVP launch or allowed to lag
  behind during internal development?
- Should user-generated topics/posts allow an explicit content-language field in
  MVP?
- Which locales should follow after `zh-CN` and `en-US`?

## Next Steps

- Expand locale coverage as user-facing pages are added.
- Add backend locale config and profile preference field during identity schema
  design.

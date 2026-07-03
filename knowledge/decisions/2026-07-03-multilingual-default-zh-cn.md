# Decision: Multilingual Product With Simplified Chinese Default

## Status

Proposed

## Context

SForum must support multilingual product features. The default language should
be Simplified Chinese.

The app is SEO-oriented and uses Nuxt for server-rendered public pages, so
locale routing and localized metadata need to be planned before pages are
implemented.

## Decision

Make internationalization a first-class architecture requirement:

- Default product locale: `zh-CN`.
- First secondary locale: `en-US`.
- Frontend i18n library: Nuxt i18n, backed by Vue I18n.
- Public route strategy: default Simplified Chinese pages are unprefixed;
  English pages use `/en/*`.
- Public pages must emit localized `lang`, `hreflang`, canonical, and Open
  Graph locale metadata.
- User-facing strings must use translation keys instead of inline prose.
- Backend APIs return stable machine-readable error codes. The frontend maps
  known codes to localized messages.
- Backend-owned user-facing templates, such as emails and notifications, live in
  a backend `localization` module and must support `zh-CN` first.
- User-generated content is stored as authored and is not automatically
  translated by default.

Locale negotiation order:

1. Localized route.
2. Locale cookie.
3. Signed-in user profile preference.
4. `Accept-Language`.
5. `zh-CN`.

## Consequences

- Every user-facing feature needs translation coverage before it is considered
  complete.
- SEO implementation must handle alternate links and canonical URLs per locale.
- Backend validation and domain errors should be designed around stable codes,
  not prose strings.
- Seed data, category labels, moderation labels, notifications, and emails need
  localization hooks.
- Future locales can be added without changing routing and domain boundaries.

## Follow-up

- Add Nuxt i18n configuration when `apps/web` is scaffolded.
- Create initial `zh-CN` and `en-US` locale catalogs.
- Add backend supported-locale config and user locale preference fields.
- Add tests that verify default Simplified Chinese and English route rendering.

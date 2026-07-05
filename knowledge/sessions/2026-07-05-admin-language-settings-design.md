# 2026-07-05 Admin Language Settings Design Handoff

## Changed

- Accepted the architecture for an admin language settings page under the system settings menu.
- Designed runtime ZIP language packs stored outside Git under `LOCALE_PACK_ROOT`/`storage/locale-packs`.
- Documented dedicated language pack tables, package manifest rules, `locale.manage`, public/admin APIs, frontend runtime message loading, and first-release frontend-only scope.
- Added decision record `knowledge/decisions/2026-07-05-runtime-language-pack-management.md`.

## Decisions

- Use ZIP packages with `sforum.locale.json`.
- First release enables uploaded packages only for frontend runtime UI messages.
- Backend message files are reserved but not applied.
- Runtime languages do not create new Nuxt SEO route prefixes.
- Built-in `zh-CN`/`en-US` overrides require both manifest opt-in and admin enable-time confirmation.

## Next

- Write the implementation plan after spec review.
- Implement `locale.manage`, migrations, locale pack service/controller, OpenAPI localization files, admin `/locales` page, and frontend runtime locale loader.

## Open Questions

- None for the first implementation plan.

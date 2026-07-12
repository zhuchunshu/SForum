# 2026-07-12 API error message i18n gap

## Changed

- Filled missing built-in API envelope translations in
  `apps/api/app/Support/Localization/messages.go` (zh-CN + en-US).
- Covered gaps that previously returned the raw reason key as `message`,
  including `site_chrome.invalid` / `site_chrome.not_found` and other newer
  admin modules (profile, database, moderation, jobs, mail, CSRF, password
  reset, forum search/reindex, extension runtime, SEO sitemap, etc.).
- Added `TestAPIErrorCodesHaveLocalizedMessages` so newly introduced
  `Code*` constants and `fiber.NewError(..., "domain.key")` reasons fail
  tests when the catalog has no translation.

## Decisions

- Keep stable machine keys in `data.reason`; only localize envelope
  `message` (and field messages via `LocalizeFields`).
- Do not localize River job names / schedule option key prefixes
  (`jobs.schedule.<id>.enabled`, `mail.deliver`, etc.) — those are not API
  user-facing messages.

## Next

- When adding a new API error code, add both zh-CN and en-US entries in
  `messages.go` in the same change (the regression test will catch misses).
- Optional later: extract catalog to JSON language packs once runtime pack
  loading is implemented.

## Open Questions

- None for this fix.

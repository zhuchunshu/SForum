# 2026-07-05 Session Handoff

## Changed

- Added `web_options(name, value)` with seeded `site.name = SForum`.
- Added backend Options module, provider, public read routes, and
  permission-protected update route.
- Added short-lived backend option caching with invalidation after updates.
- Added Nuxt `useWebOptions()` and an admin Settings page for editing the site
  name.
- Replaced key frontend SForum brand hard-coding in navigation/auth/admin title
  surfaces with runtime `site.name`.

## Decisions

- Keep `web_options` to exactly `name` and `value` for now.
- Keep secrets, infrastructure settings, worker tuning, and build-time route
  prefixes in environment configuration.
- Use typed validation in the Options service rather than arbitrary admin
  writes for unknown option names.

## Next

- Decide the next runtime options: registration open/closed, posting policy,
  default locale, or basic SEO text.
- Generate `sqlc` methods for `database/queries/options.sql` when `sqlc` is
  available in the local toolchain, then replace the small pgx adapter queries.
- Consider audit events for option changes when admin audit UI is added.

## Open Questions

- Which settings should be public frontend-readable, and which should be
  admin-only?

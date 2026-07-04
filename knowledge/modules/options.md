# Runtime Options Module

## Purpose

Owns self-hosted, operator-editable site settings that can safely change at
runtime without rebuilding or restarting the application.

## Current Status

Initial runtime option support is implemented.

- PostgreSQL migration `202607050001_web_options.sql` creates
  `web_options(name, value)` and seeds `site.name = SForum`.
- Backend module `apps/api/app/Models/Options` exposes a typed service with
  public runtime reads, admin-only reads, single-option compatibility updates,
  and batch updates.
- The backend service caches option values for a short TTL and invalidates the
  cache after admin updates.
- API routes:
  - `GET /api/v1/web-options` for public, frontend-safe options.
  - `GET /api/v1/web-options/:name` for a public option.
  - `PUT /api/v1/web-options` for the original single-option update path.
  - `GET /api/v1/admin/web-options` for all admin-manageable options with
    secret values masked.
  - `PUT /api/v1/admin/web-options` for batch admin updates.
- Updating options requires the existing `settings.manage` permission.
- Nuxt composable `useWebOptions()` provides `webOption()`, `siteName`,
  `siteUrl`, `defaultLocale`, `supportedLocales`, `humanVerificationProvider`,
  `refresh()`, `save()`, and admin batch helpers.
- Admin page `apps/web/app/pages/admin/settings/index.vue` uses page-level tabs
  for basic site settings and CAPTCHA/human-verification settings.

## Boundaries

- Runtime options are for self-hosted operator-managed settings such as site
  identity, site URL, enabled locales, default locale, registration policy,
  content policy, theme preferences, and CAPTCHA provider configuration.
- Do not move infrastructure or secret settings into `web_options`: database
  URLs, Redis passwords, Meilisearch master keys, worker counts, ports, and
  build-time route prefixes should remain in environment config.
- ALTCHA secret is the one approved admin-managed secret in `web_options` for
  now. It is never returned by public APIs and is masked in admin API
  responses; admins only see whether a secret is configured.
- Keep the table intentionally simple (`name`, `value`). Add typed validation in
  the Options service instead of adding per-option columns prematurely.
- Expose only settings that are safe for the frontend to read through public
  `GET /web-options` responses.

## Implementation Notes

- Current public options are `site.name`, `site.url`, `site.default_locale`,
  `site.supported_locales`, and `human_verification.provider`.
- Current admin-only options are `human_verification.altcha.secret`,
  `human_verification.altcha.challenge_ttl`, and
  `human_verification.altcha.cost`.
- Startup environment values are treated as first-run defaults/fallbacks.
  `bootstrap.NewAPI` calls `EnsureDefaults` so missing option rows are inserted
  without overwriting existing admin-managed values.
- `web_options` uses `name` as the primary key. This makes single-option reads
  and upserts index-backed.
- The Options service returns defaults when rows are missing, so fresh installs
  and partially migrated databases still have a usable site name.
- The frontend root component loads public options once and falls back to
  defaults if the API is temporarily unavailable.
- Locale settings can only enable built-in locale catalogs (`zh-CN`, `en-US`);
  adding a new locale still requires adding frontend and backend translations.

## Next Steps

- If settings need audit history, write changes to the existing audit event
  pattern instead of adding columns to `web_options`.
- When `sqlc` is available in the local toolchain, generate typed query methods
  for `database/queries/options.sql` and replace the small pgx adapter queries.

# Runtime Options Module

## Purpose

Owns self-hosted, operator-editable site settings that can safely change at
runtime without rebuilding or restarting the application.

## Current Status

Initial runtime option support is implemented.

- PostgreSQL migration `202607050001_web_options.sql` creates
  `web_options(name, value)` and seeds `site.name = SForum`.
- Backend module `apps/api/app/Models/Options` exposes a typed service with
  `WebOption`, `SiteName`, `List`, and `Update` methods.
- The backend service caches option values for a short TTL and invalidates the
  cache after admin updates.
- API routes:
  - `GET /api/v1/web-options`
  - `GET /api/v1/web-options/:name`
  - `PUT /api/v1/web-options`
- Updating options requires the existing `settings.manage` permission.
- Nuxt composable `useWebOptions()` provides `webOption()`, `siteName`,
  `refresh()`, and `save()`.
- Admin page `apps/web/app/pages/admin/settings/index.vue` lets operators edit
  the site name.

## Boundaries

- Runtime options are for public, site-facing settings such as site name,
  registration policy, content policy, or theme preferences.
- Do not move infrastructure or secret settings into `web_options`: database
  URLs, Redis passwords, ALTCHA secrets, Meilisearch master keys, worker counts,
  ports, and build-time route prefixes should remain in environment config.
- Keep the table intentionally simple (`name`, `value`). Add typed validation in
  the Options service instead of adding per-option columns prematurely.
- Expose only settings that are safe for the frontend to read through public
  `GET /web-options` responses.

## Implementation Notes

- `site.name` is the first option. It defaults to `SForum` so the project keeps
  its product identity while self-hosted deployments can brand their own site.
- `web_options` uses `name` as the primary key. This makes single-option reads
  and upserts index-backed.
- The Options service returns defaults when rows are missing, so fresh installs
  and partially migrated databases still have a usable site name.
- The frontend root component loads public options once and falls back to
  defaults if the API is temporarily unavailable.

## Next Steps

- Decide which settings belong in the first admin settings milestone:
  registration open/closed, default locale, posting policy, or basic SEO text.
- If settings need audit history, write changes to the existing audit event
  pattern instead of adding columns to `web_options`.
- When `sqlc` is available in the local toolchain, generate typed query methods
  for `database/queries/options.sql` and replace the small pgx adapter queries.

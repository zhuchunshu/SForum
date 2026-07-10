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
- Updating non-SEO options requires the existing `settings.manage` permission;
  `seo.*` options require the independent `seo.manage` permission, and
  attachment options require `attachment.settings.manage`.
- Nuxt composable `useWebOptions()` provides `webOption()`, `siteName`,
  `siteUrl`, `defaultLocale`, `supportedLocales`, `humanVerificationProvider`,
  `appearanceTheme`, footer content helpers, `refresh()`, `save()`, and admin
  batch helpers.
- Admin page `apps/web/app/pages/admin/settings/index.vue` uses page-level tabs
  for basic site settings, account password policy, and CAPTCHA/human-verification settings.
- Admin page `apps/web/app/pages/admin/personalization.vue` manages
  personalization settings for appearance presets and footer content from the
  System configuration sidebar folder.
- Admin page `apps/web/app/pages/admin/seo.vue` manages runtime SEO settings
  across meta/social, indexing/robots, sitemap, structured data, verification,
  and diagnostics tabs.
- Admin page `apps/web/app/pages/admin/attachments.vue` manages attachment
  storage provider settings, upload limits, path templates, public URL
  settings, secret-bearing provider credentials, and attachment governance.
- Admin page `apps/web/app/pages/admin/settings/avatar.vue` manages avatar
  default source, Gravatar base URL/hash algorithm, upload/GIF switches,
  upload size and dimension limits, compression target size/quality, and
  one-click restoration to recommended defaults. It is guarded by
  `settings.manage`.
- Forum taxonomy options now include the public default posting category slug
  and public tag-policy controls. Updating `forum.default_category_slug`
  requires `category.manage`; updating `forum.tags.*` options requires
  `tag.manage`.
- Forum pagination options are public runtime values named
  `forum.pagination.topics_per_page` and
  `forum.pagination.comments_per_page`. Both default to 20, accept 1-100,
  require `settings.manage` to update, and participate in the forum settings
  one-click reset.

## Boundaries

- Runtime options are for self-hosted operator-managed settings such as site
  identity, site URL, enabled locales, default locale, registration policy,
  content policy, appearance preferences, and CAPTCHA provider configuration.
- Do not move infrastructure or secret settings into `web_options`: database
  URLs, Redis passwords, Meilisearch master keys, worker counts, ports, and
  build-time route prefixes should remain in environment config.
- Admin-managed secrets in `web_options` are limited to approved runtime
  provider secrets such as ALTCHA and attachment storage credentials. They are
  never returned by public APIs and are masked in admin API responses; admins
  only see whether a secret is configured.
- Keep the table intentionally simple (`name`, `value`). Add typed validation in
  the Options service instead of adding per-option columns prematurely.
- Expose only settings that are safe for the frontend to read through public
  `GET /web-options` responses.
- Avatar settings are product behavior, not attachment-provider settings. They
  live in `web_options` under `avatar.*`, use `settings.manage`, and keep
  provider-specific storage behavior in the attachment module.

## Implementation Notes

- Current public options are `site.name`, `site.url`, `site.default_locale`,
  `site.supported_locales`, `human_verification.provider`,
  `human_verification.scenarios.register`,
  `human_verification.scenarios.password_reset`,
  `human_verification.scenarios.login_risk`,
  `human_verification.scenarios.post_risk`,
  `human_verification.altcha.widget.type`,
  `human_verification.altcha.widget.auto`,
  `human_verification.altcha.widget.display`,
  `human_verification.altcha.widget.hide_logo`,
  `human_verification.altcha.widget.hide_footer`,
  `human_verification.altcha.widget.workers`,
  `human_verification.altcha.widget.min_duration_ms`, `identity.password.min_length`,
  `identity.password.max_length`, `identity.password.require_lowercase`,
  `identity.password.require_uppercase`, `identity.password.require_number`,
  `identity.password.require_symbol`, `appearance.theme`,
  `footer.copyright.zh-CN`, `footer.copyright.en-US`, `footer.links`,
  `forum.default_category_slug`, `forum.tags.creation_mode`,
  `forum.tags.public_pages`, `forum.tags.max_per_topic`,
  `attachment.upload.enabled`, `attachment.max_file_size_mb`,
  `attachment.allowed_extensions`, `attachment.allowed_mime_types`,
  `avatar.allow_upload`, `avatar.default_provider`,
  `avatar.gravatar_base_url`, `avatar.gravatar_hash_algorithm`,
  `avatar.max_size_kb`, `avatar.allow_gif`, and
  `avatar.compress_enabled`.
- Current admin-only options are `human_verification.altcha.secret`,
  `human_verification.altcha.challenge_ttl`,
  `human_verification.altcha.cost`, attachment provider selection, path
  template, public base URL, default visibility, cleanup retention, all
  provider-specific credential/connection options, and avatar-only processing
  knobs `avatar.default_static_url`, `avatar.max_dimension`,
  `avatar.target_dimension`, and `avatar.compress_quality`.
- Startup environment values are treated as first-run defaults/fallbacks.
  `bootstrap.NewAPI` calls `EnsureDefaults` so missing option rows are inserted
  without overwriting existing admin-managed values.
- `web_options` uses `name` as the primary key. This makes single-option reads
  and upserts index-backed.
- The Options service returns defaults when rows are missing, so fresh installs
  and partially migrated databases still have a usable site name.
- The frontend root component loads public options once and falls back to
  defaults if the API is temporarily unavailable.
- Human-verification scenario options are public because pages need to decide
  whether to render the ALTCHA widget. API verification remains authoritative:
  the runtime verifier skips disabled purposes and enforces enabled purposes.
- ALTCHA widget behavior options are public frontend configuration for the
  web component only. They cover safe display/trigger knobs (`type`, `auto`,
  `display`, hide logo/footer, workers, and minimum duration); server
  challenge creation and token verification are still controlled by the API.
- SEO settings are frontend-safe public options under `seo.*`. They cover
  default meta/social tags, search platform verification tokens, robots
  additions, sitemap generation, and structured-data toggles. Runtime output
  still applies local/preview noindex protection based on `site.url`.
- Attachment settings use `web_options` for product runtime behavior,
  secret-masked provider credentials, and the local provider filesystem root.
  `attachment.local.root` defaults to `storage/app/attachments`; deployments
  should mount storage there or update the option to a prepared writable path.
- Locale settings can only enable built-in locale catalogs (`zh-CN`, `en-US`);
  adding a new locale still requires adding frontend and backend translations.
- Appearance preset settings use `appearance.theme` as a single public option.
  The stored key is unchanged for compatibility, but user-facing UI/docs should
  call it "配色预设 / appearance preset" rather than "theme". It accepts preset
  keys (`pine_teal`, `ocean_blue`, `violet`, `rose`, or `amber`) and the
  controlled custom format `custom:#rrggbb`.
- Footer settings are frontend-safe public options: copyright text supports
  `{year}` and `{siteName}`, and `footer.links` stores the fixed Terms,
  Privacy, and Guidelines links as normalized JSON.
- Forum taxonomy defaults are frontend-safe public options:
  `forum.default_category_slug=general`,
  `forum.tags.creation_mode=controlled`, `forum.tags.public_pages=enabled`,
  and `forum.tags.max_per_topic=5`. The Options service validates tag creation
  mode as `controlled`, `review`, or `open`; public tag pages as
  enabled/disabled; max tags per topic as `0..10`; and default category as a
  slug-shaped string.
- Password policy defaults are frontend-safe public options:
  `identity.password.min_length=12`, `identity.password.max_length=128`, and
  all composition requirements disabled. The Options service validates minimum
  length as `8..128`, maximum length as `64..512`, requires
  `max_length >= min_length`, and normalizes composition toggles to
  `enabled`/`disabled`.
- Avatar defaults are upload enabled, local initials fallback, Gravatar base
  URL `https://gravatar.com/avatar/`, Gravatar hash `sha256`, GIF disabled,
  compression enabled, max upload 2048 KB, source max edge 2048 px, output
  256x256, and JPEG quality 85. `static` fallback is valid only when
  `avatar.default_static_url` is configured.

## Next Steps

- If settings need audit history, write changes to the existing audit event
  pattern instead of adding columns to `web_options`.
- When `sqlc` is available in the local toolchain, generate typed query methods
  for `database/queries/options.sql` and replace the small pgx adapter queries.

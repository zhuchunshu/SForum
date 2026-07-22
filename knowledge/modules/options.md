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
  for basic site settings, account security (password + sessions + login
  lockout), registration/username policy, newcomer trust limits, maintenance
  mode, and CAPTCHA/human-verification settings.
- Admin page `apps/web/app/pages/admin/personalization.vue` is the unified
  personalization hub under the System folder: appearance preset + footer
  (requires `settings.appearance.manage`), plus brand/nav/announcements/legal/
  friend-links tabs (requires `settings.site.manage`). Page access is `any` of
  those two permissions; tabs filter client-side. Legacy `/site-chrome` redirects
  to `/personalization?tab=…`.
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
- Forum content-limit options are also public runtime values under
  `forum.topics.*`, `forum.comments.*`, `forum.tags.min_per_topic`, and
  `forum.reading.excerpt_rune_limit`. They require `settings.manage` (tags
  min/max require `tag.manage`) and participate in the forum settings reset.
- Wave 1 community policy pack (2026-07-12) lives mainly in
  `community_policy_options.go` (init-appended definitions) plus forum/identity
  adapters:
  - Registration: `identity.registration.mode`
    (`open|invite|approval|closed`), email-verify flags; non-`open` modes
    close public self-registration for now (invite/approval flows later).
  - Username: min/max length, charset, reserved names.
  - Login lockout: `identity.login.max_failures` /
    `identity.login.lockout_minutes` (Redis-backed).
  - Newcomer trust: `trust.new_user_*` cooldowns, daily caps, outbound-link /
    attachment forbids.
  - Maintenance: `site.maintenance.enabled|message` (middleware blocks non-admin
    writes; auth + `/admin/*` remain available to turn it off).
  - Forum reading/behavior: `forum.guest.read`, list default sort/hot window,
    author close/delete, edit marks, duplicate title policy, soft-delete
    visibility, mentions.
  Recommended defaults + validation + public exposure follow the same
  beginner-friendly pattern as other option groups.
- Wave 2 brand & legal (2026-07-12) lives in `site_brand_options.go`:
  - Brand assets (public): `site.logo_url` / `site.logo_attachment_id`,
    favicon and apple-touch URL + attachment id pairs. Empty → theme default.
  - Legal Markdown stubs (public): `legal.terms|privacy|guidelines.body.zh-CN|en-US`
    with recommended short stubs and 50k-rune cap.
  - Structured public chrome (nav, friend links, announcements) is **not** in
    `web_options`; see SiteChrome module / migration
    `202607120003_site_chrome.sql` and admin personalization tabs (panel
    `apps/web/app/components/admin/SFAdminSiteChromePanel.vue`).

## Feature flags (F4.5)

- Host catalog under `features.*` (search, registration, attachments, mentions,
  public_profiles, webhooks). Distinct from RBAC: flags kill product surfaces;
  permissions answer who may act when the surface is on.
- Admin: `GET/PUT /admin/features`, `POST /admin/features/restore-defaults`.
- Public `GET /web-options` includes only `public: true` flags (webhooks is
  admin-only).
- Plugins may declare `requiresFeatures`; enable fails if any flag is off.

## Page Registry / runtime theme flags

Registered in `apps/api/app/Models/Options/pages_registry_options.go`. Distinct
from `features.*` product surfaces. Public keys so the host can decide outlet /
skin behavior without admin session.

| Key | Default (post-P5) | Role |
| --- | --- | --- |
| `pages.registry_enabled` | `true` | Page catalog resolve + admin Pages UI |
| `themes.runtime_l0_enabled` | `true` | L0 CSS/assets without Nuxt rebuild |
| `themes.runtime_l1_enabled` | `true` | L1 template replace/add path |

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

- Current public options are `site.name`, `site.url`, `site.tagline`,
  `site.default_locale`, `site.supported_locales`, `site.timezone`,
  `site.date_format`, `site.time_format`, `site.start_of_week`,
  `human_verification.provider`,
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
  `identity.password.require_symbol`, `identity.registration.enabled`,
  `appearance.theme`,
  `footer.copyright.zh-CN`, `footer.copyright.en-US`, `footer.links`,
  `forum.default_category_slug`, `forum.tags.creation_mode`,
  `forum.tags.public_pages`, `forum.tags.max_per_topic`,
  `attachment.upload.enabled`, `attachment.max_file_size_mb`,
  `attachment.allowed_extensions`, `attachment.allowed_mime_types`,
  `avatar.allow_upload`, `avatar.default_provider`,
  `avatar.gravatar_base_url`, `avatar.gravatar_hash_algorithm`,
  `avatar.max_size_kb`, `avatar.allow_gif`, and
  `avatar.compress_enabled`.
- Current admin-only options are `site.admin_email` (operator contact, not
  SMTP From and not public), `human_verification.altcha.secret`,
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
- Site datetime display options are public runtime values managed under
  Site Settings → Basic. Defaults: `site.timezone=UTC`,
  `site.date_format=Y-m-d`, `site.time_format=H:i`, `site.start_of_week=1`
  (Monday). Timezone must be a valid IANA name (`time.LoadLocation`); date and
  time formats are whitelist presets. Storage remains UTC; frontend
  `useSiteDateTime()` / `utils/siteDateTime.ts` format timestamps for display.
  Admin UI supports one-click restore of recommended datetime defaults.
- `site.tagline` is optional public short text (max 160 runes) for navbar/auth
  branding; it is not SEO description (`seo.home.description`). Default theme
  `SFNavbar` shows it under the site name when non-empty.
  `site.admin_email` is optional admin-only contact email (RFC address form,
  max 254), distinct from mail provider From address. It is the default
  recipient for `POST /admin/mail/test` when the request omits `recipient`,
  and the mail admin UI prefills that field from admin web-options.
- Admin datetime surfaces (attachments, search rebuild, extension events /
  releases) format timestamps via `useSiteDateTime()` so they follow
  `site.timezone` / date / time presets instead of browser `toLocaleString`.
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
- Open registration is `identity.registration.enabled` (public, default
  `enabled`), managed under Site Settings → Account security. When disabled,
  `POST /auth/register` returns `403 auth.register_disabled` after the site
  already has users. Bootstrap (no users yet) always allows registration so a
  fresh install cannot lock itself out. Public `GET /auth/registration-status`
  exposes `registrationEnabled` with that bootstrap override; it still never
  leaks the super-admin bootstrap window via `nextUserIsInitialSuperAdmin`.
- Avatar defaults are upload enabled, local initials fallback, Gravatar base
  URL `https://gravatar.com/avatar/`, Gravatar hash `sha256`, GIF disabled,
  compression enabled, max upload 2048 KB, source max edge 2048 px, output
  256x256, and JPEG quality 85. `static` fallback is valid only when
  `avatar.default_static_url` is configured.

## Next Steps

- Wave 3+ richness blueprint: engagement switches, category policy, safety
  depth (see `knowledge/plans/2026-07-12-admin-settings-richness.md`).
- If settings need audit history, write changes to the existing audit event
  pattern instead of adding columns to `web_options`.
- When `sqlc` is available in the local toolchain, generate typed query methods
  for `database/queries/options.sql` and replace the small pgx adapter queries.

## SEO Workbench V2 P0 (2026-07-11)

- Product identity and search identity are separate. `site.name` remains the UI
  application name; `seo.site.inherit_site_name` and `seo.site.name` resolve the
  SEO brand name.
- Homepage settings use `seo.home.{title,description,keywords,og_title,
  og_description,og_image_url}`. Inner-page fallbacks use `seo.page.*`.
- Category, tag, topic, profile, and static-page policies use
  `seo.content_type.<type>.*` with title template, description sources, default
  image, index mode, sitemap inclusion, and schema type.
- Unknown template variables and invalid policy enums are rejected by the
  Options service. Existing v1 meta values remain compatible fallbacks.
- The v2 recommended default enables forum-content Sitemap generation. Existing
  stored operator values are not overwritten.
- `seo.content_type.*.schema_type` uses `normalizeStringChoice` (EqualFold +
  canonical PascalCase). Do not use `normalizeChoice` here: that helper
  lowercases the input and exact-matches against the allow-list, so legal
  Schema.org values like `CollectionPage` were always rejected and blocked
  every admin SEO save (`options.invalid`).

# Knowledge Index

This is the entry point for project memory.

## Latest Handoff

- **2026-07-12 E1.4 complete** (`attachment.before_upload` validate)
  - Handoff: `knowledge/sessions/2026-07-12-e1-4-attachment-before-upload.md`
  - Plan: `knowledge/plans/2026-07-12-extension-surface-density.md`
  - kind=`validate` reject-only; metadata payload only (no raw bytes); gate
    in `storePreparedUpload` (Upload / Avatar / SEO image); 422 mapping
  - **E1 core done** (≥4 sync hooks); next **E2** or **E6** service plugins

- **2026-07-12 E1.3 complete** (`user.before_register` validate)
  - Handoff: `knowledge/sessions/2026-07-12-e1-3-user-before-register.md`
  - Plan: `knowledge/plans/2026-07-12-extension-surface-density.md`
  - kind=`validate` reject-only; payload username/email/locale only (no
    password); wired on ValidateRegister + Register; identity 422 mapping

- **2026-07-12 E1.2 complete** (`topic.before_update` filter)
  - Handoff: `knowledge/sessions/2026-07-12-e1-2-topic-before-update.md`
  - Plan: `knowledge/plans/2026-07-12-extension-surface-density.md`
  - Sync filter after edit permission + author edit window; patch allowlist
    same as create; plugins may force tags even if request omits them

- **2026-07-12 E1.1 complete** (`comment.before_create` filter)
  - Handoff: `knowledge/sessions/2026-07-12-e1-1-comment-before-create.md`
  - Plan: `knowledge/plans/2026-07-12-extension-surface-density.md`
  - Sync filter after auth/topic-active; patch allowlist `content` only;
    reject → `RejectedError` / 422; catalog docs + authoring guide updated

- **2026-07-12 Wave F4 complete** (entity meta + feature flags)
  - Handoff: `knowledge/sessions/2026-07-12-f4-4-f4-5-meta-and-flags.md`
  - Decision: `knowledge/decisions/2026-07-12-entity-meta-and-feature-flags.md`
  - F4.4: EAV custom fields on user/topic; F4.5: `features.*` + requiresFeatures
  - Framework hardening F1–F4 done; density plan E1–E8 still open (E1.1 done)

- **2026-07-12 extension surface density plan**
  - Plan: `knowledge/plans/2026-07-12-extension-surface-density.md`
  - Waves E1–E5: filters → public contributions → entity meta → flags →
    workflow reference plugin
  - **North star E6–E8:** storage / search / other services fully
    plugin-selectable and configurable (mail-like L4–L6); not slot names only
  - E3/E4 overlap F4.4/F4.5 (now implemented); **E1.1–E1.4 done** (E1 core);
    next **E2** or **E6**

- **2026-07-12 posts content storage slim**
  - Handoff: `knowledge/sessions/2026-07-12-posts-excerpt-revision-slim.md`
  - Drop `posts.excerpt`; derive API excerpt from `plain_text` at read time
  - `post_revisions` source-only (no html/plain/excerpt columns)

- **2026-07-12 Wave F4.3 complete** (contribution point expansion)
  - Handoff: `knowledge/sessions/2026-07-12-f4-3-contribution-points.md`
  - Plan: `knowledge/plans/2026-07-12-framework-hardening-waves.md`
  - New points: composer toolbar, profile tabs, dashboard widgets, health checks

- **2026-07-12 Wave F4.2 complete** (catalog → documentation)
  - Handoff: `knowledge/sessions/2026-07-12-f4-2-catalog-docs.md`
  - Plan: `knowledge/plans/2026-07-12-framework-hardening-waves.md`
  - Generated `docs/extensions/catalogs/*`; `sforum extension docs generate`
  - Authoring guide: `docs/extensions/authoring-guide.md`

- **2026-07-12 Wave F4.1 complete** (SDK + contract tests)
  - Handoff: `knowledge/sessions/2026-07-12-f4-1-sdk-contract-tests.md`
  - Plan: `knowledge/plans/2026-07-12-framework-hardening-waves.md`
  - Public `apps/api/sdk/plugin`; `sforum extension test`; fixtures under
    `extensions/fixtures/plugins/`

- **2026-07-12 Wave F3 complete** (integration & reliability)
  - Handoff: `knowledge/sessions/2026-07-12-f3-integration-reliability.md`
  - Plan: `knowledge/plans/2026-07-12-framework-hardening-waves.md`
  - Outbox status machine; Idempotency-Key; webhooks; PAT; storage slot

- **2026-07-12 F2.4 Extension lifecycle**
  - Handoff: `knowledge/sessions/2026-07-12-f2-4-extension-lifecycle.md`
  - Same-id upgrade, uninstall, migration ledger (record-only), disable drain
  - Wave **F2 complete** for current scope

- **2026-07-12 F2.3 Plugin RPC resilience**
  - Handoff: `knowledge/sessions/2026-07-12-f2-3-rpc-resilience.md`
  - Per-extension concurrency + circuit breaker; hook/mail deadlines
  - Runtime `degraded` + admin circuit/failure UI

- **2026-07-12 F2.1 + F2.2 capabilities and Host API v1**
  - Handoff: `knowledge/sessions/2026-07-12-f2-capabilities-host-api.md`
  - Decision: `knowledge/decisions/2026-07-12-host-api-v1-capabilities.md`
  - Capability catalog + manifest field + enable confirm UI + grants on list
  - Host API loopback gateway, Client stubs, `extension.plugin_job`

- **2026-07-12 App store split to top-level menu (themes / plugins)**
  - Handoff: `knowledge/sessions/2026-07-12-admin-extension-store-menu-split.md`
  - Sidebar: 应用商城 as independent folder under 扩展管理
  - Routes: `/extensions/store/themes`, `/extensions/store/plugins`
  - Legacy `/extensions/store` → plugins shelf

- **2026-07-12 API error message i18n gap filled**
  - Handoff: `knowledge/sessions/2026-07-12-api-error-message-i18n-gap.md`
  - Missing catalog keys (e.g. `site_chrome.invalid`) no longer returned raw
  - Regression test walks `Code*` + `fiber.NewError` reasons vs messages.go

- **2026-07-12 fix sforum.smtp runtime failed after RouteTarget SSRF check**
  - Host accepts empty/`disabled`/`none` BaseURL for provider-only plugins
  - SMTP `RouteTarget` returns empty; rebuild via `build-builtin-plugins.sh`
  - Code: `apps/api/app/Support/Extensions/protocol.go`,
    `extensions/builtin/plugins/sforum-smtp/backend/plugin.go`

- **2026-07-12 Schedule ops workbench (enable / last-next / trigger / menu)**
  - Handoff: `knowledge/sessions/2026-07-12-schedule-ops-workbench.md`
  - Admin `/schedules`: enable/disable, last/next run, manual trigger
  - Options key `jobs.schedule.<id>.enabled`; worker skips when disabled
  - Jobs page links to dedicated schedules menu under 运维管理

- **2026-07-12 site-chrome merged into personalization**
  - Handoff: `knowledge/sessions/2026-07-12-personalization-site-chrome-merge.md`
  - Admin `/site-chrome` redirects to `/personalization?tab=…`; sidebar entry removed
  - Larger md tabs for appearance + brand/nav/announcements/legal/friend links

- **2026-07-12 Wave F1.3 + F1.4 complete (F1 done)**
  - Handoff: `knowledge/sessions/2026-07-12-f1-3-f1-4-events-audit.md`
  - F1.3: event catalog `failurePolicy`/`timeoutMs`; sync filter host timeout;
    sync deliveries + slow/failed reasons in event log
  - F1.4: settings + extension lifecycle → `audit_events`; daily
    `audit.cleanup_events` (90d); permission audits already in identity
  - F2.1/F2.2 landed; remaining F2.3/F2.4 or product tracks

- **2026-07-12 Wave F1.2 Ready + worker heartbeat implemented on main**
  - Handoff: `knowledge/sessions/2026-07-12-f1-2-ready-heartbeat.md`
  - `GET /api/v1/ready` (PG required; Redis/Meili degraded-ready)
  - Redis worker heartbeat; admin overview worker stale + queue lag

- **2026-07-12 Wave F1.1 Schedule Registry implemented on main**
  - Handoff: `knowledge/sessions/2026-07-12-f1-1-schedule-registry.md`
  - Code: `app/Support/Jobs` schedule catalog; worker builds River periodics
    only via registry; core schedules: session cleanup, web-release cleanup,
    attachment orphan cleanup (daily)
  - Admin: `GET /admin/jobs/schedules` + Jobs workbench read-only section

- **2026-07-12 host platform capabilities + phased hardening plan recorded**
  - Decision: `knowledge/decisions/2026-07-12-host-platform-capabilities.md`
  - Waves F1–F4: `knowledge/plans/2026-07-12-framework-hardening-waves.md`
  - Handoff: `knowledge/sessions/2026-07-12-framework-hardening-plan.md`
  - Architecture direction for schedule/health/Host API/capabilities/etc.

- **2026-07-12 security audit P0–P2 fixes applied on main**
  - Plan: `knowledge/plans/2026-07-12-security-audit-fix-batch.md` (commits 1–12 done)
  - Handoff: `knowledge/sessions/2026-07-12-security-audit-handoff.md`
  - Identity: no `user.manage`→`permission_override`; super_admin demote gated;
    password-reset 4xx + Redis rate limit + atomic confirm
  - Extensions: strip proxy identity headers; loopback RouteTarget; minimal
    plugin env; disable perm parity; zip inflate cap
  - Attachments: server MIME sniff + active content deny under wildcards
  - Forum: publication policy on edits; delete/move counter integrity
  - HTTP: CSRF CookieSecure aligned with session (`config.ShouldUseSecureCookie`)
  - Verification: `cd apps/api && go test ./...` green
  - Out of scope (unchanged): non-builtin enable super_admin, secret AES,
    settings PUT merge, guest attachment read, login lockout IP dimension, etc.

- **2026-07-12 Settings Wave 2 done** (Brand & public chrome)
  - Session: `knowledge/sessions/2026-07-12-admin-settings-wave2.md`
  - Decision: `knowledge/decisions/2026-07-12-site-chrome-tables.md`
  - Blueprint: `knowledge/plans/2026-07-12-admin-settings-richness.md`
  - Prior Wave 1: `knowledge/sessions/2026-07-12-admin-settings-wave1.md`
  - Next: Wave 3 engagement switches (with Iteration A), or polish attachment
    URL resolution / richer legal Markdown render.

- **2026-07-12 built-in role templates done** (`moderator` / `operator` /
  `tech_admin`)
  - Session: `knowledge/sessions/2026-07-12-builtin-role-templates-handoff.md`
  - Decision: `knowledge/decisions/2026-07-12-builtin-role-templates.md`
  - Prior Phase 1: `knowledge/sessions/2026-07-12-fine-grained-permissions-phase1-handoff.md`
  - Next: optional restore-template defaults action, or category-scoped ACL when
    needed — not required immediately.

## Current Project State

- Admin forum settings are multi-tab
  (general/topics/comments/tags/reading/behavior) with configurable
  topic/comment length limits, nesting depth, edit windows, cooldowns, daily
  caps, tag min/max, excerpt length, guest read, list sort, author actions,
  edit marks, duplicate titles, soft-delete visibility, and mentions. Limits
  are server-authoritative and exposed via public web-options for composer UX.
- Wave 1 community policy pack (2026-07-12): site settings tabs for
  registration mode/username rules, newcomer trust ladder, maintenance mode,
  and login lockout; forum guest-read enforcement; recommended defaults +
  restore. Blueprint: `knowledge/plans/2026-07-12-admin-settings-richness.md`.
- Wave 2 brand & public chrome (2026-07-12): logo/favicon/legal body options;
  `site_nav_items` / `site_friend_links` / `site_announcements` tables + public
  and admin CRUD; admin UI lives under **个性化设置** tabs (legacy
  `/site-chrome` redirects); default theme navbar/footer/announcement banner +
  `/terms|/privacy|/guidelines` pages.


- Mail and in-app notifications are implemented as a plugin-first vertical:
  core owns durable inbox/delivery records, River scheduling, provider
  selection, and permissions, while protected built-in plugin `sforum.smtp`
  exclusively owns SMTP/TLS/auth transport. Replies, mentions, moderation
  results, password reset, test mail, unread UI, and legacy SMTP adoption are
  covered.
- The Core admin now exposes a visible Mail and Notifications center with
  provider selection, global event/channel policy, custom-recipient test mail,
  current-admin test notifications, and delivery history. SMTP-specific setup
  guidance remains declared and implemented by the built-in SMTP plugin.

- Repository initialized on 2026-07-03.
- Basic documentation and knowledge-base skeleton created.
- First application scaffold has been added under `apps/web` and `apps/api`.
- Forum architecture stack has been proposed and foundation scaffolding has
  started.
- Proposed stack: Nuxt 4/Vue 3/Nuxt UI/Bun frontend; Go Fiber v3,
  PostgreSQL, Redis, and Meilisearch backend.
- Development/deployment workflow has been proposed: Docker Compose for
  production orchestration and local dependency services, local `bun run dev`
  plus `air` for frontend/API hot reload, and bilingual `deploy.sh` for
  production operations.
- `scripts/dev.sh` now starts only development dependencies: PostgreSQL, Redis,
  Meilisearch, and Mailpit. It stops old Compose-managed frontend/backend
  containers, waits for dependencies, and runs migrations by default; use
  `--no-migrate` only when testing dependency startup.
- Local frontend and backend processes read the repository root `.env`
  directly: the frontend dev supervisor loads it with `bun --env-file=../../.env`
  and exposes `PORT`/`WEB_PORT` through its fixed proxy, Nuxt dev still uses
  `--dotenv ../../.env`, Air uses `env_files = ["../../.env"]`, and
  `./scripts/api-dev.sh` is the recommended API entry because it loads `.env`
  and reports occupied API ports without stopping user processes. In
  development, the API embeds the background worker by default through
  `EMBED_WORKER_IN_API=true`, so running API `air` also consumes queued jobs
  such as uploaded theme activation. Production keeps API and worker processes
  separate by default.
- Frontend build/typecheck commands use sibling Nuxt temporary directories
  (`.nuxt-build` and `.nuxt-typecheck`) instead of nesting under the dev
  server's `.nuxt`, and generated output is ignored by dev watchers to avoid
  repeated reloads.
- Frontend production preview runs the generated Nitro server directly through
  `HOST=0.0.0.0 bun --env-file=../../.env .output/server/index.mjs`, because
  the installed `nuxi preview` command does not support `--host` and misreads
  the host value as `ROOTDIR`.
- Nuxt top-level `ignore` rules are intentionally narrower than Vite watcher
  ignores so dependency packages such as `@nuxt/ui/dist` remain discoverable by
  Nuxt component auto-imports.
- Development Compose publishes dependency services to loopback-only host ports
  so local `air` and Nuxt can connect. Production Compose still publishes only
  the `web` service to `127.0.0.1:${WEB_PORT}`, with API and internal services
  staying on the Compose network.
- Product internationalization is required from the first implementation.
  Default locale is Simplified Chinese (`zh-CN`); first secondary locale is
  English (`en-US`).
- The standalone `forum-components.html` demo uses a Pine Teal clean forum UI
  direction: teal primary actions, light surfaces, thin borders, reduced
  radius, no gradient backgrounds, no emoji icons, and a dedicated
  status/feedback component section.
- The Pine Teal demo has been split into a reusable Nuxt component library under
  `apps/web/app/components/` using the uppercase `SF` prefix. The first library
  slice includes buttons, cards, inputs, toggles, avatars, feed rows, comments,
  search, editor, pagination, progress, skeleton, empty state, alerts, badges,
  toasts, and tabs, with a dev-only `/components` preview page. The preview page
  now covers seven forum-oriented sections: foundations, feedback, forum list,
  composer flow, moderation, member profile, and loading/empty states.
- `SFEditor` is now a Tiptap-based editor with a custom toolbar, custom emoji
  node, preview mode, Markdown source mode, native JSON inspection, and a
  `content-change` payload for HTML/Markdown/native content previews.
- The default topic composer submits editor Markdown under the backend
  `content` request field and accepts Unicode tag slugs, so Chinese tags can be
  used directly while publishing.
- Forum authored content has an accepted triple-storage direction:
  `content_html_sanitized`, `content_markdown`, and `content_native_json`.
  Client HTML remains untrusted; the API must accept allowlisted content,
  regenerate derived formats, and sanitize display HTML before storage.
- The protected built-in default theme follows V32 暖橙左栏: sticky topbar,
  240px left category navigation, dense topic table (not card magazine flow),
  and topic detail dual-column (main article/comments + 280px info side card).
  Accent color still comes from `appearance.theme` (`--sf-accent`); warm orange
  is available via `amber` / `custom:#c2410c`. Homepage keeps URL-backed
  filters and SSR-first infinite scrolling without fabricated unread/hot/mine
  or participant stacks.
- The default-theme homepage feed uses client-side infinite scrolling: page 1
  remains SSR-loaded, the loaded feed is preserved through Nuxt state for
  hydration, and the left sidebar is sticky with internal scrolling.
- Base homepage responses keep shared-cache headers, while homepage query
  variants disable Nitro payload caching and return `cache-control: no-store`.
  This avoids the root-route payload file-key collision that otherwise causes
  `EISDIR` errors for URLs such as `/?q=...`.
- The thread feed row component (`SFFeedRow.vue`) has been redesigned using a compact no-excerpt layout (Left author avatar, Right title and upvote/reply actions inline, and bottom row metadata/views), doubling the layout information density.
- Sidebar accessibility was improved by fixing double padding via the `flush` property and updating text colors to `slate-500` and `slate-600` for higher contrast.
- The admin foundation now uses a dedicated Nuxt UI Dashboard shell with Nuxt
  Icon lucide icons. Source files stay under `apps/web/app/pages/admin`, while
  Nuxt rewrites the public route prefix from `NUXT_PUBLIC_ADMIN_ROUTE_PREFIX`,
  defaulting to `/control-panel`.
- Admin pages now use a low-code module registry under
  `apps/web/app/config/adminModules.ts` for sidebar entries, tab metadata,
  keep-alive component names, badges, and frontend-visible permission
  requirements. Page components register with `useAdminPage('/id')` instead of
  repeating tab/menu metadata.
- The admin personalization page remains on `/personalization`, but its
  sidebar entry now lives inside the System configuration folder.
- UI feedback guidelines now favor Toasts for user-triggered success,
  completion, copy, upload, export, reset, queued-job, and authentication
  success states. Non-error alerts/toasts support 10-second auto-dismiss
  behavior, while blocking errors and field-level validation remain visible
  until user dismissal or resolution.
- The public forum navbar no longer shows an admin entry in the logged-in user
  dropdown, avoiding direct exposure of the configurable admin route prefix.
- The public forum navbar now has a client-rendered Light/Dark mode toggle
  wired to Nuxt Color Mode. Public theme and SF component CSS include dark
  semantic variables for the forum homepage surfaces, keeping navbar, cards,
  search, feed rows, tabs, pagination, and footer chrome aligned with the
  admin shell's `.dark` class.
- Identity and permissions architecture is accepted: SForum uses one user
  system for regular users, moderators, and administrators; the first registered
  user becomes the protected initial `super_admin`; later open registrations
  receive the undeletable default `member` role; admin-managed custom
  roles/user groups are supported.
- The first admin permission management release is implemented: role RBAC now
  supports per-user permission overrides where ordinary users inherit enabled
  user-group permissions plus direct allows minus direct denies. Active
  `super_admin` users still pass every policy check and cannot receive direct
  permission overrides. The admin UI now includes user management, editable
  user-group permissions, and a permission matrix. The permission matrix now
  defaults to a limited user-group comparison view with search, explicit group
  selection, and differences-only auditing so it stays readable as custom
  user groups grow.
- Moderation now has two independent surfaces and permissions: admin
  **Moderation management** (`moderation.manage`) configures `off`/`rules`/`all`
  publication review and reads the audit trail, while the frontend
  `/moderation` workbench (`moderation.review`) processes pending topics,
  comments, and enriched reports. Pending/rejected content stays outside
  public reads, counts, and search; authors track it under `/my/content-review`.
- Role/user-group management now validates required fields at both the Nuxt
  roles form and Go identity service boundary. Empty custom roles created by
  the earlier missing validation are cleaned up by migration
  `202607060002_role_input_constraints`, and the roles table has non-blank key
  and alias checks.
- Development guidelines now require permission-aware feature design: new
  protected routes, mutations, admin screens, exports, setting updates, and
  background action triggers must identify their actor/action/resource boundary,
  keep API policy checks authoritative, and test allowed plus denied paths when
  behavior is unsafe or admin-facing.
- Admin database table management is implemented as a core, read-only
  PostgreSQL browser. It requires `database.manage`, excludes system schemas,
  shows table metadata and paged rows, masks sensitive columns by default, and
  allows one-cell reveal for primary-keyed rows.
- Admin home now uses a one-shot `GET /api/v1/admin/overview` command-center
  feed requiring `admin.access`. The endpoint aggregates runtime API memory,
  Go heap/GC/goroutine data, database pool stats, users, topics/posts,
  attachments, moderation, extensions, 7-day trends, top categories, and
  server-generated safe action summaries. The Nuxt admin index renders this as
  the bilingual "均衡指挥台 / Balanced Command Center" using Nuxt UI Dashboard
  cards and CSS-only trend bars.
- Development guidelines now require beginner-friendly defaults for new
  features: configurable flows must ship with safe recommended defaults,
  explain the recommended path in plain language, and support one-click
  restoration to defaults.
- Security verification architecture is accepted: SForum keeps human
  verification disabled by default, with ALTCHA as the first supported
  self-hosted provider for registration, password-reset initiation, and later
  risk-based actions when deployments enable it, paired with Redis-backed rate
  limits and single-use challenge tracking.
- ALTCHA-backed registration human verification is implemented as an opt-in
  identity slice. The API exposes `/api/v1/human-verification/challenge`,
  verifies ALTCHA payloads before account creation when enabled, stores
  replay/rate-limit state in Redis, and the Nuxt registration page sends the
  widget token through `humanVerification` only when the public runtime provider
  is `altcha`.
- The registration page now reads `/api/v1/auth/registration-status` and, when
  no user exists yet, warns that the first registered user will become the
  super administrator.
- Login/register error feedback is now user-actionable: login failures keep a
  single generic `auth.invalid_credentials` reason for safety, while
  registration validation returns localized `data.fields` messages for
  username, email, password, and human verification.
- Registration now validates editable fields and username/email conflicts before
  consuming ALTCHA tokens, constructs the returned current-user access inside
  the bootstrap transaction, and reports post-create session failures as
  `auth.session_unavailable` so users are guided to log in instead of retrying
  registration.
- Login and registration pages now hydrate the frontend auth state directly
  from the successful API response before navigating, avoiding an extra session
  refresh and keeping admin-route middleware from misreporting a successful
  registration as a form failure. The registration page also shows the current
  password rule below the password input.
- Browser authentication uses Redis-backed server sessions rather than
  JWT-first auth. Sessions now have a 30-day idle timeout, 180-day absolute
  timeout, 24-hour session-id renewal, login-time session reset, production
  Secure cookies, and login audit records with IP, User-Agent, time, and salted
  session hash.
- Login credential lookup now distinguishes an actual missing credential from
  internal credential-loading failures, so schema drift or database errors are
  no longer misreported as wrong passwords.
- Frontend auth refresh now preserves the current user state during transient
  API restart/timeout/gateway failures, restores browser sessions during
  browser startup when the API is available, and only redirects to login on
  confirmed 401/`auth.required` responses. Root-app SSR refreshes web options
  only so public SWR pages do not cache user-specific auth payload. Ordinary
  protected user pages with
  `requiresAuth` stay server-rendered, disable route cache, and redirect
  missing users to login even when the auth API is temporarily unavailable,
  and the admin guard now follows the same no-Nuxt-error fallback instead of
  rendering a 503 error page.
  Admin and component-preview pages are now also server-rendered (the last
  `ssr: false` routes were removed), so the entire site renders first-paint
  HTML and never serves an empty SPA shell.
- Backend API code has migrated to a Laravel-style directory shape while
  staying Go-explicit: `cmd/api` is process-focused, `bootstrap` assembles the
  runtime, `app/Http` owns the HTTP kernel, `app/Http/Controllers/*` owns
  controllers and routes, `app/Providers` owns provider wiring,
  `app/Models/*` owns domain logic, and `database/*` owns migrations, SQL, and
  generated `sqlc` code.
- Forum taxonomy Phase 1 is implemented: two-level category groups/categories,
  core tags and topic-tag joins, configurable tag creation policy, public
  category/tag filtering pages, full-list public pages `/tags` (T01 weight
  cloud) and `/categories` (C04 tile grid) in the default theme, admin
  category/tag/settings screens, `category.manage`/`tag.manage` permission
  boundaries, forum events, and modular OpenAPI coverage. Tag slugs accept
  Unicode letters/numbers plus hyphens; category/default-category slugs remain
  ASCII.
- Admin taxonomy management now supports configurable icons and icon colors for
  categories and tags. The fields are stored on `categories` and `tags`,
  exposed through existing admin taxonomy endpoints, and previewed only in the
  admin category/tag lists for this release.
- Goose migrations now run from a shared embedded migrator. API and worker
  processes run migrations at startup when `MIGRATE_ON_STARTUP=true`, guarded
  by Goose's PostgreSQL table lock. `scripts/dev.sh` and `deploy.sh` may still
  run the same migration command explicitly as a visible pre-start check.
- Jobs and queues foundation implementation has started: River-backed durable
  queue support now lives under `apps/api/app/Support/Jobs`, `cmd/worker` uses
  `bootstrap.NewWorker`, and the first search job contract is
  `search.index_topic`.
- The Jobs operator workbench is implemented under the admin shell with
  `jobs.view`/`jobs.manage`, River queue state counts, bounded job inspection,
  retry/cancel and pause/resume controls, plus the first production trusted
  Vue slots: `admin.jobs.table.columns`, `admin.jobs.row.actions`, and
  `admin.jobs.detail.sections`.
- Backend API responses now have an accepted envelope design: every API JSON
  response must include integer `code`, localized `message`, and `data`; `code`
  equals the HTTP status code, and stable machine-readable reasons live under
  `data.reason`.
- OpenAPI has been modularized: `contracts/openapi.yaml` is now a small
  entrypoint, with module-owned paths under `contracts/openapi/paths/`, schemas
  under `contracts/openapi/schemas/`, shared components under
  `contracts/openapi/components/`, and local reference validation through
  `ruby scripts/validate-openapi-refs.rb`.
- Runtime web options are now introduced through `web_options(name, value)`.
  Site name, site URL, default locale, enabled locales, and public
  human-verification provider plus verification scenario switches and ALTCHA
  widget behavior settings are frontend-safe runtime options. Admin-only
  settings include masked ALTCHA secret metadata plus ALTCHA TTL/cost, editable
  from the admin settings page by users with `settings.manage`.
- Public forum pagination is operator-configurable through
  `forum.pagination.topics_per_page` and
  `forum.pagination.comments_per_page`. Both recommended defaults are 20;
  topic, search, and comment APIs resolve them when `perPage` is omitted, while
  explicit values remain capped at 100.
- Personalization settings extend runtime web options with `appearance.theme`
  preset keys or `custom:#rrggbb` colors plus frontend-safe footer content. The
  stored key remains `appearance.theme`, but UI language now calls it a
  "配色预设 / appearance preset" to keep it distinct from installable Nuxt Layer
  themes.
- The global footer has been implemented using the Option A (Single-line Minimalist) design direction, supporting dynamic copyright data, localized links (Terms, Privacy, Guidelines) mapped to placeholder links, and full Light/Dark mode responsiveness.
- SEO Full-Chain v1 is implemented: `seo.manage` controls the SEO admin page,
  typed `seo.*` runtime options cover meta/social, robots, sitemap, structured
  data, and verification settings, and public Nuxt pages use runtime SEO helpers
  with local/preview noindex protection. The topic detail URL shape is
  configurable via `seo.topic_url_mode` (`id_slug`|`id`|`slug`, default
  `id_slug`); the detail page is a catch-all `/t/[...path]` with SSR canonical
  301, and switching modes redirects old URLs to the new canonical path.
  Decision: `knowledge/decisions/2026-07-09-configurable-topic-url-mode.md`.
- `SFIconPicker` now loads the full local Tabler and Lucide catalogs through a
  Nuxt server-side name catalog route, shows paged results, auto-loads more
  icons when the picker grid scrolls near the bottom, and primes only visible
  icon SVG data through the existing Nuxt Icon local endpoint.
- Attachment system foundation is implemented: standalone admin top-level
  "Attachment settings" page, `attachments` and `attachment_references` tables,
  `attachment.upload`/`attachment.manage`/`attachment.settings.manage`
  permissions, runtime provider settings in `web_options`, server-mediated
  upload APIs, local/Aliyun OSS/Tencent COS/FTP/SFTP storage adapters, admin
  attachment governance, and orphan cleanup boundaries. The admin attachment
  settings page now highlights a beginner-friendly local-upload recommended
  configuration and can restore those defaults in one click.
- Avatar strategy is implemented as profile-owned behavior on top of
  attachments: uploaded public avatars take priority; otherwise profiles fall
  back to configurable `initials`, `gravatar`, or `static` sources. Avatar
  runtime options live under `avatar.*`, default to local initials plus enabled
  upload/compression, and are managed from System configuration with
  `settings.manage`.
- Current-user and forum author summaries now carry the same `AvatarView`
  shape. First-party frontend user-avatar surfaces should render through
  `SFAvatar` with that view instead of hand-written initials, `UAvatar`, or
  ad-hoc URL props.
- The local attachment provider root is now the admin-only runtime option
  `attachment.local.root`, defaulting to `storage/app/attachments`; API process
  config no longer reads `ATTACHMENT_LOCAL_ROOT`.
- Extension system foundation is implemented: `extension.manage`, extension
  ZIP upload, `sforum.extension.json` manifest validation, dedicated extension
  tables, lifecycle events, independent admin extension submenu pages,
  `EXTENSION_ROOT`, and reserved plugin/theme runtime boundaries. Multi-file
  manifest authoring is implemented: optional `includes`, directory-per-locale
  identity `langs`, settings/contributions shards, `LoadPackage`, SMTP
  reference package, `make:plugin --complex`, and `extension validate`. See
  `decisions/2026-07-12-extension-manifest-split.md`. Plugins use
  enable/disable semantics; themes use activation semantics with exactly one
  active theme. Uploaded Nuxt Layer themes can now be activated through a
  single-node self-hosted runtime: the API creates an `extension_theme_releases`
  row and queues a River `extension.theme_activate` job, the worker builds an
  isolated Nuxt/Nitro artifact and health-checks it, and the web supervisor
  follows `theme-releases/current.json` to switch Nitro servers while keeping
  the previous release available. The Themes page and Extensions overview both
  show queued/building/switching progress and poll while a theme activation is
  active. Uploaded themes are incremental Nuxt Layer overlays: their files
  override the protected default theme, and missing pages, layouts, components,
  or assets inherit from `sforum.default-theme`.
  Theme runtime now converges on `theme-releases/current.json` as the single
  selection signal for both production and local dev. `current.json` carries
  `mode` (`uploaded`/`default`), an absolute `server` (Nitro entry for
  production runtime.mjs), and an absolute `layerPath` (Nuxt Layer source for
  local dev). Restoring the built-in default theme now writes a `default`
  current.json synchronously from the API service, and local `bun run dev` is a
  theme-aware supervisor (`dev-theme-runtime.mjs`). Production `runtime.mjs`
  keeps blue-green Nitro switching and preserves the old server when a
  candidate fails. Local development intentionally owns one `nuxt dev`
  process: a `current.json` change clears the proxy target, stops and waits for
  the old process group, then starts the latest layer. Local switching has a
  brief development-only outage because parallel Nuxt dev instances would
  share the build lock, generated output, cache, and HMR resources.
  Plugin runtime v1 now starts enabled plugin subprocesses through HashiCorp
  go-plugin, proxies declared plugin routes, emits lifecycle hooks, and exposes
  provider slot defaults. Built-in sync prunes stale built-in extension rows,
  and verify/enable operations require the active package path and installed
  manifest to still exist. Container images now copy built-in themes from the
  repository root into
  `/app/extensions/builtin`; extension manifests now require `description`,
  `url`, and `author` metadata. Plugins and themes can declare core-container
  admin pages and extension settings under the fixed Extensions admin
  namespace, while themes still cannot declare backend runtime, routes, hooks,
  events, jobs, providers, migrations, or permissions in v1. Plugin event and
  extension-point v1 adds a host event catalog, manifest `events` declarations
  with legacy `hooks` compatibility, delivery tracking, and the first
  synchronous filter event for `topic.before_create`. The Go developer console
  at `apps/api/cmd/sforum` can scaffold plugins and themes interactively or via
  `--no-interaction`. Typed extension contributions are now implemented as a
  host-owned ordered registry inspired by old SForum's Itf mechanism: manifests
  may declare safe `contributions[]`, admins can inspect contribution points and
  active contributions, and `forum.topic.actions` is the first runtime consumer.
- Extension Platform v2 direction is accepted: SForum should offer
  WordPress-like operator ergonomics without copying WordPress' PHP include
  runtime. Plugins extend core only through manifests, subprocess RPC, provider
  slots, events/filters, controlled routes, settings, and host-owned admin
  pages. Sidebar menu injection is opt-in through manifest metadata, `Manage`
  resolves to an in-admin route, `mail.provider` is the first recommended full
  vertical slice, and theme activation still needs future preview approval,
  richer build logs, rollback UI, and multi-node rollout support.
- Trusted admin plugin runtime architecture is accepted but not implemented.
  Uploaded plugin Vue components will be treated as fully trusted, client-only
  admin code: a `super_admin` grant is bound to the exact package digest,
  validated manifest contributions provide SSR-safe slot metadata, a generated
  static registry maps component IDs, and one unified Web Release Runtime owns
  the active theme plus trusted plugin set. Workers build artifacts, the API
  coordinates plugin runtime and activation, and the web supervisor publishes
  the actual active acknowledgement. The River job monitoring module is a
  separate follow-up project and will register the first production component
  slots. Decision:
  `decisions/2026-07-10-trusted-admin-plugin-runtime.md`; specification:
  `../docs/superpowers/specs/2026-07-10-trusted-admin-plugin-runtime-design.md`.
- Architecture guidance now treats SForum core as the host framework rather
  than a monolith of optional product verticals. Core should expose the stable
  interfaces that make plugins easy to build: events, provider slots, typed
  payloads, policy checks, SDK helpers, defaults, and protected built-in plugins.
  For shared-state areas such as payments, core may define provider-neutral
  intents, transactions, refunds, webhook idempotency, entitlements, and
  provider interfaces while plugins implement provider/vendor behavior.
- Runtime language pack management has an accepted design: add a system-menu
  admin "Language settings" page, `locale.manage`, ZIP language pack uploads
  with `sforum.locale.json`, package storage under `LOCALE_PACK_ROOT`/
  `storage/locale-packs` outside Git, dedicated `locale_pack*` tables, and
  frontend-only runtime message loading for the first release.
- Forum backend foundation is implemented: `categories`, user-facing
  `topics`, tree-shaped `comments`, shared content `posts`, and
  `post_revisions` now exist in the API schema. `posts` stores raw + sanitized
  HTML + plain text; API `excerpt` is derived at read time from plain text
  via `forum.reading.excerpt_rune_limit`. `post_revisions` keep source-only
  snapshots (raw + editor/render metadata). The Go forum module renders
  Markdown with `goldmark`, sanitizes HTML with `bluemonday`, exposes public
  category/topic/comment APIs, and treats JSON content as schema-reserved but
  not yet publishable.
- 千万级数据读路径加固已实现：Meilisearch 全文搜索完整接入（新增
  `app/Support/Search` 包、`GET /api/v1/search` 端点、forum.Service 事务后调度
  `search.index_topic`/`search.delete_topic` job、首页搜索框走专用端点），
  消除 `ListTopics` 的 ILIKE 全表扫描；Redis 缓存层（`app/Support/Cache` +
  `forum.CachedStore` 装饰器，缓存分类/标签/主题详情/主题列表，generation 失效）；
  OFFSET 深翻页 clamp（`maxTopicPage=200`）。详见
  `decisions/2026-07-08-search-cache-deep-pagination.md`。
- The `sforum seed:forum` CLI command (`apps/api/cmd/sforum`) now generates
  fake forum data (users + topics + comments) for local development and
  testing by reusing the identity/forum Service layer, so seeded data shares
  the same password hashing, Markdown rendering, slug, comment-tree path_key,
  and counters as real user data. It is append-only (random username/email
  suffixes), triggers no events, and reads `DATABASE_URL` from the
  environment or `--database-url` (config.Load does not read `.env`).
- A full security audit was completed on 2026-07-09 across auth/session/perm,
  data access, file upload/storage, frontend XSS, and config/deploy. Findings
  are tracked as a checkbox backlog in
  `decisions/2026-07-09-security-audit.md` (2 Critical, 6 High, 10 Medium,
  6 Low), prioritized C1/C2/H1 first. Confirmed-safe points are recorded to
  avoid re-auditing.
- A follow-up security review (`docs/security-review-2026-07-09.md`) flagged
  6 issues; all 5 verified ones are now fixed: P1 comment visibility bypass,
  P2 password-reset human verification, P2 production config mismatch,
  P2 attachment active-content risk (first batch), and P1 CSRF protection
  (separate milestone using Fiber v3 csrf middleware + double-submit cookie,
  `CSRF_TRUSTED_ORIGINS` config). 1 (P3 avatar attachment ID) was verified as
  a non-issue (store layer already validates ownership/status/type). Decisions
  in `decisions/2026-07-09-security-fixes.md`. The full-stack audit backlog
  (`decisions/2026-07-09-security-audit.md`) saw 20 of 23 remaining items
  fixed on 2026-07-09 (C1/C2, H1/H2/H3/H4/H6, M4-M10 except M2/M3, L1/L3-L6),
  with L2/M2/M3 deferred as documented in
  `decisions/2026-07-09-security-fixes.md`.
- Backend+frontend performance hardening (2026-07-08) covers the network and
  connection layers beyond the earlier search/cache read-path work:
  `ListComments` now uses SQL pagination (flat `LIMIT/OFFSET`; tree uses
  root-comment pagination + descendant batch fetch) instead of loading all
  comments into memory; Fiber sets `ReadTimeout`/`WriteTimeout`/`IdleTimeout`/
  `BodyLimit` and registers `compress` + `limiter` (write-only rate limiting
  via Redis storage); Redis humanverify+cache clients merged into one
  `sharedRedisClient` with explicit pool/timeout config; PG pool exposes
  MinConns/Idle/Lifetime/ConnectTimeout; Meilisearch client gets
  `http.Client.Timeout`; frontend adds `routeRules` (swr for public pages,
  full SSR for auth/admin/protected pages to avoid empty-shell white screens), `compressPublicAssets`, static-asset
  `Cache-Control`, `SFEditor`/`SFIconPicker` lazy loading, and `@nuxt/image`
  for `SFAvatar`. See `decisions/2026-07-08-performance-hardening.md`.

## Navigation

- `decisions/` - decision records for architecture, product, and process choices.
- `modules/` - notes for each feature area or system module.
- `sessions/` - short handoffs from previous work sessions.
- `glossary.md` - shared terms and domain language.
- `research.md` - library and ecosystem research notes.
- `architecture-maturity-audit.md` - living modularization scorecard and
  performance checklist (claim vs code). Last reviewed 2026-07-12: modular
  host ~7/10, performance engineering ~6/10, performance proof low.
- `plans/2026-07-12-development-directions.md` - near-term development strategy:
  five tracks, effort mix (~70/20/10), iterations A/B/C, and explicit depriorities.
- `decisions/2026-07-12-host-platform-capabilities.md` - host OS contracts:
  schedule registry, health layers, Host API, capabilities, outbox/webhooks,
  non-goals; implementation is phased.
- `plans/2026-07-12-framework-hardening-waves.md` - framework hardening
  checklist waves F1–F4 (schedule/health → third-party safety → integration →
  ecosystem); F1–F4.3 done; F4.4–F4.5 continue under surface-density plan.
- `plans/2026-07-12-extension-surface-density.md` - post-hardening extensibility:
  E1 filters, E2 contributions, E3 meta, E4 flags, E5 workflow plugin;
  **E6 storage / E7 search / E8 other provider slots** (plugin configure north
  star); E1.1–E1.4 done — next E2 or E6 if services are priority.
- `plans/2026-07-12-iteration-a-engagement-loop.md` - Iteration A implementation
  checklist: view counts, likes/reactions, bookmarks; topic lifecycle already
  mostly shipped.
- `plans/2026-07-12-admin-settings-richness.md` - full operator settings catalog
  and IA for a mainstream-rich control panel (registration/trust/forum/nav/
  engagement/moderation waves); recommended defaults and anti-patterns.
- `decisions/2026-07-06-tiptap-editor-content-storage.md` - Tiptap editor,
  triple content storage, and server-side XSS safety boundary.
- `../docs/architecture.md` - proposed technical architecture and directory
  layout.
- `modules/identity.md` - identity, registration, sessions, roles, permissions,
  human verification, and policy notes.
- `modules/backend.md` - backend stack, module boundaries, jobs, and the
  Laravel-style Go/Fiber API directory structure.
- `modules/options.md` - runtime site options, `web_options` boundaries, API
  routes, and admin settings notes.
- `modules/attachments.md` - attachment metadata, storage providers, runtime
  settings, permissions, upload flow, cleanup, API, and admin UI notes.
- `modules/extensions.md` - extension package, plugin/theme manifest,
  lifecycle, permissions, storage, and runtime-boundary notes.
- `modules/forum.md` - categories, topics, tree comments, shared content
  posts, API routes, topic lifecycle, and frontend display decisions.
- `modules/profile.md` - member public profiles and current-user profile
  settings.
- `modules/mail.md` - mail provider contract, runtime options, and password
  reset mail flow.
- `modules/moderation.md` - publication review policy, user reports, dual
  moderation workbenches, audit decisions, and author review status.
- `legacy-sforum-feature-gap.md` - inventory of SForum-old features that are
  not yet implemented in the rewrite, grouped by migration impact and suggested
  build order.
- `../contracts/README.md` - modular OpenAPI contract editing guide.
- `decisions/2026-07-04-laravel-style-http-routing.md` - accepted backend
  composition, route registration, and Laravel-style API directory decision.
- `decisions/2026-07-04-altcha-human-verification.md` - accepted ALTCHA human
  verification decision.
- `decisions/2026-07-04-api-response-envelope-localized-message.md` - accepted
  backend API envelope and localized message decision.
- `decisions/2026-07-05-browser-session-jwt-strategy.md` - accepted browser
  session lifetime, renewal, audit, and future JWT/API-token strategy.
- `decisions/2026-07-05-registration-altcha-default-disabled.md` - accepted
  registration ALTCHA default-off runtime behavior.
- `decisions/2026-07-04-configurable-admin-control-panel.md` - accepted
  configurable admin route prefix and Nuxt UI dashboard shell decision.
- `decisions/2026-07-05-local-dev-dependencies-and-processes.md` - accepted
  local development split where Compose starts dependencies and frontend/API
  run as host processes.
- `decisions/2026-07-05-startup-database-migrations.md` - accepted embedded
  Goose startup migrations for API and worker processes.
- `decisions/2026-07-05-admin-multitabs-and-layout-rules.md` - accepted admin
  multitabs, topbar breadcrumbs, larger tabs, and nested menu rules decision.
- `decisions/2026-07-05-admin-low-code-module-registry.md` - accepted
  registry-driven admin page, sidebar, tab, and permission metadata decision.
- `decisions/2026-07-05-user-permission-overrides.md` - accepted user-level
  permission override decision for precise admin-managed access.
- `decisions/2026-07-05-appearance-theme-presets.md` - accepted runtime theme
  preset and controlled custom color decision.
- `decisions/2026-07-05-seo-full-chain-v1.md` - accepted runtime SEO settings,
  `seo.manage`, robots/sitemap integration, and local noindex strategy.
- `decisions/2026-07-05-attachment-storage-providers.md` - accepted attachment
  provider adapter strategy and first provider set.
- `decisions/2026-07-05-attachment-local-root-runtime-option.md` - accepted
  local attachment root runtime option decision.
- `decisions/2026-07-05-extension-plugin-theme-foundation.md` - accepted
  plugin/theme extension foundation, storage, permission, and runtime-boundary
  decision.
- `decisions/2026-07-12-extension-manifest-split.md` - accepted and implemented
  multi-file extension manifest authoring: thin `sforum.extension.json` entry,
  optional `includes`, directory-per-locale identity `langs`, settings shards,
  `LoadPackage`, `make:plugin --complex`, `extension validate` (plan under
  `docs/superpowers/plans/2026-07-12-extension-manifest-split.md`).
- `decisions/2026-07-10-trusted-admin-plugin-runtime.md` - accepted build-time
  trusted admin component runtime, digest grants, manifest contribution
  metadata, and unified Web Release ownership; implementation is pending.
- `decisions/2026-07-06-plugin-enable-theme-activate-default-theme.md` -
  accepted plugin enable vs theme activate semantics and default-theme public
  UI ownership.
- `decisions/2026-07-07-incremental-theme-fallback.md` - accepted uploaded
  theme incremental overlay behavior with the default theme as final fallback.
- `decisions/2026-07-06-plugin-event-extension-points.md` - accepted explicit
  plugin event, listener delivery, and filter extension-point architecture.
- `decisions/2026-07-05-runtime-language-pack-management.md` - accepted runtime
  language pack storage, permission, admin page, and frontend runtime message
  loading decision.
- `decisions/2026-07-05-openapi-contract-modularization.md` - accepted
  modular OpenAPI source layout and reference-validation workflow.
- `decisions/2026-07-06-forum-topics-comments-posts.md` - accepted forum
  backend model where `topics` are user-facing posts, `comments` are tree
  replies, and `posts` is the shared content table.
- `decisions/2026-07-07-mail-provider-contract.md` - accepted mail provider
  contract (`mail.Provider` interface with noop/dev_log/smtp built-ins) and
  runtime option resolution for password reset and notifications.
- `decisions/2026-07-09-avatar-system.md` - accepted avatar strategy system:
  uploaded avatars over initials/Gravatar/static fallbacks, avatar-specific
  attachment processing, and `imaging` for JPEG/PNG compression.
- `sessions/2026-07-04-altcha-human-verification-implementation.md` - ALTCHA
  implementation handoff.
- `sessions/2026-07-04-registration-status-notice.md` - first-user
  super-admin notice implementation handoff.
- `sessions/2026-07-04-admin-foundation.md` - admin foundation implementation
  handoff.
- `sessions/2026-07-05-registration-verification-session-failure.md` -
  registration verification ordering and session-failure fix handoff.
- `sessions/2026-07-05-registration-success-navigation.md` - registration
  password hint, success-state hydration, and middleware-safe API locale
  handoff.
- `sessions/2026-07-05-login-migration-mismatch-fix.md` - missing
  `user_permission_overrides` migration root cause and login error-mapping
  hardening handoff.
- `sessions/2026-07-05-local-dev-dependencies.md` - local dependency startup
  and host-process development handoff.
- `sessions/2026-07-05-startup-database-migrations.md` - embedded startup
  migration implementation handoff.
- `sessions/2026-07-05-registration-altcha-default-disabled.md` -
  registration ALTCHA default-off implementation handoff.
- `sessions/2026-07-05-nuxt-dev-open-delay.md` - Nuxt dev 503 loading page,
  build/typecheck directory isolation, and local API port mismatch handoff.
- `sessions/2026-07-05-nuxt-preview-script.md` - Nuxt production preview script
  fix for the unsupported `nuxi preview --host` flag.
- `sessions/2026-07-05-admin-multitab-layout-upgrades.md` - admin multitabs,
  global topbar, theme adaptive sidebar, and nested menu layout upgrades handoff.
- `sessions/2026-07-05-admin-low-code-module-registry.md` - registry-driven
  admin sidebar/tab architecture handoff.
- `sessions/2026-07-05-seo-full-chain-v1.md` - SEO Full-Chain v1 implementation
  handoff.
- `sessions/2026-07-05-attachment-system-foundation.md` - attachment system
  implementation handoff.
- `sessions/2026-07-05-attachment-local-root-runtime-option.md` - attachment
  local root runtime option migration handoff.
- `sessions/2026-07-05-test-baseline-fix.md` - test baseline fix, admin/identity
  validation sync, and attachment local-root runtime option handoff.
- `sessions/2026-07-05-extension-system-foundation.md` - extension backend,
  admin UI, manifest, lifecycle, and runtime-boundary implementation handoff.
- `sessions/2026-07-12-extension-manifest-split-plan.md` - multi-file extension
  manifest split decision/plan handoff (includes + per-locale langs).
- `sessions/2026-07-12-extension-manifest-split-impl.md` - LoadPackage, SMTP
  migration, complex scaffold, and extension validate implementation handoff.
- `sessions/2026-07-05-extension-admin-submenus.md` - extension admin sidebar
  folder split into Overview, Plugins, Themes, Settings, and Event Log pages.
- `sessions/2026-07-05-admin-language-settings-design.md` - runtime language
  pack and admin language settings design handoff.
- `sessions/2026-07-05-admin-alert-autoclose-guideline.md` - admin alert/toast
  guideline requiring 10-second auto-dismiss for non-error feedback.
- `sessions/2026-07-05-openapi-contract-modularization.md` - OpenAPI split,
  validation script, and documentation handoff.
- `sessions/2026-07-05-public-navbar-hide-admin-entry.md` - public navbar admin
  entry removal handoff.
- `sessions/2026-07-05-admin-permission-management.md` - user-level permission
  overrides, admin users/roles/permissions UI, and API contract handoff.
- `sessions/2026-07-06-api-air-startup-speed.md` - API/worker Air config
  watch-scope and deprecated `build.bin` cleanup handoff.
- `sessions/2026-07-06-go-run-startup-optimization.md` - `go run` startup
  diagnosis, API/CLI helper scripts, and extension manifest dependency split.
- `sessions/2026-07-05-permission-matrix-comparison-view.md` - permission
  matrix scalability update with limited group comparison and differences-only
  audit mode.
- `sessions/2026-07-06-role-input-validation.md` - roles page empty-create bug
  fix, service/API validation, cleanup migration, OpenAPI update, and QA notes.
- `sessions/2026-07-06-theme-boundary-activation.md` - plugin enable/theme
  activate semantics, built-in default theme layer, and public UI ownership
  handoff.
- `sessions/2026-07-06-extension-stale-builtin-cleanup.md` - stale built-in
  extension row pruning and package-existence preflight handoff.
- `sessions/2026-07-07-theme-activation-runtime.md` - uploaded theme
  activation runtime, River job, web supervisor, Docker volume, admin states,
  and verification notes.
- `sessions/2026-07-07-incremental-theme-fallback.md` - incremental uploaded
  theme overlay contract, fallback tests, and knowledge-base update.
- `sessions/2026-07-07-theme-runtime-convergence.md` - `current.json` 新契约
  (`mode`/`server`/`layerPath`)、默认主题同步写 current、dev 主题感知 supervisor、
  runtime.mjs 健壮化、i18n 文案修正 handoff。
- `sessions/2026-07-07-core-forum-v1.md` - Core Forum V1 implementation
  handoff: topic lifecycle, detail/composer UI, profiles, mail + password
  reset, moderation reports, and full verification results.
- `sessions/2026-07-04-permission-aware-development-guidelines.md` -
  permission-aware feature development guideline handoff.
- `sessions/2026-07-05-auth-session-restart-resilience.md` - frontend auth
  refresh behavior for API restart/session recovery resilience.
- `sessions/2026-07-05-global-footer-implementation.md` - global footer implementation handoff.
- `sessions/2026-07-05-personalization-settings.md` - theme preset and footer
  personalization implementation handoff.
- `decisions/2026-07-08-search-cache-deep-pagination.md` - accepted Meilisearch
  full-text search integration, Redis CachedStore read cache, and deep-paging
  clamp decision.
- `decisions/2026-07-08-performance-hardening.md` - accepted backend+frontend
  performance hardening: ListComments SQL pagination, Fiber timeout/compress/
  limiter, Redis client merge, PG pool tuning, Meili timeout, routeRules, lazy
  loading, @nuxt/image.
- `sessions/2026-07-08-search-cache-deep-pagination.md` - search/cache/deep-paging
  hardening implementation handoff.
- `sessions/2026-07-08-performance-hardening.md` - performance hardening
  implementation handoff (backend network/connection layers + frontend
  caching/rendering/image optimization).
- `sessions/2026-07-08-cold-start-race-white-screen-502.md` - supervisor
  cold-start race fix: proxy listened before nuxt dev/Nitro ready, causing
  SPA login page white screen and form 502 on first access after startup.
- `sessions/2026-07-07-admin-personalization-system-config.md` - admin
  personalization sidebar move into the System configuration folder.
- `sessions/2026-07-05-custom-theme-color.md` - custom theme color picker,
  backend validation, and Nuxt UI primary-token bridge handoff.
- `sessions/2026-07-05-icon-picker.md` - reusable Tabler/Nuxt Icon picker
  implementation handoff.
- `sessions/2026-07-05-icon-picker-full-catalog.md` - full catalog,
  paginated, lazy-loading icon picker handoff.
- `sessions/2026-07-05-altcha-layout-fix.md` - ALTCHA settings section layout fix handoff.
- `sessions/2026-07-05-altcha-scenario-settings.md` - CAPTCHA scenario
  switches, ALTCHA secret generation, TTL/cost guidance, and UI validation
  handoff.
- `sessions/2026-07-06-forum-backend-foundation.md` - forum schema,
  renderer/sanitizer, routes, OpenAPI, and tree comment model handoff.
- `sessions/2026-07-08-seed-forum-command.md` - `sforum seed:forum` 假数据生成
  命令（用户/主题/评论）实现、用法、性能实测与端到端验证 handoff。
- `decisions/2026-07-09-security-audit.md` - 全栈安全审计结果与修复待办清单
  （2 Critical / 6 High / 10 Medium / 6 Low，含已确认安全项与修复优先级）。
- `decisions/2026-07-09-security-fixes.md` - 安全修复两批决策：第一批安全审阅
  5 项（评论可见性、密码重置人机验证、生产配置、附件主动内容、CSRF 防护）+ P3 订正；
  第二批全栈审计修复 20 项（C1/C2、H1-H4/H6、M4-M10 除 M2/M3、L1/L3-L6），
  L2/M2/M3 降级待办。
- `sessions/2026-07-08-auth-success-toast-guideline.md` - login/register
  success Toasts, auth page test harness path fix, theme-aware success Toast
  styling, and broader frontend Toast feedback guideline.
- `sessions/2026-07-08-admin-ssr-no-white-screen.md` - 移除全站最后的
  `ssr: false`（后台 + 组件预览页），全部页面 SSR 彻底杜绝空壳白屏。
- `sessions/2026-07-09-avatar-system.md` - avatar strategy implementation,
  admin/profile UI, OpenAPI/docs, and `/t/**` query-edit cache hardening.
- `sessions/2026-07-10-admin-taxonomy-icon-color.md` - admin category/tag icon
  and icon color configuration plus admin-list preview handoff.
- `decisions/2026-07-10-account-security-sessions.md` - accepted account security /
  login device management: `user_sessions` directory table, stable opaque `sid`,
  next-request revocation, revoke-one/revoke-others, `identity.sessions.max_devices`.
- `sessions/2026-07-10-account-security-sessions.md` - account security / login
  device management implementation handoff.
- `sessions/2026-07-11-seo-workbench-v2-p0.md` - SEO Workbench v2 P0: independent
  homepage SEO identity, content policies, image upload/references, typed public
  metadata, structured data, and forum Sitemap partitions.
- `../docs/superpowers/specs/2026-07-05-global-footer-design.md` - global footer design spec.
- `../docs/superpowers/plans/2026-07-05-global-footer.md` - global footer implementation plan.
- `../docs/superpowers/specs/2026-07-04-security-verification-design.md` -
  security verification design.
- `../docs/development-and-deployment.md` - proposed local development,
  hot-reload, Docker Compose, and production deployment workflow.
- `../apps/web` - Nuxt web scaffold with default `zh-CN` localization.
- `../apps/api` - Go Fiber API and worker scaffold.

## How To Use This In A New Session

1. Read `AGENTS.md`.
2. Read this file.
3. Open the latest handoff in `sessions/`.
4. Open related module notes.
5. Continue work and update these notes before stopping.

## Open Questions

- What is the first usable MVP scope?
- Which forum features are required versus later enhancements?
- What deployment target should the architecture optimize for?
- Should Meilisearch ship in the first executable milestone or immediately
  after core forum reads/writes?
- What production backup destination and retention policy should be used?
- Should English translations be mandatory for MVP launch or allowed to lag
  during internal development?
- Should email verification be required before posting in MVP?
- What ALTCHA challenge expiration and work cost should production use when
  human verification is explicitly enabled?

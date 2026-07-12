# Page Catalog Inventory (P0)

Status: **draft inventory for Runtime Page Registry**  
Date: 2026-07-13  
ADR: `knowledge/decisions/2026-07-13-runtime-page-registry-themes.md`  
Plan: `knowledge/plans/2026-07-13-runtime-page-registry-themes.md`

This document freezes **what exists today** so P1 can implement a stable Page
Catalog without rediscovering routes. It is **not** yet the runtime source of
truth; P1 will add a Go catalog (authoritative) plus OpenAPI/admin list.

---

## Source of truth decisions (P0 → P1)

| Concern | Decision |
| --- | --- |
| Page ID catalog (runtime) | **Go** package in P1 (e.g. `app/Support/Pages` or similar); unit-tested uniqueness |
| Author-facing table | This file + plan draft; regenerate/sync notes after P1 lands |
| Path patterns | Match **live Nuxt file routes** under the active theme layer (default: `sforum-default`) |
| i18n | `strategy: prefix_except_default`, default `zh-CN` (no prefix), secondary `en` (`/en/...`). Catalog stores **locale-stripped** path patterns; matchers must accept both. |
| Provider | Always `core` until P3 contributions |

### Dual-stack feature flags (names finalized here; code in P1+)

| Option key | Type | Default (until P5 exit) | Purpose |
| --- | --- | --- | --- |
| `pages.registry_enabled` | bool string (`true`/`false`) | `false` | Catalog + admin inspect (+ thin outlet path when wired) |
| `themes.runtime_l0_enabled` | bool string | `false` | L0 skin CSS/assets without Nuxt rebuild |
| `themes.runtime_l1_enabled` | bool string | `false` | Template replace/add via registry |
| `themes.layer_activation_enabled` | bool string | **`true`** | Legacy Nuxt Layer activate + Web Release theme path |

Notes:

- Keys are **not** registered in Options yet (P1 for `pages.registry_enabled`;
  P2/P3 for theme runtime flags). Defaults above are the intended product
  defaults when keys land.
- Production dual-stack: legacy Layer **on**; runtime features **opt-in**.
- These are **not** the same as product surface `features.*` flags (search,
  registration, …). Prefer the `pages.*` / `themes.*` namespaces so operators
  do not confuse “kill search” with “enable page registry”.

---

## Core public / member page catalog (live paths)

Paths are **file routes** relative to the theme layer or host app. Exact
implementation files are listed for inventory; P1 maps `coreComponent` to these
(or host-owned equivalents after outlet migration).

### From default theme layer

Base: `extensions/builtin/themes/sforum-default/layer/app/pages/`

| Page ID | Default path pattern | Access class | Notes | File |
| --- | --- | --- | --- | --- |
| `forum.home` | `/` | public | Latest feed; query filters; SEO/SWR on `/` and `/en` | `index.vue` |
| `forum.category.index` | `/categories` | public | Category directory | `categories/index.vue` |
| `forum.category.show` | `/c/:categorySlug` | public | Category topic list | `c/[categorySlug].vue` |
| `forum.tag.index` | `/tags` | public* | `*` gated by `forum.tags.public_pages`; 404 when off | `tags/index.vue` |
| `forum.tag.show` | `/tags/:tagSlug` | public* | Same public-pages option | `tags/[tagSlug].vue` |
| `forum.topic.show` | `/t/:path(.*)` | public | Catch-all; URL shape depends on `seo.topic_url_mode` (`id_slug` default → `/t/:id/:slug`, also `id`, `slug`) | `t/[...path].vue` |
| `forum.topic.create` | `/topics/new` | login | Composer | `topics/new.vue` |
| `forum.profile.show` | `/u/:username` | public* | `*` also product flag `features.public_profiles` | `u/[username].vue` |
| `forum.my.home` | `/my` | login | Self center hub | `my/index.vue` |
| `forum.my.content_review` | `/my/content-review` | login | Author content review queue | `my/content-review.vue` |
| `forum.settings.profile` | `/settings/profile` | login | Account profile settings (not in plan draft; **add**) | `settings/profile.vue` |
| `forum.settings.security` | `/settings/security` | login | Sessions / security | `settings/security.vue` |
| `auth.login` | `/login` | guest | `layout: auth`, `middleware: guest`; replace must keep host form island | `login.vue` |
| `auth.register` | `/register` | guest | Same; product flag `features.registration` | `register.vue` |
| `auth.forgot_password` | `/forgot-password` | public | | `forgot-password.vue` |
| `auth.reset_password` | `/reset-password` | public | Token query params | `reset-password.vue` |
| `site.terms` | `/terms` | public | Legal Markdown from options | `terms.vue` |
| `site.privacy` | `/privacy` | public | | `privacy.vue` |
| `site.guidelines` | `/guidelines` | public | | `guidelines.vue` |

### From host Nuxt app (not theme layer)

Base: `apps/web/app/pages/`

| Page ID | Default path pattern | Access class | Notes | File |
| --- | --- | --- | --- | --- |
| `forum.notifications` | `/notifications` | login | Host-owned | `notifications.vue` |
| `moderation.review` | `/moderation` | login + moderation middleware | Workbench entry | `moderation/index.vue` |
| `dev.components` | `/components` | public / robots noindex | Dev component gallery; **not** a theme replace target | `components.vue` |
| `system.not_found` | n/a | public | Framework error page / thrown 404s; no stable file route | host error utilities |

### Admin UI (out of Page Registry v1 public surface)

- Admin lives under configurable prefix, default **`/control-panel`**
  (`DEFAULT_ADMIN_ROUTE_PREFIX`), not `/admin`.
- Legacy prefix constant `LEGACY_ADMIN_ROUTE_PREFIX = '/admin'` still reserved.
- Extension admin pages: `{adminPrefix}/extensions/:extensionId/pages/...`
- **Do not** register public page contributions under admin prefixes (P3).

### Dev / reference theme overrides (not catalog expansion)

`extensions/dev/themes/sforum-signal-garden/layer/app/pages/` currently
overrides only: `index`, `login`, `register`. Same page ids as core.

---

## Reserved path prefixes

Themes/plugins **must not** register `add` paths that collide with these
prefixes (check at manifest validation in P3). Matching is path-prefix after
optional locale segment strip.

| Prefix / pattern | Why reserved |
| --- | --- |
| `/control-panel` (+ configured admin prefix) | Admin UI |
| `/admin` | Legacy admin prefix constant |
| `/api` | Nitro + Go API proxy (`server/routes/api/v1/[...path].ts`, sitemap, icons) |
| `/_nuxt`, `/__nuxt` | Framework assets |
| `/__sforum` | Host internal server routes |
| `/health` | Process health |
| `/auth/*` **API** paths | Session authority is API (`/api/v1/auth/...`); public **view** pages `/login` etc. remain replaceable chrome only |
| Attachment content URLs as served | Authenticated/public attachment download must stay host-controlled (`/api/v1/attachments/...`) |
| OAuth / external IdP callbacks | If added later under fixed paths, reserve at implementation time |

i18n note: reserved checks apply to both `/api/...` and `/en/api/...` if such
paths ever appear; today API is typically unprefixed proxy on the web origin.

---

## Layer activation & Web Release touchpoints (inventory)

Do **not** delete these in P0–P4. P5 retires theme Layer coupling only after
exit criteria.

### API / worker

| Area | Path / symbol | Role |
| --- | --- | --- |
| Activate HTTP | `POST /api/v1/admin/extensions/:id/activate` → `ActivateThemeOperation` | Queues or applies theme |
| Service | `apps/api/app/Models/Extensions/service.go` `ActivateTheme` | Builtin default sync restore + `WriteCurrent`; uploaded themes queue build |
| Job | `extension.theme_activate` — `app/Jobs/Extensions/theme_activate.go` | Build + activate; may delegate `QueueLegacyThemeRelease` |
| DB | `extension_theme_releases` | Release rows / status / layer_path |
| Theme builder | `app/Support/ThemeRuntime` `Build` / `WriteCurrent` | Writes `current.json` contract |
| Web release coordinator | `app/Support/WebReleaseCoordinator` | Writes current pointer; effects call `ActivateTheme` |
| Config | `THEME_RELEASE_ROOT` / `WEB_RELEASE_ROOT` (shared default `storage/theme-releases`) | Release root on disk |
| Permission | `extension.theme.manage` | Activate themes |
| Audit | `extension.theme_activate` | Audit action name |
| Public theme settings | `GET /api/v1/site/active-theme/settings` | Non-secret active theme settings for layer |

### Web supervisors & scripts

| Script | Role |
| --- | --- |
| `apps/web/scripts/dev-theme-runtime.mjs` | Dev: watch `current.json`, restart `nuxt dev` with layer |
| `apps/web/scripts/dev-theme-lifecycle.mjs` | Serial restart coalescing |
| `apps/web/scripts/runtime.mjs` | Prod: blue/green Nitro switch from `current.json` |
| `apps/web/scripts/theme-proxy.mjs` | Proxy + health for prod switch |
| `apps/web/scripts/web-release-contract.mjs` | `current.json` / `active.json` contract |
| `apps/web/scripts/dev-admin-compose.mjs` | Dev compose builtin admin/theme source |
| `apps/web/scripts/dev-plain.mjs` / `dev-plain-release-ack.mjs` | Dev without full theme supervisor; still ack releases |

### On-disk signals

| File | Role |
| --- | --- |
| `{WEB_RELEASE_ROOT}/current.json` | Desired release (`mode` default/uploaded, `server`, `layerPath`, …) |
| `active.json` / failure ack files | Supervisor acknowledgement (web-release contract) |
| Release artifact dirs under release root | Built Nitro / layer paths |

### Tests / validation (sample)

- `tests/validate-theme-runtime.js`
- `apps/web/tests/webReleaseContract.test.ts`, `devThemeLifecycle.test.ts`
- `apps/api/app/Jobs/Extensions/theme_activate_test.go`
- `apps/api/app/Models/Extensions/service_test.go` (ActivateTheme cases)
- `apps/api/app/Support/ThemeRuntime/builder_test.go`

### Freeze rule (extensions module)

**No new Layer-only theme features** except critical bugs. New investment goes
to Page Registry + L0/L1 runtime (this program). Layer path remains
compatibility dual-stack until P5.

---

## What is still separate (do not merge into Page Registry)

- Contribution points (nav, badges, topic actions, …)
- Events / filters
- Provider slots (mail, storage, search, …)
- Extension route proxy: `/api/v1/extensions/:extensionId/*`
- Trusted admin Web Release (admin Vue only)
- Appearance presets (`appearance.theme` / 配色) — not installable themes

---

## P1 mapping notes (for next session)

1. Implement Go catalog entries for every **Page ID** row above that is a real
   replace target (skip `dev.components` or mark `replaceable: false`).
2. First outlet: **`forum.home` only** — thin wrapper; core provider still
   renders current home Vue.
3. Admin read-only list when `pages.registry_enabled` (or always list catalog
   with provider=`core` for inspect — product choice in P1).
4. OpenAPI only for new registry endpoints.
5. Do **not** change theme packaging, Liquid, or delete Layer activation.

### Gaps vs plan draft table

Plan draft missed live routes that should be cataloged:

- `forum.settings.profile`, `forum.settings.security`
- `forum.notifications`, `moderation.review` (host pages)
- Admin is `/control-panel` by default, not only `/admin`

---

## Verification (P0)

- Docs only; no runtime behavior change.
- Cross-check: every `.vue` under default theme `layer/app/pages` appears in
  the catalog table (done 2026-07-13).
)

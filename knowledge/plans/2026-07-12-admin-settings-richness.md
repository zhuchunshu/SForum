# Admin Settings Richness Blueprint (2026-07-12)

Status: accepted working product blueprint for operator-facing configuration  
Audience: humans and AI sessions expanding admin settings without ad-hoc sprawl

Related:

- Runtime options module: `knowledge/modules/options.md`
- Development directions: `knowledge/plans/2026-07-12-development-directions.md`
- Legacy gap (partially stale): `knowledge/legacy-sforum-feature-gap.md`
- Agent rules: beginner-friendly defaults, permission-aware design, plugin-first core

---

## Goal

Make the SForum control panel feel as configurable as mainstream forum software
(Discourse, Flarum, phpBB, XenForo, Discuz), while remaining:

1. **Beginner-friendly** — recommended defaults + one-click restore on every group
2. **Permission-aware** — fine-grained manage keys; API is authoritative
3. **Plugin-first** — vendor/provider behavior stays in plugins; core owns
   stable product policy switches and contracts
4. **Maintainable** — `web_options` for scalar/runtime policy; dedicated tables
   for structured catalogs (nav items, friend links, banned words, announcements)

**Principle:** rich does not mean one giant form. Rich means complete **policy
surface**, grouped by operator mental model, with progressive disclosure
(Recommended / Advanced).

---

## Current Baseline (what exists)

| Area | Status |
| --- | --- |
| Site identity, locale, datetime | Implemented |
| Registration open/close | Wave 1: mode + username + email-verify flags (invite/approval UX deferred) |
| Password policy + max devices | Implemented (+ login lockout) |
| Human verification / ALTCHA | Implemented (rich) |
| Forum length/cooldown/daily limits/tags/pagination/excerpt | Implemented (+ guest read, list sort, behavior pack) |
| Categories / tags CRUD + icons | Implemented |
| Moderation mode `off/rules/all` | Implemented (site-level only) |
| Mail + notification channel policy | Implemented |
| Avatar strategy | Implemented |
| SEO workbench | Implemented (already rich) |
| Attachments multi-provider | Implemented (already rich) |
| Personalization (preset + footer links) | Partial |
| Extension lifecycle | Partial product shell |
| Maintenance mode | Wave 1: option + write middleware |
| Newcomer trust ladder | Wave 1: options + forum enforcement |
| Engagement (like/bookmark), scoped mods, nav editor | Missing |

---

## Target Information Architecture

Sidebar shape after full richness (folders only; pages stay registry-driven):

```text
指挥台
用户与权限
  用户 / 用户组 / 权限矩阵
论坛
  审核管理
  版块分类          ← + 版块策略覆盖
  标签
  论坛设置          ← 扩 Tab（见下）
  敏感词与自动规则  ← 新
  公告与横幅        ← 新（或并入个性化）
系统配置
  站点设置          ← 扩 Tab：注册/新人/维护/品牌
  邮件与通知
  头像
  个性化            ← + 导航 / 页脚 / 友情链接
  SEO
  搜索运营          ← 从「重建」扩到运营旋钮
  互动与社区        ← 新：赞/藏/关注/私信开关
  语言包            ← 已设计未落地
内容与文件
  附件
扩展
  （现有）
运维
  数据库 / 任务队列 / （未来）审计日志 / 备份状态
```

### Forum settings tabs (target)

| Tab | Contents |
| --- | --- |
| general | default category, homepage/list sort, guest read, category directory |
| topics | length, edit window, cooldown, daily cap, **close-replies default, author delete, auto-lock, title duplicate policy** |
| comments | length, nesting, edit/cooldown/daily, **edit mark, soft-delete visibility, mention limits** |
| tags | creation mode, public pages, min/max, **suggest-on-compose** |
| reading | excerpt, **list density, related topics, view-count rules** |
| composer | editor modes, attachment in post, draft retention, preview |
| advanced | rate-limit overrides that are product-facing (not infra env) |

### Site settings tabs (target)

| Tab | Contents |
| --- | --- |
| basic | name, url, tagline, admin email, locale, datetime |
| brand | logo, favicon, apple-touch, default share image bridge to SEO |
| registration | open/invite/approval modes, email verify gate, username rules, reserved names |
| newcomers | posting gates, outbound link gate, attachment gate, trust ladder days |
| accountSecurity | password, sessions/devices, login lockout, reauth for sensitive actions |
| verification | CAPTCHA provider + scenarios (existing) |
| maintenance | mode, message, allowlisted roles/IPs |
| legal | terms/privacy/guidelines **content or page sources** (beyond footer URLs) |

---

## Full Catalog By Domain

Legend:

- **State:** `have` | `partial` | `need` | `plugin` | `later`
- **Store:** `opt` = `web_options`; `table` = dedicated rows; `cat` = category column/JSON; `role` = RBAC
- **Perm:** primary manage permission

### A. Site identity & brand

| Key / capability | State | Store | Perm | Recommended default |
| --- | --- | --- | --- | --- |
| `site.name` | have | opt | settings.manage | SForum |
| `site.url` | have | opt | settings.manage | deploy URL |
| `site.tagline` | have | opt | settings.manage | empty |
| `site.admin_email` | have | opt | settings.manage | empty |
| `site.default_locale` / `supported_locales` | have | opt | settings.manage | zh-CN; zh-CN,en-US |
| `site.timezone` / date / time / start_of_week | have | opt | settings.manage | UTC / Y-m-d / H:i / Mon |
| `site.logo_attachment_id` or public URL | need | opt | settings.manage | empty → theme default |
| `site.favicon_attachment_id` | need | opt | settings.manage | empty |
| `site.apple_touch_icon` | need | opt | settings.manage | empty |
| `site.brand_color_sync` with appearance | later | opt | settings.manage | inherit appearance.theme |

### B. Registration & identity policy

| Key / capability | State | Store | Perm | Recommended default |
| --- | --- | --- | --- | --- |
| `identity.registration.enabled` | have | opt | settings.manage | enabled |
| `identity.registration.mode` = open \| invite \| approval \| closed | need | opt | settings.manage | open |
| `identity.registration.require_email_verification` | need | opt | settings.manage | disabled (beginner); prod recommend enabled |
| `identity.registration.block_posting_until_verified` | need | opt | settings.manage | enabled when verify on |
| `identity.registration.invite_only` (codes) | need | table+opt | settings.manage | off |
| `identity.username.min_length` / `max_length` | need | opt | settings.manage | 3 / 20 |
| `identity.username.charset` = unicode_letters_numbers \| ascii | need | opt | settings.manage | unicode_letters_numbers |
| `identity.username.reserved` list | need | opt | settings.manage | admin,system,sforum,… |
| `identity.display_name.change_cooldown_days` | need | opt | settings.manage | 30 |
| `identity.email.change_requires_reverify` | need | opt | settings.manage | enabled |
| Password composition + length | have | opt | settings.manage | min 12, soft rules off |
| `identity.sessions.max_devices` / `keep_days` | have | opt | settings.manage | 5 / 30 |
| `identity.sessions.bind_user_agent` | later | opt | settings.manage | disabled (fragile) |
| `identity.login.max_failures` / `lockout_minutes` | need | opt | settings.manage | 10 / 15 |
| `identity.login.show_generic_error` | have (hard) | code | — | always generic |
| OAuth providers enable matrix | plugin | plugin settings | extension.manage | all off until configured |
| Email verification templates | partial | mail templates | settings.manage | built-in |

### C. Newcomer / trust ladder

| Key / capability | State | Store | Perm | Recommended default |
| --- | --- | --- | --- | --- |
| `trust.new_user_days` | need | opt | settings.manage | 7 |
| `trust.new_user.topic_cooldown_seconds` | need | opt | settings.manage | 300 |
| `trust.new_user.comment_cooldown_seconds` | need | opt | settings.manage | 60 |
| `trust.new_user.daily_topic_limit` | need | opt | settings.manage | 3 |
| `trust.new_user.daily_comment_limit` | need | opt | settings.manage | 30 |
| `trust.new_user.forbid_outbound_links` | need | opt | settings.manage | enabled |
| `trust.new_user.forbid_attachments` | need | opt | settings.manage | disabled |
| `trust.new_user.require_moderation` | need | opt | moderation.manage | disabled (use global mode) |
| `trust.min_posts_before_links` | need | opt | settings.manage | 0 |
| Role-based overrides | later | role | role.manage | none |

### D. Human verification

| Key / capability | State | Store | Perm | Notes |
| --- | --- | --- | --- | --- |
| Provider + scenarios + ALTCHA knobs | have | opt | settings.manage | Keep; wire `post_risk`/`login_risk` when risk engine exists |
| Future providers (Turnstile, hCaptcha) | plugin | provider slot | settings.manage | core keeps scenario switches |

### E. Forum global policy

| Key / capability | State | Store | Perm | Recommended default |
| --- | --- | --- | --- | --- |
| default category slug | have | opt | category.manage | general |
| topics/comments per page | have | opt | settings.manage | 20 / 20 |
| topic title/content min/max runes | have | opt | settings.manage | existing recommended |
| topic edit window / cooldown / daily | have | opt | settings.manage | existing |
| comment min/max / nesting / edit / cooldown / daily | have | opt | settings.manage | existing |
| tags creation mode / public / min/max | have | opt | tag.manage | controlled / on / 0–5 |
| excerpt rune limit | have | opt | settings.manage | ~120 |
| `forum.guest.read` = public \| login_required | need | opt | settings.manage | public |
| `forum.guest.list_topics` | need | opt | settings.manage | public |
| `forum.list.default_sort` = latest \| active \| hot | need | opt | settings.manage | latest |
| `forum.list.hot_window_days` | need | opt | settings.manage | 7 |
| `forum.topics.allow_author_close_replies` | need | opt | settings.manage | enabled |
| `forum.topics.allow_author_delete` | need | opt | settings.manage | enabled (soft) |
| `forum.topics.auto_lock_idle_days` | need | opt | settings.manage | 0 (off) |
| `forum.topics.show_edit_mark` | need | opt | settings.manage | enabled after N minutes |
| `forum.topics.duplicate_title_policy` | need | opt | settings.manage | warn |
| `forum.comments.show_edit_mark` | need | opt | settings.manage | enabled |
| `forum.comments.soft_delete_visibility` | need | opt | settings.manage | author_and_staff |
| `forum.mentions.enabled` | need | opt | settings.manage | enabled |
| `forum.mentions.max_per_post` | need | opt | settings.manage | 10 |
| `forum.composer.allow_markdown_source` | need | opt | settings.manage | enabled |
| `forum.composer.draft_retention_days` | later | opt | settings.manage | 30 |
| `forum.view_count.unique_window_minutes` | need | opt | settings.manage | 30 |
| `forum.related_topics.enabled` / count | later | opt | settings.manage | off / 5 |
| Tag suggest on compose | later | opt | tag.manage | off |

### F. Category-level policy (override global)

Store as columns or `categories.settings` JSON; inherit when null.

| Capability | State | Perm |
| --- | --- | --- |
| visibility public/hidden | have | category.manage |
| default sort | have | category.manage |
| icon / color | have | category.manage |
| guest read override | need | category.manage |
| who can create topic (role keys or permission) | need | category.manage + role |
| who can comment | need | category.manage |
| moderation mode override | need | category.manage + moderation.manage |
| allow attachments | need | category.manage |
| allow tags | need | category.manage |
| min/max tags override | need | category.manage |
| topic review required | need | category.manage |
| assigned moderators | need | category.manage / new scoped role |

### G. Moderation & safety

| Key / capability | State | Store | Perm | Recommended default |
| --- | --- | --- | --- | --- |
| publication mode off/rules/all | have | table | moderation.manage | off or rules |
| report reasons catalog | partial | table | moderation.manage | built-in list |
| banned words / auto-flag rules | need | table | moderation.manage | empty + sample import |
| auto-moderation: new user + link | need | opt | moderation.manage | soft defaults |
| user mute / ban durations presets | need | opt | user.manage | 1d/7d/30d/perm |
| shadow-ban | later | — | — | avoid until product need |
| audit log retention days | need | opt | settings.manage | 90 |
| admin action log viewer | need | table | database.manage or audit.view | — |

### H. Engagement & social

| Key / capability | State | Store | Perm | Recommended default |
| --- | --- | --- | --- | --- |
| `engagement.likes.enabled` | need | opt | settings.manage | enabled (with Iteration A) |
| `engagement.likes.on_topics` / `on_comments` | need | opt | settings.manage | both on |
| `engagement.likes.undo_window_seconds` | need | opt | settings.manage | 0 unlimited |
| `engagement.bookmarks.enabled` | need | opt | settings.manage | enabled |
| `engagement.follows.enabled` | later | opt | settings.manage | disabled until feature ships |
| `engagement.private_messages.enabled` | later | opt | settings.manage | disabled |
| `engagement.private_messages.retention_days` | later | opt | settings.manage | 365 |
| `engagement.reactions.emoji_set` | later | opt | settings.manage | simple like only first |

### I. Notifications & mail

| Key / capability | State | Store | Perm | Notes |
| --- | --- | --- | --- | --- |
| Global event × channel matrix | have | opt | settings.manage | reply/mention/moderation |
| Expand events: like, bookmark, follow, PM, digest | need | opt | settings.manage | as features ship |
| Template subject/body edit | need | table or opt | settings.manage | safe vars only |
| User notification preferences | need | user table | self | defaults inherit global |
| Digest frequency | later | opt | settings.manage | off |
| Quiet hours | later | user | self | off |
| Mail provider selection | have | provider | settings.manage | plugin-owned SMTP |

### J. Personalization, navigation, public chrome

| Key / capability | State | Store | Perm | Recommended default |
| --- | --- | --- | --- | --- |
| appearance preset / custom color | have | opt | settings.manage | pine_teal |
| footer copyright + 3 legal links | have | opt | settings.manage | placeholders |
| header nav items CRUD | need | table | settings.manage | Home, Categories, Tags, Search |
| footer extra columns/links | need | table | settings.manage | empty beyond legal |
| friend links | need | table | settings.manage | empty |
| homepage announcement banner | need | table | settings.manage | empty |
| custom homepage hero HTML (sanitized) | later | opt | settings.manage | empty — high XSS care |
| maintenance mode + message | need | opt | settings.manage | off |
| legal page bodies (terms/privacy/guidelines) | need | opt or pages | settings.manage | short recommended stubs |

### K. SEO (already rich — keep extending carefully)

Keep dedicated SEO page. Avoid duplicating site.name. Add only:

| Item | State |
| --- | --- |
| topic URL mode | have |
| content-type policies | have |
| sitemap partitions | have |
| per-category noindex override | later |

### L. Attachments & avatars (already rich)

Keep dedicated pages. Future:

| Item | State |
| --- | --- |
| image max dimensions in posts | need |
| auto-strip EXIF | need |
| virus scan provider | plugin |
| per-role upload quotas | later |

### M. Search operations

| Key / capability | State | Store | Perm |
| --- | --- | --- | --- |
| rebuild index job | have | jobs | search.manage |
| `search.include_comments` | need | opt | search.manage |
| `search.min_query_length` | need | opt | search.manage |
| synonyms / stop words | later | table | search.manage |
| ranking weights UI | later | opt | search.manage |

### N. Localization

| Key / capability | State | Notes |
| --- | --- | --- |
| built-in locale enable | have | zh-CN, en-US |
| language pack upload admin | need | accepted design |
| per-user locale | partial | profile track |

### O. Extensions & providers

| Key / capability | State |
| --- | --- |
| plugin enable / theme activate | have |
| extension settings pages | have |
| uninstall / upgrade / migrations | need (Track 2) |
| store marketplace | later |
| provider slots (mail done; storage next) | partial |

### P. Operations

| Key / capability | State | Notes |
| --- | --- | --- |
| jobs workbench | have | |
| read-only database browser | have | |
| audit log UI | need | |
| backup/restore status | later | prefer deploy.sh first |
| rate limit product knobs (post/login) | need | not Redis infra secrets |
| health / overview | have | |

### Q. Explicitly NOT in core settings soup

Keep out of `web_options` product forms (env / plugins / code):

- `DATABASE_URL`, Redis, Meilisearch master key, ports, worker counts
- CSRF secrets, option encryption key
- Vendor SMTP password UI beyond mail provider plugin
- Payment gateway credentials (payment plugins)
- Arbitrary PHP/JS inject "custom user code"

---

## UX Rules For Rich Settings

1. **Every settings group** has: intro, recommended callout, save, restore defaults.
2. **Progressive disclosure:** primary switches first; numeric limits under "高级".
3. **Dependency copy:** e.g. turning on email verification explains mail provider must work; link to mail center.
4. **Toast on save/restore**; field errors stay inline.
5. **Never empty-required screens** for core community policy — defaults always work offline.
6. **Public web-options** only expose values the frontend must read; secrets stay admin-only/masked.
7. **Category overrides** show "继承站点默认" clearly.
8. **Feature flags ship with the feature** — do not add dead toggles for unbuilt modules.
9. **i18n:** all labels/help in zh-CN + en-US.
10. **Tests:** defaults resolution, permission deny, restore path for each new group.

---

## Permission Map (new keys only when capability is distinct)

| Permission | Owns |
| --- | --- |
| `settings.manage` | site, brand, registration, newcomers, engagement switches, nav, maintenance, legal |
| `category.manage` | taxonomy + category policy overrides |
| `tag.manage` | tag policy |
| `moderation.manage` | review mode, auto rules, banned words |
| `user.manage` | bans/mutes admin actions (not all policy) |
| `seo.manage` | SEO page |
| `attachment.settings.manage` | attachment settings |
| `search.manage` | search ops |
| `extension.manage` | plugins/themes |
| `locale.manage` | language packs (designed) |
| `audit.view` (new later) | audit log read |
| `jobs.view` / `jobs.manage` | queue |

Do not invent a permission per toggle.

---

## Implementation Waves (rich, but sequenced)

Ship richness in waves so each PR is testable. **Do not open 200 dead toggles.**

### Wave 0 — Catalog & IA (docs only) ✅ this document

### Wave 1 — Community policy pack (highest operator value)

Site settings expansion + forum settings expansion:

- registration mode + email verify gates
- username rules
- newcomer trust ladder
- guest read
- list default sort
- topic/comment behavior switches (close replies, auto-lock, edit marks)
- maintenance mode
- login lockout knobs

**Landing:** options service + OpenAPI + admin tabs + public web-options for UX + tests.

### Wave 2 — Brand & public chrome ✅ 2026-07-12

- logo/favicon (+ apple-touch) options + attachment id / URL
- nav menu table + admin UI
- announcement banner
- legal page content stubs (Markdown options + public pages)
- friend links
- Handoff: `knowledge/sessions/2026-07-12-admin-settings-wave2.md`

### Wave 3 — Engagement switches (with Iteration A features)

- likes/bookmarks enable + scope
- notification matrix rows for new events
- view_count unique window

### Wave 4 — Category policy + scoped moderators

- category settings JSON/columns
- moderator assignments
- per-category moderation mode

### Wave 5 — Safety depth

- banned words
- auto-flag rules
- report reason catalog UI
- audit log viewer + retention

### Wave 6 — Search ops + localization packs + extension lifecycle

- search include comments / min length
- language pack admin
- plugin uninstall/upgrade productization

### Wave 7 — Social extras (on demand)

- follows, PM, digests, reactions set
- invite codes full module
- OAuth provider admin matrix (plugin slot)

### Wave 8 — Optional gamification / payments

- only after payment framework decision
- check-in, credits as plugins

---

## Suggested Option Naming Conventions

```text
site.*                  brand & identity
identity.*              account, registration, sessions, username
trust.*                 newcomer ladder
human_verification.*    CAPTCHA (existing)
forum.*                 global forum product policy
forum.pagination.*
forum.topics.*
forum.comments.*
forum.tags.*
forum.reading.*
forum.mentions.*
forum.composer.*
engagement.*            likes, bookmarks, follows, pm
notification.*          channel matrix (existing + expand)
moderation.*            when not in moderation_settings table
appearance.* / footer.* personalization (existing)
nav.*                   only if not fully table-driven
seo.*                   existing
attachment.* / avatar.* existing
search.*                search product knobs
mail.*                  provider-owned + core from/reply
```

Category overrides: not flat global keys; store on category entity.

---

## Anti-Patterns

| Avoid | Prefer |
| --- | --- |
| One 2000-line settings page | Tabs + folders + progressive disclosure |
| Toggle without enforcement | Feature + option + policy check + test together |
| Copying Discuz option names blindly | SForum names + recommended defaults |
| Core SMTP/OAuth vendor forms | Provider plugins + host selection UI |
| Public API leaking admin email / secrets | existing public/admin split |
| Hard-coded category names in services | options + taxonomy |

---

## Success Metrics

Operator can, without code changes:

1. Close registration or switch to invite/approval
2. Require email verification before posting
3. Soften spam via newcomer limits and link gates
4. Put the site in maintenance with a friendly message
5. Control guest visibility and homepage sort
6. Tune topic/comment social behavior (close replies, auto-lock, edit marks)
7. Set logo, nav, announcement, legal text
8. Enable/disable likes and bookmarks when shipped
9. Override policy per category and assign board moderators
10. Manage banned words and see who changed critical settings

---

## Next Session Entry Points

1. Start **Wave 1** implementation against this catalog (do not invent parallel keys).
2. For each option group: defaults in Options service → validation → admin UI → i18n → OpenAPI → tests → knowledge/module update.
3. When a wave adds a table (nav, friend links, banned words), add a short decision under `knowledge/decisions/`.
4. Keep Iteration A engagement features aligned with Wave 3 option keys.

## Open Product Choices (confirm when implementing)

1. Registration **approval** queue UI complexity — ship invite+email first if approval is heavy?
2. Logo storage: attachment reference vs external URL only?
3. Legal pages: Markdown in options vs first-class `static_pages` table?
4. Category policy: JSONB column vs normalized `category_settings` table?
5. Maintenance mode: full block vs read-only public?

Defaults if unconfirmed: (1) invite+email before approval UI, (2) attachment ref + URL fallback, (3) Markdown options first, (4) JSONB inherit-null, (5) block writes + show message, staff bypass.

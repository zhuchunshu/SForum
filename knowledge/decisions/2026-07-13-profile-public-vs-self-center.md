# Decision: Public profile vs self center (UI direction)

Date: 2026-07-13  
Status: accepted (UX shell implemented in default theme; follow/cover/portfolio APIs pending)

## Context

Current public profile (`/u/:username` in default theme) is a thin card: avatar,
name, basic stats, bio, recent topics. No cover, follow graph, or tabbed content
index. HTML demos under `tmp/demos/grok/profile-center/` explored six directions.

## Decision

Split **visitor-facing public profile** from **logged-in self center** (management hub):

| Surface | Route (intent) | Demo reference | Audience |
|---------|----------------|----------------|----------|
| **Public profile** | `/u/:username` | `01-bilibili-cover.html` | Anyone (SEO / social) |
| **Self center** | e.g. `/me` or settings hub home | `03-self-dashboard.html` | Authenticated self only |

### Public profile (01 · Bilibili Cover)

- Full-width **cover** (default gradient; optional upload later).
- Avatar over cover edge; display name, badges, `@username`.
- **Follow** / message / more; stats: following, followers, topics, likes.
- Sticky **tabs**: portfolio (works), topics, comments, following, followers
  (and optional likes / extension tabs).
- Edit entry when `isSelf`: link into self center or profile settings, not a
  full management dashboard on the public page.

### Self center (03 · Self Dashboard)

- Logged-in **「我的中心」**: completeness, shortcuts, drafts, moderation
  queue, bookmarks, security/privacy entries.
- Tabs for own content plus drafts / saved — not the same layout as public.
- Clear CTA: 「查看公开主页」 → `/u/:self`.
- Distinct from granular settings pages (`/settings/profile`, security, etc.);
  those remain detail forms; self center is the **hub**.

## Non-goals (this decision)

- Does not implement follow API, cover storage, or portfolio domain model yet.
- Does not pick final path names beyond intent (`/me` vs `/my` vs nav label).
- Does not adopt demos 02/04/05/06 as primary; they remain reference only.

## Implementation notes (when building)

1. Theme: primarily `extensions/builtin/themes/sforum-default` pages.
2. Reuse existing profile API (`PublicProfile`, avatar, bio); extend contracts
   when follow / cover / portfolio land.
3. Keep `extensionTabs` on public profile; self center may surface host links
   (content review, settings) more prominently.
4. Permissions: public read vs login-required self hub; follow mutations auth +
   privacy later.
5. Beginner-friendly: safe defaults for cover, empty states that guide posting.

## Demo paths

- Index: `tmp/demos/grok/profile-center/index.html`
- Public: `tmp/demos/grok/profile-center/01-bilibili-cover.html`
- Self: `tmp/demos/grok/profile-center/03-self-dashboard.html`

## Related

- Existing page: `extensions/builtin/themes/sforum-default/layer/app/pages/u/[username].vue`
- Settings: `.../pages/settings/profile.vue`
- API types: `apps/web/app/composables/useProfileApi.ts`

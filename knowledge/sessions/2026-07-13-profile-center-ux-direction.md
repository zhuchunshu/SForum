# 2026-07-13 Session Handoff — Profile center UX direction

## Changed

- Added 6 interactive HTML demos: `tmp/demos/grok/profile-center/`
- **Product pick (user confirmed):**
  - **01 Bilibili Cover** → public profile (`/u/:username`)
  - **03 Self Dashboard** → logged-in self center / management hub
- Decision: `knowledge/decisions/2026-07-13-profile-public-vs-self-center.md`

## Decisions

- Public social space vs private management hub are **two pages**, not one
  overloaded layout with role toggles only.

## Implemented (same day, UI shell)

- Public profile: `extensions/builtin/themes/sforum-default/layer/app/pages/u/[username].vue`
  (cover gradient, stats, sticky tabs, follow toast placeholder).
- Self center: `.../pages/my/index.vue` at `/my` (completeness, shortcuts, overview/topics).
- Nav: user menu →「我的中心」`/my`;「公开主页」still `/u/:user`.
- Styles: `.../assets/css/sforum-profile.css` registered in theme `nuxt.config.ts`.
- i18n: `profile.*` + `myCenter.*` + `nav.myCenter` in zh-CN / en-US.

## Next

- Follow/followers API + real counts (replace `—` placeholders).
- Cover upload + portfolio/works domain.
- Optional: comments timeline on public profile tab.
- Do **not** mix with unrelated API attachment work on the same commit.

## Open Questions

- Portfolio tab: pin/featured topics only in v1, or attachments?
- Self-center path fixed as `/my` (alongside existing `/my/content-review`).

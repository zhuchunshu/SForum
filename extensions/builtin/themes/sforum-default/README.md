# SForum Default Theme

SForum Default Theme is the protected built-in runtime theme for v1.

It supplies Page Registry templates, public skin tokens/assets, and a Schema
Settings Document. Core owns Vue islands, admin UI, auth/session, API clients,
i18n, SEO, permissions, and reusable `SF*` components.

## Presentation ownership (do not dual-write)

| Layer | Owns | Does **not** own |
|-------|------|------------------|
| **L1 templates** (`templates/*.html`) | Page shell, chrome regions, which host islands to mount (`sf-navbar`, `sf-home-page`, …) | Business data, interactive list logic |
| **L0 tokens** (`assets/tokens.css`) | Layout/surface tokens: widths, surfaces, borders, text, radius, shadow | **Accent / 主色**（归站点 appearance）、list BEM |
| **L0 skin** (`assets/theme.css`) | Shell layout that tokens alone cannot express (sticky columns, nav chrome) | Mirroring host list/main-bar/chip/avatar CSS; hard-coded accent |
| **Host `SF*` islands** | Data, permissions, Tailwind presentation defaults | Theme-specific marketing chrome |
| **Component Registry** | Theme/plugin `wrap` / `replace` / `add` / `hide` of registered targets (SSR template and/or trusted L2) | Silent CSS monkey-patches of host BEM |

**主色**：公开壳使用 `var(--sf-accent)` 等 appearance 变量（`html[data-sforum-theme]` /
自定义色）。默认主题 **不得** 在 `tokens.css` 上覆盖 `--sf-accent*`。

List rows, main-bar, chips 是 **host components + Tailwind**，读 `var(--sf-public-*)`
与 `var(--sf-accent)`。头像一律用全局 **`SFAvatar`**（`size` + `AvatarView` /
字头色板），主题不要手写 avatar DOM。深度换肤走 L1 / Component Registry，不要在
`theme.css` 双写列表行 CSS。

Stable hooks for extension:

- Page regions: `data-theme-region`, `data-layout="fullwidth-3col"`
- List panel: `data-sf-region="topic-list"`
- Topic row root: `data-sf-component="forum.topic_list_row"`
- Avatar root class: `sf-avatar`（尺寸/色板在 `SFAvatar` 组件内）

## Public layout (fullwidth-3col)

Home and topic show use a **full-viewport-width three-column** shell (demo:
`tmp/demos/grok/forum-fullwidth-3col/`):

| Region | Home | Topic |
|--------|------|-------|
| Left | `SFHomeNavigation` | `SFHomeNavigation` (route mode) |
| Center | topic list feed | article + comments |
| Right | `SFHomeRightRail` | `SFTopicSideCard` |

Collapse: wide = 3 columns; ≤1180px hide right rail; ≤960px single column +
mobile category control.

## Dev activation

After package edits: rsync to `storage/builtin-dev/themes/sforum-default/`,
restart API (`SyncBuiltins` stages a new digest), then super_admin activate
with `approveCoreReplacements` when L1 bindings must refresh.

Uploaded themes use the same `theme.json` + assets/templates contract.
Activation switches Page Registry + skin; it does not rebuild Nuxt.

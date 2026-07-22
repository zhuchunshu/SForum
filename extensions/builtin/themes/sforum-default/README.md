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
| **L0 skin** (`assets/theme.css`, `assets/hybrid-forum.css`) | Shell layout that tokens alone cannot express (sticky columns, nav chrome, accepted forum surface) | Business data, permissions, hard-coded accent |
| **Host `SF*` islands** | Data, permissions, Tailwind presentation defaults | Theme-specific marketing chrome |
| **Component Registry** | Theme/plugin `wrap` / `replace` / `add` / `hide` of registered targets (SSR template and/or trusted L2) | Silent CSS monkey-patches of host BEM |

**主色**：公开壳使用 `var(--sf-accent)` 等 appearance 变量（`html[data-sforum-theme]` /
预设 `pine_teal|ocean_blue|violet|rose|amber` 或 `custom:#rrggbb`）。默认主题
**不得** 在 `tokens.css` / `theme.css` / `hybrid-forum.css` 上覆盖 `--sf-accent*`。

**日夜模式**：表面色走 `:root` / `.dark` 的 `--sf-public-*`；顶栏、列表、评论空态
等壳层读这些 token。暗色下强调文字可用 `var(--sf-accent-dark)` 提高对比。

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

Home and topic show use the approved Flarum-style three-column shell (demo:
`tmp/demos/sforum-hybrid-topic-list/`):

| Region | Home | Topic |
|--------|------|-------|
| Left | `SFHomeNavigation` | `SFHomeNavigation` (route mode) |
| Center | topic list feed | article + comments |
| Right | `SFHomeRightRail` | `SFTopicSideCard` |

Wide view keeps moderate `18px` outer gutters and `12px` column gaps. At
≤1180px the fixed right rail is removed from the grid. At ≤980px the navbar
becomes two rows and both rails remain available through left/right drawers.
The topic page uses one chronological comment stream; reply relationships are
shown as source links and the only reply composer stays expanded after the
final comment.

## Dev activation

After package edits: rsync to `storage/builtin-dev/themes/sforum-default/`,
restart API (`SyncBuiltins` stages a new digest), then super_admin activate
with `approveCoreReplacements` when L1 bindings must refresh.

Uploaded themes use the same `theme.json` + assets/templates contract.
Activation switches Page Registry + skin; it does not rebuild Nuxt.

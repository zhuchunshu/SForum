# 2026-07-22 Hybrid topic typography fidelity

## Changed

- 对照 `tmp/demos/sforum-hybrid-topic-list`（`data-variant="b"` 终态）用 Playwright
  量了 demo / 实站计算样式，修正默认主题话题页与顶栏字体体系。
- **标题**：去掉错误的 `Noto Serif SC` 30/700，改为 demo 终态
  `Noto Sans SC` `clamp(24px, 2vw, 30px)` / 600。
- **正文**：14px / line-height 1.9 / `#404751` / Noto Sans SC；压过 Tailwind
  `prose` 子元素默认字号。
- **h2 / pre / byline / 按钮 / 搜索 / 评论标题** 与 demo 实测对齐
  （h2 18/700；pre `#252a31` 4px 圆角 12/1.75；作者 12/700；meta 10；
  按钮 38px 高 radius 4；搜索 input 13px；评论 h2 18/600）。
- **首页 feed 标题**：20/600 无衬线（不再 27 衬线）。
- **左栏分类**：改回 demo 彩色圆点 `.sf-home-navigation__cat-dot`。
- **顶栏 active**：主色文字、无下划线（demo Flarum pass 终态）。
- 宿主选择器提升到 `.sf-theme--default …`，避免旧 L0 包 digest 抢胜。
- **主题色只走后台** `appearance.theme` / `--sf-accent*`，不写死 demo 玫红。
  误改的 `custom:#d94763` 已恢复为站点原 `ocean_blue`。
- `@font-face` 修复：Noto/DM 用 `format("truetype")` + 合法 weight 轴；
  公开主题树 `*` 继承字族，避免退回 Inter。
- 主题源码 rsync → `storage/builtin-dev`；宿主 CSS 可独立还原字体。

## Files

- `apps/web/app/assets/css/sforum-topic.css`
- `apps/web/app/assets/css/sforum-home.css`
- `apps/web/app/assets/css/sforum-theme.css`
- `apps/web/app/components/SFHomeNavigation.vue`
- `extensions/builtin/themes/sforum-default/assets/hybrid-forum.css`
- `apps/web/tests/defaultThemeHomepage.test.ts`

## Verify

- Playwright：title 28.8/600、prose 14/1.9 #404751、pre 12/#252a31、accent
  `#d94763`、logo/compose `rgb(217,71,99)`。
- `bun test` defaultThemeHomepage / TopicPage / Navbar：31 pass。

## Next

- 管理端激活 staged default theme 包，让 hybrid-forum L0 与源码 digest 一致。
- 若运营商不想用 demo 玫红，在「外观」改回预设即可；字体规格与主色解耦。

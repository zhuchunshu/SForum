# 2026-10-17 Session Handoff — 公开列表壳窄屏画布对齐 PC

## Changed

- `apps/web/app/assets/css/sforum-home.css`：把 `.sforum-home__layout` 的
  `background: var(--sf-public-surface);` 从 `@media (min-width: 981px)` 提到基规则
  （净 +2 行，958/1000）。分类/标签/搜索页共用同一类，一并生效；宿主 CSS 同时覆盖
  主题 L1 与 Core fallback 两条渲染路径，主题不可变制品无需 digest/重建/激活。
- `apps/web/tests/forum/defaultThemeHomepage.test.ts`：新增静态契约测试
  `paints the shared list shell surface at every width so narrow screens match desktop`
  （基规则必须含 surface 底、不得写成 `--sf-public-bg`），15 → 16 pass。
- `knowledge/modules/frontend.md`：Public Chrome Geometry 增加一条列表壳画布契约。

## Decisions

- 「跟 PC 一样」按既有记录解释为取 `--sf-public-surface` 白底，与 G1 白底扁平列表
  （`sforum-home.css` ≤720 注释）和 `sforum-theme.css` L1037「窄屏中列通铺 surface
  白底到视口边缘」一致，而不是把 PC 反向改成灰底。
- 不为首页开单独的 layout 选择器：桌面本来就对同族 layout 一视同仁（`hybrid-forum.css`
  ≥981 统一铺 surface），开特例会造出第二套几何契约。
- 用 token 而非硬编码白色：深色与 12 套 `data-sforum-light-background` 预设自动对齐。

## Next

- 生产发布需重新构建 Web 产物（宿主 CSS 走 Nuxt 打包）。

## Open Questions

- **已知残留**：窄屏 `.sforum-home__layout` 的 `min-height: 0` 是既有决定
  （`sforum-theme.css` ≤980 块，为免短列表被撑出一屏空白），所以空/短列表时列表下方
  仍可能露一条画布色，而 PC 不会（PC layout `min-height: calc(100vh - topbar)`）。
  修它要么回退那条决定，要么用 `(0,5,0)` 级选择器压主题 `chrome-body` 规则，均超出
  「背景对齐」范围，本轮未做。
- 底部固定导航原先是「白 97% 浮在灰底上」；画布变白后仅靠 `backdrop-filter: blur` +
  `0 -2px 14px` 阴影分隔。若视觉上被吃掉，再单独讨论是否补顶部细线。
- `sforum-topic-page__layout` / notifications / settings / moderation 的窄屏画布未取证
  （登录态页面本轮无账号），用户诉求仅首页。

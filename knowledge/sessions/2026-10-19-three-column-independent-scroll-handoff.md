# 2026-10-19 三栏独立滚动 Handoff

## Changed

- 桌面（≥981px）公开三栏壳改为三列各自独立滚动：`__sidebar` / `__main` / `__right`
  （topic 为 `__sidebar` / `__main` / `__side`）统一 `position: sticky` +
  `height: calc(100vh - var(--sf-public-topbar-height))` + `min-height: 0` +
  `overflow-y: auto` + `overscroll-behavior: contain`，布局容器加
  `align-items: start`。
- 左右栏隐藏滚动条但保留滚动能力：`scrollbar-width: none` +
  `::-webkit-scrollbar { display: none }`（`sforum-theme.css` 一处全局规则，
  主题包 `hybrid-forum.css` 同步），中列保留可见滚动条作为主滚动入口。
  隐藏后左右栏列宽与内容不溢出的栏完全一致，不出现沟槽。
- 几何三处同值：`apps/web/app/assets/css/sforum-theme.css`（旧主题 artifact 兼容块
  与 Host 回退壳）、五个页面级 CSS（`sforum-home` / `sforum-topic` /
  `sforum-settings` / `SFNotificationsPage` / `sforum-moderation`）、主题包
  `extensions/builtin/themes/sforum-default/assets/hybrid-forum.css`。
- 删除了把左右栏打回 `static` / `overflow: visible` 的反向规则，以及
  `.sforum-topic-page__sidebar/__side { position: relative }` 覆盖。
- 测试：`defaultThemeHomepage` 用例改为「三列各自滚动」，同步更新
  `defaultThemeTopicPage`、`notificationsPage` 的几何断言。

## Evidence

- 改前首页：`document.scrollHeight 1545 / clientHeight 810`，左右栏 1487px
  且 `overflow: visible`，只有中列 sticky 滚动；滚轮落在左右栏滚的是 document，
  于是整行一起动。
- 改后首页：document 810（不可滚），三列均 752px、`position: sticky`、
  `overflow-y: auto`；分别设置 `scrollTop` 为 160 / 240 / 40 互不影响，
  `window.scrollY` 保持 0。
- 话题页同构：三列 752px，`scrollHeight` 1487 / 3319 / 758，document 810。
- 移动端 500×864：单列、document 可滚 2424，侧栏 `display: contents`，右栏
  `display: none`，未受影响。
- 隐藏滚动条后：首页 1670×864 左右栏 `scrollbarWidth: none`、`offsetWidth -
  clientWidth = 1`（只有 1px 边框），中列 `thin` + gutter 10；把视口压到
  1633×700 后左右栏内容溢出（1487 / 682 vs 642），`scrollTop` 仍可设到 200 / 40，
  滚动能力保留。

## Decisions

- 不锁 shell 高度（既有测试明确禁止 `height: 100vh`），用 sticky + 定高让三列
  独立滚动，同时保留壳层页脚在文档流中可滚出。
- 左右栏不加 `scrollbar-gutter`，未溢出时列宽与改造前完全一致；只有溢出列
  出现系统滚动条。中列保留既有 `scrollbar-gutter: stable`。

## Next

- 主题包 `sforum.default-theme` 已 `extension build`（digest + validate +
  contract test 通过）并同步到 `storage/builtin-dev`。当前 active 版本是
  `16540`（含三列独立滚动），最新包 `16562` 已 stage 但未激活；两者行为一致，
  因为 `sforum-theme.css` 兼容块特异性更高、覆盖主题 CSS。激活 `16562` 可让
  主题制品与源码完全同步（含左右栏滚动条隐藏规则）。
- 通知页与个人页没有中列页脚，document 仍可滚出壳层页脚（约 69px）；滚轮在
  列内被 `contain` 拦住，不产生联动。若要彻底锁死页面滚动，需要把页脚移入
  中列（页脚宽度会从中栏宽变成跨栏宽，属于视觉变化）。

## Open Questions

- 是否把通知页/个人页的页脚并入中列，换取「document 完全不可滚」。

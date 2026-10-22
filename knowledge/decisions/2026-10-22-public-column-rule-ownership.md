# Decision: 公共三栏壳的列线归宿主所有，皮肤 CSS 永远是基底

## Status

Accepted

## Context

公开页三栏壳的列线（左栏右缘、右栏左缘的 1px 竖线）曾由宿主 CSS 与默认主题皮肤
同时声明，且两者特异性相同：

- 宿主 `app/assets/css/sforum-topic.css`：`.sforum-topic-page__sidebar { border-right: 1px solid }`（0,1,0）
- 皮肤 `extensions/builtin/themes/sforum-default/assets/theme.css`：`.sforum-topic-page__sidebar { border-right: 0 }`（0,1,0）
  + `::after { right: -12px }` 画在栏间沟槽

谁赢只取决于样式表加载顺序。SSR 首屏皮肤在前、宿主在后，列线正常；但
`app.vue` 的 `syncThemeSkin()` 在进入管理端时 `themeSkin.clear()` 卸载皮肤链，
回到公开页再 `restoreLastPublic()` + `refresh()`，`useHead` 把 `<link>` 追加到
`<head>` 末尾，皮肤就反超宿主 → 内容页两条列线一起消失。首页当时没有这个问题，
只因皮肤对 `.sforum-home__sidebar` 是重复声明同样的 1px 边框，而不是清零。

皮肤用 `::after/::before` 画沟槽线本身也不成立：这两列在 ≥981px 是
`position: sticky; overflow-y: auto`，越界的伪元素会被 overflow 裁掉。

## Decision

1. **列线所有者是宿主。** 皮肤只负责 padding、背景等表层，不声明、不清零列线。
   `sforum-topic.css` 除栏自身 1px 边框外，再用容器作用域声明一次
   （`.sforum-topic-page .sforum-topic-page__sidebar`，(0,2,0)），赢在特异性而不是顺序。
2. **皮肤 CSS 是基底，宿主 CSS 细化。** `app.vue` / `error.vue` 输出的皮肤
   `<link>` 带 `tagPriority: 'critical'`，固定在 `<head>` 最前；皮肤要覆盖宿主必须
   提高特异性（现有 `.sf-theme--default …` 已是 (0,2,0)），不能依赖后加载。
3. 默认主题源码移除 `border-right/border-left: 0` 与 ±12px 伪元素线；宿主
   `sforum-topic.css` 里的 `content: none` 继续兼容已安装的旧主题 artifact。

## Consequences

- 内容页与首页的列线由同一策略产生，管理端往返、主题切换都不会再改变几何。
- 任何新主题若想在与宿主同特异性下靠顺序取胜，会失效；这是有意的收紧，
  正确做法是提高特异性。
- 旧 artifact 仍被宿主兼容，不需要强制全站升级主题。

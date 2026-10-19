# Decision: 错误页不得硬依赖可选插件

## Status

Accepted (2026-10-19)

## Context

2026-10-19 排查 `app/error.vue` 抛 `Need to install with 'app.use' function`
（vue-i18n `NOT_INSTALLED`）时确认：这条报错是二次故障，不是根因。

- `@nuxtjs/i18n` 的 runtime 插件把 `vueApp.use(i18n)` 放在 setup 最后一步，
  `parallelPlugin` 默认 `false`；setup 中任何提前失败（消息加载、请求上下文、
  模块解析）都会留下一个"没有 i18n 的 app"。
- `app/error.vue` 原先无条件调用 `useI18n()` 与 `useLocaleHead()`；后者也只由
  该插件 `provide('localeHead')` 暴露。
- `app/components/errors/` 下 8 个组件全部调用 `useI18n()`，包括名为
  "emergency" 的兜底页。因此 i18n 失败时错误页整条渲染链二次崩溃，用户看到的
  是错误页自己的异常，原始错误被完全掩盖。

错误页是"应用已经出问题"时唯一的出口，它的可用性不能取决于任何可选插件是否
成功安装。

## Decision

1. 错误页外壳通过 `useErrorPageShell()` 解析依赖：只有 `$i18n` 与
   `$localeHead` **同时**存在才视为就绪；未就绪时不调用任何 i18n 组合式函数，
   直接使用零依赖的兜底翻译器与空 head。
2. 未就绪时整页降级为 `SFSystemErrorBootstrapFallback`：不碰 i18n、主题皮肤、
   API，只渲染状态码与双语提示。
3. 判定使用双条件而非单一信号：半安装的 i18n（只有 composer 没有
   `localeHead`）同样不可信。
4. 主题错误页（`SFSystemErrorPage`）、`useSeoMeta`、`useHead` 主题分支、
   `useSystemErrorPagePresentation` 全部纳入 `i18nReady` 门控。健康路径行为
   完全不变（`i18nReady` 为真时与改动前逐项等价）。

## Consequences

- 代价：i18n 失败时错误页丢失主题皮肤与完整错误面板，只保留最简兜底。这是
  刻意的 fail-closed 取舍——不降级就必然二次崩溃。
- 代价：`i18nReady` 依赖上游 `provide('localeHead')` 契约；
  `apps/web/tests/errors/errorPageShell.test.ts` 锁住该契约，上游变更时测试先红。
- 收益：任何插件加载失败都会显示"应用初始化失败 + 状态码"，而不是把
  插件内部错误伪装成错误页自身的异常；原始错误重新变得可诊断。
- 后续：若要让 i18n 失败时仍保留主题皮肤，需要让 `app/components/errors/`
  下所有组件对 i18n 可降级，属于独立工作项，不在本次范围内。

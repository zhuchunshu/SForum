# 2026-10-19 Session Handoff — 错误页 i18n 崩溃与错误页零依赖兜底

## Changed

- `apps/web/app/utils/errors/errorPageShell.ts`（新）：错误页 i18n 就绪判定
  （`isErrorPageI18nReady`，双条件 `$i18n` + `$localeHead`）、空 head 构造、
  文本插值、兜底翻译器。
- `apps/web/app/composables/errors/useErrorPageShell.ts`（新）：运行时装配，
  未就绪时**不调用** `useI18n()` / `useLocaleHead()`，直接返回兜底实现。
- `apps/web/app/components/errors/SFSystemErrorBootstrapFallback.vue`（新）：
  零依赖兜底页（不碰 i18n、主题皮肤、API），只渲染状态码 + 双语提示 + 重载。
- `apps/web/app/error.vue`：`useI18n()`/`useLocaleHead()` 换成
  `useErrorPageShell()`；`useSeoMeta` / `useHead` 主题分支 /
  `useSystemErrorPagePresentation` 增加 `i18nReady` 门控；模板改为
  `BootstrapFallback → SFSystemErrorPage → UApp` 三段链（既有静态契约不变）。
- `apps/web/tests/errors/errorPageShell.test.ts`（新）：6 条行为用例，其中
  「未就绪时绝不调用 i18n 组合式函数」用会抛错的替身断言。
- 删除误提交的 `apps/web/pnpm-lock.yaml`、`apps/web/pnpm-workspace.yaml`
  （`2c133486e` 混入的 12,347 行，工具链只用 bun）。
- `AGENTS.md` Commands 段：`apps/web` 只用 bun；切换包管理器/重装后必须删
  `.nuxt` 与 `node_modules/.vite` 再重启 dev server。

## Decisions

- `decisions/2026-10-19-error-page-must-not-depend-on-plugins.md`

## 事故链（已核实）

1. 现象是 vue-i18n 的 `NOT_INSTALLED`（`useI18n()` 发现
   `app.__VUE_I18N_SYMBOL__` 为空），**不是根因**。
2. `@nuxtjs/i18n` 把 `vueApp.use(i18n)` 放在 runtime 插件 setup 最后一步，
   `parallelPlugin` 默认 false；setup 中任何提前失败都留下"没有 i18n 的 app"。
3. `app/error.vue` 原先无条件 `useI18n()` + `useLocaleHead()`，而
   `app/components/errors/` 下 8 个组件全部依赖 `useI18n`（含
   `SFSystemErrorEmergencyPage`），所以 i18n 失败时错误页整条链二次崩溃，
   把原始错误覆盖成这个二义报错。
4. 触发环境：dev server 于 04:51 启动（解析到 `.pnpm/nuxt@4.6.1`），
   `bun install` 在 09:11 重写顶层 `node_modules` 指向 `.bun`，磁盘上同时存在
   `.pnpm/vue-i18n@11.4.14` 与 `.bun/vue-i18n@11.4.6`、两份 vue。
   运行中的进程继续用已加载模块，HMR 后新解析的模块来自新树。

## Next

- 重启 dev server 后确认 `/login`、`/nonexistent`、`/settings` 不再出现
  dev overlay（本次未做浏览器侧复验：:3000 与 API 在排查期间均已停止）。
- 若仍复现，先看浏览器 Console/Network 中 `NOT_INSTALLED` 之前的原始错误
  （`504 Outdated Optimize Dep` / 动态 import 失败 / 消息加载失败）。
- 可选后续：让 `app/components/errors/` 其余组件也具备 i18n 降级能力，
  使 i18n 失败时仍保留主题皮肤（本次刻意不做，避免扩大改动面）。

## Open Questions

- `@nuxtjs/i18n` 未来若改用非 `provide('localeHead')` 的方式暴露该组合式函数，
  `isErrorPageI18nReady` 会误判为未就绪 → 生产错误页全部走兜底。
  `errorPageShell.test.ts` 的「delegates to the real composables」用例锁住了
  这个契约，上游变更时测试会先红。

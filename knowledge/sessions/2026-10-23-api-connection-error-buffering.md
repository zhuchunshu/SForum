# 2026-10-23 Session Handoff

## Changed

- 「无法连接后端 API」弹窗误报治理（apps/web 前端，后端零改动）：
  - 新增探测恢复引擎 `apps/web/app/composables/errors/useApiConnectionRecovery.ts`：
    连接失败先进 pending 缓冲并按 1s→2s→4s（4s 封顶）退避探测
    `GET {apiBaseUrl}/health`（4s 超时，慢而活的部署不误判）；探测持续
    失败跨过 `GRACE_MS=15000` 才真正打开模态（首个失败点 ~15s+探测耗时）；
    打开态每 5s 探测，恢复即自动 close 并写 `recoveredAt`。
  - `useApiConnectionError.ts` 收薄为门面：`show()` 委托引擎；state 增加
    `failureCount`、`recoveredAt`；错误分类纯函数抽到
    `errors/apiConnectionErrorClassify.ts`（避免门面↔引擎循环依赖）。
  - `useApiClient.ts`：安全方法（GET/未指定 method）连接类失败原地重试一次
    （非 abort 才重试），二败才进缓冲；unsafe 方法保持单次防重复提交；默认
    timeout GET 15s / unsafe 30s（面向慢站点放宽），调用方显式传入不覆盖。
  - 新增 `plugins/assetReload.client.ts` + `utils/errors/assetReload.ts`：
    `/_nuxt/**` chunk 资源错误（capture 阶段 error）与动态 import 失败
    （unhandledrejection，含 cause 链、ERR_NETWORK_CHANGED）分流为「站点刚
    发布了新版本，正在刷新」toast + 800ms 后 reload；sessionStorage 10s 冷却
    防刷新循环；不进断线模态。
  - `SFApiConnectionModal.vue`：描述区新增「正在自动重连」行；watch
    `recoveredAt` 出恢复成功 toast（success, 10s 自动消失）。
  - i18n：`errors.apiConnection.recovering/recovered` +
    `errors.assetReload.toast`（zh-CN / en-US 同步）。

## Decisions

- 探测「上限 4 次」改为 4s 封顶的持续探测 + 宽限期失败判定：硬截断会让
  pending 悬空无结论，且与打开态 5s 监控矛盾。
- 模态实际打开点 ~15s+（首个跨过 15s 宽限的失败探测；探测自身 4s 超时在慢
  响应部署下自然拉长）：用户确认慢站点访问常见，故意比原「~10s」验收更宽，
  优先压误报而非快弹窗。
- toast 后延迟 800ms 再 reload：立即 reload 会让提示不可见，两者不可兼得。
- 测试用 bun:test 的 `jest.useFakeTimers()`（bun 无 `vi`；实测其同步 mock
  `Date.now()`，宽限判定可确定性驱动）。
- 探测路径零 Nuxt composable 调用（probeBaseUrl 由 useApiClient 显式传入，
  兜底 try/catch useRuntimeConfig），不新增 setup 外上下文风险；
  `import.meta.hot.dispose` 清理模块定时器防 HMR 泄漏。
- 未抬高 `app/composables` 根目录封闭基线：新文件全部进 `errors/` 域子目录。

## Next

- 浏览器 QA（可选但推荐）：模拟 API 抖动 2s / 宕机 30s / 发布换 chunk 三个
  场景，确认无弹窗 / ~11s 弹窗 + 恢复自动关 / toast+自动刷新无断线弹窗。
- 观察线上误报率；若宽限期/退避节奏需调，常量集中在
  `useApiConnectionRecovery.ts` 顶部（GRACE_MS、PROBE_*、MONITOR_INTERVAL_MS）。

## Open Questions

- 部署层 healthcheck / Caddy 健康感知是另一条线，本次明确不做。

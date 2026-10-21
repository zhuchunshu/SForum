# Decision: 公开只读资产按跨源可读发布

## Status

Accepted (2026-10-22)

## Context

手机 QQ 内置浏览器（X5 内核 + 云加速）打开站点时先自动刷新，随后停在 SForum 的
500 页，标题栏显示 `500 – Cross-origin script load denied by Cross-Origin-Resource-Policy header`；
同一地址在 Chrome / Edge / Safari 正常。

排查确认的事实：

- Nuxt renderer 硬编码 `crossorigin: ""`，`<script type="module">`、样式与
  modulepreload 全部带该属性；**模块脚本无论有无该属性都走 CORS**，所以入口
  chunk 只能靠 `Access-Control-Allow-Origin` 存活。
- `/_nuxt/**` 生产由 Nitro 静态处理器输出，不带 ACAO。ACAO 补丁只落在 legacy
  `deploy/caddy/Caddyfile`；zero-downtime 生产走 `upgrade.sh` 的
  `write_router_config()` 生成器，那里只写 `X-SForum-Active-Slot`，补丁从未生效。
- `/_sforum/assets/**` 由 Go API 输出 `Cross-Origin-Resource-Policy: same-origin`
  且无 ACAO，而 SSR 以 `crossorigin="anonymous"` + `integrity` 加载它；报错点名的
  就是这枚 CORP 头。
- 主题 CSS 不设 CORP、也不带 `crossorigin`，所以换源后仍能加载——与"页面有样式、
  应用起不来"的现场一致。
- 本地强制渲染 500 验证：`<title>服务器暂时没有回应好</title>`，`statusMessage`
  只进 HTTP reason phrase。截图标题栏的 `500 – Cross-origin…` 不是我们的
  `document.title`，是 X5 内核错误页。
- 错误页三个动作全是 `@click`，入口 chunk 被拒后页面没有任何出口。

## Decision

1. **公开只读资产由应用自己发跨源头**：`ACAO: *` +
   `Cross-Origin-Resource-Policy: cross-origin` + `Timing-Allow-Origin: *`。
   - `/_nuxt/**`：Nitro `routeRules.headers`（路由规则中间件先于所有 handler，
     静态处理器只在头缺失时补，不删头）。
   - `/_sforum/assets/**`：`proxyRouteRequest` 的 `onResponse` 覆盖上游头
     （sendProxy 先复制上游头，再跑 onResponse）。
   - `/_sforum/icons/**`：路由直接调用同一个 helper。
   - 私有 `/_sforum/private-assets/**`、`Extensions/frontend.go` 保持
     `same-origin`，路由规则只精确匹配，不用 `/_sforum/**` 泛匹配。
2. **扩展样式不再声明 `crossorigin`**：同源时 `integrity` 仍校验；被中间层换源时
   按 no-cors 加载（opaque 响应自动跳过 SRI），而不是整块样式被 CORS 拒载。
3. **ingress 冗余**：legacy `deploy/caddy/Caddyfile`、`upgrade.sh` 生成器与
   `deploy/router/Caddyfile.example` 都写同名头，作为"中间层抹头"的兜底，不作为
   唯一依赖。
4. **错误页动作一律是真链接**：`SFButton` 支持 `href`，`useErrorPageActions()`
   同时给出 href 与事件；有 JS 走 SPA 恢复，无 JS 由浏览器导航。

## Consequences

- 公开资产可被任意来源读取。它们本来就是 hash 定址、`immutable`、无凭据，且
  私有资产仍锁在 same-origin。
- SRI 在"内容被改写"的链路上不再生效（同源仍生效）。这是刻意取舍：中间人改写与
  SRI 天然互斥，保留 SRI 就等于让页面在 QQ 里整块样式消失。
- 管理后台在 X5 云加速下仍可能不可用，属明确边界，不为此放宽私有资产策略。
- 若某中间层直接抹掉 ACAO，模块脚本仍会被拒——这是 CORS 硬约束，只剩换域或
  关闭云加速；本决策把非模块资源全部降级为 no-cors，把影响面压到最小。

# 2026-10-22 公开资产跨源发布（QQ 内置浏览器 500）

## Changed

- `apps/web/nuxt.config.ts`：新增 `publicAssetCorsHeaders` 与三条 routeRules
  （`/_nuxt/**`、`/_sforum/assets/**`、`/_sforum/icons/**`），私有资产不匹配。
- `apps/web/server/utils/pluginRouteProxy.ts`：新增 `applyPublicAssetCorsHeaders()`，
  在 `omitCredentials` 代理的 `onResponse` 覆盖上游 `CORP: same-origin` 并补 ACAO/TAO。
- `apps/web/server/routes/_sforum/icons/[collection]/[name].get.ts`：改用同一 helper。
- `apps/api/app/Http/Controllers/Extensions/public_frontend.go`：两处公开资产 CORP
  改 `cross-origin`；`Extensions/frontend.go`（私有）保持 `same-origin`。
- `apps/web/app/app.vue`、`apps/web/app/runtime/public-extensions/assets.ts`：扩展样式
  去掉 `crossorigin`，保留 `integrity`。
- `apps/web/app/composables/errors/useErrorPageActions.ts`（新）+ `SFButton` 支持
  `href`：`SFSystemErrorActions`、`SFErrorPagePanel`、`SFSystemErrorBootstrapFallback`
  的动作全部改为真链接，JS 只做增强。
- `deploy/caddy/Caddyfile`、`upgrade.sh` 的 `write_router_config()`、
  `deploy/router/Caddyfile.example`：补 `Cross-Origin-Resource-Policy "cross-origin"`。

## Decisions

- `decisions/2026-10-22-public-asset-cross-origin.md`：公开资产按跨源可读发布；
  SRI 在改写链路上让位；私有资产边界不动。

## Verification

- `bun test` 全量 1503 通过；`bun run typecheck` 通过；`node tests/validate-architecture-boundaries.mjs` 通过。
- `go build ./...` + Extensions/Support 包测试通过。
- 本机 dev 实测：`/_sforum/assets/extensions/.../shortcodes.css` 与
  `/_sforum/icons/tabler/home.svg` 均返回 `access-control-allow-origin: *` +
  `cross-origin-resource-policy: cross-origin`（上游 API 仍是旧头，证明代理覆盖生效）。
- SSR HTML 的扩展样式 `<link>` 已无 `crossorigin`，`integrity` 保留。
- 生产 `/_nuxt/**` 的头由 Nitro routeRules 提供（dev 由 Vite 直出，看不到该头）。

## Next

- 部署后复测：`curl -sSI https://站点/_nuxt/<entry>.js` 与任一
  `/_sforum/assets/**`，确认两枚头都在；再在 QQ 内置浏览器开/关云加速各测一次。
- 若线上仍出现 500 文档，用当日 `status=500` 访问日志按 UA 定位来源（错误文档已
  `cache-control: no-store`，X5 仍重放则属代理缓存，换 URL 即可破）。

## Open Questions

- 云加速是否会对 CSS/JS 做内容改写：若改写，`integrity` 在同源场景外的失败需再观察
  一轮日志（当前策略下跨源会跳过 SRI，不应再出现整块样式丢失）。

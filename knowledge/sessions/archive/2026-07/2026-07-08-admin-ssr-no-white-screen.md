# 2026-07-08 Admin Pages SSR (No White Screen)

## Changed

- 移除 `apps/web/nuxt.config.ts` 中最后的两处 `ssr: false` 路由配置：
  - `[adminRoutePrefix]/**`：从 `ssr: false` 改为 `cache: false`，后台全部
    21 个页面改为服务端渲染。
  - `/components`：从 `ssr: false` 改为仅保留 `robots: { index: false }`，
    生产环境直接服务端渲染 404 错误页，不再先返回空壳。
- 更新 routeRules 顶部注释，明确"全部页面保持 SSR 彻底避免空壳白屏"。
- 新增 `apps/web/tests/adminRouteRendering.test.ts`：断言全站不再有任何
  `ssr: false` 路由配置，admin 路由带 `cache: false`。

## Decisions

- **全站 SSR，杜绝空壳白屏**：白屏根因是本该 SSR 或服务端重定向的页面被当成
  SPA 空壳发给浏览器，浏览器必须等客户端 JS 下载执行挂载后才有内容。登录页、
  注册页、受保护用户页此前已修复；本次把最后两片 `ssr: false`（后台 + 组件预览）
  也改为 SSR，全站再无 SPA 壳页面。
- **admin 全量 SSR 安全**：审计确认 admin 全部页面 + 布局 + 组件都是 SSR-safe
  的——所有浏览器 API（`window`/`document`/`navigator`/`localStorage`）访问都
  正确限定在 `onMounted`/`onUnmounted`、事件处理函数、或 `import.meta.client`
  守卫内。主题切换按钮已用 `<ClientOnly>` 包裹，Tiptap 编辑器实例化在 `onMounted`
  且 `<EditorContent>` 已用 `<ClientOnly>` 包裹。
- **admin 中间件 SSR 行为正确**：未登录→服务端 302 重定向 `/login`（无白屏）；
  无权限→重定向 `/`；认证服务不可用→渲染 503 错误页（而非空壳）。
- **app.vue 启动逻辑保持不动**：服务端 `Promise.all` 并行获取 web-options +
  auth-session 是当前架构下的最优解。把 auth 拆成 lazy 会让受保护页面变成串行
  等待（web-options → middleware auth），反而更慢；公开页面也几乎无收益
  （web-options 仍是瓶颈）。
- **`/components` 不再单独 SSR 关闭缓存**：它生产环境直接 404，dev 下是组件
  预览，SWR/缓存对其无实际影响，保留 `robots: { index: false }` 即可。

## Verification

- `bun test tests/adminRouteRendering.test.ts tests/authRouteRendering.test.ts
  tests/protectedRouteRendering.test.ts tests/appStartup.test.ts` → 10 pass。
- `bun run typecheck` → EXIT=0。
- `bun run build` → EXIT=0，Build complete。
- 待人工浏览器 QA：未登录访问 `/admin` 应直接服务端 302 到 `/login`（无白屏）；
  登录后访问 `/admin` 应直接出首屏 HTML（无空壳）。

## Next

- 浏览器 QA 确认 admin 首屏 HTML 正常、无 hydration mismatch 警告。
- 若后台 SSR 后首屏因 `/admin/extensions/navigation` 等数据请求变慢，可考虑给
  这些非关键数据加 `{ lazy: true }`（当前已 `default: () => []`，不阻塞渲染）。

## Open Questions

- None.

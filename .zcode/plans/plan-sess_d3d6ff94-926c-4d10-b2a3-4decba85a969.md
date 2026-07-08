# 彻底杜绝白屏：移除全站最后的 ssr:false 路由

## 问题根因

全站仅剩两个 `ssr: false` 路由，它们返回空 `#__nuxt` SPA 壳，浏览器必须等客户端 JS 下载执行后才有内容——这就是你看到的白屏：

| 路由 | 当前配置 | 影响范围 |
|---|---|---|
| `[adminRoutePrefix]/**` | `ssr: false` | 全部 21 个后台页面（overview、users、roles、permissions、forum、moderation、extensions、settings、seo、database、attachments、personalization…） |
| `/components` | `ssr: false` | 组件预览页（生产环境 404，但也先返回空壳再 404） |

其余所有页面（公开内容页、认证页、受保护用户页）已经正确 SSR。

## 方案：移除 ssr:false，让全部页面 SSR

经过 SSR 安全审计，admin 全部页面 + 布局 + 组件都是 SSR-safe 的（浏览器 API 全部限定在 `onMounted`/事件处理函数/`import.meta.client` 守卫内）。admin 中间件在 SSR 下行为正确（未登录→服务端 302 重定向 /login，无权限→重定向 /，认证服务不可用→渲染 503 错误页）。可以直接去掉 `ssr: false`。

### 改动 1：`apps/web/nuxt.config.ts` — routeRules

```diff
-      // 管理后台：SPA + 禁止索引。
-      [`${adminRoutePrefix}/**`]: { ssr: false, robots: { index: false } },
-      '/components': { ssr: false, robots: { index: false } },
+      // 管理后台：SSR + 禁缓存 + 禁止索引。未登录由 admin 中间件服务端重定向到 /login，不再返回空壳。
+      [`${adminRoutePrefix}/**`]: { cache: false, robots: { index: false } },
+      // 组件预览页：SSR（生产环境直接渲染 404 错误页，不再先返回空壳）。
+      '/components': { robots: { index: false } },
```

同时更新文件顶部的 routeRules 注释：
```diff
-    // 路由级渲染模式与缓存：公开内容页走 stale-while-revalidate，登录/注册保持 SSR 避免空壳白屏。
+    // 路由级渲染模式与缓存：公开内容页走 stale-while-revalidate，全部页面保持 SSR 彻底避免空壳白屏。
```

### 改动 2：测试 — 断言全站无 ssr:false

新增 `apps/web/tests/adminRouteRendering.test.ts`，断言全站不再有任何 `ssr: false` 路由配置，并验证 admin 路由有 `cache: false`：

```ts
import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

describe('admin route rendering', () => {
  test('does not configure any route as SPA-only to avoid empty-shell white screens', () => {
    const config = readFileSync(new URL('../nuxt.config.ts', import.meta.url), 'utf8')
    // 全站不应有任何 ssr: false 路由——所有页面都 SSR，彻底杜绝空壳白屏。
    expect(config).not.toMatch(/['"`][^'"`]+['"`]\s*:\s*\{[^}]*\bssr\s*:\s*false/)
  })

  test('disables route cache for admin pages', () => {
    const config = readFileSync(new URL('../nuxt.config.ts', import.meta.url), 'utf8')
    // admin 路由是动态 key（模板字符串），验证其配置块包含 cache: false。
    expect(config).toMatch(/adminRoutePrefix.*\*\*.*\{[^}]*cache\s*:\s*false/)
  })
})
```

### 改动 3：知识库

更新 `knowledge/sessions/` 新增 session handoff，记录"全站 SSR 化、移除最后的 ssr:false"决策。

## 不改动的部分（已确认无需调整）

- **`app.vue` 启动逻辑**：服务端 `Promise.all` 并行获取 web-options + auth-session 是最优解。拆分 auth 为 lazy 会让受保护页面变成串行（web-options → middleware auth 顺序等待），反而更慢。公开页面也几乎无收益（web-options 仍是瓶颈）。
- **admin 中间件**（`app/middleware/admin.ts`）：SSR 下行为正确，无需修改。
- **全局 auth 中间件**（`app/middleware/auth.global.ts`）：已正确处理 `requiresAuth`。
- **admin 布局**（`app/layouts/admin.vue`）：已用 `<ClientOnly>` 包裹主题切换按钮，SSR-safe。

## 验证

- `cd apps/web && bun test tests/adminRouteRendering.test.ts tests/authRouteRendering.test.ts tests/protectedRouteRendering.test.ts tests/appStartup.test.ts`
- `cd apps/web && bun run typecheck`
- `cd apps/web && bun run build`
- 浏览器 QA：未登录访问 `/admin` 应直接服务端 302 到 `/login`（无白屏）；登录后访问 `/admin` 应直接出首屏 HTML（无空壳）。
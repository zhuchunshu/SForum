# 2026-07-08 冷启动竞态导致登录页白屏与表单 502 修复

## 症状

用户报告：网站刷新偶尔白屏，尤其是登录页；提交表单也返回 502。触发场景为
"启动后第一次访问"。

## 根因

`apps/web/scripts/dev-theme-runtime.mjs`（开发）和 `apps/web/scripts/runtime.mjs`
（生产）的 `main()` 启动时序有缺陷：

```js
async function main() {
  await proxy.listen()          // (1) supervisor 先对外监听端口
  // ...
  await startDev('startup')     // (2) 这时才开始 spawn nuxt dev 子进程
}
```

supervisor 在 `proxy.listen()` 完成后就占用对外端口，但 `activeTarget` 仍是
`null`（nuxt dev / Nitro 子进程还没 spawn / 没通过健康检查）。从 `proxy.listen()`
到 `proxy.setTarget()` 之间这段冷启动窗口（开发最长 120s、生产 30s），所有入站
请求走 `theme-proxy.mjs:30-35` 的 `activeTarget === null` 分支，直接返回 502
"SForum web is starting up"。

这解释了两个症状：

1. **登录页白屏**：`/login` 是 `ssr: false`（纯 SPA），HTML 只有空
   `<div id="__nuxt">`，靠浏览器加载 `/_nuxt/.../entry.async.js` 挂载。冷启动
   窗口里该 `<script>` 请求拿到 502 → JS 加载失败 → Vue app 没机会挂载 → 白屏。
   SPA 页面对此最敏感，所以"尤其是登录页"。SSR 页面（首页）即使 JS 没加载，SSR
   渲染的 HTML 也能显示部分内容，不会完全白屏。
2. **提交表单 502**：`/api/v1/auth/login` 同样经过 supervisor（3000）→ 502。

蓝绿切换（主题切换）场景有旧子进程兜底，不受影响；问题只在初次冷启动。

## 修复

调换 `main()` 启动顺序：先等子进程 ready（`startXxx` 通过健康检查、
`proxy.setTarget` 被调用），再 `proxy.listen()` 对外监听。子进程 ready 前不占用
对外端口，浏览器访问得到 ECONNREFUSED（清晰的"服务未起"信号），而不是误导性的
502 + 空 HTML 白屏。

改动文件：
- `apps/web/scripts/dev-theme-runtime.mjs`：`main()` 里 `proxy.listen()` 移到
  `startDev()` 之后；信号处理移到 `startDev` 之前；加 `proxy.getTarget()`
  检查，初次启动失败时报错退出。
- `apps/web/scripts/runtime.mjs`（生产）：同样的调换。

不改动的部分：
- `theme-proxy.mjs` 不改：`activeTarget === null` 返回 502 是合理的运行期兜底
  （upstream 崩溃、崩溃恢复窗口）。冷启动竞态通过 listen 时序解决，不给代理核心
  加"请求排队/等待"逻辑——那会让浏览器请求挂起最长 120s，体验比 ECONNREFUSED
  更差。
- `server/routes/api/v1/[...path].ts` 不改：它的 502 是后端 Go API（air 重启）
  不可用时 Nitro 的行为，是独立的另一层，`useAuthSession` 已有 `unavailable`
  状态容错。
- 崩溃恢复逻辑不改：`child.on('exit', () => setTimeout(() => startDev(...), 1000))`
  保持原样。崩溃恢复期间会有短暂 502 窗口，但这是罕见情况，且那时 supervisor 已
  listen，无法再"先 ready 后 listen"。

## 验证

修复后重启 supervisor 并立即连续探测 3000 端口：

- 前 11 次（约 22 秒）：`HTTP 000` = ECONNREFUSED。supervisor 还没 listen，
  nuxt dev 在冷启动，端口不占用。
- 第 12 次（约 24 秒，nuxt dev ready）：`HTTP 200`。supervisor 此时才 listen。

supervisor 日志确认时序正确：
```
(startup) switched nuxt dev; public URL: ...    ← 子进程 ready（setTarget 完成）
proxy listening on 0.0.0.0:3000                 ← 然后才对外监听
public URL: ...                                 ← 此时才真正可服务
```

功能回归：`/login` HTML 200、`entry.async.js` 200、`@vite/client` 200、
`POST /api/v1/auth/login` 返回 401（凭据错误，符合预期，不再是 502）。

## Next

- 后端 `air` 重启窗口仍会让 API 代理短暂返回 502（`server/routes/api/v1/[...path].ts`
  → Go API 不可用）。这是独立的另一层，`useAuthSession` 已有 `unavailable` 容错，
  暂不处理。如需进一步优化，可在该代理路由里对连接失败做有限次重试。

## Open Questions

- 是否需要给 `theme-proxy.mjs` 的 502 响应加 `Retry-After` header，引导浏览器
  在崩溃恢复窗口自动重试？当前未加，ECONNREFUSED 场景浏览器自身会快速重试。

# 2026-07-08 Session Handoff

## Context

主题切换的蓝绿零停机 supervisor 实现已存在于工作区（HEAD 里 `theme-proxy.mjs`、
`runtime.mjs`、`dev-theme-runtime.mjs` 都已是蓝绿版本），但运行时存在致命 bug
导致代理转发 hang、unix socket 连接失败、切换后 upstream 丢失。本次会话的
实际工作是**调试并修复这些 bug**，让既存的蓝绿机制真正可用。

## Changed

- **修复 `apps/web/scripts/theme-proxy.mjs` 的 3 个致命 bug**（这是本次唯一的功能改动文件）：
  1. **`forwardRequest` / `probeUnix` 的 unix socket 转发不可靠**：原实现用
     `http.request({ createConnection: () => net.connect(socketPath) })`，实测在 unix
     socket 场景必 ECONNREFUSED（无论是否 `agent:false`）。改用 Node 原生 `socketPath`
     选项（`http.request({ socketPath })`）后正常。TCP 转发保持 `host`/`port`。
  2. **`forwardRequest` 客户端断开探测导致 hang**：原实现 `req.on('close', ...)`
     会把刚建好的上游连接毁掉——因为 keep-alive 的 GET 请求体读完即可读流立刻
     结束，触发 `close`，于是 `proxyReq.destroy()` 把响应还没回来的上游连接毁了，
     表现为请求 hang 直到超时。改为 `res.on('close')` 且只在 `!res.writableEnded`
     时才 abort 上游。
  3. **`replaceTarget` 没把 `candidate` 传给 `healthCheck`**：原实现
     `await withTimeout(healthCheck(), ...)`，而 `healthCheckTcp/Unix` 的契约是
     `healthCheck(candidate)`，负责把候选实际监听地址写到 `candidate._target`，
     切换成功后 `proxy.setTarget` 据此切换。没传 candidate → `candidate._target`
     永远 undefined → `proxy.setTarget(undefined)` → 切换后 upstream 变 null →
     后续请求全 502。改为 `healthCheck(candidate)`。
  4. 附带清理：移除修复后不再使用的 `import net from 'node:net'` 与
     `import { once } from 'node:events'` 及其 `export { once }`。
- 新增 `apps/web/tests/themeProxy.test.ts`（14 个用例，全部通过）：覆盖端口解析、
  GET/POST/header/状态码透传、X-Forwarded-For 链、无 upstream 返回 502、蓝绿切换
  成功路径、候选健康检查失败保留旧上游、TCP/unix socket 健康检查。这些测试
  正是用来钉死上面 3 个 bug 的回归保护。
- 更新 `knowledge/modules/extensions.md`：补充主题切换零停机（蓝绿）架构说明
  （此前文档只描述了"supervisor restart"，未说明蓝绿代理机制）。

## Decisions

- **unix socket 代理用 Node 原生 `socketPath` 选项**，而非 `createConnection` 选项。
  这是 bug #1 的修复方式，也是最重要的决策——实测 `createConnection` 在 unix
  socket 转发场景不可靠（ECONNREFUSED），`http.request({ socketPath })` 才对。
- **客户端断开探测改监听 `res.close`**，而非 `req.close`。这是 bug #2 的根因。
- **不重写既存的蓝绿架构**：HEAD 里的 `runtime.mjs` / `dev-theme-runtime.mjs` /
  `theme-proxy.mjs` 架构设计是对的（蓝绿、unix socket 生产 / TCP PORT=0 开发、
  健康检查、候选失败保留旧 child），只是 `theme-proxy.mjs` 实现有 3 个 bug。
  修复 + 测试覆盖即可，不推倒重来。

## 架构背景（既存，供后续参考）

主题切换零停机的既存设计：
- `theme-proxy.mjs`：supervisor 共用核心。`createThemeProxy` 持有对外端口（PORT，
  默认 3000）的 http.Server 与当前 upstream。`replaceTarget` 是蓝绿核心：起候选 →
  健康检查 → 切 upstream → drain 旧进程。候选失败永远保留旧 child。
- `runtime.mjs`（生产）：supervisor 监听 3000，子进程（Nitro server）监听每次唯一
  的 unix socket（`NITRO_UNIX_SOCKET`）。Nitro `node-server` 不支持 `PORT=0`
  （`destr("0")||3000` 落到 3000），但原生支持 `NITRO_UNIX_SOCKET`。
- `dev-theme-runtime.mjs`（开发）：supervisor 监听 3000，`bun run dev:plain`
  子进程监听 `PORT=0` 临时端口，supervisor 从 stdout 解析 `➜ Local:` 行拿端口
  （`parseDevPort`，兼容 IPv6 与 `0.0.0.0` 归一为回环）。无需 `waitForChildExit`。

## Next

- 手动冒烟（开发）：`cd apps/web && bun run dev` → 后台激活上传主题（改
  `storage/theme-releases/current.json`）→ 观察切换期间 `curl` 持续打 3000
  **无 502/无空白**，旧主题服务到新主题 ready。
- 手动冒烟（生产/Docker）：`bun run theme:runtime` → 模拟 `current.json` 切换 →
  同样验证无停机。
- 切换期间 SSE/长连接会被中断（SIGTERM 即关）。若后续要支持真正的长连接
  drain，需在 `replaceTarget` 的 `stopChild` 前加 drain timeout。

## Open Questions

- 开发环境端口解析耦合 listhen 的 stdout 格式（`➜ Local: http://...:port/`）。
  格式自 Nitro 早期稳定，且匹配失败回退为「保留旧 child」安全态；但若未来
  listhen 改了输出，开发切换会停在旧主题（生产用 unix socket 完全不受影响）。
- 生产 `fs.watch` 在某些 Docker volume / NFS 场景对 rename 可能不可靠（见
  上一个 handoff 的 Open Questions），未加轮询兜底。

## 测试

- `cd apps/web && bun test tests/themeProxy.test.ts` → 14 pass / 0 fail。
- `apps/web` 全量 `bun test` → 71 pass / 3 fail，其中 3 个失败
  （`useApiClient.test.ts` 的 register/login 用例）是工作区里**别人未提交的
  password_reset 后端改动**导致 `register.vue` 文件路径不匹配（ENOENT），
  与本次 supervisor 改动无关，且 `useApiClient.test.ts` 未被本次改动触及。

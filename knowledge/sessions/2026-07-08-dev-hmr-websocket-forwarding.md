# 2026-07-08 修复 Dev Server HMR Reload（supervisor 代理补 WebSocket 转发）

## Changed

- `apps/web/scripts/theme-proxy.mjs`：`createThemeProxy` 的 HTTP server 现在注册
  `server.on('upgrade', ...)`，新增模块私有函数 `forwardUpgrade`，把 WebSocket
  升级请求以原始 socket 隧道透传到当前 activeTarget（同时覆盖 TCP 与 unix socket
  两种 upstream）。`activeTarget===null` 时直接销毁客户端 socket。
- `apps/web/package.json`：`dev` 脚本由 `bun --env-file=../../.env scripts/dev-theme-runtime.mjs`
  改为 `node --env-file=../../.env scripts/dev-theme-runtime.mjs`。supervisor 改用
  node 运行，避开 bun 的 upgrade 缺陷（见 Decisions）。
- `apps/web/tests/themeProxy.test.ts`：新增 `WebSocket upgrade 转发（需 node 运行时）`
  测试块，覆盖 TCP upstream、activeTarget 未就绪、unix socket upstream 三种路径。
  该块在 bun 运行时下 `describe.skip`，仅由 node 覆盖。

## Decisions

- **根因 A**：共享代理模块 `theme-proxy.mjs` 的 `createThemeProxy` 只用
  `http.createServer` 处理 HTTP 请求，从未注册 `server.on('upgrade', ...)`，
  导致 Nuxt/Vite HMR 的 WebSocket 握手被直接丢弃。这是「改文件不热更新」的直接原因。
- **根因 B（bun 运行时缺陷）**：即使补上 upgrade handler，bun 的 `node:http` 兼容层
  在 `'upgrade'` 事件里 `socket.write()` 会静默丢数据，客户端永远收不到 101 Switching
  Protocols（bun#28157 / bun#9882，修复 PR bun#27237 仍在进行）。supervisor 进程
  原本由 `bun scripts/dev-theme-runtime.mjs` 运行，因此 HMR 在 bun 下仍不通。
- **选择 supervisor 改用 node 跑**（而非 dev 改走 dev:plain）：
  - node 的 `http.Server` upgrade 路径完全正常（已用 node 独立验证 forwardUpgrade
    能让客户端收到 101 + upstream 回声，并透传 X-Forwarded-For）。
  - 保留主题感知 supervisor 的全部能力（后台激活上传主题 → current.json 变化 →
    蓝绿重启 nuxt dev），不破坏现有架构与文档。
  - 改动最小：只换 supervisor 进程的运行时，子进程（`bun run dev:plain` → nuxt dev）
    与代理逻辑都不变。`dev-theme-runtime.mjs` / `theme-proxy.mjs` 全部使用 `node:`
    内置模块，无 bun 专属 API。
- WS 测试在 bun 下 skip：测试用 `bun:test` 跑，但 bun 本身踩该缺陷，无法验证 upgrade
  透传；改用 `node` 运行测试时才执行。运行时检测 `globalThis.Bun` 决定是否 skip。

## Verification

- `cd apps/web && bun test`：101 pass / 3 skip / 0 fail（WS 三例在 bun 下 skip）。
- `node /tmp/node-ws-verify.mjs`（导入真实 theme-proxy.mjs）：node 下客户端收到
  `HTTP/1.1 101 Switching Protocols` + marker，upstream 收到补全的
  `x-forwarded-for: 127.0.0.1`，upgrade 头透传。**PASS**。
- `node tests/validate-theme-runtime.js`：`Theme runtime validation passed.`（dev
  脚本断言 `dev-theme-runtime.mjs` 子串仍成立）。
- `node --check` 两个 supervisor 脚本：语法 OK。

## Next

- 手动冒烟：`cd apps/web && bun run dev`（内部用 node 跑 supervisor），浏览器
  devtools → Network → WS 面板应能看到 `__vite_ping` / HMR 的 WS 连接成功建立
  （101 Switching Protocols）；改一个 `.vue` 文件应触发热更新，无需手动刷新。
- 关注 bun#27237 合并后，可考虑把 supervisor 改回 bun 运行，并移除 WS 测试的
  bun-skip 分支。

## Open Questions

- 生产 `runtime.mjs` 也共享 `theme-proxy.mjs`，生产 web 进程现在同样具备 WS 转发
  能力。是否已有 app 级 WebSocket 功能依赖它，待确认；目前主要是给 Nitro/Nuxt
  app 未来可能引入的实时能力（如通知推送）预留通路。

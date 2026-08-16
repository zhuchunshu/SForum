# 2026-08-16 Extension authoring docs handoff

## Changed

- 新增 `docs/extensions/routes.md`（412 行）：插件声明式 HTTP 路由作者指南。
  覆盖 manifest `routes[]` 逐字段语义（schema 为权威）、action 全集
  （add/alias/redirect/rewrite/before/after/filter/wrap/replace/
  global_middleware）、guard 语义（public/login/permission/guest 可用；
  inherit/raw_request/自定义 guard 当前 release 关闭）、Protocol V2
  `InvokeRoute`/`RouteResponse` 字段表、流式模式（WithRuntimeStreams +
  RouteStream）、入口 probe（`X-SForum-Internal-Route-Probe`）与 Nuxt
  plugin-route-proxy 行为、CSRF/PAT、8 MiB 响应上限与 3 s 默认 RPC 超时、
  测试与清单。锚定 fixture `sforum-custom-content`。
- 新增 `docs/extensions/build-and-load.md`：自构建/加载循环——make:plugin
  脚手架产出、backend go.mod 的 `replace github.com/zhuchunshu/sforum/apps/api
  => <checkout>/apps/api` 接线（此前零文档）、`go build -o plugin .`、
  前端 prebuilt 资产（宿主不编译，fixtures 无 package-local build）、
  digest/validate/test/package、dev 加载三条路（EXTERNAL_EXTENSION_ROOTS /
  上传 zip / builtin staging）、信任确认与迭代循环（改包即失效）、
  内置包 build-builtin-plugins.sh 只构建硬编码 id、排障表。
- **中英双语**：两个新页均有完整中文翻译
  `docs/zh-CN/extensions/{routes,build-and-load}.md`；英文原文保留在
  `docs/extensions/`（路径稳定）；`docs/en-US/extensions/` 放指回英文原文的
  简短入口页以满足 `validate-docs.mjs` 的 zh/en 结构平行检查。zh-CN/en-US
  README 与技术参考章节、authoring-guide 顶部横幅、zh-CN 开发 README 均加
  交叉链接。
- `authoring-guide.md`：顶部链接两个新页；场景表路由行改为
  `InvokeRoute`/`RouteTarget` + 链接；新增 Reference 4（custom-content 路由
  参考，原 Reference 4→5）；打包命令节链接 build-and-load。
- `docs/zh-CN/development/README.md` 与 `docs/en-US/development/README.md`：
  新增"扩展开发入口"（zh 指向中文页，en 指向英文页，保持双语平行）。

## Decisions

- 技术文档继续全英文放在 `docs/extensions/`（路径稳定、机器校验）；双语
  手册只加入口链接，不复制内容。
- 文档中"可用 guard"以当前 release 的 `HostRouteGuardAuthorizer` 行为为准：
  custom/raw guard 已声明族但执行运行时未落地，标注为 closed。

## Next

- 无阻塞。若后续 wave 落地 guard 执行运行时，需同步更新 routes.md Guards 表。
- `routes.md` 示例代码未编译验证（伪包 acme.notes）；如需严格可编译样例，
  以 fixture `sforum-custom-content/backend/routes.go` 为权威。

## Open Questions

- 无。

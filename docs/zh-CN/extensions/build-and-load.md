# 构建、摘要与加载你的扩展（Build, digest, and load）

> **语言切换：** [English (canonical)](../../extensions/build-and-load.md) · 本文为中文翻译。
> 命令、字段名与错误码以英文原文和代码为准；中文版若与代码不一致，以代码为准。

本页讲述带可执行后端（以及可选前端资产）的插件的完整作者循环：脚手架、
Go 模块接线、构建二进制与前端产物、刷新精确摘要、打包，以及把包加载到
运行中的 SForum 实例。

配套文档：[插件路由（中文版）](./routes.md) · [插件编写指南（英文）](../../extensions/authoring-guide.md)

## 已构建包的样子

```text
my-plugin/
├── sforum.extension.json      # Manifest V3
├── README.md
├── backend/
│   ├── go.mod                 # 你的 Go 模块（见下文）
│   ├── go.sum
│   ├── *.go                   # Protocol V2 server（pluginv2.Serve）
│   └── plugin                 # 构建出的可执行文件（backend.entry）
├── frontend/
│   └── admin/dist/            # 作者预构建的 ESM/CSS（仅在需要时）
└── schemas/
    └── *.json                 # packageFiles kind "schema"（路由文档等）
```

`make:plugin` 会生成 manifest、README、占位的 `backend/plugin` stub，以及
（带 `--backend` 时）说明最小 SDK 程序的 `backend/README.md`。脚手架**不会**
生成 Go 模块——`go.mod` 需要你自己创建。

```bash
cd apps/api
go run ./cmd/sforum make:plugin \
  --id acme.notes --name "Acme Notes" --description "…" \
  --url https://example.com/acme-notes --author-name Acme \
  --backend --no-interaction --out /tmp/acme.notes
```

本地实验时脚手架默认落到 `extensions/dev/`（gitignored，永不被自动注册）。
需要可丢弃路径用 `--out`，需要受保护内置包用 `--builtin`（落在
`extensions/builtin/`）。

## 1. 接好后端 Go 模块

后端 import 公共 SDK，而 SDK 位于宿主模块
`github.com/zhuchunshu/sforum/apps/api` 内。该模块没有发布为带版本号的依赖，
因此仓库内每个包都用 `replace` 指令指向本地 checkout。工具链版本以宿主
`go.mod` 为准（当前 Go 1.26.6）。

`backend/go.mod`（仓库内包）：

```go
module github.com/zhuchunshu/sforum/extensions/builtin/plugins/sforum-smtp/backend

go 1.26.6

require github.com/zhuchunshu/sforum/apps/api v0.0.0-00010101000000-000000000000

// 指向包含 apps/api 的仓库 checkout。
replace github.com/zhuchunshu/sforum/apps/api => ../../../../../apps/api
```

- 仓库内：相对 `replace` 路径必须从你的 `backend/` 目录数出足够层级的 `..`
  到达仓库根的 `apps/api`。
- 仓库外：把 SForum clone 到任意位置，让 `replace` 指向
  `<你的-checkout>/apps/api`（绝对或相对路径均可）。宿主模块很大，
  `go mod tidy`/`go build` 会下载大量传递依赖——中国大陆网络先设置本地
  代理（见 `AGENTS.md`）。
- require 行的版本只是占位（仓库内实际使用零版本号，被 `replace` 覆盖），
  解析完全由 `replace` 决定，可写任意合法版本号。
- 仓库外插件的 `module` 路径任选，但请保持稳定。

最小 `main.go`：

```go
package main

import pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"

func main() {
	pluginv2.Serve(pluginv2.NewServer())
}
```

按需在 server 类型上添加 route、hook、provider、query、job 或 notification
处理器（见 [插件路由（中文版）](./routes.md) 与
[插件编写指南（英文）](../../extensions/authoring-guide.md)）。

```bash
cd <package-root>/backend
go mod tidy
go build -trimpath -buildvcs=false -ldflags="-s -w" -o plugin .
```

可执行文件必须落在 `backend.entry` 声明的路径（惯例是 `backend/plugin`）。
二进制缺失时 `extension test` 会失败（脚手架阶段可用
`--skip-backend-binary` / `--allow-scaffold`）。

## 2. 构建前端资产（仅在需要时）

宿主**从不**编译上传的源码。运行时挂载的一切——admin 预构建组件、公共 L2
模块、editor L2 节点——必须是**最终、自包含的 ESM/CSS 字节**，按声明路径
提交进包：

- `--prebuilt-settings` 会生成可用的 `frontend/admin/dist/settings.mjs` +
  `.css`（参考 `extensions/fixtures/plugins/sforum-prebuilt-settings/`）。
- fixture 的 L2 模块（如 `sforum-custom-content/frontend/editor/vote.mjs`）
  是手写的单文件 ES module。
- 你可以用任意打包器（esbuild、rolldown、vite 等）产出最终单文件 `.mjs`；
  包只携带输出。没有 package-local 构建步骤，也不要求 `package.json`——
  fixture README 有意记录了这一点。

每个产物都要在 `packageFiles` 中声明：`kind: "frontend"`（或按需 `asset` /
`schema` / `template`）+ 精确路径。所有摘要由 `extension digest --write`
刷新。

## 3. 刷新精确摘要

manifest 引用的每个打包文件——后端可执行文件、前端资产、schemas、模板——
都以 SHA-256 绑定在 `packageFiles` 中。任何文件变更后：

```bash
cd apps/api
go run ./cmd/sforum extension digest --write <package-root>
```

这会重写内联摘要（包括内联模板摘要）并重新校验整个包。Page Registry 主题
需要**先**补好对应的 `templates[]` 声明与 `packageFiles[]` 条目再运行它；
digest 刷新绝不会从 `theme.json` 推断模板身份或成员关系。

## 4. 校验与契约测试

```bash
go run ./cmd/sforum extension validate <package-root>            # schema + 预检
go run ./cmd/sforum extension test <package-root>                # host 契约检查
go run ./cmd/sforum extension test --json <package-root>
```

后端二进制还没构建时，`extension test --allow-scaffold`（同
`--skip-backend-binary`）可让契约检查保持通过。

## 5. 打包分发

```bash
go run ./cmd/sforum extension package <package-root> --exclude-source \
  -o /tmp/my-plugin.sforum.zip
```

`--exclude-source` 会剔除 Go/JS/TS 源码、进行中的 manifest、`testdata`
等非运行文件；zip 保留可执行文件、预构建前端、schemas 与 README。输出为
`<name>.sforum.zip` + 旁路 `.sbom.json`。默认（不带 flag）打包除
`.git`/`node_modules`/`vendor` 与既有 zip 外的几乎所有内容。

## 6. 加载到运行中的实例

| 方式 | 做法 | 后台列表 |
| --- | --- | --- |
| 外部源码集合 | 把包放到 `<root>/plugins/<id>/`（主题为 `<root>/themes/<id>/`），在 `.env` 设置 `EXTERNAL_EXTENSION_ROOTS=<root>`，重启 API | 自动出现快照；仍需信任 + 启用 |
| 上传 | 后台 → 扩展 → 插件 → 安装 zip | 已安装；信任 + 启用 |
| 内置（仓库内包） | `make:plugin --builtin`，重启 API 让 `SyncBuiltins` 注册 | 自动 |
| `extensions/dev/` | 永不扫描——用上面任一方式 | — |

`extensions/dev/` 只是临时草稿区。日常本地迭代推荐**外部源码集合**：

```sh
# .env
EXTERNAL_EXTENSION_ROOTS=/abs/path/to/sforum-plugins
```

```text
/abs/path/to/sforum-plugins/
  plugins/acme.notes/          # 你的包源码
```

- API/worker 启动时扫描每个根，把合法包以**不可变快照**复制进
  `EXTENSION_ROOT`。内容变化只产生**待审核版本**；永远不会自动提升或继承
  信任。
- 包变更后重启 API，让扫描拾取新快照。

### 启用与信任

1. 后台 → 扩展 → 插件 → 打开你的插件。
2. **启用**会为可执行内容触发信任流程（后端二进制、迁移、guard、L2 前端、
   声明的路由注册表贡献）。`super_admin` 基于精确制品（版本 + 摘要）确认。
   影响文档会列出已声明的权力、依赖与 Host/Frontend 契约版本。
3. 启用后，声明的路由、hook 与设置生效。用
   `curl http://127.0.0.1:8080/<path>` 或 Web 同源冒烟测试路由。

### 迭代循环

```text
改源码 → 构建后端/前端 → extension digest --write
→ 重启 API → 后台重新启用（制品变化则需重新信任）
```

任何相关包变更都会使既有信任授予失效（可执行制品；按 F2.4，摘要变化也会
吊销前端信任）。每次迭代改了打包字节，都要准备再点一次信任确认。若某个插件
导致 API 无法启动，带外恢复：

```bash
go run ./cmd/sforum extension disable <extension-id>
go run ./cmd/sforum extension disable-all
go run ./cmd/sforum extension list --json
```

## 内置包（仓库内）

`extensions/builtin/` 由 `SyncBuiltins` 启动扫描。`api-dev.sh` /
`worker-dev.sh` 把它暂存到 `storage/builtin-dev` 并运行
`scripts/build-builtin-plugins.sh`，该脚本**只构建硬编码在脚本里的 id**，
并在暂存树里刷新摘要（绝不改写被 git 跟踪的 manifest）。

新内置包在存在于活动内置根之后，重启即被 `SyncBuiltins` 注册，但它的后端
**不会**自动构建——直到你把 id 加进 `scripts/build-builtin-plugins.sh`
（或自己构建 + `extension digest --write`）。手动构建内置后端：

```bash
(cd extensions/builtin/plugins/<id>/backend && go test ./... && go build -o plugin .)
```

不要直接改 `storage/builtin-dev/` 或 `storage/extensions/**`；把它们当作
生成状态。

## 排障

| 现象 | 原因 / 处理 |
| --- | --- |
| `extension test` 后端失败 | `backend.entry` 处缺二进制；构建它，或脚手架阶段用 `--skip-backend-binary` |
| 启用返回 `extension.build_failed` | 某声明文件摘要或模板身份过期；运行 `extension digest --write` 并重启 |
| 路由 404 / 探测报 `miss` | 路径未被任何已启用插件路由认领；查后台路由检查器与冲突页 |
| 不安全路由 CSRF 403 | 浏览器调用缺 `X-Csrf-Token`（与 `csrf_` cookie double-submit）；用应用的 `useApiClient` 或 Bearer PAT |
| 路由超时 | 处理器超过路由 `timeoutMs`（默认 3 秒）；把大工作流化或调高声明 |
| 外部根变更不可见 | 快照是惰性的、启动时扫描；重启 API 并重新启用 |
| 每次迭代都弹信任确认 | 预期行为——打包字节变化会使授予失效 |
| 坏插件导致 API 无法启动 | 带外 CLI：`extension disable-all` / `quarantine`，再选择性启用 |

## 相关

- [插件路由（中文版）](./routes.md) · [英文原文](../../extensions/routes.md)
- [插件编写指南（英文）](../../extensions/authoring-guide.md) — 契约、设置、参考包
- [Manifest V3 目录（生成）](../../extensions/catalogs/manifest-v3.md)
- [`extensions/README.md`](../../../extensions/README.md) — 包目录地图
- CLI 参考：[开发者 CLI（中文）](../development/cli.md) · [Developer CLI（英文）](../../en-US/development/cli.md)

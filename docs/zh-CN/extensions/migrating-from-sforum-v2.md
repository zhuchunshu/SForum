# 从 SForum-old 迁移到 V3 契约

本文把 SForum-old 的作者模式（Controller、Middleware、Blade、Session、Itf 钩子、
插件辅助库）映射到**今天真实存在**的 Manifest V3 契约，列出**有意关闭**的操作，并给出
在干净临时目录里可跑通的复制粘贴快速开始。

下表的每一行都对应代码里存在的契约。SForum-old 有、而 V3 没有实现的模式，会直接写明
「不可用」与原因，而不是指向一个实际跑不通的替代品。

相关文档：[作者指南](../../extensions/authoring-guide.md)、[路由](../../extensions/routes.md)、
[运行时主题](../../extensions/runtime-themes.md)、[扩展目录约定](../../../extensions/README.md)。

## 迁移对照表

| SForum-old 模式 | V3 对应契约 | 作者要改什么 |
| --- | --- | --- |
| Controller 类 + 方法 | `routes[]` 条目（`action: add\|replace`、`path`、`methods`、`access`、`permission`、`guard`、`handler`、`requestSchema`、`responseSchema`、`timeoutMs`） | 处理器是声明的后端入口，经 Protocol V2 用**类型化文档**调用；它拿不到框架请求对象。`replace` 需要精确制品信任，且目标路由必须已冻结。 |
| 路由中间件（before/after） | `action: "before"` / `"after"` 的路由条目 + `guard: "core.guard.inherit"` + 显式 `mutableRequestFields` / `mutableResponseFields` | 中间件是**对既有路由的声明式补丁**，不是全局管线。只有你列出的字段能被修改，由宿主校验并应用。before/after 继承目标路由的授权，因此无法借此扩大权限。 |
| Blade 视图 / 带 PHP 的模板 | Page Registry `theme.json.pages[]` + Manifest V3 `templates[]` + `packageFiles[]`（三方身份）与宿主岛白名单 | 模板是惰性 HTML + 白名单内的宿主岛。没有模板代码执行、没有 PHP、没有服务端模板 include。`theme.json.pages[].template` 的每个路径必须恰好有一条 `templates[]` 声明与一条 kind 为 `template` 的 `packageFiles[]` 条目，且两处摘要一致。 |
| 插件直接写 Session | `Host API v2` 的 `PluginSessionService`（`get`/`set`/`delete`）与 SDK 助手 `Host.SessionState(ctx)` | 宿主掌管、按插件命名空间的**会话**状态，用于小型有界工作流值。宿主会在你收到的那次调用上签发一次性会话委派，你回带它、并且永远看不到 session id。状态是易失的：要能容忍未命中，值 ≤ 1 MiB，实际生效 TTL 会被收紧到 Core 会话剩余寿命。核心会话**改写**仍是关闭操作。 |
| `Itf` 接口钩子 | `hooks[]`（`id`、`contractVersion`、`name`、`kind`、`targetId`、`handler`、`inputSchema`）以及类型化的 `events[]`、`queries[]`、`providers[]` | 钩子按名字与契约版本对宿主拥有的目标声明，不是靠反射发现的任意接口实现。载荷是按声明 schema 校验的类型化文档。 |
| 插件辅助库（框架门面） | SDK `sdk/plugin/v2`（`pluginv2.NewTypedDocument`、类型化文档、设置 schema、管理面组件、guard、定时任务、命令） | 辅助库负责构造类型化文档与声明契约；它永远不会给你原始请求、会话或数据库句柄。一切特权能力都是显式声明 + 精确制品信任授权。 |

## 有意关闭的操作

以下能力是**设计上关闭**的，不要去绕：每一条都有安全、完整性或所有权上的原因。

- **原始 Fiber 访问。** 扩展拿不到框架上下文、路由器或处理器链。唯一传输是 Protocol V2 的
  类型化文档；请求头投影默认丢弃 `x-sforum-*`、逐跳头与凭据头，除非该路由显式声明了高风险
  的 `raw_request` 授权。
  **破坏性变更（2026-10-16）：** 该授权现在必须来自你的包**自行声明**的守卫
  （`guards[]` 中 `kind: raw_request`，且 `entry` 需要匹配的 `kind: executable`
  `packageFiles` 条目）。宿主自带的 `core.guard.raw_request` 已被移除，因为它让
  「零守卫代码」的路由也能拿到原始权威；仍声明该 id 的包会在**校验阶段被拒绝**，
  已安装且带着它的包会在下一次启用/恢复时 fail closed。请把守卫引用换成你自己
  声明的守卫 id。
- **核心会话改写。** 插件无法读取、铸造或改写核心会话状态，也永远不会拿到 session 标识符。
  它能做的是在宿主的会话范围下保存自己的有界状态：`PluginSessionService` +
  `Host.SessionState(ctx)`，范围限于宿主发起的那一次调用。没有宿主会话委派的调用会被拒绝，
  而不是被静默地划到某个范围里。
- **任意 Nuxt Layer / 前端代码注入。** 管理端前端资源以不可变摘要路径从包内提供；主题贡献的是
  模板与岛，不是能覆盖宿主组件的 layer。
- **插件前置中间件。** 宿主中间件先于任何扩展代码运行，且没有让扩展更早介入的钩子。路由
  `before` 条目修改的是宿主**已经完成授权**的那条路由。
- **铸造身份（actor minting）。** 扩展不能制造身份。每次请求上的 actor 投影由宿主推导
  （`actorId`、`authenticated`、权限键）；插件只能收窄，不能放宽或伪造。
- **客户端 IP / 代理判定。** 代理信任属于宿主配置；转发头只会以「客户端自己的声明」到达插件，
  宿主不会附加任何背书头。

## 参考包

聚焦参考包位于 `extensions/fixtures/plugins/`。布局契约有专门的参考包：

- `sforum-page-layout-reference` —— 同一个包里用三种方式声明公开页
  （`layout: "bare"`、`layout: "site"`、完全不写 `layout`），一处就能证明差异：
  bare 表示作者自己拥有 chrome，site 表示宿主提供恰好一个 navbar 与 footer，
  省略 layout 就是平台默认的 `site`。它的模板同时满足三方身份契约
  （`theme.json` 的每个模板路径恰好一条 `templates[]` 声明与一条 kind 为 `template`
  的 `packageFiles[]` 条目，摘要一致）。

这些夹具是 CI 夹具：带后端的夹具用 `replace … => ../../../../../apps/api`，由 Go
测试框架编译，因此不适合直接复制到仓库外使用。需要干净目录的包请用下面的快速开始。

## 干净临时目录里可跑通的快速开始

以下命令在仓库树之外执行。`make:plugin` 只依赖 CLI；纯后端路径不需要安装依赖、也不需要联网。

### 插件包：脚手架 → 刷新摘要 → 校验 → 契约测试 → 构建

```bash
mkdir -p /tmp/sforum-quickstart
cd <仓库>/apps/api

go run ./cmd/sforum make:plugin --no-interaction \
  --id acme.quickstart --name "Quickstart" \
  --description "clean temp dir proof" --url https://example.com/qs \
  --author-name Acme --backend --out /tmp/sforum-quickstart/acme.quickstart

go run ./cmd/sforum extension digest --write /tmp/sforum-quickstart/acme.quickstart
go run ./cmd/sforum extension validate /tmp/sforum-quickstart/acme.quickstart
go run ./cmd/sforum extension test /tmp/sforum-quickstart/acme.quickstart
go run ./cmd/sforum extension build --allow-scaffold --skip-install /tmp/sforum-quickstart/acme.quickstart
```

实测输出（2026-10-16，仓库之外的干净 `/tmp` 目录）：

```text
MAKE_OK
digest --write OK
validate OK
test OK
build OK
  errors:  0  warnings: 1  checks: 6
```

那一条 warning 是 `backend.scaffold_stub`：生成的后端入口在你替换成真正的 go-plugin 二进制
之前是 shell 桩。`--allow-scaffold` 就是为「还在写」的阶段保留绿灯用的。

### 主题包：脚手架 → 刷新摘要 → 校验 → 契约测试

```bash
go run ./cmd/sforum make:theme --no-interaction \
  --id acme.quicktheme --name "Quick Theme" \
  --description "theme quickstart" --url https://example.com/qt \
  --author-name Acme --out /tmp/sforum-quickstart/acme.quicktheme

go run ./cmd/sforum extension digest --write /tmp/sforum-quickstart/acme.quicktheme
go run ./cmd/sforum extension validate /tmp/sforum-quickstart/acme.quicktheme
go run ./cmd/sforum extension test /tmp/sforum-quickstart/acme.quicktheme
```

单页脚手架能通过校验与契约测试，但**不能替换拥有完整核心页面集的主题**：激活预检会拒绝
声明页面覆盖不了当前主题页面集的包。这个拒绝是有意的（残缺主题会静默丢掉页面），并且与 CLI
跑的是同一套检查。请补齐你打算拥有的页面，或把它作为新包安装而不是替换活动主题。

### 开发编辑循环

```bash
cd <仓库>/apps/api
APP_ENV=development go run ./cmd/sforum extension dev ../../extensions/dev/acme.quickstart --skip-install
```

一条命令在每次源文件改动后重建、刷新摘要、跑校验与契约测试，并打印新的包摘要与下一步。
它**不会**替你激活：精确制品仍需操作员在管理端扩展流程里确认。旗标说明见
[CLI 手册](../development/cli.md)。

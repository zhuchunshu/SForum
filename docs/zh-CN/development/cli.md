# 开发者 CLI（`sforum`）

[← 开发指南](./README.md)

`sforum` 是 SForum 的开发者控制台（类似 Laravel Artisan 的定位），源码在
`apps/api/cmd/sforum`。用于脚手架、扩展校验、精确摘要、打包、契约测试、
带外恢复、账号/数据维护与假数据种子。

所有命令默认从 `apps/api` 目录运行：

```sh
cd apps/api
go run ./cmd/sforum --help
go run ./cmd/sforum --version
```

## 命令总览

| 命令 | 作用 |
| --- | --- |
| `version` | 打印构建信息（版本、提交、构建时间） |
| `make:plugin` | 生成插件脚手架 |
| `make:theme` | 生成主题脚手架 |
| `seed:forum` | 批量写入论坛假数据（small 模式） |
| `seed:perf` | 百万级读路径种子（`seed:forum --profile=perf-1m` 的别名） |
| `users:reset-password` | 交互式重置用户密码（别名 `user:reset-password`） |
| `revisions backfill` | 分批回填论坛内容修订台账 |
| `extension validate` | 校验扩展包（含 includes / 模板预检） |
| `extension digest` | 查看或刷新 Manifest V3 `packageFiles` 摘要 |
| `extension test` | Host 契约检查（能力、事件、入口等） |
| `extension package` | 打 zip + SBOM stub |
| `extension docs generate` | 从 Go catalog 生成 host 文档 |
| `extension list` | 带外查看扩展恢复状态（不启插件代码） |
| `extension disable` / `disable-all` | 带外禁用第三方扩展 |
| `extension quarantine` | 带外隔离精确的内置/系统制品 |
| `extension command list/run` | 列出 / 运行已信任插件命令 |
| `extension api-lts` | 打印 Host/Frontend API LTS 与 shim 遥测 |
| `extension system-tier list/upsert/disable` | 带外管理系统扩展层（不加载包代码） |
| `dev:cleanup-orphan-plugins` | 停止已改属的扩展后端进程（安全用于运行中的 sforum-api） |

> 本表是命令的**权威清单**；新增或删除命令时必须同步更新本表与
> `docs/en-US/development/cli.md`（`tests/validate-docs.mjs` 会校验覆盖）。

---

## 构建信息：`version`

```sh
go run ./cmd/sforum version
go run ./cmd/sforum --version
```

输出统一构建身份：SForum 版本、Git 提交、构建时间（发布镜像与 CLI 使用同一
构建参数注入）。

---

## 脚手架：`make:plugin` / `make:theme`

交互式（会提问 ID、名称、是否带后端等）：

```sh
go run ./cmd/sforum make:plugin
go run ./cmd/sforum make:theme
```

非交互示例：

```sh
# 本地实验（默认 → extensions/dev/，gitignored，不进后台列表）
go run ./cmd/sforum make:plugin \
  --id acme.demo \
  --name "Acme Demo" \
  --description "示例插件" \
  --backend \
  --no-interaction

# 受保护内置包（→ extensions/builtin/，SyncBuiltins 会扫入后台）
go run ./cmd/sforum make:plugin \
  --id sforum.foo \
  --name "Foo" \
  --description "…" \
  --backend \
  --builtin \
  --no-interaction

# 指定输出目录
go run ./cmd/sforum make:plugin \
  --id acme.demo --name "Acme Demo" --description "…" \
  --backend --no-interaction --out /tmp/acme.demo
```

### 常用 flags

| Flag | 适用 | 说明 |
| --- | --- | --- |
| `--id` | 两者 | 稳定扩展 ID，如 `acme.demo` |
| `--name` / `--description` | 两者 | 展示名与简介 |
| `--url` / `--author-*` | 两者 | 官网与作者信息 |
| `--out` | 两者 | 输出目录；省略则按 dev/builtin 规则落盘 |
| `--builtin` | 两者 | 写到 `extensions/builtin/` 而非 `dev/` |
| `--no-interaction` | 两者 | 关闭交互提问 |
| `--backend` | plugin | 生成 `backend/plugin` stub 与 README |
| `--complex` | plugin | 多文件 manifest（includes + langs + settings 分片） |
| `--prebuilt-settings` | 两者 | 预构建 Admin 设置组件 + Schema 回退 |
| `--provider-slot` | plugin | 声明 provider slot + `provider_probe`（需 `--backend`） |

### 脚手架之后

1. 实现后端逻辑，并把可执行文件编译到 Manifest 声明的 `backend.entry`（通常是 `backend/plugin`）。
2. `extension digest --write` 刷新精确摘要。
3. `extension validate` / `extension test` 做契约检查。
4. 需要分发时再 `extension package`。

第三方插件请走公开 SDK（`apps/api/sdk/plugin`），**不要** import `app/Models/*` 等宿主业务包。完整编写约定见 [插件编写指南](../../extensions/authoring-guide.md)。

---

## 扩展包工具：`extension …`

### 校验 — `validate`

```sh
go run ./cmd/sforum extension validate <package-root>
go run ./cmd/sforum extension validate <package-root> --json   # 打印合并后的 Manifest
```

加载包、解析 `includes`、校验 Manifest V3，并对显式 V3 包做页面模板运行时预检。

Page Registry 主题采用失败即关闭的三方一致性规则：每个
`theme.json.pages[].template` 路径都必须对应唯一的 Manifest V3
`templates[]` 声明，以及一个 `kind: "template"` 的 `packageFiles[]`
条目；路径与 SHA-256 摘要必须一致。`theme.json` 只定义页面映射，不能替代
精确制品声明。缺失或摘要过期会使校验失败，并在激活时返回
`extension.build_failed`。

### 精确摘要 — `digest`

Manifest V3 用 `packageFiles` 的 SHA-256 绑定可执行文件、前端、迁移等。**改过包内文件后必须刷新**：

```sh
go run ./cmd/sforum extension digest <package-root>           # 只检查
go run ./cmd/sforum extension digest --write <package-root>   # 写回根 manifest 并再校验
```

主题新增模板时，应先补齐 `templates[]` 的身份、契约和 ViewModel 声明，以及
对应的 `packageFiles[]` 文件条目，再运行 `digest --write`。该命令会刷新
`packageFiles[]` 及已声明的内联模板摘要，但不会从 `theme.json` 推断模板身份
或文件成员关系。

### 契约测试 — `test`

```sh
go run ./cmd/sforum extension test <package-root>
go run ./cmd/sforum extension test --allow-scaffold <package-root>  # 脚手架阶段可不要求后端二进制
go run ./cmd/sforum extension test --skip-backend-binary <package-root>
go run ./cmd/sforum extension test --json <package-root>
```

对照 host catalog 检查能力、事件、贡献点、provider、job、后端入口等。
`--allow-scaffold` 是 `--skip-backend-binary` 的别名。

内置主题修改后，从 `apps/api` 依次运行：

```sh
go run ./cmd/sforum extension digest --write ../../extensions/builtin/themes/<dir>
go run ./cmd/sforum extension validate ../../extensions/builtin/themes/<dir>
go run ./cmd/sforum extension test ../../extensions/builtin/themes/<dir>
```

然后在仓库根目录运行 `./scripts/build-builtin-plugins.sh`，重启 API 让
`SyncBuiltins` 暂存新摘要，并通过管理后台激活该版本。不要直接修改
`storage/builtin-dev/` 或 `storage/extensions/**`。

### 打包 — `package`

把扩展根目录打成 zip，并生成 SBOM stub。

```sh
# 默认：根目录几乎全部文件都进 zip
go run ./cmd/sforum extension package <package-root>

# 发布包：跳过常见源码 / 开发文件
go run ./cmd/sforum extension package <package-root> --exclude-source

# 指定输出路径
go run ./cmd/sforum extension package <package-root> \
  --exclude-source \
  -o /tmp/acme.demo.sforum.zip
```

| 行为 | 说明 |
| --- | --- |
| 默认包含 | 包根下几乎所有文件 |
| 始终跳过 | `.git/`、`node_modules/`、`vendor/`、已有 `*.sforum.zip` |
| `--exclude-source` 额外跳过 | `*.go`、`go.mod`/`go.sum`、`*.vue`/`*.ts`/`*.tsx`、Sass、source map、`package.json`/`tsconfig` 等配置、`testdata/` / `__tests__/` 等 |
| 发布时通常保留 | `sforum.extension.json`、manifest 分片、`backend/plugin`、预构建 `.mjs`/`.css`、`README.md` |
| 默认输出 | `<package-root>/<目录名>.sforum.zip` + 旁路 `.sbom.json` |

打包前会先做 package validation。输出示例：

```text
package	…/acme.demo.sforum.zip
digest	…
sbom	…/acme.demo.sforum.zip.sbom.json
files	12
skipped	8	(source/dev files)   # 仅 --exclude-source 且有跳过时
```

**重要：**

- 运行时安装**不需要**源码，但默认 `package` **不会自动剥源码**；分发请加 `--exclude-source`，或先拷到干净的 release 目录再打包。
- `--exclude-source` 是启发式过滤，不是「只打 `packageFiles` 清单」。
- 上传 zip 只做惰性校验与存储；首次启用可执行逻辑需要基于精确摘要的信任确认。运营侧说明见 [扩展与主题](../usage/extensions.md)。

推荐发布循环：

```sh
# 1. 编译后端到 backend/plugin
# 2. 刷新摘要
go run ./cmd/sforum extension digest --write <package-root>
# 3. 校验 + 契约
go run ./cmd/sforum extension validate <package-root>
go run ./cmd/sforum extension test <package-root>
# 4. 打发布 zip
go run ./cmd/sforum extension package <package-root> --exclude-source -o /tmp/my-plugin.sforum.zip
```

### Host 文档 — `docs generate`

Host 表面（事件、能力、contribution、provider slot、schedule 等）变更后：

```sh
go run ./cmd/sforum extension docs generate
go run ./cmd/sforum extension docs generate --check   # CI：与已提交文档漂移则失败
```

默认写入 `docs/extensions/catalogs/`（可用 `--out` 覆盖）。

### 插件命令 — `command`

```sh
go run ./cmd/sforum extension command list
go run ./cmd/sforum extension command run <command-id>
go run ./cmd/sforum extension command run <command-id> --input '{"key":"value"}'
go run ./cmd/sforum extension command run <command-id> --input-file /tmp/input.json
```

需要可用的 `DATABASE_URL`（或 `--database-url`）。可加 `--safe-mode`
（在 `SFORUM_SAFE_MODE` 之外强制 Safe Mode）。`list --json` 输出机器可读结果。

### 带外恢复

不启动 SForum 主进程、不执行插件代码：

```sh
go run ./cmd/sforum extension list
go run ./cmd/sforum extension list --json
go run ./cmd/sforum extension disable <extension-id>
go run ./cmd/sforum extension disable-all
go run ./cmd/sforum extension quarantine <extension-id> \
  --expect-version <精确版本> \
  --expect-digest <64位hex摘要>
```

`quarantine` 只隔离**精确匹配**的内置/系统制品：`--expect-version` 与
`--expect-digest` 都必须与实际活动版本一致，避免误隔离。

### API LTS 状态

```sh
go run ./cmd/sforum extension api-lts
go run ./cmd/sforum extension api-lts --json
```

### 系统扩展层 — `system-tier`

不加载包代码，管理 SystemTier 成员（影响启动顺序与恢复语义）：

```sh
go run ./cmd/sforum extension system-tier list
go run ./cmd/sforum extension system-tier upsert <extension-id> \
  --role infra --priority 100 --enabled true
go run ./cmd/sforum extension system-tier disable <extension-id>
```

`upsert` 的 `--role` 取值为 `auth|cache|storage|infra`，`--priority` 越低越先
加载。所有子命令支持 `--database-url` 与 `--json`。

---

## 假数据：`seed:forum` 与 `seed:perf`

```sh
# config.Load 不读 .env，需先导出环境变量
set -a; . ../../.env; set +a   # 若在 apps/api 下，按你的 .env 实际路径调整

go run ./cmd/sforum seed:forum
go run ./cmd/sforum seed:forum --count=100 --users=20 --comments-max=3
go run ./cmd/sforum seed:forum --profile=perf-1m --dry-run
go run ./cmd/sforum seed:forum --database-url 'postgres://…'
```

| 特性 | 说明 |
| --- | --- |
| 写入方式 | 追加；可重复跑 |
| 事件 | 不触发领域事件 |
| 环境 | **仅开发/测试**，勿对生产库使用 |
| 依赖 | `DATABASE_URL` 或 `--database-url` |

`seed:forum` 主要 flags（完整列表以 `--help` 为准）：

| Flag | 默认 | 说明 |
| --- | --- | --- |
| `--profile` | `small` | `small` 或 `perf-1m` |
| `--count` | 1000 | 主题数（perf-1m 默认 1,000,000） |
| `--users` | 50 | 假用户数（perf-1m 默认 200） |
| `--comments-max` | 5 | 普通主题最多评论数（perf-1m 默认 0） |
| `--categories` | 0 | perf-1m 分类数（默认 20） |
| `--hot-comments` | 0 | perf-1m 热帖评论数（默认 50000） |
| `--hot-slug` | 空 | perf-1m 热帖 slug（默认 `perf-hot-thread`） |
| `--category-slug` | 空 | small 模式发布分类 slug（默认 `general`） |
| `--batch` | 20 | 日志/批大小（perf-1m 默认 5000） |
| `--dry-run` | false | 只打印计划 |
| `--confirm-perf-db` | false | perf-1m 非 dry-run 必填 |
| `--database-url` | 环境变量 | 覆盖 `DATABASE_URL` |

`seed:perf` 是 `seed:forum --profile=perf-1m` 的别名，拥有相同的 perf 参数
（`--count` 默认 1,000,000、`--users` 200、`--categories` 20、
`--hot-comments` 50000、`--batch` 5000 等），并同样要求
`--confirm-perf-db` 才能写入。

---

## 账号维护：`users:reset-password`

交互式重置任意用户密码（只读 `DATABASE_URL`/`--database-url`，不加载其他
应用配置）：

```sh
go run ./cmd/sforum users:reset-password
go run ./cmd/sforum users:reset-password --database-url 'postgres://…'
# 别名：
go run ./cmd/sforum user:reset-password
```

---

## 内容修订台账：`revisions backfill`

分批回填论坛内容修订台账：

```sh
go run ./cmd/sforum revisions backfill
go run ./cmd/sforum revisions backfill --batch 1000 --loop   # 持续执行到 pending=0
go run ./cmd/sforum revisions backfill --database-url 'postgres://…'
```

`--batch` 每批处理的 posts 数（默认 100）；`--loop` 持续执行直到无待回填
记录。

---

## 开发进程清理：`dev:cleanup-orphan-plugins`

停止已被改属/孤儿化的 SForum 扩展后端插件进程（对运行中的 `sforum-api`
子进程安全，按 PID 白名单筛选）：

```sh
go run ./cmd/sforum dev:cleanup-orphan-plugins
go run ./cmd/sforum dev:cleanup-orphan-plugins --dry-run   # 只列出 PID，不发信号
```

---

## 包目录约定（速查）

| 目录 | 用途 | 启动扫描进后台 |
| --- | --- | --- |
| `extensions/dev/` | 本地实验（脚手架默认） | 否 |
| `extensions/builtin/` | 随产品内置、受保护 | 是（`SyncBuiltins`） |
| `extensions/optional/` | 仓库可选，需安装 | 否 |
| `extensions/fixtures/` | CI / 契约夹具 | 否 |
| 运行时 `EXTENSION_ROOT` | 运营上传安装后的存储 | 否 |
| `EXTERNAL_EXTENSION_ROOTS` | 独立插件/主题源码集合 | 是（惰性快照，不自动启用） |

完整地图：[extensions/README.md](../../../extensions/README.md)。
机制与信任模型：[插件编写指南](../../extensions/authoring-guide.md)、[运营侧扩展说明](../usage/extensions.md)。

---

## 相关文档

- [API 使用](./api.md)
- [日常工作流](./workflow.md)
- [环境搭建](./setup.md)
- [测试与质量门禁](./testing.md)
- [插件编写指南](../../extensions/authoring-guide.md)
- [Host API v2](../../extensions/host-api-v2.md)

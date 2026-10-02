# 2026-10-02 Dependabot 告警收敛 Handoff

## Changed

- 全部 18 个 Go module 的 `google.golang.org/grpc` 由 `v1.82.1` 升至
  `v1.83.2`（提交 `6ddcfa399`）：`apps/api`、7 个 builtin 插件后端、
  9 个 `extensions/fixtures` 插件后端、`tests/compat`。
- `apps/api` 与 7 个 builtin 后端同步提升 `golang.org/x/net`、`x/crypto`、
  `x/sync`、`x/sys`、`x/text`、`google.golang.org/genproto/googleapis/rpc`；
  其余 10 个 module 只改 `grpc` 与 `genproto`。
- `.gitignore` 补齐 `go build ./...` 在各扩展 backend 与 `tests/compat`
  模块内生成的同名二进制规则，避免本地验证污染工作区。
- 删除 4 个遗留 PHP 分支 `v2`、`dev`、`php82`、`zhuchunshu-patch-1`；各分支
  HEAD 已用 annotated tag 归档为 `archive/php-v2-final`、
  `archive/php-dev-final`、`archive/php82-final`、`archive/php-patch-1-final`。
- 关闭 14 个已被取代的 grpc Dependabot PR（#127-#144）。
- grpc bump 改动 `apps/api/go.mod` 的 plugin-transport 依赖，改变了共享插件
  运行时摘要，触发 `tests/validate-builtin-plugin-versions.mjs` 的版本漂移
  闸门：7 个 builtin 插件 patch 升位（auth-github 1.0.7、content-policy
  1.1.6、search-site 1.0.6、smtp 1.1.6、storage-fs 1.1.7、storage-s3
  1.0.8、web-push 1.0.6），`tests/builtin-plugin-release-baseline.json` 用
  `node tests/validate-builtin-plugin-versions.mjs --write` 重写。

## Decisions

- 一次性在 18 个 module 统一 bump，而不是逐个合并 Dependabot PR：PR 只覆盖
  14 个 module，且目标版本分散在 `1.83.1` / `1.83.2`。
- 目标版本选 `v1.83.2`。三个 advisory 均在 `>= 1.83.1` 修复；`v1.84.0`
  随后才发布，留给下一轮 Dependabot PR 处理。
- composer 告警无法通过升级消除：`composer.lock` 与
  `app/Plugins/Mail/composer.lock` 都不在默认分支，只存在于废弃的旧 PHP
  分支，因此走分支清理而不是改依赖。
- `apps/api` 直接依赖 grpc（`app/Support/HostAPI/v2_database.go`、
  `app/Http/Controllers/Extensions/admin_surfaces.go`），升级后必须跑 Go 全量
  构建与测试，不能当作纯间接依赖处理。

## Next

- 容器镜像扫描的 6 个 HIGH 已在 `dec318ed4` 修复（见下节），CI 五作业全绿。
- `apps/web` 仍有 7 个 npm、2 个 docker、1 个 compose、1 个 actions 的常规
  更新 PR 待审。
- `dependabot/go_modules/apps/api/core-go-7814c4887b`（PR #152）把 grpc 提到
  `v1.84.0`，该版本仍在 GO-2026-6443 影响范围内（1.84 线要到
  `v1.85.0-dev.0.20260825072537` 才修），合并会重新弄红 govulncheck。要么把
  该 PR 钉回 `v1.83.2`，要么等 `v1.85.0` 正式版。

## Container image scan follow-up

`Container / web` 作业的 Trivy 步骤在 grpc 收敛后成为唯一失败项，报出 6 个
HIGH（CRITICAL 0），全部可修复：

| 依赖 | 原版本 | 修复版本 | 漏洞 |
| --- | --- | --- | --- |
| `@tiptap/core` | 3.27.1 | 3.31.3 | GHSA-j95f-988m-3j2f（Markdown 属性解析二次方 ReDoS） |
| `devalue` | 5.8.1 | 5.9.4 | CVE-2026-92708、GHSA-mcm9-63f2-9j32、GHSA-r9w8-h9r3-54w4、GHSA-x5rw-q4pp-hg5g |
| `sharp` | 0.35.3 | 0.35.5 | GHSA-rgj7-g3m4-5g8c（libheif） |

处理方式：12 个 `@tiptap/*` 包统一升到 3.31.3；`devalue`、`sharp` 通过
`package.json` 的 `overrides` 强制（二者是 Nuxt / ipx / nuxt-seo-utils 的间接
依赖）。Tiptap 升级后出现 `prosemirror-model` 1.25.9 与 1.25.12、
`prosemirror-view` 1.42.0 与 1.42.6 双版本，导致
`app/utils/editor/editorImageUpload.ts` 类型不兼容，用同名 `overrides`
收敛，未改业务代码。

验证：`bun run typecheck` 通过；`bun test` 904 pass / 1 fail——
`tests/extensions/pluginRouteProxy.test.ts` 的 `retry-read` 期望 200 实得
502，已用 `git stash` 回退依赖到 HEAD 版本复现同样失败，属既有环境依赖问题。

## Open Questions

- `dependabot.yml` 只跟踪 10 个 Go module，但告警会覆盖全部 18 个（含
  `extensions/fixtures/**` 与 `tests/compat`），是否要将 fixtures 纳入
  updates 配置或接受这部分噪音，尚未定论。
- `TestRevisionStreamReconcilesMissedWakeAndHeartbeats` 在 CI 上偶发失败
  （本地 100+ 次串行与 GOMAXPROCS=1 压力复现均未出现），疑似 runner 负载下的
  时间敏感断言，需要一轮针对性的去抖改造。

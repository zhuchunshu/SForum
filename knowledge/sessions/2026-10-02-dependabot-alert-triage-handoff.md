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

- 等 GitHub 依赖图重新解析（push 已触发 Configured Graph Update），确认 94 条
  告警是否自动关闭；若 composer 40 条未关闭，再批量 dismiss。
- `apps/web` 仍有 7 个 npm、2 个 docker、1 个 compose、1 个 actions 的常规
  更新 PR 待审。
- `dependabot/go_modules/apps/api/core-go-7814c4887b`（PR #152）把 grpc 提到
  `v1.84.0`，该版本仍在 GO-2026-6443 影响范围内（1.84 线要到
  `v1.85.0-dev.0.20260825072537` 才修），合并会重新弄红 govulncheck。要么把
  该 PR 钉回 `v1.83.2`，要么等 `v1.85.0` 正式版。

## Open Questions

- GitHub 对「依赖只存在于非默认分支」的告警，是否会在分支删除并刷新依赖图后
  自动关闭，需以实际刷新结果确认。
- `dependabot.yml` 只跟踪 10 个 Go module，但告警会覆盖全部 18 个（含
  `extensions/fixtures/**` 与 `tests/compat`），是否要将 fixtures 纳入
  updates 配置或接受这部分噪音，尚未定论。

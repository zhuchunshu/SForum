# 2026-10-22 发布前 CI / Dependabot 红色修复

## Changed

- `tests/validate-admin-framework.ts`：`useAdminRoutes` 静态断言改成与当前实现
  一致（经 `useLocalePath` 委托 i18n、跟随 `normalizeAdminRoutePrefix`、禁止手拼
  `locale.value`）。旧断言要求 `useI18n`/`defaultLocale` 并禁止 `useLocalePath`，
  与 2026-10-22 首注册者落地时改过的实现相反，导致 `Contracts and static gates`
  job 在 main 上必红（`Admin routes should read the active locale directly`）。
- `apps/api/app/Support/Extensions/`：新增
  `lifecycle_actor_fixture_test.go` 的 `insertLifecycleActorFixture`，带 23505 重试；
  migration / fence / runtime-activation / jobs / provider-slot 五处 users fixture
  改为调用它，`insertCleanupActor` 改为一行委托（删掉重复实现与 `pgconn` import）。
- `.github/dependabot.yml`：`/apps/web` 的 `package-ecosystem` 从 `npm` 改为 `bun`。
  `npm_and_yarn` 无法更新 `bun.lock`，每周计划更新都以
  `misconfigured_tooling` 失败（Actions 上两条常红 run 即此原因）。

## Decisions

- CI 红优先按「测试断言过时」处理：实现契约以
  `apps/web/tests/admin/adminRouteLocalization.test.ts` + `knowledge` 记录为准，
  `validate-admin-framework.ts` 是滞后门禁。
- `users_pkey` 冲突按仓库既有模式（fixture 侧重试）修，不动序列起点：共享测试库
  里别的包用显式 id 造 fixture 不推进序列，改序列会破坏依赖 `id == 1` 的断言。
- Dependabot 用 `bun` 生态（`dependabot-core` 已有 `bun/` 实现），不再用 npm
  跟踪 `apps/web`。

## Verification

- 复现：临时库显式插 `users` id 1..5（序列停在 1）后跑
  `TestProductionLifecycleMigrationsDeclaredSQLBlocksWithoutP5AndRejectsDrift`，
  旧代码报 `duplicate key value violates unique constraint "users_pkey"`，
  与 CI 日志逐字一致；换修复后同一库状态下该包全绿（48.7s）。
- `./scripts/ci/contracts-gate.sh` 全流程通过；`bun tests/validate-admin-framework.ts`
  通过；`actionlint v1.7.12` 无输出。
- `./scripts/test-api-isolated.sh`（全量 `go test ./...`）通过；
  `./scripts/test-api-isolated.sh --compat`（required cells 全 pass）与
  `go build ./...` 通过。
- 推送 `0e5dca970` 后，main 的 `CI` run `39781534509` 五个 job 全 success；
  Dependabot 的 `bun in /apps/web` 更新 run（`39781552970`）success，并新开了
  10 个 `dependabot/bun/apps/web/*` PR（diff 只含 `bun.lock` + `package.json`）。

## 追加：actions 大版本升级与 #168 / #171 处置（同日）

- 合并 #174（vite 8.1.5 → **8.3.4**，承接 #171 想修的 5 条 GHSA）。
- 新 PR #182 承接 #168：4 个 workflow 的 action 升到 checkout v7.0.1、setup-go
  v7.0.0、setup-node v7.1.0、buildx v4.4.1、login v4.6.0、metadata v6.2.0、
  build-push v7.4.0、upload-artifact v7.0.2、download-artifact v8.0.2，trivy 与
  attest 重新指向当前 SHA；同时把 `release_workflow_test.rb` 的 setup-node pin
  同步到 `949feb2413d6458794dcd2491c4babbbce0c15c1`（仍断言 Node 24 /
  registry-url / package-manager-cache=false），并留注释说明这是人工审查点。
- 升级前逐项核实：每个新 SHA 用 `gh api repos/{owner}/{repo}/commits/{sha}` 确认
  存在且对应标注版本；新版本 `action.yml` 的 inputs 与仓库用法全部兼容
  （setup-node 仍有 `package-manager-cache`，upload v7 有 `name`/`path`，
  download v8 有 `pattern`/`merge-multiple`，build-push v7 保留
  `outputs`/`platforms`/`push`/`provenance`/`sbom`/`cache-*`）；运行时均为 node24。
- #168 与 #171 已关闭，留言写清承接方。#171 的 composer 清账记录落到
  `sessions/2026-10-22-dependabot-alert-triage.md`。
- 遗留风险：release.yml 里只在发 tag 时执行的 action（login/metadata/attest/
  upload/download）无法在合并前做行为验证，下次发版是首次真跑。

## Next

- 推送后确认 main 的 `CI` 三个 job 全绿（发布 workflow 的 `verify-main-ci` 会等它）。已确认。
- 到 Dependabot 面板点一次 `apps/web` 的 “Check for updates”，让 bun 生态跑出
  新结果，历史红色 run 才会在列表里被新绿 run 取代。已自动触发并 success。
- `dependabot/github_actions/*`（PR #168）的 `Release tooling` 红是**设计上的评审点**：
  `scripts/ci/release_workflow_test.rb` 硬编码 `actions/setup-node` 的 reviewed SHA
  与 Node 24 / Trusted Publishing 参数。合并该 PR 必须同步更新这条 pin，否则 main 的
  `Release tooling` 会被自己拦红。不要为了“变绿”直接放宽该断言。已通过 #182 处理。
- 下次打 tag 时盯一遍 release 全流程（尤其 upload/download artifact 的 v7/v8 行为）。

## Open Questions

- 其它包（Identity / Entitlements / Forum / bootstrap / Http）仍有不带重试的
  `INSERT INTO users ... RETURNING id` fixture，属同源隐患；本轮未扩散修改。

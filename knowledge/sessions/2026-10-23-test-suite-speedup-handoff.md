# 2026-10-23 全量测试提速 Handoff

## Changed

- `scripts/test-api-isolated.sh`：Go 测试改成**集群级分片并行**。`SFORUM_TEST_SHARDS`
  默认 `min(nproc, 4)`，每个分片一个一次性集群 + 完整迁移，片内 `-p 1`
  （`SFORUM_TEST_PARALLEL` 保留，默认 1）。新增 `--with-compat`（分片跑完后跑 compat
  农场）与内部 `--shard i/n`；`--compat` 独立入口保留。参数原样透传给 `go test`，
  只把字面量 `./...` 换成分片包清单。
- `scripts/test.sh`：三条线并发（api / web / static），`wait` 聚合退出码，日志写
  `tests/.logs/{api,web,static}.log`，结束打印摘要 + 失败线日志尾部；新增 `--fast`。
  计时用 EXIT trap 写，不用 `cmd || status=$?`（后者会在函数体内关掉 errexit，
  实测漏报过一次 FAIL）。
- `scripts/ci/go-gate.sh`：单次调用 `--with-compat`，容器与迁移从两遍变一遍。
- `apps/web/package.json`：`typecheck` 加 `--incremental --tsBuildInfoFile
  ${NUXT_BUILD_DIR}.tsbuildinfo`；`.gitignore` 加 `tests/.logs/`。
- `apps/api/app/Models/Extensions/theme_publication_postgres_integration_test.go`：
  把 `defer admin.Close()` 提到 `defer setup.Release()` 之前注册，修掉 fixture 失败后
  puddle `Pool.Close` 永久挂住的问题。
- `AGENTS.md` Commands 段补 `--fast` 与三线并行的说明。

## Decisions

- `knowledge/decisions/2026-10-23-test-gate-parallelism.md`：为什么并行从「包级 `-p`」
  挪到「集群级分片」，以及 `--fast`、增量 typecheck、fixture 死锁修复的取舍。

## 计时（12 核，开发环境同时在跑）

| 轮次 | 缓存 | 总墙钟 | api | web | static |
| --- | --- | --- | --- | --- | --- |
| 1 | 冷 | 7:26.87 | 447s | 387s | 105s |
| 2 | 冷 | 7:24.17 | 444s | 383s | 106s |
| 3 | 热 | 6:14.35 | 41s | 374s | 89s |
| `--fast` | 热 | 0:40.71 | 26s | 40s | 21s |

基线（2026-10-20 记录）：21:31（Go 14m45s、build 4m15s、release-tooling 1m19s、
其余 1m12s）。现在 Go 段 4–5 分钟（冷）/ 41 秒（热），长尾变成 Nuxt production build。

## M0 关键证据（为什么不能只用 `-p`）

- 原命令 `SFORUM_TEST_PARALLEL=8 ./scripts/test-api-isolated.sh` 全量命中缓存
  （34/34 包 `(cached)`），证明不了 flaky；M0 用 `-count=1` 强制真跑。
- 裸 `-p 8` 真跑 8 分 04 秒，3 处失败 + 1 处挂死：
  `extension database physical resource conflict`、
  `lifecycle operation already in progress: initial plugin runtime publication`、
  `could not open relation with OID ...`（goose partial migration）、
  `core database authority conflicts with the Host ownership boundary`。
- 三个包单独 `-p 3` 一起跑 91 秒全绿，说明"排除名单"只是降概率；最终改成集群级隔离。

## Next

1. 推 PR 验证 ci.yml 三个 gate job（go / web / contracts）全绿——这是任务书验收第 5 条，
   本地只能跑到三条 gate 脚本的并集。
2. 分片均衡：现在按包名轮转，实测第 3 片 27s、第 1 片 366s；按耗时做 LPT 可再省
   1–2 分钟。
3. M6（后续独立任务）：`database/migrator` 等包内部仍串行，是 `-p` 提升后的新长尾。

## Open Questions

- Nuxt production build 受内存压力影响波动很大（本机 36s–240s），门禁墙钟会跟着抖；
  是否给 build 单独设内存/并发上限，留给后续评估。

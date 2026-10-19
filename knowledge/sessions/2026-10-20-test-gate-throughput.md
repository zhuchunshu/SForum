# 2026-10-20 测试门禁吞吐

## Changed

- `scripts/test-api-isolated.sh` 三处改动，目标是把 `./scripts/test.sh` 的
  墙钟从「每次 21 分半」拉下来。基线实测（12 核，负载 13.7，swap 打满）：
  release-tooling-gate 1m19s、Go 测试 14m45s、compat+contracts+typecheck+bun
  test 1m12s、Nuxt build 4m15s，合计 21m31s。
- 一次性 Postgres 集群改用固定宿主端口 `SFORUM_TEST_POSTGRES_PORT`（默认
  55432），占用时回退临时端口。原因：`DATABASE_URL` 里的端口每次随机，而 Go
  把测试通过 `os.Getenv` 读到的环境变量值算进结果缓存键，导致 45 个数据库包
  每次运行都重跑（实测 662 秒）。固定端口后同端口二次运行命中 `(cached)`，
  换端口立刻重跑，已验证。
- 集群加 `-c fsync=off -c synchronous_commit=off -c full_page_writes=off`
  （三项 GUC 已在容器内确认生效）。对 migrator 包只从 160.8s 降到 150.5s，
  收益约 6%：该包瓶颈不是提交落盘。
- `go test -p 1` 改为 `-p "${SFORUM_TEST_PARALLEL:-1}"`，默认行为不变。

## Decisions

- 固定端口只当缓存键，不当作用域保证：集群仍每次新建并销毁，端口冲突时回退
  临时端口，只损失缓存。缓存语义与既有的 87 个非数据库包一致，想要真跑一遍
  加 `-count=1`。
- `-p` 默认保持 1。包之间共享同一个 database，靠 schema 隔离；`app/Models/...`
  25 个包用 `-p 4` 全绿（203s → 103s），但全量套件尚未验证。

## Second pass: migrator 热点测试并行化

- `TestCoreMigratorRejectsKernelAuthorityDrift` 的 21 个子用例改成 `t.Parallel()`，
  并把 fixture 名字的时间戳后缀换成「PID + 原子自增」（并发下 `UnixNano()` 会撞名）。
  **73.8s → 14.3–16.0s**，连跑 3 次 21/21 全过。
- `database/migrator` 整包（`-count=1 -v`）**174.0s → 114.7s**。
- 模板库克隆方案被否，理由写进代码外的记录：`core_authority.go` 每次用
  `SELECT current_database()` 现算 `OwnerRoleName`，克隆库的对象归模板库角色所有，
  `validateCoreOwnershipSources` 会判成外来属主。要绕开就得 `REASSIGN OWNED`
  重建集群级授权，等于让权威收敛测试跑在与真实迁移路径不同的状态上。
- 另外 11 个 2.8–11s 的测试**故意没有并行化**：其中
  `TestCoreUpgradeCompatibility*` 带 10 秒内部截止时间和锁交错断言，12 路并发下
  容易变成 flaky，不值得为 ~30s 冒这个风险。

## Next

- `Models/Extensions`（87–114s）与 `Models/Identity`（72–90s）是剩余两个大头。
- Nuxt build 在本机内存充裕时约 36s（client/server 各 36s），swap 打满时会退化到
  58–117s 并以 `[vite:define] EPIPE` 或 `.nuxt-build/dist` ENOENT 失败——这是内存
  压力导致的构建失败，不是代码问题。

## Open Questions

- 全量 `./scripts/test.sh` 用 `SFORUM_TEST_PARALLEL=4` 是否稳定？目前只在
  `app/Models/...` 子树验证过；`sforum_core_v1`、`sforum_host_extensions`、
  `sforum_ext_s_fixture` 这类固定 schema 名如果被两个包同时创建会撞车。

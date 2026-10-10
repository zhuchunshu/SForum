# 2026-10-11 Achievements M1 Handoff

## Changed

- 新建受保护内置插件包 `extensions/builtin/plugins/sforum-gamification/`：
  - `sforum.extension.json`：Manifest V3，`database.grants: ["own_schema"]`，
    schema `sforum_gamification`，retention retain / export-30d；digest 已用
    `extension digest --write` 写入并通过 `extension validate`。
  - `backend/store.go`：own_schema 租约（`SFORUM_DATABASE_URL`/`SFORUM_DATABASE_SCHEMA`）
    + 无租约内存回退 + `schema_meta` 版本化 DDL（五表：badge_definitions、
    user_badges、user_points_ledger、level_rules、user_metric_counters）。
  - `backend/store_badges.go`：定义 CRUD、disable-not-delete、保存时规则
    JSON v1 校验（threshold/manual 语法）。
  - `backend/store_ledger.go`：`(event_id, rule_id)` 幂等账本、负值拒绝、
    聚合求和、等级查表、`(user_id, badge_slug)` 幂等授予、manual 必须带
    actor、撤销、用户徽章列表。
  - `backend/store_counters.go`：通用 `(user_id, metric)` 计数；指标白名单
    topic_count/comment_count；registered_days 拒绝落库（评估时现算）。
  - `backend/store_test.go`：单测（校验矩阵、幂等、等级、停用保留历史、
    计数白名单）+ 真库集成测试
    （`SFORUM_GAMIFICATION_TEST_DATABASE_URL` 控制，本机 127.0.0.1:15432
    通过，含 PG 唯一约束下双发幂等变异验证）。

## Decisions

- 插件 schema 用原生 `database/sql` + pgx（与 custom-content 参考实现同型），
  不引 sqlc —— sqlc 是 Core 库的生成工具链，插件 own_schema 无生成配置。
- 存储层双形态：有租约走真实 PG，无租约走内存（进程内单测可密闭运行）；
  集成测试必须打到真实库。

## Next

- M2 grant engine：规则匹配、五步流水线、等级评估、allowed/denied + 重放
  测试；backend 从信号等待占位换成 protocol v2 服务面（参考
  sforum-content-policy `plugin_v2.go` 的 HookRequest 处理）。
- M3 事件接线：manifest events（observe × 3）+ Host River 投递集成测试。

## Open Questions

- **重要**：`backend/plugin` 目前是占位二进制（只起 store、等信号），不要在
  admin 里启用 sforum.gamification，直到 M2 接上协议 v2。SyncBuiltins 若已
  stage 它也只是候选，不会自启。
- 启用插件时的历史回填（把存量 topics/comments 计入 counters）按 M0 建议
  做成管理员手动触发的 backfill 任务，在 M2/M4 落地。

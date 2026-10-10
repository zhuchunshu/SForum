# 2026-10-11 Achievements M3 Handoff

## Changed

- **M3 端到端集成测试**（`apps/api/app/Support/Extensions/IntegrationTests/
  gamification_plugin_integration_test.go`）：真实子进程 + own_schema 租约 +
  `Manager.Deliver` observe 投递，断言注册快照、成长值入账、阈值徽章授予、
  River 风格重放幂等（徽章/账本/计数三处都不双计）、未订阅事件拒绝。
- **schema v2**：`user_registrations` 注册时间快照表（user_id PK，
  幂等首写）。解决 M2 遗留的「巡检无 list 查询」问题——快照自包含，
  不依赖 HostQuery。
- **引擎**：`HandleEvent` 在 user.registered 时写快照（取接收时刻，
  周年语义容忍一天偏移）；`RunDailyScan` 按 user_id 游标分批巡检快照表，
  单用户失败不中止整轮。
- **存储**：所有 SQL 改用租约注入的 `SFORUM_DATABASE_SCHEMA`
  （宿主分配物理 schema 名，如 `sforum_s_<slug>_<hash>`），仅本地直连时
  回退自建 `sforum_gamification`。manifest 的 `database.schema` 是声明，
  不是物理名。
- manifest events 补 `resultSchema`（Host 客户端对 observe 事件也要求
  完整契约），插件 hook 返回带 schema 的 TypedDocument。

## Decisions

- 注册快照方案 = 事件时写入插件自己的表（原 M2 open question 选 a）。
- 插件 SQL 永远不硬编码物理 schema：`SFORUM_DATABASE_SCHEMA` 注入 + 回退
  常量。宿主物理命名（`extensionDatabasePhysicalName`）是宿主权威。

## Next

- M4 admin UI：adminSurfaces（徽章 CRUD、manual grant、grant log、成长值
  设置 + 恢复默认）、`sforum.gamification.manage` 权限声明、每日巡检的
  schedules 声明 + RunDailyScan 接 ExecuteJob、默认徽章/等级种子。
- M5 公开面：forum.profile.tabs 徽章墙 extensionRoute、
  `sforum.gamification.badge_granted` 通知类型。

## Open Questions

- 默认徽章种子时机：enable lifecycle（RunLifecycle）还是 admin 手动
  「恢复推荐默认」？倾向后者（Beginner-friendly 规则要求一键恢复，
  且 enable 时自动写数据与「exact-artifact 声明」精神一致但需确认
  lifecycle hook 里允许 DDL/数据写入）。

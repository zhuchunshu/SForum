# 2026-10-11 Achievements M2 Handoff

## Changed

- `backend/engine.go`：五步授予流水线。`HandleEvent` = 成长值入账 + 规则
  匹配；`EvaluateDaily` = registered_days 周年巡检（幂等键含周年年份）；
  `ManualGrant` = 管理员手动授予（threshold 徽章拒绝走 manual 路径）。
  关键幂等设计：`applyGrowth` 先写账本（`(event_id, rule_id)` 唯一），
  写入成功才加计数——River 重放时账本冲突挡住双计。
- `backend/plugin_v2.go`：占位二进制换成真 protocol v2 服务面。三条
  observe hook（`sforum.gamification.event.*`）经 SDK HookRegistry 分发
  到引擎；事件幂等锚点 = `(event name, user, 实体 id)` 稳定组合，不依赖
  delivery_id（重投会变）。成长值支持环境变量覆盖（M4 设置界面接管前的
  开发通道）。
- manifest：新增 `events[]`（observe × 3，各带 inputSchema 引用）+
  三个 payload schema packageFiles；`digest --write` + `validate` +
  `extension test`（8/8）通过。
- `engine_test.go`：阈值授予、重放不双计/不双入账、跨指标隔离、脏规则
  跳过不崩、注册成长值、元老周年幂等、manual 允许/拒绝、未知事件拒绝。

## Decisions

- observe 事件的幂等锚点用 `(name, user, entity)` 而非 delivery_id：Host
  重投时 delivery_id 会变，但业务实体键不变。
- 计数器增量与账本写入的顺序 = 账本先行：账本唯一键是幂等权威，计数是
  派生值（可由账本重放修复，与「无余额列」设计一致）。

## Next

- M3 事件接线端到端：Host River `EventDeliveryArgs` → InvokeHook → 引擎
  的真实集成测试（参考 `IntegrationTests/custom_content_reference_plugin_
  integration_test.go`），验证 manifest events 声明与 Host 目录订阅一致。
- 每日巡检的 schedules 声明（manifest schedules + River periodic）——
  M0 矩阵已确认机制，本轮未做。
- 通知发送（M5）与 admin 手动授予 RPC 面（M4）。

## Open Questions

- 巡检任务需要「所有用户」的注册时间：无 list 类 HostQuery。候选方案：
  a) 注册事件时把 registered_at 快照存进插件表（新列）；b) 管理员触发的
  backfill 任务分批用 safe_user.by_id 查询。倾向 a（自包含、无查询依赖）。

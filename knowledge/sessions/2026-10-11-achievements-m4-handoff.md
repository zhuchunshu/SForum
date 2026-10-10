# 2026-10-11 Achievements M4 (B后台) Handoff

## Changed

- **manifest**：`permissionDefinitions`（`sforum.gamification.manage`，建议
  角色 admin，assignmentPolicy host）、9 条 routes（8 条管理路由走
  `core.guard.permission` + 1 条公开徽章墙）、`jobs` + `schedules`
  （每日 03:30 UTC 巡检）、settings include（5 个选项，全带推荐默认）、
  `audit.append` 能力、11 个路由 schema。
- **插件后端**：`routes.go`（管理路由分发：徽章 CRUD 含停用不删除、
  手动授予/撤销、授予日志、等级、一键恢复默认、历史回填、公开徽章墙）；
  `engine_admin.go`（默认 8 徽章 + 5 等级；`Backfill` 用 SetCounter
  权威覆盖而非累加，重跑安全；`ProfileWall` 隐藏已停用徽章但保留授予）。
- **集成测试** `gamification_admin_integration_test.go`：经真实
  routes.Registry 冻结路由 + 运行时调用租约，验证恢复默认 → 手动授予 →
  幂等重放 → 授予日志 → 公开墙 → 撤销 → threshold 徽章拒走 manual（422）。

## Decisions

- 回填语义 = **权威覆盖**（SetCounter），不是累加：回填是重建，重跑必须安全。
  这是集成测试暴露的真实缺陷（原实现重跑会双计）。
- 响应 schema 逐条对应 manifest 声明：Host 的冻结校验要求
  RequestSchema/ResponseSchema 与声明完全一致，RouteRequest 不携带它们。
- 结构化数组（badges/grants/levels）以 JSON 字符串回传：TypedDocument
  值域只有原始类型，`[]map[string]any` 会在 proto 编码时失败。
- HTTP 路由**必须**声明 request/response schema（Host 契约完整性强制）。

## Next

- **M4B 渲染态管理界面**：后台固定标签设置页（徽章表格/编辑、手动授予表单、
  成长值设置 + 一键恢复默认、授予日志），i18n 文案，权限目录展示文案，
  桌面 + 390x844 浏览器 QA。
- M5 公开面：`forum.profile.tabs` 徽章墙（数据契约已就绪）、
  `sforum.gamification.badge_granted` 通知类型。

## Open Questions

- 管理面 UI 走「插件 adminSurfaces 声明 + Core 通用渲染器」还是「Core
  固定标签页 + 插件路由」？M4 已选后者（数据在插件路由里），与 M0 矩阵
  「Core 只加框架契约、业务在插件」一致；但后台标签页需要 Core 侧一个宿主
  壳，这是 M4B 唯一需要 Core 改动的地方，需确认几何契约（对齐
  `pages/admin/settings/index.vue`）。
- 默认徽章种子时机：当前为 admin 手动「恢复推荐默认」路由触发，未做
  enable 时自动写入（避免生命周期里做数据写入）。

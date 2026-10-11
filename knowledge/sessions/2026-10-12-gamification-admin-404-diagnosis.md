# 2026-10-12 Gamification Admin 404 Diagnosis And Sidebar Registration

## Changed

- **后台侧栏注册**（`apps/web/app/config/adminModules.ts`）：`/gamification` 页面定义
  早已存在（权限 `sforum.gamification.manage`、i18n、四个 tab 组件），但
  `adminSidebarNavigation` 从未引用它。已加入「论坛管理」folder（moderation 之后）。
  这是"菜单没出来"的唯一原因——纯前端注册缺口，DB 里 admin 角色已有 12 条
  `sforum.gamification.manage` 授权。
- **"未激活提示闪烁"修复**（`composables/admin/useAdminGamification.ts` +
  `pages/admin/gamification.vue`）：badges/levels 与 `/admin/extensions` 三个
  `useAsyncData` 并行，extensions 未就绪时 `pluginEnabled=false` 先闪"插件未启用"，
  随后 badges 的 404 先落地变成"加载失败"。新增 `pluginStatusResolved` 门控，
  告警仅在扩展列表就绪后才渲染。typecheck 绿。

## Diagnosis (404 根因, 证据链已闭合到最后一环)

- 症状：`GET /api/v1/admin/gamification/badges` = 404；route-inspector 对同路径
  报 `resolution: not_found`，**in-memory Registry revision=2**（= 仅 core
  publication + 一次改动），`extension_route_provider_selections` 表 0 行。
- 已排除的假设（全部有证据）：
  - 插件未启用 → DB `extensions.status=enabled`，active_version_id=2517。
  - Recovery-only 模式 → `/api/v1/ready` 200 ready:true（recovery 会强制 not-ready）。
  - manifest 缺路由 → version 2517 manifest 有 10 routes + 1 openapi 声明。
  - staged artifact 漂移 → storage digest `d40ace90…` 目录完整，openapi fragment
    sha256 与 manifest 声明一致；插件进程正以该 digest 运行。
  - shortcodes 路由作对照 → shortcodes manifest 0 routes（纯事件插件），所以
    revision=2 的"另一次改动"不是它；**当前没有任何插件路由成功注册过**。
  - schema catalog 构建失败会崩溃 API → 不会，API 正常 serving，说明崩溃路径
    没走，走的是 `RestoreRoutePublications` 里某处静默 continue/降级。
- 未闭合的最后一环：boot 时 `RestoreRoutePublications`（
  `lifecycle_registry_publication.go:329-359`）对 gamification 的物化为何没进
  Registry。候选：① `manager.ActiveRuntimeInstance(item.ID)` 在 restore 时刻
  返回 err → `continue` 静默跳过（插件进程在跑，但可能启动晚于 restore 调用）；
  ② `BindRouteExecutionPolicies` 对 10 条路由报 schema missing → 但这条路径会
  return err 导致 API 起不来，与现状矛盾。**发表路径全程零日志**，无法从外部
  区分。需要一次带日志的 API 重启确认（见 Next）。

## Decisions

- 修复不发裸 SQL、不绕过正式 Enable/Restart 生命周期（AGENTS.md 红线）。
- 侧栏挂载点选「论坛管理」folder：徽章/成长值是论坛运营语义，不是系统配置也不是
  扩展高级工具；permission `sforum.gamification.manage` 已过滤无权限用户。

## Next

1. **运行时取证（需用户配合）**：用户持有 air 的 pts/1 终端。请用户重启 dev API
   （或授权我 kill air 进程 795037 后由 `./scripts/api-dev.sh` 重启），观察启动
   日志中 gamification 插件握手顺序 vs route restore 顺序；同时在
   `lifecycle_registry_publication.go` 的 `ActiveRuntimeInstance` err 分支临时
   加一行 `log.Printf` 确认是否走了静默 continue。
2. 按证据修复：
   - 若是时序问题（restore 早于插件 active）：restore 应对 enabled 且有路由的
     插件做有界等待/重试，或由 runtime coordinator 在插件 ready 后触发
     `reconcileRoutes` 增量注册（该函数已存在，enable 路径在用）。
   - 修复后 `curl /api/v1/admin/gamification/badges`（带管理员会话）= 200。
3. 走完 B3（四 tab 增删改查 + 恢复推荐默认）+ M6 rendered QA（desktop + 390x844），
   闭合 handoff `2026-10-12-plugin-runtime-recovery-and-test-isolation.md` 遗留的
   profile achievements 422（`routes.go` 里 digest d40ace90 已埋临时探针，
   修复后移除并重 digest + staging rebuild + enable cycle）。
4. 集成测试一律走 `./scripts/test-api-isolated.sh`，禁止继承 dev DATABASE_URL。
5. 完成后来更新本 handoff 的 404 根因结论。

## Open Questions

- `RestoreRoutePublications` 的 `ActiveRuntimeInstance` continue 分支是否就是根因？
  （若是，这是所有"带路由插件"的共性 boot 时序缺陷，不只是 gamification。）
- 短期 workaround：在后台对 gamification 做一次 disable→enable 循环，enable 路径
  会走 `reconcileRoutes` 增量注册，可能无需改代码即可恢复路由。待用户确认后验证。

# 2026-10-11 Achievements M4B (后台管理界面) Handoff

## Changed

- **页面壳** `apps/web/app/pages/admin/gamification.vue`：严格对齐 Core
  后台设置几何契约（紧凑 `text-xl` 标题 + 带边框 `UDashboardToolbar` +
  `SFAdminFixedTabNav` + 单个 `min-w-0` 面板），四个标签页（徽章定义 /
  等级称号 / 手动授予 / 历史回填）。页签写回 query 用 `router.replace`
  （不触发完整导航，避免切页白屏）。
- **标签组件**（`components/admin/gamification/tabs/`）：
  - Badges：表格（图标/名称/标识/规则/成长值/稀有度）+ 搜索 + 新建/编辑
    表单（规则类型 + 指标 + 阈值）+ 停用（不删除）+ 一键恢复推荐默认 +
    「推荐徽章未齐全」提示。
  - Levels：按 min_points 排序列表 + 新增/编辑 + 推荐阶梯缺失提示。
  - Grants：用户 ID + 仅手动徽章下拉 → 授予 / 查看授予记录 / 撤销。
  - Backfill：JSON 载荷校验（userId 正整数）+ 示例填充 + 覆盖语义警告。
- **数据层**：`utils/admin/adminGamification.ts`（类型 + API 工厂 +
  结构化数组的 JSON 字符串解码 + 规则文案翻译 + 稀有度色）与
  `composables/admin/useAdminGamification.ts`（页面壳唯一数据所有者，
  通过 injection key 共享；统一 runAction 包装含 Toast 与错误 reason 映射；
  插件未启用时判定 `status === 'enabled'` 失败关闭）。
- **注册与文案**：`config/adminModules.ts` 新增 `/gamification`
  （权限 `sforum.gamification.manage`）；i18n 中英双语
  （zh-CN.json / en-US.json 的 `admin.gamification.*` 与
  `admin.nav.gamification`）。

## Decisions

- 页面壳持有数据、标签页注入共享：避免同一个 useAsyncData key 被多个组件
  持有而在卸载时被清理（与 admin/ai.vue 同型）。
- 插件未启用时页面仍渲染，但写操作按钮禁用 + 顶部告警给出扩展到口：
  Host 侧路由不存在，任何写都会 404/503，失败关闭比报错好。
- 回填 UI 显式声明「覆盖而非累加」并给危险提示（敏感操作）。

## Verification

- `bun run typecheck` 通过（修了三个类型问题：稀有度颜色字面量类型、
  表单 submit/click 处理器不能返回 boolean、`SFAdminFixedTabNav` 的
  `ariaLabel` prop 名）。
- `bun test` 1070 pass / 0 fail。
- `node tests/validate-architecture-boundaries.mjs` 通过；所有新增文件
  均 < 320 行。

## Open / 阻塞

- **渲染态 Browser QA 未做**：本会话 3000 端口 dev server 与 8081 API
  均未运行，无法登录管理员会话做桌面 + 390x844 校验。需要 `bun run dev`
  + `./scripts/api-dev.sh` 后补。
- **`bun run build` 在本机 OOM（SIGABRT）**：释放内存到 8GB 可用后仍在
  vite client 阶段崩溃，非本次改动引入（新增仅 7 个文件）；typecheck 与
  全量单测已作为正确性证据，生产构建需在内存更宽裕的机器或 CI 验证。
- 权限目录展示文案与角色建议种子（admin）尚未落到 Core 权限目录。

# 2026-10-11 成就系统：后台侧栏入口（方案 A）Handoff

## Changed

- `apps/web/app/config/adminModules.ts`：`adminSidebarNavigation` 新增
  `admin.nav.engagement`（图标 `i-lucide-medal`）文件夹，其下挂
  `{ type: 'page', pageId: '/gamification' }`。此前 `/gamification` 只在
  `adminPageDefinitions` 里有页面定义、从未挂进侧栏，因此
  `/control-panel/gamification` 只能手输 URL 打开、点不到。
- `apps/web/i18n/locales/{zh-CN,en-US}.json`：新增 `admin.nav.engagement`
  （内容运营 / Content operations）。
- `apps/web/tests/admin/adminGamificationNav.test.ts`（新增）：断言页面定义
  存在且持有 `sforum.gamification.manage`、既非 professionalMode 也非
  operationsMode、侧栏恰好一条 `/gamification` 入口且归属 engagement 文件夹、
  权限被拒时该文件夹无可见子项（不渲染空分组）、两个 locale 都有标签且不与
  `gamification` 重名。

## Decisions

- **本轮走方案 A（Core 侧栏挂页面），方案 C（插件声明式后台导航位置）留作终态。**
  A 是 1 处配置 + 测试就能打通「插件启用 → 菜单可见 → 页面可用」的验收链；
  C 需要新契约（manifest schema / AdminSurfaceRegistry placement 校验 / 侧栏
  渲染位置与门控 / OpenAPI / 文档 / 测试），不塞进「补个入口」这一轮。
- 入口置于独立文件夹而非塞进「系统配置」：现有 `/ai` 虽是同类插件承载面但
  放在系统配置里；成长值属于运营而非站点配置，单独成组语义更准。
- 插件未启用时菜单仍常驻（A 的语义），页面走既有 `pluginDisabled` 失败关闭态
  并给出扩展到口；「插件禁用 → 菜单消失」要等 C。

## Verification

- `bun test`：**1103 pass / 0 fail**（154 文件）。
- `bun run typecheck`：通过。
- `node tests/validate-architecture-boundaries.mjs`：通过（1742 生产文件）。
- **未做渲染态 QA**：API 未运行（8080/8081 无监听；PostgreSQL 15432 / Redis 在
  跑，Nuxt dev server 在 3000）。后台壳需要会话，无法离线验证侧栏。

## Next

- 起 API（`./scripts/api-dev.sh`）→ 登录 super_admin → 确认侧栏出现「内容运营 →
  徽章与成长值」，并在桌面 + 390x844 下检查（M6 渲染态 QA）。
- 走 `/control-panel/extensions` 正常启用 `sforum.gamification`：确认
  actor-bound grant provision、路由 200/401（不再是 503）、权限目录出现
  `sforum.gamification.manage` 与 admin 角色建议。
- 公开徽章墙 `/badges/{username}` 的入口依赖插件启用的
  `forum.profile.tabs` → 资料页右栏；启用后一并复验。
- 方案 C 的契约草案与影响面清单（含 `pages/badges/[username].vue` 里硬编码
  `extensionId === 'sforum.gamification'` 的统一化）另开工单。

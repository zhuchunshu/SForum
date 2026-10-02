# 2026-10-02 AI 辅助平台 M0 内核 Handoff

## Changed

- 后端：`app/Support/AI/`（19 个源文件，含真实供应商 live 测试与审核质量探测）、
  `app/Models/AI/service.go`（权限感知服务层 + 密钥写入/状态）、
  `Http/Controllers/AI/` 六个管理端点、`Providers/ai.go` 与 bootstrap 装配。
- 前端：`pages/admin/ai.vue`（路由壳 + 页签）、
  `components/admin/ai/tabs/SFAdminAIProvidersTab.vue`（供应商与闸门）、
  `components/admin/ai/tabs/SFAdminAIObservabilityTab.vue`（用量与执行记录）、
  `composables/admin/useAdminAI.ts`、`config/adminModules.ts` 页面与侧栏登记、
  两个语言包的 `admin.ai.*` 与 `admin.nav.ai`。
- 契约与治理：`contracts/openapi/{paths,schemas}/ai.yaml`（含密钥端点）、
  入口登记、`catalog-identities.json`、重新生成的 15 个 V3 catalog、
  路由计数断言 343 → 349。
- 迁移 `202610020001_ai_gateway.sql`、`202610020002_ai_manage_permission.sql`
  （已在开发库应用）。本地化新增 `ai.settings_invalid` / `ai.credential_required` /
  `ai.profile_not_found`。

## Decisions

- **执行路径**：Core 协议翻译 + Host 受控出站，而非插件调用；`ai.provider`
  槽位保留给非标准协议。
- **失败调用计次不计费**；**延迟用单调时钟测量**（可注入时钟只负责窗口对齐）。
- **密钥录入走独立端点** `PUT /admin/ai/profiles/:profileId/credential`：
  明文只在一次请求里存在，写入 Secret Store 后不可读回；设置文档只保留
  `sforum.secret://` 引用。现有 SettingsLifecycle 只服务静态 Options 字段，
  而 AI profile 是动态集合，因此不复用。
- 面板第一版刻意**不做**用途级失败姿态与自动裁决开关：还没有任何 purpose 存在，
  那些开关会是空壳。
- 审核质量探测结论：提示词对「合理批评」与「技术争论」两类关键反例判断正确；
  但**置信度无区分度**（10 条全落在 0.95–1.00），因此 M2 的自动裁决不能用
  置信度阈值，必须改用触发原因分流。

## Verification

- 真实 DeepSeek 调用通过（含生产 SSRF 守卫）：`reply="好"`、`usage={17,1,0}`、
  `latencyMs=517`，trace 与用量落库。
- 审核质量探测 10 条：9 条与预期一致，唯一偏差是我的预期过宽（软性推广被判
  review 属合理）。
- `go test ./...` 121 包 ok、0 FAIL；`bun test` 920 pass / 0 fail；
  `bun run typecheck` 与 `bun run build` 通过。
- `generate.mjs --check`、`validate-docs.mjs` 通过。
- 运行时探测：`/api/v1/admin/ai/*` 返回 401（未认证）而非 404；
  `/control-panel/ai` 返回 302 到登录页，证明前端路由与 admin 守卫生效。
- **未完成**：渲染态浏览器 QA（BrowserSkill 的 `bsk` CLI 未安装，无法自动化）。
  面板的桌面与 390x844 移动端视觉验证仍需人工执行。
- **架构门禁当前失败，不由本工作引入**：`app/Models/Options/service.go` 由
  1119 长到 1122 行（另一条 `client_version_options` workstream），触发 legacy cap。
- 为解除目录生成阻塞，补登了该 workstream 新组件
  `SFAdminSiteClientTab.vue` 的稳定标识（纯机械补全，未改其代码）。

## Next

1. 人工渲染态 QA：`/control-panel/ai` 桌面与 `390x844`，覆盖标题/工具栏/页签
   对齐、供应商卡片在窄屏的折叠、密钥录入与状态徽章、用量与执行记录表。
2. M2 审核助手：用「可抽取」形态写第一个真实用途，据它提炼 purpose 注册表与
   middleware 链；自动裁决改用触发原因分流而非置信度阈值。
3. 待确认数值：每扩展每日配额、站点月度预算、自动裁决置信度阈值。

## Open Questions

- 第一版 AI 用途是否暴露按板块 / 按用户组的作用域，还是保持全局并预留
  `scope` 字段。
- 管理员是否可以直接编辑 Core 拥有的审核 prompt，还是只能做受限的追加。
- live 测试与审核探测是否常驻仓库（当前保留，默认 skip）。

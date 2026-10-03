# 2026-10-02 Session Handoff

## Changed

- 新增客户端版本策略公共选项 `client.minimum_version` /
  `client.recommended_version` / `client.update_notice`
  （`apps/api/app/Models/Options/client_version_options.go`：定义、推荐默认、归一化、
  读取回退、写校验；`types.go` 常量；`service.go` 合并默认值与读取回退；
  `service_normalize.go` 分发；`service_option_validation.go` 全量校验）。
- 后台入口：`apps/web/app/components/admin/settings/site/tabs/SFAdminSiteClientTab.vue`
  与站点设置页 `client` 固定标签注册（图标 `i-lucide-smartphone`），双语 i18n
  （`admin.settings.tabs.client`、`admin.settings.client.*`），含「恢复推荐默认（不限版本）」。
- 测试：`apps/api/app/Models/Options/client_version_options_test.go`
  （公共投影、默认空、权限拒绝、格式与长度、脏数据回退）、
  `apps/web/tests/admin/settings/clientVersionPolicy.test.ts`（选项名、恢复默认、
  未保存提示、标签注册、双语键）。
- 契约与文档：`GET /web-options` 描述补充 `client.*` 与维护键；
  双语 API 手册与双语后台指南补「客户端版本策略」。
- 生成物：`docs/extensions/v3/catalog-identities.json` 与
  `docs/extensions/v3/catalogs/ui-surfaces.md` 已包含新组件身份
  `core.component.admin.sf_admin_site_client_tab`（由并发的 AI 工作流在 19:04 的
  生成器运行时加入）。

## Decisions

- 走 `GET /web-options` 公共投影，不新增 `meta/client` 端点：避免第二个配置投影与
  人工维护的成员清单。
- 默认全部为空即不限制；服务端不计算 `updateRequired`，也不记录客户端版本。
- 不引入 `Deprecation`/`Sunset` 响应头机制：当前没有任何已弃用核心路由，先有真实
  消费者再设计头契约。
- 决策记录：`decisions/2026-10-02-client-version-policy-via-public-options.md`。

## Extraction And Ratchet

- 为了不突破 `apps/api/app/Models/Options/service.go` 的遗留大文件基线（1119 行），
  把 `normalizedDefaults` 里的部署期 `Defaults` 覆盖块（站点标识/语言/人机验证密钥与
  参数）抽成 `defaults_overrides.go` 的 `applyDefaultsOverrides`，service.go 降到
  1090 行，并在同一次改动里把 `tests/architecture-boundaries-baseline.json` 的
  cap 从 1119 下调到 1090（棘轮：不留下可增长空间）。

## Verification

- `go build ./...`、`go test ./app/Models/Options/` 通过；
  `bun test` 920 pass / 0 fail；`bun run typecheck`、`bun run build` 通过；
  架构边界与文档门禁通过；OpenAPI 引用（PyYAML 等价脚本）490 条全部解析。
- HTTP 实测：`GET /api/v1/web-options` 返回 185 个公共选项，含三个 `client.*`（均为
  空值）与 `site.maintenance.enabled/message`。
- **未完成**：后台标签页的渲染态 Browser QA（desktop + 390x844）没有做——本会话的
  BrowserSkill CLI 未安装（`bsk` 不在 PATH），`computer_*` 桌面工具报
  `shell.run is not a function`，仓库内也没有 Playwright。只有构建/类型/测试级证据。

## Next

- 渲染态 QA 待补：登录后台后打开 `/control-panel/settings?tab=client`，检查标签激活
  态、标题与工具栏对齐、390x844 无横向溢出、三个字段与「恢复推荐默认」按钮可用。
- `node scripts/v3-catalog/generate.mjs --check` 目前因 AI 工作流新增的两个后台组件
  （`SFAdminAIObservabilityTab.vue`、`SFAdminAIProvidersTab.vue`）缺少 reviewed
  identity 映射而失败；本轮未代为登记，等该工作流自行补齐。
- 原生推送（FCM/APNs）与第三方登录原生回跳仍未动。

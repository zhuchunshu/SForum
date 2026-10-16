# 2026-10-16 移除站内 AI 身份注入 Handoff

## Changed

- **网关不再向 system 追加任何宿主文本**：删除 `Support/AI/identity.go`
  （`IdentitySettings` / `ApplyIdentity` / `IdentityAppendix` / `IdentityAppendixVersion`），
  `Gateway.Execute` 只保留 `ApplyRedaction`。profile 的 `Label` / `Model` / `Protocol`
  从此只出现在运营者可见的执行记录里。
- **配置面整体收敛**：`Settings.Identity`、`updateSettingsRequest.Identity`、
  `settingsFromRequest` 的装配、以及只服务该字段的 `boolPointer` 全部删除；
  OpenAPI 删掉 `AIIdentitySettings` 定义与两处 `$ref`（读/写 schema 各一处）。
- **控制台**：删除 `SFAdminAIIdentityBlock.vue` 与 `SFAdminAIReplyPromptTab` 的挂载；
  `AdminAISettings.identity` 类型删除；`AIComparableSettings.identity` 删除，归一化
  改为无条件剥离历史残留的 `identity`（旧后端回传 / 旧草稿不再造成假 dirty）；
  zh-CN / en-US 各删 `admin.ai.identity{Intro,Disclose,Hint}` 四键。
- **组件身份退役**：`docs/extensions/v3/catalog-identities.json` 中
  `core.component.admin.sf_admin_ai_identity_block` 转 `state: "retired"`，向
  `catalog-retired-identities.json` 与 `catalogs/ui-retired-identities.json` 追加同一
  条墓碑，并同步删除 `catalogs/ui-surfaces.{json,md}` 与
  `Support/ComponentCatalog/core_catalog_gen.go` 中的活动条目。
- **提示词版本递增**：`forum-reply@6`、`forum-reply-triage@3`、`diagnose@3`；
  `Models/AIReply/generator_tools_test.go` 的字面量断言同步为 `forum-reply@6`。
- **测试反转**：删除 `Support/AI/identity_test.go` 与网关侧两个注入断言，新增
  `TestGatewaySendsCallerSystemUnchanged`（哨兵 profile，断言出站 system 与调用方逐字节
  一致且不含协议名/部署说明）与 `TestGatewayKeepsEmptySystemEmpty`；Web 侧
  `tests/admin/adminAIIdentity.test.ts` 由
  `tests/admin/adminAINoModelIdentity.test.ts` 取代（契约、控制台、文案、脏值剥离四层守卫）。

## Decisions

- `decisions/2026-10-16-remove-ai-identity-injection.md`：出站 system 只由调用方决定；
  不留「打开就把后台配置送进会话」的开关；模型对自身身份的回答只能来自模型自身
  （可能含糊或自称别的产品，这是有意接受的取舍）。
- 取代 `decisions/2026-10-15-ai-reasoning-separation-and-model-identity.md` 的第 2、3 条；
  该决策的推理正文分离、回复预算跟随 profile、确定性空回复不重试三条继续有效。

## Verification

- `cd apps/api && go build ./...` 通过；`go test ./...` 全仓库 127 个包全过
  （其余包均为 no test files）。
- `cd apps/web`：`bun test` 1205 全过 / 0 fail；`bun run typecheck` 通过；
  `bun run build` 通过（首次失败是 Node 默认 2 GB 堆在 SSR 阶段 OOM，
  `NODE_OPTIONS=--max-old-space-size=6144` 后成功，与本次改动无关）。
- `node scripts/v3-catalog/generate.mjs --check`：`V3 P0 catalogs are current (16 files)`，
  即删除组件后手工维护的目录与生成器输出零漂移。
- 浏览器 QA（air 热重载的 dev API + 手动 web dev server）：
  `/control-panel/ai?tab=replyPrompt` 桌面无身份区块、无缺键残留（`身份声明` /
  `discloseModel` / `admin.ai.identity` 全文检索为零）；390x844 视口仿真下
  `scrollWidth == clientWidth`（无横向溢出），页签两行折行、区块堆叠正常。
- 既有红状态（**非本次引入**）：`node tests/validate-v3-p0-catalogs.mjs` 失败
  （UI 计数 319 vs 脚本期望 317）。在 `git worktree` 的干净 HEAD 上复现同样失败，且
  HEAD 更早就失败于该脚本第 87 行；当前工作区还包含另一条未提交工作流的改动
  （插件页面区域、扩展、论坛等 130+ 文件），全量 gate 在这条基线上本就不自洽。

## Next

- 运行时证据（可选）：真实回复或控制台诊断调用抓一次出站 system，确认只含调用方文本。
- 遗留自 2026-10-15：triage 用途的 token 预算按用途配置（推理模型下 autonomous 恒 SKIP）。

## Open Questions

- 无。上一版遗留的「身份公开性是否按用途区分」随开关移除而消失。

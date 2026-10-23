# 2026-10-23 AI tool visibility single-owner handoff

## Changed

- `apps/web/app/components/admin/ai/blocks/SFAdminAIPurposesBlock.vue`：移除工具
  复选框、手动输入与「跟随声明/自定义」按钮；保留失败姿态、只读生效数量、声明/
  已自定义徽标与 `?tab=tools` 深链。`restorePurpose` 仍会同时清掉姿态与工具覆盖。
- `apps/web/app/components/admin/ai/tabs/SFAdminAIToolsTab.vue`：用途列头在有工具
  覆盖时显示恢复按钮（`resetPurposeTools`，保留 `failurePosture`）；未注册引用整段
  交给新区块；intro 文案改为同时描述可见性与英文说明。
- `apps/web/app/components/admin/ai/blocks/SFAdminAIToolReferencesBlock.vue`（新）：
  未注册引用逐条清理 + 「先配策略、后装插件」的手动点名；只接受未注册名字，遵守
  `MaxToolsPerRequest`，移除路径不受上限限制。
- `apps/web/app/utils/admin/adminAI.ts`：`purposeToolsAreOverridden` 改为「`tools`
  字段存在即接管」，与后端 `PurposeSettings.ToolAllowlist()` 对齐；删除
  `purposeToolOptions`。
- `apps/web/app/utils/admin/adminAITools.ts`：新增 `resetPurposeTools`。
- `apps/web/app/pages/admin/ai.vue`：新增 `watch(() => route.query.tab)`。`router.replace`
  只改地址栏、不重新挂载页面；没有这个 watch，用途页签的深链只会更新 URL，内容仍
  停在原页签。
- i18n：`admin.ai.tools.manualAdd.*`、`resetPurpose*`、`tooMany`、
  `unregisteredCount`、`purposes.tools.summary/managedHint/manage`、
  `purposes.restoreHint`；删除旧用途页签工具编辑器的文案键。
- 测试：更新 `adminAIPurposes`、`adminAIToolsTab`、`adminAIToolsUtils`；新增
  `adminAIToolReferences`。
- V3 目录：`catalog-identities.json` 增加
  `core.component.admin.sf_admin_ai_tool_references_block`，重生成
  `ui-surfaces.json|md`。

## Decisions

- `decisions/2026-10-23-ai-tool-visibility-single-owner.md`：工具可见性单一编辑面、
  只读摘要、列头恢复、手动点名、`{"tools":{}}` 前后端对齐、后端结构保持不变。

## Verification

- `bun test tests/admin/{adminAIPurposes,adminAIToolsTab,adminAIToolsUtils,adminAIToolReferences}.test.ts`
  → 92 pass / 0 fail。
- `bun test`（apps/web）→ 1566 pass / 0 fail。
- `bun run typecheck` → exit 0。
- `node tests/validate-architecture-boundaries.mjs` → passed（1922 files，
  203 above the 500-line review threshold）。
- `node scripts/v3-catalog/generate.mjs` + `node tests/validate-v3-p0-catalogs.mjs`
  → 376 routes / 338 UI surfaces / 99 traceability rows。
- `./scripts/test.sh --fast` → api / web / static 三线 PASS。
- `./scripts/test.sh`（完整门，含生产 web build）→ api 156s / web 271s / static 82s，
  三线 PASS。QA 用的 dev servers 已在跑门前停止，3000/8080 已释放。
- 渲染 QA（真实 dev servers，管理员会话）：
  - `1633x810`：用途页签 `documentElement.scrollWidth === 1633`，无工具复选框，
    5 个「在模型工具中管理」深链；点击后 URL 与内容都切到 `?tab=tools`。模型工具
    页签 `scrollWidth === 1633`，`forum.reply` 列头出现恢复按钮。
  - `390x844`：两个页签 `scrollWidth === 390`；矩阵在自身容器内滚动
    （1120 vs 314），手动点名输入宽 314、区块宽 348；恢复按钮的 `aria-label` 与
    `title` 都存在。
  - 恢复：点击 `forum.reply` 列头恢复后按钮消失、面板变脏；强制 reload 未保存的
    草稿被丢弃，存储文档仍是原来的 10 个工具。
  - 手动点名：向 `forum.digest` 添加 `qa-unregistered-tool` → 引用区出现条目、
    保存后 reload 仍在；「从所有用途移除」→ 保存后条目消失。数据库
    `ai_settings.document->'purposes'` 已回到 QA 前状态（`forum.reply` 10 个工具，
    `forum.digest` 无覆盖）。
  - 截图：`tmp/ai-tool-single-owner-390-purposes.png`、
    `tmp/ai-tool-single-owner-390-tools.png`（完整 390）；
    桌面截图受 WM 500px 窗口钳制，仅左半可见，宽度证据以上面的测量为准。

## Next

- 如需要，补一条 per-tool call/failure 统计视图（沿用 2026-10-19 的 follow-up）。
- 评估是否把 `purposeCatalogEntry.toolsAllowed/toolsOverridden` 从后台 API 读模型
  中移除（本轮保留给第三方客户端）。

## Open Questions

- 用途页签是否需要在摘要里直接列出工具名（当前放在 `title` 属性里，完整清单在
  模型工具页签）。
- 后台 API 的 `purposeCatalogEntry.toolsAllowed/toolsOverridden` 目前控制台不再
  直接消费，是否保留给第三方客户端（本轮保留）。
- 未注册名字的手动点名目前只能对已声明用途生效；未声明用途仍是 API-only 形状。

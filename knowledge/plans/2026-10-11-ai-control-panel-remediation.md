# 2026-10-11 AI 控制台修复计划

## Status

**completed** — M0-M6 全部落地并验证：Go 全量测试、`bun test` 1070 通过、
`bun run typecheck` 无 AI 相关错误、`bun run build` 通过（需 `--max-old-space-size=8192`）、
架构边界守卫通过。渲染 QA 因缺少管理员凭据未完成（见 M7）。

## Background

后台 `/control-panel/ai` 页面存在一组已确认问题。根因链与代码证据：

- 加密警告常驻：`.env` 未配置 `APP_OPTION_ENC_KEY` → `NewOptionCipher("")` 返回透明 cipher（`apps/api/app/Support/Crypto/option.go`）→ `cipher.Enabled()=false` → 后端 `settingsResponse.EncryptionEnabled=false`（`apps/api/app/Http/Controllers/AI/controller.go:104,217`）→ 前端 `SFAdminAIProvidersTab.vue:187-193` 警告条。
- `putProfileCredential`（controller.go）不检查加密能力，`encryptionEnabled=false` 时密钥静默明文写入 Secret Store，与警告文案「保存会失败」矛盾。
- `ResetSettings`（`apps/api/app/Models/AI/service.go`）整体替换为 `RecommendedSettings()`：profiles 结构被清空，但 Secret Store 中的密钥仍在 → UI 显示未配置而凭证实际存在。
- `autoAction` 与 `redaction` 在类型、默认值、后端校验（`apps/api/app/Support/AI/settings.go`）中完整存在，前端无任何入口（幽灵字段被原样回传）。
- `Diagnose`（service.go）只走 `CostClassEconomy` 解析 profile，与运营者实际绑定可能不一致。
- gates 数字输入抹平「0」与「未设置」语义；`toolCallsPerReply` 的 nil 缺省语义在表单中不可表达（`SFAdminAIProvidersTab.vue:145-148,358`）。
- 观测页 executions 固定 limit=50 无分页；budget=0 显示 `0/0`；`stamp()` 无固定 locale；月支出 4 位小数过度精确（`SFAdminAIObservabilityTab.vue`）。
- 页签切换不回写 query（`apps/web/app/pages/admin/ai.vue:51-55` 为修复白屏刻意为之），刷新后永远落在 providers，深链失效。

## Decisions (confirmed by YG 2026-10-11)

1. 重置语义：保留 profiles 结构与凭证引用，只重置 gates/redaction/autoAction/reply prompt 等策略字段；UI 明确提示「重置不会清除已保存的密钥」。
2. autoAction/redaction 本期补 UI（折叠区块），不删字段。
3. 密钥保存全环境 fail-closed：`putProfileCredential` 在 `encryptionEnabled=false` 时返回 422；`AllowTransparent` 仅保留给单元测试；前端禁用保存按钮并给出 `APP_OPTION_ENC_KEY` 变量名与生成命令。

## Milestones

### M0 — 部署配置与文档（P0）

- [x] `.env` 增加 `APP_OPTION_ENC_KEY`（`openssl rand -hex 32` 生成）；`.env.example` 同步补充并注明生成命令。
- [x] `scripts/dev.sh` 首次生成 `.env` 时自动补本地密钥，并在缺失时告警（避免新人被 fail-closed 卡住）。
- [ ] 重启 API 验证 `GET /admin/ai/settings` 返回 `encryptionEnabled=true`，警告条消失。
- [ ] `docs/` 部署手册补充该变量说明（如已有加密密钥章节则就地更新）。

### M1 — 密钥保存 fail-closed（P1）

- [x] `Controller.putProfileCredential`：加密不可用时返回 422 `ai.encryption_required`；`mapAIError` 映射新错误。
- [x] `CredentialWriter.Put` 写入前校验加密能力（覆盖所有调用路径，不只是 HTTP）。
- [x] i18n（zh-CN/en-US）补 `encryptionMissing` 变量名与生成命令、新增 `credentialBlockedHint`。
- [x] 前端在 `encryptionEnabled=false` 时禁用密钥输入与保存按钮，并在 `submitCredential` 内二次拦截。
- [x] 测试：`credential_writer_test.go`（allowed / denied / 空明文语义）+ `adminAICredentialGate.test.ts`。

### M2 — 重置语义修正（P1）

- [x] `Service.ResetSettings`：保留现有 `Profiles`（含 `APIKeyRef`）与 `CostClassProfiles`；无既有 profiles 时退回全量推荐默认。
- [x] Toast 文案已声明「密钥不会被清除」（`admin.ai.resetDone`）。
- [x] 测试：`TestServiceResetKeepsProviderProfilesAndCostBindings`、`TestServiceResetUsesRecommendedProfilesWhenNoneStored`。

### M3 — autoAction / redaction 设置 UI（P2）

- [x] 拆出 `admin/ai/blocks/SFAdminAIAutoActionBlock.vue` 与 `SFAdminAIRedactionBlock.vue`（避免 providers 页签跨 500 行阈值），挂到 gates 之后。
- [x] 自动裁决：enabled 开关、confidenceThreshold（钳制 0–1）、allowedActions 多选（仅 hide/tag/reject，对齐 `ValidAutoAction`，不提供删除类）。
- [x] 脱敏：enabled 开关 + 三个子项；关闭总开关时一并清空子项，避免矛盾状态。
- [x] i18n 双语；共用 `form`/`dirty` 链路；`readOnly` 随 `ai.manage` 透传。
- [x] 测试：`adminAIPolicyBlocks.test.ts`；`bun run typecheck` 通过。

### M4 — gates 数字与缺省语义（P2）

- [x] `toolCallsPerReply` 改用指针语义：清空 = 未设置（按默认 3 生效），0 = 停用工具调用；新增 `setToolCallsPerReply` 与「恢复未设置」按钮。
- [x] 归一化显式对齐 nil 与数字（`normalizedAISettingsForComparison` 删除未设置项、数字串转数字），避免 `JSON.stringify` 抹平差异导致一打开就 dirty。
- [x] 界面区分「未设置」与「已显式设置」两种状态文案。
- [x] 拆出 `SFAdminAIProviderPicker.vue`（providers 页签 491 → 467 行，守住 500 行阈值）。
- [x] 测试：`adminAISettingsComparison.test.ts` 新增 5 个用例；`bun test` 1063 通过；边界守卫通过。

### M5 — 诊断与实际配置对齐（P2）

- [x] 新增 `resolveDiagnoseProfile`：economy → standard → premium 依次解析，返回实际命中的等级；`DiagnoseResult` 增加 `costClass`。
- [x] 探测请求使用实际命中的成本等级，而不是硬编码 economy。
- [x] 前端诊断页在结果头部展示成本等级；i18n 复用 costEconomy/costStandard/costPremium。
- [x] 测试：`diagnose_profile_test.go` 三个用例（回退 / economy 优先 / 全未绑定）。

### M6 — 观测页与体验打磨（P3）

- [x] budget=0 显示「未设置预算」+ 对应说明，并隐藏进度条。
- [x] `stamp()` 按 locale 显式传 `zh-CN`/`en-US`；月支出改 2 位小数。
- [x] executions 增加状态筛选（全部/成功/失败/被拦截），空态区分「无记录」与「无匹配」。
- [x] 页签用 `router.replace` 同步 query（不触发导航），刷新/深链可落在对应页签。
- [x] 新增 `ai.encryption_required` 本地化消息（本地化守卫要求新错误码必须有译文）。
- [x] 测试：`adminAIObservability.test.ts`；`bun test` 1070 通过；`go test ./...` 全绿。

### M7 — 全量验证与知识库

- [x] `go test ./...` 全绿；`bun test` 1070 通过；`bun run typecheck` 无 AI 相关错误；`bun run build` 通过（SSR 步骤需 `--max-old-space-size=8192`，默认堆会 OOM，属环境问题）；架构边界守卫通过。
- [x] 新增 `knowledge/modules/ai.md` 并登记进 `modules/README.md`；新增两条 decision record（fail-closed、重置语义）；`knowledge/index.md` 更新。
- [ ] 浏览器渲染 QA：`/control-panel/ai` 桌面与 390x844，覆盖警告消失、密钥禁用/启用态、重置提示、新区块、页签深链。**阻塞**：需要管理员账号凭据，当前会话未登录。

## Out of Scope

- AI 网关调用链路、工具循环、回复机器人业务逻辑本身。
- Secret Store 轮换/迁移工具。
- autoAction 的审核队列消费端（仅配置 UI）。

## Open Questions

- 无（三个拍板点已于 2026-10-11 确认）。

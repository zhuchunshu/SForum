# 2026-10-19 Session Handoff — 控制台 AI 保存 422 与上限放宽

## Changed

### 根因（`/control-panel/ai` 保存偶发 422）

`PUT /admin/ai/settings` 只有两处 422：JSON 解码失败、`Settings.Validate()` 失败。
六条触发路径全部落到这两处：

1. `maxTokens` / `timeoutMs` 用 `v-model.number`，清空后模型里是空串 `""`，
   载荷变成 `"maxTokens":""`，Go `int` 解码失败；填小数（12.5）同理。
   注意服务端对 `0`/`null` 是宽容的（`Normalized()` 会补默认值），唯独空串死在
   解码层——这个不对称正是「有时候」的来源。
2. 闸门/预算/单价只有下界钳制，没有上界 → 越过天花板被 422。
3. `isProfileComplete` 只判非空，不校验 URL 形状与字节长度 → 启用即 422。
4. 自动裁决开启且动作全空 → 422（界面只黄字提示，保存按钮仍可点）。
5. 数值入口有四套写法（`setNumber`/`setBudgetYuan`/`setMicro`/`v-model.number`）。
6. 错误只显示 `${error}`（ofetch 的 422 文本），服务端也把所有校验折叠成一个
   没有 `fields` 的 `ai.settings_invalid`。

### 改动

- `Support/AI/limits.go`：天花板按模型能力重定标（输出 128k、输入总预算 12 MiB、
  工具步数 12、月度预算 1000 万元等），新增 `LimitsView` + `CurrentLimits()`。
  **默认值不变**。
- `Support/AI/validation.go`（新文件）：`Settings.Issues()` / `Profile.Issues()` /
  `GateSettings.Issues()` 返回 `{Field, Code}`；`Validate()` 退化为包装，对外错误
  类型不变。`settings.go` 因此回到 358 行。
- `Http/Controllers/AI`：`decodeSettingsRequest` 区分 `ai.settings_malformed`
  （字段类型不对）与 `ai.settings_invalid`（取值/形状不合法），两者都带
  `data.fields`；`settingsPayload` 下发 `limits`。
- `sdk/plugin/v2`：镜像常量同步 256 / 128_000；`DefaultMaxMessageBytes` 4→16 MiB。
- `Support/Extensions`：插件进程 `GOMEMLIMIT` 48→96 MiB。
- `Localization`：`ai.settings_malformed` 与 `ai.field.*` 两语言。
- 控制台：`coerceInteger` / `coerceNumber` / `profileIssue` / `isValidEndpointUrl`
  收口数值与供应商校验；保存前拦截并显示原因；`apiErrorMessage` + `apiErrorFields`
  取代 `${error}`；`limits` 来自服务端。
- 顺带修复两处数据缺陷：`restoreDefaultPrompt` 丢 `triggerMode`；PUT 回传历史
  `reply.tools` 让「恢复默认」永远不生效（提交前剥掉）。

### 契约

`contracts/openapi/schemas/ai.yaml` 新增 `AISettingsLimits` 并挂到
`AISettingsPayload.limits`；`paths/ai.yaml` 的 PUT 422 补字段级说明。
`ruby scripts/validate-openapi-refs.rb` 通过（2864 refs / 61 files）。

### 测试

- `app/Support/AI/limits_test.go`：SDK 镜像一致性、传输容量 ≥ 输入预算、
  上限自洽、1M token 前提、`CurrentLimits()` 覆盖全部字段。
- `app/Support/AI/settings_issues_test.go`：字段路径/原因码、草稿供应商不拦、
  URL 两种失败可区分、label 按字节计。
- `app/Http/Controllers/AI/controller_test.go`：空串/小数 → `ai.settings_malformed`
  且不回显原值；越界 → 字段级 `ai.settings_invalid`；推荐文档可保存。
- `apps/web/tests/admin/adminAISaveValidation.test.ts`：15 条，含「不再使用
  `v-model.number`」与「上限来自服务端」的静态断言。
- `go test ./app/Support/AI/... ./app/Http/Controllers/AI/... ./sdk/...`、
  `bun test tests/admin`（345 pass）通过。

## Decisions

- `decisions/2026-10-19-ai-context-and-budget-ceilings.md`（新增）：放宽的数值、
  依据、三处连带约束与「放宽不削弱安全边界」的论证。
- `decisions/2026-10-17-plugin-process-memory-limits.md`：追加 96 MiB 修订。

## Next

- 插件路径的大 payload 实测（12 MiB 请求 + 96 MiB GOMEMLIMIT），确认无连续 GC 抖动。
- 图片上限（4 张 / 单张 2 MiB）与输入总预算的耦合关系需要单独决策。
- `node tests/validate-architecture-boundaries.mjs` 在本工作区有既存失败
  （Extensions controller/client、`adminExtensions.ts`、Pages registry 等，
  均与本次改动无关），未在本会话内处理。

## Open Questions

- `PATCH`/`PUT` 之间是否需要 revision 乐观锁（目前整份文档覆盖，无冲突检测）。

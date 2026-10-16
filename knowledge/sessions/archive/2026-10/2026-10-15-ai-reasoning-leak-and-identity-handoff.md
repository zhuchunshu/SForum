# 2026-10-15 AI 思考内容外泄与模型身份 Handoff

## Changed

- **推理正文不再被当成回答**：`WireResponse` / `CompletionResult` / `RunResult` /
  `aireply.CompletionOutput` 新增独立 `Reasoning` 字段；`Text` 只来自回答字段
  （openai `content`、anthropic `text` block）。修复前 `reasoning_content` /
  `thinking` 会在正文为空时被提升成 `Text`，于是回复机器人把思维链发成评论、
  控制台把它当测试回复并报「连接正常」、结果缓存存 10 分钟。
- 新增 `Support/AI/wire_reasoning.go`：`stripInlineReasoning` 剥掉内联进正文的
  `<think|thinking|reasoning|analysis>` 块（成对块反复剥离；未闭合的截断块整体
  丢弃；只认正文开头的标签，不动正文中间的字面内容）。
- `result_cache.go`：`resultCacheVersion` 1 → 2，丢弃把思考存在 `Text` 里的旧载荷。
- **自我身份注入**：新增 `Support/AI/identity.go`（`IdentitySettings` /
  `ApplyIdentity` / `IdentityAppendix` / `IdentityAppendixVersion`）。`Gateway.Execute`
  在脱敏之后、缓存键之前注入，因为只有那里同时知道本次生效的 profile 与站点是否
  允许公开。`settings.Identity.DiscloseModel` 是指针，nil = 公开。
- 提示词版本递增：`forum-reply@5`、`forum-reply-triage@2`、`diagnose@2`（身份声明
  改变了每个用途实际生效的系统指令）。
- **回复输出预算来自 profile**：`ReplyPurposeParameters.MaxTokens` ← 该用途成本等级
  绑定 profile 的 `Defaults.MaxTokens`；生成器不再写死 512。`Profile.Defaults.MaxTokens`
  此前被校验、被控制台暴露却从未被读取。
- **确定性空回复不重试**：`GenerateReplyWorker.Work` 把 `ErrEmptyReply` 翻译成
  `river.JobCancel`；错误带 `stop_reason=` 与 `reasoning=` 便于诊断。
- 诊断提示精确化：`diagnoseEmptyReplyHint(stopReason, reasoningOnly)` 区分「只产出
  思考」与「什么都没产出」。
- 控制台：新增 `SFAdminAIIdentityBlock`（挂在「回复提示词」页签），
  `AdminAISettings.identity`、`normalizedAISettingsForComparison` 的 identity 归一化、
  zh-CN/en-US 文案、OpenAPI `AIIdentitySettings`（读/写两个 schema 都带上）。
- `updateSettingsRequest` / `settingsFromRequest` 接受 `identity`。

## Decisions

- 见 `decisions/2026-10-15-ai-reasoning-separation-and-model-identity.md`：
  推理正文永不作为回答（失败关闭而非降级）；身份由网关注入；公开性可配且缺省公开；
  回复预算跟随 profile；确定性空回复取消而不是重试。

## Verification

- `go test ./...`（见下方 Next 的待补记录）、`bun test` 1151 全过、
  `bun run typecheck`、`ruby scripts/validate-openapi-refs.rb`、
  `node tests/validate-architecture-boundaries.mjs` 全过。
- 新增/反转测试：三个「回退读思考」断言反转为「思考不得成为正文」；
  内联标签剥离（成对/重复/未闭合/中间标签/大小写）；身份注入（公开/不公开/空 system/
  无名 profile/nil 缺省）；网关出站 system 断言（断言落在 system 字段而不是整段 body
  ——请求体的 model 字段本来就带模型名）；reasoning-only 不发布且 `ErrEmptyReply`
  带诊断信息；输出预算跟随 profile 与回落；`JobCancel` 与供应商失败重试的区分；
  web 侧开关缺省、归一化与双语键。

## Next

- 管理台浏览器 QA：`/control-panel/ai?tab=replyPrompt` 桌面 + 390x844（本次改动新增了
  一个设置块，需要渲染证据）。
- 可选后续：把身份声明做成按用途可选（当前对所有用途统一注入）；控制台展示生效的
  身份声明文本；triage 的 token 预算按用途配置（推理模型下 autonomous 恒 SKIP）。

## Open Questions

- 身份公开性是否需要按用途区分（例如插件用途不希望暴露模型，而面向成员的回复需要）？
  当前是站点级单一开关，理由是「模型知道自己是谁」是普遍事实，而「对成员公开」是
  运营者的策略选择。

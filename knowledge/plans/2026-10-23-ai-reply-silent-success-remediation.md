# AI 回复「每次成功、一次没回」诊断与修复

Status: **active（M1–M4 已落地，剩浏览器 QA / eval 基线 / 保留策略）**
Started: 2026-10-23
Decision records:
`decisions/2026-10-23-empty-completion-is-a-failure.md`、
`decisions/2026-10-23-reply-outcome-ledger.md`

## Problem

控制台的 AI 执行记录对 `forum.reply` 显示一整列 `succeeded`，`0 ms`、`0 / 0`
token、`forum-reply@10`，而社区里一条回复都没有。

读代码得到的结论（每条都能指向具体行）：

1. `succeeded` 描述的是**一次模型调用**，不是**回复已发布**。两者从未被连起来。
2. `0 ms + 0 / 0 token` 只有一种来源：结果缓存命中（`Support/AI/gateway.go` 的
   命中分支不写延迟、token、花费）。另一种现实可能是一条本机反代在 1 ms 内回
   200 空补全（wire 层允许 `content` 为空、`usage` 缺失）。
3. 空补全被记成成功，并被写进 Redis 结果缓存（`store()` 无条件执行）：一次坏
   运气于是变成十分钟的确定性失败——之后每个同键请求都在 0 ms 内拿回同一个空正文。
4. 生成器拿到空正文 → `ErrEmptyReply` → `river.JobCancel`，只写 slog，
   **不写执行记录**。于是「模型没产出」在控制台不可见。
5. 列表里看不出缓存命中；`cacheHit` 只在展开的 run 时间线里出现。

## Milestones

| # | 目标 | 状态 | 主要落点 |
| --- | --- | --- | --- |
| M1 | 空补全判失败、永不入缓存、命中行自我说明 | **done** | `Support/AI/empty_completion.go`（新）、`gateway.go`、`store()` 防御 |
| M2 | 回复结果独立成账 + 管理端读面 | **done** | `Support/AI/reply_outcome*.go`、迁移 `202610250001`、`Models/AIReply/outcome.go`、`GET /admin/ai/reply-outcomes`、OpenAPI |
| M3 | 幂等 + 缓存徽标/筛选 + 控制台「回复结果」区块 | **done** | `Generator.alreadyPosted`、`SFAdminAIObservabilityTab.vue`、`replyOutcomes.ts`（新）、i18n |
| M4 | 排查工具与文档 | **done** | `sforum ai:reply`（新）、`knowledge/modules/ai.md`、两份决策记录 |
| M5 | 浏览器 QA（1633×810 / 390×844）、`sforum ai:eval --diff`、结果表 30 天保留 | **open** | 见下 |

## Deviations from the approved plan

- **D1（缓存命中行不抄 token/花费）**：计划写的是「token/spend 取自缓存载荷」，
  实现改为保持 0，并在界面用徽标 + 一行说明解释。理由：`ai_executions` 的 token
  与花费是被求和看成本的口径，把被缓存那次的用量抄到命中行会让同一笔钱算两次。
  命中行的正确语义是「本次消耗 0」。
- **D2（网关在空补全时返回部分结果）**：计划没写。实现里 `Execute` 在空补全失败
  路径带回 `StopReason` / `Reasoning` / `Usage`，否则诊断页无法区分「预算被思考
  吃光」与「什么都没产出」。正文仍然为空，调用方不得发布。
- **D3（现有测试语义变更）**：`TestGatewayKeepsReasoningOutOfResultText` 原来断言
  「空正文算 succeeded」，改为 `TestGatewayReportsReasoningOnlyReplyAsEmptyCompletion`
  断言 failed + `empty_completion`；诊断的 `ok=true + reason=empty_reply` 两层含义
  保持不变（`Models/AI/service.go` 显式处理该原因码）。

## Evidence

- 新增/更新 Go 测试：`Support/AI/empty_completion_test.go`（3 例）、
  `Support/AI/reply_outcome_postgres_test.go`（真实 PostgreSQL，独立 schema）、
  `Models/AIReply/outcome_test.go`（4 例）。
- 新增 Web 测试：`tests/admin/adminAIReplyOutcomes.test.ts`（结果码文案、颜色、
  落点、两种 locale 键完整性）。
- 实机：`sforum ai:reply --topic-id 1 --confirm` 在 dev 库跑通一次真实生成
  （1823/223 token、216 字正文、`stop_reason=end`），证明工具链与提示词未回归。
- `node tests/validate-architecture-boundaries.mjs` 通过（1928 文件，204 条超阈值）。
- 迁移已在 dev 库应用（`202610250001`）。
- `./scripts/test.sh --fast` 三线 PASS（api 165s / web 56s / static 16s）。
- V3 目录：`catalog-identities.json` 增 `core.route.ai.list_reply_outcomes`，
  重生成后 377 路由 / 338 UI 表面 / 99 追溯行；`registry_test.go` 审核计数同步。

## Open items (M5)

1. 浏览器 QA：`/control-panel/ai?tab=observability` 在 1633×810 与 390×844 下确认
   徽标、筛选、回复结果区块无溢出；需要管理员会话。
2. `sforum ai:eval --diff` 与基线比对一次（会真实调用，消耗站点配额）。
3. `ai_reply_outcomes` 的 30 天保留任务（v1 只记录了约定，没有清理器）。
4. 若运营者的实例确认是「供应商空补全」而非缓存重放：把
   `profile.Defaults.MaxTokens` 调到模型声明上限以内（本机反代对
   `Qwen3.8-Flash` 声明 32768，而 profile 写的是 128000），并考虑在 providers
   页签对「预算高于模型上限」给出提示（尚未实现，属 M5 增强）。

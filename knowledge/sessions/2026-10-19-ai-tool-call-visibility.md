# 2026-10-19 AI 工具调用可见性（W1 + W2）Handoff

## Changed

- **W1 · 提示词工具面**：`Support/AI/reply_prompt.go` 新增
  `BaseSystemPrompt()` / `ComposeReplySystemPrompt(base, toolSection)` /
  `RenderToolSection([]ToolDefinition)`；`EffectiveSystemPrompt()` 改为
  `Compose(base, "")`（语义不变）。默认提示词补「站外事实」段；
  `CoreReplyPromptVersion` → **`forum-reply@9`**。
  `Support/AI/tool_loop.go` 新增 `Orchestrator.ToolSection(ctx, costClass, names)`
  （与 `Run` 同一套闸门：步数预算 > 0、descriptors 非空、profile 支持 tools）。
  `Models/AIReply/generator.go` 新增 `WithToolSectioner`；`context.go` 的
  `replyPromptExtras` 增 `toolSection`；`Models/AI/service.go` 的
  `ReplySystemPrompt` 改为返回 `BaseSystemPrompt()`。
- **W2 · 轨迹落库**：迁移 `202610190005_ai_tool_calls.sql`（已应用）、
  `Models/AIReply/tool_calls_store.go`（`PostgresToolCallStore` +
  `ToolCallRecorder`/`ToolCallReader`，`ON CONFLICT (comment_id, seq) DO NOTHING`）、
  `generator.go` 的 `WithToolCalls` + `recordToolCalls`（发布成功后写）。
- **公开读**：`Models/Forum/tool_calls.go`（`ToolCallEntry` / `ToolCallReader` /
  `publicToolCalls` 过滤四种拒绝结局）、`Forum/citations.go` 的
  `CitationList.ToolCalls` + `CitationService.WithToolCalls`、控制器与 provider 的
  `WithToolCallReader` 链、`bootstrap/api_assembly_domains.go` 一行装配。
- **契约**：`contracts/openapi/schemas/forum.yaml` 新增 `CommentToolCall` 与
  `CommentCitationList.toolCalls`（`required` 仍只有 `items`）；`paths/forum.yaml`
  的 `listCommentCitations` 描述改为「AI 活动」并写明公开过滤规则。
- **前端**：`citations.ts` → `aiActivity.ts`（新增 `visibleToolCalls` /
  `toolArgumentSummary` / `toolCallsVisible` / `activityVisible`）；
  `SFCommentCitations.vue` → `SFCommentAIActivity.vue`（一个 `<details>`、两个分区）；
  `SFComment.vue` 改挂载；i18n `topicDetail.aiActivity.*` 6 键双语；
  测试文件改名并补工具调用用例。
- **eval 一致性**：`cmd/sforum/ai_eval.go` 的参数源改返回 `BaseSystemPrompt()`，
  fixture 生成器注入 `WithToolSectioner(orchestrator)`——否则 eval 的 system 缺工具段，
  基线测的不是生产实际发出的东西。
- **V3 组件目录**：`catalog-identities.json` 把
  `core.component.shared.sfcomment_citations` 的 `source` 搬到新路径（**id 与契约不动**，
  这是 V3 的「显式 move」语义：id 是稳定标识，不必跟随文件名），并重新生成 18 个
  catalog 文件。**顺带补了两个 HEAD 上就漏登记的组件**
  （`SFEditorTableMenu.vue` / `SFEditorTablePicker.vue`，来自 `d9e3dc4f8` 的表格提交）：
  `contracts-gate.sh` 在干净 HEAD 上就因此报红，不补就无法验证本次改动。

## Decisions

`decisions/2026-10-19-ai-tool-call-visibility-and-prompt-tool-section.md`（六条）：
轨迹落新表而非扩 `ai_executions`；幂等键 `(comment_id, seq)`；公开投影去掉
`result_digest` 与四种拒绝结局；不新增路由而是 `/citations` 增量 `toolCalls`；
工具段由 Orchestrator 渲染、插在基础提示词与安全尾注之间；`ReplySystemPrompt`
只返回基础提示词。

## 关键发现（**这是本轮最重要的结论**）

**机器人在切换供应商之后从未成功调用过任何工具，与本次改动无关。**

证据链：

| 时间 | 事件 |
| --- | --- |
| 08:56 / 08:58 | `e7bc4532…` / `78bf401c…` 各有 3 行 `ai_executions` = 2 次工具往返 |
| 21:45 | `ai_settings` 被改为 `custom` → `http://127.0.0.1:3066/v1`（`agent2api`），`glm-5.3-flash` |
| 21:46 起（含 23:28 的截图 run `8681e832…`、23:57 的 run `5b6065f9…`） | 每个 `forum.reply` run 都只有 **1** 行 `ai_executions` = 零工具调用 |
| 00:01 / 00:02 | 同一 fixture `answer-searchable-channel` 从基线 3 次模型调用 / 3 次工具调用掉到 **1 / 0**；把新加的「站外事实」段整段删掉再跑，仍是 **1 / 0** |
| 00:01 | `~/.agent2api/debug/last-request.json` 显示请求**确实带了** `tools`（6 条）与 `tool_choice: auto` |

结论：SForum 侧的线格式是对的（单元测试 `TestOrchestratorRunsToolThenAnswers` 也钉住
出站 body 含 `tools`），是代理/上游没有返回 `tool_calls`。**换一个能正常返回工具调用
的供应商即可**；本次新增的提示词工具段会让模型第一次在 system 里看到「你可以联网
检索」，但提示词救不了「供应商不回 tool_calls」。

## 验证（实跑）

- `go test ./...` 全绿（含新增 `tool_calls_postgres_test.go` 4 例真 PG 用例）。
- `cd apps/web && bun test` 1358 pass / 0 fail；`bun run typecheck` 0 错。
- `node tests/validate-architecture-boundaries.mjs` passed；
  `ruby scripts/validate-openapi-refs.rb` 2878 refs OK。
- **受控浏览器证据**（合成 2 行 `ai_tool_calls`：1 条 `web-search` 成功 +
  1 条 `not_allowed`，验证后已删除）：
  `GET /api/v1/comments/5010/citations` → `{"items":[],"toolCalls":[{"tool":"web-search",
  "arguments":"{\"query\":…}","latencyMs":842}]}`（`not_allowed` 被过滤、
  `resultDigest` 不出现）；页面恰好 1 个 `.sf-comment-ai-activity`，折叠标题
  「工具调用（1）」，展开后是 `web-search` + 检索词 + `842 ms`；1633 与 390 宽均无横向溢出。
  截图：`tmp/ai-activity-desktop.png`、`tmp/ai-activity-mobile.png`（gitignore）。
- **真实模型证据**：run `5b6065f9…` 的 `prompt_version=forum-reply@9`，证明新提示词
  已在线上生效。

## Next

1. **换供应商或修 `agent2api` 的 tools 透传**，然后重跑一次站外提问：
   预期该 run ≥2 行 `ai_executions`、`ai_tool_calls` 至少 1 行 `web-search`、
   页面出现「工具调用（1）」。
2. 工具调用恢复正常后重生成 eval 基线：
   `sforum ai:eval --fixture-dir cmd/sforum/testdata/ai-eval --diff --confirm`
   （`@9` 让 10 条 fixture 的 `promptDigest` 全变，且基线现在还是 3 次调用的旧行为）。
   **本轮刻意没有重生成**——把「工具不可用」写成基线会毁掉这条门禁的信号。
3. P2（未做，可裁）：`GET /admin/ai/runs/:runId` 增量返回 `toolCalls`（不过滤拒绝
   结局），`SFAdminAIRunTimeline.vue` 加一段「工具调用」。存储层已具备
   `PostgresToolCallStore.ListByRun`。
4. 全部改动未 commit。

## Open Questions

- 公开 `arguments` 是否可接受，最终由运营者定。若不可接受，只需在
  `Models/Forum/tool_calls.go` 的 `publicToolCalls` 里清空该字段，前端会自然不显示。
- 公开评论 DTO 仍不带 `isBot`，因此每条评论都会取一次 `/citations`（4 并发限流）。
  加 `isBot` 能把请求数降到「只有 AI 评论」，但那是公开身份契约变更，单独决策。

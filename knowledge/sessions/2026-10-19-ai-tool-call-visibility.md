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

## 关键发现与最终结局

**「机器人不搜索」的根因不在本仓库，而在供应商。** 链路排查与最终修复：

| 时间 | 事件 |
| --- | --- |
| 08:56 / 08:58 | `e7bc4532…` / `78bf401c…` 各有 3 行 `ai_executions` = 2 次工具往返（工具链本来是好的） |
| 21:45 | `ai_settings` 改到 `custom` → `http://127.0.0.1:3066/v1`（`agent2api`），`glm-5.3-flash` |
| 21:46 起（含 23:28 的截图 run `8681e832…`） | 每个 `forum.reply` run 都只有 **1** 行 = 零工具调用 |
| 00:01 / 00:02 | 同一 fixture 从基线 3 次调用 / 3 次工具掉到 **1 / 0**；把新加的「站外事实」段整段删掉再跑仍是 **1 / 0** |
| 00:01 | `~/.agent2api/debug/last-request.json` 显示请求**确实带了** `tools` 与 `tool_choice: auto` |
| 00:19 | 换成 `deepseek-v4.1-flash` 后全部 `502 OrcaTerm 建会话失败（Invalid or expired AccessToken）`——代理上游账号 token 过期，模型根本没被调到 |
| 00:23 | 换 key 后恢复：**3 次模型调用 / 3 次工具调用**（`answer-searchable-channel`） |

所以本仓库的线格式自始至终是对的（`TestOrchestratorRunsToolThenAnswers` 钉住出站
body 含 `tools`）；`agent2api` 那条链上有两个独立故障：先是「收了 tools 不回
tool_calls」，再是「上游账号 token 过期」。**换供应商或修代理即可，代码侧无需再动。**

**真实端到端证据**（2026-10-20 00:23，主题 1236，评论 5014 触发 → 5015 回复）：
run `105b53c7eed18e877bcfdd1d4987db01` 在 `ai_executions` 有 **3** 行（502/503/504），
`ai_tool_calls` 写 **4** 行：`web-search`（成功，3347ms）、`web-search`（失败，
10206ms）、`fetch-url`（失败，4506ms）、`web-search`（`step_exhausted`，预算 3 用完）。
公开端点只回前 3 条（`step_exhausted` 被过滤、`resultDigest` 不出现），页面折叠块显示
「工具调用（3）」且两条标「失败」。**模型在两次检索失败后如实说「最新 release 的
具体版本号我没能核实」，没有编版本号**——这正是 `@9` 那一段站外事实规则想要的
行为，也顺带证明了工具段确实进了 system。

## 验证（实跑）

- `go test ./...` 全绿（含新增 `tool_calls_postgres_test.go` 4 例真 PG 用例）。
- `cd apps/web && bun test` 1358 pass / 0 fail；`bun run typecheck` 0 错。
- `node tests/validate-architecture-boundaries.mjs` passed；
  `ruby scripts/validate-openapi-refs.rb` 2878 refs OK。
- **真实浏览器证据**（见上，run `105b53c7…`）：
  `GET /api/v1/comments/5015/citations` 返回 3 条 `toolCalls`（`step_exhausted` 那条
  被过滤、`resultDigest` 不出现）；页面恰好 1 个 `.sf-comment-ai-activity`，折叠标题
  「工具调用（3）」，展开是两条 `web-search` + 一条 `fetch-url`，两条带「失败」标记；
  1023/390 两档均无横向溢出。截图：`tmp/ai-activity-live.png`、
  `tmp/ai-activity-live-mobile.png`（gitignore）。
- **eval 基线已重生成**（2026-10-20 00:26，`--confirm` 10 条全部干净、无 `error`）：
  `answer-searchable-channel` 3/3、`answer-searchable-pg-index` 3/3、
  `crosslang-en-question` 2/2、`longthread-debug-escalation` 3/3、
  `longthread-migration-review` 2/2、`refuse-private-data` 2/2、
  `image-diagram-question` 3/3；三条无需工具的用例 1/0。
  `promptDigest` 全变（`@9` + 工具段），这是预期。

## Next

1. ~~换供应商~~ **已解决**（2026-10-20 00:23，`deepseek-v4.1-flash` + 新 key）。
2. ~~重生成 eval 基线~~ **已完成**（见上）。若供应商再换，先跑
   `ai:eval --diff`（不带 `--confirm`）确认指纹是否漂移，再决定是否 `--confirm` 重写。
3. P2（未做，可裁）：`GET /admin/ai/runs/:runId` 增量返回 `toolCalls`（不过滤拒绝
   结局），`SFAdminAIRunTimeline.vue` 加一段「工具调用」。存储层已具备
   `PostgresToolCallStore.ListByRun`。
4. **已知可调项**：`gates.toolCallsPerReply` 现在是 3，这次真实回复刚好在第 4 步被
   截断（`step_exhausted`）。工具变多之后，站外核实 + 读页面很容易超预算；若运营者
   希望机器人更「肯查」，把它调到 5（`DefaultToolCallsPerReply`）更合适。
5. 提交：`81084a698` 已推；本轮补丁（`url` 摘要 + 基线）单独一个提交。

## Open Questions

- 公开 `arguments` 是否可接受，最终由运营者定。若不可接受，只需在
  `Models/Forum/tool_calls.go` 的 `publicToolCalls` 里清空该字段，前端会自然不显示。
- 公开评论 DTO 仍不带 `isBot`，因此每条评论都会取一次 `/citations`（4 并发限流）。
  加 `isBot` 能把请求数降到「只有 AI 评论」，但那是公开身份契约变更，单独决策。

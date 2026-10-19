# AI 工具调用：让机器人真的去搜 + 把调用记录给用户看

Status: **completed** (2026-10-19，2026-10-20 真实端到端验证通过)

## 0. 一句话结论

两个问题彼此独立，但共用一条链路：

- **没调用搜索** 不是接线问题，是**模型没被要求去搜**——它确实拿到了 `web-search`
  工具，但一次都没调。
- **看不到调用记录** 是因为轨迹只写进了 slog 一行日志，**没有任何持久化**，所以
  用户看不到、运营者也看不到，只能靠推断。

因此方案分两半：**W1 让工具进入提示词（行为）**，**W2 把轨迹落库并公开读
（可见性）**。W2 同时是 W1 的诊断工具——先能看见，才谈得上调优。

## 1. 已核实的现状（证据）

| 事实 | 证据 |
| --- | --- |
| 截图那次回复**零工具调用** | run `8681e832bf95531db01d41b3aec34a1b` 在 `ai_executions` 只有 **1** 行；带一次工具调用的 run 必然 ≥2 次模型调用（先要工具、再答） |
| 工具循环本身是好的 | 同库 run `e7bc4532…` / `78bf401c…` 各有 **3** 行 → 两次工具往返，就是这个 `glm-5.3-flash` 跑出来的 |
| 工具**已经**对模型可见 | `ai_settings.purposes["forum.reply"].tools.allowed` 含 `web-search`、`fetch-url`；`sforum.web-search` 状态 `enabled`，插件进程在跑 |
| 提示词完全没提站外检索 | `DefaultReplySystemPrompt` 只有「站内工具」一段；全文无「联网 / 站外 / 最新」 |
| 轨迹只进日志 | `RunResult.ToolTrace` → `logGeneratedReply` 的 `tool_trace` 字段；`ai_executions` 无此列，`GET /admin/ai/runs/:runId` 只回放 `ai_executions` |
| 已有可照抄的公开读范式 | `ai_citations` 表 + `CitationService.CommentCitations` + `GET /comments/:id/citations`（guest-read + 主题级可见性 + 404 不可区分） |

`ai_citations` 对该 run 也是 0 行——模型连 `[[cite:…]]` 都没写，同一类「提示词
没要求」的病灶。

## 2. 目标与验收

- **验收 1（行为）**：在主题 1236 提一个必须站外核实的问题，机器人先 `web-search`
  再作答；该 run 在 `ai_executions` ≥2 行，`ai_tool_calls` 至少 1 行 `web-search`。
- **验收 2（可见性）**：该 AI 评论下方出现可展开的「工具调用（1）」，含工具名、
  参数摘要、成败、耗时；零工具调用时**整块不渲染**。
- **验收 3（不回归）**：普通评论仍然零渲染；`GET /comments/:id/citations` 既有
  `items` 字段逐字不变。

## 3. 设计决策

### D1 · 工具面由宿主渲染进 system，不把插件名写进 Core 提示词

Core 默认提示词里写死 `web-search` 有三个问题：把某个插件的存在性固化进框架、
运营者自定义提示词后立刻失效、下一个插件还得再改一次。

改为：宿主把**本次真正会发给模型的工具**渲染成一段清单，插在「基础提示词」之后、
「安全尾注」之前。

```
本次可调用的工具（由站点安装的扩展提供）：
- web-search：检索站外互联网，返回标题、链接、摘要与发布时间。
- forum-search：搜索站内主题与评论。
```

`ToolSection` 由 **Orchestrator** 生成，因为只有它同时知道「用途白名单」「登记表」
「本次是否真的带工具（步数预算 / profile 是否支持 tools）」。这样「提示词宣称有」
与「请求真的带上」是同一个判断，不会出现「提示词说有、模型一调就是不可用」。

同时给默认提示词补一段**通用**规则（不点名任何插件）：站外事实必须先用检索工具
核实；用户明确要求「搜索 / 查一下」时先检索；没有可用检索工具时直说无法核实。

`CoreReplyPromptVersion` → `forum-reply@9`。

**代价**：提示词变了 → `ai_eval` 基线的 prompt 指纹全变，必须重生成
（`sforum ai:eval --diff --confirm`）；`ai_topic_state` 里 `promptVersion=@8`
的行整体失效（下次写入时基底清空，属既定语义，不需要迁移）。

### D2 · 轨迹落新表，不扩 `ai_executions`

`ExecutionRecord` 的契约原文是「刻意不含提示词与响应正文：那两者可能包含用户
内容」。工具 `arguments` 是模型输出、`result_digest` 是工具结果片段，都是用户
内容。与 `ai_citations` 同一条理由。

### D3 · 公开投影做减法：不给结果片段，不给「被拒绝」结局

- **不公开** `result_digest`：它是工具返回给模型的正文片段。工具按
  `ViewerUserID` 取数，**提问者看得见的东西读者不一定看得见**（私密板块 / 受限
  附件），公开它等于开一个越权旁路。
- **不公开** `failure ∈ {unknown_tool, not_allowed, step_exhausted,
  budget_exhausted}`：这四种是「模型撞到了用途白名单配置」，公开等于把运营者的
  工具策略摊给读者看。
- **公开**：工具名、`arguments`（写入侧已按站点脱敏规则处理并截断到 1 KiB）、
  `isError`、`latencyMs`。`internal_error` 属于真实尝试且失败，公开（显示为
  「失败」）。
- 同一张表两个投影：公开读过滤，admin 读全量。

### D4 · 不新增路由，`/comments/:id/citations` 增量返回 `toolCalls`

考虑过 `GET /comments/:id/tool-calls` 与 `GET /comments/:id/ai-activity`，都否掉：

- 页面上一页 20 条评论，现有实现已经是「每条 1 个请求（4 并发限流）」，再加一条
  端点变 40 个。
- 新路由要动 core 路由目录：`core_catalog_gen.go`、
  `core_guard_authorizer.go` 的 `forumReadPolicyRoute`、`registry_test.go` 的 374
  计数、OpenAPI 索引、`docs/extensions/v3/catalogs/routes.md`。成本与收益不成比例。

改为**增量字段**：`CommentCitationList` 保留 `required: [items]`，新增可选
`toolCalls`。对任何既有消费者都是无破坏的。

**代价**：路径名 `citations` 与载荷不再一一对应。处理方式是在 OpenAPI 描述与
代码注释里明确它现在的语义是「AI 产物活动（依据 + 工具调用）」，前端组件同步
改名 `SFCommentAIActivity.vue`，路径本身不动（改名才是真正的破坏）。

### D5 · 写入时机与幂等

与 `recordCitations` 同一位置：**发布成功之后**写，失败只 `Warn` 不阻断（一条已
发布的评论不该因为记录失败变成重试任务、重复发帖）。

幂等键用 `UNIQUE (comment_id, seq)`，`seq` 是 trace 切片下标。**不能用 `step`**：
同一条 assistant 消息里的并行工具调用共享同一个 `step`。

### D6 · 提示词组装点收敛到一处

现在尾注由 `ReplySettings.EffectiveSystemPrompt()` 拼，工具段又要插在尾注之前，
两个拼装点必然分叉。

收敛为：`supportai.ComposeReplySystemPrompt(base, toolSection)` = base + 工具段
（非空时）+ 尾注（**永远最后**）。`ReplyParameterSource.ReplySystemPrompt` 改为
返回**基础提示词**（无尾注），生成器负责组装。

`cmd/sforum/ai_eval.go` 的参数源保持返回 `EffectiveSystemPrompt()`（等价于
`Compose(base, "")`），因此 **eval 侧逐字不变**，只受 D1 的版本号影响。

## 4. 任务

### P0-1 提示词工具面（W1）

写作用域：`apps/api/app/Support/AI/`、`apps/api/app/Models/AI/service.go`、
`apps/api/app/Models/AIReply/`、`apps/api/app/Providers/ai_reply_generator.go`

- `Support/AI/reply_prompt.go`
  - 新增 `BaseSystemPrompt()`；`EffectiveSystemPrompt()` 改为
    `ComposeReplySystemPrompt(BaseSystemPrompt(), "")`（语义不变）。
  - 新增 `ComposeReplySystemPrompt(base, toolSection string) string`。
  - 新增 `RenderToolSection([]ToolDefinition) string`：空集合返回 `""`；每行
    `- 名称：描述首句`，描述截断到 120 rune；按名字排序。
  - `DefaultReplySystemPrompt` 增「站外事实」段；`CoreReplyPromptVersion` →
    `forum-reply@9`。
- `Support/AI/tool_loop.go`
  - 新增 `func (o *Orchestrator) ToolSection(ctx, names []string) string`：读
    settings → 与 `Run` 完全相同的闸门判断（`steps > 0`、`descriptors` 非空、
    `toolsSupported`）→ `RenderToolSection`。不可用时返回 `""`。
- `Models/AI/service.go`：`ReplySystemPrompt` 改返回 `BaseSystemPrompt()`。
- `Models/AIReply/generator.go`：可选依赖 `toolSectioner` + `WithToolSectioner`；
  `Generate` 算出 `toolSection` 传给 `buildReplyPrompt`；system 由
  `ComposeReplySystemPrompt` 组装。
- `Models/AIReply/context.go`：`replyPromptExtras` 增 `toolSection`。
- `Providers/ai_reply_generator.go`：`orchestrator != nil` 时注入。

### P0-2 轨迹落库（W2 后端）

写作用域：`apps/api/app/Models/AIReply/`、`apps/api/database/migrations/`、
`apps/api/app/Providers/`

迁移 `apps/api/database/migrations/202610190005_ai_tool_calls.sql`（见 §5）。

- `Models/AIReply/tool_calls_store.go`：`ToolCallRecorder` +
  `PostgresToolCallStore`，`INSERT … ON CONFLICT (comment_id, seq) DO NOTHING`，
  条数上限 `MaxToolCallsPerReply`（12）。
- `generator.go`：`WithToolCalls(store)` + `recordToolCalls`（发布成功后调用，
  失败只 Warn）。
- `Providers/ai_reply_generator.go`：与 `WithCitations` 同一段装配。

### P0-3 公开读

写作用域：`apps/api/app/Models/Forum/`

- `Models/Forum/tool_calls.go`（新）：`ToolCallEntry{Tool, Arguments, IsError,
  Failure, LatencyMS}`（**无** `resultDigest`）；`ToolCallReader`；
  `ToolCallService`（与 `CitationService` 同形：持有 `*Service`、复用
  `visibleCommentTopic`、不往 `Service` 上加方法或字段）。
- `Models/Forum/citations.go`：`CitationList` 增
  `ToolCalls []ToolCallEntry \`json:"toolCalls,omitempty"\``；`CitationService`
  注入 `ToolCallReader`。

### P0-4 契约

- `contracts/openapi/schemas/forum.yaml`：新增 `CommentToolCall`；
  `CommentCitationList.properties` 增 `toolCalls`；`required` **保持 `[items]`**。
- `contracts/openapi/paths/forum.yaml`：`listCommentCitations` 的 summary/
  description 改成「AI 产物活动」，写明 `toolCalls` 的公开过滤规则。

### P0-5 前端

写作用域：`apps/web/app/components/forum/`、`apps/web/i18n/locales/`、
`apps/web/tests/forum/`

- `components/forum/citations.ts` → `aiActivity.ts`：保留既有导出，新增
  `visibleToolCalls()` 与 `toolArgumentSummary()`。
- `components/forum/SFCommentCitations.vue` → `SFCommentAIActivity.vue`：一个
  `<details>`，两个分区「工具调用（N）」「依据（N）」；渲染条件改为
  `toolCalls.length || citations.length`；工具名用等宽徽标原样显示；**永不
  `v-html`**。
- `components/forum/SFComment.vue`：改挂载点与 import。
- i18n `topicDetail.aiActivity.{summary,loading,toolCallsTitle,citationsTitle,
  failed,latency}`，zh-CN + en-US；`topicDetail.citations.*` 保留不动。
- `tests/forum/commentCitations.test.ts` → `commentAIActivity.test.ts`。

### P1 验证

1. 真实触发：主题 1236 提「SForum 是什么论坛程序、谁开发的」→ 断言该 run
   `ai_executions` ≥2 行、`ai_tool_calls` 有 `web-search` 行。
2. 浏览器：AI 评论下出现「工具调用（1）」，展开后参数是检索词；普通评论无块。
3. 回归：`GET /comments/5008/citations` 仍返回 `items`；404 语义不变。
4. 门禁：`go test ./...`、`cd apps/web && bun test`、`bun run typecheck`、
   `ruby scripts/validate-openapi-refs.rb`、
   `node tests/validate-architecture-boundaries.mjs`（不新增告警/失败）。
5. `sforum ai:eval --diff --confirm` 重生成 G 基线。

### P2 管理面（可单独裁掉）

- `GET /admin/ai/runs/:runId` 增量返回 `toolCalls`（**不过滤**拒绝结局）。
- `SFAdminAIRunTimeline.vue` 加一段「工具调用」，含 `failure` 原值。

## 5. 数据模型

```sql
-- +goose Up
-- +sforum OnlineSafe
CREATE TABLE ai_tool_calls (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  run_id TEXT NOT NULL DEFAULT '',
  topic_id BIGINT NOT NULL DEFAULT 0,
  comment_id BIGINT NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
  seq INT NOT NULL,             -- trace 切片下标；并行调用共享 step，不能用 step 做键
  step INT NOT NULL DEFAULT 0,  -- 模型调用序号，供 admin 回放
  tool TEXT NOT NULL,
  arguments TEXT NOT NULL DEFAULT '',
  arguments_bytes INT NOT NULL DEFAULT 0,
  result_bytes INT NOT NULL DEFAULT 0,
  result_digest TEXT NOT NULL DEFAULT '',
  is_error BOOLEAN NOT NULL DEFAULT false,
  failure TEXT NOT NULL DEFAULT '',
  latency_ms INT NOT NULL DEFAULT 0,
  CONSTRAINT ai_tool_calls_comment_seq_key UNIQUE (comment_id, seq)
);
CREATE INDEX ai_tool_calls_comment_idx ON ai_tool_calls (comment_id, seq);
CREATE INDEX ai_tool_calls_run_idx ON ai_tool_calls (run_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS ai_tool_calls_run_idx;
DROP INDEX IF EXISTS ai_tool_calls_comment_idx;
DROP TABLE IF EXISTS ai_tool_calls;
```

## 6. 边界与失败模式

| 场景 | 行为 |
| --- | --- |
| 工具失败（`isError`） | 公开可见，标「失败」 |
| 模型编工具名 / 撞白名单 | 落库但**不公开**；admin 可见 |
| 回复发布失败 | 不写 `ai_tool_calls`（评论不存在，FK 也拦得住） |
| 评论软删 / 主题转隐藏 | 公开读 404，与「评论不存在」不可区分 |
| 主题级联删除 | `ON DELETE CASCADE` 清掉记录，不留孤儿 |
| River 重试 | `(comment_id, seq)` 冲突即忽略，不产生重复行 |
| 插件被停用 | 历史记录仍展示，新 run 不再有该工具 |
| 运营者自定义了提示词 | 工具段由宿主追加，**仍然生效**（D1 的关键收益） |

## 7. 风险与假设

- **假设**：运营者保留 `web-search` 在 `forum.reply` 白名单（当前是）。若被移除，
  提示词段随之消失，行为退回现状——这是正确行为。
- **假设**：公开 `arguments` 可接受。写入侧已过 `RedactText` 并截断 1 KiB。若判定
  参数也不该公开，只需在 P0-3 的过滤里去掉 `Arguments`，其余不动。
- **风险**：`glm-5.3-flash` 仍可能不调工具。D1 是当前最有效的杠杆；若仍不调，
  下一步是 `tool_choice` 强制（按用户措辞里的检索意图），属另一个决策。
- **风险**：提示词变长。工具段每行一条、描述截 120 rune、上限
  `MaxToolsPerRequest`（16）；`CacheKey` 已含 system 文本，缓存不会被污染。

## 8. 不做的事

- 不做「工具调用实时流」：回复发布后才出现，没有中间态可流。
- 不公开 token / 成本 / 模型名。
- 不动 `ai_executions` 表结构与 `ExecutionRecord`。
- 不给 `web-search` 结果做正文预览——那是 `fetch-url` 的职责。

## 9. 进度账本

| 任务 | 状态 | 证据 |
| --- | --- | --- |
| P0-1 提示词工具面 | **done** | `ComposeReplySystemPrompt` / `RenderToolSection` / `Orchestrator.ToolSection` / `WithToolSectioner`；`forum-reply@9` 已在 run `5b6065f9…` 生效 |
| P0-2 轨迹落库 | **done** | 迁移 `202610190005` 已应用；`TestPostgresToolCalls*` 4 例真 PG 通过（含重放幂等与级联） |
| P0-3 公开读 | **done** | `CitationList.ToolCalls` + `publicToolCalls`；受控数据下端点只返回 1 条（`not_allowed` 被过滤、`resultDigest` 不出现） |
| P0-4 契约 | **done** | `CommentToolCall` + `toolCalls`（`required` 仍只有 `items`）；`validate-openapi-refs` 2878 refs OK |
| P0-5 前端 | **done** | `SFCommentAIActivity.vue` 一个块两分区；web 1358 pass、typecheck 0 错；1633/390 截图齐 |
| P1 验证 | **done** | run `105b53c7…`：3 次模型调用、`ai_tool_calls` 4 行（3 条公开、1 条 `step_exhausted` 被过滤）；页面「工具调用（3）」；eval 基线已按 `@9` 重生成 |
| P2 管理面 | pending（可裁） | 存储层已备 `ListByRun` |

## 10. 结论补充（实现后回填）

**W1/W2 的代码是对的；线上「机器人不搜索」的根因在本仓库之外，且是两层故障。**

- 2026-10-19 21:45 供应商切到本机 `agent2api`（`127.0.0.1:3066`）后，所有
  `forum.reply` run 都只有 1 次模型调用；同一 eval fixture 从 3 次调用 / 3 次工具
  调用掉到 1 / 0；把本轮新增的提示词段整段删掉再跑仍是 1 / 0；而
  `~/.agent2api/debug/last-request.json` 显示请求确实带了 `tools` 与
  `tool_choice: auto`。→ 代理收了工具却不回 `tool_calls`。
- 2026-10-19 00:19 换成 `deepseek-v4.1-flash` 后变成全量 `502 OrcaTerm 建会话失败
  （Invalid or expired AccessToken）`：代理上游账号 token 过期，模型根本没被调到。
- 2026-10-20 00:23 换 key 后恢复，真实触发验证通过（见 §9 P1 与 handoff）。

**可调项**：`gates.toolCallsPerReply` 当前是 3，这次真实回复在第 4 步被截断
（`step_exhausted`）。工具变多之后「检索 + 读页面」很容易超预算，运营者若希望机器人
更肯查，调到 5（`DefaultToolCallsPerReply`）更合适。

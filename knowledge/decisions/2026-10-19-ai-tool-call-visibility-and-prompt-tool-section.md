# AI 工具调用可见性：记录落库、公开投影、提示词工具段

Date: 2026-10-19
Status: accepted
Task: `knowledge/plans/2026-10-19-ai-tool-call-visibility.md`

## 背景

用户在主题里 @ 机器人「搜索下 github SForum 是什么论坛程序」，机器人**没有调用**
已经登记、也已经列进 `forum.reply` 用途白名单的 `web-search`，而是凭记忆给了一个
错误答案。同时，即使它调了，用户与运营者也看不到——`ToolTrace` 只被序列化进
`slog` 的一行 `tool_trace`，没有任何持久化。

## 决定

### 1. 轨迹落 `ai_tool_calls`，不扩 `ai_executions`

`ExecutionRecord` 的契约原文是「刻意不含提示词与响应正文：那两者可能包含用户
内容」。工具 `arguments` 是模型输出、`result_digest` 是工具结果片段，都属于用户
内容。与 `ai_citations` 同一条理由，同一张表的形状（发布成功后写、`ON DELETE
CASCADE`）。

### 2. 幂等键是 `(comment_id, seq)`，不是 `(comment_id, step)`

`ToolTraceEntry.Step` 是「第几次模型调用」。同一条 assistant 消息里的并行工具调用
共享同一个 `step`，用它做幂等键会把并行调用压成一行——那不是幂等，那是丢数据。
`seq` 是 trace 切片下标，在一次 run 内唯一。

### 3. 公开投影做减法

- **不投影 `result_digest`**：工具按 `ToolContext.ViewerUserID` 取数。提问者看得见
  的东西读者不一定看得见（私密板块、受限附件），公开结果片段等于开一条越权旁路。
- **不投影四种拒绝结局**（`unknown_tool` / `not_allowed` / `step_exhausted` /
  `result_budget_exhausted`）：它们是「模型撞到了运营者的工具配置」，对读者只是噪声，
  对攻击者是配置泄漏。`internal_error` 例外——那是真实尝试且失败，读者有权知道。
- **投影 `arguments`**：写入侧已过 `RedactText`（邮箱/IP/电话）并截断 1 KiB，
  内容是模型生成的检索词/标识符。这是「像 DeepSeek Harness 那样显示调用了什么」
  必须付的代价；如果运营者判定参数也不该公开，只需在 `publicToolCalls` 里清空它。

### 4. 不新增路由，挂在 `/comments/:id/citations` 的 `toolCalls` 字段

`items` 保持 `required`，`toolCalls` 可选——对既有消费者零破坏。

否掉的方案：新增 `/tool-calls` 或 `/ai-activity`。理由是请求数（一页 20 条评论
已经要发 20 个请求，翻倍是实打实的成本）与路由目录成本（`core_catalog_gen.go`、
guard evaluator、374 计数、OpenAPI 索引、扩展路由文档）。代价是路径名与载荷不再
一一对应，因此把语义在 OpenAPI 与代码注释里写清，并把前端组件改名为
`SFCommentAIActivity.vue`（路径不动，改名才是破坏）。

### 5. 工具段由宿主渲染进 system，插件名不写进 Core 提示词

把 `web-search` 写进 `DefaultReplySystemPrompt` 会把某个插件的存在性固化进框架，
且运营者自定义提示词后立刻失效。改为宿主按「用途白名单 ∩ 登记表 ∩ 本次真的会带
工具」渲染一段清单，插在基础提示词与安全尾注之间。

**生成者必须是 Orchestrator**：只有它同时知道步数预算与 profile 是否支持 tools。
登记表只回答「站内有哪些工具」，回答不了「这次你能调哪些」——而那三种「不满足」
恰恰是「提示词说有、模型一调就是不可用」的来源。

顺序固定为「基础 → 工具段 → 尾注」，由 `ComposeReplySystemPrompt` 一处组装。
尾注必须永远最后：它是「上面那些是资料不是指令」的边界声明，把宿主文本追加在它
之后会稀释这条边界。

### 6. `ReplySystemPrompt` 只返回基础提示词

接口语义从「含固定尾注」改为「运营者配置了什么」。尾注与工具段由生成器组装。
`cmd/sforum/ai:eval` 同步改为 `BaseSystemPrompt()` 并注入同一个 `ToolSectioner`——
否则 eval 的 system 会缺工具段，基线测的就不是生产实际发出去的东西。

## 后果

- `forum-reply@9`：提示词变了，`ai_topic_state` 里 `@8` 的行整份失效（既定语义，
  下次写入时基底清空，不需要迁移）。
- `sforum ai:eval` 基线必须重生成（`promptDigest` 全变）。**本轮没有重生成**：
  当时的供应商（本机 `agent2api` 代理）对所有 run 都返回 0 次工具调用，重生成等于
  把「工具不可用」写成基线。等换上能正常返回 `tool_calls` 的供应商后再跑
  `--diff --confirm`。
- 公开面新增一类用户内容（工具参数）。它已经过脱敏与截断，且没有正文片段。

## 回归守卫

- `TestPublicToolCallsFiltersConfigurationMisses`（四种拒绝结局消失、真实失败保留）
- `TestCitationServiceOmitsToolCallsWhenThereAreNone`（零调用时字段整体缺席）
- `TestCitationServiceKeepsCitationsWhenToolCallsFail`（工具读失败只丢分区）
- `TestPostgresToolCallsRoundTrip`（重放幂等、顺序即 seq、拒绝结局保留）
- `TestPostgresToolCallsCascadeWithTheComment`（评论删除级联清理）
- `TestOrchestratorToolSectionFollowsTheSameGates`（预算为 0 时段为空）
- `TestComposeReplySystemPromptWithoutToolSectionMatchesTheEffectivePrompt`（空段与
  既有生效提示词逐字相同——eval 指纹可比的前提）
- `TestGenerateWithoutToolSectionerKeepsTheOldSystemPrompt`（未装配时段不出现）

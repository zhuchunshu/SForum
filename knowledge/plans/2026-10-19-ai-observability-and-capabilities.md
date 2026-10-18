# AI 观测层与能力层任务书（A 批 + G + B 批）

Status: **ready** — 计划已审阅通过，可开工
Date: 2026-10-19
Last updated: 2026-10-19

Goal: 让 AI 工具循环对运营者第一次可见（调用轨迹、run 级关联、回放），再在这个
基础上扩容能力（预算、工具、用途、证据链、会话状态），使每一次能力改动都可以被
测量、被解释、被回滚。

## Required Reading

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/ai.md`
4. `knowledge/decisions/2026-10-02-ai-provider-gateway.md`
5. `knowledge/decisions/2026-10-12-ai-scope-mcp-skills-streaming.md`
6. `knowledge/decisions/2026-10-15-ai-reasoning-separation-and-model-identity.md`
7. `knowledge/decisions/2026-10-18-ai-tool-context-and-catalog.md`
8. this task book

## 硬约束（任何实现都不得违反）

1. **`ExecutionRecord` 不含用户内容。** 工具参数与结果摘要属于用户内容，只能进
   日志与新表，不进 `ai_executions` 的正文类字段。
2. **AI 只起草，人类定稿。** 本任务书不含写动作面；`SubmitDecision` 语义不变。
3. **绝无 AI 事件链触发 AI。** `trigger.go` 的两道防循环闸保持原样。
4. **工具以提问者身份取数。** 所有工具的可见性判定用 `ToolContext.ViewerUserID`，
   不得改用 bot 账号身份。
5. **一切模型调用走网关。** 不得绕过 `Gateway.Execute`（成本、配额、trace、缓存、
   出站守卫只有这一处实现）。
6. **行号会漂。** 文中行号是 2026-10-19 HEAD 的定位提示，实现时以符号名为准。

## 里程碑与依赖

```
A1 ──► A2 ──┐
            ├──► A3 ──► A4
G（独立，可在 A 批之后任意时点）

G ──► B1 / B2 / B3 / B4 ──► B5
```

| 里程碑 | 任务 | 串行工期 | 可中断点 |
|---|---|---|---|
| M1 | A1, A2, A3, A4, G | 5.5 人日 | 每项独立可停 |
| M2 | B1, B2, B3 | 6.5 人日 | 每项独立可停 |
| M3 | B4, B5 | 10 人日 | B4 不完成则 B5 不动 |

串行合计 22 人日（约 4.5 周）。并行方案见下一节，日历工期约 10–12 天。

## 并行执行方案

### 轨道划分

四条可并行的轨道。每条轨道内部串行，轨道之间只经过下面写明的握手点。

| 轨道 | 任务链 | 独占写作用域 | 工期 |
|---|---|---|---|
| T1 观测后端 | A1 → A2 → A3 → A4 | `Support/AI/tool_loop.go`、`limits.go`、`store.go`、`gateway.go`、`postgres_store.go`、`Http/Controllers/AI/*`、迁移 001 | 5.5 天 |
| T2 eval | G | `cmd/sforum/ai_eval.go`、`command.go`、`cmd/sforum/testdata/ai-eval/**` | 1.5–2 天 |
| T3 工具 | B2 | `Models/AITools/**`、`reply_prompt.go`、`bootstrap/api_assembly_domains.go` | 3 天 |
| T4 用途 | B3 | `Support/AI/purpose.go`、`Models/AIDerived/**`、`Jobs/AIDerived/**` | 3 天 |
| T5 证据 + 状态 | B4 → B5 | `Models/AIReply/context.go`、`generator.go`、`poster.go`、迁移 002/003、前端渲染组件 | 后端 4 天、B5 4 天 |
| T6 前端 | A4 前端 + B4 渲染 + B5 无可视面 | `apps/web/app/composables/admin/useAdminAI.ts`、`components/admin/ai/**`、`i18n/locales/*` | 3 天（可与 T1/T5 并行，按契约先行） |

### 写作用域冲突矩阵（并行前必须核对）

| 文件 | 归属轨道 | 说明 |
|---|---|---|
| `Support/AI/tool_loop.go` | T1 独占 | A3 的 run id 生成也在这里，故 A2 → A3 必须串行 |
| `Support/AI/limits.go` | T1 独占 | A2 加 trace 常量、B1 改预算常量，都在这里 → **B1 必须排在 A2 之后** |
| `Support/AI/store.go` / `gateway.go` / `postgres_store.go` | T1 独占 | A3 + A4 的 `ListRunExecutions` |
| `Support/AI/purpose.go` | T4 独占 | T3 只读它的 `CoreReplyToolAllowlist` 引用，不写 |
| `Models/AITools/aitools.go` | T3 独占 | `Deps` 与 `NewBuiltinRegistry` |
| `Models/AIReply/context.go` / `generator.go` | T5 独占 | A2 只改 `CompletionOutput` 的字段（与 T5 的 `BotCommentInput` 不同结构，可同文件不同段落，但**仍需串行**：A2 先落地再启动 T5） |
| `apps/web/**/useAdminAI.ts` | T6 独占 | A3 的类型扩字段由 T6 代改，T1 不碰前端 |
| 迁移文件 | 各自命名 | 001=A3、002=B4、003=B5，序号不冲突 |

**唯一需要串行的跨轨道点**：T1 的 A2 完成后，T5 才能改 `Models/AIReply/context.go`。这是全程唯一的硬握手。

### 建议执行时序（日历日）

```
Day 1     T1:A1+A2   T2:G        T6:准备（读契约）
Day 2     T1:A3      T2:G        T6:A4 前端壳
Day 3     T1:A4后端  T3:B2       T4:B3      T6:A4 前端
Day 4     T1:A4      T3:B2       T4:B3      T6:A4
Day 5     T1:B1      T3:B2       T4:B3      T6:—
Day 6     T5:B4后端  ...
Day 7     T5:B4后端  T6:B4 渲染
Day 8     T5:B4后端  T6:B4 渲染
Day 9     T5:B5      T6:—
Day 10    T5:B5      收尾 + 全量验证
Day 11-12 缓冲（联调、浏览器 QA、文档）
```

### 两档排期

| 档位 | 关键路径 | 日历工期 | 前置条件 | 代价 |
|---|---|---|---|---|
| 稳态 | `A2(0.5) → B4 后端(4) → B5(4)` = 8.5 天 | **9–11 天** | 六轨道按上表并行 | 需要 3–4 个执行者 + 一个协调人看账本 |
| 激进 | 同上，但 B5 的存储层（表 + 读写 + 失效策略）在 B4 第 3 天定稿 `Citation` 结构后即开工 | **7–9 天** | 先 interface-first 冻结 `Citation` / `TopicState` 结构 | 结构一旦冻结就不能改；改一次等于两轨重做 |

**B1 不在关键路径上**：它只改 `limits.go` 的三个常量，0.5 天，插在 T1 任何空档即可。
唯一约束是排在 A2 之后（两者都动 `limits.go`）。

**压缩的边界**：能压的是串行等待与缓冲，压不了的是验证矩阵 L1–L5。
若为了缩短工期削减测试，缺陷会在 M3 之后集中爆发，返工成本高于省下的时间。
**任何压缩方案都不得减少上表的测试用例数量。**

任何并行都压不过的关键依赖只有一条：B4 → B5 的数据依赖（状态里存的是
B4 校验过的事实）。这不是排期问题，是设计问题——想绕过它只能放弃 B5 的价值。

### 并行纪律（每个执行者必须遵守）

1. **开工前读本任务书的「进度账本」**，确认自己负责的任务当前状态与他人的占位。
2. **只改自己轨道的作用域**。需要改别人的文件时，先在账本里登记，由该轨道 owner 执行。
3. **测试按包跑**，不要 `go test ./...`（并行时会互相打断）；收尾阶段才跑全量。
4. **迁移文件序号预先分配**，不得临时改名。
5. **每次提交只含一个任务的改动**，提交信息带任务编号（如 `A2: tool trace`）。

## 记忆连续性（三层，全部必须实现）

"不丢记忆"在这件事里有三个不同的对象，三者都要单独防。

### 层 1 — 运行期：一次 run 内的部分结果不得被丢弃

**现状**：`Orchestrator.Run` 在网关失败时**已经**返回部分 `RunResult`（`return result, err`），但 `GatewayCompleter.Complete` 直接 `return CompletionOutput{}, err`，把已完成的工具调用与 token 用量全部丢掉。重试从零开始，上一次的检索结果作废。

**改进**（并入 A2）：

- `GatewayCompleter.Complete` 在错误路径返回**已填充的** `CompletionOutput`（`ToolTrace`、`Usage`、`ModelCalls`、`ToolCalls`）与 `error`。
- `Generator.Generate` 在错误路径把这些写进 `Warn` 日志，并放进 `error` 的包装文本（`tool_calls=2 usage=1200/300`），让重试的一方知道上次做到哪。
- **不做断点续跑**：River 的重试是整任务重跑，续跑需要把中间消息序列化进 job payload，代价远大于收益。本任务只保证"部分结果可被观察到"。

### 层 2 — 会话期：B5 的状态必须区分失效与丢失

B5 的三重失效是**主动丢弃**（版本变更、超期、引用失效）。丢失是另一回事：进程崩溃、并发覆盖、文档损坏。防丢失措施（并入 B5）：

1. **写入用乐观锁**：`UPDATE ... WHERE revision = $n`，影响行数为 0 时**放弃本次写入**（不覆盖后写者）。后写者不一定是更正确的那个。
2. **读取时容错**：`document` 无法解析 → 记 `Warn`、当作无状态、不 panic、下次写入时重建。
3. **写入时机**：发布成功之后。发布失败却写状态，下一轮会引用一段没人看过的结论。
4. **状态自检日志**：每次注入状态时记 `facts=N, dropped=M, reason=<过期|版本不符|引用失效>`。"为什么这次它不知道上次的事"必须能从日志回答。

### 层 3 — 开发期：任何执行者接手都不丢上下文

多 agent 并行最大的风险不是写冲突，是**上下文丢失**——A 做了 80% 的 A3，B 接手时不知道做到哪、为什么这么选。

**进度账本是唯一真相来源**（见文末表格）。规则：

1. 开工前读账本；**每完成一个任务立刻更新**，不要攒到最后写 handoff。
2. 每行必须写：状态、已改文件、验证到哪一步、**下一个接手点**、已知未解问题。
3. 状态只有四种：`未开始` / `进行中` / `待验证` / `已完成`。`进行中` 必须写"停在哪个文件哪一步"。
4. 任何偏离本任务书的决定，先写进账本再执行——否则下一个执行者会按原任务书改回去。
5. 收尾时才把账本汇总成 `knowledge/sessions/` 的 hot handoff。

## 任务索引

| # | 任务 | 依赖 | 工期 | 主要验收 |
|---|---|---|---|---|
| A1 | slog 探针 | 无 | 0.5h | 日志可读，结论记录到 handoff |
| A2 | ToolTrace | A1 | 0.5d | 8 个单测 + 日志含 trace |
| A3 | correlation id | 无（可与 A2 并行） | 1d | 迁移 + 8 处透传 + 参数化测试 |
| A4 | 轨迹回放 | A3 | 2d | 新端点 + 前端时间线 + 浏览器 QA |
| G | eval 基线 | 无 | 2d | `--diff` 能报出人为改动 |
| B1 | 放宽预算 | G | 0.5d | eval 对比证明收益 |
| B2 | 补 3 个工具 | G | 3d | 每工具 allowed/denied 测试 |
| B3 | 扩 2 条 purpose | G | 3d | 手动触发端到端 |
| B4 | 证据链溯源 | G | 6d | 引用校验 + 渲染 |
| B5 | 会话状态 | B4 | 4d | 三重失效测试 |

---

# M1 — 观测层

## A1 — slog 探针（0.5 小时）

### 目标

用最小成本验证一个假设：`execute()` 把四种拒绝压成同一段给模型看的文本，是否真
的造成排查困难。结论决定 A2 的必要性强度（不决定做不做——A2 仍按计划落地，因为它
同时是 B4 的输入端证据）。

### 改动

`apps/api/app/Support/AI/tool_loop.go` 的 `execute()`（约 :353）。

三个分支各加一条 `slog.WarnContext`：

| 分支（现行号） | 条件 | 日志字段 |
|---|---|---|
| :356 | `!ok \|\| !toolAllowed(...)` | `tool`, `reason`（区分 `unknown` / `not_allowed`）, `allowlist` |
| :360 | `budget <= 0` | `tool`, `reason=result_budget_exhausted`, `budget` |
| :369 | `err != nil` | `tool`, `reason=internal_error`, `err` |

### 易错点

- 分支 :356 现在把「未登记」与「不在白名单」合在一起判断，日志里必须拆开——这正是
  要验证的区别。
- 不要动返回给模型的文本。探针只加日志，不改行为。

### 验收

在 dev 环境触发一次 @机器人，确认日志可读。**把「6 行日志够不够定位问题」的结论
写进本任务的 handoff**，作为 A2 字段集的输入。

### 收尾

A2 落地后，这三条探针由 trace 输出取代。若保留，注释标明它是 A2 的冗余。

---

## A2 — ToolTrace（0.5 天）

### 目标

`Orchestrator.Run` 返回本次 run 的每一次工具调用明细，透传到 `aireply` 层并写进
结构化日志。

### 前置

A1 结论。若 A1 证明「仅工具名 + 原因」就够用，字段集仍按本任务书落地（B4 需要
参数与摘要），但日志级别可以降一档。

### 改动文件

1. `apps/api/app/Support/AI/tool_loop.go`
2. `apps/api/app/Support/AI/limits.go`
3. `apps/api/app/Models/AIReply/context.go`
4. `apps/api/app/Models/AIReply/completer.go`
5. `apps/api/app/Models/AIReply/generator.go`
6. `apps/api/app/Support/AI/tool_loop_test.go`

### 实现步骤

**步骤 1 — 类型定义**（`tool_loop.go`，紧跟 `ToolResult` 之后）

```go
// ToolTraceEntry 是一次 run 里的单次工具调用记录。
//
// 它是运营者可见面的数据源：参数与结果摘要都是用户内容，只能进日志与新表，
// 不进 ExecutionRecord（后者声明不含用户内容）。
type ToolTraceEntry struct {
	Step           int    `json:"step"`
	Tool           string `json:"tool"`
	Arguments      string `json:"arguments,omitempty"`
	ArgumentsBytes int    `json:"argumentsBytes,omitempty"`
	ResultBytes    int    `json:"resultBytes"`
	ResultDigest   string `json:"resultDigest,omitempty"`
	IsError        bool   `json:"isError,omitempty"`
	Failure        string `json:"failure,omitempty"`
	LatencyMS      int    `json:"latencyMs"`
}

const (
	ToolFailureUnknownTool     = "unknown_tool"
	ToolFailureNotAllowed      = "not_allowed"
	ToolFailureStepExhausted   = "step_exhausted"
	ToolFailureBudgetExhausted = "result_budget_exhausted"
	ToolFailureInternal        = "internal_error"
)
```

`RunResult` 追加字段：

```go
ToolTrace []ToolTraceEntry `json:"toolTrace,omitempty"`
```

**步骤 2 — 常量**（`limits.go`）

```go
// ToolTrace 是给人看的记录，不是回放载荷：参数与摘要都按这个上限截断，
// 原始字节数另存，让「被截了」这件事本身可见。
MaxToolTraceArgumentsBytes = 1_024
MaxToolTraceDigestBytes    = 256
```

**步骤 3 — `execute()` 签名变更**

```go
func (o *Orchestrator) execute(
	ctx context.Context,
	call ToolCall,
	env ToolContext,
	allowlist []string,
	budget int,
	redaction RedactionSettings,
	step int,
) (content string, used int, trace ToolTraceEntry)
```

内部逻辑：

1. `started := time.Now()`（**不用 `o.clock`**，见易错点 3）。
2. 每次返回前组装 `trace`：`Tool`、`Failure`、`LatencyMS`、`Step`。
3. `Arguments`：`RedactText(redaction, call.Arguments)`，再 `TruncateBytes(..., MaxToolTraceArgumentsBytes)`；
   `ArgumentsBytes = len(call.Arguments)`（截断前的原值）。
4. `ResultDigest`：取**最终回填给模型的那段文本**（即 `content` 变量，含 `IsError`
   前缀与截断标记），丢给 `RedactText` 后截前 `MaxToolTraceDigestBytes` 字节。
5. `ResultBytes = len(content)`。

**步骤 4 — 循环内采集**（`Run` 的 for 循环）

- `remaining <= 0` 分支（约 :318）：这是**第 5 种结局**，在 `execute()` 之外。同样
  append 一条 `ToolTraceEntry{Tool: call.Name, Arguments: <已脱敏截断>,
  ArgumentsBytes: len(call.Arguments), Failure: ToolFailureStepExhausted, Step: result.ModelCalls}`。
- 正常分支：`content, used, trace := o.execute(...)` → `result.ToolTrace = append(result.ToolTrace, trace)`。
- `RedactionSettings` 从 `Run` 已取到的 `settings.Redaction` 传入（`Run` 开头 `o.gateway.Settings` 那处）。

**步骤 5 — AIReply 透出**

`context.go` 的 `CompletionOutput`：

```go
type CompletionOutput struct {
	Text       string
	StopReason string
	Reasoning  string
	// ToolTrace 是本次 run 的工具调用轨迹。不带工具的补全为空。
	ToolTrace []supportai.ToolTraceEntry
	// 以下三项此前被逐层丢弃，一并透出：日志需要回答「这次回复花了多少」。
	Usage      supportai.Usage
	ModelCalls int
	ToolCalls  int
}
```

`completer.go` 的 `Complete`：工具分支填充上述四项；非工具分支 `Usage` 填
`result.Usage`、`ModelCalls` 填 1。

**步骤 6 — 日志**（`generator.go` 的 `Generate`）

- 发布成功：`slog.InfoContext(ctx, "aireply: reply generated", ...)`，属性：
  `topicId`, `purpose`, `modelCalls`, `toolCalls`, `inputTokens`, `outputTokens`,
  `toolCallsPerRun`, `tool_trace`（`json.Marshal` 后的字符串）。
- 发布失败 / 空回复：同一组属性，级别 `Warn`。
- **`Failure != ""` 的条目存在时，级别提到 `Warn`** —— 让配置漂移自己浮出来。

### 易错点（逐条核对）

1. **`execute()` 只有 1 个生产调用点**（`Run` 循环内），但测试里可能直接调用；改
   签名后先 `go build ./...` 确认没有第二个调用点。
2. **`Step` 取 `result.ModelCalls`**，此时循环里刚 `result.ModelCalls++` 过，值就是
   本次模型调用的序号。不要新增独立计数器——两个计数器就是两个真相。
3. **耗时用 `time.Since(started)`，不用可注入 clock**。`Gateway` 的 `clock` 语义是
   窗口对齐，被固定住时若承担耗时测量，延迟会恒为 0（`gateway.go:222` 的既有注释
   就是这个坑）。
4. **参数必须主动脱敏**。`Gateway` 的 `ApplyRedaction` 只作用于出站 `CompletionRequest`，
   而工具参数是模型输出，从未经过脱敏。漏了这一步，邮箱与 IP 会以明文进日志。
5. **`ResultDigest` 取最终 `content`**，不是 `ToolResult.Content` 原文。运营者要解释
   的是「模型看到了什么」，`IsError` 前缀与「结果过长已截断」正是模型看到的东西。
6. **超预算分支也要 trace**。`remaining <= 0` 时模型仍可能发起调用，这类调用在
   `execute()` 里不会经过——漏掉它，trace 就解释不了「AI 在乱撞」。
7. **`RunResult` 是纯追加**。不加字段、不改既有字段语义，避免影响现有调用方。
8. **`tool_loop.go` 改动后约 485 行**，逼近 500 闸门。若超过，先拆 `ToolTraceEntry`
   与采集逻辑到 `tool_trace.go`（同包新文件），不要动架构基线。

### 测试用例（`tool_loop_test.go`，复用 `fakeTool` / `scriptedInvoker` / `newOrchestrator`）

| # | 用例 | 断言要点 |
|---|---|---|
| T-A2-1 | 正常路径 | `len(trace)==1`；`Step==1`；`Tool=="forum-search"`；`Arguments=="{\"query\":\"并发\"}"`；`Failure==""`；`ResultBytes>0`；`LatencyMS>=0` |
| T-A2-2 | 未知工具 | `Failure==ToolFailureUnknownTool`；run 仍成功收尾（有正文） |
| T-A2-3 | 不在白名单 | 工具已登记但 `ToolNames` 未列 → `Failure==ToolFailureNotAllowed` |
| T-A2-4 | step 预算耗尽 | 在既有 `TestOrchestratorRespectsStepBudget` 上追加：超出预算的调用在 trace 里 `Failure==ToolFailureStepExhausted` |
| T-A2-5 | 结果预算耗尽 | 把单条预算压到 1 字节，连调两次 → 第二次 `ToolFailureBudgetExhausted` |
| T-A2-6 | 内部故障 | `fakeTool{err: errors.New("boom")}` → `Failure==ToolFailureInternal`；回填文本不含 `boom` |
| T-A2-7 | 参数截断 | `Arguments` 灌 2KB → `ArgumentsBytes==2048`；`len(Arguments)<=MaxToolTraceArgumentsBytes` 且是合法 UTF-8 |
| T-A2-8 | 脱敏 | settings 开 email 脱敏，参数含 `a@b.com` → trace 内为 `[redacted-email]` |
| T-A2-9 | 顺序 | 三次调用 → trace 按 `Step` 单调不减 |

`Models/AIReply/generator_test.go`：`fakeCompleter` 返回带 `ToolTrace` 的
`CompletionOutput`，断言 `Generate` 行为与契约不变（不耦合日志格式）。

### 验收

```
cd apps/api && go build ./...
cd apps/api && go test ./app/Support/AI/... -run 'ToolTrace|Orchestrator' -v
cd apps/api && go test ./app/Models/AIReply/...
node tests/validate-architecture-boundaries.mjs
```

---

## A3 — correlation id（1 天）

### 目标

一次 run 的全部模型调用能被聚合：生成 run id，写进既有 `Metadata.ResourceType` /
`ResourceID`，落进 `ai_executions`，为 A4 与将来的写动作审计提供地基。

### 关键事实

`Metadata.ResourceType` / `ResourceID`（`contract.go:102`）**已存在、已被
`Normalized()` 修剪空白、但从未被任何生产代码读取**——既不进 `CacheKey`（`CacheKey`
只读 `Metadata.CallerExtensionID`），也不进 `ExecutionRecord`。本次是接通，不是新造。

### 改动文件

1. `apps/api/app/Support/AI/tool_loop.go`（生成 run id）
2. `apps/api/app/Support/AI/store.go`（`ExecutionRecord` 加两字段）
3. `apps/api/app/Support/AI/gateway.go`（8 处 `record` 调用点透传）
4. `apps/api/app/Support/AI/postgres_store.go`（`RecordExecution` + `ListExecutions`）
5. `apps/api/database/migrations/<ts>_ai_execution_resource.sql`（新迁移）
6. `apps/api/app/Support/AI/gateway_test.go`（参数化测试）
7. `apps/web/app/composables/admin/useAdminAI.ts`（前端类型）

### 实现步骤

**步骤 1 — run id 生成**（`tool_loop.go`）

```go
// newRunID 生成一次 run 的关联 id。16 字节随机 hex：同一次 run 的所有模型调用
// 共享它，不同 run 之间不会碰撞。
func newRunID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// 随机源不可用时退回时间戳：可回放性是观测能力，不该让回复失败。
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(buf[:])
}
```

`Run` 开头生成一次，写进每次 `CompletionRequest.Metadata`：

```go
metadata := input.Metadata
metadata.ResourceType = "ai.run"
metadata.ResourceID = runID
```

**注意**：`input.Metadata` 是值类型（`Metadata` struct），拷贝后再改，不污染调用方。

**步骤 2 — `ExecutionRecord`**

```go
	// ResourceType / ResourceID 是归属对象。AI 的一次 run 用它把所有模型调用
	// 串成一条时间线（type="ai.run"，id=runID）。
	ResourceType string `json:"resourceType,omitempty"`
	ResourceID   string `json:"resourceId,omitempty"`
```

**步骤 3 — `Gateway.record` 的 8 处调用点**

`gateway.go` 中每个 `g.record(ctx, ExecutionRecord{...})` 都要加
`ResourceType: request.Metadata.ResourceType, ResourceID: request.Metadata.ResourceID`。

涉及位置（现行号）：:142（disabled）、:156（purpose unknown）、:174（gate decision
denied）、:188（profile unavailable）、:210（cache hit）、:230（provider failed）、
:320（succeeded 的 `g.store` 路径内）。

**用 `rg -n 'g.record\\(' gateway.go` 数一遍，必须与改动数一致。**

**步骤 4 — 迁移**

`apps/api/database/migrations/202610190001_ai_execution_resource.sql`：

```sql
-- +goose Up
-- +sforum OnlineSafe
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '1min';

ALTER TABLE ai_executions
  ADD COLUMN resource_type TEXT NOT NULL DEFAULT '',
  ADD COLUMN resource_id TEXT NOT NULL DEFAULT '';

CREATE INDEX ai_executions_resource_idx
  ON ai_executions (resource_type, resource_id, created_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS ai_executions_resource_idx;
ALTER TABLE ai_executions
  DROP COLUMN IF EXISTS resource_id,
  DROP COLUMN IF EXISTS resource_type;
```

**步骤 5 — 存储层**

`RecordExecution` 的 `INSERT` 列表与 `$n` 占位、`ListExecutions` 的 `SELECT` 与
`rows.Scan`，三处**必须同步**。列顺序建议插在 `error_summary` 之后，减少 diff 噪声。

**步骤 6 — 前端类型**

`useAdminAI.ts` 的 `AdminAIExecution` 追加 `resourceType: string`、`resourceId: string`。
后端 JSON tag 是 `resourceType` / `resourceId`（`omitempty`），字段名必须一致。

### 易错点

1. **绝不把 run id 写进 `CacheKey`**。`CacheKey`（`gateway.go` 约 :356）只摘
   `Purpose`、`CallerExtensionID`、`profile.ID`、`Model`、`PromptVersion`、
   `ResponseFormat`、`ToolChoice`、`Revision`、`System`、工具声明与消息体。加 run id
   会让每次 run 都是新键——缓存永久失效，而且不会有报错，只会账单上涨。
2. **8 处调用点逐个核对**。漏一处，那类失败（如 denied）就没有 run id，回放里出现
   「孤立记录」。
3. **`Run` 里 `settings` 只读一次**。`ResourceType` 来自 `Metadata`，不从 settings 读。
4. **迁移必须 OnlineSafe 且 Down 可逆**。`DROP COLUMN` 前先 `DROP INDEX`。
5. **`ListExecutions` 的 Scan 顺序**与 SELECT 列顺序严格一致，否则运行期才炸。

### 测试用例

| # | 用例 | 断言要点 |
|---|---|---|
| T-A3-1 | run id 存在于每一次调用 | 一次 2 步 run → `len(traces.records)==2` 且两条 `ResourceID` 相同、非空 |
| T-A3-2 | 参数化四路径 | denied(disabled) / denied(gate) / failed(provider) / succeeded 各一条，每条 `ResourceType=="ai.run"` |
| T-A3-3 | 缓存命中 | 第二条命中的记录带 `CacheHit==true` 且 `ResourceID` 与首次一致 |
| T-A3-4 | 不同 run 不串 | 连续两次 `Run` → 两条记录的 `ResourceID` 不同 |
| T-A3-5 | 缓存键未变 | 同一请求两次 `CacheKey(request, profile, rev)` 相等（证明 run id 未进键） |

**T-A3-5 是这条任务的安全带**：它会在有人「顺手」把 metadata 整个写进 CacheKey 时
立刻失败。

### 验收

```
cd apps/api && go build ./...
cd apps/api && go test ./app/Support/AI/... -run 'Execution|CacheKey' -v
cd apps/api && go run ./cmd/migrate   # 本地库跑一次，确认 Up/Down
cd apps/web && bun run typecheck
```

---

## A4 — 轨迹回放（2 天）

### 目标

在控制台观测页签里，从任意一条执行记录展开看到同一 run 的时间线。

### 后端

**新增路由**（`app/Http/Controllers/AI/routes.go`）：

```go
admin.Get("/runs/:runId", h.getRunTimeline)
```

**控制器方法**（`controller.go`）：

```go
// getRunTimeline 返回一次 AI run 的全部执行记录，按时间正序。
// 回放是查询能力的组合，不是新的持久化面。
func (h *Controller) getRunTimeline(c fiber.Ctx) error
```

- 校验 `runId` 非空、长度 ≤ 128（`MaxToolCallIDLen` 同量级即可）。
- 调 `ExecutionStore` 的新方法：

```go
// ListRunExecutions 按归属对象读取一次 run 的全部记录。
// 它是 A4 的唯一新查询面：不新增表，只按 (resource_type, resource_id) 索引查。
ListRunExecutions(ctx context.Context, resourceType, resourceID string, limit int) ([]ExecutionRecord, error)
```

- `resourceType` 固定传 `"ai.run"`，由控制器常量约束，不接受的用户输入。
- 返回 `{ items: [...] }`，与 `listExecutions` 同形状。
- 上限 200，默认 50。

**OpenAPI**：`contracts/openapi/paths/ai.yaml` 加 `GET /admin/ai/runs/{runId}`，
`schemas/ai.yaml` 里 `AIExecution` 加两字段，新增 `AIRunTimeline`（`{items: AIExecution[]}`）。

### 前端

`useAdminAI.ts`：

- 新增 `useAdminAIRunTimeline(runId: Ref<string | null>)`，`useAsyncData` + `request()`。
- `AdminAIExecution` 已含两字段（A3 完成）。

`SFAdminAIObservabilityTab.vue`：

- 表格行可点击 → 展开该 run 的时间线抽屉/内联区。
- 时间线项显示：序号、status、model、latencyMs、tokens、（若为工具步）无额外信息——
  **工具调用明细在日志里，不在执行记录里**，界面不得伪造。
- **两种必须显式标注的情况**：
  - `cacheHit=true` 且只有一条记录 → 标注「本次命中结果缓存，未进入工具循环」。
  - `status=denied` → 标注「未出站」并显示 `gateReason`。
- 无 `resourceId` 的历史记录 → 不显示展开入口，不报错。
- 新控件带 `aria-label` 与 `title`；`390x844` 下不出现横向溢出。

### 易错点

1. **历史记录没有 run id**（A3 之前的行 `resource_id=''`）。前端必须容忍，后端按空
   串查询要返回空列表而不是全表扫描。
2. **同一 run 的记录数随步数增长**。界面不要假设固定条数。
3. **不新增表**。用 `(resource_type, resource_id)` 索引查即可。
4. **`runId` 来自 URL**，必须是纯字符串比较，不进任何拼接 SQL。用参数化查询。
5. **时间线顺序**：`created_at ASC, id ASC`。同毫秒内靠 id 兜底。

### 测试用例

| # | 用例 | 断言要点 |
|---|---|---|
| T-A4-1 | 按 run 聚合 | 2 步 run → 返回 2 条，升序 |
| T-A4-2 | 空 runId | 返回空列表，不 500 |
| T-A4-3 | 未知 runId | 返回空列表 + 200 |
| T-A4-4 | 权限 | 无 `ai.manage` → 403 |
| T-A4-5 | 上限 | 请求 500 条 → 实际 ≤200 |

### 验收

- 桌面与 `390x844` 浏览器 QA：点击行 → 时间线出现；缓存命中 run 有标注；无横向溢出。
- `ruby scripts/validate-openapi-refs.rb`。
- `cd apps/web && bun run typecheck && bun test`。

---

## G — eval 基线（2 天）

### 目标

改提示词、改工具、改预算前后能产出可 diff 的快照，让退化在发布前暴露。

### 为什么是 CLI 而不是 `tests/*.mjs`

现有 `tests/` 惯例是**静态源码检查**（`readFileSync` + markers 比对），不是运行时
调用。eval 需要真实 DB 装配与真实模型调用，放进 `tests/` 会破坏那条惯例，且会误入
CI。CLI 有现成的 DB 装配路径（`newSeedCommand` 同源），且按需手动执行。

### 实现

**命令**：`sforum ai:eval`（`apps/api/cmd/sforum/` 新文件 `ai_eval.go`，注册进
`command.go` 的 `newRootCommand`）。

```
sforum ai:eval --fixture-dir <dir> [--baseline-dir <dir>] [--diff] [--confirm]
```

**装配前提**（写进命令帮助与文档）：

- `DATABASE_URL` 或 `--database-url`：读 `ai_settings`。
- `APP_OPTION_ENC_KEY`：读 Secret Store 里的 provider 密钥。**缺失时命令直接失败
  并说明原因**，不要静默降级成「跑不了」。
- 网关必须 `Enabled=true` 且 `forum.reply` 的 profile 可解析。

**fixture 形态**（`apps/api/cmd/sforum/testdata/ai-eval/*.json`）：

```json
{
  "id": "concurrency-channel",
  "topicTitle": "Go 并发里 channel 怎么用",
  "topicBody": "……",
  "comments": [
    {"author": {"username": "alice", "displayName": "Alice"}, "content": "……", "isBot": false}
  ],
  "triggerCommentIndex": 0,
  "triggerUserID": 77,
  "botUserID": 1
}
```

10 条起步，覆盖而非统计：可检索答案的 ×2、该拒答的 ×2、长楼 ×2、带图 ×2、
跨语言 ×2。

**运行**：装配真实 `Gateway` + `Orchestrator` + `aireply.Generator`；`ThreadReader`
用 fixture 驱动的假实现（`fixtureThreadReader`）。对每条 fixture 跑一次 `Generate`
的**前半段**（读上下文 → 组装提示词 → 调 completer），**不发布**——避免污染开发库。

**快照输出**（`<baseline-dir>/<id>.json`）：

```json
{
  "id": "...",
  "text": "...",
  "stopReason": "...",
  "toolTrace": [...],
  "modelCalls": 2,
  "usage": {"inputTokens": 0, "outputTokens": 0}
}
```

**diff 模式**：逐字段比较，输出：

```
[changed] concurrency-channel
  text: 132 → 118 chars
  toolTrace: 2 → 1 calls (forum-search dropped)
[unchanged] 3 fixtures
```

**成本保护**：不带 `--confirm` 时只打印「将执行 N 次真实调用，约 X tokens」并退出。

### 易错点

1. **不发布**。eval 走 completer 而不走 poster，否则每次跑都在开发库留垃圾帖。
2. **不写执行记录**（或写到可识别的 purpose）。若走 `Gateway.Execute`，执行记录会
   正常写入——可接受，但要在文档里说明「eval 会消耗配额」，并把 purpose 标清楚。
3. **`APP_OPTION_ENC_KEY` 缺失是硬失败**，不要退化成「用空密钥读不到就跳过」。
4. **fixture 不含真实用户 PII**。正文可脱敏改写，用户名用 `alice`/`bob`。
5. **基线文件进仓库**，diff 才是跨机器可比的。放在 `testdata/ai-eval/baseline/`。

### 验收

- 跑一次生成基线。
- 人为把 `DefaultReplySystemPrompt` 改一句话 → `--diff` 必须报出变化。
- 恢复提示词 → `--diff` 报 `unchanged`。

---

# M2 — 能力层（预算、工具、用途）

> **门禁**：M2 任一项动手前，G 的基线必须已存在。没有基线的能力改动不上线。

## B1 — 放宽预算（0.5 天）

### 目标

减少「第三跳被截断导致模型凭已有信息编」。

### 改动

`apps/api/app/Support/AI/limits.go`：

```go
DefaultToolCallsPerReply = 5   // was 3
MaxToolCallsPerReply     = 5   // 不变：进程安全边界，抬它需要单独决策记录
MaxToolResultBytes       = 12_000  // was 8_000
MaxToolResultRunBytes    = 24_000  // was 16_000
```

### 为什么两个结果预算一起抬

只抬单条而累计不变，等于把截断点从第一条挪到第二条——模型仍然在半截信息上作答，
但你会以为问题解决了。

### 为什么硬顶不动

`MaxToolCallsPerReply` 是「运营者不能越过」的边界。抬它是一次决策，不是一次调参。

### 易错点

- `settings.go` 里站点闸门字段的范围校验要跟着改（`ResolvedToolCallsPerReply` 的
  默认值与上限）。
- 控制台 providers 页签的滑块/输入范围同步。
- 相关测试里写死的 `3` 要改；**先 `rg -n 'DefaultToolCallsPerReply|ToolCallsPerReply'` 数一遍**。

### 验收

用 G 对比改前改后：`toolTrace` 里 `step_exhausted` 应减少，`text` 变化需人工 triage。
收益不成立就回退——这是唯一一项可以「做了再撤」的改动。

---

## B2 — 补 3 个工具（3 天）

### 目标

补齐三个因「窗口太小 / 只能平铺读」造成的重复劳动与自相矛盾。

### 工具 1 — `forum-comment-thread`

**动机**：`forum-topic-read` 只能分页读平铺评论，回答「这条追问有没有人回过」要
多次翻页。

**依赖**：`CommentReader` 扩一个方法（`aitools.go`）：

```go
// CommentThreadReader 读取某条评论的回复子树。
type CommentThreadReader interface {
	ListCommentReplies(ctx context.Context, input forum.CommentListInput) (forum.CommentList, error)
}
```

若 `forum.Service` 没有对应方法，用 `ListComments` + `ParentID` 过滤实现适配器；
**优先复用现有方法**，确有缺口再加领域方法。

**参数**：`{commentId, depth<=2, limit<=30}`。

### 工具 2 — `forum-attachment-read`

**动机**：主楼与回复里的附件（配置片段、日志、截图 OCR 文本）当前完全不可读。

**依赖**：`attachments.Service.OpenContent`（与 `providers.AIReplyImageReader`
同一条授权链）。以 bot 账号身份读取。

**参数**：`{attachmentId}`。返回文本内容（截断）或元数据（文件名、类型、大小）。
二进制图片不在本工具范围（多模态已由 `PartImage` 承担）。

### 工具 3 — `ai-self-history`

**动机**：`ReplyContext.RecentComments` 带 `IsBot` 但窗口只有
`RecentCommentWindow = 10` 条，超窗即丢，导致机器人自相矛盾。

**依赖**：新增窄接口，按 `(topicID, botUserID)` 读该 bot 在本话题的历史发言：

```go
// BotHistoryReader 读取某个机器人账号在指定话题里的历史发言。
type BotHistoryReader interface {
	ListBotComments(ctx context.Context, topicID, botUserID int64, limit int) ([]forum.Comment, error)
}
```

**参数**：`{topicId?, limit<=10}`。默认当前话题（`ToolContext.TopicID`）。

### 注册与白名单

1. `aitools.Deps` 加三个依赖字段。
2. `NewBuiltinRegistry` 的 `register` 列表各加一行（依赖为 nil 时不注册）。
3. `bootstrap/api_assembly_domains.go` 的 `aitools.Deps{...}` 补齐。
4. `reply_prompt.go` 的 `CoreReplyToolAllowlist` 追加三个名字。
5. `CoreReplyPromptVersion` 从 `forum-reply@6` 升到 `forum-reply@7`（提示词里列了
   工具能力，工具集变化必须升版本，否则缓存键与历史记录对不上产出版本）。

### 易错点

1. **可见性判定一律用 `ToolContext.ViewerUserID`**。不得因为「bot 是系统账号」而
   放宽——那正是写动作面被否决的同一个理由。
2. **不可见资源统一回答「不存在」**，不回答「无权限」（`TopicReadTool` 的既有纪律）。
3. **依赖为 nil 时不注册**：工具存在但必然失败，比工具不存在更浪费模型的步数。
4. **参数解析失败返回给模型看的说明**，不走 `error`（`decodeArguments` 的既有约定）。
5. **提示词版本必须升**。漏升会让 `CacheKey` 命中旧结果，看起来「改了没效果」。
6. **每个工具的输出按 rune 截断**（`clip`），不要按字节切多字节字符。

### 测试用例（每个工具成对）

| 用例 | 断言 |
|---|---|
| allowed | 可见资源返回正文 + 链接；`ResultBytes > 0` |
| denied | 不可见资源返回「不存在」，不含任何资源元信息 |
| 参数非法 | 返回说明文本，`IsError` 语义一致，不 panic |
| 截断 | 超长输入被 `clip` 且是合法 UTF-8 |

### 验收

- `go test ./app/Models/AITools/...`
- 运行时：dev 触发一次回复，确认三个工具在 `ToolRegistry.Available()` 里出现，
  且控制台用途页签的已注册工具选择器能看到。

---

## B3 — 扩 2 条 purpose（3 天）

### 目标

给算力第二个出口：答案蒸馏与长楼导览，都是只读 + 单次补全。

### 声明（`Support/AI/purpose.go`）

```go
CorePurposeForumDigest = "forum.digest"
CorePurposeForumTour   = "forum.tour"
```

| 字段 | digest | tour |
|---|---|---|
| CostClass | economy | economy |
| FailurePosture | 与 forum.reply 同 | 同 |
| ToolAllowlist | `forum-topic-read`, `forum-search` | `forum-topic-read`, `forum-topic-list` |
| ResourceType | `topic` | `topic` |
| PromptVersion | `forum-digest@1` | `forum-tour@1` |
| BlastRadius | `single` | `single` |

### Job 与触发面

- `Jobs/AIReply/` 同级新增 `Jobs/AIDerived/`（或并入现有 AI job 包，按职责取舍）。
- job kind：`ai.forum_digest` / `ai.forum_tour`，`MaxAttempts: 3`，`QueueAI`。
- **触发面只做运营者手动**：控制台话题操作区的按钮或 CLI 子命令。不做定时、不做
  事件驱动——避免重新引入「自主触发」的争议面。
- 产物：由运营者预览后手动发布，或直接落成草稿。**默认不自动发布**。

### 易错点

1. **不注册新权限键就复用 `ai.manage`**：手动触发是管理动作，不需要新权限。
2. **新增 purpose 必须过 `PurposeRegistry.Register` 的校验**：id 格式、双语标签非空。
3. **失败姿态照抄 forum.reply**，不要发明新的降级语义。
4. **`DefaultPurposeRegistry()` 是进程单例**，新增后 `purpose_test.go` 的「唯一条目」
   断言会失败，要同步更新。

### 验收

- 两个 purpose 在控制台用途清单出现。
- 手动触发一次，产物落成预期位置，`ai_executions` 里能看到对应 purpose。
- 用 G 的机制为这两个 purpose 各建 2 条 fixture（可选，但推荐）。

---

# M3 — 能力层（证据、状态）

## B4 — 证据链溯源（6 天）

### 目标

AI 产物携带可校验的结构化引用；引用不存在时在校验阶段被发现，而不是发布之后。

### 产物结构

`Models/AIReply/context.go`：

```go
// Citation 是 AI 产物的一条出处。
type Citation struct {
	TopicID   int64  `json:"topicId,omitempty"`
	CommentID int64  `json:"commentId,omitempty"`
	Excerpt   string `json:"excerpt,omitempty"`
}

type BotCommentInput struct {
	TopicID  int64
	ParentID int64
	Content  string
	Citations []Citation  // 新增
}
```

### 生成侧

1. 提示词追加格式要求：正文里用 `[[cite:topic:42]]` / `[[cite:comment:913]]` 标记出处。
2. 生成器解析：正则提取标记 → 从正文剥离 → 得到干净正文 + 引用列表。
3. 校验：逐条查资源是否存在且可见（复用 `ThreadReader` / forum reader）。不存在
   的整条丢弃并记 `Warn` 日志。
4. `Content` 只放剥离后的正文；`Citations` 单独传。

### 存储

新表 `ai_citations`：

```sql
CREATE TABLE ai_citations (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  run_id TEXT NOT NULL DEFAULT '',
  topic_id BIGINT NOT NULL DEFAULT 0,
  comment_id BIGINT NOT NULL DEFAULT 0,
  excerpt TEXT NOT NULL DEFAULT ''
);
CREATE INDEX ai_citations_run_idx ON ai_citations (run_id, created_at DESC);
CREATE INDEX ai_citations_target_idx ON ai_citations (topic_id, comment_id);
```

`run_id` 与 A3 对齐。

### 渲染

前端在 AI 产出评论下渲染「依据」折叠块，链接到真实楼层。零引用时不渲染。

### 易错点

1. **模型不遵守格式时降级为「无引用」，不报错**。格式是概率性的，一次不遵守不该
   让整次回复失败。
2. **剥离标记后要校验正文非空**——如果整段正文只有标记，会产出空评论（已有
   `ErrEmptyReply` 语义，要接上）。
3. **校验失败整条丢弃**，不要把未校验的引用写进库。
4. **`Excerpt` 从被引用资源里取**，不从模型输出里取——模型可能编造引文。
5. **不改变 `SubmitDecision` 语义**：引用是展示层，不是审核输入。

### 验收

- 引用存在 → 渲染正确、可跳转。
- 引用不存在 → 该条被丢弃，日志有记录，回复正常发布。
- 正文只有标记 → 走 `ErrEmptyReply` 路径。

---

## B5 — 会话状态（4 天）

### 目标

同一话题的多次回复共享已查证事实，减少重复查询与自我矛盾。

### 存储

```sql
CREATE TABLE ai_topic_state (
  topic_id BIGINT NOT NULL,
  bot_user_id BIGINT NOT NULL,
  document JSONB NOT NULL,
  revision BIGINT NOT NULL DEFAULT 1,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (topic_id, bot_user_id)
);
```

### 文档形态

```go
type TopicState struct {
	LastRunID     string     `json:"lastRunId"`
	PromptVersion string     `json:"promptVersion"`
	VerifiedFacts []Fact     `json:"verifiedFacts"` // 只存通过 B4 校验的引用事实
	LastReplies   []ReplyRef `json:"lastReplies"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}
```

**只存"通过校验的引用事实"，不存模型自由文本。** 这是它与聊天历史的根本区别。

### 注入

`buildReplyPrompt` 增加一段：

```
你在本话题里此前已确认过的事实：
- （引用 42 楼）……
```

上限 20 条 / 2KB，超出按时间倒序保留最近。

### 失效（三重，全部必须实现）

1. `PromptVersion` 与当前用途声明不一致 → 整份失效。
2. `UpdatedAt` 超过 14 天 → 失效。
3. 引用的楼层被删除或转为不可见 → 该条 `VerifiedFact` 在下次读取时剔除（不是整份失效）。

### 易错点

0. **失效不等于丢失。** 本节的「三重失效」是主动丢弃；防丢失的四条措施（乐观锁、
   读取容错、写入时机、自检日志）在「记忆连续性 · 层 2」里，必须一起实现。只做
   失效不做防丢失，状态会在并发下被后写者静默覆盖。
1. **过期状态比没有状态更糟**。错误事实一旦进持久记忆，会在后续每轮被当前提引用，
   而 B4 只校验「引用存在」，不校验「引用支持这个结论」。
2. **写入必须在发布成功之后**。发布失败却写了状态，下一轮会引用一段没人看过的结论。
3. **状态读取失败按「没有状态」处理**，不让整次回复失败。
4. **并发**：`revision` 用乐观更新，冲突时丢弃本次写入而不是覆盖（后写者不一定是
   更正确的那个）。
5. **失败事实剔除是逐条**，不整份删——整份删会让状态在一次可见性变更后全部丢失。

### 验收

- 同一话题连续两次 @：第二次提示词里含第一次已确认的事实。
- 改提示词版本 → 状态失效，提示词里不再出现。
- 把被引用楼层删除 → 该条事实消失，其余保留。

---

# 数据库迁移清单

| 迁移文件 | 内容 | 任务 |
|---|---|---|
| `202610190001_ai_execution_resource.sql` | `ai_executions` 加 `resource_type` / `resource_id` + 索引 | A3 |
| `202610190002_ai_citations.sql` | 新表 `ai_citations` + 两个索引 | B4 |
| `202610190003_ai_topic_state.sql` | 新表 `ai_topic_state` | B5 |

全部 `+goose Up` / `+goose Down` 成对，`+sforum OnlineSafe`，
`SET LOCAL lock_timeout = '5s'`。

---

# 契约与前端清单

| 面 | 改动 | 任务 |
|---|---|---|
| OpenAPI `paths/ai.yaml` | `GET /admin/ai/runs/{runId}` | A4 |
| OpenAPI `schemas/ai.yaml` | `AIExecution` 加 `resourceType` / `resourceId`；新增 `AIRunTimeline` | A3 / A4 |
| `useAdminAI.ts` | `AdminAIExecution` 扩字段；新增 run 时间线 composable | A3 / A4 |
| `SFAdminAIObservabilityTab.vue` | 行展开 + 时间线 + 缓存命中标注 | A4 |
| 双语键 | 所有新文案 `zh-CN` / `en-US` | 全部 |

---

# 全局陷阱清单（实现前通读，逐条核对）

1. **`Gateway.record` 有 8 个调用点**，A3 必须全改。`rg -n 'g.record\(' gateway.go` 数一遍。
2. **`CacheKey` 不含 `ResourceType` / `ResourceID`**。加 run id 安全；把它写进
   `CacheKey` 会让缓存永久失效且不报错。
3. **`Gateway` 的脱敏只作用于出站请求**。工具参数是模型输出，trace 侧必须主动脱敏。
4. **耗时用 `time.Since`**，不用可注入 clock（注入 clock 的语义是窗口对齐）。
5. **`Step` 用 `ModelCalls`**，不新增第二个计数器。
6. **缓存命中的 run 只有一条记录**：正确行为，界面必须标注，否则被误读成「没调工具」。
7. **`execute()` 的四个拒绝分支 + 循环内的 `remaining<=0` 是五种结局**，trace 必须
   全部覆盖。
8. **`ExecutionRecord` 不含用户内容**：trace 与引用都不进这张表。
9. **迁移 Up/Down 成对**，先 `DROP INDEX` 再 `DROP COLUMN`。
10. **`PostgresStore` 的 `RecordExecution` / `ListExecutions` 三处同步**（INSERT 列、
    SELECT 列、Scan 顺序）。
11. **前端字段名与 Go JSON tag 一致**：`resourceType` / `resourceId`。
12. **新工具一律以 `ViewerUserID` 判定可见性**，不得用 bot 身份。
13. **`Deps` 为 nil 时不注册工具**。
14. **工具集变化必须升 `CoreReplyPromptVersion`**，否则缓存键对不上产出版本。
15. **B4 引用解析失败降级为无引用**，不报错。
16. **B5 状态在发布成功之后写**，且三重失效全部实现。
17. **`tool_loop.go` 逼近 500 行闸门**。超过就拆 `tool_trace.go`（同包），不动基线。
18. **eval 不发布、不写垃圾帖**；`APP_OPTION_ENC_KEY` 缺失是硬失败。

---

# 验证矩阵

| 层 | 内容 | 命令 / 方式 |
|---|---|---|
| L1 单元 | A2 九例、A3 五例、A4 五例、B2 每工具四例、B4/B5 allowed+denied | `go test ./app/Support/AI/... ./app/Models/AIReply/... ./app/Models/AITools/...` |
| L2 集成 | 真实 PostgreSQL 装配下的执行记录与聚合查询 | `go test ./app/Support/AI/... -run Postgres` |
| L3 架构 | 文件行数、扁平目录、基线非增长 | `node tests/validate-architecture-boundaries.mjs` |
| L4 契约 | OpenAPI `$ref` 完整性 | `ruby scripts/validate-openapi-refs.rb` |
| L5 前端 | 类型 + 单测 | `cd apps/web && bun run typecheck && bun test` |
| L6 运行时 | dev 触发一次 @机器人：日志含 `tool_trace`、邮箱已脱敏、执行记录带 run id | 人工 |
| L7 浏览器 | 观测页签桌面 + `390x844`：展开时间线、缓存命中标注、无横向溢出 | 人工 QA |

**发布前必须同时满足 L1–L5 全绿，L6/L7 有截图或日志证据。**

---

# 回滚手册

| 任务 | 回滚方式 | 是否有数据残留 |
|---|---|---|
| A1 | 删日志行 | 无 |
| A2 | 删 `ToolTrace` 字段与采集 | 无（纯内存） |
| A3 | 迁移 Down + 删字段 | 有（历史行 `resource_id` 归零，可接受） |
| A4 | 删路由与前端入口 | 无 |
| G | 删命令与 testdata | 无 |
| B1 | 恢复 `limits.go` 三个常量 | 无 |
| B2 | 从 `NewBuiltinRegistry` 与 allowlist 移除 | 无 |
| B3 | 移除 purpose 声明（`DefaultPurposeRegistry` 断言同步） | 可能留有 job 记录 |
| B4 | 移除引用解析；`ai_citations` 保留但不再写入 | 有 |
| B5 | 移除注入；`ai_topic_state` 保留 | 有 |

---

# 不在范围内

- 自主触发链（AI 产物再触发 AI）。
- 持久 goal / 无人值守的自主多轮。
- MCP 进 Core、通用 subagent 编排。
- 把 run trace 塞进 `ai_executions`。
- 前台交互面（需重开 `2026-10-12` 决议 #3）。
- 写动作面（需重写「AI 只起草、人类定稿」）。
- 写动作面按 bot 权限任意写。

---

# 进度账本

**唯一真相来源。** 每完成一个任务立刻更新本表，不要攒到收尾。并行执行时，
其他轨道靠这张表判断能否开工。

状态取值：`未开始` / `进行中` / `待验证` / `已完成`。
`进行中` 必须写清停在哪个文件的哪一步；`待验证` 必须写清验证卡在哪条命令。

| 任务 | 轨道 | 状态 | 已改文件 | 验证到哪 | 下一个接手点 | 未解问题 |
|---|---|---|---|---|---|---|
| A1 slog 探针 | T1 | 未开始 | — | — | 在 `execute()` 四分支加日志 | — |
| A2 ToolTrace | T1 | 未开始 | — | — | 读 A1 结论后定义 `ToolTraceEntry` | — |
| A3 correlation id | T1 | 未开始 | — | — | 生成 run id + 迁移 001 + 8 处 record | — |
| A4 轨迹回放 | T1 | 未开始 | — | — | 新端点 + `ListRunExecutions`（前端交 T6） | — |
| G eval 基线 | T2 | 未开始 | — | — | `sforum ai:eval` 子命令 + 10 条 fixture | — |
| B1 放宽预算 | T1 | 未开始 | — | — | 等 A2 落地后再改 `limits.go` | — |
| B2 三个工具 | T3 | 未开始 | — | — | 等 G 基线存在后开工 | — |
| B3 两条 purpose | T4 | 未开始 | — | — | 等 G 基线存在后开工 | — |
| B4 证据链 | T5 | 未开始 | — | — | 等 A2 落地 + G 基线存在 | — |
| B5 会话状态 | T5 | 未开始 | — | — | 等 B4 的 Citation 结构定稿 | — |

**账本更新示例**（`进行中` 的正确写法）：

```
| A3 correlation id | T1 | 进行中 | tool_loop.go(已加 run id)、store.go(已加两字段) | go build 通过；迁移未跑 | 下一步：gateway.go 的 8 处 record，已在 142/156 完成，剩 174/188/210/230/320 | 第 320 处在 g.store 内，需确认 request 是否已脱敏后仍带 metadata |
```

---

# 交付时同步更新

1. `knowledge/modules/ai.md`：新增「可观测性」与「能力面」两节。
2. `knowledge/sessions/`：一个 hot handoff，格式按 `AGENTS.md`。
3. `knowledge/plans/README.md`：本计划状态从 `ready` → `active` → 完成后移入归档。
4. `knowledge/index.md`：Latest Handoff 指向新 handoff。
5. 如有架构取舍（例如 B5 选 Postgres 而非 Redis），补一条 `knowledge/decisions/`。
